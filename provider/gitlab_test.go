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

package provider_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/artur-sak13/gitmv/auth"
	"github.com/artur-sak13/gitmv/provider"

	"gotest.tools/assert"

	"github.com/stretchr/testify/suite"
	gitlab "github.com/xanzy/go-gitlab"
)

const (
	gitlabUserName    = "tester"
	gitlabOrgName     = "testorg"
	gitlabProjectName = "test-project"
	// gitlabProjectID   = "8675309"
)

type GitlabProviderSuite struct {
	suite.Suite
	mux      *http.ServeMux
	server   *httptest.Server
	provider *provider.GitlabProvider
}

func (s *GitlabProviderSuite) SetupSuite() {
	mux, server, prov := setup(s)
	s.mux = mux
	s.server = server
	s.provider = prov
}

func (s *GitlabProviderSuite) TearDownSuite() {
	s.server.Close()
}

func setup(s *GitlabProviderSuite) (*http.ServeMux, *httptest.Server, *provider.GitlabProvider) {
	mux := http.NewServeMux()
	configureGitlabMock(s, mux)

	server := httptest.NewServer(mux)

	c := gitlab.NewClient(nil, "")
	_ = c.SetBaseURL(server.URL)

	id := auth.NewAuthID(server.URL, "test-token", gitlabOrgName)

	// Gitlab provider that we want to test
	prov := provider.WithGitlabClient(c, id)

	return mux, server, prov.(*provider.GitlabProvider)
}

func configureGitlabMock(s *GitlabProviderSuite, mux *http.ServeMux) {
	mux.HandleFunc("/api/v4/projects", func(w http.ResponseWriter, r *http.Request) {
		src, err := ioutil.ReadFile("test_data/gitlab/projects.json")

		s.Require().Nil(err)
		_, _ = w.Write(src)
	})

	mux.HandleFunc(fmt.Sprintf("/api/v4/projects/%d/issues", 4), func(w http.ResponseWriter, r *http.Request) {
		src, err := ioutil.ReadFile("test_data/gitlab/issues.json")

		s.Require().Nil(err)
		_, _ = w.Write(src)
	})
}

func TestIsHosted(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			"test hosted with https",
			"https://gitlab.com",
			true,
		}, {
			"test hosted with http",
			"http://gitlab.com",
			true,
		}, {
			"test self hosted with https",
			"https://gitlab.example.com",
			false,
		}, {
			"test self hosted with http",
			"http://gitlab.example.com",
			false,
		}, {
			"test self hosted with port",
			"http://gitlab.example.com:8080",
			false,
		}, {
			"test self hosted with a path",
			"http://gitlab.example.com/somepath",
			false,
		}, {
			"test empty url",
			"",
			true,
		}, {
			"test unexpected input",
			"\nsomethingsomething\n--;",
			false,
		}, {
			"test comment escape characters",
			`/\/\*/ /\*\//`,
			false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			result := provider.IsHosted(tt.input)
			assert.Equal(t, tt.want, result)
		})
	}
}

func (s *GitlabProviderSuite) TestListRepositories() {
	require := s.Require()
	tests := []struct {
		testDescription  string
		org              string
		expectedRepoName string
		expectedSSHURL   string
		expectedHTTPSURL string
	}{
		{
			testDescription: "Repository without organization",
			org:             gitlabUserName, expectedRepoName: "userproject",
			expectedSSHURL:   "git@gitlab.com:tester/userproject.git",
			expectedHTTPSURL: "https://gitlab.com/tester/userproject.git",
		},
		{
			testDescription: "Test Repository",
			org:             "", expectedRepoName: gitlabProjectName,
			expectedSSHURL:   "git@gitlab.com:test-user/test-project.git",
			expectedHTTPSURL: "https://gitlab.com/test-user/test-project.git",
		},
		{
			testDescription: "Organization Repository",
			org:             gitlabOrgName, expectedRepoName: "orgproject",
			expectedSSHURL:   "git@gitlab.com:testorg/orgproject.git",
			expectedHTTPSURL: "https://gitlab.com/testorg/orgproject.git",
		},
	}
	for i, tt := range tests {
		repositories, err := s.provider.GetRepositories()
		require.Nil(err)
		require.Len(repositories, 3)
		require.Equal(tt.expectedRepoName, repositories[i].Name)
		require.Equal(tt.expectedSSHURL, repositories[i].SSHURL)
		require.Equal(tt.expectedHTTPSURL, repositories[i].CloneURL)
	}
}

func (s *GitlabProviderSuite) TestGetIssues() {
	require := s.Require()
	tests := []struct {
		testDescription string
		expectedIID     int
		expectedRepo    string
		expectedTitle   string
		expectedBody    string
		expectedState   string
		labels          []provider.GitLabel
	}{
		{
			"Get issues with quoted titles",
			95,
			gitlabProjectName,
			"Change \"billmeth\" to \"Payor\" and \"tppaid\" to \"paid\"",
			"Makes more sense.",
			"closed",
			[]provider.GitLabel{},
		},
		{
			"Get issues with labels",
			94,
			gitlabProjectName,
			"Convert Demo Test data on dev01 to new data model in a new demo instance",
			"",
			"closed",
			[]provider.GitLabel{
				provider.GitLabel{
					Name: "To Do",
				},
			},
		},
		{
			"Get issues",
			1,
			gitlabProjectName,
			"Ut commodi ullam eos dolores perferendis nihil sunt.",
			"Omnis vero earum sunt corporis dolor et placeat.",
			"closed", []provider.GitLabel{},
		},
	}
	for i, tt := range tests {
		issues, err := s.provider.GetIssues(4, gitlabProjectName)
		require.Nil(err)
		require.Len(issues, 3)
		require.Equal(tt.expectedIID, issues[i].Number)
		require.Equal(tt.expectedRepo, issues[i].Repo)
		require.Equal(tt.expectedTitle, issues[i].Title)
		require.Equal(tt.expectedBody, issues[i].Body)
		require.Equal(tt.expectedState, issues[i].State)
		require.Equal(tt.labels, issues[i].Labels)
	}
}

func TestGitlabProviderSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestGitlabProviderSuite in short mode")
	} else {
		suite.Run(t, new(GitlabProviderSuite))
	}
}
