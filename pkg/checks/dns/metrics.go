// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package dns

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/telekom/sparrow/pkg/checks"
)

const (
	statusMetric    = "sparrow_dns_status"
	durationMetric  = "sparrow_dns_duration_seconds"
	countMetric     = "sparrow_dns_check_count"
	histogramMetric = "sparrow_dns_duration"
)

// metrics defines the metric collectors of the DNS check
type metrics struct {
	status    *prometheus.GaugeVec
	duration  *prometheus.GaugeVec
	count     *prometheus.CounterVec
	histogram *prometheus.HistogramVec
}

// newMetrics initializes metric collectors of the dns check
func newMetrics() metrics {
	return metrics{
		status: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: statusMetric,
				Help: "Specifies if the target can be resolved.",
			},
			[]string{checks.LabelTarget},
		),
		duration: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: durationMetric,
				Help: "Duration of DNS resolution attempts in seconds.",
			},
			[]string{checks.LabelTarget},
		),
		count: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: countMetric,
				Help: "Total number of DNS checks performed on the target and if they were successful.",
			},
			[]string{checks.LabelTarget},
		),
		histogram: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: histogramMetric,
				Help: "Histogram of response times for DNS checks in seconds.",
			},
			[]string{checks.LabelTarget},
		),
	}
}

// GetCollectors returns all metric collectors
func (m *metrics) GetCollectors() []prometheus.Collector {
	return []prometheus.Collector{
		m.status,
		m.duration,
		m.count,
		m.histogram,
	}
}

// Set sets the metrics of one lookup target result
func (m *metrics) Set(target string, results map[string]result, status float64) {
	m.duration.WithLabelValues(target).Set(results[target].Total)
	m.histogram.WithLabelValues(target).Observe(results[target].Total)
	m.status.WithLabelValues(target).Set(status)
	m.count.WithLabelValues(target).Inc()
}

// Remove removes the metrics of one lookup target
func (m *metrics) Remove(target string) error {
	if !m.status.DeleteLabelValues(target) {
		return checks.ErrMetricNotFound{Label: target}
	}

	if !m.duration.DeleteLabelValues(target) {
		return checks.ErrMetricNotFound{Label: target}
	}

	if !m.count.DeleteLabelValues(target) {
		return checks.ErrMetricNotFound{Label: target}
	}

	if !m.histogram.DeleteLabelValues(target) {
		return checks.ErrMetricNotFound{Label: target}
	}

	return nil
}
