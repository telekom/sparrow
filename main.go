// sparrow
// (C) 2023, Deutsche Telekom IT GmbH
//
// Deutsche Telekom IT GmbH and all other contributors /
// copyright owners license this file to you under the Apache
// License, Version 2.0 (the "License"); you may not use this
// file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package main

import (
	"github.com/telekom/sparrow/cmd"
	"github.com/telekom/sparrow/pkg"
)

// version is the current version of sparrow
// It is set at build time by using -ldflags "-X main.version=x.x.x"
var version string

func init() { //nolint:gochecknoinits // Required for version to be set on build
	pkg.Version = version
}

func main() {
	cmd.Execute(version)
}
