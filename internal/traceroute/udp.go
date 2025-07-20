// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package traceroute

import (
	"context"
	"errors"
	"net"
	"syscall"
	"time"

	"github.com/telekom/sparrow/internal/logger"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sys/unix"
)

var (
	_ Client = (*udpClient)(nil)
	_ tracer = (*udpClient)(nil)
)

type udpClient struct {
	// dialUDP abstracts the creation of a UDP socket with TTL configured
	dialUDP func(ctx context.Context, addr net.Addr, ttl int, timeout time.Duration) (netConn, error)
}

// newUDPClient constructs a UDP-based traceroute client using run-as-non-root pattern.
func newUDPClient() *udpClient {
	return &udpClient{dialUDP: dialUDP}
}

// Run executes traceroute hops for each target over UDP, mirroring the generic Client interface.
// It spawns a hopper for each target, collects hop results up to opts.MaxTTL, and logs them.
func (c *udpClient) Run(ctx context.Context, targets []Target, opts *Options) (Result, error) {
	tracer := trace.SpanFromContext(ctx).TracerProvider().Tracer("traceroute.udpClient")
	ctx, sp := tracer.Start(ctx, "Run", trace.WithAttributes(
		attribute.Int("traceroute.targets.count", len(targets)),
		attribute.Int("traceroute.options.max_hops", opts.MaxTTL),
		attribute.Stringer("traceroute.options.timeout", opts.Timeout),
	))
	defer sp.End()

	res := make(Result, len(targets))
	for _, target := range targets {
		hops := make(chan Hop, opts.MaxTTL)
		target.hopChan = hops

		go func(t Target) {
			h := &hopper{
				client:     c,
				target:     &t,
				otelTracer: tracer,
				opts:       *opts,
			}
			h.run(ctx)
			h.wg.Wait()
			close(hops)
		}(target)

		results := collectResults(hops)
		res[target] = results
		logHops(ctx, results)
	}

	return res, nil
}

// trace performs a single UDP probe against the target at the given TTL and processes the kernel-generated ICMP response.
// We rely on the kernel to build and handle ICMP messages from TTL expiry or port unreachable, so no raw socket required.
func (c *udpClient) trace(ctx context.Context, target Target, opts Options) error {
	span := trace.SpanFromContext(ctx)
	log := logger.FromContext(ctx)
	log.DebugContext(ctx, "Starting UDP traceroute hop", "target", target)

	targetAddr, err := target.ToAddr()
	if err != nil {
		return wrapError(ctx, err, "failed to convert target to address")
	}

	nc, err := c.dialUDP(ctx, targetAddr, target.hopTTL, opts.Timeout)
	if err != nil {
		return wrapError(ctx, err, "failed to dial UDP connection")
	}
	defer func() { _ = nc.Close() }()

	listener, err := newErrQueueListener(nc.Conn, nc.port)
	if err != nil {
		return wrapError(ctx, err, "failed creating errQueueListener")
	}
	defer func() { _ = listener.Close() }()

	// We need to send a single byte to trigger the ICMP error response.
	if _, werr := nc.Write([]byte{0}); werr != nil {
		return wrapError(ctx, werr, "failed sending UDP probe")
	}

	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(opts.Timeout))
	defer cancel()
	start := time.Now()
	packet, err := listener.Read(ctx)
	// Order matters: First check for expected errors,
	// then handle unexpected errors.
	switch {
	// Timeout error: we didn't receive an ICMP message within
	// the specified timeout, which means the hop did not reach
	// the target or the ICMP error was not received.
	case errors.Is(err, context.DeadlineExceeded):
		hop := Hop{
			Latency: time.Since(start),
			Addr:    HopAddress{IP: "*"},
			TTL:     target.hopTTL,
			Reached: false,
		}
		log.DebugContext(ctx, "ICMP read timeout exceeded, no response received")
		span.AddEvent("ICMP read timeout exceeded", trace.WithAttributes(
			attribute.Bool("traceroute.target.reached", hop.Reached),
			attribute.Stringer("traceroute.target.hop", hop),
			attribute.String("traceroute.target.hop.error", err.Error()),
		))
		target.hopChan <- hop
		return nil

	// Unexpected error: we failed to read an ICMP message
	// and it's not because of the reasons above.
	case err != nil:
		return wrapError(ctx, err, "failed to read ICMP message")

	default:
		hop := Hop{
			Latency: time.Since(start),
			Addr:    newHopAddress(packet.remoteAddr),
			Name:    resolveName(packet.remoteAddr),
			TTL:     target.hopTTL,
			Reached: packet.reached,
		}
		log.DebugContext(ctx, "Received ICMP message", "port", packet.port, "routerAddr", packet.remoteAddr)
		span.AddEvent("ICMP message received", trace.WithAttributes(
			attribute.Bool("traceroute.target.reached", hop.Reached),
			attribute.Stringer("traceroute.target.hop", hop),
		))
		target.hopChan <- hop
		return nil
	}
}

// dialUDP sets up a UDP socket with the desired TTL and timeout parameters.
// We bind to a random local port so the kernel returns ICMP replies to this socket.
func dialUDP(ctx context.Context, addr net.Addr, ttl int, timeout time.Duration) (netConn, error) {
	port := randomPort()
	dialer := net.Dialer{
		LocalAddr: &net.UDPAddr{Port: port},
		Timeout:   timeout,
		ControlContext: func(_ context.Context, _, _ string, c syscall.RawConn) error {
			var opErr error
			if err := c.Control(func(fd uintptr) {
				opErr = errors.Join(
					unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_TTL, ttl), // #nosec G115
					unix.SetsockoptInt(int(fd), unix.SOL_IP, unix.IP_RECVERR, 1),   // #nosec G115
				)
			}); err != nil {
				return err
			}
			return opErr
		},
	}

	conn, err := dialer.DialContext(ctx, "udp", addr.String())
	if err != nil {
		return netConn{}, err
	}

	return netConn{Conn: conn, port: port}, nil
}
