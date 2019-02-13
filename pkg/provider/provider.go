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

import "time"

type (
	// GitRepository stores general git repository data
	GitRepository struct {
		Name        string
		Description string
		CloneURL    string
		SSHURL      string
		Owner       string
		Archived    bool
		Fork        bool
		Empty       bool
		PID         int
	}
	// GitIssue stores general git SaaS issue data
	GitIssue struct {
		Repo      string
		PID       int
		Number    int
		Title     string
		Body      string
		State     string
		Labels    []GitLabel
		User      *GitUser
		Assignees []GitUser
	}
	// GitLabel stores general git SaaS label data
	GitLabel struct {
		Repo        string
		Name        string
		Color       string
		Description string
	}

	// GitUser stores general git SaaS user data
	GitUser struct {
		Login string
		Name  string
		Email string
	}

	// GitIssueComment stores general SaaS git issue comment data
	GitIssueComment struct {
		Repo      string
		IssueNum  int
		User      GitUser
		Body      string
		CreatedAt time.Time
		UpdatedAt time.Time
	}
)

// ToGitLabels converts a list of strings into a list of GitLabels
func ToGitLabels(names []string) []GitLabel {
	answer := []GitLabel{}
	for _, n := range names {
		answer = append(answer, GitLabel{Name: n})
	}
	return answer
}

// ToGitLabelStringSlice converts a list of GitLabels to a list of strings
func ToGitLabelStringSlice(labels []GitLabel) *[]string {
	labelStrings := []string{}
	for _, label := range labels {
		name := label.Name
		if name != "" {
			labelStrings = append(labelStrings, name)
		}
	}
	return &labelStrings
}

// UsersToString converts a list of GitUsers to a pointer to a slice of user strings
func UsersToString(users []GitUser) *[]string {
	var result []string
	for _, user := range users {
		result = append(result, user.Login)
	}
	return &result
}
