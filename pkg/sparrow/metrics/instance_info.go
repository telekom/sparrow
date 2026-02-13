// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	instanceInfoMetricName = "sparrow_instance_info"
	instanceInfoHelp       = "Ownership and platform metadata for this Sparrow instance. Emitted once per instance for alert routing and multi-team correlation."
)

// RegisterInstanceInfo registers the sparrow_instance_info info-style metric on the given registry.
// It sets the gauge to 1 with labels team_name, team_email, platform, and instance_name.
// Empty strings are allowed for optional metadata; instanceName should be the Sparrow DNS name.
func RegisterInstanceInfo(registry *prometheus.Registry, instanceName, teamName, teamEmail, platform string) error {
	info := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: instanceInfoMetricName,
			Help: instanceInfoHelp,
		},
		[]string{"team_name", "team_email", "platform", "instance_name"},
	)
	info.WithLabelValues(teamName, teamEmail, platform, instanceName).Set(1)
	return registry.Register(info)
}
