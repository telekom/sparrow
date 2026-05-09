// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"fmt"
	"maps"
	"slices"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
)

const (
	instanceInfoMetric = "sparrow_instance_info"
	instanceInfoHelp   = "Ownership and platform metadata for this Sparrow instance. Emitted once per instance for alert routing and multi-team correlation."
	instanceNameLabel  = "instance_name"
)

// RegisterInstanceInfo registers the sparrow_instance_info info-style metric on the given registry.
// It sets the gauge to 1 with labels instance_name and any user-defined metadata keys.
// Empty strings are allowed for metadata values; instanceName should be the Sparrow DNS name.
// Metadata keys must be valid Prometheus label names and must not include "instance_name".
func RegisterInstanceInfo(registry *prometheus.Registry, instanceName string, metadata map[string]string) error {
	if metadata == nil {
		metadata = map[string]string{}
	}

	labels := make([]string, 0, len(metadata)+1)
	values := make([]string, 0, len(metadata)+1)
	labels = append(labels, instanceNameLabel)
	values = append(values, instanceName)

	keys := slices.Collect(maps.Keys(metadata))
	for _, label := range keys {
		if label == instanceNameLabel {
			return fmt.Errorf("metadata key %q is reserved", label)
		}
		if !model.UTF8Validation.IsValidLabelName(label) {
			return fmt.Errorf("metadata key %q is not a valid Prometheus label name", label)
		}
		labels = append(labels, label)
		values = append(values, metadata[label])
	}

	info := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: instanceInfoMetric,
			Help: instanceInfoHelp,
		},
		labels,
	)
	info.WithLabelValues(values...).Set(1)
	return registry.Register(info)
}
