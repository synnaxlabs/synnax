// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package tailscale registers a certificate source that serves Tailscale-provisioned
// certificates. Blank-import it to make the "tailscale" source available to the factory:
//
//	import _ "github.com/synnaxlabs/synnax/pkg/security/cert/tailscale"
package tailscale

import (
	"crypto/tls"

	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"tailscale.com/client/local"
)

func init() { cert.RegisterSource(cert.SourceTypeTailscale, newSource) }

// source serves certificates from the local tailscaled daemon. The daemon fetches and
// caches them, so GetCertificate defers entirely to it.
type source struct{ client local.Client }

func newSource(cert.SourceConfig) (cert.Source, error) { return &source{}, nil }

// GetCertificate implements cert.Source.
func (s *source) GetCertificate(hi *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return s.client.GetCertificate(hi)
}
