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
		AllowKeyReuse: config.False(),
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
func (c *Factory) CreateCAPair() error {
	exists, err := c.FS.Exists(c.CACertPath)
	if err != nil {
		return err
	}

	var key crypto.PrivateKey
	if !exists {
		key, err = rsa.GenerateKey(rand.Reader, c.KeySize)
		if err != nil {
			return err
		}
		p, err := xpem.FromPrivateKey(key)
		if err != nil {
			return err
		}
		if err := c.writePEM(c.CAKeyPath, p /* multi */, false); err != nil {
			return err
		}
	} else {
		if !*c.AllowKeyReuse {
			return errors.Newf("CA key %s already exists, but reuse is not allowed", c.CAKeyPath)
		}
		p, err := c.readPEM(c.CAKeyPath)
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
	return c.writePEM(c.CACertPath, xpem.FromCertBytes(b) /*multi */, true)
}

func (c *Factory) CreateCAPairIfMissing() error {
	exists, err := c.FS.Exists(c.CACertPath)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return c.CreateCAPair()
}

// CreateNodePairIfMissing creates a new node certificate and its private key if they do not already exist.
func (c *Factory) CreateNodePairIfMissing() error {
	exists, err := c.FS.Exists(c.NodeCertPath)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return c.CreateNodePair()
}

// CreateNodePair creates a new node certificate and its private key.
func (c *Factory) CreateNodePair() error {
	ca, caPrivate, err := c.Loader.LoadCAPair()
	if err != nil {
		return err
	}

	if len(c.Hosts) == 0 {
		return errors.Wrap(validate.Error, "[cert] - no hosts provided")
	}

	nodeKey, err := rsa.GenerateKey(rand.Reader, c.KeySize)
	if err != nil {
		return err
	}

	keyP, err := xpem.FromPrivateKey(nodeKey)
	if err != nil {
		return err
	}
	if err = c.writePEM(c.NodeKeyPath, keyP, false); err != nil {
		return err
	}

	base, err := newBasex509()
	if err != nil {
		return err
	}
	base.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}

	for _, h := range c.Hosts {
		if ip := net.ParseIP(h.Host()); ip != nil {
			base.IPAddresses = append(base.IPAddresses, ip)
		} else {
			base.DNSNames = append(base.DNSNames, h.Host())
		}
	}

	b, err := x509.CreateCertificate(rand.Reader, base, ca, nodeKey.Public(), caPrivate)
	if err != nil {
		return err
	}

	return c.writePEM(c.NodeCertPath, xpem.FromCertBytes(b) /* multi */, false)
}

func (c *Factory) readPEM(p string) (b *pem.Block, err error) {
	return b, c.withFile(p, c.readFlag(), func(f xfs.File) error {
		b, err = xpem.Read(f)
		return err
	})
}

func (c *Factory) writePEM(p string, block *pem.Block, multi bool) error {
	return c.withFile(p, c.writeFlag(), func(f xfs.File) error {
		blocks, err := xpem.ReadMany(f)
		if len(blocks) > 0 && !multi {
			return errors.Newf("file %s already contains a PEM block, and multi is false", p)
		}
		if err != nil {
			return err
		}
		blocks = append(blocks, block)
		return xpem.Write(f, blocks...)
	})
}

func (c *Factory) withFile(p string, flag int, fn func(fs xfs.File) error) (err error) {
	f, err := c.FS.Open(p, flag)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Combine(err, f.Close())
	}()
	err = fn(f)
	return
}

func (c *Factory) writeFlag() int {
	if *c.AllowKeyReuse {
		return os.O_CREATE | os.O_RDWR
	}
	return os.O_CREATE | os.O_RDWR | os.O_EXCL
}

func (c *Factory) readFlag() int { return os.O_RDONLY }
