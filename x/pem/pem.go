package pem

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"github.com/cockroachdb/errors"
	"io"
)

const (
	TypeECDSAPrivateKey   = "EC PRIVATE KEY"
	TypeRSAPrivateKey     = "RSA PRIVATE KEY"
	TypeED25519PrivateKey = "PRIVATE KEY"
	TypeCertificate       = "CERTIFICATE"
)

func FromPrivateKey(key crypto.PrivateKey) (*pem.Block, error) {
	switch key := key.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{
			Type:  TypeRSAPrivateKey,
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		}, nil
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(key)
		return &pem.Block{
			Type:  TypeECDSAPrivateKey,
			Bytes: b,
		}, errors.Wrap(err, "[security] - failed to marshal ECDSA private key")
	case *ed25519.PrivateKey:
		b, err := x509.MarshalPKCS8PrivateKey(key)
		return &pem.Block{
			Type:  TypeED25519PrivateKey,
			Bytes: b,
		}, errors.Wrap(err, "[security] - failed to marshal ed25519 private key")
	}
	return nil, errors.New("[security] - unsupported key type")
}

func FromCertBytes(b []byte) *pem.Block {
	return &pem.Block{Type: TypeCertificate, Bytes: b}
}

func ToPrivateKey(b *pem.Block) (crypto.PrivateKey, error) {
	switch b.Type {
	case TypeRSAPrivateKey:
		return x509.ParsePKCS1PrivateKey(b.Bytes)
	case TypeECDSAPrivateKey:
		return x509.ParseECPrivateKey(b.Bytes)
	case TypeED25519PrivateKey:
		return x509.ParsePKCS8PrivateKey(b.Bytes)
	}
	return nil, errors.New("[security] - unsupported key type")
}

func Write(w io.Writer, block *pem.Block) error { return pem.Encode(w, block) }

func Read(r io.Reader) (*pem.Block, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	p, _ := pem.Decode(b)
	return p, nil
}
