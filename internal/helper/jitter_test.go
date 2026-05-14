// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package helper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestApplyJitter(t *testing.T) {
	tests := []struct {
		name      string
		duration  time.Duration
		factor    float64
		wantMin   time.Duration
		wantMax   time.Duration
		wantExact bool
	}{
		{
			name:      "factor 0 returns exact duration",
			duration:  10 * time.Second,
			factor:    0,
			wantExact: true,
		},
		{
			name:     "factor 0.2 bounds",
			duration: 10 * time.Second,
			factor:   0.2,
			wantMin:  8 * time.Second,
			wantMax:  10 * time.Second,
		},
		{
			name:     "factor 1.0 full range",
			duration: 10 * time.Second,
			factor:   1.0,
			wantMin:  0,
			wantMax:  10 * time.Second,
		},
		{
			name:      "zero duration",
			duration:  0,
			factor:    0.5,
			wantExact: true,
		},
		{
			name:      "negative factor treated as 0",
			duration:  10 * time.Second,
			factor:    -0.5,
			wantExact: true,
		},
		{
			name:     "factor above 1 clamped to 1",
			duration: 10 * time.Second,
			factor:   2.0,
			wantMin:  0,
			wantMax:  10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantExact {
				got := ApplyJitter(tt.duration, tt.factor)
				assert.Equal(t, tt.duration, got)
				return
			}

			// Run many iterations to verify bounds
			const iterations = 1000
			for range iterations {
				got := ApplyJitter(tt.duration, tt.factor)
				assert.GreaterOrEqual(t, got, tt.wantMin, "below minimum")
				assert.LessOrEqual(t, got, tt.wantMax, "above maximum")
			}
		})
	}
}
