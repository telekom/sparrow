// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package helper

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/caas-team/sparrow/internal/logger"
)

type RetryConfig struct {
	Count int           `yaml:"count"`
	Delay time.Duration `yaml:"delay"`
}

// Effector will be the function called by the Retry function
type Effector func(context.Context) error

// Retry will retry the run the effector function in an exponential backoff
func Retry(effector Effector, rc RetryConfig) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		log := logger.FromContext(ctx)
		for r := 1; ; r++ {
			err := effector(ctx)
			if err == nil || r > rc.Count {
				return err
			}

			delay := getExpBackoff(rc.Delay, r)
			log.WarnContext(ctx, fmt.Sprintf("Effector call failed, retrying in %v", delay))

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}
}

// calculate the exponential delay for a given iteration
// first iteration is 1
func getExpBackoff(initialDelay time.Duration, iteration int) time.Duration {
	if iteration <= 1 {
		return initialDelay
	}
	return time.Duration(math.Pow(2, float64(iteration-1))) * initialDelay
}
