// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package latency

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"time"

	"github.com/telekom/sparrow/internal/helper"
	"github.com/telekom/sparrow/internal/logger"
	"github.com/telekom/sparrow/pkg/checks"
)

const (
	minInterval = 100 * time.Millisecond
	minTimeout  = 1 * time.Second
)

// Config defines the configuration parameters for a latency check
type Config struct {
	Targets  []string           `json:"targets,omitempty" yaml:"targets,omitempty"`
	Retry    helper.RetryConfig `json:"retry" yaml:"retry"`
	Interval time.Duration      `json:"interval" yaml:"interval"`
	Timeout  time.Duration      `json:"timeout" yaml:"timeout"`
}

// For returns the name of the check
func (c *Config) For() string {
	return CheckName
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	for _, t := range c.Targets {
		u, err := url.Parse(t)
		if err != nil {
			return checks.ErrInvalidConfig{CheckName: c.For(), Field: "targets", Reason: "invalid target URL"}
		}

		if u.Scheme != "https" && u.Scheme != "http" {
			return checks.ErrInvalidConfig{CheckName: c.For(), Field: "targets", Reason: "target URLs must start with 'https://' or 'http://'"}
		}
	}

	if c.Interval < minInterval {
		return checks.ErrInvalidConfig{CheckName: c.For(), Field: "interval", Reason: fmt.Sprintf("interval must be at least %v", minInterval)}
	}

	if c.Timeout < minTimeout {
		return checks.ErrInvalidConfig{CheckName: c.For(), Field: "timeout", Reason: fmt.Sprintf("timeout must be at least %v", minTimeout)}
	}

	return nil
}

// Enrich adds the global targets to the configuration
func (c *Config) Enrich(ctx context.Context, targets []checks.GlobalTarget) {
	log := logger.FromContext(ctx)
	for _, t := range targets {
		target := t.URL.String()
		if !slices.Contains(c.Targets, target) {
			log.DebugContext(ctx, "Adding target to latency check", "target", target)
			c.Targets = append(c.Targets, target)
		}
	}
}
