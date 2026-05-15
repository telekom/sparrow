<!--
SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH

SPDX-License-Identifier: CC-BY-4.0
-->

# Developer Documentation

- [Using `reuse` for license handling](#using-reuse-for-license-handling)
  - [Install](#install)
  - [CLI usage](#cli-usage)
  - [`REUSE.toml`](#reusetoml)
- [Running tests](#running-tests)

## Using `reuse` for license handling

`reuse` is a
[CLI tool](https://reuse.readthedocs.io/en/latest/index.html)
to handle licenses with
[SPDX standard](https://spdx.dev/use/overview/) in a
repository.

How `reuse` is setup and used is explained in the
[Telekom Org documentation][reuse-setup].

For details about license handling with `reuse` checkout
this [page][reuse-license].

[reuse-setup]: https://telekom.github.io/reuse-template/using-the-template/setup-reuse
[reuse-license]: https://telekom.github.io/reuse-template/using-the-template/setup-license

### Install

Install `reuse` following the
[official instructions][reuse-install].

`reuse` is embedded as a pre-commit hook. Run
`pre-commit install` to use it. The hook checks the
repository for license compliance.

[reuse-install]: https://reuse.readthedocs.io/en/latest/readme.html#install

### CLI Usage

> `reuse lint`

Verify the compliance of the repository.

> `reuse annotate -c "Deutsche Telekom IT GmbH" \`
> `-l "Apache-2.0" --recursive --skip-existing \`
> `--fallback-dot-license <PATH-TO-FILE/DIR>`

Annotate a file or directory with the `Apache-2.0` SPDX
license header. If the tool cannot append a header, it
creates a `.license` file.

> `reuse annotate -c "Deutsche Telekom IT GmbH" \`
> `-l "CC-BY-4.0" --recursive --skip-existing \`
> `--fallback-dot-license ./docs/*.md`

Use the `CC-BY-4.0` SPDX license header for documentation
(e.g. `.md`) files.

### `REUSE.toml`

If a license header is not suitable for a file or directory
(e.g. auto-generated files), use the `REUSE.toml`
configuration file instead.

Add the file path to `REUSE.toml` or create a new
`[[annotations]]` section (see `./REUSE.toml` for an
example).

## Running Tests

Unit tests run with the Go toolchain and modules specified
in `go.mod`.

**First run:** If the Go version in `go.mod` is not yet
installed, the first `go test` (or any `go` command)
downloads the toolchain and dependencies. This can take
**2–5 minutes** depending on the network. If your IDE or
test runner uses a short timeout (e.g. 30–60 seconds), the
first run may time out. Use a longer timeout or run from a
terminal:

```bash
go test ./...
```

**Run only metrics (e.g. instance_info) tests:**

```bash
go test ./pkg/sparrow/metrics/ \
  -run 'InstanceInfo' -v -count=1
```

**Run all tests with race detector and coverage (as in
CI):**

```bash
go mod download
go test --race --count=1 \
  --coverprofile cover.out -v ./...
```
