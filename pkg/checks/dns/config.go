// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package dns

import (
	"fmt"
	"strings"
	"time"

	"github.com/telekom/sparrow/internal/helper"
	"github.com/telekom/sparrow/pkg/checks"
)

const (
	minInterval = 100 * time.Millisecond
	minTimeout  = 200 * time.Millisecond
)

// Config defines the configuration parameters for a DNS check
type Config struct {
	Targets  []string           `json:"targets" yaml:"targets"`
	Interval time.Duration      `json:"interval" yaml:"interval"`
	Timeout  time.Duration      `json:"timeout" yaml:"timeout"`
	Retry    helper.RetryConfig `json:"retry" yaml:"retry"`
}

// For returns the name of the check
func (c *Config) For() string {
	return CheckName
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	for _, t := range c.Targets {
		if strings.HasPrefix(t, "https://") || strings.HasPrefix(t, "http://") {
			return checks.ErrInvalidConfig{CheckName: c.For(), Field: "targets", Reason: "target URLs must not start with 'https://' or 'http://'"}
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
