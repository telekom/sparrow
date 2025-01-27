// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package config

import "errors"

var (
	// ErrInvalidSparrowName is returned when the sparrow name is invalid
	ErrInvalidSparrowName = errors.New("invalid sparrow name")
	// ErrInvalidLoaderInterval is returned when the loader interval is invalid
	ErrInvalidLoaderInterval = errors.New("invalid loader interval")
	// ErrInvalidLoaderHttpURL is returned when the loader http url is invalid
	ErrInvalidLoaderHttpURL = errors.New("invalid loader http url")
	// ErrInvalidLoaderHttpRetryCount is returned when the loader http retry count is invalid
	ErrInvalidLoaderHttpRetryCount = errors.New("invalid loader http retry count")
	// ErrInvalidLoaderFilePath is returned when the loader file path is invalid
	ErrInvalidLoaderFilePath = errors.New("invalid loader file path")
)
