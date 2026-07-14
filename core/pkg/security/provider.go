// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package security

import (
	"crypto"
	"crypto/tls"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// TLSProvider assembles listener TLS configurations for services that require them.
type TLSProvider interface {
	// TLSConfigFor combines the node-wide TLS policy with the certificate served by src,
	// producing the tls.Config for a single listener. It returns nil in insecure mode.
	// The returned config's behavior may change over time as src reloads its certificate.
	TLSConfigFor(src cert.Source) *tls.Config
	// NodeClientConfig returns the tls.Config the node uses when dialing peers, backed by
	// the node's own certificate. It returns nil in insecure mode.
	NodeClientConfig() *tls.Config
}

// KeyProvider provides information of private keys for the node.
type KeyProvider interface {
	// NodePrivate returns the private key of the node's TLS certificate.
	NodePrivate() crypto.PrivateKey
}

// Provider provides security information and services for the node. It's important to note
// that Provider itself does not implement any security mechanisms, but rather provides
// configuration and information for other components to implement them.
type Provider interface {
	TLSProvider
	KeyProvider
}

// ProviderConfig is the configuration for creating a new Provider.
type ProviderConfig struct {
	// Insecure indicates whether the node should run in insecure mode.
	Insecure *bool
	cert.LoaderConfig
	// KeySize is the size of private key to use in case key generation is required.
	KeySize int
}

var (
	_ config.Config[ProviderConfig] = ProviderConfig{}
	// DefaultProviderConfig is the default configuration for the security
	// Provider.
	DefaultProviderConfig = ProviderConfig{
		LoaderConfig: cert.DefaultLoaderConfig,
		Insecure:     new(true),
		KeySize:      cert.DefaultFactoryConfig.KeySize,
	}
)

// Override implements Properties.
func (s ProviderConfig) Override(other ProviderConfig) ProviderConfig {
	s.LoaderConfig = s.LoaderConfig.Override(other.LoaderConfig)
	s.Insecure = override.Nil(s.Insecure, other.Insecure)
	return s
}

// Validate implements Properties.
func (s ProviderConfig) Validate() error {
	v := validate.New("security.provider")
	validate.NotNil(v, "insecure", s.Insecure)
	v.Exec(s.LoaderConfig.Validate)
	return v.Error()
}

// NewProvider opens a new security Provider using the given configuration.
func NewProvider(configs ...ProviderConfig) (Provider, error) {
	cfg, err := config.New(DefaultProviderConfig, configs...)
	if err != nil {
		return nil, err
	}
	return lo.Ternary(*cfg.Insecure, newInsecureProvider, newSecureProvider)(cfg)
}
