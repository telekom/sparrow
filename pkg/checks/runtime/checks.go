// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"iter"
	"slices"
	"sync"

	"github.com/telekom/sparrow/pkg/checks"
)

// Checks holds all the checks.
type Checks struct {
	mu     sync.RWMutex
	checks []checks.Check // = *checks.Check
}

// Add adds a new check.
func (c *Checks) Add(check checks.Check) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.checks = append(c.checks, check)
}

// Delete deletes a check.
func (c *Checks) Delete(check checks.Check) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, exist := range c.checks {
		if exist.Name() == check.Name() {
			c.checks = append(c.checks[:i], c.checks[i+1:]...)
			return
		}
	}
}

// Iter returns configured checks in an iterable format
func (c *Checks) Iter() iter.Seq[checks.Check] {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return slices.Values(slices.Clone(c.checks))
}
