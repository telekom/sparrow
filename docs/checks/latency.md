<!--
SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH

SPDX-License-Identifier: CC-BY-4.0
-->

# Latency check

The latency check measures HTTP/1.1 round-trip time between
sparrow instances or to arbitrary endpoints.

## Configuration

| Field         | Type              | Description                                                                                            |
| ------------- | ----------------- | ------------------------------------------------------------------------------------------------------ |
| `interval`    | `duration`        | Interval between latency checks.                                                                       |
| `timeout`     | `duration`        | Timeout for each probe.                                                                                |
| `retry.count` | `integer`         | Number of retries on failure.                                                                          |
| `retry.delay` | `duration`        | Initial delay between retries.                                                                         |
| `targets`     | `list of strings` | URLs to probe. The [target manager][target-manager] merges global targets into this list when enabled. |

[target-manager]: ../configuration.md#target-manager

### Example

```yaml
latency:
  interval: 10s
  timeout: 30s
  retry:
    count: 3
    delay: 1s
  targets:
    - https://example.com/
    - https://google.com/
```

## Metrics

- `sparrow_latency_duration_seconds` (Gauge) — Latency with
  status. **Deprecated** — use `sparrow_latency_seconds`.
  Labelled with `target`, `status`.
- `sparrow_latency_seconds` (Gauge) — Latency of targets.
  Labelled with `target`.
- `sparrow_latency_count` (Counter) — Total latency checks.
  Labelled with `target`.
- `sparrow_latency_duration` (Histogram) — Latency
  distribution in seconds. Labelled with `target`.

## See Also

- [Checks overview](../checks.md)
- [Observability](../observability.md)
