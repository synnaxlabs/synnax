package mock

import (
	"crypto"
	"crypto/rsa"
)

type KeyProvider struct {
	Key *rsa.PrivateKey
}

func (m KeyProvider) NodeSecret() crypto.PrivateKey {
	return m.Key
}
