package provider

import (
	"fmt"
	"sync"

	"github.com/artur-sak13/gitmv/auth"
)

// FakeIssue stores information about git issues and their associated comments
type FakeIssue struct {
	Issue    *GitIssue
	Comments []*GitIssueComment
}

// FakeRepository stores information about a new git repository
type FakeRepository struct {
	GitRepo     *GitRepository
	Issues      *sync.Map
	Labels      []*GitLabel
	Private     bool
	Description string
	issueCount  int
}

// FakeProvider stores a thread safe hashmap of repository data
type FakeProvider struct {
	Repositories *sync.Map
}

// NewFakeProvider creates a new fake provider
func NewFakeProvider() GitProvider {
	provider := &FakeProvider{
		Repositories: &sync.Map{},
	}

	return provider
}

// CreateRepository creates a new Fake repository
func (f *FakeProvider) CreateRepository(srcRepo *GitRepository) (*GitRepository, error) {
	gitRepo := &GitRepository{
		Name: srcRepo.Name,
	}

	repo := &FakeRepository{
		GitRepo:     gitRepo,
		Private:     true,
		Description: srcRepo.Description,
		Issues:      &sync.Map{},
		Labels:      []*GitLabel{},
		issueCount:  0,
	}

	result, loaded := f.Repositories.LoadOrStore(srcRepo.Name, repo)
	if loaded {
		return nil, fmt.Errorf("repository %s already exists", (*result.(*FakeRepository)).GitRepo.Name)
	}

	return gitRepo, nil
}

// MigrateRepo migrates a git repo from an existing provider
func (f *FakeProvider) MigrateRepo(repo *GitRepository, token string) (string, error) {
	return "complete", nil
}

func (f *FakeProvider) GetImportProgress(repo string) (string, error) {
	return "complete", nil
}

// CreateIssue creates a new fake issue
func (f *FakeProvider) CreateIssue(issue *GitIssue) (*GitIssue, error) {
	fakeRepo, ok := f.Repositories.Load(issue.Repo)
	if !ok {
		return nil, fmt.Errorf("repository '%s' not found", issue.Repo)
	}

	fakeRepo.(*FakeRepository).issueCount++

	newIssue := &FakeIssue{
		Issue:    issue,
		Comments: []*GitIssueComment{},
	}

	fakeRepo.(*FakeRepository).Issues.Store(issue.Number, newIssue)

	return issue, nil
}

// CreateIssueComment creates a new fake issue comment
func (f *FakeProvider) CreateIssueComment(comment *GitIssueComment) error {
	number := comment.IssueNum

	fakeRepo, ok := f.Repositories.Load(comment.Repo)

	if !ok {
		return fmt.Errorf("repository '%s' not found", comment.Repo)
	}

	repoIssue, ok := fakeRepo.(*FakeRepository).Issues.Load(number)
	if !ok {
		return fmt.Errorf("issue number '%d' does not exist for %s", number, comment.Repo)
	}
	issue := repoIssue.(*FakeIssue)
	issue.Comments = append(issue.Comments, comment)
	return nil
}

// CreateLabel creates a new fake issue label
func (f *FakeProvider) CreateLabel(label *GitLabel) (*GitLabel, error) {
	fakeRepo, ok := f.Repositories.Load(label.Repo)
	if !ok {
		return nil, fmt.Errorf("repository '%s' not found", label.Repo)
	}
	repo := fakeRepo.(*FakeRepository)

	repo.Labels = append(repo.Labels, label)

	return label, nil
}

// GetAuthToken returns a string with a user's api authentication token
func (f *FakeProvider) GetAuth() *auth.ID {
	return auth.NewAuthID("git.example.com", "test-token", "/fake/.ssh/id_rsa", "fakeorg")
}

// GetRepositories gets the fake provider's repositories
func (f *FakeProvider) GetRepositories() ([]*GitRepository, error) {
	return nil, fmt.Errorf("not implemented")
}

// GetIssues gets the fake provider's issues
func (f *FakeProvider) GetIssues(pid int, repo string) ([]*GitIssue, error) {
	return nil, fmt.Errorf("not implemented")
}

// GetComments gets the fake provider's comments
func (f *FakeProvider) GetComments(pid, issueNum int, repo string) ([]*GitIssueComment, error) {
	return nil, fmt.Errorf("not implemented")
}

// GetLabels gets the fake provider's labels
func (f *FakeProvider) GetLabels(pid int, repo string) ([]*GitLabel, error) {
	return nil, fmt.Errorf("not implemented")
}
