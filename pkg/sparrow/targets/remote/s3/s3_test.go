// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package s3

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/telekom/sparrow/pkg/checks"
	"github.com/telekom/sparrow/pkg/sparrow/targets/remote"
)

// listBucketResultXML is a template for ListObjectsV2 responses.
// Parameters: bucket name, prefix, then pairs of (key, size) as Contents entries.
const listBucketResultXML = `<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult>
  <Name>%s</Name>
  <Prefix>%s</Prefix>
  <IsTruncated>false</IsTruncated>
  %s
</ListBucketResult>`

// listBucketContentXML is a single Contents entry for listBucketResultXML.
const listBucketContentXML = `<Contents>
    <Key>%s</Key>
    <Size>%d</Size>
  </Contents>`

// newTestClient creates a client pointing at the given httptest.Server.
func newTestClient(t *testing.T, server *httptest.Server, prefix string) remote.Interactor {
	t.Helper()
	endpoint := strings.TrimPrefix(server.URL, "http://")
	useSSL := false
	c, err := New(&Config{
		Endpoint: endpoint,
		Bucket:   "test-bucket",
		Region:   "us-east-1",
		Prefix:   prefix,
		UseSSL:   &useSSL,
		Auth: AuthConfig{
			Provider: "static",
			Static: StaticAuthConfig{
				AccessKeyID:     "minioadmin",
				SecretAccessKey: "minioadmin",
			},
		},
	})
	require.NoError(t, err)
	return c
}

// writeObjectResponse writes a JSON body with the standard headers
// minio-go requires (Content-Length, Last-Modified, ETag).
func writeObjectResponse(w http.ResponseWriter, data []byte, etag string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
	w.Header().Set("ETag", etag)
	w.Write(data) //nolint:errcheck
}

func TestNew_StaticAuth(t *testing.T) {
	i, err := New(&Config{
		Endpoint: "s3.amazonaws.com",
		Bucket:   "test-bucket",
		Region:   "eu-central-1",
		Auth: AuthConfig{
			Provider: "static",
			Static: StaticAuthConfig{
				AccessKeyID:     "test-access-key",
				SecretAccessKey: "test-secret-key",
			},
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, i)
}

func TestNew_DefaultRegion(t *testing.T) {
	i, err := New(&Config{
		Endpoint: "s3.amazonaws.com",
		Bucket:   "test-bucket",
		Auth: AuthConfig{
			Provider: "static",
			Static: StaticAuthConfig{
				AccessKeyID:     "test",
				SecretAccessKey: "test",
			},
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, i)
}

func TestNew_OIDCReturnsError(t *testing.T) {
	_, err := New(&Config{
		Endpoint: "s3.amazonaws.com",
		Bucket:   "test-bucket",
		Auth: AuthConfig{
			Provider: "oidc",
			OIDC: OIDCAuthConfig{ //gosec:disable G101 -- false positive on test fixture
				TokenPath: "/var/run/secrets/token",
				RoleARN:   "arn:aws:iam::123:role/test",
			},
		},
	})
	require.ErrorIs(t, err, ErrOIDCNotImplemented)
}

func TestNew_UnknownProvider(t *testing.T) {
	_, err := New(&Config{
		Endpoint: "s3.amazonaws.com",
		Bucket:   "test-bucket",
		Auth: AuthConfig{
			Provider: "unknown",
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown auth provider")
}

func TestClient_ObjectKey(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		fileName string
		want     string
	}{
		{
			name:     "no prefix",
			prefix:   "",
			fileName: "sparrow-1.json",
			want:     "sparrow-1.json",
		},
		{
			name:     "with prefix",
			prefix:   "targets",
			fileName: "sparrow-1.json",
			want:     "targets/sparrow-1.json",
		},
		{
			name:     "prefix with trailing slash",
			prefix:   "targets/",
			fileName: "sparrow-1.json",
			want:     "targets/sparrow-1.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &client{config: Config{Prefix: tt.prefix}}
			got := c.objectKey(tt.fileName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_DeleteFile_EmptyName(t *testing.T) {
	c := &client{config: Config{Bucket: "test"}}
	err := c.DeleteFile(context.Background(), remote.File{Name: ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "filename is empty")
}

func TestClient_FetchFiles(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	target1 := checks.GlobalTarget{Url: "https://sparrow-1.example.com", LastSeen: now}
	target2 := checks.GlobalTarget{Url: "https://sparrow-2.example.com", LastSeen: now}

	target1JSON, _ := json.Marshal(target1)
	target2JSON, _ := json.Marshal(target2)

	contents := fmt.Sprintf(
		"%s\n  %s\n  %s",
		fmt.Sprintf(listBucketContentXML, "sparrow-1.json", len(target1JSON)),
		fmt.Sprintf(listBucketContentXML, "sparrow-2.json", len(target2JSON)),
		fmt.Sprintf(listBucketContentXML, "README.md", 100),
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Query().Get("list-type") == "2":
			w.Header().Set("Content-Type", "application/xml")
			fmt.Fprintf(w, listBucketResultXML, "test-bucket", "", contents)

		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "sparrow-1.json"):
			writeObjectResponse(w, target1JSON, `"etag1"`)

		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "sparrow-2.json"):
			writeObjectResponse(w, target2JSON, `"etag2"`)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	c := newTestClient(t, server, "")
	targets, err := c.FetchFiles(context.Background())
	require.NoError(t, err)
	assert.Len(t, targets, 2)
	assert.Equal(t, target1, targets[0])
	assert.Equal(t, target2, targets[1])
}

func TestClient_PutFile(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && strings.Contains(r.URL.Path, "sparrow-1.json") {
			body, _ := io.ReadAll(r.Body)
			receivedBody = body
			w.Header().Set("ETag", `"etag1"`)
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := newTestClient(t, server, "")

	file := remote.File{
		Name:    "sparrow-1.json",
		Content: checks.GlobalTarget{Url: "https://sparrow-1.example.com", LastSeen: now},
	}

	err := c.PutFile(context.Background(), file)
	require.NoError(t, err)

	expectedJSON, _ := json.Marshal(file.Content)
	assert.Contains(t, string(receivedBody), string(expectedJSON))
}

func TestClient_DeleteFile(t *testing.T) {
	var deleteCalled bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "sparrow-1.json") {
			deleteCalled = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := newTestClient(t, server, "")

	err := c.DeleteFile(context.Background(), remote.File{Name: "sparrow-1.json"})
	require.NoError(t, err)
	assert.True(t, deleteCalled)
}

func TestClient_FetchFiles_WithPrefix(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	target := checks.GlobalTarget{Url: "https://sparrow-1.example.com", LastSeen: now}
	targetJSON, _ := json.Marshal(target)

	contents := fmt.Sprintf(listBucketContentXML, "targets/sparrow-1.json", len(targetJSON))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Query().Get("list-type") == "2":
			assert.Equal(t, "targets/", r.URL.Query().Get("prefix"))
			w.Header().Set("Content-Type", "application/xml")
			fmt.Fprintf(w, listBucketResultXML, "test-bucket", "targets/", contents)

		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "targets/sparrow-1.json"):
			writeObjectResponse(w, targetJSON, `"etag1"`)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	c := newTestClient(t, server, "targets")
	targets, err := c.FetchFiles(context.Background())
	require.NoError(t, err)
	assert.Len(t, targets, 1)
	assert.Equal(t, target, targets[0])
}

func TestClient_FetchFiles_ETagCache(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	target := checks.GlobalTarget{Url: "https://sparrow-1.example.com", LastSeen: now}
	targetJSON, _ := json.Marshal(target)

	contents := fmt.Sprintf(listBucketContentXML, "sparrow-1.json", len(targetJSON))
	fetchCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Query().Get("list-type") == "2":
			w.Header().Set("Content-Type", "application/xml")
			fmt.Fprintf(w, listBucketResultXML, "test-bucket", "", contents)

		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "sparrow-1.json"):
			// On second fetch the client should send If-None-Match
			if inm := r.Header.Get("If-None-Match"); inm != "" {
				fetchCount++
				w.WriteHeader(http.StatusNotModified)
				return
			}
			fetchCount++
			writeObjectResponse(w, targetJSON, `"etag-abc"`)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	c := newTestClient(t, server, "")

	// First fetch — downloads the object
	targets, err := c.FetchFiles(context.Background())
	require.NoError(t, err)
	require.Len(t, targets, 1)
	assert.Equal(t, target, targets[0])
	assert.Equal(t, 1, fetchCount, "first fetch should download the object")

	// Second fetch — should get 304 and return cached
	targets, err = c.FetchFiles(context.Background())
	require.NoError(t, err)
	require.Len(t, targets, 1)
	assert.Equal(t, target, targets[0])
	assert.Equal(t, 2, fetchCount, "second fetch should hit server but get 304")
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr error
	}{
		{
			name:    "missing endpoint",
			config:  Config{Bucket: "b", Auth: AuthConfig{Provider: "static", Static: StaticAuthConfig{AccessKeyID: "a", SecretAccessKey: "s"}}},
			wantErr: ErrMissingEndpoint,
		},
		{
			name:    "missing bucket",
			config:  Config{Endpoint: "e", Auth: AuthConfig{Provider: "static", Static: StaticAuthConfig{AccessKeyID: "a", SecretAccessKey: "s"}}},
			wantErr: ErrMissingBucket,
		},
		{
			name:    "static missing access key",
			config:  Config{Endpoint: "e", Bucket: "b", Auth: AuthConfig{Provider: "static", Static: StaticAuthConfig{SecretAccessKey: "s"}}},
			wantErr: ErrMissingAccessKey,
		},
		{
			name:    "static missing secret key",
			config:  Config{Endpoint: "e", Bucket: "b", Auth: AuthConfig{Provider: "static", Static: StaticAuthConfig{AccessKeyID: "a"}}},
			wantErr: ErrMissingSecretKey,
		},
		{
			name:    "oidc missing role ARN",
			config:  Config{Endpoint: "e", Bucket: "b", Auth: AuthConfig{Provider: "oidc", OIDC: OIDCAuthConfig{TokenPath: "/t"}}},
			wantErr: ErrMissingRoleARN,
		},
		{
			name:    "oidc missing token path",
			config:  Config{Endpoint: "e", Bucket: "b", Auth: AuthConfig{Provider: "oidc", OIDC: OIDCAuthConfig{RoleARN: "arn"}}},
			wantErr: ErrMissingTokenPath,
		},
		{
			name:    "unknown provider",
			config:  Config{Endpoint: "e", Bucket: "b", Auth: AuthConfig{Provider: "magic"}},
			wantErr: ErrUnknownProvider,
		},
		{
			name:   "valid static",
			config: Config{Endpoint: "e", Bucket: "b", Auth: AuthConfig{Provider: "static", Static: StaticAuthConfig{AccessKeyID: "a", SecretAccessKey: "s"}}},
		},
		{
			name:   "valid oidc",
			config: Config{Endpoint: "e", Bucket: "b", Auth: AuthConfig{Provider: "oidc", OIDC: OIDCAuthConfig{RoleARN: "arn", TokenPath: "/t"}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate(context.Background())
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
