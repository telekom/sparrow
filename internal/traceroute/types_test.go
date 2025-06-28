// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package traceroute

import (
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHopAddress_String(t *testing.T) {
	type fields struct {
		IP   string
		Port int
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{name: "No Port", fields: fields{IP: "100.1.1.7"}, want: "100.1.1.7"},
		{name: "With Port", fields: fields{IP: "100.1.1.7", Port: 80}, want: "100.1.1.7:80"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := HopAddress{
				IP:   tt.fields.IP,
				Port: tt.fields.Port,
			}
			if got := a.String(); got != tt.want {
				t.Errorf("HopAddress.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newHopAddress(t *testing.T) {
	type args struct {
		addr net.Addr
	}
	tests := []struct {
		name string
		args args
		want HopAddress
	}{
		{
			name: "Works with TCP",
			args: args{
				addr: &net.TCPAddr{IP: net.ParseIP("100.1.1.7"), Port: 80},
			},
			want: HopAddress{
				IP:   "100.1.1.7",
				Port: 80,
			},
		},
		{
			name: "Works with UDP",
			args: args{
				addr: &net.UDPAddr{IP: net.ParseIP("100.1.1.7"), Port: 80},
			},
			want: HopAddress{
				IP:   "100.1.1.7",
				Port: 80,
			},
		},
		{
			name: "Works with IP",
			args: args{
				addr: &net.IPAddr{IP: net.ParseIP("100.1.1.7")},
			},
			want: HopAddress{
				IP: "100.1.1.7",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newHopAddress(tt.args.addr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newHopAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHop_String(t *testing.T) {
	tests := []struct {
		name     string
		hop      Hop
		expected string
	}{
		{
			name: "Resolved host, reached",
			hop: Hop{
				TTL:     1,
				Addr:    newTestAddress(t, "192.168.0.1"),
				Name:    "router.local",
				Latency: 12 * time.Millisecond,
				Reached: true,
			},
			expected: "1   router.local",
		},
		{
			name: "Unresolved host, not reached",
			hop: Hop{
				TTL:     2,
				Addr:    newTestAddress(t, "10.0.0.1"),
				Name:    "",
				Latency: 25 * time.Millisecond,
				Reached: false,
			},
			expected: "2   10.0.0.1",
		},
		{
			name: "Long hostname gets truncated",
			hop: Hop{
				TTL:     3,
				Addr:    newTestAddress(t, "1.2.3.4"),
				Name:    "254-254-254-254.very.long.name.example.telekom.com",
				Latency: 123456 * time.Microsecond,
				Reached: true,
			},
			expected: "3   1.2.3.4",
		},
		{
			name: "Exactly max length hostname (45 chars)",
			hop: Hop{
				TTL:     4,
				Addr:    newTestAddress(t, "4.4.4.4"),
				Name:    "host.exactly.forty.five.chars.telekom.net",
				Latency: 3 * time.Millisecond,
				Reached: true,
			},
			expected: "4   host.exactly.forty.five.chars.telekom.net",
		},
		{
			name: "Short hostname, low TTL",
			hop: Hop{
				TTL:     5,
				Addr:    newTestAddress(t, "5.5.5.5"),
				Name:    "r",
				Latency: 300 * time.Microsecond,
				Reached: false,
			},
			expected: "5   r",
		},
		{
			name: "High TTL and zero latency",
			hop: Hop{
				TTL:     30,
				Addr:    newTestAddress(t, "8.8.8.8"),
				Name:    "",
				Latency: 0,
				Reached: true,
			},
			expected: "30  8.8.8.8",
		},
		{
			name: "Very high TTL (3-digit)",
			hop: Hop{
				TTL:     123,
				Addr:    newTestAddress(t, "9.9.9.9"),
				Name:    "gateway",
				Latency: 78 * time.Millisecond,
				Reached: true,
			},
			expected: "123  gateway",
		},
		{
			name: "TTL zero edge case",
			hop: Hop{
				TTL:     0,
				Addr:    newTestAddress(t, "0.0.0.0"),
				Name:    "unknown",
				Latency: 5 * time.Millisecond,
				Reached: false,
			},
			expected: "0   unknown",
		},
		{
			name: "Address is * string",
			hop: Hop{
				TTL:     7,
				Addr:    HopAddress{IP: "*"},
				Name:    "",
				Latency: 1 * time.Millisecond,
				Reached: false,
			},
			expected: "7   *",
		},
		{
			name: "Hostname is * placeholder",
			hop: Hop{
				TTL:     8,
				Addr:    newTestAddress(t, "203.0.113.42"),
				Name:    "*",
				Latency: 2 * time.Millisecond,
				Reached: true,
			},
			expected: "8   *",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := tt.hop.String()
			assert.Equal(t, out[:len(tt.expected)], tt.expected, "Hop string should contain expected address and name")
			assert.Contains(t, out, tt.hop.Latency.String(), "Hop string should contain latency")
			if tt.hop.Reached {
				assert.Contains(t, out, "(reached)", "Hop string should indicate it was reached")
			} else {
				assert.NotContains(t, out, "(reached)", "Hop string should not indicate it was reached")
			}
		})
	}
}

func newTestAddress(t testing.TB, s string) HopAddress {
	t.Helper()
	ip, port, err := net.SplitHostPort(s)
	if err != nil {
		ip = s // if no port is provided, use the whole string as IP
	}

	if port != "" {
		return HopAddress{IP: ip, Port: 0}
	}

	return HopAddress{IP: ip, Port: 0}
}
