package provider

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

type FakeIssue struct {
	Issue    *GitIssue
	Comments []*GitIssueComment
}

type FakeRepository struct {
	GitRepo     *GitRepository
	Issues      *sync.Map
	Labels      []*GitLabel
	Private     bool
	Description string
	issueCount  int
}

type FakeProvider struct {
	Repositories *sync.Map
}

// NewFakeRepository creates a new fake provider
func NewFakeProvider() GitProvider {
	provider := &FakeProvider{
		Repositories: &sync.Map{},
	}

	return provider
}

func (f *FakeProvider) CreateRepository(name, description string) (*GitRepository, error) {
	gitRepo := &GitRepository{
		Name: name,
	}

	repo := &FakeRepository{
		GitRepo:     gitRepo,
		Private:     true,
		Description: description,
		Issues:      &sync.Map{},
		Labels:      []*GitLabel{},
		issueCount:  0,
	}
	logrus.WithField("repo", gitRepo.Name).Info("creating repo")

	result, loaded := f.Repositories.LoadOrStore(name, repo)
	if loaded {
		return nil, fmt.Errorf("repository %s already exists", (*result.(*FakeRepository)).GitRepo.Name)
	}

	return gitRepo, nil
}

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

	// logrus.WithFields(logrus.Fields{
	// 	"IID":   issue.Number,
	// 	"issue": issue.Title,
	// 	"state": issue.State,
	// }).Info("creating issue")

	fakeRepo.(*FakeRepository).Issues.Store(issue.Number, newIssue)

	return issue, nil
}

func (f *FakeProvider) CreateIssueComment(comment *GitIssueComment) error {
	number := comment.IssueNum

	fakeRepo, ok := f.Repositories.Load(comment.Repo)

	if !ok {
		return fmt.Errorf("repository '%s' not found", comment.Repo)
	}

	// logrus.WithFields(logrus.Fields{
	// 	"repo":    comment.Repo,
	// 	"comment": comment.Body,
	// }).Info("creating comment")

	repoIssue, ok := fakeRepo.(*FakeRepository).Issues.Load(number)
	if !ok {
		return fmt.Errorf("issue number '%d' does not exist for %s", number, comment.Repo)
	}
	issue := repoIssue.(*FakeIssue)
	issue.Comments = append(issue.Comments, comment)
	return nil
}

func (f *FakeProvider) CreateLabel(label *GitLabel) (*GitLabel, error) {
	fakeRepo, ok := f.Repositories.Load(label.Repo)
	if !ok {
		return nil, fmt.Errorf("repository '%s' not found", label.Repo)
	}
	repo := fakeRepo.(*FakeRepository)

	// logrus.WithFields(logrus.Fields{
	// 	"repo":  label.Repo,
	// 	"label": label.Name,
	// 	"color": label.Color,
	// }).Info("creating label")

	repo.Labels = append(repo.Labels, label)

	return label, nil
}

func (f *FakeProvider) GetRepositories() ([]*GitRepository, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *FakeProvider) GetIssues(pid int, repo string) ([]*GitIssue, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *FakeProvider) GetComments(pid, issueNum int, repo string) ([]*GitIssueComment, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *FakeProvider) GetLabels(pid int, repo string) ([]*GitLabel, error) {
	return nil, fmt.Errorf("not implemented")
}
