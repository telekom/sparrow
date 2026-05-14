// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package targets

import (
	"context"
	"time"

	"github.com/telekom/sparrow/internal/logger"
	"github.com/telekom/sparrow/pkg/checks"
	"github.com/telekom/sparrow/pkg/sparrow/targets/interactor"
)

const (
	schemeHTTP  = "http"
	schemeHTTPS = "https"
)

// TargetManager handles the management of globalTargets for
// a Sparrow instance
type TargetManager interface {
	// Reconcile fetches the global targets from the configured
	// endpoint and updates the local state
	Reconcile(ctx context.Context) error
	// GetTargets returns the current global targets
	GetTargets() []checks.GlobalTarget
	// Shutdown shuts down the target manager
	// and unregisters the instance as a global target
	Shutdown(ctx context.Context) error
}

// General is the general configuration of the target manager
type General struct {
	// The interval for the target reconciliation process
	CheckInterval time.Duration `yaml:"checkInterval" mapstructure:"checkInterval"`
	// How often the instance should register itself as a global target.
	// A duration of 0 means no registration.
	RegistrationInterval time.Duration `yaml:"registrationInterval" mapstructure:"registrationInterval"`
	// How often the instance should update its registration as a global target.
	// A duration of 0 means no update.
	UpdateInterval time.Duration `yaml:"updateInterval" mapstructure:"updateInterval"`
	// The amount of time a target can be unhealthy
	// before it is removed from the global target list.
	// A duration of 0 means no removal.
	UnhealthyThreshold time.Duration `yaml:"unhealthyThreshold" mapstructure:"unhealthyThreshold"`
	// Scheme is the scheme used for the remote target manager
	// Can either be http or https
	Scheme string `yaml:"scheme" mapstructure:"scheme"`
	// Jitter is the jitter factor applied to polling intervals.
	// A value between 0.0 (no jitter) and 1.0 (full jitter).
	// The actual interval will be in [interval*(1-jitter), interval].
	Jitter float64 `yaml:"jitter" mapstructure:"jitter"`
}

// TargetManagerConfig is the configuration for the target manager
type TargetManagerConfig struct {
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`
	// Type defines which target manager to use
	Type interactor.Type `yaml:"type" mapstructure:"type"`
	// General is the general configuration of the target manager
	General `yaml:",inline" mapstructure:",squash"`
	// Config is the configuration for the Config target manager
	interactor.Config `yaml:",inline" mapstructure:",squash"`
}

func (c *TargetManagerConfig) Validate(ctx context.Context) error {
	log := logger.FromContext(ctx)
	if c.CheckInterval <= 0 {
		log.Error("The check interval should be above 0", "interval", c.CheckInterval)
		return ErrInvalidCheckInterval
	}
	if c.RegistrationInterval < 0 {
		log.Error("The registration interval should be equal or above 0", "interval", c.RegistrationInterval)
		return ErrInvalidRegistrationInterval
	}
	if c.UnhealthyThreshold < 0 {
		log.Error("The unhealthy threshold should be equal or above 0", "threshold", c.UnhealthyThreshold)
		return ErrInvalidUnhealthyThreshold
	}
	if c.UpdateInterval < 0 {
		log.Error("The update interval should be equal or above 0", "interval", c.UpdateInterval)
		return ErrInvalidUpdateInterval
	}

	if c.Scheme != schemeHTTP && c.Scheme != schemeHTTPS {
		log.Error("The scheme should be either of: 'http', 'https'", "scheme", c.Scheme)
		return ErrInvalidScheme
	}

	if c.Jitter < 0 || c.Jitter > 1 {
		log.Error("The jitter factor should be between 0.0 and 1.0", "jitter", c.Jitter)
		return ErrInvalidJitter
	}

	switch c.Type {
	case interactor.Gitlab:
		return nil
	case interactor.S3:
		return c.S3.Validate(ctx)
	default:
		log.Error("Invalid interactor type", "type", c.Type)
		return ErrInvalidInteractorType
	}
}
