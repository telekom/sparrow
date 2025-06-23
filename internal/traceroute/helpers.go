// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package traceroute

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"slices"

	"github.com/telekom/sparrow/internal/logger"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sys/unix"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	// basePort is the starting port for the TCP connection
	basePort = 30000
	// portRange is the range of ports to generate a random port from
	portRange = 10000
)

// randomPort returns a random port in the interval [30000, 40000)
func randomPort() int {
	return rand.N(portRange) + basePort // #nosec G404 // math.rand is fine here, we're not doing encryption
}

// resolveName performs a reverse DNS lookup for the given IP address.
// If the lookup fails or returns no names, it returns an empty string.
func resolveName(addr net.Addr) string {
	if addr == nil {
		return ""
	}

	ip := ipFromAddr(addr)
	if ip == nil {
		return ""
	}

	names, err := net.LookupAddr(ip.String())
	if err != nil || len(names) == 0 {
		return ""
	}
	return names[0]
}

// ipFromAddr extracts the IP address from a [net.Addr].
func ipFromAddr(addr net.Addr) net.IP {
	switch a := addr.(type) {
	case *net.UDPAddr:
		return a.IP
	case *net.TCPAddr:
		return a.IP
	case *net.IPAddr:
		return a.IP
	}
	return nil
}

// collectResults collects the results from the channel and returns a sorted slice of hops.
// It filters out hops with a TTL of 0 and removes duplicates, keeping only the first occurrence of each TTL.
// The hops are sorted by TTL in ascending order.
func collectResults(ch <-chan Hop) []Hop {
	hops := []Hop{}
	for hop := range ch {
		if hop.TTL == 0 {
			continue
		}
		hops = append(hops, hop)
	}

	if len(hops) == 0 {
		return hops
	}

	slices.SortFunc(hops, func(a, b Hop) int {
		return a.TTL - b.TTL
	})

	filtered := make([]Hop, 0, len(hops))
	seen := make(map[int]bool)
	for _, hop := range hops {
		if !seen[hop.TTL] {
			filtered = append(filtered, hop)
			seen[hop.TTL] = true
			if hop.Reached {
				// If we reached the target, we can stop collecting hops.
				break
			}
		}
	}

	return filtered
}

// logHops logs the hops in a structured format.
func logHops(ctx context.Context, hops []Hop) {
	log := logger.FromContext(ctx)
	for _, hop := range hops {
		log.DebugContext(ctx, hop.String())
	}
}

// wrapError wraps an error with a message and logs it.
// It also records the error in the current OpenTelemetry span.
func wrapError(ctx context.Context, err error, msg string, args ...any) error {
	if err == nil {
		return nil
	}
	log := logger.FromContext(ctx)
	span := trace.SpanFromContext(ctx)
	caser := cases.Title(language.English)

	log.ErrorContext(ctx, caser.String(msg), append([]any{"error", err}, args...)...)
	span.SetStatus(codes.Error, fmt.Sprintf(msg+": %v", args...))
	span.RecordError(err)
	return fmt.Errorf("%s: %w", fmt.Sprintf(msg, args...), err)
}

// recordTCPError records the error from dialing a TCP connection.
// If the error is nil or [unix.EHOSTUNREACH], it returns nil.
func recordTCPError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	log := logger.FromContext(ctx)
	span := trace.SpanFromContext(ctx)

	// No route to host is a special error because of how traceroute works.
	// We are expecting the connection to fail because of TTL expiry.
	span.RecordError(err)
	if !errors.Is(err, unix.EHOSTUNREACH) {
		log.ErrorContext(ctx, "Failed to dial TCP connection", "error", err)
		span.AddEvent("TCP connection failed", trace.WithAttributes(
			attribute.String("traceroute.target.error", err.Error()),
		))
		span.SetStatus(codes.Error, "Failed to dial TCP connection")
		return fmt.Errorf("failed to dial TCP connection: %w", err)
	}

	span.SetStatus(codes.Error, "No route to host")
	return nil
}
