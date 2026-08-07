// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package auto provides a certificate source that self-signs from the built-in CA,
// deriving its SANs from the listener address.
package auto

import (
	"crypto/tls"
	"sync"

	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// SourceType is the configuration token selecting the auto source.
const SourceType = "auto"

// NewSource builds an auto source that self-signs a certificate for host from ca's
// built-in CA. It returns validate.ErrValidation if host is empty.
func NewSource(ca *cert.Factory, host address.Address) (cert.Source, error) {
	if host == "" {
		return nil, errors.Wrap(
			validate.ErrValidation,
			"auto source requires a listener address",
		)
	}
	return &source{ca: ca, host: host}, nil
}

var _ cert.Source = (*source)(nil)

// source self-signs a certificate for its listener's address from the built-in CA. It
// signs once and caches for the node's lifetime.
type source struct {
	ca     *cert.Factory
	host   address.Address
	mu     sync.Mutex
	cached *tls.Certificate
}

// GetCertificate implements cert.Source.
func (s *source) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cached != nil {
		return s.cached, nil
	}
	if err := s.ca.CreateCAPairIfMissing(); err != nil {
		return nil, err
	}
	c, err := s.ca.SignNodeCert([]address.Address{s.host})
	if err != nil {
		return nil, err
	}
	s.cached = c
	return c, nil
}
