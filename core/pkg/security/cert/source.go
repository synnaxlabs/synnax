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
	"io"
	"os"
	"sync"
	"time"

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

// Source type identifiers used in configuration.
const (
	SourceTypeFile      = "file"
	SourceTypeAuto      = "auto"
	SourceTypeTailscale = "tailscale"
)

// SourceConfig configures a single listener's certificate Source.
type SourceConfig struct {
	// Type selects the Source implementation. One of the SourceType constants.
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

// Constructor builds a Source from its configuration.
type Constructor func(SourceConfig) (Source, error)

var registry = map[string]Constructor{
	SourceTypeFile: newFileSource,
	SourceTypeAuto: newAutoSource,
}

// RegisterSource registers a Source constructor under the given type name. It lets a
// source living in its own package (e.g. tailscale) join the factory without the base
// package importing it.
func RegisterSource(sourceType string, c Constructor) { registry[sourceType] = c }

// NewSource builds the Source identified by cfg.Type.
func NewSource(cfg SourceConfig) (Source, error) {
	c, ok := registry[cfg.Type]
	if !ok {
		return nil, errors.Newf("unknown certificate source %q", cfg.Type)
	}
	return c(cfg)
}

// fileSource serves a certificate from PEM files, reloading them when they change so
// certificate rotation needs no restart.
type fileSource struct {
	fs       xfs.FS
	certPath string
	keyPath  string
	mu       sync.Mutex
	cached   *tls.Certificate
	certMod  time.Time
	keyMod   time.Time
}

func newFileSource(cfg SourceConfig) (Source, error) {
	if cfg.Cert == "" || cfg.Key == "" {
		return nil, errors.New("[cert] - file source requires both a cert and a key path")
	}
	fs := cfg.FS
	if fs == nil {
		fs = xfs.Default
	}
	return &fileSource{fs: fs, certPath: cfg.Cert, keyPath: cfg.Key}, nil
}

// GetCertificate implements Source.
func (s *fileSource) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	certMod, err := s.modTime(s.certPath)
	if err != nil {
		return nil, err
	}
	keyMod, err := s.modTime(s.keyPath)
	if err != nil {
		return nil, err
	}
	if s.cached != nil && certMod.Equal(s.certMod) && keyMod.Equal(s.keyMod) {
		return s.cached, nil
	}
	c, err := s.load()
	if err != nil {
		return nil, err
	}
	s.cached, s.certMod, s.keyMod = c, certMod, keyMod
	return c, nil
}

func (s *fileSource) modTime(path string) (time.Time, error) {
	info, err := s.fs.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

func (s *fileSource) load() (*tls.Certificate, error) {
	certPEM, err := s.readAll(s.certPath)
	if err != nil {
		return nil, err
	}
	keyPEM, err := s.readAll(s.keyPath)
	if err != nil {
		return nil, err
	}
	c, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *fileSource) readAll(path string) ([]byte, error) {
	f, err := s.fs.Open(path, os.O_RDONLY)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return io.ReadAll(f)
}

// autoSource self-signs a certificate from the built-in CA, deriving its SANs from the
// listener address. It generates once and caches for the node's lifetime.
type autoSource struct {
	factory *Factory
	mu      sync.Mutex
	cached  *tls.Certificate
}

func newAutoSource(cfg SourceConfig) (Source, error) {
	if cfg.Address == "" {
		return nil, errors.New("[cert] - auto source requires a listener address")
	}
	f, err := NewFactory(FactoryConfig{
		LoaderConfig: LoaderConfig{
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
	return &autoSource{factory: f}, nil
}

// GetCertificate implements Source.
func (s *autoSource) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
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
