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

	"github.com/artur-sak13/gitmv/auth"

	gitlab "github.com/xanzy/go-gitlab"
)

// GitlabProvider implements the provider interface for GitLab
type GitlabProvider struct {
	Client  *gitlab.Client
	Context context.Context
	ID      *auth.ID
}

type getterFn func(opts gitlab.ListOptions) (*gitlab.Response, error)

// NewGitlabProvider creates a new GitLab client which implements the provider interface
func NewGitlabProvider(id *auth.ID) (GitProvider, error) {
	client := gitlab.NewClient(nil, id.Token)
	if !IsHosted(id.URL) {
		if err := client.SetBaseURL(id.URL); err != nil {
			return nil, err
		}
	}
	return WithGitlabClient(client, id), nil
}

// IsHosted checks if the specified URL is a Git SaaS provider
func IsHosted(u string) bool {
	u = strings.TrimSuffix(u, "/")
	return u == "" || u == "https://gitlab.com" || u == "http://gitlab.com"
}

// WithGitlabClient creates a new GitProvider with a Gitlab client
// This function is exported to create mock clients in tests
func WithGitlabClient(client *gitlab.Client, id *auth.ID) GitProvider {
	return &GitlabProvider{
		Client: client,
		ID:     id,
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
	owner := ""
	if project.Owner != nil {
		owner = project.Owner.Username
	}
	return &GitRepository{
		Name:        project.Path,
		Description: project.Description,
		SSHURL:      project.SSHURLToRepo,
		Owner:       owner,
		Archived:    project.Archived,
		CloneURL:    project.HTTPURLToRepo,
		Fork:        project.ForkedFromProject != nil,
		Empty:       project.Statistics.CommitCount == 0,
		PID:         project.ID,
	}
}

// GetIssues retrieves a full list of Issues for a project
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
		gitissue.Repo = repo
		gitissue.PID = pid
		gitissue.User = g.GetUserByID(issue.Author.ID)
		gitissue.Assignees = g.getAssignees(issue.Assignees)

		issues = append(issues, gitissue)
	}

	return issues, nil
}

func fromGitlabIssue(issue *gitlab.Issue) *GitIssue {
	return &GitIssue{
		Number: issue.IID,
		Title:  issue.Title,
		Body:   issue.Description,
		State:  issue.State,
		Labels: ToGitLabels(issue.Labels),
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
func (g *GitlabProvider) GetComments(pid, issueNum int, repo string) ([]*GitIssueComment, error) {
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

	return fromGitlabComments(repo, issueNum, list), nil
}

func fromGitlabComments(repo string, issueNum int, notes []*gitlab.Note) []*GitIssueComment {
	var result []*GitIssueComment

	for _, note := range notes {
		result = append(result, fromGitlabComment(repo, issueNum, note))
	}

	return result
}

func fromGitlabComment(repo string, issueNum int, note *gitlab.Note) *GitIssueComment {
	return &GitIssueComment{
		Repo:     repo,
		IssueNum: issueNum,
		User: GitUser{
			Login: note.Author.Username,
			Name:  note.Author.Name,
			Email: note.Author.Email,
		},
		Body:      note.Body,
		CreatedAt: *note.CreatedAt,
		UpdatedAt: *note.UpdatedAt,
	}
}

// GetLabels retrieves a full list of labels associated with a project
func (g *GitlabProvider) GetLabels(pid int, repo string) ([]*GitLabel, error) {
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

	return fromGitlabLabels(repo, list), nil
}

// GetAuthToken returns a string with a user's api authentication token
func (g *GitlabProvider) GetAuth() *auth.ID {
	return g.ID
}

func fromGitlabLabels(repo string, labels []*gitlab.Label) []*GitLabel {
	var result []*GitLabel
	for _, label := range labels {
		result = append(result, fromGitlabLabel(repo, label))
	}
	return result
}

func fromGitlabLabel(repo string, label *gitlab.Label) *GitLabel {
	return &GitLabel{
		Repo:        repo,
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

func lastPage(resp *gitlab.Response) bool {
	return resp == nil || resp.CurrentPage >= resp.TotalPages || resp.NextPage == 0
}

// CreateRepository creates a new GitLab repository
func (g *GitlabProvider) CreateRepository(repo *GitRepository) (*GitRepository, error) {
	// TODO: Implement
	return nil, fmt.Errorf("gitlab CreateRepository not implemented")
}

// MigrateRepo migrates a git repo from an existing provider
func (g *GitlabProvider) MigrateRepo(repo *GitRepository, token string) (string, error) {
	return "", fmt.Errorf("gitlab MigrateRepo not implemented")
}

func (g *GitlabProvider) GetImportProgress(repo string) (string, error) {
	return "", fmt.Errorf("gitlab GetImportProgress not implemented")
}

// CreateIssue creates a new GitLab issue
func (g *GitlabProvider) CreateIssue(issue *GitIssue) (*GitIssue, error) {
	// TODO: Implement
	return nil, fmt.Errorf("gitlab CreateIssue not implemented")
}

// CreateIssueComment creates a new GitLab issue note/comment
func (g *GitlabProvider) CreateIssueComment(issueNum int, comment *GitIssueComment) error {
	// TODO: Implement
	return fmt.Errorf("gitlab CreateIssueComment not implemented")
}

// CreateLabel creates a new GitLab issue label
func (g *GitlabProvider) CreateLabel(label *GitLabel) (*GitLabel, error) {
	// TODO: Implement
	return nil, fmt.Errorf("gitlab CreateLabel not implemented")
}
