// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package dns

import (
	"context"
	"net"
)

//go:generate go tool moq -out resolver_moq.go . Resolver
type Resolver interface {
	LookupAddr(ctx context.Context, addr string) ([]string, error)
	LookupHost(ctx context.Context, addr string) ([]string, error)
	SetDialer(d *net.Dialer)
}

type resolver struct {
	*net.Resolver
}

func NewResolver() Resolver {
	return &resolver{
		Resolver: &net.Resolver{
			// We need to set this so the custom dialer is used
			PreferGo: true,
		},
	}
}

func (r *resolver) SetDialer(d *net.Dialer) {
	r.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
		return d.DialContext(ctx, network, address)
	}
}
