package traceroute

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandomPort(t *testing.T) {
	// randomPort should always return [basePort, basePort+portRange)
	for range 1000 {
		p := randomPort()
		assert.GreaterOrEqual(t, p, basePort, "randomPort should be >= basePort")
		assert.Less(t, p, basePort+portRange, "randomPort should be < basePort+portRange")
	}
}

func TestIPFromAddr(t *testing.T) {
	tests := []struct {
		name     string
		addr     net.Addr
		expected net.IP
	}{
		{"TCPAddr", &net.TCPAddr{IP: net.ParseIP("1.2.3.4"), Port: 80}, net.ParseIP("1.2.3.4")},
		{"UDPAddr", &net.UDPAddr{IP: net.ParseIP("5.6.7.8"), Port: 53}, net.ParseIP("5.6.7.8")},
		{"IPAddr", &net.IPAddr{IP: net.ParseIP("9.10.11.12")}, net.ParseIP("9.10.11.12")},
		{"UnixAddr (unsupported)", &net.UnixAddr{Name: "/tmp/x", Net: "unix"}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ipFromAddr(tt.addr)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestResolveName(t *testing.T) {
	tests := []struct {
		name string
		addr net.Addr
		want string
	}{
		{"nil Addr", nil, ""},
		{"unsupported Addr", &net.UnixAddr{Name: "/tmp/x", Net: "unix"}, ""},
		{"no reverse record", &net.IPAddr{IP: net.ParseIP("203.0.113.1")}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, resolveName(tt.addr))
		})
	}

	// And one "happy path" using loopback, which almost always maps to localhost
	t.Run("loopback resolves", func(t *testing.T) {
		loop := &net.IPAddr{IP: net.ParseIP("127.0.0.1")}
		name := resolveName(loop)
		// On most systems this will be "localhost." or similar
		assert.NotEmpty(t, name, "expected a non-empty name for 127.0.0.1")
		assert.Contains(t, name, "localhost", "expected substring 'localhost' in %q", name)
	})
}

func TestCollectResults(t *testing.T) {
	tests := []struct {
		name     string
		input    []Hop
		expected []Hop
	}{
		{
			name:     "empty channel",
			input:    []Hop{},
			expected: []Hop{},
		},
		{
			name:  "filters out TTL zero",
			input: []Hop{{TTL: 0}, {TTL: 2}, {TTL: 0}, {TTL: 1}},
			expected: []Hop{
				{TTL: 1},
				{TTL: 2},
			},
		},
		{
			name:  "sorts hops by TTL",
			input: []Hop{{TTL: 3}, {TTL: 1}, {TTL: 2}},
			expected: []Hop{
				{TTL: 1},
				{TTL: 2},
				{TTL: 3},
			},
		},
		{
			name:  "removes duplicate TTLs, keeping first occurrence",
			input: []Hop{{TTL: 1}, {TTL: 2}, {TTL: 1}, {TTL: 3}, {TTL: 2}},
			expected: []Hop{
				{TTL: 1},
				{TTL: 2},
				{TTL: 3},
			},
		},
		{
			name:  "combined filter, sort and dedupe",
			input: []Hop{{TTL: 0}, {TTL: 4}, {TTL: 2}, {TTL: 3}, {TTL: 2}, {TTL: 1}, {TTL: 0}},
			expected: []Hop{
				{TTL: 1},
				{TTL: 2},
				{TTL: 3},
				{TTL: 4},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := make(chan Hop, len(tt.input))
			for _, h := range tt.input {
				ch <- h
			}
			close(ch)

			got := collectResults(ch)
			assert.Equal(t, tt.expected, got, "collectResults(%v)", tt.input)
		})
	}
}
