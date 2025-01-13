// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package sparrow

type ErrShutdown struct {
	errAPI     error
	errTarMan  error
	errMetrics error
}

func (e ErrShutdown) HasError() bool {
	return e.errAPI != nil || e.errTarMan != nil || e.errMetrics != nil
}
