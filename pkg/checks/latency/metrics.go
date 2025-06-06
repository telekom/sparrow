// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package latency

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/telekom/sparrow/pkg/checks"
)

// metrics defines the metric collectors of the latency check
type metrics struct {
	totalDuration *prometheus.GaugeVec
	count         *prometheus.CounterVec
	histogram     *prometheus.HistogramVec
}

// newMetrics initializes metric collectors of the latency check
func newMetrics() metrics {
	return metrics{
		totalDuration: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "sparrow_latency_seconds",
				Help: "Latency for each target",
			},
			[]string{
				"target",
			},
		),
		count: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "sparrow_latency_count",
				Help: "Count of latency checks done",
			},
			[]string{
				"target",
			},
		),
		histogram: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "sparrow_latency_duration",
				Help: "Latency of targets in seconds",
			},
			[]string{
				"target",
			},
		),
	}
}

// Remove removes the metrics which have the passed target as a label
func (m metrics) Remove(label string) error {
	if !m.totalDuration.Delete(map[string]string{"target": label}) {
		return checks.ErrMetricNotFound{Label: label}
	}

	if !m.count.Delete(map[string]string{"target": label}) {
		return checks.ErrMetricNotFound{Label: label}
	}

	if !m.histogram.Delete(map[string]string{"target": label}) {
		return checks.ErrMetricNotFound{Label: label}
	}

	return nil
}
