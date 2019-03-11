// The MIT License (MIT)
//
// Copyright (c) 2019 Artur Sak
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/artur-sak13/gitmv/migrator"

	"github.com/artur-sak13/gitmv/auth"
	"github.com/artur-sak13/gitmv/provider"

	"github.com/artur-sak13/gitmv/version"

	"github.com/genuinetools/pkg/cli"
	"github.com/sirupsen/logrus"

	"github.com/google/gops/agent"
)

// *     [X] Make it work
// ?     [?] Make it fast
// TODO: [ ] Make it elegant
// TODO: Fix incorrect issue numbers between source and dest repos
var (
	githubToken string
	gitlabToken string
	gitlabUser  string
	customURL   string
	org         string
	debug       bool
	dryrun      bool
)

func main() {
	p := cli.NewProgram()
	p.Name = "gitmv"
	p.Description = "A command line tool to migrate repos between GitLab and GitHub"

	p.GitCommit = version.GITCOMMIT
	p.Version = version.VERSION

	p.Commands = []cli.Command{
		&reposCommand{},
		&issuesCommand{},
		&wikisCommand{},
	}

	p.FlagSet = flag.NewFlagSet("global", flag.ExitOnError)
	p.FlagSet.StringVar(&githubToken, "github-token", os.Getenv("GITHUB_TOKEN"), "GitHub API token (or env var GITHUB_TOKEN)")
	p.FlagSet.StringVar(&gitlabToken, "gitlab-token", os.Getenv("GITLAB_TOKEN"), "GitLab API token (or env var GITLAB_TOKEN)")
	p.FlagSet.StringVar(&gitlabUser, "gitlab-user", os.Getenv("GITLAB_USER"), "GitLab Username")

	p.FlagSet.StringVar(&org, "org", os.Getenv("GHORG"), "GitHub org to move repositories")

	p.FlagSet.StringVar(&customURL, "url", os.Getenv("GITLAB_URL"), "Custom GitLab URL")
	p.FlagSet.StringVar(&customURL, "u", os.Getenv("GITLAB_URL"), "Custom GitLab URL")

	p.FlagSet.BoolVar(&debug, "debug", false, "enable debug logging")
	p.FlagSet.BoolVar(&debug, "d", false, "enable debug logging")

	p.FlagSet.BoolVar(&dryrun, "dry-run", false, "do not run migration just print the changes that would occur")

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

	p.Run()
}

func runCommand(ctx context.Context, cmd func(context.Context, provider.GitProvider, provider.GitProvider) error) error {
	// On ^C, or SIGTERM handle exit.
	signals := make(chan os.Signal, 0)
	signal.Notify(signals, os.Interrupt)
	signal.Notify(signals, syscall.SIGTERM)

	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)

	go func() {
		for sig := range signals {
			cancel()
			logrus.Infof("Received %s, exiting.", sig.String())
			os.Exit(0)
		}
	}()

	if err := agent.Listen(agent.Options{
		ShutdownCleanup: true, // automatically closes on os.Interrupt
	}); err != nil {
		logrus.Fatalf("gops agent failed: %v", err)
	}

	glAuth := auth.NewAuthID(customURL, gitlabToken, "")
	ghAuth := auth.NewAuthID("", githubToken, org)

	src, err := provider.NewGitlabProvider(glAuth)
	if err != nil {
		return err
	}

	dest, err := provider.NewGithubProvider(ctx, ghAuth)
	if err != nil {
		return err
	}

	if err := cmd(ctx, src, dest); err != nil {
		logrus.Fatalf("error: %v", err)
		os.Exit(1)
	}

	return nil
}

func runMigration(ctx context.Context, args []string) error {
	// On ^C, or SIGTERM handle exit.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	signal.Notify(signals, syscall.SIGTERM)

	var cancel context.CancelFunc
	_, cancel = context.WithCancel(ctx)
	go func() {
		for sig := range signals {
			cancel()
			logrus.Infof("received %s, exiting.", sig.String())
			os.Exit(0)
		}
	}()

	if err := agent.Listen(agent.Options{
		ShutdownCleanup: true, // automatically closes on os.Interrupt
	}); err != nil {
		logrus.Fatalf("gops agent failed: %v", err)
	}

	a := auth.NewAuthID(customURL, gitlabToken, "")
	src, err := provider.NewGitlabProvider(a)
	if err != nil {
		logrus.Fatalf("error initializing GitProvider: %v", err)
		os.Exit(1)
	}

	var dest provider.GitProvider

	if dryrun {
		dest = provider.NewFakeProvider()
	} else {
		id := auth.NewAuthID("", githubToken, org)
		dest, err = provider.NewGithubProvider(ctx, id)
		if err != nil {
			logrus.Fatalf("error initializing GitProvider: %v", err)
			os.Exit(1)
		}
	}
	mig := migrator.NewMigrator(src, dest)
	err = mig.Run()
	if err != nil {
		logrus.Fatalf("error moving repos: %v", err)
		os.Exit(1)
	}

	return nil
}
