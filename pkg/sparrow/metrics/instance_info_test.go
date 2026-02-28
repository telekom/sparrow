// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestRegisterInstanceInfo(t *testing.T) {
	registry := prometheus.NewRegistry()

	err := RegisterInstanceInfo(registry, "sparrow.example.com", map[string]string{
		"team_name":  "platform-team",
		"team_email": "platform@example.com",
		"platform":   "k8s-prod-eu",
	})
	if err != nil {
		t.Fatalf("RegisterInstanceInfo() error = %v", err)
	}

	metrics, err := registry.Gather()
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}

	var found bool
	for _, mf := range metrics {
		if mf.GetName() == instanceInfoMetricName {
			found = true
			if len(mf.GetMetric()) != 1 {
				t.Errorf("expected 1 metric, got %d", len(mf.GetMetric()))
			}
			for _, m := range mf.GetMetric() {
				if m.GetGauge().GetValue() != 1 {
					t.Errorf("expected value 1, got %v", m.GetGauge().GetValue())
				}
				labels := make(map[string]string)
				for _, lp := range m.GetLabel() {
					labels[lp.GetName()] = lp.GetValue()
				}
				if labels["instance_name"] != "sparrow.example.com" || labels["team_name"] != "platform-team" ||
					labels["team_email"] != "platform@example.com" || labels["platform"] != "k8s-prod-eu" {
					t.Errorf("unexpected labels: %v", labels)
				}
			}
			break
		}
	}
	if !found {
		t.Error("sparrow_instance_info metric not found in registry")
	}
}

func TestRegisterInstanceInfo_emptyMetadata(t *testing.T) {
	registry := prometheus.NewRegistry()

	err := RegisterInstanceInfo(registry, "sparrow.example.com", nil)
	if err != nil {
		t.Fatalf("RegisterInstanceInfo() with empty metadata error = %v", err)
	}

	metrics, err := registry.Gather()
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}

	for _, mf := range metrics {
		if mf.GetName() == instanceInfoMetricName {
			for _, m := range mf.GetMetric() {
				labels := make(map[string]string)
				for _, lp := range m.GetLabel() {
					labels[lp.GetName()] = lp.GetValue()
				}
				if labels["instance_name"] != "sparrow.example.com" {
					t.Errorf("expected instance_name=sparrow.example.com, got %v", labels)
				}
			}
			return
		}
	}
	t.Error("sparrow_instance_info metric not found")
}

func TestRegisterInstanceInfo_doubleRegistration(t *testing.T) {
	registry := prometheus.NewRegistry()

	err := RegisterInstanceInfo(registry, "sparrow.example.com", map[string]string{
		"team_name":  "team-a",
		"team_email": "team-a@example.com",
		"platform":   "k8s-prod",
	})
	if err != nil {
		t.Fatalf("first RegisterInstanceInfo() error = %v", err)
	}

	err2 := RegisterInstanceInfo(registry, "other.example.com", map[string]string{
		"team_name":  "team-b",
		"team_email": "team-b@example.com",
		"platform":   "k8s-staging",
	})
	if err2 == nil {
		t.Fatal("expected second RegisterInstanceInfo to return an error (duplicate collector)")
	}

	var alreadyErr prometheus.AlreadyRegisteredError
	if !errors.As(err2, &alreadyErr) {
		t.Errorf("expected AlreadyRegisteredError, got %T: %v", err2, err2)
	}
}

func TestRegisterInstanceInfo_partialMetadata(t *testing.T) {
	registry := prometheus.NewRegistry()

	err := RegisterInstanceInfo(registry, "sparrow.example.com", map[string]string{
		"team_name": "platform-team",
	})
	if err != nil {
		t.Fatalf("RegisterInstanceInfo() with partial metadata error = %v", err)
	}

	metrics, err := registry.Gather()
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}

	for _, mf := range metrics {
		if mf.GetName() == instanceInfoMetricName {
			for _, m := range mf.GetMetric() {
				labels := make(map[string]string)
				for _, lp := range m.GetLabel() {
					labels[lp.GetName()] = lp.GetValue()
				}
				if labels["instance_name"] != "sparrow.example.com" || labels["team_name"] != "platform-team" {
					t.Errorf("unexpected labels (expected partial metadata): %v", labels)
				}
			}
			return
		}
	}
	t.Error("sparrow_instance_info metric not found")
}
