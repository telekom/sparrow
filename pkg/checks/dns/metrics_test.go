// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package dns

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestMetrics_GetCollectors(t *testing.T) {
	tests := []struct {
		name    string
		metrics metrics
	}{
		{
			name:    "success with metrics constructor",
			metrics: newMetrics(),
		},
		{
			name: "success with custom metrics",
			metrics: metrics{
				status: prometheus.NewGaugeVec(
					prometheus.GaugeOpts{
						Name: "sparrow_dns_status",
						Help: "Specifies if the target can be resolved.",
					},
					[]string{"target"},
				),
				duration: prometheus.NewGaugeVec(
					prometheus.GaugeOpts{
						Name: "sparrow_dns_duration",
						Help: "Duration of DNS resolution attempts in seconds.",
					},
					[]string{"target"},
				),
				count: prometheus.NewCounterVec(
					prometheus.CounterOpts{
						Name: "sparrow_dns_check_count",
						Help: "Total number of DNS checks performed on the target and if they were successful.",
					},
					[]string{"target"},
				),
				histogram: prometheus.NewHistogramVec(
					prometheus.HistogramOpts{
						Name: "sparrow_dns_duration",
						Help: "Histogram of response times for DNS checks in seconds.",
					},
					[]string{"target"},
				),
			},
		},
	}
	for _, tt := range tests {
		tt.metrics.Set("test", make(map[string]result, 1), float64(1))

		if tt.metrics.GetCollectors() == nil {
			t.Errorf("metrics.GetCollectors() = %v", tt.metrics.GetCollectors())
		}
	}
}
