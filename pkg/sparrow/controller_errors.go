// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package sparrow

import (
	"fmt"

	"github.com/telekom/sparrow/pkg/checks"
)

type ErrRunningCheck struct {
	Check checks.Check
	Err   error
}

func (e *ErrRunningCheck) Error() string {
	return fmt.Sprintf("check %s failed: %v", e.Check.Name(), e.Err)
}

type ErrCreateOpenapiSchema struct {
	name string
	err  error
}

func (e ErrCreateOpenapiSchema) Error() string {
	return fmt.Sprintf("failed to get schema for check %s: %v", e.name, e.err)
}
