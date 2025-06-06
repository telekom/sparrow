// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package sparrow

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/telekom/sparrow/internal/logger"
	"github.com/telekom/sparrow/pkg/checks"
	"github.com/telekom/sparrow/pkg/checks/runtime"
	"github.com/telekom/sparrow/pkg/db"
	"github.com/telekom/sparrow/pkg/factory"
	"github.com/telekom/sparrow/pkg/sparrow/metrics"
)

// ChecksController is responsible for managing checks.
type ChecksController struct {
	db      db.DB
	metrics metrics.Provider
	checks  runtime.Checks
	cResult chan checks.ResultDTO
	cErr    chan error
	done    chan struct{}
}

// NewChecksController creates a new ChecksController.
func NewChecksController(dbase db.DB, m metrics.Provider) *ChecksController {
	return &ChecksController{
		db:      dbase,
		metrics: m,
		checks:  runtime.Checks{},
		cResult: make(chan checks.ResultDTO, 8), //nolint:mnd // Buffered channel to avoid blocking the checks
		cErr:    make(chan error, 1),
		done:    make(chan struct{}, 1),
	}
}

// Run runs the ChecksController with handling results and errors.
func (cc *ChecksController) Run(ctx context.Context) error {
	log := logger.FromContext(ctx)

	for {
		select {
		case result := <-cc.cResult:
			cc.db.Save(result)
		case err := <-cc.cErr:
			var runErr *ErrRunningCheck
			if errors.As(err, &runErr) {
				cc.UnregisterCheck(ctx, runErr.Check)
			}
			log.ErrorContext(ctx, "Error while running check", "error", err)
		case <-ctx.Done():
			return ctx.Err()
		case <-cc.done:
			cc.cErr <- nil
			return nil
		}
	}
}

// Shutdown shuts down the ChecksController.
func (cc *ChecksController) Shutdown(ctx context.Context) {
	log := logger.FromContext(ctx)
	log.Info("Shutting down checks controller")

	for _, c := range cc.checks.Iter() {
		cc.UnregisterCheck(ctx, c)
	}
	cc.done <- struct{}{}
	close(cc.done)
	close(cc.cResult)
}

// Reconcile reconciles the checks.
// It registers new checks, updates existing checks and unregisters checks not in the new config.
func (cc *ChecksController) Reconcile(ctx context.Context, cfg runtime.Config) {
	log := logger.FromContext(ctx)

	newChecks, err := factory.NewChecksFromConfig(cfg)
	if err != nil {
		log.ErrorContext(ctx, "Failed to create checks from config", "error", err)
		return
	}

	// Update existing checks and create a list of checks to unregister
	var unregList []checks.Check
	for _, c := range cc.checks.Iter() {
		conf := cfg.For(c.Name())
		if conf == nil {
			unregList = append(unregList, c)
			continue
		}

		err = c.UpdateConfig(conf)
		if err != nil {
			log.ErrorContext(ctx, "Failed to set config for check", "check", c.Name(), "error", err)
		}
		delete(newChecks, c.Name())
	}

	// Unregister checks not in the new config
	for _, c := range unregList {
		cc.UnregisterCheck(ctx, c)
	}

	// Register new checks
	for _, c := range newChecks {
		cc.RegisterCheck(ctx, c)
	}
}

// RegisterCheck registers a new check.
func (cc *ChecksController) RegisterCheck(ctx context.Context, check checks.Check) {
	log := logger.FromContext(ctx).With("check", check.Name())

	// Add prometheus collectors of check to registry
	for _, collector := range check.GetMetricCollectors() {
		if err := cc.metrics.GetRegistry().Register(collector); err != nil {
			log.ErrorContext(ctx, "Could not add metrics collector to registry", "error", err)
		}
	}

	go func() {
		err := check.Run(ctx, cc.cResult)
		if err != nil {
			log.ErrorContext(ctx, "Failed to run check", "error", err)
			cc.cErr <- &ErrRunningCheck{
				Check: check,
				Err:   err,
			}
		}
	}()

	cc.checks.Add(check)
}

// UnregisterCheck unregisters a check.
func (cc *ChecksController) UnregisterCheck(ctx context.Context, check checks.Check) {
	log := logger.FromContext(ctx).With("check", check.Name())

	// Remove prometheus collectors of check from registry
	for _, metricsCollector := range check.GetMetricCollectors() {
		if !cc.metrics.GetRegistry().Unregister(metricsCollector) {
			log.ErrorContext(ctx, "Could not remove metrics collector from registry")
		}
	}

	check.Shutdown()
	cc.checks.Delete(check)
}

var oapiBoilerplate = openapi3.T{
	// this object should probably be user defined
	OpenAPI: "3.0.0",
	Info: &openapi3.Info{
		Title:       "Sparrow Metrics API",
		Description: "Serves metrics collected by sparrows checks",
		Contact: &openapi3.Contact{
			URL:   "https://caas.telekom.de",
			Email: "caas-request@telekom.de",
			Name:  "CaaS Team",
		},
	},
	Paths: &openapi3.Paths{
		Extensions: make(map[string]any),
	},
	Extensions: make(map[string]any),
	Components: &openapi3.Components{
		Schemas: make(openapi3.Schemas),
	},
	Servers: openapi3.Servers{},
}

// GenerateCheckSpecs generates the OpenAPI specifications for the given checks
// Returns the complete OpenAPI specification for all checks
func (cc *ChecksController) GenerateCheckSpecs(ctx context.Context) (openapi3.T, error) {
	log := logger.FromContext(ctx)
	doc := oapiBoilerplate
	for _, c := range cc.checks.Iter() {
		name := c.Name()
		ref, err := c.Schema()
		if err != nil {
			log.Error("Failed to get schema for check", "name", name, "error", err)
			return openapi3.T{}, &ErrCreateOpenapiSchema{name: name, err: err}
		}

		routeDesc := fmt.Sprintf("Returns the performance data for check %s", name)
		bodyDesc := fmt.Sprintf("Metrics for check %s", name)
		responses := &openapi3.Responses{}
		responses.Set(fmt.Sprint(http.StatusOK), &openapi3.ResponseRef{
			Value: &openapi3.Response{
				Description: &bodyDesc,
				Content:     openapi3.NewContentWithSchemaRef(ref, []string{"application/json"}),
			},
		})
		doc.Paths.Set(fmt.Sprintf("/v1/metrics/%s", name), &openapi3.PathItem{
			Description: name,
			Get: &openapi3.Operation{
				Description: routeDesc,
				Tags:        []string{"Metrics", name},
				Responses:   responses,
			},
		})
	}

	return doc, nil
}
