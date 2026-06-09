<!--
SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH

SPDX-License-Identifier: CC-BY-4.0
-->

# Configuration

Sparrow configuration has two layers:

1. **Startup configuration** — technical settings for the
   instance itself (API address, loader, telemetry, target
   manager). Set once at startup.
2. **Checks configuration** — which checks to run and their
   parameters. The [loader](#loader) fetches this dynamically
   at runtime.

## Startup Configuration

All CLI flags are documented in the
[CLI reference](reference/sparrow_run.md).

Configuration sources in order of priority (highest first):

1. CLI flags
2. Environment variables
3. Specified configuration file (`--config`)
4. Default configuration file

Every config key can be set via environment variable by writing
the path delimited with `_`:

```bash
export SPARROW_NAME="sparrow.example.com"
export SPARROW_LOADER_HTTP_TOKEN="xxxxxx"
export SPARROW_ANY_OTHER_OPTION="Some value"
```

### Instance Metadata (Optional)

You can attach arbitrary key-value metadata to an instance.
This metadata is exposed as the `sparrow_instance_info`
Prometheus metric (see [observability](observability.md)).

Keys must be valid Prometheus label names (e.g. `team_name`,
`platform`, `region`). Sparrow reserves the key
`instance_name` and sets it automatically.

### Example Startup Configuration

```yaml
# DNS name the sparrow is exposed on
name: sparrow.example.com

# Optional: instance metadata
# metadata:
#   team_name: platform-team
#   team_email: platform@example.com
#   platform: k8s-prod-eu
#   region: eu-west-1

loader:
  type: http
  interval: 30s
  http:
    url: https://myconfig.example.com/config.yaml
    token: xxxxxxx
    timeout: 30s
    retry:
      delay: 10s
      count: 3
  file:
    path: ./config.yaml

api:
  address: :8080
  tls:
    enabled: true
    certPath: mycert.pem
    keyPath: mykey.key

targetManager:
  enabled: true
  type: gitlab
  checkInterval: 1m
  registrationInterval: 1m
  updateInterval: 120m
  unhealthyThreshold: 360m
  scheme: http
  gitlab:
    baseUrl: https://gitlab.com
    token: glpat-xxxxxxxx
    projectId: 18923
    branch: main

telemetry:
  enabled: true
  exporter: grpc
  url: localhost:4317
  token: ""
  tls:
    enabled: true
    certPath: ""
```

## Loader

The loader fetches the [checks configuration](checks.md) at
runtime.

| Loader | Description                                         |
| ------ | --------------------------------------------------- |
| `http` | Retrieves config from a remote endpoint (default).  |
| `file` | Loads config from a local file. Not for production. |

Set `loader.interval` to `0` to fetch the configuration only
once. The target manager does not work with a one-shot
configuration.

## Logging

| Variable     | Options                             | Description       |
| ------------ | ----------------------------------- | ----------------- |
| `LOG_LEVEL`  | `DEBUG`, `INFO`, `WARNING`, `ERROR` | Minimum log level |
| `LOG_FORMAT` | `JSON`, `TEXT`                      | Log output format |

## Target Manager

The target manager enables sparrow instances to discover
each other and register as check targets on a remote
backend. When enabled, each instance:

1. **Registers itself** as a global target at the configured
   `registrationInterval`.
2. **Discovers peers** by polling the backend at
   `checkInterval` and adding them to its local target list.
3. **Merges** global targets with any statically configured
   targets — both lists feed into each check.
4. **Cleans up** targets that have not been seen within the
   `unhealthyThreshold` window (set to `0` to skip cleanup).

| Field                                | Description                                            |
| ------------------------------------ | ------------------------------------------------------ |
| `targetManager.enabled`              | Enable the target manager (default: `false`).          |
| `targetManager.type`                 | Backend type. Currently only `gitlab`.                 |
| `targetManager.scheme`               | `http` or `https`. Must match `api.tls.enabled`.       |
| `targetManager.checkInterval`        | How often to poll for new targets.                     |
| `targetManager.unhealthyThreshold`   | Remove targets unseen for this long. `0` = no cleanup. |
| `targetManager.registrationInterval` | How often to register. `0` = no registration.          |
| `targetManager.updateInterval`       | How often to refresh registration. `0` = no update.    |
| `targetManager.gitlab.baseUrl`       | GitLab instance URL.                                   |
| `targetManager.gitlab.token`         | GitLab API token.                                      |
| `targetManager.gitlab.projectId`     | Project ID used as state backend.                      |
| `targetManager.gitlab.branch`        | Branch for state file (defaults to `main`).            |

### GitLab Backend

The GitLab target manager uses a GitLab project as a shared
state backend. Each sparrow instance commits a state file
named after its DNS name to the configured branch:

```json
{
  "url": "<SCHEME>://<SPARROW_DNS_NAME>",
  "lastSeen": "2026-04-30T12:00:00Z"
}
```

Other instances read these files to build their global
target list.

## See Also

- [Checks overview](checks.md)
- [Observability](observability.md) — metrics, traces,
  dashboards
- [CLI reference](reference/sparrow_run.md)
