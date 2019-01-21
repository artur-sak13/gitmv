package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v21/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type issueService interface {
	Create(ctx context.Context, owner string, repo string, issue *github.IssueRequest) (*github.Issue, *github.Response, error)
	CreateComment(ctx context.Context, owner, repo string, number int, comment *github.IssueComment) (*github.IssueComment, *github.Response, error)
	CreateLabel(ctx context.Context, owner, repo string, label *github.Label) (*github.Label, *github.Response, error)
	CreateMilestone(ctx context.Context, owner, repo string, milestone *github.Milestone) (*github.Milestone, *github.Response, error)
}

type repositoryService interface {
	Create(ctx context.Context, org string, repo *github.Repository) (*github.Repository, *github.Response, error)
}

type migrationService interface {
	StartImport(ctx context.Context, owner, repo string, in *github.Import) (*github.Import, *github.Response, error)
}

type githubClient struct {
	issueService      issueService
	repositoryService repositoryService
	migrationService  migrationService
	dryRun            bool
}

func NewGitHubClient(ctx context.Context, githubToken string, dryRun bool) *githubClient {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	return &githubClient{
		issueService:      client.Issues,
		repositoryService: client.Repositories,
		migrationService:  client.Migrations,
		dryRun:            dryRun,
	}
}

func (c *githubClient) newRepo(ctx context.Context, name, org, description string) (*github.Repository, error) {
	repo := &github.Repository{
		Name:        github.String(name),
		Private:     github.Bool(true),
		Description: github.String(description),
	}
	r, _, err := c.repositoryService.Create(ctx, org, repo)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (c *githubClient) createIssue(ctx context.Context, org, repo, title, body string, labels, assignees []string) (*github.Issue, error) {
	issue := &github.IssueRequest{
		Title: &title,
		Body:  &body,
	}
	if len(labels) > 0 {
		issue.Labels = &labels
	}
	if len(assignees) > 0 {
		issue.Assignees = &assignees
	}
	result, _, err := c.issueService.Create(ctx, org, repo, issue)
	if err != nil {
		return nil, err
	}
	return result, nil

}

func (c *githubClient) migrateRepo(ctx context.Context, gitlabToken, sourcerepo string) error {
	parts := strings.SplitN(sourcerepo, "/", 5)
	if len(parts) < 5 {
		return fmt.Errorf("could not parse repo name into owner and repo for %s, got: %#v", sourcerepo, parts)
	}

	// Must create repository before running import
	im := &github.Import{
		VCSURL:      github.String(sourcerepo),
		VCS:         github.String("git"),
		VCSUsername: github.String(parts[3]),
		VCSPassword: github.String(gitlabToken),
	}
	imprt, _, err := c.migrationService.StartImport(ctx, "twopt", parts[4], im)
	if err != nil {
		return err
	}
	logrus.Infof("Importing %s", *imprt.Status)
	return nil
}
