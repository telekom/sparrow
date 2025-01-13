// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package interactor

import (
	"github.com/caas-team/sparrow/pkg/sparrow/targets/remote"
	"github.com/caas-team/sparrow/pkg/sparrow/targets/remote/gitlab"
)

// Config contains the configuration for the remote interactor
type Config struct {
	// Gitlab contains the configuration for the gitlab interactor
	Gitlab gitlab.Config `yaml:"gitlab" mapstructure:"gitlab"`
}

type Type string

const (
	Gitlab Type = "gitlab"
)

func (t Type) Interactor(cfg *Config) remote.Interactor {
	switch t { //nolint:gocritic // won't be a single switch case with the implementation of #66
	case Gitlab:
		return gitlab.New(cfg.Gitlab)
	}
	return nil
}
