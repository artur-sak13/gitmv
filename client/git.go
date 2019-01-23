package client

import (
	"os"

	"github.com/sirupsen/logrus"

	"gopkg.in/src-d/go-billy.v4/memfs"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

// TODO: Make sure this is worth the extra imports
func migrateWiki(url string) {
	fs := memfs.New()
	pks, err := ssh.NewPublicKeysFromFile("git", "/Users/Artur/.ssh/id_rsa", "")
	if err != nil {
		logrus.Errorf("unable to read private key: %v", err)
	}

	r, err := git.Clone(memory.NewStorage(), fs, &git.CloneOptions{
		URL:      url,
		Auth:     pks,
		Progress: os.Stdout,
	})
	if err != nil {
		logrus.Errorf("error cloning repository %v", err)
	}

	err = r.DeleteRemote("origin")
	if err != nil {
		logrus.Errorf("error removing git remote %v", err)
	}
	_, err = r.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{url},
	})
	if err != nil {
		logrus.Errorf("error creating remote repo %v", err)
	}
	err = r.Push(&git.PushOptions{
		Auth:     pks,
		Progress: os.Stdout,
	})

	if err != nil {
		logrus.Errorf("error could not push to remote repo %v", err)
	}
}
