// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"context"
	"fmt"

	"github.com/telekom/sparrow/internal/logger"
)

// Config holds the configuration for OpenTelemetry
type Config struct {
	// Enabled is a flag to enable or disable the OpenTelemetry
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`
	// Exporter is the otlp exporter used to export the traces
	Exporter Exporter `yaml:"exporter" mapstructure:"exporter"`
	// Url is the Url of the collector to which the traces are exported
	Url string `yaml:"url" mapstructure:"url"`
	// Token is the token used to authenticate with the collector
	Token string `yaml:"token" mapstructure:"token"`
	// TLS holds the tls configuration
	TLS TLSConfig `yaml:"tls" mapstructure:"tls"`
}

type TLSConfig struct {
	// Enabled is a flag to enable or disable the tls
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`
	// CertPath is the path to the tls certificate file.
	// This is only required if the otel backend uses custom TLS certificates.
	CertPath string `yaml:"certPath" mapstructure:"certPath"`
}

func (c *Config) Validate(ctx context.Context) error {
	log := logger.FromContext(ctx)
	if err := c.Exporter.Validate(); err != nil {
		log.ErrorContext(ctx, "Invalid exporter", "error", err)
		return err
	}

	if c.Exporter.IsExporting() && c.Url == "" {
		log.ErrorContext(ctx, "Url is required for otlp exporter", "exporter", c.Exporter)
		return fmt.Errorf("url is required for otlp exporter %q", c.Exporter)
	}
	return nil
}
