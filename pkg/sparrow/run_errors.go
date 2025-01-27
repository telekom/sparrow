// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package sparrow

// ErrShutdown holds any errors that may
// have occurred during shutdown of the Sparrow
type ErrShutdown struct {
	errAPI     error
	errTarMan  error
	errMetrics error
}

// HasError returns true if any of the errors are set
func (e ErrShutdown) HasError() bool {
	return e.errAPI != nil || e.errTarMan != nil || e.errMetrics != nil
}
