package main

import (
	"context"
	"flag"

	"github.com/artur-sak13/gitmv/provider"
)

const wikisHelp = `Migrate all wikis from one Git provider to another.`

func (cmd *wikisCommand) Name() string      { return "wikis" }
func (cmd *wikisCommand) Args() string      { return "[OPTIONS]" }
func (cmd *wikisCommand) ShortHelp() string { return wikisHelp }
func (cmd *wikisCommand) LongHelp() string  { return wikisHelp }
func (cmd *wikisCommand) Hidden() bool      { return false }

func (cmd *wikisCommand) Register(fs *flag.FlagSet) {}

type wikisCommand struct{}

func (cmd *wikisCommand) Run(ctx context.Context, args []string) error {
	return runCommand(ctx, cmd.handleWikis)
}

// handleWikis will return
func (cmd *wikisCommand) handleWikis(ctx context.Context, src, dest provider.GitProvider) error {
	return nil
}
