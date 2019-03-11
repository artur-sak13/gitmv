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
	"io/ioutil"
	"net"
	"os"
	"strings"

	"github.com/artur-sak13/gitmv/auth"

	"golang.org/x/crypto/ssh"

	"gopkg.in/src-d/go-billy.v4/memfs"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	gitssh "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

func MigrateWiki(repo *GitRepository, id *auth.ID) error {
	fs := memfs.New()
	storer := memory.NewStorage()

	s := fmt.Sprintf("%s/.ssh/id_rsa", os.Getenv("HOME"))
	key, err := ioutil.ReadFile(s)
	if err != nil {
		return fmt.Errorf("error reading private key: %v", err)
	}

	signer, err := ssh.ParsePrivateKey([]byte(key))
	if err != nil {
		return fmt.Errorf("error parsing private key: %v", err)
	}
	hostKeyCallback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		return nil
	}

	auth := &gitssh.PublicKeys{
		User:   "git",
		Signer: signer,
		HostKeyCallbackHelper: gitssh.HostKeyCallbackHelper{
			HostKeyCallback: hostKeyCallback,
		},
	}
	wikiURL := strings.TrimSuffix(repo.SSHURL, ".git") + ".wiki.git"

	r, err := git.Clone(storer, fs, &git.CloneOptions{
		URL:      wikiURL,
		Auth:     auth,
		Progress: os.Stdout,
	})

	if err == transport.ErrEmptyRemoteRepository {
		return nil
	}

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

	refspec := config.RefSpec("+" + config.DefaultPushRefSpec)
	err = r.Push(&git.PushOptions{
		Auth: auth,
		RefSpecs: []config.RefSpec{
			refspec,
		},
		Progress: os.Stdout,
	})

	if err != nil {
		fmt.Printf("Need to create wiki for: %s\n", newWikiURL)
		fmt.Printf("returned error: %v\n", err)
		return nil
	}

	return nil
}
