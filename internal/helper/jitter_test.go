// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package helper

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestApplyJitter_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		factor   float64
		want     time.Duration
	}{
		{
			name:     "factor 0 returns exact duration",
			duration: 10 * time.Second,
			factor:   0,
			want:     10 * time.Second,
		},
		{
			name:     "zero duration",
			duration: 0,
			factor:   0.5,
			want:     0,
		},
		{
			name:     "negative factor treated as identity",
			duration: 10 * time.Second,
			factor:   -0.5,
			want:     10 * time.Second,
		},
		{
			name:     "NaN factor treated as identity",
			duration: 10 * time.Second,
			factor:   math.NaN(),
			want:     10 * time.Second,
		},
		{
			name:     "positive Inf factor treated as identity",
			duration: 10 * time.Second,
			factor:   math.Inf(1),
			want:     10 * time.Second,
		},
		{
			name:     "negative Inf factor treated as identity",
			duration: 10 * time.Second,
			factor:   math.Inf(-1),
			want:     10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplyJitter(tt.duration, tt.factor)
			assert.Equal(t, tt.want, got)
		})
	}
}

func FuzzApplyJitter(f *testing.F) {
	f.Add(int64(10*time.Second), 0.0)
	f.Add(int64(10*time.Second), 0.2)
	f.Add(int64(10*time.Second), 1.0)
	f.Add(int64(0), 0.5)
	f.Add(int64(10*time.Second), -0.5)
	f.Add(int64(10*time.Second), 2.0)
	f.Add(int64(time.Millisecond), 0.99)
	f.Add(int64(math.MaxInt64), 0.5)

	f.Fuzz(func(t *testing.T, durationNs int64, factor float64) {
		d := time.Duration(durationNs)
		got := ApplyJitter(d, factor)

		// Identity: when factor is non-positive, NaN, Inf, or d <= 0
		isIdentity := factor <= 0 || d <= 0 ||
			math.IsNaN(factor) || math.IsInf(factor, 0)
		if isIdentity {
			assert.Equal(t, d, got, "expected identity")
			return
		}

		// Clamp factor for bound assertions
		f := min(factor, 1.0)

		// Property: result never exceeds original
		assert.LessOrEqual(t, got, d, "above maximum")

		// Property: result respects bounded minimum
		minD := time.Duration(float64(d) * (1 - f))
		assert.GreaterOrEqual(t, got, minD, "below minimum")

		// Property: result is non-negative when d > 0
		assert.GreaterOrEqual(t, got, time.Duration(0), "negative result")
	})
}
