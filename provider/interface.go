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

import "github.com/artur-sak13/gitmv/auth"

// GitProvider
type GitProvider interface {
	// Create methods
	CreateRepository(*GitRepository) (*GitRepository, error)

	CreateIssue(*GitIssue) (*GitIssue, error)

	CreateIssueComment(int, *GitIssueComment) error

	CreateLabel(*GitLabel) (*GitLabel, error)

	MigrateRepo(*GitRepository, string) (string, error)

	// Read methods
	GetRepositories() ([]*GitRepository, error)

	GetIssues(int, string) ([]*GitIssue, error)

	GetComments(int, int, string) ([]*GitIssueComment, error)

	GetLabels(int, string) ([]*GitLabel, error)

	GetAuth() *auth.ID

	GetImportProgress(string) (string, error)

	// ValidateRepositoryName(org string, name string) error
}
