<!--
SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH

SPDX-License-Identifier: CC-BY-4.0
-->

# Generate SBOM with Syft

This doc can be used to generate a SBOM manually with [Syft](https://github.com/anchore/syft).

## Usage

Install the Syft binary.

Use the following command to generate a simple SBOM file form the repository:

```shell
syft .
```

Alternative output variants can be found [here](https://github.com/anchore/syft/wiki/Output-Formats).

Use the following command to generate a SBOM markdown file using the `example.sbom.tmpl` goTemplate template file:

```shell
SYFT_GOLANG_SEARCH_REMOTE_LICENSES=true syft ghcr.io/telekom/sparrow:v0.5.0 -o template -t scripts/sbom/example.sbom.tmpl
```

Setting the env variable `SYFT_GOLANG_SEARCH_REMOTE_LICENSES=true` will ensure to lookup licenses remotely. In this example the sparrow image in version `v0.5.0` is scanned.
