// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cert

import (
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
	"io"
	"io/fs"
	"os"
)

// LoaderConfig is the configuration for creating a new Loader.
type LoaderConfig struct {
	alamos.Instrumentation
	// CertsDir is the directory where the certificates are stored.
	CertsDir string
	// CAKeyPath is the path to the CA private key. This is relative to CertsDir.
	CAKeyPath string
	// CACertPath is the path to the CA certificate. This is relative to CertsDir.
	CACertPath string
	// NodeKeyPath is the path to the node private key. This is relative to CertsDir.
	NodeKeyPath string
	// NodeCertPath is the path to the node certificate. This is relative to CertsDir.
	NodeCertPath string
	// FS is the filesystem to use.
	FS xfs.FS
}

var (
	_ config.Config[LoaderConfig] = LoaderConfig{}
	// DefaultLoaderConfig is the default configuration for a Loader.
	DefaultLoaderConfig = LoaderConfig{
		CertsDir:     "/usr/local/synnax/certs",
		CAKeyPath:    "ca.key",
		CACertPath:   "ca.crt",
		NodeKeyPath:  "node.key",
		NodeCertPath: "node.crt",
		FS:           xfs.Default,
	}
)

func (l LoaderConfig) AbsoluteCAKeyPath() string {
	return l.CertsDir + "/" + l.CAKeyPath
}

func (l LoaderConfig) AbsoluteCACertPath() string {
	return l.CertsDir + "/" + l.CACertPath
}

func (l LoaderConfig) AbsoluteNodeKeyPath() string {
	return l.CertsDir + "/" + l.NodeKeyPath
}

func (l LoaderConfig) AbsoluteNodeCertPath() string {
	return l.CertsDir + "/" + l.NodeCertPath
}

// Override implements Properties.
func (l LoaderConfig) Override(other LoaderConfig) LoaderConfig {
	l.CertsDir = override.String(l.CertsDir, other.CertsDir)
	l.CAKeyPath = override.String(l.CAKeyPath, other.CAKeyPath)
	l.CACertPath = override.String(l.CACertPath, other.CACertPath)
	l.NodeKeyPath = override.String(l.NodeKeyPath, other.NodeKeyPath)
	l.NodeCertPath = override.String(l.NodeCertPath, other.NodeCertPath)
	l.FS = override.Nil(l.FS, other.FS)
	l.Instrumentation = override.Zero(l.Instrumentation, other.Instrumentation)
	return l
}

// Validate implements Properties.
func (l LoaderConfig) Validate() error {
	v := validate.New("cert.loader")
	validate.NotEmptyString(v, "certs_dir", l.CertsDir)
	validate.NotEmptyString(v, "ca_key_path", l.CAKeyPath)
	validate.NotEmptyString(v, "ca_cert_path", l.CACertPath)
	validate.NotEmptyString(v, "node_key_path", l.NodeKeyPath)
	validate.NotEmptyString(v, "node_cert_path", l.NodeCertPath)
	validate.NotNil(v, "fs", l.FS)
	return v.Error()
}

// Loader is a certificate Loader.
type Loader struct{ LoaderConfig }

// NewLoader creates a new Loader using the given configuration. Returns an error if the
// configuration is invalid. If the directory at LoaderConfig.CertsDir does not exist,
// it is created.
func NewLoader(configs ...LoaderConfig) (*Loader, error) {
	cfg, err := config.New(DefaultLoaderConfig, configs...)
	if err != nil {
		return nil, err
	}
	cfg.FS, err = cfg.FS.Sub(cfg.CertsDir)
	return &Loader{LoaderConfig: cfg}, err
}

// LoadCAPair loads the CA certificate and its private key. If multiple
// certificates are found in the CA certificate file, the first one is used.
func (l *Loader) LoadCAPair() (c *x509.Certificate, k crypto.PrivateKey, err error) {
	c, k, err = l.loadX509(l.CACertPath, l.CAKeyPath)
	if errors.Is(err, fs.ErrNotExist) {
		err = errors.Wrapf(err, "CA certificate not found")
	}
	return
}

// LoadCAs loads all CA certificates from the CA certificate file.
func (l *Loader) LoadCAs() ([]*x509.Certificate, error) {
	certBytes, err := l.readAll(l.CACertPath)
	if err != nil {
		return nil, err
	}
	var (
		certs []*x509.Certificate
		block *pem.Block
	)
	for len(certBytes) > 0 {
		block, certBytes = pem.Decode(certBytes)
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		certs = append(certs, cert)
	}
	return certs, nil
}

// LoadNodePair loads the node certificate and its private key.
func (l *Loader) LoadNodePair() (c *x509.Certificate, k crypto.PrivateKey, err error) {
	c, k, err = l.loadX509(l.NodeCertPath, l.NodeKeyPath)
	if errors.Is(err, fs.ErrNotExist) {
		err = errors.Wrapf(err, "node certificate not found")
	}
	return
}

// LoadNodeTLS loads the node TLS certificate.
func (l *Loader) LoadNodeTLS() (c *tls.Certificate, err error) {
	c, err = l.loadTLS(l.NodeCertPath, l.NodeKeyPath)
	if errors.Is(err, fs.ErrNotExist) {
		err = errors.Wrapf(err, "node certificate not found")
	}
	return
}

func (l *Loader) loadX509(certPath, keyPath string) (*x509.Certificate, crypto.PrivateKey, error) {
	c, err := l.loadTLS(certPath, keyPath)
	if err != nil {
		return nil, nil, err
	}
	certParsed, err := x509.ParseCertificate(c.Certificate[0])
	if err != nil {
		return nil, nil, err
	}
	return certParsed, c.PrivateKey, nil
}

func (l *Loader) loadTLS(certPath, keyPath string) (*tls.Certificate, error) {
	certPEM, err := l.readAll(certPath)
	if err != nil {
		return nil, err
	}
	keyPEM, err := l.readAll(keyPath)
	if err != nil {
		return nil, err
	}
	c, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}
	return &c, err
}

func (l *Loader) readAll(path string) ([]byte, error) {
	f, err := l.FS.Open(path, os.O_RDONLY)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			l.L.Error("failed to close file", zap.String("path", path), zap.Error(err))
		}
	}()
	return io.ReadAll(f)
}
