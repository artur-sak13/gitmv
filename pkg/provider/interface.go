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

type GitProvider interface {
	// Create methods
	CreateRepository(name, description string) (*GitRepository, error)

	CreateIssue(repo string, issue *GitIssue) (*GitIssue, error)

	CreateIssueComment(repo string, number int, comment *GitIssueComment) error

	// Read methods
	GetRepositories() ([]*GitRepository, error)

	GetIssues(pid int, repo string) ([]*GitIssue, error)

	GetComments(pid, issueNum int) ([]*GitIssueComment, error)

	GetLabels(pid int) ([]*GitLabel, error)

	// ValidateRepositoryName(org string, name string) error
}
