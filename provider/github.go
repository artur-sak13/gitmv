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

	"github.com/artur-sak13/gitmv/auth"

	"github.com/google/go-github/v21/github"
	"golang.org/x/oauth2"
)

// TODO: Make sure to retry on failure and attempt to "sync" updates between runs
// GitHubProvider implements the provider interface for GitHub
type GithubProvider struct {
	Client  *github.Client
	Context context.Context
	ID      *auth.ID

	retries int
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
		Client:  client,
		Context: ctx,
		ID:      id,
		retries: 5,
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
	status, err := g.GetImportProgress(repo.Name)
	if err == nil {
		return status, nil
	}
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
	// TODO: Implement
	return nil, fmt.Errorf("github GetRepositories not implemented")
}

// GetIssues retrieves a list of issues associated with a GitHub repository
func (g *GithubProvider) GetIssues(pid int, repo string) ([]*GitIssue, error) {
	// TODO: Implement
	return nil, fmt.Errorf("github GetIssues not implemented")
}

// GetComments retrieves a list of issue comments associated with a GitHub issue
func (g *GithubProvider) GetComments(pid, issueNum int, repo string) ([]*GitIssueComment, error) {
	// TODO: Implement
	return nil, fmt.Errorf("github GetComments not implemented")
}

// GetLabels retrieves a list of labels associated with a GitHub repository
func (g *GithubProvider) GetLabels(pid int, repo string) ([]*GitLabel, error) {
	// TODO: Implement
	return nil, fmt.Errorf("github GetLabels not implemented")
}
