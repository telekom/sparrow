// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package traceroute

import (
	"context"
	"errors"
	"net"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"golang.org/x/sys/unix"
)

func TestTCPClient_trace(t *testing.T) {
	tgt := Target{Protocol: ProtocolTCP, Address: "1.2.3.4", Port: 8080}
	tgt.hopTTL = 3

	tests := []struct {
		name        string
		dialErr     error
		icmpPacket  icmpPacket
		icmpErr     error
		wantErr     bool
		wantAddr    string
		wantReached bool
	}{
		{
			name:        "tcp success",
			dialErr:     nil,
			wantErr:     false,
			wantAddr:    "1.2.3.4",
			wantReached: true,
		},
		{
			name:    "dial record error short-circuit",
			dialErr: errors.New("network failure"),
			wantErr: true,
		},
		{
			name:        "ttl expired timeout",
			dialErr:     unix.EHOSTUNREACH,
			icmpErr:     context.DeadlineExceeded,
			wantErr:     false,
			wantAddr:    "*",
			wantReached: false,
		},
		{
			name:    "icmp not available",
			dialErr: unix.EHOSTUNREACH,
			icmpErr: errICMPNotAvailable,
			wantErr: true,
		},
		{
			name:        "intermediate router",
			dialErr:     unix.EHOSTUNREACH,
			icmpPacket:  icmpPacket{remoteAddr: newAddr(t, "9.8.7.6"), port: 8080},
			wantErr:     false,
			wantAddr:    "9.8.7.6",
			wantReached: false,
		},
		{
			name:    "icmp read error",
			dialErr: unix.EHOSTUNREACH,
			icmpErr: errors.New("icmp read error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &tcpClient{
				dialTCP: func(_ context.Context, addr net.Addr, _ int, _ time.Duration) (netConn, error) {
					require.Contains(t, addr.String(), ":8080")
					if tt.dialErr != nil {
						return netConn{}, tt.dialErr
					}
					return netConn{Conn: nil, port: 0}, nil
				},
				newICMPListener: func(_ int) (icmpListener, error) {
					return &icmpListenerMock{
						ReadFunc: func(_ context.Context) (icmpPacket, error) {
							return tt.icmpPacket, tt.icmpErr
						},
						CloseFunc: func() error { return nil },
					}, nil
				},
			}

			hops := make(chan Hop, 1)
			tgt.hopChan = hops
			opts := Options{MaxTTL: 3, Timeout: time.Millisecond}

			err := client.trace(t.Context(), tgt, opts)
			if tt.wantErr {
				require.Error(t, err)
				if tt.dialErr != nil || tt.icmpErr != nil {
					assert.True(t, errors.Is(err, tt.icmpErr) || errors.Is(err, tt.dialErr), "unexpected error: %v", err)
				}
				return
			}
			require.NoError(t, err)

			hop := <-hops
			assert.Equal(t, tt.wantReached, hop.Reached)
			assert.Contains(t, hop.Addr.String(), tt.wantAddr)
		})
	}
}

func TestTCPClient_Run(t *testing.T) {
	client := &tcpClient{
		dialTCP: func(_ context.Context, addr net.Addr, ttl int, timeout time.Duration) (netConn, error) {
			if ttl == 1 {
				t.Logf("Dialing %s with TTL %d and timeout %s", addr, ttl, timeout)
				return netConn{Conn: nil, port: 30000}, nil
			}
			t.Logf("Simulating unreachable host for %s with TTL %d", addr, ttl)
			return netConn{port: 30000}, syscall.EHOSTUNREACH
		},
		newICMPListener: func(port int) (icmpListener, error) {
			return &icmpListenerMock{
				ReadFunc: func(_ context.Context) (icmpPacket, error) {
					assert.Equal(t, 30000, port, "Expected ICMP read on port 30000")
					t.Log("Simulating ICMP read timeout")
					return icmpPacket{}, context.DeadlineExceeded
				},
				CloseFunc: func() error { return nil },
			}, nil
		},
	}

	tgt := Target{Protocol: ProtocolTCP, Address: "4.3.2.1", Port: 80}
	ctx, span := noop.NewTracerProvider().Tracer("").Start(t.Context(), "run")
	defer span.End()

	opts := &Options{MaxTTL: 3, Timeout: time.Millisecond}
	res, err := client.Run(ctx, []Target{tgt}, opts)
	require.NoError(t, err)

	// Gather only the successful (Reached=true) hops
	var reachedHops []Hop
	for _, hops := range res {
		for _, h := range hops {
			if h.Reached {
				reachedHops = append(reachedHops, h)
			}
		}
	}

	// since only TTL=1 succeeded, we expect exactly one reached hop
	require.Len(t, reachedHops, 1)
	require.True(t, reachedHops[0].Reached)
}

func newAddr(t testing.TB, ip string) net.Addr {
	t.Helper()
	addr := &net.TCPAddr{IP: net.ParseIP(ip)}
	require.NotNil(t, addr.IP, "failed to parse IP address: %s", ip)
	return addr
}
