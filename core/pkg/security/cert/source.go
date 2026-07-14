// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cert

import (
	"crypto/tls"

	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
)

// Source resolves the certificate a listener presents during a TLS handshake. It matches
// the crypto/tls GetCertificate callback, so a Source plugs directly into a listener's
// tls.Config.
type Source interface {
	// GetCertificate returns the certificate for the given handshake.
	GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error)
}

// SourceConfig configures a single listener's certificate Source.
type SourceConfig struct {
	// Type selects the SourceFactory that builds the source.
	Type string
	// FS is the filesystem the file and auto sources read from.
	FS xfs.FS
	// Cert and Key are the PEM paths served by a file source.
	Cert string
	Key  string
	// Address is the listener address. The auto source derives its SANs from it.
	Address address.Address
	// CertsDir and the CA paths give the auto source access to the built-in CA.
	CertsDir   string
	CAKeyPath  string
	CACertPath string
	// KeySize is the key size the auto source generates.
	KeySize int
}

// SourceFactory builds the Source for a single listener. Each concrete source ships its
// own factory declaring the Type it answers to, so the base package never enumerates the
// available sources. The set of factories is assembled at the composition root and passed
// to Resolve.
type SourceFactory interface {
	// Type is the configuration token this factory answers to.
	Type() string
	// NewSource validates cfg and builds the source. It is called only when cfg.Type
	// matches Type.
	NewSource(cfg SourceConfig) (Source, error)
}

// Resolve builds the source cfg selects by matching cfg.Type against the given factories.
func Resolve(factories []SourceFactory, cfg SourceConfig) (Source, error) {
	for _, f := range factories {
		if f.Type() == cfg.Type {
			return f.NewSource(cfg)
		}
	}
	return nil, errors.Newf("unknown certificate source %q", cfg.Type)
}
