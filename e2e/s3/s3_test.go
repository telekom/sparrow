// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package s3_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	minioclient "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/minio"

	"github.com/telekom/sparrow/pkg/checks"
	"github.com/telekom/sparrow/pkg/sparrow/targets/remote"
	"github.com/telekom/sparrow/pkg/sparrow/targets/remote/s3"
)

const (
	testBucket   = "sparrow-targets"
	testUser     = "minioadmin"
	testPassword = "minioadmin"
)

var (
	testEndpoint string
	testCtx      = context.Background()
)

// TestMain starts a MinIO container, creates the test bucket,
// and tears everything down after all tests complete.
func TestMain(m *testing.M) {
	container, err := minio.Run(
		testCtx,
		"minio/minio:RELEASE.2024-01-16T16-07-38Z",
		minio.WithUsername(testUser),
		minio.WithPassword(testPassword),
	)
	if err != nil {
		log.Fatalf("failed to start MinIO container: %s", err)
	}

	connStr, err := container.ConnectionString(testCtx)
	if err != nil {
		log.Fatalf("failed to get connection string: %s", err)
	}
	testEndpoint = connStr

	// Create the test bucket
	mc, err := minioclient.New(testEndpoint, &minioclient.Options{
		Creds:  credentials.NewStaticV4(testUser, testPassword, ""),
		Secure: false,
	})
	if err != nil {
		log.Fatalf("failed to create MinIO client: %s", err)
	}
	if err := mc.MakeBucket(testCtx, testBucket, minioclient.MakeBucketOptions{}); err != nil {
		log.Fatalf("failed to create bucket: %s", err)
	}

	code := m.Run()

	if err := container.Terminate(testCtx); err != nil {
		log.Printf("failed to terminate MinIO container: %s", err)
	}

	os.Exit(code)
}

// newInteractor creates a new S3 interactor pointing at the test MinIO.
func newInteractor(t *testing.T, prefix string) remote.Interactor {
	t.Helper()
	useSSL := false
	i, err := s3.New(&s3.Config{
		Endpoint: testEndpoint,
		Bucket:   testBucket,
		Region:   "us-east-1",
		Prefix:   prefix,
		UseSSL:   &useSSL,
		Auth: s3.AuthConfig{
			Provider: "static",
			Static: s3.StaticAuthConfig{
				AccessKeyID:     testUser,
				SecretAccessKey: testPassword,
			},
		},
	})
	require.NoError(t, err)
	return i
}

// cleanBucket removes all objects from the bucket.
func cleanBucket(t *testing.T) {
	t.Helper()
	mc, err := minioclient.New(testEndpoint, &minioclient.Options{
		Creds:  credentials.NewStaticV4(testUser, testPassword, ""),
		Secure: false,
	})
	require.NoError(t, err)

	for obj := range mc.ListObjects(testCtx, testBucket, minioclient.ListObjectsOptions{Recursive: true}) {
		require.NoError(t, obj.Err)
		err := mc.RemoveObject(testCtx, testBucket, obj.Key, minioclient.RemoveObjectOptions{})
		require.NoError(t, err)
	}
}

// TestS3_FullLifecycle exercises the complete interactor lifecycle:
// post -> fetch (present) -> put (update) -> fetch (updated) -> delete -> fetch (gone).
func TestS3_FullLifecycle(t *testing.T) {
	cleanBucket(t)
	i := newInteractor(t, "")

	now := time.Now().UTC().Truncate(time.Second)
	target := checks.GlobalTarget{
		Url:      "https://sparrow-1.example.com",
		LastSeen: now,
	}

	// Post
	err := i.PostFile(testCtx, remote.File{
		Name:    "sparrow-1.json",
		Content: target,
	})
	require.NoError(t, err)

	// Fetch — should find the target
	targets, err := i.FetchFiles(testCtx)
	require.NoError(t, err)
	require.Len(t, targets, 1)
	assert.Equal(t, target.Url, targets[0].Url)
	assert.Equal(t, target.LastSeen, targets[0].LastSeen)

	// Put — update LastSeen
	updatedTime := now.Add(5 * time.Minute)
	target.LastSeen = updatedTime
	err = i.PutFile(testCtx, remote.File{
		Name:    "sparrow-1.json",
		Content: target,
	})
	require.NoError(t, err)

	// Fetch — should see updated LastSeen
	targets, err = i.FetchFiles(testCtx)
	require.NoError(t, err)
	require.Len(t, targets, 1)
	assert.Equal(t, updatedTime, targets[0].LastSeen)

	// Delete
	err = i.DeleteFile(testCtx, remote.File{Name: "sparrow-1.json"})
	require.NoError(t, err)

	// Fetch — should be empty
	targets, err = i.FetchFiles(testCtx)
	require.NoError(t, err)
	assert.Empty(t, targets)
}

// TestS3_ETagCaching verifies that a second FetchFiles returns
// cached data when the objects have not changed (304 Not Modified).
func TestS3_ETagCaching(t *testing.T) {
	cleanBucket(t)
	i := newInteractor(t, "")

	target := checks.GlobalTarget{
		Url:      "https://sparrow-cached.example.com",
		LastSeen: time.Now().UTC().Truncate(time.Second),
	}

	err := i.PostFile(testCtx, remote.File{
		Name:    "sparrow-cached.json",
		Content: target,
	})
	require.NoError(t, err)

	// First fetch — populates ETag cache
	targets1, err := i.FetchFiles(testCtx)
	require.NoError(t, err)
	require.Len(t, targets1, 1)

	// Second fetch — should use cache (304)
	targets2, err := i.FetchFiles(testCtx)
	require.NoError(t, err)
	require.Len(t, targets2, 1)
	assert.Equal(t, targets1[0].Url, targets2[0].Url)
	assert.Equal(t, targets1[0].LastSeen, targets2[0].LastSeen)
}

// TestS3_PrefixFiltering verifies that only objects under the
// configured prefix are returned, and non-.json files are excluded.
func TestS3_PrefixFiltering(t *testing.T) {
	cleanBucket(t)

	// Create interactor with prefix
	prefixed := newInteractor(t, "env/prod")
	unprefixed := newInteractor(t, "")

	target := checks.GlobalTarget{
		Url:      "https://sparrow-prefixed.example.com",
		LastSeen: time.Now().UTC().Truncate(time.Second),
	}

	// Post via prefixed interactor
	err := prefixed.PostFile(testCtx, remote.File{
		Name:    "sparrow-a.json",
		Content: target,
	})
	require.NoError(t, err)

	// Post via unprefixed interactor (different key space)
	err = unprefixed.PostFile(testCtx, remote.File{
		Name:    "sparrow-b.json",
		Content: target,
	})
	require.NoError(t, err)

	// Also create a non-json file under the prefix via raw client
	mc, err := minioclient.New(testEndpoint, &minioclient.Options{
		Creds:  credentials.NewStaticV4(testUser, testPassword, ""),
		Secure: false,
	})
	require.NoError(t, err)

	_, err = mc.PutObject(testCtx, testBucket, "env/prod/README.md",
		nil, 0, minioclient.PutObjectOptions{})
	require.NoError(t, err)

	// Prefixed interactor should only see sparrow-a.json
	targets, err := prefixed.FetchFiles(testCtx)
	require.NoError(t, err)
	require.Len(t, targets, 1)
	assert.Equal(t, "https://sparrow-prefixed.example.com", targets[0].Url)

	// Unprefixed interactor sees sparrow-b.json and
	// env/prod/sparrow-a.json (both are .json in the bucket)
	allTargets, err := unprefixed.FetchFiles(testCtx)
	require.NoError(t, err)
	assert.Len(t, allTargets, 2)
}

// TestS3_ConcurrentRegistration verifies that multiple goroutines
// can register simultaneously without data loss or errors.
func TestS3_ConcurrentRegistration(t *testing.T) {
	cleanBucket(t)
	i := newInteractor(t, "concurrent")

	const numWorkers = 10
	now := time.Now().UTC().Truncate(time.Second)

	var wg sync.WaitGroup
	errs := make([]error, numWorkers)

	for n := range numWorkers {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			errs[id] = i.PostFile(testCtx, remote.File{
				Name: fmt.Sprintf("sparrow-%d.json", id),
				Content: checks.GlobalTarget{
					Url:      fmt.Sprintf("https://sparrow-%d.example.com", id),
					LastSeen: now,
				},
			})
		}(n)
	}

	wg.Wait()

	for idx, err := range errs {
		assert.NoError(t, err, "worker %d failed", idx)
	}

	// A separate interactor fetches (to avoid sharing cache state)
	fetcher := newInteractor(t, "concurrent")
	targets, err := fetcher.FetchFiles(testCtx)
	require.NoError(t, err)
	assert.Len(t, targets, numWorkers)
}

// TestS3_UnhealthyTargetFiltering verifies that targets with old
// LastSeen timestamps can be identified for filtering.
// Note: the interactor itself does not filter — that is the manager's
// job. This test confirms the data round-trips correctly for the
// manager to make the filtering decision.
func TestS3_UnhealthyTargetFiltering(t *testing.T) {
	cleanBucket(t)
	i := newInteractor(t, "")

	now := time.Now().UTC().Truncate(time.Second)
	healthy := checks.GlobalTarget{
		Url:      "https://healthy.example.com",
		LastSeen: now,
	}
	unhealthy := checks.GlobalTarget{
		Url:      "https://unhealthy.example.com",
		LastSeen: now.Add(-24 * time.Hour),
	}

	err := i.PostFile(testCtx, remote.File{
		Name:    "healthy.json",
		Content: healthy,
	})
	require.NoError(t, err)

	err = i.PostFile(testCtx, remote.File{
		Name:    "unhealthy.json",
		Content: unhealthy,
	})
	require.NoError(t, err)

	targets, err := i.FetchFiles(testCtx)
	require.NoError(t, err)
	require.Len(t, targets, 2)

	// Simulate manager filtering with 1h threshold
	threshold := 1 * time.Hour
	var healthyTargets []checks.GlobalTarget
	for _, tgt := range targets {
		if time.Since(tgt.LastSeen) < threshold {
			healthyTargets = append(healthyTargets, tgt)
		}
	}

	assert.Len(t, healthyTargets, 1)
	assert.Equal(t, "https://healthy.example.com", healthyTargets[0].Url)
}
