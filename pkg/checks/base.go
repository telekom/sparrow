// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"context"
	"sync"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/telekom/sparrow/internal/helper"
)

// DefaultRetry provides a default configuration for the retry mechanism
var DefaultRetry = helper.RetryConfig{
	Count: 3,
	Delay: time.Second,
}

// Check implementations are expected to perform specific monitoring tasks and report results.
//
//go:generate moq -out base_moq.go . Check
type Check interface {
	// Run is called once, to start running the check. The check should
	// run until the context is canceled and handle problems itself.
	// Returning a non-nil error will cause the shutdown of the check.
	Run(ctx context.Context, cResult chan ResultDTO) error
	// Shutdown is called once when the check is unregistered or sparrow shuts down
	Shutdown()
	// UpdateConfig is called once when the check is registered
	// This is also called while the check is running, if the remote config is updated
	// This should return an error if the config is invalid
	UpdateConfig(config Runtime) error
	// GetConfig returns the current configuration of the check
	GetConfig() Runtime
	// Name returns the name of the check
	Name() string
	// Schema returns an openapi3.SchemaRef of the result type returned by the check
	Schema() (*openapi3.SchemaRef, error)
	// GetMetricCollectors allows the check to provide prometheus metric collectors
	GetMetricCollectors() []prometheus.Collector
	// RemoveLabelledMetrics allows the check to remove the prometheus metrics
	// of the check whose `target` label matches the passed value
	RemoveLabelledMetrics(target string) error
}

// CheckBase is a struct providing common fields used by implementations of the Check interface.
// It serves as a foundational structure that should be embedded in specific check implementations.
type CheckBase struct {
	// Mutex for thread-safe access to shared resources within the check implementation
	Mu sync.Mutex
	// Signal channel used to notify about shutdown of a check
	DoneChan chan struct{}
}

// Runtime is the interface that all check configurations must implement
type Runtime interface {
	// For returns the name of the check being configured
	For() string
	// Validate checks if the configuration is valid
	Validate() error
}

// Result encapsulates the outcome of a check run.
type Result struct {
	// Data contains performance metrics about the check run
	Data any `json:"data"`
	// Timestamp is the UTC time the check was run
	Timestamp time.Time `json:"timestamp"`
}

// ResultDTO is a data transfer object used to associate a check's name with its result.
type ResultDTO struct {
	Name   string
	Result *Result
}

// GlobalTarget includes the basic information regarding
// other Sparrow instances, which this Sparrow can communicate with.
type GlobalTarget struct {
	Url      string    `json:"url"`
	LastSeen time.Time `json:"lastSeen"`
}
