// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package traceroute

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"slices"
	"time"

	"github.com/telekom/sparrow/internal/helper"
	"github.com/telekom/sparrow/internal/logger"
	"github.com/telekom/sparrow/pkg/checks"
)

// Config is the configuration for the traceroute check
type Config struct {
	// Targets is a list of targets to traceroute to
	Targets []Target `json:"targets" yaml:"targets" mapstructure:"targets"`
	// Retry defines if and how to retry a target
	Retry helper.RetryConfig `json:"retry" yaml:"retry" mapstructure:"retry"`
	// MaxHops is the maximum number of hops to try before giving up
	MaxHops int `json:"maxHops" yaml:"maxHops" mapstructure:"maxHops"`
	// Interval is the time to wait between check iterations
	Interval time.Duration `json:"interval" yaml:"interval" mapstructure:"interval"`
	// Timeout is the maximum time to wait for a response from a hop
	Timeout time.Duration `json:"timeout" yaml:"timeout" mapstructure:"timeout"`
}

func (c *Config) For() string {
	return CheckName
}

func (c *Config) Validate() error {
	if c.Timeout <= 0 {
		return checks.ErrInvalidConfig{CheckName: CheckName, Field: "traceroute.timeout", Reason: "must be greater than 0"}
	}
	if c.Interval <= 0 {
		return checks.ErrInvalidConfig{CheckName: CheckName, Field: "traceroute.interval", Reason: "must be greater than 0"}
	}

	for i, t := range c.Targets {
		ip := net.ParseIP(t.Addr)
		if ip != nil {
			continue
		}

		_, err := url.Parse(t.Addr)
		if err != nil {
			return checks.ErrInvalidConfig{CheckName: CheckName, Field: fmt.Sprintf("traceroute.targets[%d].addr", i), Reason: "invalid url or ip"}
		}
	}
	return nil
}

// Enrich adds the global targets to the configuration
func (c *Config) Enrich(ctx context.Context, targets []checks.GlobalTarget) {
	log := logger.FromContext(ctx)
	for _, t := range targets {
		u, err := t.URL()
		if err != nil {
			log.ErrorContext(ctx, "Failed to get URL from target", "target", t, "error", err)
			continue
		}

		// Error handling is not necessary here, as the URL has been validated before.
		port, _ := t.Port()
		if !slices.ContainsFunc(c.Targets, func(t Target) bool {
			return t.Addr == u.Hostname() && t.Port == port
		}) {
			c.Targets = append(c.Targets, Target{
				Addr: u.Hostname(),
				Port: port,
			})
		}
	}
}
