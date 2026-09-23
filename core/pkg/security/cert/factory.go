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
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"net"
	"os"

	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	xpem "github.com/synnaxlabs/x/encoding/pem"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// nextSuffix names the files a replacement is written to before it is renamed over the
// pair in use, so a failed signature or write never destroys the pair on disk.
const nextSuffix = ".next"

// FactoryConfig is the configuration for creating a new Factory.
type FactoryConfig struct {
	// AllowKeyReuse allows the CA key to be reused if it already exists.
	AllowKeyReuse *bool
	LoaderConfig
	// Hosts is the list of hosts to use for the node certificate.
	Hosts []address.Address
	// KeySize is the size of the private key to generate. It sizes RSA keys only.
	KeySize int
	// KeyAlgorithm is the algorithm the generated private keys use.
	KeyAlgorithm KeyAlgorithm
}

var (
	_ config.Config[FactoryConfig] = FactoryConfig{}
	// DefaultFactoryConfig is the default configuration for a Factory.
	DefaultFactoryConfig = FactoryConfig{
		LoaderConfig:  DefaultLoaderConfig,
		KeySize:       2048,
		KeyAlgorithm:  KeyAlgorithmRSA,
		AllowKeyReuse: new(false),
	}
)

// Override implements [config.Config].
func (f FactoryConfig) Override(other FactoryConfig) FactoryConfig {
	f.KeySize = override.Numeric(f.KeySize, other.KeySize)
	f.KeyAlgorithm = override.String(f.KeyAlgorithm, other.KeyAlgorithm)
	f.Hosts = override.Slice(f.Hosts, other.Hosts)
	f.AllowKeyReuse = override.Nil(f.AllowKeyReuse, other.AllowKeyReuse)
	f.LoaderConfig = f.LoaderConfig.Override(other.LoaderConfig)
	return f
}

// Validate implements [config.Config].
func (f FactoryConfig) Validate() error {
	v := validate.New("cert.factory")
	v.Positive("key_size", f.KeySize)
	v.Exec(f.KeyAlgorithm.validate)
	v.NotNil("allow_key_reuse", f.AllowKeyReuse)
	v.Exec(f.LoaderConfig.Validate)
	return v.Error()
}

// Factory generates self-signed certificates.
type Factory struct {
	// Loader reads back the certificates and keys the Factory writes.
	Loader Loader
	cfg    FactoryConfig
}

// NewFactory creates a new Factory.
func NewFactory(configs ...FactoryConfig) (*Factory, error) {
	cfg, err := config.New(DefaultFactoryConfig, configs...)
	if err != nil {
		return nil, err
	}
	loader, err := NewLoader(cfg.LoaderConfig)
	if err != nil {
		return nil, err
	}
	cfg.LoaderConfig = loader.cfg
	cfg.FS = loader.cfg.FS
	return &Factory{cfg: cfg, Loader: *loader}, nil
}

// Config returns the configuration the Factory was built with. The returned copy is
// inert: changing it does not change what the Factory generates.
func (f *Factory) Config() FactoryConfig { return f.cfg }

// CreateCAPair creates a new CA certificate and its private key.
func (f *Factory) CreateCAPair() error {
	key, err := f.caKey()
	if err != nil {
		return err
	}
	base, err := newBasex509()
	if err != nil {
		return err
	}
	base.BasicConstraintsValid = true
	base.IsCA = true
	base.MaxPathLen = 1
	base.KeyUsage |= x509.KeyUsageCertSign
	base.KeyUsage |= x509.KeyUsageContentCommitment
	b, err := x509.CreateCertificate(rand.Reader, base, base, key.Public(), key)
	if err != nil {
		return err
	}
	return f.writePEM(f.cfg.CACertPath, xpem.FromCertBytes(b) /* multi */, true)
}

// caKey generates the CA private key and writes it to CAKeyPath, or reads back the key
// already there when AllowKeyReuse permits it.
func (f *Factory) caKey() (crypto.Signer, error) {
	exists, err := f.cfg.FS.Exists(f.cfg.CACertPath)
	if err != nil {
		return nil, err
	}
	if !exists {
		key, err := f.cfg.KeyAlgorithm.GenerateKey(f.cfg.KeySize)
		if err != nil {
			return nil, err
		}
		p, err := xpem.FromPrivateKey(key)
		if err != nil {
			return nil, err
		}
		return key, f.writePEM(f.cfg.CAKeyPath, p /* multi */, false)
	}
	if !*f.cfg.AllowKeyReuse {
		return nil, errors.Newf(
			"CA key %s already exists, but reuse is not allowed",
			f.cfg.CAKeyPath,
		)
	}
	p, err := f.readPEM(f.cfg.CAKeyPath)
	if err != nil {
		return nil, err
	}
	key, err := xpem.ToPrivateKey(p)
	if err != nil {
		return nil, err
	}
	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, errors.Newf("CA key %s cannot sign certificates", f.cfg.CAKeyPath)
	}
	return signer, nil
}

// CreateTokenKeyIfMissing creates the key a Core signs authentication tokens with, if
// it does not already exist. A Core whose certificate uses a post-quantum algorithm
// needs it, because ML-DSA has no JWT signing method. Ed25519 is fixed rather than
// configurable: the key never appears in a certificate, so it needs no agreement with a
// peer, and EdDSA keeps tokens small.
func (f *Factory) CreateTokenKeyIfMissing() error {
	exists, err := f.cfg.FS.Exists(f.cfg.TokenKeyPath)
	if err != nil || exists {
		return err
	}
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	p, err := xpem.FromPrivateKey(key)
	if err != nil {
		return err
	}
	return f.writePEM(f.cfg.TokenKeyPath, p /* multi */, false)
}

// CreateCAPairIfMissing creates the CA certificate and its private key if the
// certificate does not already exist. An existing pair is left untouched.
func (f *Factory) CreateCAPairIfMissing() error {
	exists, err := f.cfg.FS.Exists(f.cfg.CACertPath)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return f.CreateCAPair()
}

// CreateAll creates everything a Core needs to serve securely: the CA pair if it does
// not exist, the token signing key if the configured algorithm cannot sign JWTs, and
// the node pair if it is missing or does not cover every host in Hosts. Callers that do
// not own the files on disk must not use it.
func (f *Factory) CreateAll() error {
	if err := f.CreateCAPairIfMissing(); err != nil {
		return err
	}
	if f.cfg.KeyAlgorithm.PostQuantum() {
		if err := f.CreateTokenKeyIfMissing(); err != nil {
			return err
		}
	}
	return f.CreateNodePairIfStale()
}

// CreateNodePairIfStale creates the node certificate and its private key if they do not
// exist, and replaces them when the existing certificate does not cover every host in
// Hosts. A covering certificate is left untouched. Callers that do not own the files on
// disk must not use it.
func (f *Factory) CreateNodePairIfStale() error {
	exists, err := f.cfg.FS.Exists(f.cfg.NodeCertPath)
	if err != nil {
		return err
	}
	if !exists {
		return f.CreateNodePair()
	}
	c, _, err := f.Loader.LoadNodePair()
	if err != nil {
		return err
	}
	uncovered := f.uncoveredHosts(c)
	if len(uncovered) == 0 {
		return nil
	}
	f.cfg.L.Info(
		"replacing node certificate and key: they do not cover every configured listener",
		zap.Strings("uncovered_hosts", uncovered),
		zap.String("cert", f.cfg.AbsoluteNodeCertPath()),
		zap.String("key", f.cfg.AbsoluteNodeKeyPath()),
	)
	certPath, keyPath := f.cfg.NodeCertPath+nextSuffix, f.cfg.NodeKeyPath+nextSuffix
	if err = f.cfg.FS.Remove(certPath); err != nil {
		return err
	}
	if err = f.cfg.FS.Remove(keyPath); err != nil {
		return err
	}
	if err = f.writeNodePair(certPath, keyPath); err != nil {
		return err
	}
	if err = f.cfg.FS.Rename(keyPath, f.cfg.NodeKeyPath); err != nil {
		return err
	}
	return f.cfg.FS.Rename(certPath, f.cfg.NodeCertPath)
}

// uncoveredHosts returns the hosts in Hosts that c is not valid for, using the same
// matcher a client applies during a handshake.
func (f *Factory) uncoveredHosts(c *x509.Certificate) []string {
	var uncovered []string
	for _, h := range f.cfg.Hosts {
		if c.VerifyHostname(h.Host()) != nil {
			uncovered = append(uncovered, h.Host())
		}
	}
	return uncovered
}

// CreateNodePair creates a new node certificate and its private key.
func (f *Factory) CreateNodePair() error {
	return f.writeNodePair(f.cfg.NodeCertPath, f.cfg.NodeKeyPath)
}

// writeNodePair generates a key, signs a certificate for Hosts with the CA, and writes
// both to the given paths.
func (f *Factory) writeNodePair(certPath, keyPath string) error {
	nodeKey, err := f.cfg.KeyAlgorithm.GenerateKey(f.cfg.KeySize)
	if err != nil {
		return err
	}
	b, err := f.signNodeCert(nodeKey, f.cfg.Hosts)
	if err != nil {
		return err
	}
	keyP, err := xpem.FromPrivateKey(nodeKey)
	if err != nil {
		return err
	}
	if err = f.writePEM(keyPath, keyP, false); err != nil {
		return err
	}
	return f.writePEM(certPath, xpem.FromCertBytes(b) /* multi */, false)
}

// SignNodeCert signs an in-memory node certificate for the given hosts using the CA,
// without touching the filesystem. The caller supplies the hosts because they vary per
// listener; the factory supplies the CA the certificate chains to.
func (f *Factory) SignNodeCert(hosts []address.Address) (*tls.Certificate, error) {
	nodeKey, err := f.cfg.KeyAlgorithm.GenerateKey(f.cfg.KeySize)
	if err != nil {
		return nil, err
	}
	b, err := f.signNodeCert(nodeKey, hosts)
	if err != nil {
		return nil, err
	}
	return &tls.Certificate{Certificate: [][]byte{b}, PrivateKey: nodeKey}, nil
}

func (f *Factory) signNodeCert(
	nodeKey crypto.Signer,
	hosts []address.Address,
) ([]byte, error) {
	ca, caPrivate, err := f.Loader.LoadCAPair()
	if err != nil {
		return nil, err
	}
	if len(hosts) == 0 {
		return nil, errors.Wrap(validate.ErrValidation, "no hosts provided")
	}
	base, err := newBasex509()
	if err != nil {
		return nil, err
	}
	base.Subject = pkix.Name{CommonName: "Synnax Node"}
	base.ExtKeyUsage = []x509.ExtKeyUsage{
		x509.ExtKeyUsageServerAuth,
		x509.ExtKeyUsageClientAuth,
	}
	for _, h := range hosts {
		if ip := net.ParseIP(h.Host()); ip != nil {
			base.IPAddresses = append(base.IPAddresses, ip)
		} else {
			base.DNSNames = append(base.DNSNames, h.Host())
		}
	}
	return x509.CreateCertificate(rand.Reader, base, ca, nodeKey.Public(), caPrivate)
}

func (f *Factory) readPEM(p string) (b *pem.Block, err error) {
	return b, f.withFile(p, f.readFlag(), func(file xfs.File) error {
		b, err = xpem.Read(file)
		return err
	})
}

func (f *Factory) writePEM(p string, block *pem.Block, multi bool) error {
	return f.withFile(p, f.writeFlag(), func(file xfs.File) error {
		blocks, err := xpem.ReadMany(file)
		if err != nil {
			return err
		}
		if len(blocks) > 0 && !multi {
			return errors.Newf(
				"file %s already contains a PEM block, and multi is false",
				p,
			)
		}
		blocks = append(blocks, block)
		return xpem.Write(file, blocks...)
	})
}

func (f *Factory) withFile(p string, flag int, fn func(fs xfs.File) error) (err error) {
	file, err := f.cfg.FS.Open(p, flag)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Combine(err, file.Close())
	}()
	err = fn(file)
	return err
}

func (f *Factory) writeFlag() int {
	if *f.cfg.AllowKeyReuse {
		return os.O_CREATE | os.O_RDWR
	}
	return os.O_CREATE | os.O_RDWR | os.O_EXCL
}

func (f *Factory) readFlag() int { return os.O_RDONLY }
