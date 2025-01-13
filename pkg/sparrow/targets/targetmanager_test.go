// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package targets

import (
	"context"
	"testing"
	"time"
)

func TestTargetManagerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     TargetManagerConfig
		wantErr bool
	}{
		{
			name:    "empty config",
			wantErr: true,
		},
		{
			name: "valid config - non-zero values",
			cfg: TargetManagerConfig{
				Type: "gitlab",
				General: General{
					Scheme:               "http",
					UnhealthyThreshold:   1 * time.Second,
					CheckInterval:        1 * time.Second,
					RegistrationInterval: 1 * time.Second,
					UpdateInterval:       1 * time.Second,
				},
			},
		},
		{
			name: "valid config - zero values",
			cfg: TargetManagerConfig{
				Type: "gitlab",
				General: General{
					Scheme:               "http",
					UnhealthyThreshold:   0,
					CheckInterval:        1 * time.Second,
					RegistrationInterval: 0,
					UpdateInterval:       0,
				},
			},
		},
		{
			name: "invalid config - zero check interval",
			cfg: TargetManagerConfig{
				Type: "gitlab",
				General: General{
					Scheme:               "http",
					UnhealthyThreshold:   1 * time.Second,
					CheckInterval:        0,
					RegistrationInterval: 1 * time.Second,
					UpdateInterval:       1 * time.Second,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid config - negative values",
			cfg: TargetManagerConfig{
				Type: "gitlab",
				General: General{
					Scheme:               "http",
					UnhealthyThreshold:   -1 * time.Second,
					CheckInterval:        1 * time.Second,
					RegistrationInterval: 1 * time.Second,
					UpdateInterval:       1 * time.Second,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid config - unknown interactor",
			cfg: TargetManagerConfig{
				Type: "unknown",
				General: General{
					Scheme:               "http",
					UnhealthyThreshold:   1 * time.Second,
					CheckInterval:        1 * time.Second,
					RegistrationInterval: 1 * time.Second,
					UpdateInterval:       1 * time.Second,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid config - wrong scheme",
			cfg: TargetManagerConfig{
				Type: "gitlab",
				General: General{
					Scheme:               "tcp",
					UnhealthyThreshold:   1 * time.Second,
					CheckInterval:        1 * time.Second,
					RegistrationInterval: 1 * time.Second,
					UpdateInterval:       1 * time.Second,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid config - no scheme",
			cfg: TargetManagerConfig{
				Type: "gitlab",
				General: General{
					UnhealthyThreshold:   1 * time.Second,
					CheckInterval:        1 * time.Second,
					RegistrationInterval: 1 * time.Second,
					UpdateInterval:       1 * time.Second,
				},
			},
			wantErr: true,
		},
		{
			name: "valid config - http",
			cfg: TargetManagerConfig{
				Type: "gitlab",
				General: General{
					Scheme:               "http",
					UnhealthyThreshold:   1 * time.Second,
					CheckInterval:        1 * time.Second,
					RegistrationInterval: 1 * time.Second,
					UpdateInterval:       1 * time.Second,
				},
			},
			wantErr: false,
		},
		{
			name: "valid config - https",
			cfg: TargetManagerConfig{
				Type: "gitlab",
				General: General{
					Scheme:               "https",
					UnhealthyThreshold:   1 * time.Second,
					CheckInterval:        1 * time.Second,
					RegistrationInterval: 1 * time.Second,
					UpdateInterval:       1 * time.Second,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.Validate(context.Background()); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
