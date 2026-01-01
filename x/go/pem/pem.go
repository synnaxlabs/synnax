// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pem

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"github.com/synnaxlabs/x/errors"
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

// Write writes the PEM blocks to the writer.
func Write(w io.Writer, blocks ...*pem.Block) error {
	for _, b := range blocks {
		if err := pem.Encode(w, b); err != nil {
			return err
		}
	}
	return nil
}

// Read reads the first PEM block from the reader.
func Read(r io.Reader) (*pem.Block, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	p, _ := pem.Decode(b)
	return p, nil
}

// ReadMany reads all PEM blocks from the reader.
func ReadMany(r io.Reader) (blocks []*pem.Block, err error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return
	}
	for {
		var p *pem.Block
		p, b = pem.Decode(b)
		if p == nil {
			break
		}
		blocks = append(blocks, p)
	}
	return

}
