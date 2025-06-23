// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package traceroute

import (
	"cmp"
	"context"
	"fmt"
	"time"

	"github.com/telekom/sparrow/internal/helper"
)

var _ Client = (*genericClient)(nil)

// defaultOptions provides a set of default options for the traceroute.
var defaultOptions = Options{
	MaxTTL:  30,
	Timeout: 60 * time.Second,
	Retry: helper.RetryConfig{
		Count: 3,
		Delay: 1 * time.Second,
	},
}

// Client is able to run a traceroute to one or more targets.
//
//go:generate go tool moq -out client_moq.go . Client
type Client interface {
	// Run executes the traceroute for the given targets with the specified options.
	// Returns a Result containing the hops for each target, or an error if the traceroute fails.
	Run(ctx context.Context, targets []Target, opts *Options) (Result, error)
}

type genericClient struct {
	// tcp is the [tcpClient] that implements the traceroute using TCP.
	tcp Client
}

// NewClient creates a new [Client] that can be used to run traceroutes.
func NewClient() Client {
	return &genericClient{
		tcp: newTCPClient(),
	}
}

// Run executes the traceroute for the given targets with the specified options.
func (c *genericClient) Run(ctx context.Context, targets []Target, opts *Options) (Result, error) {
	for _, target := range targets {
		if err := target.Validate(); err != nil {
			return nil, fmt.Errorf("invalid target %s: %w", target, err)
		}
	}

	return c.tcp.Run(ctx, targets, cmp.Or(opts, &defaultOptions))
}
