<!--
SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH

SPDX-License-Identifier: CC-BY-4.0
-->

# Changelog

## [v0.6.0](https://github.com/telekom/sparrow/releases/tag/v0.6.0) (DATE)

⚠️ This release contains potential breaking changes ⚠️

The API returns lowercase keys instead of capitalized keys. Ensure that your code handles this change to avoid issues.

* [FEATURE] added Changelog file
* [BUGFIX] DNS endpoint `/v1/metrics/dns` returns lowercase keys instead of capitalized keys.
* [BUGFIX] Traceroute endpoint `/v1/metrics/traceroute` returns lowercase keys instead of capitalized keys.
