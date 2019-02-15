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

package provider

import (
	"fmt"
	"os"
	"strings"

	"github.com/artur-sak13/gitmv/auth"
	"github.com/sirupsen/logrus"

	"gopkg.in/src-d/go-billy.v4/memfs"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

func MigrateWiki(repo *GitRepository, id *auth.ID) error {
	fs := memfs.New()

	pks, err := ssh.NewPublicKeysFromFile("git", id.SSHKeyPath, "")
	if err != nil {
		logrus.Errorf("unable to read private key: %v", err)
	}

	wikiURL := strings.TrimSuffix(repo.SSHURL, ".git") + ".wiki.git"
	r, err := git.Clone(memory.NewStorage(), fs, &git.CloneOptions{
		URL:      wikiURL,
		Auth:     pks,
		Progress: os.Stdout,
	})

	if err != nil {
		return fmt.Errorf("error cloning repository %v", err)
	}

	err = r.DeleteRemote("origin")
	if err != nil {
		return fmt.Errorf("error removing git remote %v", err)
	}

	newWikiURL := fmt.Sprintf("git@github.com:%s/%s.wiki.git", id.Owner, repo.Name)
	_, err = r.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{newWikiURL},
	})

	if err != nil {
		return fmt.Errorf("error creating remote repo %v", err)
	}

	err = r.Push(&git.PushOptions{
		Auth:     pks,
		Progress: os.Stdout,
	})

	if err != nil {
		return fmt.Errorf("error could not push to remote repo %v", err)
	}

	return nil
}
