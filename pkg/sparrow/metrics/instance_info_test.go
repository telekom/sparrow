// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterInstanceInfo(t *testing.T) {
	registry := prometheus.NewRegistry()

	err := RegisterInstanceInfo(registry, "sparrow.example.com", map[string]string{
		"team_name":  "platform-team",
		"team_email": "platform@example.com",
		"platform":   "k8s-prod-eu",
	})
	require.NoError(t, err, "RegisterInstanceInfo() should succeed")

	metrics, err := registry.Gather()
	require.NoError(t, err, "Gather() should succeed")

	wantLabels := map[string]string{
		"instance_name": "sparrow.example.com",
		"team_name":     "platform-team",
		"team_email":    "platform@example.com",
		"platform":      "k8s-prod-eu",
	}

	assertMetricsContainLabels(t, metrics, wantLabels)
}

func TestRegisterInstanceInfo_emptyMetadata(t *testing.T) {
	registry := prometheus.NewRegistry()

	err := RegisterInstanceInfo(registry, "sparrow.example.com", nil)
	require.NoError(t, err, "RegisterInstanceInfo() with nil metadata should succeed")

	metrics, err := registry.Gather()
	require.NoError(t, err, "Gather() should succeed")

	wantLabels := map[string]string{
		"instance_name": "sparrow.example.com",
	}

	assertMetricsContainLabels(t, metrics, wantLabels)
}

func TestRegisterInstanceInfo_doubleRegistration(t *testing.T) {
	registry := prometheus.NewRegistry()

	err := RegisterInstanceInfo(registry, "sparrow.example.com", map[string]string{
		"team_name":  "team-a",
		"team_email": "team-a@example.com",
		"platform":   "k8s-prod",
	})
	require.NoError(t, err, "first RegisterInstanceInfo() should succeed")

	err2 := RegisterInstanceInfo(registry, "other.example.com", map[string]string{
		"team_name":  "team-b",
		"team_email": "team-b@example.com",
		"platform":   "k8s-staging",
	})
	require.Error(t, err2, "second RegisterInstanceInfo() should return an error due to duplicate collector")

	assert.ErrorAs(t, err2, &prometheus.AlreadyRegisteredError{}, "expected an AlreadyRegisteredError")
}

func TestRegisterInstanceInfo_partialMetadata(t *testing.T) {
	registry := prometheus.NewRegistry()

	err := RegisterInstanceInfo(registry, "sparrow.example.com", map[string]string{
		"team_name": "platform-team",
	})
	require.NoError(t, err, "RegisterInstanceInfo() with partial metadata should succeed")

	metrics, err := registry.Gather()
	require.NoError(t, err, "Gather() should succeed")

	wantLabels := map[string]string{
		"instance_name": "sparrow.example.com",
		"team_name":     "platform-team",
	}

	assertMetricsContainLabels(t, metrics, wantLabels)
}

func assertMetricsContainLabels(t *testing.T, metrics []*io_prometheus_client.MetricFamily, wantLabels map[string]string) {
	t.Helper()
	found := false
	for _, mf := range metrics {
		if mf.GetName() != instanceInfoMetric {
			continue
		}
		found = true

		assert.Len(t, mf.GetMetric(), 1, "expected exactly one metric")

		const wantVal float64 = 1
		for _, m := range mf.GetMetric() {
			assert.Equal(t, wantVal, m.GetGauge().GetValue(), "%q metric value expected %d", instanceInfoMetric, wantVal)

			labels := make(map[string]string)
			for _, lp := range m.GetLabel() {
				labels[lp.GetName()] = lp.GetValue()
			}
			assert.Equal(t, wantLabels, labels, "want labels %v", wantLabels)
		}
	}
	assert.True(t, found, "sparrow_instance_info metric not found in registry")
}
