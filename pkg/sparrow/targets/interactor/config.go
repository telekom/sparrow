// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package interactor

import (
	"fmt"

	"github.com/telekom/sparrow/pkg/sparrow/targets/remote"
	"github.com/telekom/sparrow/pkg/sparrow/targets/remote/gitlab"
	"github.com/telekom/sparrow/pkg/sparrow/targets/remote/s3"
)

// Config contains the configuration for the remote interactor
type Config struct {
	// Gitlab contains the configuration for the gitlab interactor
	Gitlab gitlab.Config `yaml:"gitlab" mapstructure:"gitlab"`
	// S3 contains the configuration for the S3-compatible interactor
	S3 s3.Config `yaml:"s3" mapstructure:"s3"`
}

// Type defines the type of remote interactor
type Type string

const (
	Gitlab Type = "gitlab"
	S3     Type = "s3"
)

// Interactor creates a new remote interactor based on the configured type
func (t Type) Interactor(cfg *Config) (remote.Interactor, error) {
	switch t {
	case Gitlab:
		return gitlab.New(cfg.Gitlab), nil
	case S3:
		return s3.New(&cfg.S3)
	default:
		return nil, fmt.Errorf("unknown interactor type: %q", string(t))
	}
}
