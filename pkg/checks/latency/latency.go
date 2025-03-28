// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package latency

import (
	"context"
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
	_ checks.Check   = (*Latency)(nil)
	_ checks.Runtime = (*Config)(nil)
)

const CheckName = "latency"

// Latency is a check that measures the latency to an endpoint
type Latency struct {
	checks.CheckBase
	config  Config
	metrics metrics
}

// NewCheck creates a new instance of the latency check
func NewCheck() checks.Check {
	return &Latency{
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

// result represents the result of a single latency check for a specific target
type result struct {
	Code  int     `json:"code"`
	Error *string `json:"error"`
	Total float64 `json:"total"`
}

// Run starts the latency check
func (l *Latency) Run(ctx context.Context, cResult chan checks.ResultDTO) error {
	ctx, cancel := logger.NewContextWithLogger(ctx)
	defer cancel()
	log := logger.FromContext(ctx)

	log.Info("Starting latency check", "interval", l.config.Interval.String())
	for {
		select {
		case <-ctx.Done():
			log.Error("Context canceled", "err", ctx.Err())
			return ctx.Err()
		case <-l.DoneChan:
			return nil
		case <-time.After(l.config.Interval):
			res := l.check(ctx)

			cResult <- checks.ResultDTO{
				Name: l.Name(),
				Result: &checks.Result{
					Data:      res,
					Timestamp: time.Now(),
				},
			}
			log.Debug("Successfully finished latency check run")
		}
	}
}

func (l *Latency) Shutdown() {
	l.DoneChan <- struct{}{}
	close(l.DoneChan)
}

// UpdateConfig sets the configuration for the latency check
func (l *Latency) UpdateConfig(cfg checks.Runtime) error {
	if c, ok := cfg.(*Config); ok {
		l.Mu.Lock()
		defer l.Mu.Unlock()

		for _, target := range l.config.Targets {
			if !slices.Contains(c.Targets, target) {
				err := l.metrics.Remove(target)
				if err != nil {
					return err
				}
			}
		}

		l.config = *c
		return nil
	}

	return checks.ErrConfigMismatch{
		Expected: CheckName,
		Current:  cfg.For(),
	}
}

// GetConfig returns the current configuration of the latency Check
func (l *Latency) GetConfig() checks.Runtime {
	l.Mu.Lock()
	defer l.Mu.Unlock()
	return &l.config
}

// Name returns the name of the check
func (l *Latency) Name() string {
	return CheckName
}

// Schema provides the schema of the data that will be provided
// by the latency check
func (l *Latency) Schema() (*openapi3.SchemaRef, error) {
	return checks.OpenapiFromPerfData[map[string]result](make(map[string]result))
}

// GetMetricCollectors returns all metric collectors of check
func (l *Latency) GetMetricCollectors() []prometheus.Collector {
	return []prometheus.Collector{
		l.metrics.totalDuration,
		l.metrics.count,
		l.metrics.histogram,
	}
}

// RemoveLabelledMetrics removes the metrics which have the passed target as a label
func (l *Latency) RemoveLabelledMetrics(target string) error {
	return l.metrics.Remove(target)
}

// check performs a latency check using a retry function
// to get the latency to all targets
func (l *Latency) check(ctx context.Context) map[string]result {
	log := logger.FromContext(ctx)
	log.Debug("Checking latency")
	if len(l.config.Targets) == 0 {
		log.Debug("No targets defined")
		return map[string]result{}
	}
	log.Debug("Getting latency status for each target in separate routine", "amount", len(l.config.Targets))

	var mu sync.Mutex
	var wg sync.WaitGroup
	results := map[string]result{}

	client := &http.Client{
		Timeout: l.config.Timeout,
	}
	for _, t := range l.config.Targets {
		target := t
		wg.Add(1)
		lo := log.With("target", target)

		getLatencyRetry := helper.Retry(func(ctx context.Context) error {
			res, err := getLatency(ctx, client, target)
			mu.Lock()
			defer mu.Unlock()
			results[target] = res
			if err != nil {
				return err
			}
			return nil
		}, l.config.Retry)

		go func() {
			defer wg.Done()

			lo.Debug("Starting retry routine to get latency status")
			if err := getLatencyRetry(ctx); err != nil {
				lo.Error("Error while checking latency", "error", err)
			}

			lo.Debug("Successfully got latency status of target")
			mu.Lock()
			defer mu.Unlock()

			l.metrics.totalDuration.WithLabelValues(target).Set(results[target].Total)
			l.metrics.count.WithLabelValues(target).Inc()
			l.metrics.histogram.WithLabelValues(target).Observe(results[target].Total)
		}()
	}

	log.Debug("Waiting for all routines to finish")
	wg.Wait()

	log.Debug("Successfully got latency status from all targets")
	return results
}

// getLatency performs an HTTP get request and returns ok if request succeeds
func getLatency(ctx context.Context, c *http.Client, url string) (result, error) {
	log := logger.FromContext(ctx).With("url", url)
	var res result

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		log.Error("Error while creating request", "error", err)
		errval := err.Error()
		res.Error = &errval
		return res, err
	}

	start := time.Now()
	resp, err := c.Do(req) //nolint:bodyclose // Closed in defer below
	if err != nil {
		log.Error("Error while checking latency", "error", err)
		errval := err.Error()
		res.Error = &errval
		return res, err
	}
	end := time.Now()

	res.Code = resp.StatusCode
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	res.Total = end.Sub(start).Seconds()
	return res, nil
}
