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
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net"
	"os"

	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/override"
	xpem "github.com/synnaxlabs/x/pem"
	"github.com/synnaxlabs/x/validate"
)

// FactoryConfig is the configuration for creating a new Factory.
type FactoryConfig struct {
	// AllowKeyReuse allows the CA key to be reused if it already exists.
	AllowKeyReuse *bool
	LoaderConfig
	// Hosts is the list of hosts to use for the node certificate.
	Hosts []address.Address
	// KeySize is the size of the private key to generate.
	KeySize int
}

var (
	_ config.Config[FactoryConfig] = FactoryConfig{}
	// DefaultFactoryConfig returns the default configuration for a Factory.
	DefaultFactoryConfig = FactoryConfig{
		LoaderConfig:  DefaultLoaderConfig,
		KeySize:       2048,
		AllowKeyReuse: new(false),
	}
)

// Override implements [config.Config].
func (f FactoryConfig) Override(other FactoryConfig) FactoryConfig {
	f.KeySize = override.Numeric(f.KeySize, other.KeySize)
	f.Hosts = override.Slice(f.Hosts, other.Hosts)
	f.AllowKeyReuse = override.Nil(f.AllowKeyReuse, other.AllowKeyReuse)
	f.LoaderConfig = f.LoaderConfig.Override(other.LoaderConfig)
	return f
}

// Validate implements [config.Config].
func (f FactoryConfig) Validate() error {
	v := validate.New("cert.factory")
	validate.Positive(v, "key_size", f.KeySize)
	validate.NotNil(v, "allow_key_reuse", f.AllowKeyReuse)
	v.Exec(f.LoaderConfig.Validate)
	return v.Error()
}

// Factory generates self-signed certificates.
type Factory struct {
	Loader Loader
	FactoryConfig
}

// NewFactory creates a new Factory.
func NewFactory(configs ...FactoryConfig) (*Factory, error) {
	cfg, err := config.New(DefaultFactoryConfig, configs...)
	if err != nil {
		return nil, err
	}
	loader, err := NewLoader(cfg.LoaderConfig)
	cfg.LoaderConfig = loader.LoaderConfig
	cfg.FS = loader.FS
	return &Factory{FactoryConfig: cfg, Loader: *loader}, err
}

// CreateCAPair creates a new CA certificate and its private key.
func (f *Factory) CreateCAPair() error {
	exists, err := f.FS.Exists(f.CACertPath)
	if err != nil {
		return err
	}

	var key crypto.PrivateKey
	if !exists {
		key, err = rsa.GenerateKey(nil, f.KeySize)
		if err != nil {
			return err
		}
		p, err := xpem.FromPrivateKey(key)
		if err != nil {
			return err
		}
		if err := f.writePEM(f.CAKeyPath, p /* multi */, false); err != nil {
			return err
		}
	} else {
		if !*f.AllowKeyReuse {
			return errors.Newf(
				"CA key %s already exists, but reuse is not allowed",
				f.CAKeyPath,
			)
		}
		p, err := f.readPEM(f.CAKeyPath)
		if err != nil {
			return err
		}
		key, err = xpem.ToPrivateKey(p)
		if err != nil {
			return err
		}
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

	b, err := x509.CreateCertificate(nil, base, base, key.(crypto.Signer).Public(), key)
	if err != nil {
		return err
	}
	return f.writePEM(f.CACertPath, xpem.FromCertBytes(b) /*multi */, true)
}

func (f *Factory) CreateCAPairIfMissing() error {
	exists, err := f.FS.Exists(f.CACertPath)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return f.CreateCAPair()
}

// CreateNodePairIfMissing creates a new node certificate and its private key if they do
// not already exist.
func (f *Factory) CreateNodePairIfMissing() error {
	exists, err := f.FS.Exists(f.NodeCertPath)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return f.CreateNodePair()
}

// CreateNodePair creates a new node certificate and its private key.
func (f *Factory) CreateNodePair() error {
	nodeKey, err := rsa.GenerateKey(nil, f.KeySize)
	if err != nil {
		return err
	}
	b, err := f.signNodeCert(nodeKey, f.Hosts)
	if err != nil {
		return err
	}
	keyP, err := xpem.FromPrivateKey(nodeKey)
	if err != nil {
		return err
	}
	if err = f.writePEM(f.NodeKeyPath, keyP, false); err != nil {
		return err
	}
	return f.writePEM(f.NodeCertPath, xpem.FromCertBytes(b) /* multi */, false)
}

// SignNodeCert signs an in-memory node certificate for the given hosts using the CA,
// without touching the filesystem. The caller supplies the hosts because they vary per
// listener; the factory supplies the CA the certificate chains to.
func (f *Factory) SignNodeCert(hosts []address.Address) (*tls.Certificate, error) {
	nodeKey, err := rsa.GenerateKey(nil, f.KeySize)
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
	nodeKey *rsa.PrivateKey,
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
		if len(blocks) > 0 && !multi {
			return errors.Newf(
				"file %s already contains a PEM block, and multi is false",
				p,
			)
		}
		if err != nil {
			return err
		}
		blocks = append(blocks, block)
		return xpem.Write(file, blocks...)
	})
}

func (f *Factory) withFile(p string, flag int, fn func(fs xfs.File) error) (err error) {
	file, err := f.FS.Open(p, flag)
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
	if *f.AllowKeyReuse {
		return os.O_CREATE | os.O_RDWR
	}
	return os.O_CREATE | os.O_RDWR | os.O_EXCL
}

func (f *Factory) readFlag() int { return os.O_RDONLY }
