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

type GitlabProvider struct {
	Client  *gitlab.Client
	Context context.Context
	ID      *auth.ID
	DryRun  bool
}

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
	result, err := getRepositories(g.Client)
	if err != nil {
		return nil, err
	}

	var repos []*GitRepository
	for _, project := range result {
		repos = append(repos, fromGitlabProject(project))
	}
	return repos, nil
}

func getRepositories(c *gitlab.Client) ([]*gitlab.Project, error) {
	opts := &gitlab.ListProjectsOptions{
		Statistics: gitlab.Bool(true),
		ListOptions: gitlab.ListOptions{
			Page:    1,
			PerPage: 100,
		},
	}
	var list []*gitlab.Project

	for {
		projects, resp, err := c.Projects.ListProjects(opts)

		if err != nil {
			return list, err
		}

		list = append(list, projects...)

		if lastPage(resp) {
			break
		}

		opts.Page = resp.NextPage
	}
	filtered := filter(list, func(proj *gitlab.Project) bool {
		return proj.Statistics.CommitCount != 0 && proj.ForkedFromProject == nil
	})
	return filtered, nil

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
	result, err := getIssues(g.Client, pid)
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

func getIssues(c *gitlab.Client, pid int) ([]*gitlab.Issue, error) {
	opts := gitlab.ListProjectIssuesOptions{
		ListOptions: gitlab.ListOptions{
			Page:    1,
			PerPage: 100,
		},
	}
	var list []*gitlab.Issue

	for {
		issues, resp, err := c.Issues.ListProjectIssues(pid, &opts)
		if err != nil {
			return list, err
		}

		list = append(list, issues...)

		if lastPage(resp) {
			break
		}

		opts.Page = resp.NextPage
	}
	return list, nil
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
	opts := &gitlab.ListIssueNotesOptions{
		ListOptions: gitlab.ListOptions{
			Page:    1,
			PerPage: 100,
		},
	}

	var list []*gitlab.Note

	for {
		notes, resp, err := g.Client.Notes.ListIssueNotes(pid, issueNum, opts, gitlab.WithSudo(2))
		if err != nil {
			return fromGitlabComments(list), err
		}
		list = append(list, notes...)
		if lastPage(resp) {
			break
		}
		opts.Page = resp.NextPage
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
	opts := gitlab.ListLabelsOptions{
		Page:    1,
		PerPage: 100,
	}

	var list []*gitlab.Label

	for {
		labels, resp, err := g.Client.Labels.ListLabels(pid, &opts)
		if err != nil {
			return fromGitlabLabels(list), err
		}

		list = append(list, labels...)

		if lastPage(resp) {
			break
		}

		opts.Page = resp.NextPage
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

// func (c *GitlabProvider) depaginate(opts *gitlab.ListOptions, call func() ([]interface{}, *gitlab.Response, error)) ([]interface{}, error) {
// 	var list []interface{}
// 	wrapper := func() (*gitlab.Response, error) {
// 		items, resp, err := call()
// 		if err == nil {
// 			list = append(list, items...)
// 		}
// 		return resp, err
// 	}

// 	opts.Page = 1
// 	opts.PerPage = 100
// 	for {
// 		resp, err := wrapper()
// 		if err != nil {
// 			return list, fmt.Errorf("error while depaginating page %d/%d: %v", opts.Page, resp.PreviousPage, err)
// 		}
// 		if lastPage(resp) {
// 			break
// 		}
// 		opts.Page = resp.NextPage
// 	}
// 	return list, nil
// }

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

func (g *GitlabProvider) CreateRepository(name, description string) (*GitRepository, error) {
	// TODO: Implement
	return nil, nil
}

func (g *GitlabProvider) CreateIssue(repo string, issue *GitIssue) (*GitIssue, error) {
	// TODO: Implement
	return nil, nil
}

func (g *GitlabProvider) CreateIssueComment(repo string, number int, comment *GitIssueComment) error {
	// TODO: Implement
	return nil
}
