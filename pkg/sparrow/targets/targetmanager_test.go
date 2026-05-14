// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package targets

import (
	"context"
	"testing"
	"time"

	"github.com/telekom/sparrow/pkg/sparrow/targets/remote/s3"
)

func TestTargetManagerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     TargetManagerConfig
		s3Cfg   *s3.Config
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
					Scheme:               schemeHTTP,
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
					Scheme:               schemeHTTP,
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
					Scheme:               schemeHTTP,
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
					Scheme:               schemeHTTP,
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
					Scheme:               schemeHTTP,
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
					Scheme:               schemeHTTP,
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
					Scheme:               schemeHTTPS,
					UnhealthyThreshold:   1 * time.Second,
					CheckInterval:        1 * time.Second,
					RegistrationInterval: 1 * time.Second,
					UpdateInterval:       1 * time.Second,
				},
			},
			wantErr: false,
		},
		{
			name: "valid config - s3",
			cfg: TargetManagerConfig{
				Type: "s3",
				General: General{
					Scheme:        schemeHTTPS,
					CheckInterval: 1 * time.Second,
				},
			},
			// S3 config populated inline via interactor.Config
			s3Cfg: &s3.Config{
				Endpoint: "s3.example.com",
				Bucket:   "test",
				Auth: s3.AuthConfig{
					Provider: "static",
					Static: s3.StaticAuthConfig{
						AccessKeyID:     "a",
						SecretAccessKey: "s",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid config - s3 missing bucket",
			cfg: TargetManagerConfig{
				Type: "s3",
				General: General{
					Scheme:        schemeHTTPS,
					CheckInterval: 1 * time.Second,
				},
			},
			s3Cfg: &s3.Config{
				Endpoint: "s3.example.com",
				Auth: s3.AuthConfig{
					Provider: "static",
					Static: s3.StaticAuthConfig{
						AccessKeyID:     "a",
						SecretAccessKey: "s",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "valid config - jitter 0.0",
			cfg: TargetManagerConfig{
				Type: "gitlab",
				General: General{
					Scheme:        schemeHTTPS,
					CheckInterval: 1 * time.Second,
					Jitter:        0.0,
				},
			},
			wantErr: false,
		},
		{
			name: "valid config - jitter 1.0",
			cfg: TargetManagerConfig{
				Type: "gitlab",
				General: General{
					Scheme:        schemeHTTPS,
					CheckInterval: 1 * time.Second,
					Jitter:        1.0,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid config - jitter negative",
			cfg: TargetManagerConfig{
				Type: "gitlab",
				General: General{
					Scheme:        schemeHTTPS,
					CheckInterval: 1 * time.Second,
					Jitter:        -0.1,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid config - jitter above 1",
			cfg: TargetManagerConfig{
				Type: "gitlab",
				General: General{
					Scheme:        schemeHTTPS,
					CheckInterval: 1 * time.Second,
					Jitter:        1.5,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.s3Cfg != nil {
				tt.cfg.S3 = *tt.s3Cfg
			}
			if err := tt.cfg.Validate(context.Background()); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
