// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package health

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/telekom/sparrow/pkg/checks"
)

// metrics contains the metric collectors for the Health check
type metrics struct {
	*prometheus.GaugeVec
}

// newMetrics initializes metric collectors of the health check
func newMetrics() metrics {
	return metrics{
		GaugeVec: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "sparrow_health_up",
				Help: "Health of targets",
			},
			[]string{
				"target",
			},
		),
	}
}

// Remove removes a metric with a specific label
func (m *metrics) Remove(label string) error {
	if !m.DeleteLabelValues(label) {
		return checks.ErrMetricNotFound{Label: label}
	}
	return nil
}
