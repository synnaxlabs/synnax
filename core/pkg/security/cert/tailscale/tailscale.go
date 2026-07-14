// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package tailscale provides a certificate source backed by the local tailscaled daemon.
// It lives in its own package so its dependency never enters the base cert package; add
// tailscale.Factory to the source factory list to enable it.
package tailscale

import (
	"crypto/tls"

	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
	"tailscale.com/client/local"
)

// SourceType is the configuration token selecting the tailscale source.
const SourceType = "tailscale"

// Factory builds tailscale sources.
type Factory struct{}

var _ cert.SourceFactory = Factory{}

// Type implements cert.SourceFactory.
func (Factory) Type() string { return SourceType }

// NewSource implements cert.SourceFactory.
func (Factory) NewSource(cfg cert.SourceConfig) (cert.Source, error) {
	if cfg.Cert != "" || cfg.Key != "" {
		return nil, errors.Wrap(validate.ErrValidation, "[cert] - tailscale source must not set cert or key")
	}
	return &source{}, nil
}

// source serves certificates from the local tailscaled daemon. The daemon fetches and
// caches them, so GetCertificate defers entirely to it.
type source struct{ client local.Client }

// GetCertificate implements cert.Source.
func (s *source) GetCertificate(hi *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return s.client.GetCertificate(hi)
}
