package client

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/go-github/v21/github"
	"github.com/sirupsen/logrus"
	gitlab "github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"
)

type (
	issueService interface {
		Create(ctx context.Context, owner string, repo string, issue *github.IssueRequest) (*github.Issue, *github.Response, error)
		CreateComment(ctx context.Context, owner, repo string, number int, comment *github.IssueComment) (*github.IssueComment, *github.Response, error)
		CreateLabel(ctx context.Context, owner, repo string, label *github.Label) (*github.Label, *github.Response, error)
		CreateMilestone(ctx context.Context, owner, repo string, milestone *github.Milestone) (*github.Milestone, *github.Response, error)
	}

	repositoryService interface {
		Create(ctx context.Context, org string, repo *github.Repository) (*github.Repository, *github.Response, error)
	}

	migrationService interface {
		StartImport(ctx context.Context, owner, repo string, in *github.Import) (*github.Import, *github.Response, error)
	}

	teamsService interface {
		CreateTeam(ctx context.Context, org string, team github.NewTeam) (*github.Team, *github.Response, error)
	}

	GitHubProvider struct {
		issueService      issueService
		repositoryService repositoryService
		migrationService  migrationService
		client            *github.Client
		Context           context.Context
		dryRun            bool
		org               string
	}
)

// TODO: See if groups should be mapped to teams or if all projects should be in org namespace
func NewGitHubProvider(ctx context.Context, org, githubToken string, dryRun bool) *GitHubProvider {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	return &GitHubProvider{
		issueService:      client.Issues,
		repositoryService: client.Repositories,
		migrationService:  client.Migrations,
		client:            client,
		Context:           ctx,
		dryRun:            dryRun,
		org:               org,
	}
}

func (c *GitHubProvider) CreateRepository(glproject *gitlab.Project) (*GitRepository, error) {
	repo := &github.Repository{
		Name:        github.String(glproject.Name),
		Private:     github.Bool(true),
		Description: github.String(glproject.Description),
	}

	r, _, err := c.repositoryService.Create(c.Context, c.org, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository %s/%s due to: %s", c.org, glproject.Name, err)
	}
	return fromGithubRepo(r), nil
}

func fromGithubRepo(repo *github.Repository) *GitRepository {
	return &GitRepository{
		Name:     *repo.Name,
		CloneURL: *repo.CloneURL,
		HTMLURL:  *repo.HTMLURL,
		SSHURL:   *repo.SSHURL,
		Fork:     *repo.Fork,
	}
}

func (c *GitHubProvider) RepositoryExists(ctx context.Context, org, name string) bool {
	_, r, err := c.client.Repositories.Get(ctx, org, name)
	if err == nil {
		return true
	}
	return r != nil && r.StatusCode == 404
}

func (c *GitHubProvider) createIssue(ctx context.Context, repo string, glIssue *gitlab.Issue, assignees []string) (*github.Issue, error) {
	issue := &github.IssueRequest{
		Title: &glIssue.Title,
		Body:  &glIssue.Description,
	}
	if len(glIssue.Labels) > 0 {
		issue.Labels = &glIssue.Labels
	}
	if len(assignees) > 0 {
		issue.Assignees = &assignees
	}
	result, _, err := c.issueService.Create(ctx, c.org, repo, issue)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *GitHubProvider) fromGithubIssue(org, name string, number int, i *github.Issue) (*GitIssue, error) {
	isPull := i.IsPullRequest()

	labels := []GitLabel{}
	for _, label := range i.Labels {
		label := label // Pin to scope
		labels = append(labels, fromGithubLabel(&label))
	}

	assignees := []GitUser{}

	for _, assignee := range i.Assignees {
		assignees = append(assignees, *fromGithubUser(assignee))
	}

	return &GitIssue{
		Number:        &number,
		State:         i.State,
		Title:         *i.Title,
		Body:          *i.Body,
		IsPullRequest: isPull,
		Labels:        labels,
		User:          fromGithubUser(i.User),
		CreatedAt:     i.CreatedAt,
		UpdatedAt:     i.UpdatedAt,
		ClosedAt:      i.ClosedAt,
		ClosedBy:      fromGithubUser(i.ClosedBy),
		Assignees:     assignees,
	}, nil
}

func fromGithubUser(user *github.User) *GitUser {
	if user == nil {
		return nil
	}
	return &GitUser{
		Login:     *user.Login,
		Name:      *user.Name,
		Email:     *user.Email,
		AvatarURL: *user.AvatarURL,
	}
}

func fromGithubLabel(label *github.Label) GitLabel {
	return GitLabel{
		Name:  *label.Name,
		Color: *label.Color,
		URL:   *label.URL,
	}
}

func (c *GitHubProvider) createIssueComment(ctx context.Context, repo string, number int, note *gitlab.Note) (*github.IssueComment, error) {
	comment := &github.IssueComment{
		Body:      &note.Body,
		CreatedAt: note.CreatedAt,
		UpdatedAt: note.UpdatedAt,
	}

	result, _, err := c.issueService.CreateComment(ctx, c.org, repo, number, comment)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *GitHubProvider) createLabel(ctx context.Context, repo string, glLabel *gitlab.Label) (*github.Label, error) {
	label := &github.Label{
		Name:        &glLabel.Name,
		Color:       &glLabel.Color,
		Description: &glLabel.Description,
	}

	result, _, err := c.issueService.CreateLabel(ctx, c.org, repo, label)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *GitHubProvider) createMilestone(ctx context.Context, repo string, glMilestone *gitlab.Milestone) (*github.Milestone, error) {
	var isotime time.Time = time.Time(*glMilestone.DueDate)

	milestone := &github.Milestone{
		Title:       &glMilestone.Title,
		Description: &glMilestone.Description,
		State:       &glMilestone.State,
		CreatedAt:   glMilestone.CreatedAt,
		UpdatedAt:   glMilestone.UpdatedAt,
		DueOn:       &isotime,
	}

	result, _, err := c.issueService.CreateMilestone(ctx, c.org, repo, milestone)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *GitHubProvider) migrateRepo(ctx context.Context, org, gitlabToken, sourcerepo string) error {
	u, err := url.Parse(sourcerepo)
	if err != nil {
		return fmt.Errorf("could not parse repo name into owner and repo %v", err)
	}

	// Must create repository before running import
	im := &github.Import{
		VCSURL:      github.String(sourcerepo),
		VCS:         github.String("git"),
		VCSUsername: github.String(strings.Split(u.RequestURI(), "/")[1]),
		VCSPassword: github.String(gitlabToken),
	}
	imprt, _, err := c.migrationService.StartImport(ctx, org, u.RequestURI(), im)
	if err != nil {
		return err
	}
	logrus.Infof("Importing %s", *imprt.Status)
	return nil
}
