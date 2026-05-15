<!--
SPDX-FileCopyrightText: 2026 Deutsche Telekom IT GmbH

SPDX-License-Identifier: CC-BY-4.0
-->

# Quickstart

Run your first health check in under five minutes.

## Prerequisites

Download and install the sparrow binary. See
[Installation](installation.md#binary) for the exact commands.

## Create a checks configuration

Sparrow loads its checks configuration at runtime from a separate
file. Create `checks.yaml` with a single health check targeting
`http://example.com`:

```yaml
health:
  interval: 30s
  targets:
    - url: http://example.com
```

## Run sparrow

Start sparrow pointing at the file you just created:

```sh
sparrow run \
  --sparrowName sparrow.local \
  --loaderType file \
  --loaderFilePath ./checks.yaml
```

Sparrow starts the API on `:8080` and reloads `checks.yaml` every
five minutes.

## Verify

Check the raw JSON results:

```sh
curl http://localhost:8080/v1/metrics/health
```

Or scrape the Prometheus endpoint:

```sh
curl http://localhost:8080/metrics
```

Look for `sparrow_health_up` in the Prometheus output — a value of
`1` means the target is reachable.

## Next steps

- [Configuration](configuration.md) — change the check interval,
  switch to the HTTP loader, enable TLS
- [Checks overview](checks.md) — add latency, DNS, or traceroute
  checks
- [Observability](observability.md) — connect Prometheus and
  Grafana dashboards
