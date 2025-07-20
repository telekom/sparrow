// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package traceroute

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/telekom/sparrow/internal/logger"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"golang.org/x/sys/unix"
)

// rawListener is a listener for ICMP messages over a raw socket.
// It requires NET_RAW capabilities to be created successfully.
type rawListener struct {
	// conn is the ICMP packet connection used to listen for ICMP messages.
	conn *icmp.PacketConn
	// recvPort is the port we are interested in receiving ICMP messages for.
	recvPort int
	// canICMP indicates whether the listener was successfully created
	// with NET_RAW capabilities, meaning it can read ICMP messages.
	canICMP bool
}

// newRawListener creates a new [rawListener] that listens for ICMP messages
// on the default IP address and port. If the listener cannot be created due to
// permission issues, it returns a listener that indicates ICMP is not available,
// but does not return an error.
func newRawListener(wantPort int) (icmpListener, error) {
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err == nil {
		return &rawListener{conn: conn, recvPort: wantPort, canICMP: true}, nil
	}

	if errors.Is(err, unix.EPERM) {
		return &rawListener{conn: nil, recvPort: wantPort, canICMP: false}, nil
	}

	return nil, fmt.Errorf("failed to create ICMP listener: %w", err)
}

// Read receives all ICMP messages on the listener's connection until
// it either receives a message on the specified port or the timeout is exceeded.
//
// Returns [errICMPNotAvailable] if the listener was created without NET_RAW capabilities,
// meaning ICMP is not available for reading.
func (l *rawListener) Read(ctx context.Context) (icmpPacket, error) {
	if !l.canICMP {
		return icmpPacket{}, errICMPNotAvailable
	}
	log := logger.FromContext(ctx)

	for {
		select {
		case <-ctx.Done():
			return icmpPacket{}, ctx.Err()
		default:
		}

		log.DebugContext(ctx, "Reading ICMP message")
		pkt, err := l.recvPacket(ctx)
		if err != nil {
			log.ErrorContext(ctx, "Failed to receive ICMP packet", "error", err)
			continue
		}

		if pkt.port != l.recvPort {
			log.DebugContext(ctx, "Received ICMP message on another port, ignoring",
				"expectedPort", l.recvPort,
				"receivedPort", pkt.port)
			continue
		}

		return *pkt, nil
	}
}

// recvPacket reads the next ICMP packet from the listener's connection.
func (l *rawListener) recvPacket(ctx context.Context) (*icmpPacket, error) {
	log := logger.FromContext(ctx)
	deadline, ok := ctx.Deadline()
	if !ok || deadline.IsZero() {
		log.DebugContext(ctx, "No deadline set for ICMP read")
		return nil, context.Canceled
	}

	if err := l.conn.SetReadDeadline(deadline); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	buf := make([]byte, mtuSize)
	n, src, err := l.conn.ReadFrom(buf)
	if err != nil {
		// This is most probably a timeout or a closed connection
		return nil, fmt.Errorf("failed to read from ICMP socket: %w", err)
	}

	msg, err := icmp.ParseMessage(ipv4.ICMPTypeTimeExceeded.Protocol(), buf[:n])
	if err != nil {
		return nil, fmt.Errorf("failed to parse ICMP message: %w", err)
	}

	packet, err := newICMPPacket(src, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to create ICMP packet from received message: %w", err)
	}
	log.DebugContext(ctx, "Received ICMP packet",
		"type", msg.Type,
		"routerAddr", packet.remoteAddr,
		"port", packet.port,
		"reached", packet.reached,
	)
	return packet, nil
}

// newICMPPacket creates a new ICMP packet from the given ICMP message and source address.
func newICMPPacket(src net.Addr, msg *icmp.Message) (*icmpPacket, error) {
	// Extract the TCP segment from the ICMP message.
	// The TCP segment comes after the IP header.
	var tcpSegment []byte
	switch msg.Type {
	case ipv4.ICMPTypeTimeExceeded:
		tcpSegment = msg.Body.(*icmp.TimeExceeded).Data[ipv4.HeaderLen:]
	case ipv4.ICMPTypeDestinationUnreachable:
		tcpSegment = msg.Body.(*icmp.DstUnreach).Data[ipv4.HeaderLen:]
	// Currently, we do not support IPv6 ICMP messages.
	// If we ever do, the header size is [ipv6.HeaderLen].
	case ipv6.ICMPTypeTimeExceeded:
		return nil, fmt.Errorf("ipv6 ICMP messages are not supported")
	case ipv6.ICMPTypeDestinationUnreachable:
		return nil, fmt.Errorf("ipv6 ICMP messages are not supported")
	default:
		return nil, fmt.Errorf("unexpected ICMP message type: %v", msg.Type)
	}

	// In the TCP segment, the first two bytes are the destination port.
	if len(tcpSegment) < 2 {
		return nil, fmt.Errorf("tcp segment too short: %d bytes", len(tcpSegment))
	}

	destPort := int(tcpSegment[0])<<8 + int(tcpSegment[1])
	unreachable := msg.Type == ipv4.ICMPTypeDestinationUnreachable || msg.Type == ipv6.ICMPTypeDestinationUnreachable

	return &icmpPacket{
		remoteAddr: src,
		port:       destPort,
		reached:    unreachable && msg.Code == icmpUnreachablePort,
	}, nil
}

// Close closes the ICMP listener connection.
//
// It is safe to call this method even if the listener was not successfully created
// or if it does not have NET_RAW capabilities.
func (l *rawListener) Close() error {
	if l.conn != nil {
		return l.conn.Close()
	}
	return nil
}
