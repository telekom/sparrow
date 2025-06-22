package traceroute

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsTracerouteError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"icmp not available", errICMPNotAvailable, true},
		{"wrapped icmp not available", fmt.Errorf("wrap: %w", errICMPNotAvailable), true},
		{"deadline exceeded", context.DeadlineExceeded, true},
		{"wrapped deadline exceeded", fmt.Errorf("ctx error: %w", context.DeadlineExceeded), true},
		{"some other error", errors.New("foo"), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTracerouteError(tt.err)
			assert.Equal(t, tt.want, got, "isTracerouteError(%v)", tt.err)
		})
	}
}
