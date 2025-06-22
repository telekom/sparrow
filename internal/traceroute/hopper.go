package traceroute

import (
	"context"
	"sync"

	"github.com/telekom/sparrow/internal/helper"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// tracer is an interface that defines the methods required for executing a traceroute.
//
//go:generate go tool moq -out tracer_moq.go . tracer
type tracer interface {
	// trace executes the traceroute for the given target with the specified options.
	trace(ctx context.Context, target Target, opts Options) error
}

// hopper is responsible for managing the execution of traceroute hops for a target.
type hopper struct {
	client     tracer
	wg         sync.WaitGroup
	otelTracer trace.Tracer
	target     *Target
	opts       Options
}

// run executes the traceroute hops for the target.
// It's the callers responsibility to collect the results
// from the hop channel of the target after calling this method.
func (h *hopper) run(ctx context.Context) {
	for ttl := 1; ttl <= h.opts.MaxTTL; ttl++ {
		h.wg.Add(1)
		go func() {
			defer h.wg.Done()
			ctx, hopSpan := h.otelTracer.Start(ctx, h.target.String(), trace.WithAttributes(
				attribute.Stringer("traceroute.target.address", h.target),
				attribute.Int("traceroute.target.ttl", ttl),
			))
			defer hopSpan.End()
			hopSpan.SetAttributes(
				attribute.Stringer("traceroute.target.address", h.target),
				attribute.Int("traceroute.target.ttl", ttl),
			)

			retry := helper.Retry(func(ctx context.Context) error {
				return h.client.trace(ctx, h.target.withHopTTL(ttl), h.opts)
			}, h.opts.Retry)

			if err := retry(ctx); err != nil {
				hopSpan.RecordError(err)
				hopSpan.SetStatus(codes.Error, "Failed to execute hop trace")
				hopSpan.End()
				return
			}
		}()
	}
}
