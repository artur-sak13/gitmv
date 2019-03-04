package main

import (
	"context"
	"flag"
	"fmt"

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
	repos, err := src.GetRepositories()
	if err != nil {
		return err
	}
	github := dest.(*provider.GithubProvider)

	err = github.LoadCache()
	if err != nil {
		return err
	}
	count := 0
	for _, repo := range repos {
		if repo.Fork || repo.Empty {
			continue
		}
		if _, ok := github.Repocache[repo.Name]; !ok {
			count++
			fmt.Printf("Missing repo: %s\n", repo.Name)
		}
	}
	fmt.Printf("Repos missing: %d\n", count)
	// b, err := json.MarshalIndent(github.Repocache, "", " ")
	// if err != nil {
	// 	return err
	// }
	// fmt.Printf("Repocache: %s", string(b))
	// github.PrintCache()

	return nil
}
