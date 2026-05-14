#!/bin/bash
# SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
#
# SPDX-License-Identifier: Apache-2.0

# S3 e2e tests require Docker (testcontainers-go).
# Skip gracefully if Docker is not available.
if ! command -v docker &>/dev/null || ! docker info &>/dev/null 2>&1; then
  echo "[ SKIP ]: Docker not available, skipping S3 e2e tests"
  exit 0
fi

cd "$(git rev-parse --show-toplevel)" || exit 1
go test -tags e2e -race -count=1 -v -timeout=5m ./e2e/s3/
