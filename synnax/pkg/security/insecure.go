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
	if err != nil {
		return nil, err
	}
	return &insecureProvider{nodeSecret: key}, nil
}

// TLS implements TLSProvider.
func (p *insecureProvider) TLS() *tls.Config { return nil }

// NodeSecret implements KeyProvider.
func (p *insecureProvider) NodeSecret() crypto.PrivateKey { return p.nodeSecret }
