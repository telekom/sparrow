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
	_ Client = (*tcpClient)(nil)
	_ tracer = (*tcpClient)(nil)
)

type tcpClient struct {
	dialTCP         func(ctx context.Context, addr net.Addr, ttl int, timeout time.Duration) (tcpConn, error)
	newICMPListener func() (icmpListener, error)
}

// newTCPClient creates a new TCP client for performing traceroutes.
func newTCPClient() *tcpClient {
	return &tcpClient{
		dialTCP:         dialTCP,
		newICMPListener: newICMPListener,
	}
}

// Run executes the traceroute for the given targets using TCP.
// It returns a Result containing the hops for each target, or an error if the traceroute fails.
func (c *tcpClient) Run(ctx context.Context, targets []Target, opts *Options) (Result, error) {
	tracer := trace.SpanFromContext(ctx).TracerProvider().Tracer("traceroute.tcpClient")
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
				target:     &t,
				client:     c,
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

func (c *tcpClient) trace(ctx context.Context, target Target, opts Options) error {
	span := trace.SpanFromContext(ctx)
	log := logger.FromContext(ctx)
	log.DebugContext(ctx, "Starting TCP trace", "target", target)

	targetAddr, err := target.ToAddr()
	if err != nil {
		return wrapError(ctx, err, "failed to convert target to address")
	}

	il, err := c.newICMPListener()
	if err != nil {
		return wrapError(ctx, err, "failed to create ICMP listener")
	}
	defer func() { _ = il.Close() }()

	start := time.Now()
	conn, err := c.dialTCP(ctx, targetAddr, target.hopTTL, opts.Timeout)
	defer func() { _ = conn.Close() }()

	// Happiest path: we successfully established a TCP connection
	// to the target, which means we reached the destination and
	// the traceroute is complete with this hop.
	if err == nil {
		hop := Hop{
			Latency: time.Since(start),
			Addr:    newHopAddress(targetAddr),
			Name:    resolveName(targetAddr),
			TTL:     target.hopTTL,
			Reached: true,
		}
		log.DebugContext(ctx, "TCP connection established", "port", conn.port, "addr", targetAddr)
		span.AddEvent("TCP connection established", trace.WithAttributes(
			attribute.Stringer("traceroute.target.hop", hop),
			attribute.Bool("traceroute.target.reached", hop.Reached),
		))

		target.hopChan <- hop
		return nil
	}

	// Unexpected error: we failed to establish a TCP connection
	// due to an error other than [unix.EHOSTUNREACH], which
	// indicates that our TTL is too low to reach the target
	// and is expected behavior for traceroute.
	if rErr := recordTCPError(ctx, err); rErr != nil {
		return rErr
	}

	packet, err := il.Read(ctx, conn.port, opts.Timeout)
	switch {
	// Unexpected error: we failed to read an ICMP message
	// and it's not because of capabilities/exceeded timeout.
	case err != nil && !isTracerouteError(err):
		return wrapError(ctx, err, "failed to read ICMP message")

	// User error: we don't have the necessary capabilities
	// to open a raw socket for reading ICMP messages.
	case errors.Is(err, errICMPNotAvailable):
		return wrapError(ctx, err, "ICMP not available for reading")

	// Timeout error: we didn't receive an ICMP message within
	// the specified timeout, which is expected when routers
	// do not respond to our traceroute probes.
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

	// Expected ICMP message received: we received an ICMP message
	// indicating that the TTL has expired, which is the expected behavior
	// of traceroute.
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

// tcpConn represents a TCP connection with a specific port.
type tcpConn struct {
	conn net.Conn
	port int
}

// dialTCP dials a TCP connection to the given address with the specified TTL.
func dialTCP(ctx context.Context, addr net.Addr, ttl int, timeout time.Duration) (tcpConn, error) {
	port := randomPort()

	// Dialer with control function to set IP_TTL
	dialer := net.Dialer{
		LocalAddr: &net.TCPAddr{
			Port: port,
		},
		Timeout: timeout,
		ControlContext: func(_ context.Context, _, _ string, c syscall.RawConn) error {
			var opErr error
			if err := c.Control(func(fd uintptr) {
				opErr = unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_TTL, ttl) // #nosec G115 // The net package is safe to use
			}); err != nil {
				return err
			}
			return opErr
		},
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr.String())
	switch {
	case err == nil:
		return tcpConn{conn: conn, port: port}, nil
	case errors.Is(err, unix.EADDRINUSE):
		// If the address is already in use,
		// we just retry with a new random port.
		return dialTCP(ctx, addr, ttl, timeout)
	default:
		return tcpConn{conn: conn, port: port}, err
	}
}

// Close closes the TCP connection.
func (tc *tcpConn) Close() error {
	if tc.conn != nil {
		return tc.conn.Close()
	}
	return nil
}
