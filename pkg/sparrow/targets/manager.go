// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package targets

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	smetrics "github.com/telekom/sparrow/pkg/sparrow/metrics"

	"github.com/telekom/sparrow/internal/logger"
	"github.com/telekom/sparrow/pkg/checks"
	"github.com/telekom/sparrow/pkg/sparrow/targets/remote"
)

var _ TargetManager = (*manager)(nil)

const shutdownTimeout = 30 * time.Second

// manager implements the TargetManager interface
type manager struct {
	// targets contains the current global targets
	targets []checks.GlobalTarget
	// mu is used for mutex locking/unlocking
	mu sync.RWMutex
	// done is used to signal the reconciliation routine to stop
	done chan struct{}
	// name is the DNS name used for self-registration
	name string
	// registered contains whether the instance has already registered itself as a global target
	registered bool
	// cfg contains the general configuration for the target manager
	cfg General
	// interactor is the remote interactor used to interact with the remote state backend
	interactor remote.Interactor
	// metrics allows access to the central metrics provider
	metrics metrics
	// metricsProvider is the metrics provider used to register the metrics
	metricsProvider smetrics.Provider
}

// metrics contains the prometheus metrics for the target manager
type metrics struct {
	registered prometheus.Gauge
}

// newMetrics creates a new metrics struct
func newMetrics() metrics {
	return metrics{
		registered: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "sparrow_target_manager_registered",
			Help: "Indicates whether the instance is registered as a global target",
		}),
	}
}

// NewManager creates a new target manager
func NewManager(name string, cfg TargetManagerConfig, mp smetrics.Provider) TargetManager { //nolint:gocritic // no performance concerns yet
	m := newMetrics()
	mp.GetRegistry().MustRegister(m.registered)

	return &manager{
		name:            name,
		cfg:             cfg.General,
		mu:              sync.RWMutex{},
		done:            make(chan struct{}, 1),
		interactor:      cfg.Type.Interactor(&cfg.Config),
		metrics:         m,
		metricsProvider: mp,
	}
}

// Reconcile reconciles the targets of the target manager.
// The global targets are parsed from a remote state backend.
//
// The global targets are evaluated for their healthiness
// and unhealthy targets are filtered out.
func (t *manager) Reconcile(ctx context.Context) error {
	log := logger.FromContext(ctx)

	checkTimer := startTimer(t.cfg.CheckInterval)
	registrationTimer := startTimer(t.cfg.RegistrationInterval)
	updateTimer := startTimer(t.cfg.UpdateInterval)

	defer checkTimer.Stop()
	defer registrationTimer.Stop()
	defer updateTimer.Stop()

	log.InfoContext(ctx, "Starting target manager reconciliation")
	for {
		select {
		case <-ctx.Done():
			log.ErrorContext(ctx, "Error while reconciling targets", "error", ctx.Err())
			return ctx.Err()
		case <-t.done:
			log.InfoContext(ctx, "Target manager reconciliation stopped")
			return nil
		case <-checkTimer.C:
			err := t.refreshTargets(ctx)
			if err != nil {
				log.WarnContext(ctx, "Failed to get global targets", "error", err)
			}
			checkTimer.Reset(t.cfg.CheckInterval)
		case <-registrationTimer.C:
			err := t.register(ctx)
			if err != nil {
				log.WarnContext(ctx, "Failed to register self as global target", "error", err)
			}
			registrationTimer.Reset(t.cfg.RegistrationInterval)
		case <-updateTimer.C:
			err := t.update(ctx)
			if err != nil {
				log.WarnContext(ctx, "Failed to update registration", "error", err)
			}
			updateTimer.Reset(t.cfg.UpdateInterval)
		}
	}
}

// GetTargets returns the current global targets
func (t *manager) GetTargets() []checks.GlobalTarget {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.targets
}

// Shutdown shuts down the target manager
func (t *manager) Shutdown(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	errC := ctx.Err()
	log := logger.FromContext(ctx)
	log.DebugContext(ctx, "Shut down signal received")
	ctxS, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if t.registered {
		f := remote.File{
			AuthorEmail:   fmt.Sprintf("%s@sparrow", t.name),
			AuthorName:    t.name,
			CommitMessage: "Unregistering global target",
		}
		f.SetFileName(fmt.Sprintf("%s.json", t.name))
		err := t.interactor.DeleteFile(ctxS, f)
		if err != nil {
			log.ErrorContext(ctx, "Failed to shutdown gracefully", "error", err)
			return fmt.Errorf("failed to shutdown gracefully: %w", errors.Join(errC, err))
		}
		t.registered = false
		t.metrics.registered.Set(0)
		log.InfoContext(ctx, "Successfully unregistered as global target")
	}

	select {
	case t.done <- struct{}{}:
		log.DebugContext(ctx, "Stopping gitlab reconciliation routine")
	default:
	}

	return nil
}

// register registers the current instance as a global target
func (t *manager) register(ctx context.Context) error {
	log := logger.FromContext(ctx)

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.registered {
		log.DebugContext(ctx, "Already registered as global target")
		return nil
	}

	f := remote.File{
		AuthorEmail:   fmt.Sprintf("%s@sparrow", t.name),
		AuthorName:    t.name,
		CommitMessage: "Initial registration",
		Content:       checks.GlobalTarget{Url: fmt.Sprintf("%s://%s", t.cfg.Scheme, t.name), LastSeen: time.Now().UTC()},
	}
	f.SetFileName(fmt.Sprintf("%s.json", t.name))

	log.DebugContext(ctx, "Registering as global target")
	err := t.interactor.PostFile(ctx, f)
	if err != nil {
		log.ErrorContext(ctx, "Failed to register global gitlabTargetManager", "error", err)
		return err
	}
	log.InfoContext(ctx, "Successfully registered")
	t.registered = true
	t.metrics.registered.Set(1)

	return nil
}

// update updates the registration file of the current sparrow instance
func (t *manager) update(ctx context.Context) error {
	log := logger.FromContext(ctx)

	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.registered {
		log.DebugContext(ctx, "Not registered as global target, no update done.")
		return nil
	}

	f := remote.File{
		AuthorEmail:   fmt.Sprintf("%s@sparrow", t.name),
		AuthorName:    t.name,
		CommitMessage: "Updated registration",
		Content:       checks.GlobalTarget{Url: fmt.Sprintf("%s://%s", t.cfg.Scheme, t.name), LastSeen: time.Now().UTC()},
	}
	f.SetFileName(fmt.Sprintf("%s.json", t.name))

	log.DebugContext(ctx, "Updating instance registration")
	err := t.interactor.PutFile(ctx, f)
	if err != nil {
		log.ErrorContext(ctx, "Failed to update registration", "error", err)
		return err
	}
	log.DebugContext(ctx, "Successfully updated registration")
	return nil
}

// refreshTargets updates the targets with the latest available healthy targets
func (t *manager) refreshTargets(ctx context.Context) error {
	log := logger.FromContext(ctx)
	t.mu.Lock()
	defer t.mu.Unlock()
	var healthyTargets []checks.GlobalTarget
	targets, err := t.interactor.FetchFiles(ctx)
	if err != nil {
		log.ErrorContext(ctx, "Failed to update global targets", "error", err)
		return err
	}

	// filter unhealthy targets - this may be removed in the future
	for _, target := range targets {
		if !t.registered && target.Url == fmt.Sprintf("%s://%s", t.cfg.Scheme, t.name) {
			log.DebugContext(ctx, "Found self as global target", "lastSeenMinsAgo", time.Since(target.LastSeen).Minutes())
			t.registered = true
			t.metrics.registered.Set(1)
		}

		if t.cfg.UnhealthyThreshold == 0 {
			healthyTargets = append(healthyTargets, target)
			continue
		}

		if time.Now().Add(-t.cfg.UnhealthyThreshold).After(target.LastSeen) {
			log.DebugContext(ctx, "Skipping unhealthy target", "target", target)
			continue
		}
		healthyTargets = append(healthyTargets, target)
	}

	t.targets = healthyTargets
	log.DebugContext(ctx, "Updated global targets", "targets", len(t.targets))
	return nil
}

// startTimer creates a new timer with the given duration.
// If the duration is 0, the timer is stopped.
func startTimer(d time.Duration) *time.Timer {
	res := time.NewTimer(d)
	if d == 0 {
		res.Stop()
	}
	return res
}
