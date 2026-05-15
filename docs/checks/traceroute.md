<!--
SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH

SPDX-License-Identifier: CC-BY-4.0
-->

# Traceroute check

The traceroute check traces the network path to targets,
reporting hop-by-hop latency and reachability.

## Configuration

| Field            | Type              | Description                    |
| ---------------- | ----------------- | ------------------------------ |
| `interval`       | `duration`        | Interval between traces.       |
| `timeout`        | `duration`        | Timeout per hop.               |
| `retry.count`    | `integer`         | Number of retries on failure.  |
| `retry.delay`    | `duration`        | Initial delay between retries. |
| `maxHops`        | `integer`         | Maximum hops before giving up. |
| `targets`        | `list of objects` | Targets to trace.              |
| `targets[].addr` | `string`          | IP address or DNS name.        |
| `targets[].port` | `uint16`          | Target port (default: 80).     |

### Example

```yaml
traceroute:
  interval: 5s
  timeout: 3s
  retry:
    count: 3
    delay: 1s
  maxHops: 30
  targets:
    - addr: 8.8.8.8
      port: 53
    - addr: www.google.com
      port: 80
```

## Optional Capabilities

Sparrow does not need extra permissions for basic traceroute.
However, some data (e.g. the IP of a hop that dropped a
packet) requires raw socket access:

- Run as root:

  ```bash
  sudo sparrow run --config config.yaml
  ```

- Or grant `CAP_NET_RAW`:

  ```bash
  sudo setcap 'cap_net_raw=ep' sparrow
  ```

## Metrics and Observability

The traceroute check exposes data through three channels,
each serving a different use case:

| Channel         | What it carries                                         | Best for                                          |
| --------------- | ------------------------------------------------------- | ------------------------------------------------- |
| **Prometheus**  | Aggregate gauges (total duration, min hops)             | Alerting, dashboard summaries                     |
| **REST API**    | Full hop-by-hop JSON breakdown                          | Ad-hoc debugging, integration with custom tooling |
| **OTLP traces** | Per-hop spans with latency, TTL, and address attributes | Distributed tracing backends, deep path analysis  |

Prometheus uses a pull model — the collector scrapes
`/metrics` periodically. This works well for scalar
summaries but cannot efficiently represent variable-depth,
structured data like a full traceroute path. The traceroute
check therefore pushes detailed per-hop data as OTLP trace
spans, where each hop becomes a child span with latency,
address, and TTL attributes. Any OTLP-compatible backend
(Jaeger, Grafana Tempo, etc.) can ingest these spans.

The REST API provides the same detail as a pull-based JSON
endpoint for consumers that do not run a tracing backend.

### Prometheus Metrics

- `sparrow_traceroute_check_duration_ms` (Gauge) — Total
  trace duration per target. Labelled with `target`.
- `sparrow_traceroute_minimum_hops` (Gauge) — Minimum hops
  to reach a target. Labelled with `target`.

### REST API

The traceroute check exposes detailed hop-by-hop data at
`/v1/metrics/traceroute`. This JSON format mirrors
traditional traceroute output:

```bash
$ traceroute -T -q 1 100.1.2.2
 1  200.2.0.1 (200.2.0.1)  2 ms
 2  11.0.0.34 (11.0.0.34)  5 ms
 ...
```

Equivalent API response:

```json
{
  "data": {
    "100.1.2.2": {
      "min_hops": 1,
      "hops": {
        "1": [
          {
            "name": "router.example.com",
            "latency": 2,
            "addr": { "ip": "200.2.0.1", "port": 80 },
            "ttl": 1,
            "reached": false
          }
        ]
      }
    }
  },
  "timestamp": "2024-07-26T15:49:39.607+02:00"
}
```

## See Also

- [Checks overview](../checks.md)
- [API](../api.md)
- [Observability](../observability.md)
- [Traceroute test lab](../dev/traceroute-testing.md)
