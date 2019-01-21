package client

import (
	"time"

	gitlab "github.com/xanzy/go-gitlab"
)

type project struct {
	owner string
	name  string
}

type comment struct {
	author  string
	body    string
	created time.Time
}

type issue struct {
	number       int
	numUserNotes int
	title        string
	description  string
	author       string
	state        string
	comments     []comment
}

type milestone struct {
	title       string
	description string
	state       string
	startdate   time.Time
	duedate     time.Time
}

type wiki struct {
	title   string
	content string
}

type label struct {
	name        string
	color       string
	description string
}

type projectData struct {
	project    project
	issues     []issue
	milestones []milestone
	labels     []label
	wikis      []wiki
}

type (
	gitlabClient struct{ client *gitlab.Client }

	discussions []*gitlab.Discussion
)

func NewGitlabClient(customURL, gitlabToken string) (*gitlabClient, error) {
	client := gitlab.NewClient(nil, gitlabToken)
	if err := client.SetBaseURL(customURL); err != nil {
		return nil, err
	}
	return &gitlabClient{client}, nil
}

func (c *gitlabClient) GetProjects(page, perPage int) ([]*gitlab.Project, error) {
	opts := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			Page:    page,
			PerPage: perPage,
		},
	}
	var list []*gitlab.Project

	for {
		projects, resp, err := c.client.Projects.ListProjects(opts)

		if err != nil {
			return nil, err
		}

		list = append(list, projects...)

		if lastPage(resp) {
			break
		}

		opts.Page = resp.NextPage
	}
	return list, nil
}

func (c *gitlabClient) getIssues(pid, page, perPage int) ([]*gitlab.Issue, error) {
	opts := gitlab.ListProjectIssuesOptions{
		ListOptions: gitlab.ListOptions{
			Page:    page,
			PerPage: perPage,
		},
	}
	var list []*gitlab.Issue

	for {
		issues, resp, err := c.client.Issues.ListProjectIssues(pid, &opts)
		if err != nil {
			return nil, err
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
		comments, err = c.issueComments(pid, srcissue.ID)
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

func (c *gitlabClient) issueComments(pid, issueNum int) ([]comment, error) {
	opts := &gitlab.ListIssueDiscussionsOptions{
		Page:    1,
		PerPage: 100,
	}

	var comments []comment

	for {
		discuss, resp, err := c.client.Discussions.ListIssueDiscussions(pid, issueNum, opts)
		if err != nil {
			return nil, err
		}
		comments = append(comments, discussions(discuss).toComments()...)

		if lastPage(resp) {
			break
		}
		opts.Page = resp.NextPage
	}
	return comments, nil
}

func (discuss discussions) toComments() []comment {
	comments := []comment{}

	for _, diss := range discuss {
		for _, note := range diss.Notes {
			c := comment{
				author:  note.Author.Email,
				body:    note.Body,
				created: *note.CreatedAt,
			}
			comments = append(comments, c)
		}
	}
	return comments

}

func (c *gitlabClient) getMilestones(pid, page, perPage int) ([]*gitlab.Milestone, error) {
	opts := gitlab.ListMilestonesOptions{
		ListOptions: gitlab.ListOptions{
			Page:    page,
			PerPage: perPage,
		},
	}

	var list []*gitlab.Milestone

	for {
		milestones, resp, err := c.client.Milestones.ListMilestones(pid, &opts)
		if err != nil {
			return nil, err
		}
		list = append(list, milestones...)

		if lastPage(resp) {
			break
		}

		opts.Page = resp.NextPage
	}
	return list, nil
}

func (c *gitlabClient) getLabels(pid, page, perPage int) ([]*gitlab.Label, error) {
	opts := gitlab.ListLabelsOptions{
		Page:    page,
		PerPage: perPage,
	}

	var list []*gitlab.Label

	for {
		labels, resp, err := c.client.Labels.ListLabels(pid, &opts)
		if err != nil {
			return nil, err
		}

		list = append(list, labels...)

		if lastPage(resp) {
			break
		}

		opts.Page = resp.NextPage
	}
	return list, nil
}

func (c *gitlabClient) getWikis(pid int) ([]*gitlab.Wiki, error) {
	opts := gitlab.ListWikisOptions{
		WithContent: gitlab.Bool(true),
	}

	wikis, _, err := c.client.Wikis.ListWikis(pid, &opts)
	if err != nil {
		return nil, err
	}

	return wikis, nil
}

func lastPage(resp *gitlab.Response) bool {
	return resp == nil || resp.CurrentPage >= resp.TotalPages || resp.NextPage == 0
}
