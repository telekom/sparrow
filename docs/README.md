<!--
SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH

SPDX-License-Identifier: CC-BY-4.0
-->

# Sparrow documentation

## Getting Started

- [Installation](installation.md) — binary, container image,
  Helm chart
- [Configuration](configuration.md) — startup config, loader,
  logging, target manager

## Checks

- [Checks overview](checks.md) — how checks work, target
  management
- [Health](checks/health.md) — HTTP health probes
- [Latency](checks/latency.md) — round-trip latency
  measurement
- [DNS](checks/dns.md) — DNS resolution monitoring
- [Traceroute](checks/traceroute.md) — network path tracing

## Reference

- [API](api.md) — HTTP API endpoints
- [Observability](observability.md) — metrics, traces,
  Grafana dashboards
- [CLI reference](reference/sparrow.md) — command-line flags
  and subcommands
- [Helm chart values](../chart/README.md) — all available
  Helm values

## Contributing

- [Developer guide](dev/README.md)
- [Traceroute test lab](dev/traceroute-testing.md)
- [Generating an SBOM](dev/sbom.md)
- [Contributing guidelines](../CONTRIBUTING.md)
- [Code of conduct](../CODE_OF_CONDUCT.md)

## Design Documents

- [Ownership metadata](ownership-metadata-design.md)
