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
package migrator

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/artur-sak13/gitmv/provider"
)

// Migrator stores the src and target git providers and an error channel
type Migrator struct {
	Src    provider.GitProvider
	Dest   provider.GitProvider
	Errors chan error
}

// NewMigrator creates a new git migrator
func NewMigrator(src, dest provider.GitProvider) *Migrator {
	return &Migrator{
		Src:    src,
		Dest:   dest,
		Errors: make(chan error),
	}
}

// Run processes git import jobs
func (m *Migrator) Run() error {
	repos, err := m.Src.GetRepositories()
	if err != nil {
		return fmt.Errorf("error getting repos: %v", err)
	}

	start := time.Now()
	wg := sync.WaitGroup{}
	importwg := sync.WaitGroup{}
	count := 0

	for _, repo := range repos {
		if repo.Fork || repo.Empty {
			continue
		}
		wg.Add(1)
		importwg.Add(1)
		count++

		destRepo, err := m.Dest.CreateRepository(repo)
		if err != nil {
			return fmt.Errorf("error creating repository: %v", err)
		}

		logrus.WithFields(logrus.Fields{
			"repo": destRepo.Name,
			"url":  destRepo.CloneURL,
		}).Infof("creating new repo")

		status, err := m.Dest.MigrateRepo(repo, m.Src.GetAuth().Token)
		if err != nil {
			return fmt.Errorf("error failed to migrate repository: %v", err)
		}

		logrus.WithFields(logrus.Fields{
			"repo":   repo.Name,
			"status": status,
		}).Infof("importing repo")

		go m.waitForImport(repo.Name, &importwg)

		go func(repo *provider.GitRepository) {
			m.processIssues(repo)
			m.processLabels(repo)
			if err := provider.MigrateWiki(destRepo, m.Dest.GetAuth()); err != nil {
				m.Errors <- fmt.Errorf("failed to migrate wiki for %s: %v", repo.Name, err)
			}
			wg.Done()
		}(repo)

	}
	wg.Wait()
	logrus.Infof("processed %d repositories in %s\n", count, time.Since(start))

	importwg.Wait()
	logrus.Infof("done waiting for repository imports")

	if len(m.Errors) > 0 {
		for err := range m.Errors {
			logrus.Error(err)
		}
		return fmt.Errorf("errors occured during migration")
	}

	return nil
}

func (m *Migrator) waitForImport(repo string, wg *sync.WaitGroup) {
	retries := 5
	for retryCount := 1; retryCount <= retries; retryCount++ {
		status, err := m.Dest.GetImportProgress(repo)
		if err != nil {
			m.Errors <- fmt.Errorf("failed to retrieve import progress: %v", err)
		}
		if status == "complete" {
			logrus.Infof("%s finished importing", repo)
			wg.Done()
			return
		}
		sleepForAttempt(retryCount)
	}
	logrus.Warnf("import status retries exhausted for %s", repo)
	wg.Done()
}

func sleepForAttempt(retryCount int) {
	maxDelay := 20 * time.Second
	delay := time.Second * time.Duration(math.Exp2(float64(retryCount)))
	if delay > maxDelay {
		delay = maxDelay
	}
	time.Sleep(delay)
}

func (m *Migrator) processIssues(repo *provider.GitRepository) {
	issues, err := m.Src.GetIssues(repo.PID, repo.Name)
	if err != nil {
		logrus.Errorf("error getting issues: %v", err)
		m.Errors <- fmt.Errorf("failed to retrieve issues: %v", err)
		return
	}
	var wg sync.WaitGroup
	wg.Add(len(issues))

	go func() {
		for _, issue := range issues {

			logrus.WithFields(logrus.Fields{
				"IID":   issue.Number,
				"issue": issue.Title,
				"state": issue.State,
			}).Info("creating issue")

			_, err := m.Dest.CreateIssue(issue)
			if err != nil {
				logrus.Errorf("error creating issue: %v", err)
				m.Errors <- fmt.Errorf("failed to create issue: %v", err)
				wg.Done()
				return
			}
			m.processComments(issue, &wg)
		}
	}()
	wg.Wait()
}

func (m *Migrator) processComments(issue *provider.GitIssue, wg *sync.WaitGroup) {
	comments, err := m.Src.GetComments(issue.PID, issue.Number, issue.Repo)
	if err != nil {
		logrus.Errorf("error getting comments: %v", err)
		m.Errors <- fmt.Errorf("failed to retrieve project comments: %v", err)
		wg.Done()
		return
	}
	go func() {
		for _, comment := range comments {

			logrus.WithFields(logrus.Fields{
				"repo":    comment.Repo,
				"comment": comment.Body,
			}).Info("creating comment")

			err := m.Dest.CreateIssueComment(comment)
			if err != nil {
				logrus.Errorf("error creating comments for repo %s: %v", issue.Repo, err)
				m.Errors <- fmt.Errorf("failed to create comment: %v", err)
				wg.Done()
				return
			}
		}
		wg.Done()
	}()
}

func (m *Migrator) processLabels(repo *provider.GitRepository) {
	labels, err := m.Src.GetLabels(repo.PID, repo.Name)
	if err != nil {
		logrus.Errorf("error getting labels: %v", err)
		m.Errors <- fmt.Errorf("failed to retrieve labels: %v", err)
		return
	}
	wg := &sync.WaitGroup{}
	wg.Add(len(labels))

	for _, label := range labels {
		go func(label *provider.GitLabel) {

			logrus.WithFields(logrus.Fields{
				"repo":  label.Repo,
				"label": label.Name,
				"color": label.Color,
			}).Info("creating label")

			_, err := m.Dest.CreateLabel(label)
			if err != nil {
				logrus.Errorf("error creating label: %v", err)
				m.Errors <- fmt.Errorf("failed to create label: %v", err)
				wg.Done()
				return
			}
			wg.Done()
		}(label)
	}
	wg.Wait()
}
