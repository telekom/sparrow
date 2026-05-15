<!--
SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH

SPDX-License-Identifier: CC-BY-4.0
-->

# Generating an SBOM with Syft

## Quick Start

Install the Syft binary, then scan the repository:

```shell
syft .
```

See the [Syft output formats][syft-formats] for alternative
output variants.

[syft-formats]: https://github.com/anchore/syft/wiki/Output-Formats

## Generate a Markdown SBOM from a Container Image

Use the Go template in `scripts/sbom/example.sbom.tmpl` to
produce a Markdown-formatted SBOM. Set
`SYFT_GOLANG_SEARCH_REMOTE_LICENSES=true` so Syft looks up
Go module licenses remotely:

```shell
SYFT_GOLANG_SEARCH_REMOTE_LICENSES=true \
  syft ghcr.io/telekom/sparrow:v0.5.0 \
  -o template -t scripts/sbom/example.sbom.tmpl
```

Replace `v0.5.0` with the version you want to scan.

## See Also

- [Developer guide](README.md)
