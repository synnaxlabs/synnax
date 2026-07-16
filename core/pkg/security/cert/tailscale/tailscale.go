// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package tailscale provides a certificate source backed by the local tailscaled daemon.
// It lives in its own package so its dependency never enters the base cert package.
package tailscale

import (
	"crypto/tls"

	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
	"tailscale.com/client/local"
)

// SourceType is the configuration token selecting the Tailscale source.
const SourceType = "tailscale"

// DefaultClient returns a client backed by the local tailscaled daemon. It is the client
// NewSource expects in production; tests substitute their own.
func DefaultClient() cert.Source { return &local.Client{} }

// NewSource builds a Tailscale source that resolves certificates for host through client. It
// returns validate.ErrValidation if host is empty, since tailscaled resolves
// certificates by FQDN.
func NewSource(client cert.Source, host string) (cert.Source, error) {
	if host == "" {
		return nil, errors.Wrap(
			validate.ErrValidation,
			"Tailscale certificate source requires a listener host; tailscaled resolves certificates by FQDN",
		)
	}
	return &source{client: client, host: host}, nil
}

// source serves certificates from the local tailscaled daemon. The daemon fetches and
// caches them, so GetCertificate defers entirely to it.
type source struct {
	client cert.Source
	host   string
}

// GetCertificate implements cert.Source. tailscaled selects the certificate by the
// handshake SNI; when a client omits it, fall back to the listener's configured host so
// the daemon still serves the right name. A non-empty SNI is honored as-is: the cert
// comes from a public CA keyed to the tailnet FQDN, so the client must dial that name
// for verification to pass; rewriting its SNI server-side cannot change that.
func (s *source) GetCertificate(hi *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if hi.ServerName == "" {
		hi.ServerName = s.host
	}
	return s.client.GetCertificate(hi)
}
