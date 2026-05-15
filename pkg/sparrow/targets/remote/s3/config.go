// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package s3

import (
	"context"
	"errors"
	"fmt"

	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/telekom/sparrow/internal/logger"
)

var (
	// ErrMissingEndpoint is returned when the S3 endpoint is not configured
	ErrMissingEndpoint = errors.New("s3: endpoint is required")
	// ErrMissingBucket is returned when the S3 bucket is not configured
	ErrMissingBucket = errors.New("s3: bucket is required")
	// ErrMissingAccessKey is returned when static auth is used without an access key
	ErrMissingAccessKey = errors.New("s3: accessKeyID is required for static auth")
	// ErrMissingSecretKey is returned when static auth is used without a secret key
	ErrMissingSecretKey = errors.New("s3: secretAccessKey is required for static auth")
	// ErrMissingRoleARN is returned when OIDC auth is used without a role ARN
	ErrMissingRoleARN = errors.New("s3: roleARN is required for OIDC auth")
	// ErrMissingTokenPath is returned when OIDC auth is used without a token path
	ErrMissingTokenPath = errors.New("s3: tokenPath is required for OIDC auth")
	// ErrUnknownProvider is returned when an unsupported auth provider is configured
	ErrUnknownProvider = errors.New("s3: unknown auth provider")
)

// Config contains the configuration for the S3-compatible interactor
type Config struct {
	// Endpoint is the S3-compatible endpoint URL (e.g. "s3.amazonaws.com", "minio.internal:9000")
	Endpoint string `yaml:"endpoint" mapstructure:"endpoint"`
	// Bucket is the name of the S3 bucket to store target files
	Bucket string `yaml:"bucket" mapstructure:"bucket"`
	// Region is the S3 region (defaults to "eu-central-1")
	Region string `yaml:"region" mapstructure:"region"`
	// Prefix is an optional key prefix for target files within the bucket
	Prefix string `yaml:"prefix" mapstructure:"prefix"`
	// UseSSL enables HTTPS for the S3 connection (defaults to true)
	UseSSL *bool `yaml:"useSSL" mapstructure:"useSSL"`

	// Auth contains the authentication configuration
	Auth AuthConfig `yaml:"auth" mapstructure:"auth"`
}

// Validate checks the S3 configuration for required fields and consistency
func (c *Config) Validate(ctx context.Context) error {
	log := logger.FromContext(ctx)

	if c.Endpoint == "" {
		log.Error("S3 endpoint is required")
		return ErrMissingEndpoint
	}
	if c.Bucket == "" {
		log.Error("S3 bucket is required")
		return ErrMissingBucket
	}

	return c.Auth.validate(ctx)
}

// CredentialProvider identifies the authentication method for S3
type CredentialProvider string

const (
	credentialStatic CredentialProvider = "static"
	credentialOIDC   CredentialProvider = "oidc"
)

// AuthConfig contains authentication settings for the S3 interactor
type AuthConfig struct {
	// Provider selects the authentication method: "static" or "oidc"
	Provider CredentialProvider `yaml:"provider" mapstructure:"provider"`
	// Static contains static credential configuration
	Static StaticAuthConfig `yaml:"static" mapstructure:"static"`
	// OIDC contains OIDC/WIF credential configuration (not yet implemented)
	OIDC OIDCAuthConfig `yaml:"oidc" mapstructure:"oidc"`
}

// validate checks the auth configuration for required fields
func (a *AuthConfig) validate(ctx context.Context) error {
	log := logger.FromContext(ctx)

	switch a.Provider {
	case credentialStatic, "":
		if a.Static.AccessKeyID == "" {
			log.Error("S3 accessKeyID is required for static auth")
			return ErrMissingAccessKey
		}
		if a.Static.SecretAccessKey == "" {
			log.Error("S3 secretAccessKey is required for static auth")
			return ErrMissingSecretKey
		}
		return nil
	case credentialOIDC:
		if a.OIDC.RoleARN == "" {
			log.Error("S3 roleARN is required for OIDC auth")
			return ErrMissingRoleARN
		}
		if a.OIDC.TokenPath == "" {
			log.Error("S3 tokenPath is required for OIDC auth")
			return ErrMissingTokenPath
		}
		return nil
	default:
		log.Error("Unknown S3 auth provider", "provider", a.Provider)
		return fmt.Errorf("%w: %q", ErrUnknownProvider, a.Provider)
	}
}

// newCredentials builds the MinIO credentials object based on the configured provider
func (a *AuthConfig) newCredentials() (*credentials.Credentials, error) {
	switch a.Provider {
	case credentialStatic, "":
		return credentials.NewStaticV4(
			a.Static.AccessKeyID,
			a.Static.SecretAccessKey,
			a.Static.SessionToken,
		), nil
	case credentialOIDC:
		return nil, ErrOIDCNotImplemented
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownProvider, a.Provider)
	}
}

// StaticAuthConfig contains static S3 credentials
type StaticAuthConfig struct {
	// AccessKeyID is the S3 access key
	AccessKeyID string `yaml:"accessKeyID" mapstructure:"accessKeyID"`
	// SecretAccessKey is the S3 secret key
	SecretAccessKey string `yaml:"secretAccessKey" mapstructure:"secretAccessKey"`
	// SessionToken is an optional session token for temporary credentials
	SessionToken string `yaml:"sessionToken" mapstructure:"sessionToken"`
}

// OIDCAuthConfig contains OIDC/Workload Identity Federation settings
type OIDCAuthConfig struct {
	// TokenPath is the file path to read the OIDC token from (e.g. K8s projected volume)
	TokenPath string `yaml:"tokenPath" mapstructure:"tokenPath"`
	// RoleARN is the role to assume via STS
	RoleARN string `yaml:"roleARN" mapstructure:"roleARN"`
	// STSEndpoint is the STS endpoint for token exchange (optional, defaults to AWS STS)
	STSEndpoint string `yaml:"stsEndpoint" mapstructure:"stsEndpoint"`
}
