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

// Factory builds auto sources.
type Factory struct{}

var _ cert.SourceFactory = Factory{}

// Type implements cert.SourceFactory.
func (Factory) Type() string { return SourceType }

// NewSource implements cert.SourceFactory.
func (Factory) NewSource(cfg cert.SourceConfig) (cert.Source, error) {
	if cfg.Address == "" {
		return nil, errors.Wrap(validate.ErrValidation, "[cert] - auto source requires a listener address")
	}
	if cfg.Cert != "" || cfg.Key != "" {
		return nil, errors.Wrap(validate.ErrValidation, "[cert] - auto source must not set cert or key")
	}
	f, err := cert.NewFactory(cert.FactoryConfig{
		LoaderConfig: cert.LoaderConfig{
			FS:         cfg.FS,
			CertsDir:   cfg.CertsDir,
			CAKeyPath:  cfg.CAKeyPath,
			CACertPath: cfg.CACertPath,
		},
		Hosts:         []address.Address{cfg.Address},
		KeySize:       cfg.KeySize,
		AllowKeyReuse: new(true),
	})
	if err != nil {
		return nil, err
	}
	return &source{factory: f}, nil
}

// source self-signs a certificate from the built-in CA, deriving its SANs from the
// listener address. It generates once and caches for the node's lifetime.
type source struct {
	factory *cert.Factory
	mu      sync.Mutex
	cached  *tls.Certificate
}

// GetCertificate implements cert.Source.
func (s *source) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cached != nil {
		return s.cached, nil
	}
	if err := s.factory.CreateCAPairIfMissing(); err != nil {
		return nil, err
	}
	c, err := s.factory.CreateNodeTLS()
	if err != nil {
		return nil, err
	}
	s.cached = c
	return c, nil
}
