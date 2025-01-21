<!--
SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH

SPDX-License-Identifier: CC-BY-4.0
-->

# Developer Documentation <!-- omit from toc -->

- [Using `reuse` for license handling](#using-reuse-for-license-handling)
  - [Install](#install)
  - [CLI Usage](#cli-usage)
  - [`REUSE.toml`](#reusetoml)

## Using `reuse` for license handling

`reuse` is a [cli tool](https://reuse.readthedocs.io/en/latest/index.html) to handle licenses with [SPDX standard](https://spdx.dev/use/overview/) in a repository.

How `reuse` is setup and used is explained in the [Telekom Org documentation](https://telekom.github.io/reuse-template/using-the-template/setup-reuse).

For details about license handling with `reuse` checkout this [page](https://telekom.github.io/reuse-template/using-the-template/setup-license).

### Install

Install `reuse` following the [official instructions](https://reuse.readthedocs.io/en/latest/readme.html#install).

`reuse` is embedded as a pre-commit hook. Install `pre-commit install` to use it. The hook is checking the repository to stay compliant in terms of licenses.

### CLI Usage

> `reuse lint`

Verify the compliance of the repository.

> `reuse annotate -c "Deutsche Telekom IT GmbH" -l "Apache-2.0" --recursive --skip-existing --fallback-dot-license <PATH-TO-FILE/DIR>`

Annotate a file or directory with the `Apache-2.0` SPDX license header. In case the tool is not able to append a header, a `.license` file is created.

> `reuse annotate -c "Deutsche Telekom IT GmbH" -l "CC-BY-4.0" --recursive --skip-existing --fallback-dot-license ./docs/*.md`

Use the `CC-BY-4.0` SPDX license header for documentation (eg. `.md`) files.

### `REUSE.toml`

In case a license header is not suitable for a file or directory (eg. auto-generated files) the `REUSE.toml` configuration file can be used.

Add the file path to the `REUSE.toml` or create a new `[[annotations]]` section (see `./REUSE.toml` for an example).
