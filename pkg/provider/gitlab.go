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
	"strings"

	"github.com/artur-sak13/gitmv/pkg/auth"

	gitlab "github.com/xanzy/go-gitlab"
)

// GitlabProvider
type GitlabProvider struct {
	Client  *gitlab.Client
	Context context.Context
	ID      *auth.ID
	DryRun  bool
}

type getterFn func(opts gitlab.ListOptions) (*gitlab.Response, error)

// NewGitlabProvider creates a new GitLab client which implements the provider interface
func NewGitlabProvider(id *auth.ID, dryRun bool) (GitProvider, error) {
	client := gitlab.NewClient(nil, id.Token)
	if !IsHosted(id.URL) {
		if err := client.SetBaseURL(id.URL); err != nil {
			return nil, err
		}
	}
	return WithGitlabClient(id, client, dryRun), nil
}

// IsHosted checks if the specified URL is a Git SaaS provider
func IsHosted(u string) bool {
	u = strings.TrimSuffix(u, "/")
	return u == "" || u == "https://gitlab.com" || u == "http://gitlab.com"
}

// WithGitlabClient creates a new GitProvider with a Gitlab client
// This function is exported to create mock clients in tests
func WithGitlabClient(id *auth.ID, client *gitlab.Client, dryRun bool) GitProvider {
	return &GitlabProvider{
		Client: client,
		ID:     id,
		DryRun: dryRun,
	}
}

// GetRepositories gets a list of all repositories in the target Gitlab instance
// For >100 repositories this _depaginates_ the responses and appends them to one slice
func (g *GitlabProvider) GetRepositories() ([]*GitRepository, error) {
	var result []*gitlab.Project
	projectOpts := gitlab.ListProjectsOptions{Statistics: gitlab.Bool(true)}

	_, err := depaginate(func(opts gitlab.ListOptions) (*gitlab.Response, error) {
		projectOpts.ListOptions = opts

		projects, resp, err := g.Client.Projects.ListProjects(&projectOpts)

		result = append(result, projects...)
		return resp, err
	})

	if err != nil {
		return nil, err
	}

	var repos []*GitRepository
	for _, project := range result {
		repos = append(repos, fromGitlabProject(project))
	}
	return repos, nil
}

func fromGitlabProject(project *gitlab.Project) *GitRepository {
	return &GitRepository{
		Name:     project.Name,
		HTMLURL:  project.WebURL,
		SSHURL:   project.SSHURLToRepo,
		CloneURL: project.HTTPURLToRepo,
		Fork:     project.ForkedFromProject != nil,
		PID:      project.ID,
	}
}

// GetComments retrieves a full list of Issues for a project
// For >100 issues this _depaginates_ the responses and appends them to one slice
func (g *GitlabProvider) GetIssues(pid int, repo string) ([]*GitIssue, error) {
	var result []*gitlab.Issue
	issueOpts := gitlab.ListProjectIssuesOptions{}

	_, err := depaginate(func(opts gitlab.ListOptions) (*gitlab.Response, error) {
		issueOpts.ListOptions = opts

		issues, resp, err := g.Client.Issues.ListProjectIssues(pid, &issueOpts)

		result = append(result, issues...)
		return resp, err
	})

	if err != nil {
		return nil, err
	}

	var issues []*GitIssue

	for _, issue := range result {
		gitissue := fromGitlabIssue(issue)
		gitissue.Owner = g.ID.Owner
		gitissue.Repo = repo
		gitissue.User = g.GetUserByID(issue.Author.ID)
		gitissue.Assignees = g.getAssignees(issue.Assignees)

		issues = append(issues, gitissue)
	}

	return issues, nil
}

func fromGitlabIssue(issue *gitlab.Issue) *GitIssue {
	return &GitIssue{
		Number:    issue.IID,
		Title:     issue.Title,
		Body:      issue.Description,
		State:     issue.State,
		Labels:    ToGitLabels(issue.Labels),
		CreatedAt: *issue.CreatedAt,
		UpdatedAt: *issue.UpdatedAt,
		ClosedAt:  *issue.ClosedAt,
	}
}

func (g *GitlabProvider) getAssignees(assignees []*gitlab.IssueAssignee) []GitUser {
	users := []GitUser{}
	for _, assignee := range assignees {
		user := g.GetUserByID(assignee.ID)
		if user != nil {
			users = append(users, *user)
		}

	}
	return users
}

// GetUserByID looks up a user by ID and lifts them to the GitUser type
func (g *GitlabProvider) GetUserByID(uid int) *GitUser {
	user, _, err := g.Client.Users.GetUser(uid, gitlab.WithSudo(2))
	if err != nil {
		return nil
	}
	return fromGitlabUser(user)
}

func fromGitlabUser(user *gitlab.User) *GitUser {
	if user == nil {
		return nil
	}
	return &GitUser{
		Login: user.Username,
		Name:  user.Name,
		Email: user.Email,
	}
}

// GetComments retrieves a full list of comments for a project issue
// For >100 comments this _depaginates_ the responses and appends them to one slice
func (g *GitlabProvider) GetComments(pid, issueNum int) ([]*GitIssueComment, error) {
	var list []*gitlab.Note
	noteOpts := gitlab.ListIssueNotesOptions{}

	_, err := depaginate(func(opts gitlab.ListOptions) (*gitlab.Response, error) {
		noteOpts.ListOptions = opts

		notes, resp, err := g.Client.Notes.ListIssueNotes(pid, issueNum, &noteOpts, gitlab.WithSudo(2))

		list = append(list, notes...)
		return resp, err
	})

	if err != nil {
		return nil, err
	}

	return fromGitlabComments(list), nil
}

func fromGitlabComments(notes []*gitlab.Note) []*GitIssueComment {
	var result []*GitIssueComment

	for _, note := range notes {
		result = append(result, fromGitlabComment(note))
	}

	return result
}

func fromGitlabComment(note *gitlab.Note) *GitIssueComment {
	return &GitIssueComment{
		User: GitUser{
			Email: note.Author.Email,
		},
		Body:      note.Body,
		CreatedAt: *note.CreatedAt,
		UpdatedAt: *note.UpdatedAt,
	}
}

// GetLabels retrieves a full list of labels associated with a project
func (g *GitlabProvider) GetLabels(pid int) ([]*GitLabel, error) {
	var list []*gitlab.Label
	var labelOpts gitlab.ListLabelsOptions

	_, err := depaginate(func(opts gitlab.ListOptions) (*gitlab.Response, error) {
		labelOpts = gitlab.ListLabelsOptions(opts)

		issues, resp, err := g.Client.Labels.ListLabels(pid, &labelOpts)

		list = append(list, issues...)
		return resp, err
	})

	if err != nil {
		return nil, err
	}

	return fromGitlabLabels(list), nil
}

func fromGitlabLabels(labels []*gitlab.Label) []*GitLabel {
	var result []*GitLabel
	for _, label := range labels {
		result = append(result, fromGitlabLabel(label))
	}
	return result
}

func fromGitlabLabel(label *gitlab.Label) *GitLabel {
	return &GitLabel{
		Name:        label.Name,
		Color:       label.Color,
		Description: label.Description,
	}
}

func depaginate(closure getterFn) (*gitlab.Response, error) {
	var resp *gitlab.Response
	var err error

	opts := gitlab.ListOptions{
		Page:    1,
		PerPage: 100,
	}

	for {
		resp, err = closure(opts)
		if err != nil || lastPage(resp) {
			break
		}
		opts.Page = resp.NextPage
	}
	return resp, err
}

func filter(projects []*gitlab.Project, fn func(p *gitlab.Project) bool) []*gitlab.Project {
	projs := []*gitlab.Project{}
	for _, project := range projects {
		if fn(project) {
			projs = append(projs, project)
		}
	}
	return projs
}

func lastPage(resp *gitlab.Response) bool {
	return resp == nil || resp.CurrentPage >= resp.TotalPages || resp.NextPage == 0
}

// CreateRepository
func (g *GitlabProvider) CreateRepository(name, description string) (*GitRepository, error) {
	// TODO: Implement
	return nil, nil
}

// CreateIssue
func (g *GitlabProvider) CreateIssue(repo string, issue *GitIssue) (*GitIssue, error) {
	// TODO: Implement
	return nil, nil
}

// CreateIssueComment
func (g *GitlabProvider) CreateIssueComment(repo string, number int, comment *GitIssueComment) error {
	// TODO: Implement
	return nil
}
