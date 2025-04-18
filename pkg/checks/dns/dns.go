// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package dns

import (
	"context"
	"net"
	"slices"
	"sync"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/telekom/sparrow/internal/helper"
	"github.com/telekom/sparrow/internal/logger"
	"github.com/telekom/sparrow/pkg/checks"
)

var (
	_ checks.Check   = (*DNS)(nil)
	_ checks.Runtime = (*Config)(nil)
)

const CheckName = "dns"

// DNS is a check that resolves the names and addresses
type DNS struct {
	checks.CheckBase
	config  Config
	metrics metrics
	client  Resolver
}

func (d *DNS) GetConfig() checks.Runtime {
	d.Mu.Lock()
	defer d.Mu.Unlock()
	return &d.config
}

func (d *DNS) Name() string {
	return CheckName
}

// NewCheck creates a new instance of the dns check
func NewCheck() checks.Check {
	return &DNS{
		CheckBase: checks.CheckBase{
			Mu:       sync.Mutex{},
			DoneChan: make(chan struct{}, 1),
		},
		config: Config{
			Retry: checks.DefaultRetry,
		},
		metrics: newMetrics(),
		client:  NewResolver(),
	}
}

// result represents the result of a single DNS check for a specific target
type result struct {
	Resolved []string `json:"resolved"`
	Error    *string  `json:"error"`
	Total    float64  `json:"total"`
}

// Run starts the dns check
func (d *DNS) Run(ctx context.Context, cResult chan checks.ResultDTO) error {
	ctx, cancel := logger.NewContextWithLogger(ctx)
	defer cancel()
	log := logger.FromContext(ctx)

	log.Info("Starting dns check", "interval", d.config.Interval.String())
	for {
		select {
		case <-ctx.Done():
			log.Error("Context canceled", "err", ctx.Err())
			return ctx.Err()
		case <-d.DoneChan:
			return nil
		case <-time.After(d.config.Interval):
			res := d.check(ctx)

			cResult <- checks.ResultDTO{
				Name: d.Name(),
				Result: &checks.Result{
					Data:      res,
					Timestamp: time.Now(),
				},
			}
			log.Debug("Successfully finished dns check run")
		}
	}
}

func (d *DNS) Shutdown() {
	d.DoneChan <- struct{}{}
	close(d.DoneChan)
}

func (d *DNS) UpdateConfig(cfg checks.Runtime) error {
	if c, ok := cfg.(*Config); ok {
		d.Mu.Lock()
		defer d.Mu.Unlock()

		for _, target := range d.config.Targets {
			if !slices.Contains(c.Targets, target) {
				err := d.metrics.Remove(target)
				if err != nil {
					return err
				}
			}
		}

		d.config = *c
		return nil
	}

	return checks.ErrConfigMismatch{
		Expected: CheckName,
		Current:  cfg.For(),
	}
}

// Schema provides the schema of the data that will be provided
// by the dns check
func (d *DNS) Schema() (*openapi3.SchemaRef, error) {
	return checks.OpenapiFromPerfData(make(map[string]result))
}

// GetMetricCollectors returns all metric collectors of check
func (d *DNS) GetMetricCollectors() []prometheus.Collector {
	return d.metrics.GetCollectors()
}

// RemoveLabelledMetrics removes the metrics which have the passed
// target as a label
func (d *DNS) RemoveLabelledMetrics(target string) error {
	return d.metrics.Remove(target)
}

// check performs DNS checks for all configured targets using a custom net.Resolver.
// Returns a map where each target is associated with its DNS check result.
func (d *DNS) check(ctx context.Context) map[string]result {
	log := logger.FromContext(ctx)
	log.Debug("Checking dns")
	if len(d.config.Targets) == 0 {
		log.Debug("No targets defined")
		return map[string]result{}
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	results := map[string]result{}

	d.client.SetDialer(&net.Dialer{
		Timeout: d.config.Timeout,
	})

	log.Debug("Getting dns status for each target in separate routine", "amount", len(d.config.Targets))
	for _, t := range d.config.Targets {
		target := t
		wg.Add(1)
		lo := log.With("target", target)

		getDNSRetry := helper.Retry(func(ctx context.Context) error {
			res, err := getDNS(ctx, d.client, target)
			mu.Lock()
			defer mu.Unlock()
			results[target] = res
			if err != nil {
				return err
			}
			return nil
		}, d.config.Retry)

		go func() {
			defer wg.Done()
			status := 1

			lo.Debug("Starting retry routine to get dns status")
			if err := getDNSRetry(ctx); err != nil {
				status = 0
				lo.Warn("Error while looking up address", "error", err)
			}
			lo.Debug("DNS check completed for target")

			mu.Lock()
			defer mu.Unlock()
			d.metrics.Set(target, results, float64(status))
		}()
	}
	wg.Wait()

	log.Debug("Successfully resolved names/addresses from all targets")
	return results
}

// getDNS performs a DNS resolution for the given address using the specified net.Resolver.
// If the address is an IP address, LookupAddr is used to perform a reverse DNS lookup.
// If the address is a hostname, LookupHost is used to find its IP addresses.
// Returns a result struct containing the outcome of the DNS query.
func getDNS(ctx context.Context, c Resolver, address string) (result, error) {
	log := logger.FromContext(ctx).With("address", address)
	var res result

	var lookupFunc func(context.Context, string) ([]string, error)
	ip := net.ParseIP(address)
	if ip != nil {
		lookupFunc = c.LookupAddr
	} else {
		lookupFunc = c.LookupHost
	}

	start := time.Now()
	resp, err := lookupFunc(ctx, address)
	if err != nil {
		log.Error("Error while looking up address", "error", err)
		errval := err.Error()
		res.Error = &errval
		return res, err
	}
	rtt := time.Since(start).Seconds()

	res.Resolved = resp
	res.Total = rtt

	return res, nil
}
