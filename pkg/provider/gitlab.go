package provider

import (
	"context"
	"strings"

	gitlab "github.com/xanzy/go-gitlab"
)

type GitlabProvider struct {
	Client  *gitlab.Client
	Context context.Context

	token string
	URL   string
}

func NewGitlabProvider(url, gitlabToken string) (GitProvider, error) {
	client := gitlab.NewClient(nil, gitlabToken)
	if !IsHosted(url) {
		if err := client.SetBaseURL(url); err != nil {
			return nil, err
		}
	}
	return WithGitlabClient(url, gitlabToken, client)
}

func IsHosted(u string) bool {
	u = strings.TrimSuffix(u, "/")
	return u == "" || u == "https://gitlab.com" || u == "http://gitlab.com"
}

func WithGitlabClient(url, token string, client *gitlab.Client) (GitProvider, error) {
	return &GitlabProvider{
		Client: client,
		token:  token,
		URL:    url,
	}, nil
}

func (g *GitlabProvider) ListRepositories() ([]*GitRepository, error) {
	result, err := getRepositories(g.Client)
	if err != nil {
		return nil, err
	}

	var repos []*GitRepository
	for _, p := range result {
		repos = append(repos, fromGitlabProject(p))
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

func fromGitlabProject(p *gitlab.Project) *GitRepository {
	return &GitRepository{
		Name:     p.Name,
		HTMLURL:  p.WebURL,
		SSHURL:   p.SSHURLToRepo,
		CloneURL: p.HTTPURLToRepo,
		Fork:     p.ForkedFromProject != nil,
	}
}

func (g *GitlabProvider) GetIssues(pid int, org, repo string) ([]*GitIssue, error) {
	opts := gitlab.ListProjectIssuesOptions{
		ListOptions: gitlab.ListOptions{
			Page:    1,
			PerPage: 100,
		},
	}
	var list []*gitlab.Issue

	for {
		issues, resp, err := g.Client.Issues.ListProjectIssues(pid, &opts)
		if err != nil {
			return fromGitlabIssues(list, org, repo), err
		}

		list = append(list, issues...)

		if lastPage(resp) {
			break
		}

		opts.Page = resp.NextPage
	}
	return fromGitlabIssues(list, org, repo), nil
}

func fromGitlabIssues(issues []*gitlab.Issue, owner, repo string) []*GitIssue {
	var result []*GitIssue

	for _, v := range issues {
		result = append(result, fromGitlabIssue(v, owner, repo))
	}
	return result
}

func fromGitlabIssue(issue *gitlab.Issue, owner, repo string) *GitIssue {
	return &GitIssue{
		Number:    &issue.IID,
		Owner:     owner,
		Repo:      repo,
		Title:     issue.Title,
		Body:      issue.Description,
		Labels:    ToGitLabels(issue.Labels),
		State:     &issue.State,
		CreatedAt: issue.CreatedAt,
		UpdatedAt: issue.UpdatedAt,
		ClosedAt:  issue.ClosedAt,
	}

}

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
		CreatedAt: note.CreatedAt,
		UpdatedAt: note.UpdatedAt,
	}
}

func (g *GitlabProvider) GetLabels(pid int) ([]*gitlab.Label, error) {
	opts := gitlab.ListLabelsOptions{
		Page:    1,
		PerPage: 100,
	}

	var list []*gitlab.Label

	for {
		labels, resp, err := g.Client.Labels.ListLabels(pid, &opts)
		if err != nil {
			return list, err
		}

		list = append(list, labels...)

		if lastPage(resp) {
			break
		}

		opts.Page = resp.NextPage
	}
	return list, nil
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

func (g *GitlabProvider) Kind() string {
	return "gitlab"
}

func (g *GitlabProvider) IsGitHub() bool {
	return false
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
