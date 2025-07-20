// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package traceroute

import (
	"context"
	"net"
)

// icmpListener is an interface for reading ICMP messages.
//
//go:generate go tool moq -out icmp_moq.go . icmpListener
type icmpListener interface {
	Read(ctx context.Context) (icmpPacket, error)
	Close() error
}

// icmpPacket represents a received ICMP packet.
type icmpPacket struct {
	// remoteAddr is the address of the device (typically a router)
	// that sent the ICMP message in response to our traceroute probe.
	remoteAddr net.Addr
	// port is the parsed destination port from the TCP segment
	// contained in the ICMP message.
	port int
	// reached indicates whether the ICMP message indicates that the destination
	// was reached or not. This is true for ICMP messages of [ipv4.ICMPTypeDestinationUnreachable]
	// and [ipv6.ICMPTypeDestinationUnreachable].
	reached bool
}

// ICMP codes for Destination Unreachable messages.
// For more information, see:
// https://www.iana.org/assignments/icmp-parameters/icmp-parameters.xhtml#icmp-parameters-codes-3
const (
	// icmpUnreachableHost is the ICMP code for Destination Unreachable - "Host Unreachable" messages.
	icmpUnreachableHost = 1
	// icmpUnreachablePort is the ICMP code for Destination Unreachable - "Port Unreachable" messages.
	icmpUnreachablePort = 3
)
