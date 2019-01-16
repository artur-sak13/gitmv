package main

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/go-github/v21/github"
	gitlab "github.com/xanzy/go-gitlab"
)

func Test_main(t *testing.T) {
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			main()
		})
	}
}

func Test_runCommand(t *testing.T) {
	type args struct {
		ctx  context.Context
		args []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := runCommand(tt.args.ctx, tt.args.args); (err != nil) != tt.wantErr {
				t.Errorf("runCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_newGitlabClient(t *testing.T) {
	tests := []struct {
		name    string
		want    *gitlab.Client
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newGitlabClient()
			if (err != nil) != tt.wantErr {
				t.Errorf("newGitlabClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newGitlabClient() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newGitHubClient(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
		want *github.Client
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newGitHubClient(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newGitHubClient() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_migrator_getProjects(t *testing.T) {
	type fields struct {
		ghClient *github.Client
		glClient *gitlab.Client
	}
	type args struct {
		page    int
		perPage int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &migrator{
				ghClient: tt.fields.ghClient,
				glClient: tt.fields.glClient,
			}
			if err := m.getProjects(tt.args.page, tt.args.perPage); (err != nil) != tt.wantErr {
				t.Errorf("migrator.getProjects() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_migrator_getIssues(t *testing.T) {
	type fields struct {
		ghClient *github.Client
		glClient *gitlab.Client
	}
	type args struct {
		pid     int
		page    int
		perPage int
		list    []*gitlab.Issue
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*gitlab.Issue
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &migrator{
				ghClient: tt.fields.ghClient,
				glClient: tt.fields.glClient,
			}
			got, err := m.getIssues(tt.args.pid, tt.args.page, tt.args.perPage, tt.args.list)
			if (err != nil) != tt.wantErr {
				t.Errorf("migrator.getIssues() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("migrator.getIssues() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_migrator_getMilestones(t *testing.T) {
	type fields struct {
		ghClient *github.Client
		glClient *gitlab.Client
	}
	type args struct {
		pid     int
		page    int
		perPage int
		list    []*gitlab.Milestone
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*gitlab.Milestone
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &migrator{
				ghClient: tt.fields.ghClient,
				glClient: tt.fields.glClient,
			}
			got, err := m.getMilestones(tt.args.pid, tt.args.page, tt.args.perPage, tt.args.list)
			if (err != nil) != tt.wantErr {
				t.Errorf("migrator.getMilestones() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("migrator.getMilestones() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_migrator_getMergeRequests(t *testing.T) {
	type fields struct {
		ghClient *github.Client
		glClient *gitlab.Client
	}
	type args struct {
		pid     int
		page    int
		perPage int
		list    []*gitlab.MergeRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*gitlab.MergeRequest
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &migrator{
				ghClient: tt.fields.ghClient,
				glClient: tt.fields.glClient,
			}
			got, err := m.getMergeRequests(tt.args.pid, tt.args.page, tt.args.perPage, tt.args.list)
			if (err != nil) != tt.wantErr {
				t.Errorf("migrator.getMergeRequests() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("migrator.getMergeRequests() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_migrator_getLabels(t *testing.T) {
	type fields struct {
		ghClient *github.Client
		glClient *gitlab.Client
	}
	type args struct {
		pid     int
		page    int
		perPage int
		list    []*gitlab.Label
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*gitlab.Label
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &migrator{
				ghClient: tt.fields.ghClient,
				glClient: tt.fields.glClient,
			}
			got, err := m.getLabels(tt.args.pid, tt.args.page, tt.args.perPage, tt.args.list)
			if (err != nil) != tt.wantErr {
				t.Errorf("migrator.getLabels() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("migrator.getLabels() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_migrator_getWikis(t *testing.T) {
	type fields struct {
		ghClient *github.Client
		glClient *gitlab.Client
	}
	type args struct {
		pid     int
		page    int
		perPage int
		list    []*gitlab.Wiki
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*gitlab.Wiki
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &migrator{
				ghClient: tt.fields.ghClient,
				glClient: tt.fields.glClient,
			}
			got, err := m.getWikis(tt.args.pid, tt.args.page, tt.args.perPage, tt.args.list)
			if (err != nil) != tt.wantErr {
				t.Errorf("migrator.getWikis() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("migrator.getWikis() = %v, want %v", got, tt.want)
			}
		})
	}
}
