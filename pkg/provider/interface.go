package provider

// TODO: Add functions to interface
type GitProvider interface {
	ListRepositories() ([]*GitRepository, error)

	// ValidateRepositoryName(org string, name string) error

	IsGitHub() bool

	Kind() string

	GetIssues(pid int, org, repo string) ([]*GitIssue, error)
}