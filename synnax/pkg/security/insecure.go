package security

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
)

// insecureProvider is an implementation of Provider for use in insecure clusters.
type insecureProvider struct {
	nodeSecret *rsa.PrivateKey
}

func newInsecureProvider(cfg ProviderConfig) (Provider, error) {
	key, err := rsa.GenerateKey(rand.Reader, cfg.KeySize)
	return &insecureProvider{nodeSecret: key}, err
}

// TLS implements TLSProvider.
func (p *insecureProvider) TLS() *tls.Config { return nil }

// NodePrivate implements KeyProvider.
func (p *insecureProvider) NodePrivate() crypto.PrivateKey { return p.nodeSecret }
