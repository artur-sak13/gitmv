package main

import (
	"context"
	"flag"

	"github.com/artur-sak13/gitmv/provider"
)

const issuesHelp = `Migrate all issues from one Git provider to another.`

func (cmd *issuesCommand) Name() string      { return "issues" }
func (cmd *issuesCommand) Args() string      { return "[OPTIONS]" }
func (cmd *issuesCommand) ShortHelp() string { return issuesHelp }
func (cmd *issuesCommand) LongHelp() string  { return issuesHelp }
func (cmd *issuesCommand) Hidden() bool      { return false }

func (cmd *issuesCommand) Register(fs *flag.FlagSet) {}

type issuesCommand struct{}

func (cmd *issuesCommand) Run(ctx context.Context, args []string) error {
	return runCommand(ctx, cmd.handleIssues)
}

// handleIssues will return
func (cmd *issuesCommand) handleIssues(ctx context.Context, src, dest provider.GitProvider) error {
	return nil
}
