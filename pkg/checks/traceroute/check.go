// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package traceroute

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/telekom/sparrow/internal/logger"
	"github.com/telekom/sparrow/internal/traceroute"
	"github.com/telekom/sparrow/pkg/checks"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var _ checks.Check = (*Traceroute)(nil)

const CheckName = "traceroute"

func NewCheck() checks.Check {
	c := &Traceroute{
		CheckBase: checks.CheckBase{
			Mu:       sync.Mutex{},
			DoneChan: make(chan struct{}, 1),
		},
		config:  Config{},
		client:  traceroute.NewClient(),
		metrics: newMetrics(),
	}
	c.tracer = otel.Tracer(c.Name())
	return c
}

type Traceroute struct {
	checks.CheckBase
	config  Config
	metrics metrics
	client  traceroute.Client
	tracer  trace.Tracer
}

type result map[string][]traceroute.Hop

// Run runs the check in a loop sending results to the provided channel
func (tr *Traceroute) Run(ctx context.Context, cResult chan checks.ResultDTO) error {
	ctx, cancel := logger.NewContextWithLogger(ctx)
	defer cancel()
	log := logger.FromContext(ctx)

	log.InfoContext(ctx, "Starting traceroute check", "interval", tr.config.Interval.String())
	for {
		select {
		case <-ctx.Done():
			log.ErrorContext(ctx, "Context canceled", "error", ctx.Err())
			return ctx.Err()
		case <-tr.DoneChan:
			return nil
		case <-time.After(tr.config.Interval):
			res := tr.check(ctx)
			tr.metrics.MinHops(res)
			cResult <- checks.ResultDTO{
				Name: tr.Name(),
				Result: &checks.Result{
					Data:      res,
					Timestamp: time.Now(),
				},
			}
			log.DebugContext(ctx, "Successfully finished traceroute check run")
		}
	}
}

// GetConfig returns the current configuration of the check
func (tr *Traceroute) GetConfig() checks.Runtime {
	tr.Mu.Lock()
	defer tr.Mu.Unlock()
	return &tr.config
}

func (tr *Traceroute) check(ctx context.Context) result {
	log := logger.FromContext(ctx)
	ctx, span := tr.tracer.Start(ctx, "traceroute.check")
	defer span.End()

	tr.Mu.Lock()
	defer tr.Mu.Unlock()

	if len(tr.config.Targets) == 0 {
		log.WarnContext(ctx, "No targets configured for traceroute check")
		return result{}
	}

	results, err := tr.client.Run(ctx, tr.config.Targets, &tr.config.Options)
	if err != nil {
		log.ErrorContext(ctx, "Failed to run traceroute", "error", err)
		span.SetStatus(codes.Error, "Failed to run traceroute")
		span.RecordError(err)
		return result{}
	}

	return aggregateResults(results)
}

// Shutdown is called once when the check is unregistered or sparrow shuts down
func (tr *Traceroute) Shutdown() {
	tr.DoneChan <- struct{}{}
	close(tr.DoneChan)
}

// UpdateConfig is called once when the check is registered
// This is also called while the check is running, if the remote config is updated
// This should return an error if the config is invalid
func (tr *Traceroute) UpdateConfig(cfg checks.Runtime) error {
	if c, ok := cfg.(*Config); ok {
		tr.Mu.Lock()
		defer tr.Mu.Unlock()

		for _, target := range tr.config.Targets {
			if !slices.Contains(c.Targets, target) {
				err := tr.metrics.Remove(target.Address)
				if err != nil {
					return err
				}
			}
		}

		tr.config = *c
		return nil
	}

	return checks.ErrConfigMismatch{
		Expected: CheckName,
		Current:  cfg.For(),
	}
}

// Schema returns an openapi3.SchemaRef of the result type returned by the check
func (tr *Traceroute) Schema() (*openapi3.SchemaRef, error) {
	return checks.OpenapiFromPerfData(result{})
}

// GetMetricCollectors allows the check to provide prometheus metric collectors
func (tr *Traceroute) GetMetricCollectors() []prometheus.Collector {
	return tr.metrics.List()
}

// Name returns the name of the check
func (tr *Traceroute) Name() string {
	return CheckName
}

// RemoveLabelledMetrics removes the metrics which have the passed
// target as a label
func (tr *Traceroute) RemoveLabelledMetrics(target string) error {
	return tr.metrics.Remove(target)
}

func aggregateResults(res traceroute.Result) result {
	agg := result{}
	for target, hops := range res {
		if len(hops) == 0 {
			// If no hops were found, we still want to return the target with an empty
			// slice to indicate that the traceroute was attempted.
			agg[target.String()] = []traceroute.Hop{}
			continue
		}
		// Aggregate hops for the target
		agg[target.String()] = hops
	}
	return agg
}
