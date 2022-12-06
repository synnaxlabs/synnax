package tls

import (
	"crypto/rsa"
	"crypto/tls"
)

type Provider interface {
	Config() *tls.Config
	RSA() (*rsa.PrivateKey, error)
}
