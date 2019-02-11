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
	// GitRepository
	GitRepository struct {
		Name     string
		HTMLURL  string
		CloneURL string
		SSHURL   string
		Fork     bool
		Empty    bool
		PID      int
	}
	// GitIssue
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
	// GitIssueComment
	GitIssueComment struct {
		Repo      string
		IssueNum  int
		User      GitUser
		Body      string
		CreatedAt time.Time
		UpdatedAt time.Time
	}
	// GitUser
	GitUser struct {
		Login string
		Name  string
		Email string
	}
	// GitLabel
	GitLabel struct {
		Repo        string
		Name        string
		Color       string
		Description string
	}
)

// ToGitLabels
func ToGitLabels(names []string) []GitLabel {
	answer := []GitLabel{}
	for _, n := range names {
		answer = append(answer, GitLabel{Name: n})
	}
	return answer
}

// ToGitLabelStringSlice
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

// UserToString
func UsersToString(users []GitUser) *[]string {
	var result []string
	for _, user := range users {
		result = append(result, user.Login)
	}
	return &result
}
