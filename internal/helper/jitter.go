// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package helper

import (
	"math/rand/v2"
	"time"
)

// ApplyJitter applies full jitter with a bounded minimum to a duration.
// The returned duration is in the range [d*(1-factor), d].
// A factor of 0 returns d unchanged. Factor must be in [0.0, 1.0].
func ApplyJitter(d time.Duration, factor float64) time.Duration {
	if factor <= 0 || d <= 0 {
		return d
	}
	if factor > 1 {
		factor = 1
	}

	minDuration := float64(d) * (1 - factor)
	jitterRange := float64(d) * factor

	return time.Duration(minDuration + rand.Float64()*jitterRange) //nolint:gosec // jitter does not need crypto rand
}
