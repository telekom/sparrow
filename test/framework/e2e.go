// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/telekom/sparrow/pkg/config"
	"github.com/telekom/sparrow/pkg/sparrow"
	"github.com/telekom/sparrow/test/framework/builder"
)

var _ Runner = (*E2E)(nil)

// E2E is an end-to-end test.
type E2E struct {
	config  config.Config
	t       *testing.T
	sparrow *sparrow.Sparrow

	checks []builder.Check
	buf    bytes.Buffer

	server *http.Server

	running int32
}

// WithChecks sets the checks in the test.
func (e *E2E) WithChecks(checks ...builder.Check) *E2E {
	e.checks = checks
	for _, b := range checks {
		e.buf.Write(b.YAML(e.t))
	}
	return e
}

// UpdateChecks updates the checks of the test.
func (e *E2E) UpdateChecks(checks ...builder.Check) *E2E {
	e.checks = checks
	e.buf.Reset()
	for _, b := range checks {
		e.buf.Write(b.YAML(e.t))
	}

	// Write the config to file only if no remote server is used.
	if e.server == nil {
		if err := e.writeCheckConfig(); err != nil {
			e.t.Fatalf("Failed to write check config: %v", err)
		}
	}

	return e
}

// Run starts the test. If a remote server is configured it runs it in a goroutine.
func (e *E2E) Run(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&e.running, 0, 1) {
		e.t.Fatal("E2E.Run must be called once")
	}

	if e.server != nil {
		go func() {
			if err := e.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				e.t.Errorf("Failed to start server: %v", err)
			}
		}()
		defer func() {
			if err := e.server.Shutdown(ctx); err != nil {
				e.t.Errorf("Failed to shutdown server: %v", err)
			}
		}()
	} else {
		if err := e.writeCheckConfig(); err != nil {
			e.t.Fatalf("Failed to write check config: %v", err)
		}
	}

	return e.sparrow.Run(ctx)
}

// AwaitAll waits for provided URL to be ready, the loader to reload the configuration,
// and all checks to be executed before proceeding.
//
// Must be called after the e2e test started with [E2E.Run].
func (e *E2E) AwaitAll(url string) *E2E {
	e.t.Helper()
	const failureTimeout = 5 * time.Second
	e.AwaitStartup(url, failureTimeout).
		AwaitLoader().
		AwaitChecks()
	return e
}

// AwaitStartup waits for the provided URL to be ready.
//
// Must be called after the e2e test started with [E2E.Run].
func (e *E2E) AwaitStartup(u string, failureTimeout time.Duration) *E2E {
	e.t.Helper()
	const backoff = 100 * time.Millisecond

	// Initial delay to allow the server to start.
	<-time.After(backoff)
	if !e.isRunning() {
		e.t.Fatal("E2E.AwaitStartup must be called after E2E.Run")
	}

	deadline := time.Now().Add(failureTimeout)
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, u, http.NoBody)
		if err != nil {
			e.t.Fatalf("Failed to create request: %v", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return e
			}
		}

		<-time.After(backoff)
	}

	e.t.Fatalf("%s did not become ready within %v", u, failureTimeout)
	return e
}

// AwaitLoader waits for the loader to reload the configuration.
//
// Must be called after the e2e test started with [E2E.Run].
func (e *E2E) AwaitLoader() *E2E {
	e.t.Helper()
	if !e.isRunning() {
		e.t.Fatal("E2E.AwaitLoader must be called after E2E.Run")
	}

	e.t.Logf("Waiting %s for loader to reload configuration", e.config.Loader.Interval.String())
	<-time.After(e.config.Loader.Interval)
	return e
}

// AwaitChecks waits for all checks to be executed before proceeding.
//
// Must be called after the e2e test started with [E2E.Run].
func (e *E2E) AwaitChecks() *E2E {
	e.t.Helper()
	if !e.isRunning() {
		e.t.Fatal("E2E.AwaitChecks must be called after E2E.Run")
	}

	wait := 5 * time.Second
	for _, check := range e.checks {
		wait = max(wait, check.ExpectedWaitTime())
	}
	e.t.Logf("Waiting %s for checks to be executed", wait.String())
	<-time.After(wait)
	return e
}

// writeCheckConfig writes the check config to a file at the provided path.
func (t *E2E) writeCheckConfig() error {
	const fileMode = 0o755
	path := "testdata/checks.yaml"
	err := os.MkdirAll(filepath.Dir(path), fileMode)
	if err != nil {
		return fmt.Errorf("failed to create %q: %w", filepath.Dir(path), err)
	}

	err = os.WriteFile(path, t.buf.Bytes(), fileMode)
	if err != nil {
		return fmt.Errorf("failed to write %q: %w", path, err)
	}
	return nil
}

// isRunning returns true if the test is running.
func (t *E2E) isRunning() bool {
	return atomic.LoadInt32(&t.running) == 1
}

// WithRemote sets up a remote server to serve the check config.
func (t *E2E) WithRemote() *E2E {
	t.server = &http.Server{
		Addr:              "localhost:50505",
		Handler:           http.HandlerFunc(t.serveConfig),
		ReadHeaderTimeout: 3 * time.Second,
	}
	return t
}

// serveConfig serves the check config over HTTP as text/yaml.
func (t *E2E) serveConfig(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/yaml")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write(t.buf.Bytes())
	if err != nil {
		t.t.Fatalf("Failed to write response: %v", err)
	}
}
