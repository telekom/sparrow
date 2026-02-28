<!--
SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH

SPDX-License-Identifier: CC-BY-4.0
-->

# Design: Sparrow Instance Ownership Metadata (Issue #354)

## Summary

Sparrow exposes optional instance metadata via a dedicated Prometheus **info metric** (`sparrow_instance_info`), so operators can identify owners, route alerts correctly, and correlate metrics across multiple Sparrow deployments.

## Why Option 1 (Dedicated Info Metric)

- **Prometheus best practice:** Info-style metrics (gauge with value 1 and descriptive labels) are the standard way to expose static attributes (e.g. `kube_pod_info`, `node_uname_info`). They avoid polluting every time series with extra labels and keep cardinality under control.
- **No impact on existing metrics:** We do **not** add metadata labels to check metrics (health, latency, DNS, traceroute). That would multiply cardinality and complicate existing dashboards. Joining with `sparrow_instance_info` in PromQL when needed is explicit and flexible.
- **Works without target manager:** The metric is registered at startup from startup config only. It does not depend on the target manager or any runtime component.
- **Single emission per instance:** The metric is registered once during `sparrow.New()` and emits one time series per instance. No periodic updates or lifecycle complexity.

## Implementation Choices

1. **Config shape:** `metadata` is a map of arbitrary key-value pairs under startup config (e.g. `team_name`, `platform`, `region`). All fields optional; omitted keys are not emitted as labels. The key `instance_name` is reserved and set from the Sparrow DNS name.
2. **Registration point:** Instance info is registered in `sparrow.New()` after the metrics provider is created. Registration failure is logged but non-fatal so the process still starts.
3. **Metrics package:** A small `RegisterInstanceInfo(registry, instanceName, metadata)` in `pkg/sparrow/metrics` keeps the metrics package independent of `pkg/config` and makes the behaviour easy to test.
4. **Helm:** Metadata is optional under `sparrowConfig` in values; backward compatibility is preserved when metadata is not provided.

## Prometheus Usage

- **Alert routing:** Alertmanager or routing rules can use `sparrow_instance_info` to add ownership metadata to alerts.
- **Dashboards:** `group_left(...) sparrow_instance_info` joins metadata onto any Sparrow metric by scrape `instance`.
- **Multi-team views:** Filter or group by `team_name`, `platform`, `region`, or any other configured labels without changing existing metric names or labels.

## Deliverables

- **Code:** `pkg/config` (Metadata map), `pkg/sparrow/metrics` (RegisterInstanceInfo + test), `pkg/sparrow` (registration in New).
- **Helm:** `chart/values.yaml` extended with commented metadata example; config is merged into existing sparrowConfig.
- **Docs:** README (metadata config, instance info metric, PromQL examples), this design summary.
