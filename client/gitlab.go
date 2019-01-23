package client

import (
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

	gitlabClient struct {
		client    *gitlab.Client
		token     string
		customURL string
	}
)

// TODO: Check if cleaning the data is worth the additional iterations
// TODO: Handle group projects differently (map to GitHub teams)
func NewGitlabClient(customURL, gitlabToken string) (*gitlabClient, error) {
	client := gitlab.NewClient(nil, gitlabToken)
	if err := client.SetBaseURL(customURL); err != nil {
		return nil, err
	}
	return &gitlabClient{
		client:    client,
		token:     gitlabToken,
		customURL: customURL,
	}, nil
}

func (c *gitlabClient) GetProjects() ([]*gitlab.Project, error) {
	opts := &gitlab.ListProjectsOptions{
		Statistics: gitlab.Bool(true),
		ListOptions: gitlab.ListOptions{
			Page:    1,
			PerPage: 100,
		},
	}
	var list []*gitlab.Project

	for {
		projects, resp, err := c.client.Projects.ListProjects(opts)

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

func (c *gitlabClient) getIssues(pid int) ([]*gitlab.Issue, error) {
	opts := gitlab.ListProjectIssuesOptions{
		ListOptions: gitlab.ListOptions{
			Page:    1,
			PerPage: 100,
		},
	}
	var list []*gitlab.Issue

	for {
		issues, resp, err := c.client.Issues.ListProjectIssues(pid, &opts)
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

func (c *gitlabClient) issue(pid int, srcissue *gitlab.Issue) (*issue, error) {
	comments := []comment{}
	if srcissue.UserNotesCount != 0 {
		var err error
		comments, err = c.issueNotes(pid, srcissue.ID)
		if err != nil {
			return nil, err
		}
	}

	return &issue{
		number:       srcissue.ID,
		title:        srcissue.Title,
		numUserNotes: srcissue.UserNotesCount,
		description:  srcissue.Description,
		author:       srcissue.Author.Name,
		state:        srcissue.State,
		comments:     comments,
	}, nil

}

func (c *gitlabClient) issueNotes(pid, issueNum int) ([]comment, error) {
	opts := &gitlab.ListIssueNotesOptions{
		ListOptions: gitlab.ListOptions{
			Page:    1,
			PerPage: 100,
		},
	}

	var comments []comment

	for {
		notes, resp, err := c.client.Notes.ListIssueNotes(pid, issueNum, opts, gitlab.WithSudo(2))
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

func (c *gitlabClient) getMilestones(pid int) ([]*gitlab.Milestone, error) {
	opts := gitlab.ListMilestonesOptions{
		ListOptions: gitlab.ListOptions{
			Page:    1,
			PerPage: 100,
		},
	}

	var list []*gitlab.Milestone

	for {
		milestones, resp, err := c.client.Milestones.ListMilestones(pid, &opts)
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

func (c *gitlabClient) getLabels(pid int) ([]*gitlab.Label, error) {
	opts := gitlab.ListLabelsOptions{
		Page:    1,
		PerPage: 100,
	}

	var list []*gitlab.Label

	for {
		labels, resp, err := c.client.Labels.ListLabels(pid, &opts)
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

// func (c *gitlabClient) depaginate(opts *gitlab.ListOptions, call func() ([]interface{}, *gitlab.Response, error)) ([]interface{}, error) {
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
