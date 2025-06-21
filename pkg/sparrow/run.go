// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package sparrow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/telekom/sparrow/internal/logger"
	"github.com/telekom/sparrow/pkg/api"
	"github.com/telekom/sparrow/pkg/checks"
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
		shutOnce:   sync.Once{},
	}

	if cfg.HasTargetManager() {
		gm := targets.NewManager(cfg.SparrowName, cfg.TargetManager, m)
		sparrow.tarMan = gm
	}
	sparrow.loader = config.NewLoader(cfg, sparrow.cRuntime)

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
		case cfg := <-s.cRuntime:
			cfg = s.enrichTargets(ctx, cfg)
			s.controller.Reconcile(ctx, cfg)
		case <-ctx.Done():
			s.shutdown(ctx)
		case err := <-s.cErr:
			if err != nil {
				log.Error("Non-recoverable error in sparrow component", "error", err)
				s.shutdown(ctx)
			}
		case <-s.cDone:
			log.InfoContext(ctx, "Sparrow was shut down")
			return ErrFinalShutdown
		}
	}
}

// enrichTargets updates the targets of the sparrow's checks with the
// global targets. Per default, the two target lists are merged.
func (s *Sparrow) enrichTargets(ctx context.Context, cfg runtime.Config) runtime.Config {
	if cfg.Empty() || s.tarMan == nil {
		return cfg
	}
	var gts []checks.GlobalTarget
	for _, t := range s.tarMan.GetTargets() {
		// We don't need to enrich the configs with the own hostname
		if s.config.SparrowName == t.Hostname() {
			continue
		}
		gts = append(gts, t)
	}
	return cfg.Enrich(ctx, gts)
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
