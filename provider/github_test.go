package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"testing"

	"github.com/artur-sak13/gitmv/auth"

	"github.com/google/go-github/v21/github"
)

const (
	baseURLPath            = "/api-v3"
	githubOrgName          = "o"
	mediaTypeImportPreview = "application/vnd.github.barred-rock-preview"
)

func setup() (*GithubProvider, *http.ServeMux, string, func()) {
	mux := http.NewServeMux()

	apiHandler := http.NewServeMux()
	apiHandler.Handle(baseURLPath+"/", http.StripPrefix(baseURLPath, mux))
	apiHandler.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintln(os.Stderr, "FAIL: Client.BaseURL path prefix is not preserved in the request URL:")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "\t"+req.URL.String())
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "\tDid you accidentally use an absolute endpoint URL rather than relative?")
		fmt.Fprintln(os.Stderr, "\tSee https://github.com/google/go-github/issues/752 for information.")
		http.Error(w, "github.Client.BaseURL path prefix is not preserved in the request URL.", http.StatusInternalServerError)
	})

	server := httptest.NewServer(apiHandler)

	client := github.NewClient(nil)
	u, _ := url.Parse(server.URL + baseURLPath + "/")
	client.BaseURL = u
	client.UploadURL = u

	id := auth.NewAuthID(server.URL, "p", "~/fake/.ssh/id_rsa", githubOrgName)
	prov := WithGithubClient(context.Background(), client, id)

	return prov.(*GithubProvider), mux, server.URL, server.Close
}

func TestMigrateRepo(t *testing.T) {
	prov, mux, _, teardown := setup()
	defer teardown()
	repo := &GitRepository{
		Name:     "r",
		CloneURL: "r",
		Owner:    "u",
	}

	input := &github.Import{
		VCS:         github.String("git"),
		VCSURL:      &repo.CloneURL,
		VCSUsername: github.String("u"),
		VCSPassword: github.String("p"),
	}

	mux.HandleFunc("/repos/o/r/import", func(w http.ResponseWriter, r *http.Request) {
		v := new(github.Import)
		json.NewDecoder(r.Body).Decode(v)

		testMethod(t, r, "PUT")
		testHeader(t, r, "Accept", mediaTypeImportPreview)
		if !reflect.DeepEqual(v, input) {
			t.Errorf("Request body = %+v, want %+v", v, input)
		}

		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"status":"importing"}`)
	})

	got, err := prov.MigrateRepo(repo, "p")
	if err != nil {
		t.Errorf("StartImport returned error: %v", err)
	}
	want := "importing"
	if !reflect.DeepEqual(got, want) {
		t.Errorf("StartImport = %+v, want %+v", got, want)
	}
}

func TestImportProgress(t *testing.T) {
	prov, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/import", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testHeader(t, r, "Accept", mediaTypeImportPreview)
		fmt.Fprint(w, `{"status":"complete"}`)
	})

	got, err := prov.GetImportProgress("r")
	if err != nil {
		t.Errorf("ImportProgress returned error: %v", err)
	}
	want := "complete"
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ImportProgress = %+v, want %+v", got, want)
	}
}

func testMethod(t *testing.T, r *http.Request, want string) {
	if got := r.Method; got != want {
		t.Errorf("Request method: %v, want %v", got, want)
	}
}

func testHeader(t *testing.T, r *http.Request, header string, want string) {
	if got := r.Header.Get(header); got != want {
		t.Errorf("Header.Get(%q) returned %q, want %q", header, got, want)
	}
}
