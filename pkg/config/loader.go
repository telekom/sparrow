// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"

	"github.com/caas-team/sparrow/pkg/checks/runtime"
)

//go:generate moq -out loader_moq.go . Loader
type Loader interface {
	// Run starts the loader routine.
	// The loader should be able
	// to handle all errors by itself and retry if necessary.
	// If the context is canceled,
	// the Run method returns an error.
	Run(context.Context) error
	// Shutdown stops the loader routine.
	Shutdown(context.Context)
}

// NewLoader Get a new typed runtime configuration loader
func NewLoader(cfg *Config, cRuntime chan<- runtime.Config) Loader {
	switch cfg.Loader.Type {
	case "http":
		return NewHttpLoader(cfg, cRuntime)
	default:
		return NewFileLoader(cfg, cRuntime)
	}
}
