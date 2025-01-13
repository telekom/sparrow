// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package remote

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"

	"github.com/caas-team/sparrow/pkg/checks"
)

// Interactor handles the interaction with the remote state backend
// It is responsible for CRUD operations on the global targets repository
type Interactor interface {
	// FetchFiles fetches the files from the global targets repository
	FetchFiles(ctx context.Context) ([]checks.GlobalTarget, error)
	// PutFile updates the file in the repository
	PutFile(ctx context.Context, file File) error
	// PostFile creates the file in the repository
	PostFile(ctx context.Context, file File) error
	// DeleteFile deletes the file from the repository
	DeleteFile(ctx context.Context, file File) error
}

// File represents a file in the global targets repository
type File struct {
	AuthorEmail   string
	AuthorName    string
	CommitMessage string
	Content       checks.GlobalTarget
	Name          string
}

// Serialize serializes the file to a byte slice. The branch is used to determine the branch to commit to
// The serialized file is base64 encoded.
func (f *File) Serialize(branch string) (b []byte, err error) {
	content, err := json.Marshal(f.Content)
	if err != nil {
		return nil, err
	}

	// base64 encode the content
	enc := base64.NewEncoder(base64.StdEncoding, bytes.NewBuffer(content))
	_, err = enc.Write(content)
	defer func() {
		err = errors.Join(err, enc.Close())
	}()

	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]string{
		"branch":         branch,
		"author_email":   f.AuthorEmail,
		"author_name":    f.AuthorName,
		"content":        string(content),
		"commit_message": f.CommitMessage,
	})
}

// SetFileName sets the filename of the File
func (f *File) SetFileName(name string) {
	f.Name = name
}
