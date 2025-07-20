// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package traceroute

import (
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/telekom/sparrow/internal/traceroute"
	"github.com/telekom/sparrow/pkg/checks"
)

// Config is the configuration for the traceroute check
type Config struct {
	// Targets is a list of targets to traceroute to.
	Targets []traceroute.Target `json:"targets" yaml:"targets" mapstructure:"targets"`
	// Interval is the interval at which to run the traceroute check.
	Interval time.Duration `json:"interval" yaml:"interval" mapstructure:"interval"`
	// Options are the options for the traceroute check.
	traceroute.Options `json:",inline" yaml:",inline" mapstructure:",squash"`
}

func (c *Config) For() string {
	return CheckName
}

func (c *Config) Validate() error {
	if c.Interval <= 0 {
		return checks.ErrInvalidConfig{CheckName: CheckName, Field: "traceroute.interval", Reason: "must be greater than 0"}
	}

	if c.Timeout <= 0 {
		return checks.ErrInvalidConfig{CheckName: CheckName, Field: "traceroute.timeout", Reason: "must be greater than 0"}
	}

	for i, t := range c.Targets {
		ip := net.ParseIP(t.Address)
		if ip != nil {
			continue
		}

		_, err := url.Parse(t.Address)
		if err != nil {
			return checks.ErrInvalidConfig{CheckName: CheckName, Field: fmt.Sprintf("traceroute.targets[%d].addr", i), Reason: "invalid url or ip"}
		}
	}
	return nil
}
