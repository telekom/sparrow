<!--
SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH

SPDX-License-Identifier: CC-BY-4.0
-->

# Installation

Sparrow is available as a static binary, a container image,
and a Helm chart. See the
[release notes](https://github.com/telekom/sparrow/releases)
for the latest version.

## Binary

Download and extract the binary for your platform.
Replace `${RELEASE_VERSION}` with the desired version:

```sh
export RELEASE_VERSION=0.5.0
```

```sh
curl https://github.com/telekom/sparrow/releases/download/v${RELEASE_VERSION}/sparrow_${RELEASE_VERSION}_linux_amd64.tar.gz -Lo sparrow.tar.gz
curl https://github.com/telekom/sparrow/releases/download/v${RELEASE_VERSION}/sparrow_${RELEASE_VERSION}_checksums.txt -Lo checksums.txt
```

```sh
tar -xf sparrow.tar.gz
```

## Container Image

Each release publishes container images to the
[GitHub registry](https://github.com/telekom/sparrow/pkgs/container/sparrow).

```sh
docker run ghcr.io/telekom/sparrow --help
```

Mount a startup configuration file:

```sh
docker run -v /config:/config \
  ghcr.io/telekom/sparrow --config /config/config.yaml
```

## Helm

```sh
helm -n sparrow upgrade -i sparrow \
  oci://ghcr.io/telekom/charts/sparrow --create-namespace
```

The default Helm values use the `file` loader with a ConfigMap.
Define the `checksConfig` section to set the ConfigMap contents.

To switch to the `http` loader at runtime:

```yaml
sparrowConfig:
  name: sparrow.example.com
  loader:
    type: http
    interval: 30s
    http:
      url: https://url-to-checks-config.de/api/config%2Eyaml

checksConfig: {}
```

> [!IMPORTANT]
> Do not put secrets (loader token, target manager token) in
> `values.yaml`. Create a Kubernetes secret with the relevant
> environment variable (`SPARROW_LOADER_HTTP_TOKEN`,
> `SPARROW_TARGETMANAGER_GITLAB_TOKEN`) and reference it via
> `envFromSecrets`.

For all available Helm values see the
[chart README](../chart/README.md).

## Next Steps

- [Configuration](configuration.md) — startup config, loader,
  logging
- [Checks overview](checks.md) — available checks and how to
  enable them
