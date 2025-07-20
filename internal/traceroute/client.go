// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package traceroute

import (
	"cmp"
	"context"
	"fmt"
	"sync"
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
	// udp is the [udpClient] that implements the traceroute using UDP.
	udp Client
}

// NewClient creates a new [Client] that can be used to run traceroutes.
func NewClient() Client {
	return &genericClient{
		tcp: newTCPClient(),
		udp: newUDPClient(),
	}
}

// Run executes the traceroute for the given targets with the specified options.
func (c *genericClient) Run(ctx context.Context, targets []Target, opts *Options) (Result, error) {
	opts = cmp.Or(opts, &defaultOptions)
	groups, err := groupTargets(targets)
	if err != nil {
		return nil, fmt.Errorf("failed to group targets: %w", err)
	}

	r := &runner{
		res:  make(chan Result, len(targets)),
		errs: make(chan error, len(targets)),
	}

	if t, ok := groups[ProtocolTCP]; ok {
		r.run(ctx, func(ctx context.Context) (Result, error) {
			return c.tcp.Run(ctx, t, opts)
		})
	}

	if t, ok := groups[ProtocolUDP]; ok {
		r.run(ctx, func(ctx context.Context) (Result, error) {
			return c.udp.Run(ctx, t, opts)
		})
	}

	return r.collect(ctx)
}

// groupTargets validates and groups targets by their protocol.
func groupTargets(targets []Target) (map[Protocol][]Target, error) {
	tg := map[Protocol][]Target{}
	for _, target := range targets {
		if err := target.Validate(); err != nil {
			return nil, fmt.Errorf("invalid target %s: %w", target, err)
		}

		tg[target.Protocol] = append(tg[target.Protocol], target)
	}
	return tg, nil
}

// runner is a helper struct that manages the execution of traceroute tasks concurrently.
// It can run multiple traceroute tasks in parallel and collect their results.
type runner struct {
	// wg is the [sync.WaitGroup] used to wait for all traceroute tasks to complete.
	wg sync.WaitGroup
	// res is a channel to collect results from the traceroute tasks.
	res chan Result
	// errs is a channel to collect errors from the traceroute tasks.
	errs chan error
}

// run starts a new traceroute task in a goroutine.
func (r *runner) run(ctx context.Context, f func(ctx context.Context) (Result, error)) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		res, err := f(ctx)
		if err != nil {
			r.errs <- err
			return
		}
		r.res <- res
	}()
}

// collect waits for all traceroute tasks to complete and collects their results.
func (r *runner) collect(ctx context.Context) (Result, error) {
	go func() {
		r.wg.Wait()
		close(r.res)
		close(r.errs)
	}()

	res := Result{}
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-r.errs:
			if err != nil {
				return nil, err
			}
		case r, ok := <-r.res:
			if !ok {
				return res, nil
			}
			for target, hops := range r {
				res[target] = hops
			}
		}
	}
}
