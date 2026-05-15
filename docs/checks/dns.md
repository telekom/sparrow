<!--
SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH

SPDX-License-Identifier: CC-BY-4.0
-->

# DNS check

The DNS check monitors domain name resolution performance
and reliability.

> [!CAUTION]
> Starting from v0.6.0, the API returns lowercase keys
> instead of capitalized keys.

## Configuration

| Field         | Type              | Description                                                                                                        |
| ------------- | ----------------- | ------------------------------------------------------------------------------------------------------------------ |
| `interval`    | `duration`        | Interval between DNS checks.                                                                                       |
| `timeout`     | `duration`        | Timeout for each lookup.                                                                                           |
| `retry.count` | `integer`         | Number of retries on failure.                                                                                      |
| `retry.delay` | `duration`        | Initial delay between retries.                                                                                     |
| `targets`     | `list of strings` | Domains or IPs to look up. The [target manager][target-manager] merges global targets into this list when enabled. |

[target-manager]: ../configuration.md#target-manager

### Example

```yaml
dns:
  interval: 10s
  timeout: 30s
  retry:
    count: 3
    delay: 1s
  targets:
    - www.example.com
    - www.google.com
```

## Metrics

- `sparrow_dns_status` (Gauge) — Lookup status. Labelled
  with `target`.
- `sparrow_dns_check_count` (Counter) — Total DNS checks.
  Labelled with `target`.
- `sparrow_dns_duration_seconds` (Gauge) — Resolution
  duration. Labelled with `target`.
- `sparrow_dns_duration` (Histogram) — Resolution time
  distribution. Labelled with `target`.

## See Also

- [Checks overview](../checks.md)
- [Observability](../observability.md)
