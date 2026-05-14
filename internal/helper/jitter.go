// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package helper

import (
	"math"
	"math/rand/v2"
	"time"
)

// ApplyJitter adds a random jitter to d. factor is a percentage of d
// that will be subtracted from d, so the returned duration is in the
// range [d*(1-factor), d]. A factor of 0 returns d unchanged.
// factor must be in [0.0, 1.0]; values above 1.0 are clamped to 1.0.
func ApplyJitter(d time.Duration, factor float64) time.Duration {
	if math.IsNaN(factor) || math.IsInf(factor, 0) || factor <= 0 || d <= 0 {
		return d
	}
	if factor > 1 {
		factor = 1
	}

	minDuration := float64(d) * (1 - factor)
	jitterRange := float64(d) * factor

	//gosec:disable G404 -- jitter does not need crypto rand
	return time.Duration(minDuration + rand.Float64()*jitterRange)
}
