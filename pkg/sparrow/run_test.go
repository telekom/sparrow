// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package sparrow

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/telekom/sparrow/pkg/api"
	"github.com/telekom/sparrow/pkg/checks"
	"github.com/telekom/sparrow/pkg/checks/dns"
	"github.com/telekom/sparrow/pkg/checks/health"
	"github.com/telekom/sparrow/pkg/checks/latency"
	"github.com/telekom/sparrow/pkg/checks/runtime"
	"github.com/telekom/sparrow/pkg/checks/traceroute"
	"github.com/telekom/sparrow/pkg/config"
	"github.com/telekom/sparrow/pkg/sparrow/targets"
	"github.com/telekom/sparrow/pkg/sparrow/targets/interactor"
	"github.com/telekom/sparrow/pkg/sparrow/targets/remote/gitlab"
	managermock "github.com/telekom/sparrow/pkg/sparrow/targets/test"
	"github.com/telekom/sparrow/test"
)

// TestSparrow_Run_FullComponentStart tests that the Run method starts the API,
// loader and a targetManager all start.
func TestSparrow_Run_FullComponentStart(t *testing.T) {
	c := &config.Config{
		Api: api.Config{ListeningAddress: ":9090"},
		Loader: config.LoaderConfig{
			Type:     "file",
			File:     config.FileLoaderConfig{Path: "../config/test/data/config.yaml"},
			Interval: time.Second * 1,
		},
		TargetManager: targets.TargetManagerConfig{
			Enabled: true,
			Type:    "gitlab",
			General: targets.General{
				CheckInterval:        time.Second * 1,
				RegistrationInterval: time.Second * 1,
				UnhealthyThreshold:   time.Second * 1,
			},
			Config: interactor.Config{
				Gitlab: gitlab.Config{
					BaseURL:   test.GitlabBaseURL,
					Token:     "my-cool-token",
					ProjectID: 42,
				},
			},
		},
	}

	s := New(c)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() { require.ErrorIs(t, s.Run(ctx), ErrFinalShutdown) }()

	t.Log("Running sparrow for 100ms")
	<-time.After(100 * time.Millisecond)
}

// TestSparrow_Run_ContextCancel tests that after a context cancels the Run method
// will return an error and all started components will be shut down.
func TestSparrow_Run_ContextCancel(t *testing.T) {
	c := &config.Config{
		Api: api.Config{ListeningAddress: ":9090"},
		Loader: config.LoaderConfig{
			Type:     "file",
			File:     config.FileLoaderConfig{Path: "../config/test/data/config.yaml"},
			Interval: time.Second * 1,
		},
	}

	s := New(c)
	s.tarMan = &managermock.MockTargetManager{}
	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		err := s.Run(ctx)
		t.Logf("Sparrow exited with error: %v", err)
		if err == nil {
			t.Error("Sparrow.Run() should have errored out, no error received")
		}
	}()

	t.Log("Running sparrow for 10ms")
	time.Sleep(time.Millisecond * 10)

	t.Log("Canceling context and waiting for shutdown")
	cancel()
	time.Sleep(time.Millisecond * 30)
}

// TestSparrow_enrichTargets tests that the enrichTargets method
// updates the targets of the configured checks.
func TestSparrow_enrichTargets(t *testing.T) {
	t.Parallel()
	now := time.Now()
	gt := test.SparrowAZ1(t, now)

	tests := []struct {
		name          string
		config        runtime.Config
		globalTargets []checks.GlobalTarget
		expected      runtime.Config
	}{
		{
			name:          "no config",
			config:        runtime.Config{},
			globalTargets: []checks.GlobalTarget{gt},
			expected:      runtime.Config{},
		},
		{
			name: "config with no targets",
			config: runtime.Config{
				Health: &health.Config{
					Targets: nil,
				},
				Latency: &latency.Config{
					Targets: nil,
				},
			},
			globalTargets: []checks.GlobalTarget{gt},
			expected: runtime.Config{
				Health: &health.Config{
					Targets: []string{gt.String()},
				},
				Latency: &latency.Config{
					Targets: []string{gt.String()},
				},
			},
		},
		{
			name: "config with targets (health + latency)",
			config: runtime.Config{
				Health: &health.Config{
					Targets: []string{"https://telekom.com"},
				},
				Latency: &latency.Config{
					Targets: []string{"https://telekom.com"},
				},
			},
			globalTargets: []checks.GlobalTarget{gt},
			expected: runtime.Config{
				Health: &health.Config{
					Targets: []string{"https://telekom.com", gt.String()},
				},
				Latency: &latency.Config{
					Targets: []string{"https://telekom.com", gt.String()},
				},
			},
		},
		{
			name: "config with targets (dns)",
			config: runtime.Config{
				Dns: &dns.Config{
					Targets: []string{"telekom.com"},
				},
			},
			globalTargets: []checks.GlobalTarget{gt},
			expected: runtime.Config{
				Dns: &dns.Config{
					Targets: []string{"telekom.com", gt.Hostname()},
				},
			},
		},
		{
			name: "config with targets (traceroute)",
			config: runtime.Config{
				Traceroute: &traceroute.Config{
					Targets: []traceroute.Target{
						{Addr: "telekom.com", Port: 443},
					},
				},
			},
			globalTargets: []checks.GlobalTarget{gt},
			expected: runtime.Config{
				Traceroute: &traceroute.Config{
					Targets: []traceroute.Target{
						{Addr: "telekom.com", Port: 443},
						{Addr: gt.Hostname(), Port: portOrFail(t, gt)},
					},
				},
			},
		},
		{
			name: "config has a target already present in global targets - no duplicates",
			config: runtime.Config{
				Health: &health.Config{
					Targets: []string{gt.String()},
				},
			},
			globalTargets: []checks.GlobalTarget{gt},
			expected: runtime.Config{
				Health: &health.Config{
					Targets: []string{gt.String()},
				},
			},
		},
		{
			name: "global targets contains self - do not add to config",
			config: runtime.Config{
				Health: &health.Config{
					Targets: []string{gt.String()},
				},
			},
			globalTargets: []checks.GlobalTarget{
				gt,
				test.SparrowLocal(t, now),
			},
			expected: runtime.Config{
				Health: &health.Config{
					Targets: []string{gt.String()},
				},
			},
		},
		{
			name: "global targets contains http and https - dns validation still works does not fail and splits off scheme",
			config: runtime.Config{
				Dns: &dns.Config{
					Targets: []string{},
				},
			},
			globalTargets: []checks.GlobalTarget{gt, test.SparrowAZ2(t, now)},
			expected: runtime.Config{
				Dns: &dns.Config{
					Targets: []string{gt.Hostname(), test.SparrowAZ2(t, now).Hostname()},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Sparrow{
				tarMan: &managermock.MockTargetManager{
					Targets: tt.globalTargets,
				},
				config: &config.Config{
					SparrowName: test.SparrowLocalName,
				},
			}
			got := s.enrichTargets(t.Context(), tt.config)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func portOrFail(t testing.TB, g checks.GlobalTarget) int {
	t.Helper()
	port, err := g.Port()
	require.NoError(t, err)
	return port
}
