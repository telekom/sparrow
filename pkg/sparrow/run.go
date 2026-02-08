// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package sparrow

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/telekom/sparrow/internal/logger"
	"github.com/telekom/sparrow/pkg/api"
	"github.com/telekom/sparrow/pkg/checks/runtime"
	"github.com/telekom/sparrow/pkg/config"
	"github.com/telekom/sparrow/pkg/db"
	"github.com/telekom/sparrow/pkg/sparrow/metrics"
	"github.com/telekom/sparrow/pkg/sparrow/targets"
)

const shutdownTimeout = time.Second * 90

// Sparrow is the main struct of the sparrow application
type Sparrow struct {
	// config is the startup configuration of the sparrow
	config *config.Config
	// db is the database used to store the check results
	db db.DB
	// api is the sparrow's API
	api api.API
	// loader is used to load the runtime configuration
	loader config.Loader
	// tarMan is the target manager that is used to manage global targets
	tarMan targets.TargetManager
	// metrics is used to collect metrics
	metrics metrics.Provider
	// controller is used to manage the checks
	controller *ChecksController
	// cRuntime is used to signal that the runtime configuration has changed
	cRuntime chan runtime.Config
	// cErr is used to handle non-recoverable errors of the sparrow components
	cErr chan error
	// cDone is used to signal that the sparrow was shut down because of an error
	cDone chan struct{}
	// cTargets is used to signal when the target list has changed
	cTargets chan struct{}
	// runtimeConfig stores the latest runtime configuration for reapplication
	runtimeConfig runtime.Config
	// configMutex protects access to runtimeConfig
	configMutex sync.RWMutex
	// shutOnce is used to ensure that the shutdown function is only called once
	shutOnce sync.Once
}

// New creates a new sparrow from a given configfile
func New(cfg *config.Config) *Sparrow {
	m := metrics.New(cfg.Telemetry)
	dbase := db.NewInMemory()

	sparrow := &Sparrow{
		config:     cfg,
		db:         dbase,
		api:        api.New(cfg.Api),
		metrics:    m,
		controller: NewChecksController(dbase, m),
		cRuntime:   make(chan runtime.Config, 1),
		cErr:       make(chan error, 1),
		cDone:      make(chan struct{}, 1),
		cTargets:   make(chan struct{}, 1),
		shutOnce:   sync.Once{},
	}

	if cfg.HasTargetManager() {
		gm := targets.NewManager(cfg.SparrowName, cfg.TargetManager, m, sparrow.cTargets)
		sparrow.tarMan = gm
	}
	sparrow.loader = config.NewLoader(cfg, sparrow.cRuntime)

	// Register ownership metadata as Prometheus info metric (once per instance)
	if err := metrics.RegisterInstanceInfo(m.GetRegistry(), cfg.SparrowName, cfg.Metadata.Team.Name, cfg.Metadata.Team.Email, cfg.Metadata.Platform); err != nil {
		// Non-fatal: instance can run without the info metric
		// Logging requires context; use background with logger for startup
		log := logger.FromContext(context.Background())
		log.Error("Failed to register sparrow_instance_info metric", "error", err)
	}

	return sparrow
}

// Run starts the sparrow
func (s *Sparrow) Run(ctx context.Context) error {
	ctx, cancel := logger.NewContextWithLogger(ctx)
	log := logger.FromContext(ctx)
	defer cancel()

	err := s.metrics.InitTracing(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize tracing: %w", err)
	}

	go func() {
		s.cErr <- s.loader.Run(ctx)
	}()
	go func() {
		if s.tarMan != nil {
			s.cErr <- s.tarMan.Reconcile(ctx)
		}
	}()

	go func() {
		s.cErr <- s.startupAPI(ctx)
	}()

	go func() {
		s.cErr <- s.controller.Run(ctx)
	}()

	for {
		select {
		// New runtime configuration available
		case cfg := <-s.cRuntime:
			cfg = s.enrichTargets(ctx, cfg)
			s.controller.Reconcile(ctx, cfg)
			s.configMutex.Lock()
			s.runtimeConfig = cfg
			s.configMutex.Unlock()
		// Targets changed
		case <-s.cTargets:
			s.configMutex.RLock()
			cfg := s.runtimeConfig
			s.configMutex.RUnlock()
			if !cfg.Empty() {
				cfg = s.enrichTargets(ctx, cfg)
				s.controller.Reconcile(ctx, cfg)
				log.DebugContext(ctx, "Reapplied configuration due to target changes")
			}
		case <-ctx.Done():
			s.shutdown(ctx)
		case err := <-s.cErr:
			if err != nil {
				log.Error("Non-recoverable error in sparrow component", "error", err)
				s.shutdown(ctx)
			}
		case <-s.cDone:
			log.InfoContext(ctx, "Sparrow was shut down")
			return fmt.Errorf("sparrow was shut down")
		}
	}
}

// enrichTargets updates the targets of the sparrow's checks with the
// global targets. Per default, the two target lists are merged.
func (s *Sparrow) enrichTargets(ctx context.Context, cfg runtime.Config) runtime.Config {
	l := logger.FromContext(ctx)
	if cfg.Empty() || s.tarMan == nil {
		return cfg
	}

	for _, gt := range s.tarMan.GetTargets() {
		u, err := url.Parse(gt.Url)
		if err != nil {
			l.Error("Failed to parse global target URL", "error", err, "url", gt.Url)
			continue
		}

		// split off hostWithoutPort because it could contain a port
		hostWithoutPort := strings.Split(u.Host, ":")[0]
		if hostWithoutPort == s.config.SparrowName {
			continue
		}

		if cfg.HasHealthCheck() && !slices.Contains(cfg.Health.Targets, u.String()) {
			cfg.Health.Targets = append(cfg.Health.Targets, u.String())
		}
		if cfg.HasLatencyCheck() && !slices.Contains(cfg.Latency.Targets, u.String()) {
			cfg.Latency.Targets = append(cfg.Latency.Targets, u.String())
		}
		if cfg.HasDNSCheck() && !slices.Contains(cfg.Dns.Targets, hostWithoutPort) {
			cfg.Dns.Targets = append(cfg.Dns.Targets, hostWithoutPort)
		}
	}

	return cfg
}

// shutdown shuts down the sparrow and all managed components gracefully.
// It returns an error if one is present in the context or if any of the
// components fail to shut down.
func (s *Sparrow) shutdown(ctx context.Context) {
	errC := ctx.Err()
	log := logger.FromContext(ctx)
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	s.shutOnce.Do(func() {
		log.InfoContext(ctx, "Shutting down sparrow")
		var sErrs ErrShutdown
		if s.tarMan != nil {
			sErrs.errTarMan = s.tarMan.Shutdown(ctx)
		}
		sErrs.errAPI = s.api.Shutdown(ctx)
		sErrs.errMetrics = s.metrics.Shutdown(ctx)
		s.loader.Shutdown(ctx)
		s.controller.Shutdown(ctx)

		if sErrs.HasError() {
			log.ErrorContext(ctx, "Failed to shutdown gracefully", "contextError", errC, "errors", sErrs)
		}

		// Signal that shutdown is complete
		s.cDone <- struct{}{}
	})
}
