package provider

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/go-github/v21/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type GitHubProvider struct {
	Client  *github.Client
	Context context.Context
	dryRun  bool
	org     string
}

func NewGitHubProvider(ctx context.Context, org, githubToken string, dryRun bool) *GitHubProvider {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	return &GitHubProvider{
		Client:  client,
		Context: ctx,
		dryRun:  dryRun,
		org:     org,
	}
}

func (c *GitHubProvider) CreateRepository(name, description string) (*GitRepository, error) {
	repo := &github.Repository{
		Name:        github.String(name),
		Private:     github.Bool(true),
		Description: github.String(description),
	}

	r, _, err := c.Client.Repositories.Create(c.Context, c.org, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository %s/%s due to: %s", c.org, name, err)
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
	_, r, err := c.Client.Repositories.Get(ctx, org, name)
	if err == nil {
		return true
	}
	return r != nil && r.StatusCode == 404
}

func (c *GitHubProvider) CreateIssue(repo string, glIssue *GitIssue) (*GitIssue, error) {
	labels := []string{}
	for _, label := range glIssue.Labels {
		name := label.Name
		if name != "" {
			labels = append(labels, name)
		}
	}
	issue := &github.IssueRequest{
		Title:     &glIssue.Title,
		Body:      &glIssue.Body,
		Labels:    &labels,
		Assignees: usersToString(glIssue.Assignees),
	}

	result, _, err := c.Client.Issues.Create(c.Context, c.org, repo, issue)
	if err != nil {
		return nil, err
	}

	number := 0
	if result.Number != nil {
		number = *result.Number
	}
	return c.fromGithubIssue(c.org, repo, number, result)
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

func usersToString(users []GitUser) *[]string {
	var result []string
	for _, user := range users {
		result = append(result, user.Login)
	}
	return &result
}

func fromGithubLabel(label *github.Label) GitLabel {
	return GitLabel{
		Name:        *label.Name,
		Color:       *label.Color,
		Description: *label.Description,
	}
}

func (c *GitHubProvider) CreateIssueComment(repo string, number int, comment *GitIssueComment) error {
	issueComment := &github.IssueComment{
		User:      &github.User{Email: &comment.User.Email},
		Body:      &comment.Body,
		CreatedAt: comment.CreatedAt,
		UpdatedAt: comment.UpdatedAt,
	}
	_, _, err := c.Client.Issues.CreateComment(c.Context, c.org, repo, number, issueComment)
	if err != nil {
		return err
	}
	return nil
}

func (c *GitHubProvider) CreateLabel(ctx context.Context, repo string, glLabel *GitLabel) (*github.Label, error) {
	label := &github.Label{
		Name:        &glLabel.Name,
		Color:       &glLabel.Color,
		Description: &glLabel.Description,
	}

	result, _, err := c.Client.Issues.CreateLabel(ctx, c.org, repo, label)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *GitHubProvider) MigrateRepo(gitlabToken, sourcerepo string) error {
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
	result, _, err := c.Client.Migrations.StartImport(c.Context, c.org, u.RequestURI(), im)
	if err != nil {
		return err
	}
	logrus.Infof("Importing %s", *result.Status)
	return nil
}
