// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package traceroute

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"syscall"

	"github.com/telekom/sparrow/internal/logger"
	"golang.org/x/net/ipv4"
	"golang.org/x/sys/unix"
)

// errQueueListener is a listener for ICMP messages via the UDP socket error queue.
// It requires the UDP socket to have IP_RECVERR enabled.
type errQueueListener struct {
	conn     net.Conn
	rawConn  syscall.RawConn
	recvPort int
	oobBuf   []byte
}

const (
	// icmpHeaderLengthMask is the mask to extract the ICMP header length
	// from the first byte of the data buffer.
	icmpHeaderLengthMask = 0x0F
	// byteMultiplier is used to convert the header length from 4-byte words to bytes.
	byteMultiplier = 4
	// oobBufSize is the size of the out-of-band buffer used for receiving extended error messages.
	oobBufSize = 512
	// dataBufSize is the size of the data buffer used for receiving messages.
	dataBufSize = 64
)

// newErrQueueListener wraps a UDP connection in an errQueueListener that
// reads ICMP errors from the kernel error queue for the given destination port.
func newErrQueueListener(conn net.Conn, wantPort int) (icmpListener, error) {
	sc, ok := conn.(syscall.Conn)
	if !ok {
		return nil, fmt.Errorf("the provided connection does not implement syscall.Conn: %T", conn)
	}

	rc, err := sc.SyscallConn()
	if err != nil {
		return nil, fmt.Errorf("failed to get RawConn: %w", err)
	}

	return &errQueueListener{
		conn:     conn,
		rawConn:  rc,
		recvPort: wantPort,
		oobBuf:   make([]byte, oobBufSize),
	}, nil
}

// Read waits until an ICMP error for the target port or timeout/context-cancel.
func (l *errQueueListener) Read(ctx context.Context) (icmpPacket, error) {
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
			if errors.Is(err, context.DeadlineExceeded) {
				return icmpPacket{}, context.DeadlineExceeded
			}
			log.ErrorContext(ctx, "Failed to receive ICMP packet", "error", err)
			continue
		}

		return *pkt, nil
	}
}

// recvPacket performs a single Recvmsg(..., MSG_ERRQUEUE) and parses one ICMP error.
func (l *errQueueListener) recvPacket(ctx context.Context) (*icmpPacket, error) {
	log := logger.FromContext(ctx)
	deadline, ok := ctx.Deadline()
	if !ok || deadline.IsZero() {
		log.DebugContext(ctx, "No deadline set for ICMP read")
		return nil, context.Canceled
	}

	if err := l.conn.SetReadDeadline(deadline); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	var pkt *icmpPacket
	var opErr error
	err := l.rawConn.Read(func(fd uintptr) bool {
		msg, rerr := recvMsg(fd, l.oobBuf, unix.MSG_ERRQUEUE)
		if rerr != nil {
			log.ErrorContext(ctx, "Failed to receive message", "error", rerr)
			opErr = rerr
			return true
		}

		pkt, opErr = parseExtendedErr(ctx, msg)
		return true
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read from raw connection: %w", err)
	}

	if opErr == nil {
		return pkt, nil
	}

	if errors.Is(opErr, unix.EAGAIN) || errors.Is(opErr, unix.EWOULDBLOCK) {
		log.DebugContext(ctx, "No ICMP error received, socket error queue is empty")
		return nil, context.DeadlineExceeded
	}

	return nil, fmt.Errorf("failed to read ICMP error: %w", opErr)
}

// Close closes the underlying [net.Conn].
func (l *errQueueListener) Close() error {
	return l.conn.Close()
}

// socketMsg represents a message received from the socket.
type socketMsg struct {
	// from is the source address of the message.
	from net.Addr
	// port is the destination port of the message.
	// This is the local port number we sent the probe from.
	port int
	// oob is the out-of-band data received with the message.
	// This contains the extended error information from the kernel.
	oob []byte
}

// unixRecvMsg is a wrapper around the [unix.Recvmsg] function.
// It allows us to mock the function in tests.
var unixRecvMsg = unix.Recvmsg

// recvMsg receives a message from the socket and extracts the source address and destination port.
var recvMsg = func(fd uintptr, oob []byte, flags int) (*socketMsg, error) {
	dataBuf := make([]byte, dataBufSize)
	n, oobn, _, from, err := unixRecvMsg(int(fd), dataBuf, oob, flags)
	if err != nil {
		return nil, fmt.Errorf("failed to receive message: %w", err)
	}

	// Extract the IP header length from the first byte of the data buffer.
	headerLen := int(dataBuf[0]&icmpHeaderLengthMask) * byteMultiplier
	if n < headerLen+4 {
		return nil, errors.New("received packet too small for UDP header")
	}

	dstPort := int(binary.BigEndian.Uint16(dataBuf[headerLen+2 : headerLen+4]))
	return &socketMsg{
		from: addrFromSocket(from),
		port: dstPort,
		oob:  oob[:oobn],
	}, nil
}

// parseExtendedErr decodes SOL_IP / IP_RECVERR control messages for both TimeExceeded and DestinationUnreachable.
var parseExtendedErr = func(ctx context.Context, msg *socketMsg) (*icmpPacket, error) {
	log := logger.FromContext(ctx)
	cms, err := unix.ParseSocketControlMessage(msg.oob)
	if err != nil {
		return nil, fmt.Errorf("failed to parse control messages: %w", err)
	}

	for _, cm := range cms {
		if cm.Header.Level != unix.SOL_IP || cm.Header.Type != unix.IP_RECVERR {
			continue
		}

		ee, err := newSockExtendedErr(cm.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode extended error: %w", err)
		}

		timeExceeded := ee.Type == uint8(ipv4.ICMPTypeTimeExceeded)
		destUnreachable := ee.Type == uint8(ipv4.ICMPTypeDestinationUnreachable)
		if !timeExceeded && !destUnreachable {
			log.DebugContext(ctx, "Received unexpected ICMP type", "extendedErr", fmt.Sprintf("%+v", ee))
			return nil, fmt.Errorf("unexpected ICMP type %d with code %d", ee.Type, ee.Code)
		}

		return &icmpPacket{
			remoteAddr: msg.from,
			port:       msg.port,
			reached:    destUnreachable && ee.Code == icmpUnreachablePort,
		}, nil
	}

	return nil, errors.New("no SOL_IP/IP_RECVERR message found")
}

// minExtendedErrSize is the minimum size of the extended error structure
// as defined in the Linux kernel documentation:
// https://man7.org/linux/man-pages/man7/ip.7.html
const minExtendedErrSize = 16

// newSockExtendedErr converts the first 16 bytes of an OOB buffer into a [unix.SockExtendedErr].
func newSockExtendedErr(data []byte) (unix.SockExtendedErr, error) {
	if len(data) < minExtendedErrSize {
		return unix.SockExtendedErr{}, fmt.Errorf("extended error too short: %d bytes", len(data))
	}

	return unix.SockExtendedErr{
		Errno:  binary.LittleEndian.Uint32(data[0:4]),
		Origin: data[4],
		Type:   data[5],
		Code:   data[6],
		Info:   binary.LittleEndian.Uint32(data[8:12]),
		Data:   binary.LittleEndian.Uint32(data[12:16]),
	}, nil
}
