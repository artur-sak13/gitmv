package main

import (
	"context"
	"flag"

	"github.com/artur-sak13/gitmv/provider"
)

const reposHelp = `Migrate all repos from one Git provider to another.`

func (cmd *reposCommand) Name() string      { return "repos" }
func (cmd *reposCommand) Args() string      { return "[OPTIONS]" }
func (cmd *reposCommand) ShortHelp() string { return reposHelp }
func (cmd *reposCommand) LongHelp() string  { return reposHelp }
func (cmd *reposCommand) Hidden() bool      { return false }

func (cmd *reposCommand) Register(fs *flag.FlagSet) {}

type reposCommand struct{}

func (cmd *reposCommand) Run(ctx context.Context, args []string) error {
	return runCommand(ctx, cmd.handleRepos)
}

// handleRepo will return
func (cmd *reposCommand) handleRepos(ctx context.Context, src, dest provider.GitProvider) error {
	err := dest.(*provider.GithubProvider).LoadCache()
	if err != nil {
		return err
	}
	dest.(*provider.GithubProvider).PrintCache()
	return nil
	// repos, err := src.GetRepositories()
	// if err != nil {
	// 	return fmt.Errorf("error getting repos: %v", err)
	// }

	// for _, repo := range repos {
	// 	if repo.Fork || repo.Empty {
	// 		continue
	// 	}

	// 	destRepo, err := dest.CreateRepository(repo)
	// 	if err != nil {
	// 		return fmt.Errorf("error creating repository: %v", err)
	// 	}

	// 	logrus.WithFields(logrus.Fields{
	// 		"repo": destRepo.Name,
	// 		"url":  destRepo.CloneURL,
	// 	}).Infof("creating new repo")

	// 	status, err := dest.MigrateRepo(repo, src.GetAuth().Token)
	// 	if err != nil {
	// 		return fmt.Errorf("error failed to migrate repository: %v", err)
	// 	}

	// 	logrus.WithFields(logrus.Fields{
	// 		"repo":   repo.Name,
	// 		"status": status,
	// 	}).Infof("importing repo")
	// }

	// return nil
}
