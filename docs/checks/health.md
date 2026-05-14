<!--
SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH

SPDX-License-Identifier: CC-BY-4.0
-->

# Health check

The health check performs HTTP/1.1 probes against a list of
target endpoints. Sparrow also exposes its own health endpoint
so other instances can probe it.

## Configuration

| Field         | Type              | Description                                                                                            |
| ------------- | ----------------- | ------------------------------------------------------------------------------------------------------ |
| `interval`    | `duration`        | Interval between health checks.                                                                        |
| `timeout`     | `duration`        | Timeout for each probe.                                                                                |
| `retry.count` | `integer`         | Number of retries on failure.                                                                          |
| `retry.delay` | `duration`        | Initial delay between retries.                                                                         |
| `targets`     | `list of strings` | URLs to probe. The [target manager][target-manager] merges global targets into this list when enabled. |

[target-manager]: ../configuration.md#target-manager

### Example

```yaml
health:
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

- `sparrow_health_up` (Gauge) — Health of targets. Labelled
  with `target`.

## See Also

- [Checks overview](../checks.md)
- [Observability](../observability.md)
