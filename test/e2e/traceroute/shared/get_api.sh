# SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
#
# SPDX-License-Identifier: Apache-2.0

curl -s localhost:8080/v1/metrics/traceroute > /shared/api.json
curl -s localhost:8080/metrics > /shared/prometheus.txt