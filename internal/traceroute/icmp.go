package traceroute

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/telekom/sparrow/internal/logger"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"golang.org/x/sys/unix"
)

// icmpListener is an interface for reading ICMP messages.
//
//go:generate go tool moq -out icmp_moq.go . icmpListener
type icmpListener interface {
	Read(ctx context.Context, wantPort int, timeout time.Duration) (icmpPacket, error)
	Close() error
}

// icmpPacketListener is a listener for ICMP messages.
type icmpPacketListener struct {
	// conn is the ICMP packet connection used to listen for ICMP messages.
	conn *icmp.PacketConn
	// canICMP indicates whether the listener was successfully created
	// with NET_RAW capabilities, meaning it can read ICMP messages.
	canICMP bool
}

// newICMPListener creates a new ICMP listener on the default IP address and port.
// If the listener cannot be created due to permission issues, it returns a listener
// that indicates ICMP is not available, but does not return an error.
func newICMPListener() (icmpListener, error) {
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err == nil {
		return &icmpPacketListener{conn: conn, canICMP: true}, nil
	}

	if errors.Is(err, unix.EPERM) {
		return &icmpPacketListener{conn: nil, canICMP: false}, nil
	}

	return nil, fmt.Errorf("failed to create ICMP listener: %w", err)
}

// Read receives all ICMP messages on the listener's connection until
// it either receives a message on the specified port or the timeout is exceeded.
//
// Returns [errICMPNotAvailable] if the listener was created without NET_RAW capabilities,
// meaning ICMP is not available for reading.
func (il *icmpPacketListener) Read(ctx context.Context, recvPort int, timeout time.Duration) (icmpPacket, error) {
	if !il.canICMP {
		return icmpPacket{}, errICMPNotAvailable
	}
	log := logger.FromContext(ctx)
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		log.DebugContext(ctx, "Reading ICMP message")
		packet, err := il.recvPacket(ctx, timeout)
		if err != nil {
			log.ErrorContext(ctx, "Failed to receive ICMP packet", "error", err)
			continue
		}

		if packet.port != recvPort {
			log.DebugContext(ctx, "Received ICMP message on another port, ignoring",
				"expectedPort", recvPort,
				"receivedPort", packet.port)
			continue
		}

		return *packet, nil
	}

	log.DebugContext(ctx, "ICMP read timeout exceeded")
	return icmpPacket{}, context.DeadlineExceeded
}

// recvPacket reads the next ICMP packet from the listener's connection.
func (il *icmpPacketListener) recvPacket(ctx context.Context, timeout time.Duration) (*icmpPacket, error) {
	log := logger.FromContext(ctx)
	if err := il.conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	// mtuSize is the maximum transmission unit size
	const mtuSize = 1500
	buf := make([]byte, mtuSize)
	n, src, err := il.conn.ReadFrom(buf)
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

// codePortUnreachable is the ICMP code for Destination Unreachable - "Port Unreachable" messages.
// For more information, see:
// https://www.iana.org/assignments/icmp-parameters/icmp-parameters.xhtml#icmp-parameters-codes-3
const codePortUnreachable = 3

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
		reached:    unreachable && msg.Code == codePortUnreachable,
	}, nil
}

// Close closes the ICMP listener connection.
//
// It is safe to call this method even if the listener was not successfully created
// or if it does not have NET_RAW capabilities.
func (il *icmpPacketListener) Close() error {
	if il.conn != nil {
		return il.conn.Close()
	}
	return nil
}
