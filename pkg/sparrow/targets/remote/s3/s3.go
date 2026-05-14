// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package s3

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/minio/minio-go/v7"

	"github.com/telekom/sparrow/internal/logger"
	"github.com/telekom/sparrow/pkg/checks"
	"github.com/telekom/sparrow/pkg/sparrow/targets/remote"
)

// defaultRegion is used when no region is configured
const defaultRegion = "eu-central-1"

var _ remote.Interactor = (*client)(nil)

// client implements remote.Interactor for S3-compatible object storage
type client struct {
	config Config
	s3     *minio.Client
	// etags stores the last known ETag for each object key.
	// Used for conditional GETs to reduce bandwidth.
	etags map[string]string
	// cache stores the last successfully fetched target for each key.
	// Returned on 304 Not Modified responses.
	cache map[string]checks.GlobalTarget
}

// New creates a new S3-compatible interactor
func New(cfg *Config) (remote.Interactor, error) {
	if cfg.Region == "" {
		cfg.Region = defaultRegion
	}

	useSSL := true
	if cfg.UseSSL != nil {
		useSSL = *cfg.UseSSL
	}

	creds, err := cfg.Auth.newCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to build S3 credentials: %w", err)
	}

	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  creds,
		Secure: useSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	return &client{
		config: *cfg,
		s3:     mc,
		etags:  make(map[string]string),
		cache:  make(map[string]checks.GlobalTarget),
	}, nil
}

// objectKey returns the full S3 key for a given file name
func (c *client) objectKey(name string) string {
	if c.config.Prefix == "" {
		return name
	}
	return path.Join(c.config.Prefix, name)
}

// FetchFiles lists and fetches all .json target files from the configured S3 bucket
func (c *client) FetchFiles(ctx context.Context) ([]checks.GlobalTarget, error) {
	log := logger.FromContext(ctx)
	log.DebugContext(ctx, "Fetching target files from S3", "bucket", c.config.Bucket, "prefix", c.config.Prefix)

	var targets []checks.GlobalTarget

	prefix := c.config.Prefix
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}

	for obj := range c.s3.ListObjects(ctx, c.config.Bucket, opts) {
		if obj.Err != nil {
			log.ErrorContext(ctx, "Error listing S3 objects", "error", obj.Err)
			return nil, fmt.Errorf("failed to list S3 objects: %w", obj.Err)
		}

		// Only process .json files
		if !strings.HasSuffix(obj.Key, ".json") {
			continue
		}

		target, err := c.fetchObject(ctx, obj.Key)
		if err != nil {
			log.ErrorContext(ctx, "Failed to fetch target file", "key", obj.Key, "error", err)
			return nil, err
		}
		targets = append(targets, target)
	}

	log.InfoContext(ctx, "Successfully fetched all target files from S3", "count", len(targets))
	return targets, nil
}

// fetchObject retrieves and unmarshals a single target file from S3.
// Uses ETag-based conditional GETs to avoid re-downloading unchanged objects.
func (c *client) fetchObject(ctx context.Context, key string) (checks.GlobalTarget, error) {
	var target checks.GlobalTarget

	opts := minio.GetObjectOptions{}
	if etag, ok := c.etags[key]; ok {
		if err := opts.SetMatchETagExcept(etag); err != nil {
			return target, fmt.Errorf("failed to set If-None-Match for %q: %w", key, err)
		}
	}

	obj, err := c.s3.GetObject(ctx, c.config.Bucket, key, opts)
	if err != nil {
		return target, fmt.Errorf("failed to get S3 object %q: %w", key, err)
	}
	defer func() { _ = obj.Close() }()

	data, err := io.ReadAll(obj)
	if err != nil {
		// A 304 Not Modified is surfaced as an ErrorResponse when reading
		if errResp, ok := errors.AsType[minio.ErrorResponse](err); ok && errResp.StatusCode == http.StatusNotModified {
			if cached, ok := c.cache[key]; ok {
				logger.FromContext(ctx).DebugContext(ctx, "S3 object not modified, using cache", "key", key)
				return cached, nil
			}
		}
		return target, fmt.Errorf("failed to read S3 object %q: %w", key, err)
	}

	if err = json.Unmarshal(data, &target); err != nil {
		return target, fmt.Errorf("failed to unmarshal S3 object %q: %w", key, err)
	}

	// Update ETag cache from the object metadata
	info, err := obj.Stat()
	if err == nil && info.ETag != "" {
		c.etags[key] = info.ETag
		c.cache[key] = target
	} else {
		logger.FromContext(ctx).WarnContext(ctx, "Failed to stat S3 object for ETag caching", "key", key, "error", err)
	}

	return target, nil
}

// PostFile creates a new target file in S3.
// S3 PutObject is idempotent so this is equivalent to PutFile.
func (c *client) PostFile(ctx context.Context, file remote.File) error {
	return c.putObject(ctx, file)
}

// PutFile updates an existing target file in S3.
// S3 PutObject is idempotent so this is equivalent to PostFile.
func (c *client) PutFile(ctx context.Context, file remote.File) error {
	return c.putObject(ctx, file)
}

// putObject uploads a target file to S3
func (c *client) putObject(ctx context.Context, file remote.File) error {
	log := logger.FromContext(ctx)
	key := c.objectKey(file.Name)

	data, err := json.Marshal(file.Content)
	if err != nil {
		return fmt.Errorf("failed to marshal target content: %w", err)
	}

	log.DebugContext(ctx, "Uploading target file to S3", "key", key)
	_, err = c.s3.PutObject(ctx, c.config.Bucket, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: "application/json",
	})
	if err != nil {
		log.ErrorContext(ctx, "Failed to upload target file to S3", "key", key, "error", err)
		return fmt.Errorf("failed to put S3 object %q: %w", key, err)
	}

	log.DebugContext(ctx, "Successfully uploaded target file to S3", "key", key)
	return nil
}

// DeleteFile removes a target file from S3
func (c *client) DeleteFile(ctx context.Context, file remote.File) error {
	log := logger.FromContext(ctx)
	key := c.objectKey(file.Name)

	if file.Name == "" {
		return fmt.Errorf("filename is empty")
	}

	log.DebugContext(ctx, "Deleting target file from S3", "key", key)
	err := c.s3.RemoveObject(ctx, c.config.Bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		log.ErrorContext(ctx, "Failed to delete target file from S3", "key", key, "error", err)
		return fmt.Errorf("failed to delete S3 object %q: %w", key, err)
	}

	log.DebugContext(ctx, "Successfully deleted target file from S3", "key", key)
	delete(c.etags, key)
	delete(c.cache, key)
	return nil
}
