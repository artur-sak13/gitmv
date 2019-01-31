package client

import (
	"context"
	"strings"
	"time"

	gitlab "github.com/xanzy/go-gitlab"
)

type (
	project struct {
		owner               string
		name                string
		path_with_namespace string
		description         string
		// "user or group project"
		kind string
	}

	group struct {
		name        string
		description string
		reponames   []string
	}

	comment struct {
		author  string
		body    string
		created time.Time
	}

	issue struct {
		number       int
		numUserNotes int
		title        string
		description  string
		author       string
		state        string
		comments     []comment
	}

	milestone struct {
		title       string
		description string
		state       string
		startdate   time.Time
		duedate     time.Time
	}

	label struct {
		name        string
		color       string
		description string
	}

	projectData struct {
		project    project
		issues     []issue
		milestones []milestone
		labels     []label
	}

	GitlabProvider struct {
		Client  *gitlab.Client
		Context context.Context

		token string
		URL   string
	}
)

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

func filter(projects []*gitlab.Project, fn func(p *gitlab.Project) bool) []*gitlab.Project {
	prjs := []*gitlab.Project{}
	for _, project := range projects {
		if fn(project) {
			prjs = append(prjs, project)
		}
	}
	return prjs
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

// func (g *GitlabProvider) issue(pid int, srcissue *gitlab.Issue) (*issue, error) {
// 	comments := []comment{}
// 	if srcissue.UserNotesCount != 0 {
// 		var err error
// 		comments, err = g.issueNotes(pid, srcissue.ID)
// 		if err != nil {
// 			return nil, err
// 		}
// 	}

// 	return &issue{
// 		number:       srcissue.ID,
// 		title:        srcissue.Title,
// 		numUserNotes: srcissue.UserNotesCount,
// 		description:  srcissue.Description,
// 		author:       srcissue.Author.Name,
// 		state:        srcissue.State,
// 		comments:     comments,
// 	}, nil

// }

func fromGitlabIssues(issues []*gitlab.Issue, owner, repo string) []*GitIssue {
	var result []*GitIssue

	for _, v := range issues {
		result = append(result, fromGitlabIssue(v, owner, repo))
	}
	return result
}

func fromGitlabIssue(issue *gitlab.Issue, owner, repo string) *GitIssue {
	var labels []GitLabel
	for _, v := range issue.Labels {
		labels = append(labels, GitLabel{Name: v})
	}
	return &GitIssue{
		Number:    &issue.IID,
		Owner:     owner,
		Repo:      repo,
		Title:     issue.Title,
		Body:      issue.Description,
		Labels:    labels,
		CreatedAt: issue.CreatedAt,
		UpdatedAt: issue.UpdatedAt,
		ClosedAt:  issue.ClosedAt,
	}

}

func (g *GitlabProvider) issueNotes(pid, issueNum int) ([]comment, error) {
	opts := &gitlab.ListIssueNotesOptions{
		ListOptions: gitlab.ListOptions{
			Page:    1,
			PerPage: 100,
		},
	}

	var comments []comment

	for {
		notes, resp, err := g.Client.Notes.ListIssueNotes(pid, issueNum, opts, gitlab.WithSudo(2))
		if err != nil {
			return comments, err
		}
		comments = append(comments, toComments(notes)...)
		if lastPage(resp) {
			break
		}
		opts.Page = resp.NextPage
	}
	return comments, nil
}

func toComments(notes []*gitlab.Note) []comment {
	comments := []comment{}

	for _, note := range notes {
		c := comment{
			author:  note.Author.Email,
			body:    note.Body,
			created: *note.CreatedAt,
		}
		comments = append(comments, c)
	}

	return comments
}

func (g *GitlabProvider) getMilestones(pid int) ([]*gitlab.Milestone, error) {
	opts := gitlab.ListMilestonesOptions{
		ListOptions: gitlab.ListOptions{
			Page:    1,
			PerPage: 100,
		},
	}

	var list []*gitlab.Milestone

	for {
		milestones, resp, err := g.Client.Milestones.ListMilestones(pid, &opts)
		if err != nil {
			return list, err
		}
		list = append(list, milestones...)

		if lastPage(resp) {
			break
		}

		opts.Page = resp.NextPage
	}
	return list, nil
}

func (g *GitlabProvider) getLabels(pid int) ([]*gitlab.Label, error) {
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
func lastPage(resp *gitlab.Response) bool {
	return resp == nil || resp.CurrentPage >= resp.TotalPages || resp.NextPage == 0
}
