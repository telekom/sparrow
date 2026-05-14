<!--
SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH

SPDX-License-Identifier: CC-BY-4.0
-->

# API

> [!CAUTION]
> Starting from v0.6.0, the API returns lowercase keys
> instead of capitalized keys.

Sparrow exposes an HTTP API for accessing check results.
Each check registers its own endpoint:

| Endpoint                   | Description                                          |
| -------------------------- | ---------------------------------------------------- |
| `/v1/metrics/{check-name}` | Results for a specific check                         |
| `/metrics`                 | Prometheus metrics                                   |
| `/openapi`                 | OpenAPI specification                                |
| `/`                        | Health endpoint for other Sparrow instances to probe |

Configure the API address and TLS settings in the
[startup configuration](configuration.md).

## See Also

- [Checks overview](checks.md)
- [Observability](observability.md)
- [Traceroute API metrics](checks/traceroute.md#api-metrics)
