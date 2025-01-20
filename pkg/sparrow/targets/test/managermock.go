// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package managermock

import (
	"context"

	"github.com/telekom/sparrow/pkg/checks"

	"github.com/telekom/sparrow/internal/logger"
)

// MockTargetManager is a mock implementation of the TargetManager interface
type MockTargetManager struct {
	Targets []checks.GlobalTarget
}

func (m *MockTargetManager) Reconcile(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Info("MockReconcile called")
	return nil
}

func (m *MockTargetManager) Shutdown(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Info("MockShutdown called")
	return nil
}

func (m *MockTargetManager) GetTargets() []checks.GlobalTarget {
	log := logger.FromContext(context.Background())
	log.Info("MockGetTargets called, returning", "targets", len(m.Targets))
	return m.Targets
}
