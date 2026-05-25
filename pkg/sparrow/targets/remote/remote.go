// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package remote

import (
	"context"

	"github.com/telekom/sparrow/pkg/checks"
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
	// Name is the filename (e.g. "sparrow-1.json")
	Name string
	// Content is the global target data stored in the file
	Content checks.GlobalTarget
}
