package provider

import (
	"reflect"
	"sync"
	"testing"
)

func TestNewFakeProvider(t *testing.T) {
	tests := []struct {
		name string
		want *FakeProvider
	}{
		{
			name: "test fake provider created",
			want: &FakeProvider{
				Repositories: &sync.Map{},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := NewFakeProvider(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewFakeProvider() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFakeProvider_CreateRepository(t *testing.T) {
	type fields struct {
		Repositories *sync.Map
	}
	type args struct {
		GitRepository *GitRepository
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *GitRepository
		wantErr bool
	}{
		{
			name: "test create repository",
			fields: fields{
				Repositories: &sync.Map{},
			},
			args: args{
				GitRepository: &GitRepository{
					Name:        "testrepo",
					Description: "just a test repo nothing to see here...",
				},
			},
			want: &GitRepository{
				Name: "testrepo",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			f := &FakeProvider{
				Repositories: tt.fields.Repositories,
			}
			got, err := f.CreateRepository(tt.args.GitRepository)
			if (err != nil) != tt.wantErr {
				t.Errorf("FakeProvider.CreateRepository() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FakeProvider.CreateRepository() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFakeProvider_CreateLabel(t *testing.T) {
	type fields struct {
		Repositories *sync.Map
	}
	type args struct {
		label *GitLabel
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *GitLabel
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			f := &FakeProvider{
				Repositories: tt.fields.Repositories,
			}
			got, err := f.CreateLabel(tt.args.label)
			if (err != nil) != tt.wantErr {
				t.Errorf("FakeProvider.CreateLabel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FakeProvider.CreateLabel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFakeProvider_CreateIssueComment(t *testing.T) {
	type fields struct {
		Repositories *sync.Map
	}
	type args struct {
		comment *GitIssueComment
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			f := &FakeProvider{
				Repositories: tt.fields.Repositories,
			}
			if err := f.CreateIssueComment(tt.args.comment.IssueNum, tt.args.comment); (err != nil) != tt.wantErr {
				t.Errorf("FakeProvider.CreateIssueComment() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// func TestFakeProvider_CreateIssue(t *testing.T) {
// 	type fields struct {
// 		Repositories *sync.Map
// 	}
// 	type args struct {
// 		repo  string
// 		issue *GitIssue
// 	}
// 	repos := &sync.Map{}
// 	testRepo := &FakeRepository{
// 		GitRepo: &GitRepository{
// 			Name: "testrepo",
// 		},
// 		Issues:      &sync.Map{},
// 		Private:     true,
// 		Description: "just a test repo nothing to see here...",
// 		issueCount:  0,
// 	}

// 	tests := []struct {
// 		name    string
// 		fields  fields
// 		input   *GitIssue
// 		want    *GitIssue
// 		wantErr bool
// 	}{
// 		{
// 			name: "test create issue",
// 			fields: fields{
// 				Repositories: repos.Store("testrepo", testRepo),
// 			},
// 			input: &GitIssue{
// 				Repo:   "testrepo",
// 				Number: 1,
// 				Title:  "test issue",
// 				Body:   "something went wrong",
// 				State:  "closed",
// 				Labels: ToGitLabels([]string{"good first issue", "test issue"}),
// 			},
// 			want: &GitIssue{
// 				Repo:   "testrepo",
// 				Number: 1,
// 				Title:  "test issue",
// 				Body:   "something went wrong",
// 				State:  "closed",
// 				Labels: ToGitLabels([]string{"good first issue", "test issue"}),
// 			},
// 			wantErr: false,
// 		},
// 	}
// 	for _, tt := range tests {
// 		tt := tt
// 		t.Run(tt.name, func(t *testing.T) {
// 			f := &FakeProvider{
// 				Repositories: tt.fields.Repositories,
// 			}
// 			got, err := f.CreateIssue(tt.input)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("FakeProvider.CreateIssue() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			if !reflect.DeepEqual(got, tt.want) {
// 				t.Errorf("FakeProvider.CreateIssue() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }
func TestFakeProvider_CreateIssue(t *testing.T) {
	type fields struct {
		Repositories *sync.Map
	}
	type args struct {
		issue *GitIssue
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *GitIssue
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			f := &FakeProvider{
				Repositories: tt.fields.Repositories,
			}
			got, err := f.CreateIssue(tt.args.issue)
			if (err != nil) != tt.wantErr {
				t.Errorf("FakeProvider.CreateIssue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FakeProvider.CreateIssue() = %v, want %v", got, tt.want)
			}
		})
	}
}
