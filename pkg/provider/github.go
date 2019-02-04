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
	"net/url"

	"github.com/artur-sak13/gitmv/pkg/auth"

	"github.com/google/go-github/v21/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type GithubProvider struct {
	Client  *github.Client
	Context context.Context
	ID      *auth.ID

	DryRun bool
}

func NewGithubProvider(ctx context.Context, id *auth.ID, dryRun bool) (GitProvider, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: id.Token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	return WithGithubClient(ctx, client, id, dryRun), nil
}

func WithGithubClient(ctx context.Context, client *github.Client, id *auth.ID, dryRun bool) GitProvider {
	return &GithubProvider{
		Client:  client,
		Context: ctx,
		ID:      id,
		DryRun:  dryRun,
	}
}

func (g *GithubProvider) CreateRepository(name, description string) (*GitRepository, error) {
	repo := &github.Repository{
		Name:        github.String(name),
		Private:     github.Bool(true),
		Description: github.String(description),
	}

	r, _, err := g.Client.Repositories.Create(g.Context, g.ID.Owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository %s/%s due to: %s", g.ID.Owner, name, err)
	}
	return fromGithubRepo(r), nil
}

func fromGithubRepo(repo *github.Repository) *GitRepository {
	return &GitRepository{
		Name:     repo.GetName(),
		CloneURL: repo.GetCloneURL(),
		HTMLURL:  repo.GetHTMLURL(),
		SSHURL:   repo.GetSSHURL(),
		Fork:     repo.GetFork(),
	}
}

func (g *GithubProvider) RepositoryExists(name string) bool {
	_, r, err := g.Client.Repositories.Get(g.Context, g.ID.Owner, name)
	if err == nil {
		return true
	}
	return r != nil && r.StatusCode == 404
}

func (g *GithubProvider) CreateIssue(repo string, issue *GitIssue) (*GitIssue, error) {
	issueRequest := &github.IssueRequest{
		Title:     &issue.Title,
		Body:      &issue.Body,
		Labels:    ToGitLabelStringSlice(issue.Labels),
		Assignees: UsersToString(issue.Assignees),
	}

	result, _, err := g.Client.Issues.Create(g.Context, g.ID.Owner, repo, issueRequest)
	if err != nil {
		return nil, err
	}

	number := 0
	if result.Number != nil {
		number = *result.Number
	}
	return fromGithubIssue(repo, number, result), nil
}

func fromGithubIssue(name string, number int, issue *github.Issue) *GitIssue {
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
		CreatedAt: issue.GetCreatedAt(),
		UpdatedAt: issue.GetUpdatedAt(),
		ClosedAt:  issue.GetClosedAt(),
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

func (g *GithubProvider) CreateIssueComment(repo string, number int, comment *GitIssueComment) error {
	issueComment := &github.IssueComment{
		User:      &github.User{Email: &comment.User.Email},
		Body:      &comment.Body,
		CreatedAt: &comment.CreatedAt,
		UpdatedAt: &comment.UpdatedAt,
	}
	_, _, err := g.Client.Issues.CreateComment(g.Context, g.ID.Owner, repo, number, issueComment)
	if err != nil {
		return err
	}
	return nil
}

func (g *GithubProvider) CreateLabel(repo string, srcLabel *GitLabel) (*GitLabel, error) {
	label := &github.Label{
		Name:        &srcLabel.Name,
		Color:       &srcLabel.Color,
		Description: &srcLabel.Description,
	}

	result, _, err := g.Client.Issues.CreateLabel(g.Context, g.ID.Owner, repo, label)
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

func (g *GithubProvider) MigrateRepo(repo *GitRepository, token string) error {
	u, err := url.Parse(repo.CloneURL)
	if err != nil {
		return fmt.Errorf("could not parse repo name into owner and repo %v", err)
	}

	// Must create repository before running import
	repoImport := &github.Import{
		VCS:         github.String("git"),
		VCSURL:      &repo.CloneURL,
		VCSUsername: &g.ID.Owner,
		VCSPassword: &token,
	}
	result, _, err := g.Client.Migrations.StartImport(g.Context, g.ID.Owner, u.RequestURI(), repoImport)
	if err != nil {
		return err
	}
	logrus.Infof("importing %s", *result.Status)
	return nil
}

func (g *GithubProvider) GetRepositories() ([]*GitRepository, error) {
	// TODO: Implement
	return nil, nil
}

func (g *GithubProvider) GetIssues(pid int, repo string) ([]*GitIssue, error) {
	// TODO: Implement
	return nil, nil
}

func (g *GithubProvider) GetComments(pid, issueNum int) ([]*GitIssueComment, error) {
	// TODO: Implement
	return nil, nil
}

func (g *GithubProvider) GetLabels(pid int) ([]*GitLabel, error) {
	// TODO: Implement
	return nil, nil
}
