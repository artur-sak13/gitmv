package provider

import "time"

type (
	GitRepository struct {
		Name     string
		HTMLURL  string
		CloneURL string
		SSHURL   string
		Fork     bool
	}

	GitIssue struct {
		Owner         string
		Repo          string
		Number        *int
		Title         string
		Body          string
		State         *string
		Labels        []GitLabel
		CreatedAt     *time.Time
		UpdatedAt     *time.Time
		ClosedAt      *time.Time
		IsPullRequest bool
		User          *GitUser
		ClosedBy      *GitUser
		Assignees     []GitUser
	}

	GitIssueComment struct {
		User      GitUser
		Body      string
		CreatedAt *time.Time
		UpdatedAt *time.Time
	}

	GitUser struct {
		URL       string
		Login     string
		Name      string
		Email     string
		AvatarURL string
	}

	GitLabel struct {
		Name        string
		Color       string
		Description string
	}
)

func ToGitLabels(names []string) []GitLabel {
	answer := []GitLabel{}
	for _, n := range names {
		answer = append(answer, GitLabel{Name: n})
	}
	return answer
}
