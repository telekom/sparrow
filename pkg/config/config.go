// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"time"

	"github.com/telekom/sparrow/pkg/sparrow/metrics"
	"github.com/telekom/sparrow/pkg/sparrow/targets"

	"github.com/telekom/sparrow/internal/helper"
	"github.com/telekom/sparrow/pkg/api"
)

type Config struct {
	// SparrowName is the DNS name of the sparrow
	SparrowName string `yaml:"name" mapstructure:"name"`
	// Loader is the configuration for the loader
	Loader LoaderConfig `yaml:"loader" mapstructure:"loader"`
	// Api is the configuration for the api server
	Api api.Config `yaml:"api" mapstructure:"api"`
	// TargetManager is the configuration for the target manager
	TargetManager targets.TargetManagerConfig `yaml:"targetManager" mapstructure:"targetManager"`
	// Telemetry is the configuration for the telemetry
	Telemetry metrics.Config `yaml:"telemetry" mapstructure:"telemetry"`
}

// LoaderConfig is the configuration for loader
type LoaderConfig struct {
	Type     string           `yaml:"type" mapstructure:"type"`
	Interval time.Duration    `yaml:"interval" mapstructure:"interval"`
	Http     HttpLoaderConfig `yaml:"http" mapstructure:"http"`
	File     FileLoaderConfig `yaml:"file" mapstructure:"file"`
}

// HttpLoaderConfig is the configuration for the http loader
type HttpLoaderConfig struct {
	Url      string             `yaml:"url" mapstructure:"url"`
	Token    string             `yaml:"token" mapstructure:"token"`
	Timeout  time.Duration      `yaml:"timeout" mapstructure:"timeout"`
	RetryCfg helper.RetryConfig `yaml:"retry" mapstructure:"retry"`
}

// FileLoaderConfig is the configuration for the file loader
type FileLoaderConfig struct {
	Path string `yaml:"path" mapstructure:"path"`
}

// HasTargetManager returns true if the config has a target manager
func (c *Config) HasTargetManager() bool {
	return c.TargetManager.Enabled
}

// HasTelemetry returns true if the config has telemetry enabled
func (c *Config) HasTelemetry() bool {
	return c.Telemetry.Enabled
}
