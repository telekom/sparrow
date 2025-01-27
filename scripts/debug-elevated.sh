# SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
#
# SPDX-License-Identifier: Apache-2.0

# Description: Debug the application with elevated privileges
# This is only necessary when debugging issues with the traceroute check,
# as it requires elevated privileges
# to createa a raw socket
#
# Usage: 
# 1. Create a config for debugging in .tmp/config.yaml and a .tmp/runtime.yaml
#
# 2. Run the following command from the root of the project
# ./scripts/debug-elevated.sh
# 
# 3. Attach to the debugger with launch.json in vscode
go build -gcflags '-N -l'  -o .tmp/app ./ &&  sudo dlv exec .tmp/app  -- run --config .tmp/config.yaml
