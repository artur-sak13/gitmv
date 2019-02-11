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
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/artur-sak13/gitmv/pkg/provider"
)

// Migrator
type Migrator struct {
	Src    provider.GitProvider
	Dest   provider.GitProvider
	Errors chan error
}

type Worker struct {
	Migrator  *Migrator
	WaitGroup *sync.WaitGroup
}

// NewMigrator
func NewMigrator(src, dest provider.GitProvider) *Migrator {
	return &Migrator{
		Src:    src,
		Dest:   dest,
		Errors: make(chan error),
	}
}

func (m *Migrator) NewWorker() *Worker {
	return &Worker{
		Migrator: m,
	}
}

// Run
// TODO: Process concurrently and wait for imports to complete
func (m *Migrator) Run() error {
	repos, err := m.Src.GetRepositories()
	if err != nil {
		return fmt.Errorf("failed to retrieve repos: %v", err)
	}

	const maxgoroutines = 20

	start := time.Now()
	wg := sync.WaitGroup{}
	wg.Add(len(repos))

	for _, repo := range repos {
		if repo.Fork || repo.Empty {
			continue
		}

		_, err := m.Dest.CreateRepository(repo.Name, "")
		if err != nil {
			return fmt.Errorf("failed to create repository: %v", err)
		}
		worker := m.NewWorker()
		go func(repo *provider.GitRepository) {
			worker.processIssues(repo)
			worker.processLabels(repo)
			wg.Done()
		}(repo)

	}
	wg.Wait()
	logrus.Infof("processed %d repositories in %s\n", len(repos), time.Since(start))

	if len(m.Errors) > 0 {
		for err := range m.Errors {
			logrus.Error(err)
		}
		return fmt.Errorf("errors occured during migration")
	}

	return nil
}

func (w *Worker) processIssues(repo *provider.GitRepository) {
	issues, err := w.Migrator.Src.GetIssues(repo.PID, repo.Name)
	if err != nil {
		logrus.Errorf("error in process issues")
		w.Migrator.Errors <- fmt.Errorf("failed to retrieve issues: %v", err)
	}
	var wg sync.WaitGroup
	wg.Add(len(issues))

	go func() {
		for _, issue := range issues {
			_, err := w.Migrator.Dest.CreateIssue(issue)
			if err != nil {
				logrus.Errorf("error in process issues: %v", err)
				w.Migrator.Errors <- fmt.Errorf("failed to create issue: %v", err)
			}
			w.processComments(issue, &wg)
		}
	}()
	wg.Wait()
}

func (w *Worker) processComments(issue *provider.GitIssue, wg *sync.WaitGroup) {
	comments, err := w.Migrator.Src.GetComments(issue.PID, issue.Number, issue.Repo)
	if err != nil {
		logrus.Errorf("error getting comments for issue %s of repo %s: %v", issue.Title, issue.Repo, err)
		logrus.Errorf("logging issue...%v", *issue)
		w.Migrator.Errors <- fmt.Errorf("failed to retrieve project comments: %v", err)
		wg.Done()
		return
	}
	go func() {
		for _, comment := range comments {
			err := w.Migrator.Dest.CreateIssueComment(comment)
			if err != nil {
				logrus.Errorf("error creating comments for repo %s: %v", issue.Repo, err)
				w.Migrator.Errors <- fmt.Errorf("failed to create comment: %v", err)
			}
		}
		wg.Done()
	}()
}

func (w *Worker) processLabels(repo *provider.GitRepository) {
	labels, err := w.Migrator.Src.GetLabels(repo.PID, repo.Name)
	if err != nil {
		logrus.Errorf("error in process labels: %v", err)
		w.Migrator.Errors <- fmt.Errorf("failed to retrieve labels: %v", err)
	}
	wg := &sync.WaitGroup{}
	wg.Add(len(labels))

	for _, label := range labels {
		go func(label *provider.GitLabel) {
			_, err := w.Migrator.Dest.CreateLabel(label)
			if err != nil {
				logrus.Errorf("error in process labels: %v", err)
				w.Migrator.Errors <- fmt.Errorf("failed to create label: %v", err)
			}
			wg.Done()
		}(label)
	}
	wg.Wait()
}
