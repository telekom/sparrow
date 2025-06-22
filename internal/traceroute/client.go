// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package traceroute

import (
	"context"
	"fmt"
)

var (
	_ Client = (*genericClient)(nil)
)

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

func NewClient() Client {
	return &genericClient{
		tcp: newTCPClient(),
	}
}

func (c *genericClient) Run(ctx context.Context, targets []Target, opts *Options) (Result, error) {
	for _, target := range targets {
		if err := target.Validate(); err != nil {
			return nil, fmt.Errorf("invalid target %s: %w", target, err)
		}
	}

	return c.tcp.Run(ctx, targets, opts)
}
