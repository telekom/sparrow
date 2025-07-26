// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package traceroute

import (
	"context"
	"errors"
)

// errICMPNotAvailable is returned when ICMP is not available due to lack of NET_RAW capabilities.
// This typically occurs when the process does not have the necessary permissions to create an ICMP socket
// or when running in an environment where ICMP is restricted (e.g., some containerized environments).
var errICMPNotAvailable = errors.New("no NET_RAW capabilities, ICMP not available")

// isTracerouteError checks if the error is related to common
// and expected traceroute errors.
func isTracerouteError(err error) bool {
	return errors.Is(err, errICMPNotAvailable) ||
		errors.Is(err, context.DeadlineExceeded)
}
