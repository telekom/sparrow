<!--
SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH

SPDX-License-Identifier: CC-BY-4.0
-->

# Checks

Sparrow runs periodic checks to monitor infrastructure and
network health. The [loader](configuration.md#loader) fetches
the checks configuration at runtime and starts each enabled
check.

## Available Checks

| Check                              | Description                                    |
| ---------------------------------- | ---------------------------------------------- |
| [Health](checks/health.md)         | HTTP/1.1 health probes to target endpoints     |
| [Latency](checks/latency.md)       | Round-trip latency measurement between targets |
| [DNS](checks/dns.md)               | DNS resolution monitoring for domains and IPs  |
| [Traceroute](checks/traceroute.md) | Network path tracing to targets                |

## Checks Configuration Format

Enable a check by adding its section to the checks
configuration YAML (the file the
[loader](configuration.md#loader) fetches):

```yaml
health:
  targets: []
```

For loader configuration options, see the
[CLI reference](reference/sparrow_run.md).

## Target Management

Each check accepts a `targets` list. You can populate this
list in two ways:

- **Static targets** — define URLs or addresses directly in
  the checks configuration file.
- **Global targets** — enable the
  [target manager](configuration.md#target-manager) to let
  sparrow instances discover and register each other
  automatically.

When both sources exist, sparrow **merges** them: every check
receives the union of its static targets and the global
targets from the target manager. See
[Target manager](configuration.md#target-manager) for setup
details.

## See Also

- [Configuration](configuration.md) — startup config and
  loader setup
- [Observability](observability.md) — metrics emitted by
  each check
