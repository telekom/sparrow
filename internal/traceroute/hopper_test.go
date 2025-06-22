package traceroute

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/telekom/sparrow/internal/helper"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestHopper_run(t *testing.T) {
	tests := []struct {
		name      string
		maxTTL    int
		wantCalls int
	}{
		{"zero hops", 0, 0},
		{"one hop", 1, 1},
		{"three hops", 3, 3},
		{"five hops", 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := make(chan Target, tt.maxTTL)
			mock := &tracerMock{
				traceFunc: func(_ context.Context, tgt Target, _ Options) error {
					ch <- tgt
					return nil
				},
			}

			h := &hopper{
				target:     &Target{Address: "127.0.0.1", Port: 1234},
				client:     mock,
				otelTracer: noop.NewTracerProvider().Tracer("test"),
				opts: Options{
					MaxTTL:  tt.maxTTL,
					Retry:   helper.RetryConfig{Count: 1, Delay: time.Millisecond},
					Timeout: 2 * time.Millisecond,
				},
			}

			h.run(t.Context())
			h.wg.Wait()
			close(ch)

			var got []int
			for tgt := range ch {
				got = append(got, tgt.hopTTL)
			}

			want := make([]int, tt.maxTTL)
			for i := range want {
				want[i] = i + 1
			}

			assert.ElementsMatch(t, want, got,
				"expected tracer to be called once for each ttl 1..%d, got %v", tt.maxTTL, got)
		})
	}
}

func TestHopper_run_retry(t *testing.T) {
	var (
		mu          sync.Mutex
		invocations []int
		callCount   int
	)

	mock := &tracerMock{
		traceFunc: func(_ context.Context, _ Target, _ Options) error {
			mu.Lock()
			defer mu.Unlock()
			callCount++
			invocations = append(invocations, callCount)
			if callCount < 3 {
				return fmt.Errorf("transient error %d", callCount)
			}
			return nil
		},
	}

	h := &hopper{
		target:     &Target{Address: "127.0.0.1", Port: 1234},
		client:     mock,
		otelTracer: noop.NewTracerProvider().Tracer("test"),
		opts: Options{
			MaxTTL:  1,
			Retry:   helper.RetryConfig{Count: 2, Delay: 0},
			Timeout: time.Millisecond,
		},
	}

	h.run(t.Context())
	h.wg.Wait()

	require.Len(t, invocations, 3, "expected 3 total attempts, got %d", len(invocations))
	assert.Equal(t, []int{1, 2, 3}, invocations)
}
