// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/telekom/sparrow/internal/helper"
	"github.com/telekom/sparrow/internal/logger"
	"github.com/telekom/sparrow/pkg/checks/runtime"
	"gopkg.in/yaml.v3"
)

type HttpLoader struct {
	cfg      LoaderConfig
	cRuntime chan<- runtime.Config
	done     chan struct{}
	client   *http.Client
}

func NewHttpLoader(cfg *Config, cRuntime chan<- runtime.Config) *HttpLoader {
	return &HttpLoader{
		cfg:      cfg.Loader,
		cRuntime: cRuntime,
		done:     make(chan struct{}, 1),
		client: &http.Client{
			Timeout: cfg.Loader.Http.Timeout,
		},
	}
}

// Run gets the runtime configuration from the remote file of the configured http endpoint.
// The config will be loaded periodically defined by the loader interval configuration.
// If the interval is 0, the configuration is only fetched once and the loader is disabled.
func (h *HttpLoader) Run(ctx context.Context) error {
	ctx, cancel := logger.NewContextWithLogger(ctx)
	defer cancel()
	log := logger.FromContext(ctx)

	var cfg runtime.Config
	getConfigRetry := helper.Retry(func(ctx context.Context) (err error) {
		cfg, err = h.getRuntimeConfig(ctx)
		return err
	}, h.cfg.Http.RetryCfg)

	// Get the runtime configuration once on startup
	err := getConfigRetry(ctx)
	if err != nil {
		log.Warn("Could not get remote runtime configuration", "error", err)
		err = fmt.Errorf("could not get remote runtime configuration: %w", err)
	}
	h.cRuntime <- cfg

	if h.cfg.Interval == 0 {
		log.Info("HTTP Loader disabled")
		return err
	}

	tick := time.NewTicker(h.cfg.Interval)
	defer tick.Stop()

	for {
		select {
		case <-h.done:
			log.Info("HTTP Loader terminated")
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			if err := getConfigRetry(ctx); err != nil {
				log.Warn("Could not get remote runtime configuration", "error", err)
				tick.Reset(h.cfg.Interval)
				continue
			}

			log.Info("Successfully got remote runtime configuration")
			h.cRuntime <- cfg
			tick.Reset(h.cfg.Interval)
		}
	}
}

// GetRuntimeConfig gets the remote runtime configuration
func (hl *HttpLoader) getRuntimeConfig(ctx context.Context) (cfg runtime.Config, err error) {
	log := logger.FromContext(ctx).With("url", hl.cfg.Http.Url)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, hl.cfg.Http.Url, http.NoBody)
	if err != nil {
		log.Error("Could not create http GET request", "error", err.Error())
		return cfg, err
	}
	if hl.cfg.Http.Token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", hl.cfg.Http.Token))
	}

	res, err := hl.client.Do(req) //nolint:bodyclose
	if err != nil {
		log.Error("Http get request failed", "error", err.Error())
		return cfg, err
	}
	defer func(Body io.ReadCloser) {
		cErr := Body.Close()
		if cErr != nil {
			log.Error("Failed to close response body", "error", cErr)
			err = errors.Join(cErr, err)
		}
	}(res.Body)

	if res.StatusCode != http.StatusOK {
		log.Error("Http get request failed", "status", res.Status)
		return cfg, fmt.Errorf("request failed, status is %s", res.Status)
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error("Could not read response body", "error", err.Error())
		return cfg, err
	}
	log.Debug("Successfully got response")

	if err := yaml.Unmarshal(b, &cfg); err != nil {
		log.Error("Could not unmarshal response", "error", err.Error())
		return cfg, err
	}

	return cfg, nil
}

// Shutdown stops the loader
func (hl *HttpLoader) Shutdown(ctx context.Context) {
	log := logger.FromContext(ctx)
	select {
	case hl.done <- struct{}{}:
		log.Debug("Sending signal to shut down http loader")
	default:
	}
}
