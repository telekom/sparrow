// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"errors"
	"iter"

	"github.com/telekom/sparrow/pkg/checks"
	"github.com/telekom/sparrow/pkg/checks/dns"
	"github.com/telekom/sparrow/pkg/checks/health"
	"github.com/telekom/sparrow/pkg/checks/latency"
	"github.com/telekom/sparrow/pkg/checks/traceroute"
)

// Config holds the runtime configuration
// for the various checks
// the sparrow supports
type Config struct {
	Health     *health.Config     `yaml:"health" json:"health"`
	Latency    *latency.Config    `yaml:"latency" json:"latency"`
	Dns        *dns.Config        `yaml:"dns" json:"dns"`
	Traceroute *traceroute.Config `yaml:"traceroute" json:"traceroute"`
}

// Empty returns true if no checks are configured
func (c Config) Empty() bool {
	return c.size() == 0
}

func (c Config) Validate() (err error) {
	for cfg := range c.Iter() {
		if vErr := cfg.Validate(); vErr != nil {
			err = errors.Join(err, vErr)
		}
	}

	return err
}

// Iter returns configured checks as an iterator
func (c Config) Iter() iter.Seq[checks.Runtime] {
	return func(yield func(checks.Runtime) bool) {
		if c.Health != nil {
			if !yield(c.Health) {
				return
			}
		}
		if c.Latency != nil {
			if !yield(c.Latency) {
				return
			}
		}
		if c.Dns != nil {
			if !yield(c.Dns) {
				return
			}
		}
		if c.Traceroute != nil {
			if !yield(c.Traceroute) {
				return
			}
		}
	}
}

// size returns the number of checks configured
func (c Config) size() int {
	size := 0
	if c.HasHealthCheck() {
		size++
	}
	if c.HasLatencyCheck() {
		size++
	}
	if c.HasDNSCheck() {
		size++
	}
	if c.HasTracerouteCheck() {
		size++
	}
	return size
}

// HasHealthCheck returns true if the check has a health check configured
func (c Config) HasHealthCheck() bool {
	return c.Health != nil
}

// HasLatencyCheck returns true if the check has a latency check configured
func (c Config) HasLatencyCheck() bool {
	return c.Latency != nil
}

// HasDNSCheck returns true if the check has a dns check configured
func (c Config) HasDNSCheck() bool {
	return c.Dns != nil
}

// HasTracerouteCheck returns true if the check has a traceroute check configured
func (c Config) HasTracerouteCheck() bool {
	return c.Traceroute != nil
}

// HasCheck returns true if the check has a check with the given name configured
func (c Config) HasCheck(name string) bool {
	switch name {
	case health.CheckName:
		return c.HasHealthCheck()
	case latency.CheckName:
		return c.HasLatencyCheck()
	case dns.CheckName:
		return c.HasDNSCheck()
	case traceroute.CheckName:
		return c.HasTracerouteCheck()
	default:
		return false
	}
}

// For returns the runtime configuration for the check with the given name
func (c Config) For(name string) checks.Runtime {
	switch name {
	case health.CheckName:
		if c.HasHealthCheck() {
			return c.Health
		}
	case latency.CheckName:
		if c.HasLatencyCheck() {
			return c.Latency
		}
	case dns.CheckName:
		if c.HasDNSCheck() {
			return c.Dns
		}
	case traceroute.CheckName:
		if c.HasTracerouteCheck() {
			return c.Traceroute
		}
	}
	return nil
}
