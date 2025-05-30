// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package health

import (
	"testing"
	"time"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Targets:  []string{"http://localhost:8080"},
				Interval: 100 * time.Millisecond,
				Timeout:  1 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "invalid targets - invalid url",
			config: Config{
				Targets:  []string{"://localhost:8080"},
				Interval: 100 * time.Millisecond,
				Timeout:  1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "invalid targets - invalid scheme",
			config: Config{
				Targets:  []string{"localhost:8080"},
				Interval: 100 * time.Millisecond,
				Timeout:  1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "invalid interval",
			config: Config{
				Targets:  []string{"http://localhost:8080"},
				Interval: 10 * time.Millisecond,
				Timeout:  1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "invalid timeout",
			config: Config{
				Targets:  []string{"http://localhost:8080"},
				Interval: 100 * time.Millisecond,
				Timeout:  100 * time.Millisecond,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
