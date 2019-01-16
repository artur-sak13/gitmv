package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/artur-sak13/gitmv/version"

	"golang.org/x/oauth2"

	"github.com/genuinetools/pkg/cli"
	"github.com/google/go-github/v21/github"
	"github.com/sirupsen/logrus"
	gitlab "github.com/xanzy/go-gitlab"
)

// TODO: Get ssh keys for users
// TODO: Abstract get* functions to make this DRY
// TODO: Process concurrently and wait for imports to complete
// TODO: Add option to "dry-run" migration
// TODO: Generate docs

const TESTREPO string = "https://gitlab.twopoint.io/artur.sak/winaws"

var (
	githubToken string
	gitlabToken string
	customURL   string
	debug       bool
)

func main() {
	p := cli.NewProgram()
	p.Name = "gitmv"
	p.Description = "A command line tool to migrate repos between GitLab and Github"

	p.GitCommit = version.GITCOMMIT
	p.Version = version.VERSION

	p.FlagSet = flag.NewFlagSet("global", flag.ExitOnError)
	p.FlagSet.StringVar(&githubToken, "github-token", os.Getenv("GITHUB_TOKEN"), "GitHub API token (or env var GITHUB_TOKEN)")
	p.FlagSet.StringVar(&gitlabToken, "gitlab-token", os.Getenv("GITLAB_TOKEN"), "GitLab API token (or env var GITLAB_TOKEN)")

	p.FlagSet.StringVar(&customURL, "url", os.Getenv("GITLAB_URL"), "Custom GitLab URL")
	p.FlagSet.StringVar(&customURL, "u", os.Getenv("GITLAB_URL"), "Custom GitLab URL")

	p.FlagSet.BoolVar(&debug, "debug", false, "enable debug logging")
	p.FlagSet.BoolVar(&debug, "d", false, "enable debug logging")

	p.Before = func(ctx context.Context) error {
		if debug {
			logrus.SetLevel(logrus.DebugLevel)
		}

		if len(githubToken) < 1 {
			return errors.New("github token cannot be empty")
		}

		if len(gitlabToken) < 1 {
			return errors.New("gitlab token cannot be empty")
		}

		return nil
	}

	p.Action = runCommand

	p.Run()
}

func runCommand(ctx context.Context, args []string) error {
	// On ^C, or SIGTERM handle exit.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	signal.Notify(signals, syscall.SIGTERM)

	var cancel context.CancelFunc
	_, cancel = context.WithCancel(ctx)
	go func() {
		for sig := range signals {
			cancel()
			logrus.Infof("Received %s, exiting.", sig.String())
			os.Exit(0)
		}
	}()

	client, err := newGitlabClient()
	if err != nil {
		return err
	}

	m := &migrator{
		ghClient: newGitHubClient(ctx),
		glClient: client,
	}

	if err := m.migrateRepo(ctx, TESTREPO); err != nil {
		return err
	}

	// page := 1
	// perPage := 100
	// logrus.Debugf("Getting projects...")
	// if err := m.getProjects(page, perPage); err != nil {
	// 	logrus.Errorf("Failed to get repos, %v\n", err)
	// 	return err
	// }

	return nil
}

func newGitlabClient() (*gitlab.Client, error) {
	client := gitlab.NewClient(nil, gitlabToken)
	if err := client.SetBaseURL(customURL); err != nil {
		return nil, err
	}
	return client, nil
}

func newGitHubClient(ctx context.Context) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	return client
}

type migrator struct {
	ghClient *github.Client
	glClient *gitlab.Client
}

func (m *migrator) newRepo(ctx context.Context, name, org, description string) (*github.Repository, error) {
	repo := &github.Repository{
		Name:        github.String(name),
		Description: github.String(description),
	}
	r, _, err := m.ghClient.Repositories.Create(ctx, org, repo)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (m *migrator) migrateRepo(ctx context.Context, sourcerepo string) error {
	parts := strings.SplitN(sourcerepo, "/", 2)
	if len(parts) < 2 {
		return fmt.Errorf("could not parse repo name into owner and repo for %s, got: %#v", sourcerepo, parts)
	}
	// Must create repository before running import
	im := &github.Import{
		VCSURL:      github.String(sourcerepo),
		VCS:         github.String("git"),
		VCSUsername: github.String(parts[0]),
		VCSPassword: github.String(gitlabToken),
	}
	imprt, _, err := m.ghClient.Migrations.StartImport(ctx, "twopt", parts[1], im)
	if err != nil {
		return err
	}
	logrus.Infof("Importing %s", *imprt.Status)
	return nil
}

func (m *migrator) getProjects(page, perPage int) error {
	opts := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			Page:    page,
			PerPage: perPage,
		},
	}
	projects, resp, err := m.glClient.Projects.ListProjects(opts)

	if err != nil {
		return err
	}

	for i, project := range projects {
		fmt.Printf("Found project #%d: %s\n", i+1, project.HTTPURLToRepo)
		issues, err := m.getIssues(project.ID, 1, 100, []*gitlab.Issue{})
		if err != nil {
			return err
		}
		for _, issue := range issues {
			fmt.Printf("Found issue %s for project %s\n", issue.Title, project.Name)
		}
	}

	fmt.Printf("You have %d projects\n", len(projects))

	if resp == nil || resp.CurrentPage >= resp.TotalPages || resp.NextPage == 0 {
		return nil
	}

	page = resp.NextPage
	return m.getProjects(page, perPage)
}

func (m *migrator) getIssues(pid, page, perPage int, list []*gitlab.Issue) ([]*gitlab.Issue, error) {
	opts := gitlab.ListProjectIssuesOptions{
		ListOptions: gitlab.ListOptions{
			Page:    page,
			PerPage: perPage,
		},
	}
	issues, resp, err := m.glClient.Issues.ListProjectIssues(pid, &opts)
	if err != nil {
		return nil, err
	}

	if resp == nil || resp.CurrentPage >= resp.TotalPages || resp.NextPage == 0 {
		return append(list, issues...), nil
	}

	page = resp.NextPage
	list = append(list, issues...)
	return m.getIssues(pid, page, perPage, list)
}

func (m *migrator) getMilestones(pid, page, perPage int, list []*gitlab.Milestone) ([]*gitlab.Milestone, error) {
	opts := gitlab.ListMilestonesOptions{
		ListOptions: gitlab.ListOptions{
			Page:    page,
			PerPage: perPage,
		},
	}
	milestones, resp, err := m.glClient.Milestones.ListMilestones(pid, &opts)
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.CurrentPage >= resp.TotalPages || resp.NextPage == 0 {
		return append(list, milestones...), nil
	}

	page = resp.NextPage
	list = append(list, milestones...)
	return m.getMilestones(pid, page, perPage, list)
}

func (m *migrator) getMergeRequests(pid, page, perPage int, list []*gitlab.MergeRequest) ([]*gitlab.MergeRequest, error) {
	opts := gitlab.ListProjectMergeRequestsOptions{
		ListOptions: gitlab.ListOptions{
			Page:    page,
			PerPage: perPage,
		},
	}
	mergeReqs, resp, err := m.glClient.MergeRequests.ListProjectMergeRequests(pid, &opts)
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.CurrentPage >= resp.TotalPages || resp.NextPage == 0 {
		return append(list, mergeReqs...), nil
	}

	page = resp.NextPage
	list = append(list, mergeReqs...)
	return m.getMergeRequests(pid, page, perPage, list)
}

func (m *migrator) getLabels(pid, page, perPage int, list []*gitlab.Label) ([]*gitlab.Label, error) {
	opts := gitlab.ListLabelsOptions{
		Page:    page,
		PerPage: perPage,
	}

	labels, resp, err := m.glClient.Labels.ListLabels(pid, &opts)
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.CurrentPage >= resp.TotalPages || resp.NextPage == 0 {
		return append(list, labels...), nil
	}

	page = resp.NextPage
	list = append(list, labels...)
	return m.getLabels(pid, page, perPage, list)
}

func (m *migrator) getWikis(pid, page, perPage int, list []*gitlab.Wiki) ([]*gitlab.Wiki, error) {
	opts := gitlab.ListWikisOptions{
		WithContent: gitlab.Bool(true),
	}

	wikis, resp, err := m.glClient.Wikis.ListWikis(pid, &opts)
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.CurrentPage >= resp.TotalPages || resp.NextPage == 0 {
		return append(list, wikis...), nil
	}

	page = resp.NextPage
	list = append(list, wikis...)
	return m.getWikis(pid, page, perPage, list)
}
