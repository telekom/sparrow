// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"fmt"
)

// ErrConfigMismatch is returned when a configuration is of the wrong type
type ErrConfigMismatch struct {
	Expected string
	Current  string
}

func (e ErrConfigMismatch) Error() string {
	return fmt.Sprintf("config mismatch: expected type %v, got %v", e.Expected, e.Current)
}

// ErrInvalidConfig is returned when a configuration is invalid
type ErrInvalidConfig struct {
	CheckName string
	Field     string
	Reason    string
}

func (e ErrInvalidConfig) Error() string {
	return fmt.Sprintf("invalid configuration field %q in check %q: %s", e.Field, e.CheckName, e.Reason)
}

// ErrMetricNotFound is returned when a metric is not found
type ErrMetricNotFound struct {
	Label string
}

func (e ErrMetricNotFound) Error() string {
	return fmt.Sprintf("metric %q not found", e.Label)
}
