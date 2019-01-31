package client_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/artur-sak13/gitmv/client"

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
	provider *client.GitlabProvider
}

func (s *GitlabProviderSuite) SetupSuite() {
	mux, server, provider := setup(s)
	s.mux = mux
	s.server = server
	s.provider = provider
}

func (s *GitlabProviderSuite) TearDownSuite() {
	s.server.Close()
}

func setup(s *GitlabProviderSuite) (*http.ServeMux, *httptest.Server, *client.GitlabProvider) {
	mux := http.NewServeMux()
	configureGitlabMock(s, mux)

	server := httptest.NewServer(mux)

	c := gitlab.NewClient(nil, "")
	c.SetBaseURL(server.URL)

	// Gitlab provider that we want to test
	provider, _ := client.WithGitlabClient(server.URL, "test", c)

	return mux, server, provider.(*client.GitlabProvider)
}

func configureGitlabMock(s *GitlabProviderSuite, mux *http.ServeMux) {
	mux.HandleFunc("/api/v4/projects", func(w http.ResponseWriter, r *http.Request) {
		src, err := ioutil.ReadFile("test_data/gitlab/projects.json")

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

// func TestIsSelfHosted(t *testing.T) {
// 	type args struct {
// 		u string
// 	}
// 	tests := []struct {
// 		name string
// 		args args
// 		want bool
// 	}{
// 		// TODO: Add test cases.
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			if got := client.IsSelfHosted(tt.args.u); got != tt.want {
// 				t.Errorf("IsSelfHosted() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }

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

func TestGitlabProviderSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestGitlabProviderSuite in short mode")
	} else {
		suite.Run(t, new(GitlabProviderSuite))
	}
}
