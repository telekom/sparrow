<!--
SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH

SPDX-License-Identifier: CC-BY-4.0
-->

# Sparrow — Infrastructure Monitoring

<!-- markdownlint-disable -->
<p align="center">
  <a href="/../../commits/" title="Last Commit"><img alt="Last Commit" src="https://img.shields.io/github/last-commit/telekom/sparrow?style=flat"></a>
  <a href="/../../issues" title="Open Issues"><img alt="Open Issues" src="https://img.shields.io/github/issues/telekom/sparrow?style=flat"></a>
  <a href="./LICENSE" title="License"><img alt="License" src="https://img.shields.io/badge/License-Apache%202.0-green.svg?style=flat"></a>
</p>
<!-- markdownlint-enable -->

Sparrow is an infrastructure monitoring agent that runs
periodic checks from its own vantage point — health probes,
latency measurement, DNS resolution, and traceroute — and
exposes the results as Prometheus metrics and a REST API.

## Quick Start

Install via Helm:

```sh
helm -n sparrow upgrade -i sparrow \
  oci://ghcr.io/telekom/charts/sparrow --create-namespace
```

Or run the binary directly:

```sh
sparrow run --sparrowName sparrow.example.com
```

See [Installation](docs/installation.md) for all options
(binary, container image, Helm).

## Available Checks

| Check                                   | Description                                 |
| --------------------------------------- | ------------------------------------------- |
| [Health](docs/checks/health.md)         | HTTP/1.1 health probes to endpoints         |
| [Latency](docs/checks/latency.md)       | Round-trip time between sparrow instances   |
| [DNS](docs/checks/dns.md)               | Domain resolution monitoring                |
| [Traceroute](docs/checks/traceroute.md) | Network path tracing with hop-by-hop detail |

## Documentation

Full documentation lives in the [`docs/`](docs/README.md)
directory:

- [Installation](docs/installation.md)
- [Configuration](docs/configuration.md)
- [Checks overview](docs/checks.md)
- [API](docs/api.md)
- [Observability](docs/observability.md) — metrics, traces,
  Grafana dashboards
- [CLI reference](docs/reference/sparrow.md)
- [Helm chart values](chart/README.md)

## Contributing

Contribution and feedback is encouraged and always welcome.
See our [contribution guidelines](CONTRIBUTING.md) for
details.

## Code of Conduct

This project has adopted the [Contributor Covenant][contributor-covenant]
in version 2.1 as our code of conduct. Please see the details in our
[CODE_OF_CONDUCT.md][code-of-conduct].

By participating in this project, you agree to abide by
its [Code of Conduct](CODE_OF_CONDUCT.md) at all times.

[contributor-covenant]: https://www.contributor-covenant.org
[code-of-conduct]: CODE_OF_CONDUCT.md

## Working Language

We decided to apply *English* as the primary project language.

Consequently, all content will be made
available primarily in English.

We also ask all interested people to use
English as the preferred language to create
issues, in their code (comments, documentation, etc.)
and when you send pull requests to us.

The application itself and all end-user facing content
will be made available in other languages as needed.

## Support and Feedback

<!-- markdownlint-disable MD033 -->
| Type       | Channel                                                                                                                                                           |
| ---------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Issues** | <a href="/../../issues/new/choose" title="General Discussion"><img alt="Issues" src="https://img.shields.io/github/issues/telekom/sparrow?style=flat-square"></a> |
<!-- markdownlint-enable MD033 -->

## Licensing

This project follows the
[REUSE standard](https://reuse.software/). Each file contains
copyright and license information. License texts are in the
[LICENSES](./LICENSES) folder. See the
[REUSE developer guide](https://telekom.github.io/reuse-template/).
