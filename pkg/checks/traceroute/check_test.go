// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package traceroute

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"github.com/telekom/sparrow/internal/traceroute"
)

func TestCheck(t *testing.T) {
	cases := []struct {
		name string
		c    *Traceroute
		want result
	}{
		{
			name: "Success 5 hops",
			c:    newTraceroute(t, Config{Options: traceroute.Options{MaxTTL: 5, Timeout: 1 * time.Second}, Targets: []traceroute.Target{{Address: "8.8.8.8"}}}),
			want: result{
				"8.8.8.8": {
					{Addr: traceroute.HopAddress{IP: "0.0.0.1"}, Latency: 1 * time.Second, Reached: false, TTL: 1},
					{Addr: traceroute.HopAddress{IP: "0.0.0.2"}, Latency: 2 * time.Second, Reached: false, TTL: 2},
					{Addr: traceroute.HopAddress{IP: "0.0.0.3"}, Latency: 3 * time.Second, Reached: false, TTL: 3},
					{Addr: traceroute.HopAddress{IP: "0.0.0.4"}, Latency: 4 * time.Second, Reached: false, TTL: 4},
					{Addr: traceroute.HopAddress{IP: "123.0.0.123", Port: 53}, Name: "google-public-dns-a.google.com", Latency: 69 * time.Second, Reached: true, TTL: 5},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := c.c.check(t.Context())

			if !cmp.Equal(res, c.want) {
				diff := cmp.Diff(res, c.want)
				t.Errorf("unexpected result: +want -got\n%s", diff)
			}
		})
	}
}

func newTraceroute(t testing.TB, cfg Config) *Traceroute {
	t.Helper()
	c, ok := NewCheck().(*Traceroute)
	require.True(t, ok, "NewCheck should return a Traceroute check")
	c.config = cfg
	c.client = &traceroute.ClientMock{
		RunFunc: func(ctx context.Context, targets []traceroute.Target, opts *traceroute.Options) (traceroute.Result, error) {
			res := make(traceroute.Result, len(targets))
			for _, target := range targets {
				hops := make([]traceroute.Hop, opts.MaxTTL)
				for i := 0; i < opts.MaxTTL; i++ {
					hops[i] = traceroute.Hop{
						Addr:    traceroute.HopAddress{IP: net.IPv4(0, 0, 0, byte(i+1)).String()},
						Latency: time.Duration(i+1) * time.Second,
						TTL:     i + 1,
					}
				}

				if target.Address == "8.8.8.8" {
					hops[opts.MaxTTL-1] = traceroute.Hop{
						Addr:    traceroute.HopAddress{IP: net.IPv4(123, 0, 0, 123).String(), Port: 53},
						Name:    "google-public-dns-a.google.com",
						Latency: 69 * time.Second,
						Reached: true,
						TTL:     opts.MaxTTL,
					}
				}
				res[target] = hops
			}
			return res, nil
		},
	}
	return c
}
