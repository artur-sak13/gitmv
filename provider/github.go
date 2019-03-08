// The MIT License (MIT)
//
// Copyright (c) 2019 Artur Sak
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package provider

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/artur-sak13/gitmv/auth"

	"github.com/google/go-github/v21/github"
	"golang.org/x/oauth2"
)

type CachedIssue struct {
	Issue *GitIssue

	commentMu sync.RWMutex
	Comments  map[time.Time]*GitIssueComment
}

type CachedRepo struct {
	Repo *GitRepository

	issueMu sync.RWMutex
	Issues  map[string]*CachedIssue

	labelMu sync.RWMutex
	Labels  map[string]*GitLabel
}

// GitHubProvider implements the provider interface for GitHub
type GithubProvider struct {
	Client  *github.Client
	Context context.Context
	ID      *auth.ID

	retries   int
	Repocache map[string]*CachedRepo
	Members   map[string]*github.User
}

// NewGithubProvider creates a new GitHub clients which implements the provider interface
func NewGithubProvider(ctx context.Context, id *auth.ID) (GitProvider, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: id.Token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	return WithGithubClient(ctx, client, id), nil
}

// WithGithubClient creates a new GitProvider with a GitHub client
func WithGithubClient(ctx context.Context, client *github.Client, id *auth.ID) GitProvider {
	return &GithubProvider{
		Client:    client,
		Context:   ctx,
		ID:        id,
		retries:   5,
		Repocache: make(map[string]*CachedRepo),
	}
}

// CreateRepository creates a new GitHub repository
func (g *GithubProvider) CreateRepository(srcRepo *GitRepository) (*GitRepository, error) {
	repo := &github.Repository{
		Name:        github.String(strings.TrimSpace(srcRepo.Name)),
		Private:     github.Bool(true),
		Description: github.String(strings.TrimSpace(srcRepo.Description)),
		Archived:    github.Bool(srcRepo.Archived),
	}
	if !g.RepositoryExists(srcRepo.Name) {
		r, _, err := g.Client.Repositories.Create(g.Context, g.ID.Owner, repo)
		if err != nil {
			return nil, fmt.Errorf("failed to create repository %s/%s due to: %s", g.ID.Owner, srcRepo.Name, err)
		}
		return fromGithubRepo(r), nil
	}
	return fromGithubRepo(repo), nil
}

func fromGithubRepo(repo *github.Repository) *GitRepository {
	return &GitRepository{
		Name:        repo.GetName(),
		Description: repo.GetDescription(),
		CloneURL:    repo.GetCloneURL(),
		SSHURL:      repo.GetSSHURL(),
		Archived:    repo.GetArchived(),
		Fork:        repo.GetFork(),
		Empty:       repo.GetSize() == 0,
		PID:         int(repo.GetID()),
	}
}

// RepositoryExists checks if a given repostory already exists in GitHub
func (g *GithubProvider) RepositoryExists(name string) bool {
	_, _, err := g.Client.Repositories.Get(g.Context, g.ID.Owner, name)
	return err == nil
	// return r != nil && r.StatusCode == 404
}

// CreateIssue creates a new GitHub issue
func (g *GithubProvider) CreateIssue(issue *GitIssue) (*GitIssue, error) {
	issueRequest := &github.IssueRequest{
		Title:  github.String(strings.TrimSpace(issue.Title)),
		Body:   github.String(strings.TrimSpace(issue.Body)),
		State:  github.String(strings.TrimSpace(issue.State)),
		Labels: ToGitLabelStringSlice(issue.Labels),
	}
	if issue.Assignees != nil && len(issue.Assignees) > 0 {
		var assignees []string
		for _, assignee := range issue.Assignees {
			assignees = append(assignees, g.Members[assignee.Email].GetLogin())
		}
		issueRequest.Assignees = &assignees
	}

	result, _, err := g.Client.Issues.Create(g.Context, g.ID.Owner, issue.Repo, issueRequest)
	if err != nil {
		return nil, err
	}

	number := 0
	if result.Number != nil {
		number = result.GetNumber()
	}
	return fromGithubIssue(number, result), nil
}

func fromGithubIssue(number int, issue *github.Issue) *GitIssue {
	labels := []GitLabel{}
	if issue.Labels != nil && len(issue.Labels) > 0 {
		for _, label := range issue.Labels {
			label := label // Pin to scope
			labels = append(labels, *fromGithubLabel(&label))
		}
	}

	assignees := []GitUser{}
	if issue.Assignees != nil && len(issue.Assignees) > 0 {
		for _, assignee := range issue.Assignees {
			assignees = append(assignees, *fromGithubUser(assignee))
		}
	}

	return &GitIssue{
		Number:    number,
		State:     issue.GetState(),
		Title:     issue.GetTitle(),
		Body:      issue.GetBody(),
		Labels:    labels,
		User:      fromGithubUser(issue.GetUser()),
		Assignees: assignees,
	}
}

func fromGithubUser(user *github.User) *GitUser {
	return &GitUser{
		Login: user.GetLogin(),
		Name:  user.GetName(),
		Email: user.GetEmail(),
	}
}

// CreateIssueComment creates a new GitHub issue comment
func (g *GithubProvider) CreateIssueComment(comment *GitIssueComment) error {
	issueComment := &github.IssueComment{
		User:      g.Members[comment.User.Email],
		Body:      github.String(strings.TrimSpace(comment.Body)),
		CreatedAt: &comment.CreatedAt,
		UpdatedAt: &comment.UpdatedAt,
	}
	_, _, err := g.Client.Issues.CreateComment(g.Context, g.ID.Owner, comment.Repo, comment.IssueNum, issueComment)
	if err != nil {
		return err
	}
	return nil
}

// CreateLabel creates a new GitHub issue label
func (g *GithubProvider) CreateLabel(srcLabel *GitLabel) (*GitLabel, error) {
	label := &github.Label{
		Name:        github.String(strings.TrimSpace(srcLabel.Name)),
		Color:       github.String(strings.Trim(srcLabel.Color, "#\r\n\t")),
		Description: github.String(strings.TrimSpace(srcLabel.Description)),
	}

	result, _, err := g.Client.Issues.CreateLabel(g.Context, g.ID.Owner, srcLabel.Repo, label)
	if err != nil {
		return nil, err
	}
	return fromGithubLabel(result), nil
}

func fromGithubLabel(label *github.Label) *GitLabel {
	return &GitLabel{
		Name:        label.GetName(),
		Color:       label.GetColor(),
		Description: label.GetDescription(),
	}
}

// MigrateRepo migrates a repo from an existing provider into GitHub
func (g *GithubProvider) MigrateRepo(repo *GitRepository, token string) (string, error) {
	// Must create repository before running import
	repoImport := &github.Import{
		VCS:         github.String("git"),
		VCSURL:      github.String(repo.CloneURL),
		VCSUsername: github.String(repo.Owner),
		VCSPassword: github.String(token),
	}
	result, _, err := g.Client.Migrations.StartImport(g.Context, g.ID.Owner, repo.Name, repoImport)
	if err != nil {
		return "", err
	}
	return result.GetStatus(), nil
}

// GetImportProgress checks the progress of a previously started GitHub import
func (g *GithubProvider) GetImportProgress(repoName string) (string, error) {
	migration, _, err := g.Client.Migrations.ImportProgress(g.Context, g.ID.Owner, repoName)
	if err != nil {
		return "", err
	}
	return migration.GetStatus(), nil
}

// type retryAbort struct{ error }

// func (r *retryAbort) Error() string {
// 	return fmt.Sprintf("aborting retry loop: %v", r.error)
// }

// func (g *GithubProvider) sleepForAttempt(retryCount int) {
// 	maxDelay := 20 * time.Second
// 	delay := time.Second * time.Duration(math.Exp2(float64(retryCount)))
// 	if delay > maxDelay {
// 		delay = maxDelay
// 	}
// 	time.Sleep(delay)
// }

// func (g *GithubProvider) retry(action string, call func() (*github.Response, error)) (*github.Response, error) {
// 	var err error
// 	var resp *github.Response

// 	for retryCount := 0; retryCount <= g.retries; retryCount++ {
// 		if resp, err = call(); err == nil {
// 			return resp, nil
// 		}
// 		switch err := err.(type) {
// 		case *github.RateLimitError:
// 			return resp, err
// 		case *github.TwoFactorAuthError:
// 			return resp, err
// 		case *retryAbort:
// 			return resp, err
// 		}

// 		if retryCount == g.retries {
// 			return resp, err
// 		}
// 		logrus.Errorf("error %s: %v. Retrying...\n", action, err)

// 		g.sleepForAttempt(retryCount)
// 	}
// 	return resp, err
// }

// GetAuth returns a string with a user's api authentication token
func (g *GithubProvider) GetAuth() *auth.ID {
	return g.ID
}

// GetRepositories retrieves a list of GitHub repositories for the organization/owner
func (g *GithubProvider) GetRepositories() ([]*GitRepository, error) {
	repoOpts := github.RepositoryListByOrgOptions{}
	var result []*github.Repository

	_, err := g.depaginate(func(opts github.ListOptions) (*github.Response, error) {
		repoOpts.ListOptions = opts

		repos, resp, err := g.Client.Repositories.ListByOrg(g.Context, g.ID.Owner, &repoOpts)

		result = append(result, repos...)
		return resp, err
	})

	if err != nil {
		return nil, err
	}

	repoListOpts := github.RepositoryListOptions{}
	_, err = g.depaginate(func(opts github.ListOptions) (*github.Response, error) {
		repoListOpts.ListOptions = opts
		repos, resp, err := g.Client.Repositories.List(g.Context, "", &repoListOpts)

		result = append(result, repos...)
		return resp, err
	})

	if err != nil {
		return nil, err
	}

	var repos []*GitRepository
	for _, repo := range result {
		repos = append(repos, fromGithubRepo(repo))
	}
	return repos, nil
}

// GetIssues retrieves a list of issues associated with a GitHub repository
func (g *GithubProvider) GetIssues(pid int, repo string) ([]*GitIssue, error) {
	issueOpts := github.IssueListByRepoOptions{
		State: "all",
	}

	var result []*github.Issue
	_, err := g.depaginate(func(opts github.ListOptions) (*github.Response, error) {
		issueOpts.ListOptions = opts

		issues, resp, err := g.Client.Issues.ListByRepo(g.Context, g.ID.Owner, repo, &issueOpts)

		result = append(result, issues...)
		return resp, err
	})

	if err != nil {
		return nil, err
	}

	var issues []*GitIssue

	for _, issue := range result {
		issues = append(issues, fromGithubIssue(issue.GetNumber(), issue))
	}

	return issues, nil

}

// GetComments retrieves a list of issue comments associated with a GitHub issue
func (g *GithubProvider) GetComments(pid, issueNum int, repo string) ([]*GitIssueComment, error) {
	var list []*github.IssueComment
	_, err := g.depaginate(func(opts github.ListOptions) (*github.Response, error) {
		comments, resp, err := g.Client.Issues.ListComments(
			g.Context,
			g.ID.Owner,
			repo,
			issueNum,
			&github.IssueListCommentsOptions{
				ListOptions: opts,
			})
		list = append(list, comments...)

		return resp, err
	})

	if err != nil {
		return nil, err
	}
	var comments []*GitIssueComment
	for _, comment := range list {
		comments = append(comments, fromGithubComment(repo, issueNum, comment))
	}

	return comments, nil
}

func fromGithubComment(repo string, issueNum int, comment *github.IssueComment) *GitIssueComment {
	return &GitIssueComment{
		Repo:      repo,
		IssueNum:  issueNum,
		User:      *fromGithubUser(comment.User),
		Body:      comment.GetBody(),
		CreatedAt: comment.GetCreatedAt(),
		UpdatedAt: comment.GetUpdatedAt(),
	}
}

// GetLabels retrieves a list of labels associated with a GitHub repository
func (g *GithubProvider) GetLabels(pid int, repo string) ([]*GitLabel, error) {
	var list []*github.Label

	_, err := g.depaginate(func(opts github.ListOptions) (*github.Response, error) {
		labels, resp, err := g.Client.Issues.ListLabels(g.Context, g.ID.Owner, repo, &opts)

		list = append(list, labels...)
		return resp, err
	})

	if err != nil {
		return nil, err
	}

	var labels []*GitLabel
	for _, label := range list {
		labels = append(labels, fromGithubLabel(label))
	}

	return labels, nil
}

func (g *GithubProvider) getMembers() ([]*github.User, error) {
	memberOpts := github.ListMembersOptions{}
	var users []*github.User
	_, err := g.depaginate(func(opts github.ListOptions) (*github.Response, error) {
		memberOpts.ListOptions = opts
		members, resp, err := g.Client.Organizations.ListMembers(g.Context, g.ID.Owner, &memberOpts)

		users = append(users, members...)

		return resp, err
	})

	if err != nil {
		return nil, err
	}

	return users, nil
}

func (g *GithubProvider) setMemberMap(members []*github.User) {
	g.Members = make(map[string]*github.User)
	for _, member := range members {
		if email := member.GetEmail(); email != "" {
			g.Members[email] = member
		}
	}
}

func (g *GithubProvider) LoadCache() error {
	repos, err := g.GetRepositories()
	if err != nil {
		return err
	}
	members, err := g.getMembers()
	if err != nil {
		return err
	}

	g.setMemberMap(members)

	wg := sync.WaitGroup{}
	for _, repo := range repos {
		cachedrepo := &CachedRepo{
			Repo:    repo,
			issueMu: sync.RWMutex{},
			Issues:  make(map[string]*CachedIssue),
			labelMu: sync.RWMutex{},
			Labels:  make(map[string]*GitLabel),
		}

		wg.Add(1)

		go func(cachedrepo *CachedRepo) {
			defer wg.Done()
			g.fillIssues(cachedrepo)
			g.fillLabels(cachedrepo)
		}(cachedrepo)

		g.Repocache[repo.Name] = cachedrepo
	}
	wg.Wait()
	return nil
}

func (g *GithubProvider) fillIssues(cachedrepo *CachedRepo) error {
	issues, err := g.GetIssues(cachedrepo.Repo.PID, cachedrepo.Repo.Name)
	if err != nil {
		return err
	}

	for _, issue := range issues {
		cacheissue := &CachedIssue{Issue: issue, Comments: make(map[time.Time]*GitIssueComment)}
		comments, err := g.GetComments(cachedrepo.Repo.PID, issue.Number, cachedrepo.Repo.Name)
		if err != nil {
			return err
		}

		wg := sync.WaitGroup{}

		for _, comment := range comments {
			wg.Add(1)
			go func(comment *GitIssueComment) {
				defer wg.Done()

				cacheissue.commentMu.Lock()
				cacheissue.Comments[comment.CreatedAt] = comment
				cacheissue.commentMu.Unlock()
			}(comment)
		}
		wg.Wait()
		cachedrepo.Issues[issue.Title] = cacheissue
	}

	return nil
}

func (g *GithubProvider) NewCachedIssue(issue *GitIssue) *CachedIssue {
	return &CachedIssue{
		Issue:    issue,
		Comments: make(map[time.Time]*GitIssueComment),
	}
}

func (g *GithubProvider) fillLabels(cachedrepo *CachedRepo) error {
	labels, err := g.GetLabels(cachedrepo.Repo.PID, cachedrepo.Repo.Name)
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}

	for _, label := range labels {
		wg.Add(1)
		go func(label *GitLabel) {
			defer wg.Done()

			cachedrepo.labelMu.Lock()
			cachedrepo.Labels[label.Name] = label
			cachedrepo.labelMu.Unlock()
		}(label)
	}
	wg.Wait()
	return nil
}

// TODO: Swap out with regular maps protected by RWMutexes
func (g *GithubProvider) PrintCache() {
	for k, v := range g.Repocache {
		fmt.Printf("Print Repo - Key: %v, Value: %v\n", k, v.Repo.CloneURL)

		for key, val := range v.Issues {
			fmt.Printf("Print Issue - Key: %v, Value: %v\n", key, val.Issue.Body)

			for _, j := range val.Comments {
				fmt.Printf("Print Comment - Key: _, Value: %v\n", j.Body)
			}
		}

		for key, val := range v.Labels {
			fmt.Printf("Print Label - Key: %v, Value: %v\n", key, val.Name)
		}
	}
}

func (g *GithubProvider) depaginate(closure func(opts github.ListOptions) (*github.Response, error)) (*github.Response, error) {
	var response *github.Response
	var err error

	opts := github.ListOptions{
		Page:    1,
		PerPage: 100,
	}

	for {
		response, err = closure(opts)
		if err != nil || response.NextPage == 0 {
			break
		}
		opts.Page = response.NextPage
	}

	return response, err
}
