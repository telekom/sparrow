// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/caas-team/sparrow/cmd"
)

// Version is the current version of sparrow
// It is set at build time by using -ldflags "-X main.version=x.x.x"
var version string

func main() {
	cmd.Execute(version)
}
