// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package factory

import (
	"errors"

	"github.com/telekom/sparrow/pkg/checks"
	"github.com/telekom/sparrow/pkg/checks/dns"
	"github.com/telekom/sparrow/pkg/checks/health"
	"github.com/telekom/sparrow/pkg/checks/latency"
	"github.com/telekom/sparrow/pkg/checks/runtime"
	"github.com/telekom/sparrow/pkg/checks/traceroute"
)

// newCheck creates a new check instance from the given name
func newCheck(cfg checks.Runtime) (checks.Check, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}

	if f, ok := registry[cfg.For()]; ok {
		c := f()
		err := c.UpdateConfig(cfg)
		return c, err
	}
	return nil, errors.New("unknown check type")
}

// NewChecksFromConfig creates all checks defined provided config
func NewChecksFromConfig(cfg runtime.Config) (map[string]checks.Check, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	result := make(map[string]checks.Check)
	for c := range cfg.Iter() {
		check, err := newCheck(c)
		if err != nil {
			return nil, err
		}
		result[check.Name()] = check
	}
	return result, nil
}

// registry is a convenience map to create new checks
var registry = map[string]func() checks.Check{
	health.CheckName:     health.NewCheck,
	latency.CheckName:    latency.NewCheck,
	dns.CheckName:        dns.NewCheck,
	traceroute.CheckName: traceroute.NewCheck,
}
