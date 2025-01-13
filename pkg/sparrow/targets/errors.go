// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package targets

import "errors"

var (
	// ErrInvalidCheckInterval is returned when the check interval is invalid
	ErrInvalidCheckInterval = errors.New("invalid check interval")
	// ErrInvalidRegistrationInterval is returned when the registration interval is invalid
	ErrInvalidRegistrationInterval = errors.New("invalid registration interval")
	// ErrInvalidUnhealthyThreshold is returned when the unhealthy threshold is invalid
	ErrInvalidUnhealthyThreshold = errors.New("invalid unhealthy threshold")
	// ErrInvalidUpdateInterval is returned when the update interval is invalid
	ErrInvalidUpdateInterval = errors.New("invalid update interval")
	// ErrInvalidInteractorType is returned when the interactor type isn't recognized
	ErrInvalidInteractorType = errors.New("invalid interactor type")
	// ErrInvalidScheme is returned when the scheme is not http or https
	ErrInvalidScheme = errors.New("scheme must be 'http' of 'https'")
)
