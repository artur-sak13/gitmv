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

	"github.com/artur-sak13/gitmv/version"

	"github.com/genuinetools/pkg/cli"
	"github.com/sirupsen/logrus"
)

// TODO: Implement rest of cli
// TODO: Use closures and partial function application to remove repetition
// TODO: Get ssh keys for users
// TODO: Process concurrently and wait for imports to complete
// TODO: Add option to "dry-run" migration

var (
	githubToken string
	gitlabToken string
	gitlabUser  string
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
	p.FlagSet.StringVar(&gitlabUser, "gitlab-user", os.Getenv("GITLAB_USER"), "GitLab Username")

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

	// glClient, err := client.NewGitlabClient(customURL, gitlabToken)
	// if err != nil {
	// 	return err
	// }

	// ghClient := client.NewGitHubClient(ctx, githubToken, true)
	// logrus.Debugf("Getting projects...")

	// projects, err := glClient.GetProjects()

	// if err != nil {
	// 	logrus.Errorf("failed to get repos, %v\n", err)
	// 	return err
	// }

	return nil
}
