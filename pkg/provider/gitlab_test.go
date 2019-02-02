package provider_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/artur-sak13/gitmv/pkg/provider"

	"gotest.tools/assert"

	"github.com/stretchr/testify/suite"
	gitlab "github.com/xanzy/go-gitlab"
)

const (
	gitlabUserName    = "tester"
	gitlabOrgName     = "testorg"
	gitlabProjectName = "test-project"
	gitlabProjectID   = "8675309"
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
	c.SetBaseURL(server.URL)

	// Gitlab provider that we want to test
	prov, _ := provider.WithGitlabClient(server.URL, "test", c)

	return mux, server, prov.(*provider.GitlabProvider)
}

func configureGitlabMock(s *GitlabProviderSuite, mux *http.ServeMux) {
	mux.HandleFunc("/api/v4/projects", func(w http.ResponseWriter, r *http.Request) {
		src, err := ioutil.ReadFile("test_data/gitlab/projects.json")

		s.Require().Nil(err)
		w.Write(src)
	})

	mux.HandleFunc(fmt.Sprintf("/api/v4/projects/%d/issues", 4), func(w http.ResponseWriter, r *http.Request) {
		src, err := ioutil.ReadFile("test_data/gitlab/issues.json")

		s.Require().Nil(err)
		w.Write(src)
	})

	// gitlabRouter := testutil.Router{
	// 	fmt.Sprintf("/api/v4/projects/%s?page=1&per_page=100&statistics=true", gitlabProjectID): testutil.MethodMap{
	// 		"GET": "project.json",
	// 	},
	// }
	// for path, methodMap := range gitlabRouter {
	// 	mux.HandleFunc(path, testutil.GetMockAPIResponseFromFile("test_data/gitlab", methodMap))
	// }
}

func TestIsHosted(t *testing.T) {
	t.Parallel()
	tests := []struct {
		testDescription string
		input           string
		want            bool
	}{
		{
			"Hosted-with-HTTPS",
			"https://gitlab.com",
			true,
		}, {
			"Hosted-with-HTTP",
			"http://gitlab.com",
			true,
		}, {
			"Self-hosted-with-HTTPS",
			"https://gitlab.example.com",
			false,
		}, {
			"Self-hosted-with-HTTP",
			"http://gitlab.example.com",
			false,
		}, {
			"Self-hosted-with-port",
			"http://gitlab.example.com:8080",
			false,
		}, {
			"Self-hosted-with-path",
			"http://gitlab.example.com/somepath",
			false,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.testDescription, func(t *testing.T) {
			result := provider.IsHosted(test.input)
			assert.Equal(t, test.want, result)
		})
	}
}

func (s *GitlabProviderSuite) TestListRepositories() {
	require := s.Require()
	scenarios := []struct {
		testDescription  string
		org              string
		expectedRepoName string
		expectedSSHURL   string
		expectedHTTPSURL string
		expectedHTMLURL  string
	}{
		{
			"Repository without organization",
			gitlabUserName, "userproject",
			"git@gitlab.com:tester/userproject.git",
			"https://gitlab.com/tester/userproject.git",
			"https://gitlab.com/tester/userproject",
		},
		{
			"Test Repository",
			"", gitlabProjectName,
			"git@gitlab.com:test-user/test-project.git",
			"https://gitlab.com/test-user/test-project.git",
			"https://gitlab.com/test-user/test-project",
		},
		{
			"Organization Repository",
			gitlabOrgName,
			"orgproject",
			"git@gitlab.com:testorg/orgproject.git",
			"https://gitlab.com/testorg/orgproject.git",
			"https://gitlab.com/testorg/orgproject",
		},
	}
	for i, scen := range scenarios {
		repositories, err := s.provider.ListRepositories()
		require.Nil(err)
		require.Len(repositories, 3)
		require.Equal(scen.expectedRepoName, repositories[i].Name)
		require.Equal(scen.expectedSSHURL, repositories[i].SSHURL)
		require.Equal(scen.expectedHTTPSURL, repositories[i].CloneURL)
		require.Equal(scen.expectedHTMLURL, repositories[i].HTMLURL)
	}
}

func (s *GitlabProviderSuite) TestGetIssues() {
	require := s.Require()
	closed := "closed"
	scenarios := []struct {
		testDescription string
		expectedOwner   string
		expectedRepo    string
		expectedTitle   string
		expectedBody    string
		expectedState   *string
		labels          []provider.GitLabel
	}{
		{
			"Get issues",
			gitlabOrgName, gitlabProjectName,
			"Ut commodi ullam eos dolores perferendis nihil sunt.",
			"Omnis vero earum sunt corporis dolor et placeat.",
			&closed, []provider.GitLabel{},
		},
	}
	for i, scen := range scenarios {
		issues, err := s.provider.GetIssues(4, gitlabOrgName, gitlabProjectName)
		require.Nil(err)
		require.Len(issues, 1)
		require.Equal(scen.expectedOwner, issues[i].Owner)
		require.Equal(scen.expectedRepo, issues[i].Repo)
		require.Equal(scen.expectedTitle, issues[i].Title)
		require.Equal(scen.expectedBody, issues[i].Body)
		require.Equal(*scen.expectedState, *issues[i].State)
		require.Equal(scen.labels, issues[i].Labels)
	}
}

func TestGitlabProviderSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestGitlabProviderSuite in short mode")
	} else {
		suite.Run(t, new(GitlabProviderSuite))
	}
}
