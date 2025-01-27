// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package api

import "fmt"

type ErrCreateOpenapiSchema struct {
	name string
	err  error
}

func (e ErrCreateOpenapiSchema) Error() string {
	return fmt.Sprintf("failed to get schema for check %s: %v", e.name, e.err)
}
