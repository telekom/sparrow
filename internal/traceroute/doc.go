// Package traceroute provides a traceroute implementation that
// falls back to ICMP time-exceeded and destination-unreachable
// messages when setting TTL limits on outgoing probes.
//
// It exposes a [Client] for running traceroutes against one or
// more targets with configurable [Options].
// Under the hood it dials TCP connections with incrementing TTLs,
// listens for ICMP responses when TCP connections fail, and collects hop
// results in order, de-duplicating and stopping early when the destination is
// reached.
//
// Key features:
//   - Pure-Go TCP dialing with IP_TTL control via x/sys/unix (no external
//     traceroute binary required)
//   - Optional raw-socket ICMP listener with graceful fallback when NET_RAW
//     capabilities are unavailable
//   - Concurrency via goroutines and channels, with result collection, sorting,
//     and de-duplication
//   - Built-in OpenTelemetry spans and events for tracing each hop and errors
//   - Configurable retry policy, timeouts, and maximum hops via Options
//   - Fully mockable internals (icmpListener, tracer, Client) for unit testing
//
// Typical usage:
//
//	client := traceroute.NewClient()
//	opts   := &traceroute.Options{MaxTTL: 30, Timeout: 2*time.Second, Retry: retryCfg}
//	res, err := client.Run(ctx, []traceroute.Target{{Protocol: traceroute.ProtocolTCP, Address: "8.8.8.8", Port: 53}}, opts)
//	// res maps each Target to its slice of Hop results
//
// See each sub-package or type for more detailed documentation on exposed types
// and functions.
package traceroute
