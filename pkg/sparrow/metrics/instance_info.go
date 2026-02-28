// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"fmt"
	"regexp"
	"sort"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	instanceInfoMetricName = "sparrow_instance_info"
	instanceInfoHelp       = "Ownership and platform metadata for this Sparrow instance. Emitted once per instance for alert routing and multi-team correlation."
)

// RegisterInstanceInfo registers the sparrow_instance_info info-style metric on the given registry.
// It sets the gauge to 1 with labels instance_name and any user-defined metadata keys.
// Empty strings are allowed for metadata values; instanceName should be the Sparrow DNS name.
// Metadata keys must be valid Prometheus label names and must not include "instance_name".
func RegisterInstanceInfo(registry *prometheus.Registry, instanceName string, metadata map[string]string) error {
	if metadata == nil {
		metadata = map[string]string{}
	}

	keys := make([]string, 0, len(metadata))
	for k := range metadata {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	labels := make([]string, 0, len(keys)+1)
	values := make([]string, 0, len(keys)+1)
	labels = append(labels, "instance_name")
	values = append(values, instanceName)

	for _, k := range keys {
		if k == "instance_name" {
			return fmt.Errorf("metadata key %q is reserved", k)
		}
		if !isValidLabelName(k) {
			return fmt.Errorf("metadata key %q is not a valid Prometheus label name", k)
		}
		labels = append(labels, k)
		values = append(values, metadata[k])
	}

	info := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: instanceInfoMetricName,
			Help: instanceInfoHelp,
		},
		labels,
	)
	info.WithLabelValues(values...).Set(1)
	return registry.Register(info)
}

var labelNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func isValidLabelName(name string) bool {
	return labelNamePattern.MatchString(name)
}
