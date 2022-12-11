package mock

import (
	"crypto"
	"crypto/rsa"
)

// KeyProvider is a mock implementation of security.KeyProvider
// that wraps an RSA private key.
type KeyProvider struct {
	Key *rsa.PrivateKey
}

// NodeSecret implements security.KeyProvider.
func (m KeyProvider) NodeSecret() crypto.PrivateKey { return m.Key }
