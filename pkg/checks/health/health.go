// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package health

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/telekom/sparrow/internal/helper"
	"github.com/telekom/sparrow/internal/logger"
	"github.com/telekom/sparrow/pkg/checks"
)

var (
	_            checks.Check   = (*Health)(nil)
	_            checks.Runtime = (*Config)(nil)
	stateMapping                = map[int]string{
		0: "unhealthy",
		1: "healthy",
	}
)

const CheckName = "health"

// Health is a check that measures the availability of an endpoint
type Health struct {
	checks.CheckBase
	config  Config
	metrics metrics
}

// NewCheck creates a new instance of the health check
func NewCheck() checks.Check {
	return &Health{
		CheckBase: checks.CheckBase{
			Mu:       sync.Mutex{},
			DoneChan: make(chan struct{}, 1),
		},
		config: Config{
			Retry: checks.DefaultRetry,
		},
		metrics: newMetrics(),
	}
}

// Run starts the health check
func (h *Health) Run(ctx context.Context, cResult chan checks.ResultDTO) error {
	ctx, cancel := logger.NewContextWithLogger(ctx)
	defer cancel()
	log := logger.FromContext(ctx)

	interval := h.GetConfig().(*Config).Interval

	log.Info("Starting health check", "interval", interval.String())
	for {
		select {
		case <-ctx.Done():
			log.Error("Context canceled", "err", ctx.Err())
			return ctx.Err()
		case <-h.DoneChan:
			log.Debug("Soft shut down")
			return nil
		case <-time.After(interval):
			res := h.check(ctx)

			cResult <- checks.ResultDTO{
				Name: h.Name(),
				Result: &checks.Result{
					Data:      res,
					Timestamp: time.Now(),
				},
			}
			log.Debug("Successfully finished health check run")

			// Re-read interval in case config was updated
			interval = h.GetConfig().(*Config).Interval
		}
	}
}

// Shutdown is called once when the check is unregistered or sparrow shuts down
func (h *Health) Shutdown() {
	h.DoneChan <- struct{}{}
	close(h.DoneChan)
}

// UpdateConfig sets the configuration for the health check
func (h *Health) UpdateConfig(cfg checks.Runtime) error {
	if c, ok := cfg.(*Config); ok {
		h.Mu.Lock()
		defer h.Mu.Unlock()

		for _, target := range h.config.Targets {
			if !slices.Contains(c.Targets, target) {
				err := h.metrics.Remove(target)
				if err != nil {
					return err
				}
			}
		}

		h.config = *c
		return nil
	}

	return checks.ErrConfigMismatch{
		Expected: CheckName,
		Current:  cfg.For(),
	}
}

// GetConfig returns a copy of the current configuration of the check
func (h *Health) GetConfig() checks.Runtime {
	h.Mu.Lock()
	defer h.Mu.Unlock()
	// Return a copy to prevent race conditions when the config is read while being updated
	configCopy := h.config
	return &configCopy
}

// Name returns the name of the check
func (h *Health) Name() string {
	return CheckName
}

// Schema provides the schema of the data that will be provided
// by the health check
func (h *Health) Schema() (*openapi3.SchemaRef, error) {
	return checks.OpenapiFromPerfData[map[string]string](map[string]string{})
}

// GetMetricCollectors returns all metric collectors of check
func (h *Health) GetMetricCollectors() []prometheus.Collector {
	return []prometheus.Collector{
		h.metrics,
	}
}

// RemoveLabelledMetrics removes the metrics which have the passed
// target as a label
func (h *Health) RemoveLabelledMetrics(target string) error {
	return h.metrics.Remove(target)
}

// check performs a health check using a retry function
// to get the health status for all targets
func (h *Health) check(ctx context.Context) map[string]string {
	log := logger.FromContext(ctx)
	log.Debug("Checking health")

	// Get a copy of the config to avoid race conditions
	cfg := h.GetConfig().(*Config)

	if len(cfg.Targets) == 0 {
		log.Debug("No targets defined")
		return map[string]string{}
	}
	log.Debug("Getting health status for each target in separate routine", "amount", len(cfg.Targets))

	var wg sync.WaitGroup
	var mu sync.Mutex
	results := map[string]string{}

	client := &http.Client{
		Timeout: cfg.Timeout,
	}
	for _, t := range cfg.Targets {
		target := t
		wg.Add(1)
		l := log.With("target", target)

		getHealthRetry := helper.Retry(func(ctx context.Context) error {
			return getHealth(ctx, client, target)
		}, cfg.Retry)

		go func() {
			defer wg.Done()
			state := 1

			l.Debug("Starting retry routine to get health status")
			if err := getHealthRetry(ctx); err != nil {
				state = 0
				l.Warn(fmt.Sprintf("Health check failed after %d retries", cfg.Retry.Count), "error", err)
			}

			l.Debug("Successfully got health status of target", "status", stateMapping[state])
			mu.Lock()
			defer mu.Unlock()
			results[target] = stateMapping[state]

			h.metrics.WithLabelValues(target).Set(float64(state))
		}()
	}

	log.Debug("Waiting for all routines to finish")
	wg.Wait()

	log.Debug("Successfully got health status from all targets")
	return results
}

// getHealth performs an HTTP get request and returns ok if status code is 200
func getHealth(ctx context.Context, client *http.Client, url string) error {
	log := logger.FromContext(ctx).With("url", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		log.Error("Error while creating request", "error", err)
		return err
	}

	resp, err := client.Do(req) //nolint:bodyclose // Closed in defer below
	if err != nil {
		log.Error("Error while requesting health", "error", err)
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Error("Failed to close response body", "error", err.Error())
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Warn("Health request was not ok (HTTP Status 200)", "status", resp.Status)
		return fmt.Errorf("request failed, status is %s", resp.Status)
	}

	return nil
}
