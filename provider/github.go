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
	"sync"

	"github.com/artur-sak13/gitmv/auth"

	"github.com/sirupsen/logrus"

	"github.com/google/go-github/v21/github"
	"golang.org/x/oauth2"
)

type cachedIssue struct {
	issue    *GitIssue
	comments *sync.Map
}

type cachedRepo struct {
	repo   *GitRepository
	issues *sync.Map
	labels *sync.Map
}

// TODO: Sync changes between runs
// GitHubProvider implements the provider interface for GitHub
type GithubProvider struct {
	Client  *github.Client
	Context context.Context
	ID      *auth.ID

	retries   int
	repocache *sync.Map
}

// NewGithubProvider creates a new GitHub clients which implements the provider interface
func NewGithubProvider(ctx context.Context, id *auth.ID) (GitProvider, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: id.Token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	gh := WithGithubClient(ctx, client, id)
	if err := gh.(*GithubProvider).loadCache(); err != nil {
		return nil, err
	}
	return gh, nil
}

// WithGithubClient creates a new GitProvider with a GitHub client
func WithGithubClient(ctx context.Context, client *github.Client, id *auth.ID) GitProvider {
	return &GithubProvider{
		Client:    client,
		Context:   ctx,
		ID:        id,
		retries:   5,
		repocache: &sync.Map{},
	}
}

// CreateRepository creates a new GitHub repository
func (g *GithubProvider) CreateRepository(srcRepo *GitRepository) (*GitRepository, error) {
	repo := &github.Repository{
		Name:        github.String(srcRepo.Name),
		Private:     github.Bool(true),
		Description: github.String(srcRepo.Description),
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
		Title:     &issue.Title,
		Body:      &issue.Body,
		State:     &issue.State,
		Labels:    ToGitLabelStringSlice(issue.Labels),
		Assignees: UsersToString(issue.Assignees),
	}

	result, _, err := g.Client.Issues.Create(g.Context, g.ID.Owner, issue.Repo, issueRequest)
	if err != nil {
		return nil, err
	}

	number := 0
	if result.Number != nil {
		number = *result.Number
	}
	return fromGithubIssue(number, result), nil
}

func fromGithubIssue(number int, issue *github.Issue) *GitIssue {
	labels := []GitLabel{}
	for _, label := range issue.Labels {
		label := label // Pin to scope
		labels = append(labels, *fromGithubLabel(&label))
	}

	assignees := []GitUser{}

	for _, assignee := range issue.Assignees {
		assignees = append(assignees, *fromGithubUser(assignee))
	}

	return &GitIssue{
		Number:    number,
		State:     issue.GetState(),
		Title:     issue.GetTitle(),
		Body:      issue.GetBody(),
		Labels:    labels,
		User:      fromGithubUser(issue.User),
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
		User: &github.User{

			Email: &comment.User.Email,
		},
		Body:      &comment.Body,
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
		Name:        &srcLabel.Name,
		Color:       &srcLabel.Color,
		Description: &srcLabel.Description,
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
		VCSURL:      &repo.CloneURL,
		VCSUsername: &repo.Owner,
		VCSPassword: &token,
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

// GetAuthToken returns a string with a user's api authentication token
func (g *GithubProvider) GetAuth() *auth.ID {
	return g.ID
}

// GetRepositories retrieves a list of GitHub repositories for the organization/owner
func (g *GithubProvider) GetRepositories() ([]*GitRepository, error) {
	repoOpts := github.RepositoryListOptions{}
	var result []*github.Repository

	_, err := g.depaginate(func(opts github.ListOptions) (*github.Response, error) {
		repoOpts.ListOptions = opts

		repos, resp, err := g.Client.Repositories.List(g.Context, g.ID.Owner, &repoOpts)

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

func (g *GithubProvider) loadCache() error {
	repos, err := g.GetRepositories()
	if err != nil {
		return err
	}

	for _, repo := range repos {
		cachedrepo := &cachedRepo{repo: repo, issues: &sync.Map{}, labels: &sync.Map{}}
		issues, err := g.GetIssues(repo.PID, repo.Name)
		if err != nil {
			return err
		}
		for _, issue := range issues {
			cacheissue := &cachedIssue{issue: issue, comments: &sync.Map{}}
			comments, err := g.GetComments(repo.PID, issue.Number, repo.Name)
			if err != nil {
				return err
			}
			for _, comment := range comments {
				cacheissue.comments.Store(comment.CreatedAt, comment)
			}
			cachedrepo.issues.Store(issue.Number, cacheissue)
		}

		labels, err := g.GetLabels(repo.PID, repo.Name)
		if err != nil {
			return err
		}

		for _, label := range labels {
			cachedrepo.labels.Store(label.Name, label)
		}

		g.repocache.Store(repo.PID, cachedrepo)
	}
	return nil
}

// TODO: Swap out with regular maps protected by RWMutexes
func (g *GithubProvider) PrintCache() {
	g.repocache.Range(func(k, v interface{}) bool {
		fmt.Printf("Repo: %s, URL: %s", k.(string))

		v.(*cachedRepo).issues.Range(func(k, v interface{}) bool {
			logrus.WithFields(logrus.Fields{
				"IID":   k.(string),
				"issue": v.(*cachedIssue).issue.Title,
			})
			v.(*cachedIssue).comments.Range(func(k, v interface{}) bool {
				logrus.WithFields(logrus.Fields{
					"comment": v.(*GitIssueComment).Body,
				})
				return true
			})
			return true
		})

		v.(*cachedRepo).labels.Range(func(k, v interface{}) bool {
			logrus.WithFields(logrus.Fields{
				"label": k.(string),
				"color": v.(*GitLabel).Color,
			})
			return true
		})
		return true
	})
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
