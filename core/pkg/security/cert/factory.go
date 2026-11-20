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
	"bytes"
	"crypto"
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

var _ config.Config[FactoryConfig] = FactoryConfig{}

// DefaultFactoryConfig returns the default configuration for a Factory.
func DefaultFactoryConfig() FactoryConfig {
	return FactoryConfig{
		LoaderConfig:  DefaultLoaderConfig,
		KeySize:       2048,
		KeyAlgorithm:  KeyAlgorithmRSA,
		AllowKeyReuse: new(false),
	}
}

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
	cfg, err := config.New(DefaultFactoryConfig(), configs...)
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
	b, err := signCACert(key)
	if err != nil {
		return err
	}
	return f.writePEM(f.cfg.CACertPath, xpem.FromCertBytes(b) /* multi */, true)
}

func signCACert(key crypto.Signer) ([]byte, error) {
	base, err := newBasex509()
	if err != nil {
		return nil, err
	}
	base.BasicConstraintsValid = true
	base.IsCA = true
	base.MaxPathLen = 1
	base.KeyUsage |= x509.KeyUsageCertSign
	base.KeyUsage |= x509.KeyUsageContentCommitment
	return x509.CreateCertificate(rand.Reader, base, base, key.Public(), key)
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
	return f.readCAKey()
}

func (f *Factory) readCAKey() (crypto.Signer, error) {
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

// CreateCAPairIfMissing creates the CA certificate and its private key if the
// certificate does not exist. An existing certificate is left untouched, even without
// its key. When AllowKeyReuse is set, a new certificate is signed with the existing
// key.
func (f *Factory) CreateCAPairIfMissing() error {
	exists, err := f.cfg.FS.Exists(f.cfg.CACertPath)
	if err != nil || exists {
		return err
	}
	keyExists, err := f.cfg.FS.Exists(f.cfg.CAKeyPath)
	if err != nil {
		return err
	}
	if keyExists && *f.cfg.AllowKeyReuse {
		key, rErr := f.readCAKey()
		if rErr != nil {
			return rErr
		}
		b, sErr := signCACert(key)
		if sErr != nil {
			return sErr
		}
		return f.writePEM(f.cfg.CACertPath, xpem.FromCertBytes(b) /* multi */, true)
	}
	return f.replacePair(f.cfg.CACertPath, f.cfg.CAKeyPath, f.writeCAPair)
}

func (f *Factory) writeCAPair(certPath, keyPath string) error {
	key, err := f.cfg.KeyAlgorithm.GenerateKey(f.cfg.KeySize)
	if err != nil {
		return err
	}
	b, err := signCACert(key)
	if err != nil {
		return err
	}
	return f.writeKeyAndCert(certPath, keyPath, key, b)
}

func (f *Factory) pairExists(certPath, keyPath string) (bool, error) {
	certExists, err := f.cfg.FS.Exists(certPath)
	if err != nil || !certExists {
		return false, err
	}
	return f.cfg.FS.Exists(keyPath)
}

// CreateAll creates everything a Core needs to serve securely: the CA pair if it does
// not exist, and the node pair if CreateNodePairIfStale finds it stale. Callers that do
// not own the files on disk must not use it.
func (f *Factory) CreateAll() error {
	if err := f.CreateCAPairIfMissing(); err != nil {
		return err
	}
	return f.CreateNodePairIfStale()
}

// CreateNodePairIfStale creates the node certificate and its private key if either does
// not exist, and replaces them when the existing certificate is not a leaf the CA
// signed or does not cover every host in Hosts. Any other certificate is left
// untouched. Callers that do not own the files on disk must not use it.
func (f *Factory) CreateNodePairIfStale() error {
	exists, err := f.pairExists(f.cfg.NodeCertPath, f.cfg.NodeKeyPath)
	if err != nil {
		return err
	}
	if !exists {
		return f.replaceNodePair()
	}
	c, _, err := f.Loader.LoadNodePair()
	if err != nil {
		return err
	}
	cas, err := f.Loader.LoadCAs()
	if err != nil {
		return err
	}
	if err = checkIssuedBy(c, cas); err != nil {
		f.cfg.L.Warn(
			"replacing node certificate and key: they do not chain to the CA",
			zap.Error(err),
			zap.String("cert", f.cfg.AbsoluteNodeCertPath()),
			zap.String("key", f.cfg.AbsoluteNodeKeyPath()),
		)
		if rErr := f.replaceNodePair(); rErr != nil {
			return errors.Wrapf(
				rErr,
				"node certificate %s does not chain to the CA (%v), and replacing it "+
					"failed. Restore the CA key at %s, or replace the node certificate "+
					"and key by hand",
				f.cfg.AbsoluteNodeCertPath(),
				err,
				f.cfg.AbsoluteCAKeyPath(),
			)
		}
		return nil
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
	return f.replaceNodePair()
}

// checkIssuedBy returns an error unless one of cas signed c and c is not self-issued.
// OpenSSL reads a leaf whose subject matches its issuer as self-signed and rejects it,
// even when a trusted CA signed it.
func checkIssuedBy(c *x509.Certificate, cas []*x509.Certificate) error {
	if bytes.Equal(c.RawSubject, c.RawIssuer) {
		return errors.Newf("certificate subject %s matches its issuer", c.Subject)
	}
	for _, ca := range cas {
		if c.CheckSignatureFrom(ca) == nil {
			return nil
		}
	}
	return errors.New("no CA certificate signed the certificate")
}

func (f *Factory) replaceNodePair() error {
	return f.replacePair(f.cfg.NodeCertPath, f.cfg.NodeKeyPath, f.writeNodePair)
}

// replacePair writes a new pair beside the current one with write, then renames it over
// the current one.
func (f *Factory) replacePair(
	certPath, keyPath string,
	write func(certPath, keyPath string) error,
) error {
	nextCert, nextKey := certPath+nextSuffix, keyPath+nextSuffix
	if err := f.cfg.FS.Remove(nextCert); err != nil {
		return err
	}
	if err := f.cfg.FS.Remove(nextKey); err != nil {
		return err
	}
	if err := write(nextCert, nextKey); err != nil {
		return err
	}
	if err := f.cfg.FS.Rename(nextKey, keyPath); err != nil {
		return err
	}
	return f.cfg.FS.Rename(nextCert, certPath)
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
	return f.writeKeyAndCert(certPath, keyPath, nodeKey, b)
}

func (f *Factory) writeKeyAndCert(
	certPath, keyPath string,
	key crypto.Signer,
	cert []byte,
) error {
	keyP, err := xpem.FromPrivateKey(key)
	if err != nil {
		return err
	}
	if err = f.writePEM(keyPath, keyP, false); err != nil {
		return err
	}
	return f.writePEM(certPath, xpem.FromCertBytes(cert) /* multi */, false)
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
