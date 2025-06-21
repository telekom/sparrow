// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package gitlab

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/telekom/sparrow/pkg/checks"
	"github.com/telekom/sparrow/pkg/sparrow/targets/remote"
	"github.com/telekom/sparrow/test"
)

func Test_gitlab_fetchFileList(t *testing.T) {
	type file struct {
		Name string `json:"name"`
	}
	tests := []struct {
		name     string
		want     []string
		wantErr  bool
		mockBody []file
		mockCode int
	}{
		{
			name:     "success - 0 targets",
			want:     nil,
			wantErr:  false,
			mockCode: http.StatusOK,
			mockBody: []file{},
		},
		{
			name:     "success - 0 targets with 1 file",
			want:     nil,
			wantErr:  false,
			mockCode: http.StatusOK,
			mockBody: []file{
				{
					Name: "README.md",
				},
			},
		},
		{
			name: "success - 1 target",
			want: []string{
				"sparrow.telekom.com.json",
			},
			wantErr:  false,
			mockCode: http.StatusOK,
			mockBody: []file{
				{
					Name: "sparrow.telekom.com.json",
				},
			},
		},
		{
			name: "success - 2 targets",
			want: []string{
				"sparrow.telekom.com.json",
				"sparrow2.telekom.com.json",
			},
			wantErr:  false,
			mockCode: http.StatusOK,
			mockBody: []file{
				{
					Name: "sparrow.telekom.com.json",
				},
				{
					Name: "sparrow2.telekom.com.json",
				},
			},
		},
		{
			name:     "failure - API error",
			want:     nil,
			wantErr:  true,
			mockCode: http.StatusInternalServerError,
		},
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := httpmock.NewJsonResponderOrPanic(tt.mockCode, tt.mockBody)
			httpmock.RegisterResponder(http.MethodGet, fmt.Sprintf("%s/projects/1/repository/tree?order_by=id&pagination=keyset&per_page=%d&ref=main&sort=asc", test.GitlabAPIURL, paginationPerPage), resp)

			g := &client{
				config: Config{
					BaseURL:   test.GitlabBaseURL,
					ProjectID: 1,
					Token:     "test",
					Branch:    fallbackBranch,
				},
				client: http.DefaultClient,
			}
			got, err := g.fetchFileList(t.Context())
			if (err != nil) != tt.wantErr {
				t.Fatalf("FetchFiles() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("FetchFiles() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// The filelist and url are the same, so we HTTP responders can
// be created without much hassle
func Test_gitlab_FetchFiles(t *testing.T) {
	type file struct {
		Name string `json:"name"`
	}

	tests := []struct {
		name     string
		want     []checks.GlobalTarget
		fileList []file
		wantErr  bool
		mockCode int
	}{
		{
			name:     "success - 0 targets",
			want:     nil,
			wantErr:  false,
			mockCode: http.StatusOK,
		},
		{
			name: "success - 1 target",
			want: []checks.GlobalTarget{
				{
					URL:      test.ToURLOrFail(t, "sparrow.telekom.com"),
					LastSeen: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			fileList: []file{
				{
					Name: "sparrow.telekom.com.json",
				},
			},
			wantErr:  false,
			mockCode: http.StatusOK,
		},
		{
			name: "success - 2 targets",
			want: []checks.GlobalTarget{
				{
					URL:      test.ToURLOrFail(t, "sparrow.telekom.com"),
					LastSeen: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					URL:      test.ToURLOrFail(t, "sparrow2.telekom.com"),
					LastSeen: time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			fileList: []file{
				{
					Name: "sparrow.telekom.com.json",
				},
				{
					Name: "sparrow2.telekom.com.json",
				},
			},
			wantErr:  false,
			mockCode: http.StatusOK,
		},
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	g := &client{
		config: Config{
			BaseURL:   test.GitlabBaseURL,
			ProjectID: 1,
			Token:     "test",
			Branch:    fallbackBranch,
		},
		client: http.DefaultClient,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i, target := range tt.want {
				resp := httpmock.NewJsonResponderOrPanic(tt.mockCode, target)
				httpmock.RegisterResponder(http.MethodGet, fmt.Sprintf("%s/projects/1/repository/files/%s/raw?ref=main", test.GitlabAPIURL, tt.fileList[i].Name), resp)
			}
			resp := httpmock.NewJsonResponderOrPanic(tt.mockCode, tt.fileList)
			httpmock.RegisterResponder(http.MethodGet, fmt.Sprintf("%s/projects/1/repository/tree?order_by=id&pagination=keyset&per_page=%d&ref=main&sort=asc", test.GitlabAPIURL, paginationPerPage), resp)

			got, err := g.FetchFiles(t.Context())
			if (err != nil) != tt.wantErr {
				t.Fatalf("FetchFiles() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("FetchFiles() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_gitlab_fetchFiles_error_cases(t *testing.T) {
	type file struct {
		Name string `json:"name"`
	}
	type mockResponses struct {
		response checks.GlobalTarget
		err      bool
	}

	tests := []struct {
		name          string
		mockResponses []mockResponses
		fileList      []file
	}{
		{
			name: "failure - direct API error",
			mockResponses: []mockResponses{
				{
					err: true,
				},
			},
			fileList: []file{
				{
					Name: "test",
				},
			},
		},
		{
			name: "failure - API error after one successful request",
			mockResponses: []mockResponses{
				{
					response: checks.GlobalTarget{
						URL:      test.ToURLOrFail(t, "test"),
						LastSeen: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					},
					err: false,
				},
				{
					response: checks.GlobalTarget{},
					err:      true,
				},
			},
			fileList: []file{
				{Name: "test"},
				{Name: "test2-will-fail"},
			},
		},
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	g := &client{
		config: Config{
			BaseURL:   test.GitlabBaseURL,
			ProjectID: 1,
			Token:     "test",
			Branch:    fallbackBranch,
		},
		client: http.DefaultClient,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i, target := range tt.mockResponses {
				if target.err {
					errResp := httpmock.NewStringResponder(http.StatusInternalServerError, "")
					httpmock.RegisterResponder(http.MethodGet, fmt.Sprintf("%s/projects/1/repository/files/%s/raw?ref=main", test.GitlabAPIURL, tt.fileList[i].Name), errResp)
					continue
				}
				resp := httpmock.NewJsonResponderOrPanic(http.StatusOK, target)
				httpmock.RegisterResponder(http.MethodGet, fmt.Sprintf("%s/projects/1/repository/files/%s/raw?ref=main", test.GitlabAPIURL, tt.fileList[i].Name), resp)
			}

			_, err := g.FetchFiles(t.Context())
			if err == nil {
				t.Fatalf("Expected error but got none.")
			}
		})
	}
}

func TestClient_PutFile(t *testing.T) { //nolint:dupl // no need to refactor yet
	now := time.Now()
	tests := []struct {
		name     string
		file     remote.File
		mockCode int
		wantErr  bool
	}{
		{
			name: "success",
			file: remote.File{
				AuthorEmail: "test@sparrow",
				AuthorName:  "sparrow",
				Content: checks.GlobalTarget{
					URL:      test.ToURLOrFail(t, "https://sparrow.telekom.com"),
					LastSeen: now,
				},
				CommitMessage: "test-commit",
				Name:          "sparrow.telekom.com.json",
			},
			mockCode: http.StatusOK,
		},
		{
			name: "failure - API error",
			file: remote.File{
				AuthorEmail: "test@sparrow",
				AuthorName:  "sparrow",
				Content: checks.GlobalTarget{
					URL:      test.ToURLOrFail(t, "https://sparrow.telekom.com"),
					LastSeen: now,
				},
				CommitMessage: "test-commit",
				Name:          "sparrow.telekom.com.json",
			},
			mockCode: http.StatusInternalServerError,
			wantErr:  true,
		},
		{
			name:    "failure - empty file",
			wantErr: true,
		},
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	g := &client{
		config: Config{
			BaseURL:   test.GitlabBaseURL,
			ProjectID: 1,
			Token:     "test",
			Branch:    fallbackBranch,
		},
		client: http.DefaultClient,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr {
				resp := httpmock.NewStringResponder(tt.mockCode, "")
				httpmock.RegisterResponder(http.MethodPut, fmt.Sprintf("%s/projects/1/repository/files/%s", test.GitlabAPIURL, tt.file.Name), resp)
			} else {
				resp := httpmock.NewJsonResponderOrPanic(tt.mockCode, tt.file)
				httpmock.RegisterResponder(http.MethodPut, fmt.Sprintf("%s/projects/1/repository/files/%s", test.GitlabAPIURL, tt.file.Name), resp)
			}

			if err := g.PutFile(t.Context(), tt.file); (err != nil) != tt.wantErr {
				t.Fatalf("PutFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_PostFile(t *testing.T) { //nolint:dupl // no need to refactor yet
	now := time.Now()
	tests := []struct {
		name     string
		file     remote.File
		mockCode int
		wantErr  bool
	}{
		{
			name: "success",
			file: remote.File{
				AuthorEmail: "test@sparrow",
				AuthorName:  "sparrow",
				Content: checks.GlobalTarget{
					URL:      test.ToURLOrFail(t, "https://sparrow.telekom.com"),
					LastSeen: now,
				},
				CommitMessage: "test-commit",
				Name:          "sparrow.telekom.com.json",
			},
			mockCode: http.StatusCreated,
		},
		{
			name: "failure - API error",
			file: remote.File{
				AuthorEmail: "test@sparrow",
				AuthorName:  "sparrow",
				Content: checks.GlobalTarget{
					URL:      test.ToURLOrFail(t, "https://sparrow.telekom.com"),
					LastSeen: now,
				},
				CommitMessage: "test-commit",
				Name:          "sparrow.telekom.com.json",
			},
			mockCode: http.StatusInternalServerError,
			wantErr:  true,
		},
		{
			name:    "failure - empty file",
			wantErr: true,
		},
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	g := &client{
		config: Config{
			BaseURL:   test.GitlabBaseURL,
			ProjectID: 1,
			Token:     "test",
			Branch:    fallbackBranch,
		},
		client: http.DefaultClient,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr {
				resp := httpmock.NewStringResponder(tt.mockCode, "")
				httpmock.RegisterResponder(http.MethodPost, fmt.Sprintf("%s/projects/1/repository/files/%s", test.GitlabAPIURL, tt.file.Name), resp)
			} else {
				resp := httpmock.NewJsonResponderOrPanic(tt.mockCode, tt.file)
				httpmock.RegisterResponder(http.MethodPost, fmt.Sprintf("%s/projects/1/repository/files/%s", test.GitlabAPIURL, tt.file.Name), resp)
			}

			if err := g.PostFile(t.Context(), tt.file); (err != nil) != tt.wantErr {
				t.Fatalf("PostFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_DeleteFile(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		mockCode int
		wantErr  bool
	}{
		{
			name:     "success",
			fileName: "sparrow.telekom.com.json",
			mockCode: http.StatusNoContent,
		},
		{
			name:     "failure - API error",
			fileName: "sparrow.telekom.com.json",
			mockCode: http.StatusInternalServerError,
			wantErr:  true,
		},
		{
			name:     "failure - empty file",
			wantErr:  true,
			fileName: "",
		},
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	projID := 1
	g := &client{
		config: Config{
			BaseURL:   test.GitlabBaseURL,
			ProjectID: 1,
			Token:     "test",
			Branch:    fallbackBranch,
		},
		client: http.DefaultClient,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := httpmock.NewStringResponder(tt.mockCode, "")
			httpmock.RegisterResponder(http.MethodDelete, fmt.Sprintf("%s/projects/%d/repository/files/%s", test.GitlabAPIURL, projID, tt.fileName), resp)

			f := remote.File{
				Name:          tt.fileName,
				CommitMessage: "Deleted registration file",
				AuthorName:    "sparrow-test",
				AuthorEmail:   "sparrow-test@sparrow",
			}
			if err := g.DeleteFile(t.Context(), f); (err != nil) != tt.wantErr {
				t.Fatalf("DeleteFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_fetchDefaultBranch(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		want     string
		response any
	}{
		{
			name: "success",
			code: http.StatusOK,
			want: "master",
			response: []branch{
				{
					Name:    "master",
					Default: true,
				},
			},
		},
		{
			name: "success - multiple branches",
			code: http.StatusOK,
			want: "release",
			response: []branch{
				{
					Name:    "master",
					Default: false,
				},
				{
					Name:    "release",
					Default: true,
				},
			},
		},
		{
			name: "success - multiple branches without default",
			code: http.StatusOK,
			want: fallbackBranch,
			response: []branch{
				{
					Name:    "master",
					Default: false,
				},
				{
					Name:    "release",
					Default: false,
				},
			},
		},
		{
			name: "failure - API error",
			code: http.StatusInternalServerError,
			want: fallbackBranch,
		},
		{
			name: "failure - invalid response",
			code: http.StatusOK,
			want: fallbackBranch,
			response: struct {
				Invalid bool `json:"invalid"`
			}{
				Invalid: true,
			},
		},
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	g := &client{
		config: Config{
			BaseURL:   test.GitlabBaseURL,
			ProjectID: 1,
			Token:     "test",
		},
		client: http.DefaultClient,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := httpmock.NewJsonResponderOrPanic(tt.code, tt.response)
			httpmock.RegisterResponder(http.MethodGet, fmt.Sprintf("%s/projects/1/repository/branches", test.GitlabAPIURL), resp)

			got := g.fetchDefaultBranch()
			if got != tt.want {
				t.Errorf("(*client).fetchDefaultBranch() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_client_fetchNextFileList(t *testing.T) {
	type mockRespFile struct {
		Name string `json:"name"`
	}
	type mockResponder struct {
		reqUrl     string
		linkHeader string
		statusCode int
		response   []mockRespFile
	}

	tests := []struct {
		name    string
		mock    []mockResponder
		want    []string
		wantErr bool
	}{
		{
			name: "success - no pagination",
			mock: []mockResponder{
				{
					reqUrl:     fmt.Sprintf("%s/pagination", test.GitlabAPIURL),
					linkHeader: "",
					statusCode: http.StatusOK,
					response:   []mockRespFile{{Name: "file1.json"}, {Name: "file2.json"}},
				},
			},
			want: []string{
				"file1.json",
				"file2.json",
			},
			wantErr: false,
		},
		{
			name: "success - with pagination",
			mock: []mockResponder{
				{
					reqUrl:     fmt.Sprintf("%s/pagination", test.GitlabAPIURL),
					linkHeader: fmt.Sprintf("<%s/pagination?page=2>; rel=\"next\"", test.GitlabAPIURL),
					statusCode: http.StatusOK,
					response:   []mockRespFile{{Name: "file1.json"}},
				},
				{
					reqUrl:     fmt.Sprintf("%s/pagination?page=2", test.GitlabAPIURL),
					linkHeader: fmt.Sprintf("<%s/pagination?page=3>; rel=\"next\"", test.GitlabAPIURL),
					statusCode: http.StatusOK,
					response:   []mockRespFile{{Name: "file2.json"}},
				},
				{
					reqUrl:     fmt.Sprintf("%s/pagination?page=3", test.GitlabAPIURL),
					linkHeader: "",
					statusCode: http.StatusOK,
					response:   []mockRespFile{{Name: "file3.json"}},
				},
			},
			want:    []string{"file1.json", "file2.json", "file3.json"},
			wantErr: false,
		},
		{
			name: "fail - bad request while paginated requests",
			mock: []mockResponder{
				{
					reqUrl:     fmt.Sprintf("%s/pagination", test.GitlabAPIURL),
					linkHeader: fmt.Sprintf("<%s/pagination?page=2>; rel=\"next\"", test.GitlabAPIURL),
					statusCode: http.StatusOK,
					response:   []mockRespFile{{Name: "file1.json"}},
				},
				{
					reqUrl:     fmt.Sprintf("%s/pagination?page=2", test.GitlabAPIURL),
					linkHeader: "",
					statusCode: http.StatusBadRequest,
					response:   []mockRespFile{{Name: "file2.json"}},
				},
			},
			want:    []string{},
			wantErr: true,
		},
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	c := &client{
		config: Config{
			BaseURL:   test.GitlabBaseURL,
			ProjectID: 1,
			Token:     "test",
		},
		client: http.DefaultClient,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// prepare http mock responder for paginated requests
			for _, responder := range tt.mock {
				httpmock.RegisterResponder(http.MethodGet, responder.reqUrl, func(req *http.Request) (*http.Response, error) {
					// Check if header are properly set
					token := req.Header.Get("PRIVATE-TOKEN")
					cType := req.Header.Get("Content-Type")
					if token == "" || cType == "" {
						t.Error("Some header not properly set", "PRIVATE-TOKEN", token != "", "Content-Type", cType != "")
					}

					resp, err := httpmock.NewJsonResponse(responder.statusCode, responder.response)

					// Add link header for next page (pagination)
					resp.Header.Set(linkHeader, responder.linkHeader)
					return resp, err
				})
			}

			got, err := c.fetchNextFileList(t.Context(), tt.mock[0].reqUrl)
			if (err != nil) != tt.wantErr {
				t.Errorf("fetchNextFileList() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fetchNextFileList() got = %v, want %v", got, tt.want)
			}
		})
	}
}
