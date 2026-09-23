// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package pem encodes and decodes the PEM blocks that hold Synnax TLS material: private
// keys and certificates. It wraps encoding/pem and crypto/x509 to pick the correct
// block type and marshaling for each key algorithm.
package pem

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/mldsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

const (
	blockTypeECPrivateKey    = "EC PRIVATE KEY"
	blockTypeRSAPrivateKey   = "RSA PRIVATE KEY"
	blockTypePKCS8PrivateKey = "PRIVATE KEY"
	blockTypeCertificate     = "CERTIFICATE"
)

var errUnsupportedKey = errors.Wrap(validate.ErrValidation, "unsupported key type")

// FromPrivateKey encodes key as a PEM block. It accepts RSA, ECDSA, Ed25519 and ML-DSA
// keys, and returns validate.ErrValidation for any other algorithm.
func FromPrivateKey(key crypto.PrivateKey) (*pem.Block, error) {
	switch key := key.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{
			Type:  blockTypeRSAPrivateKey,
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		}, nil
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(key)
		return &pem.Block{
			Type:  blockTypeECPrivateKey,
			Bytes: b,
		}, errors.Wrap(err, "failed to marshal ECDSA private key")
	case ed25519.PrivateKey:
		b, err := x509.MarshalPKCS8PrivateKey(key)
		return &pem.Block{
			Type:  blockTypePKCS8PrivateKey,
			Bytes: b,
		}, errors.Wrap(err, "failed to marshal ed25519 private key")
	case *mldsa.PrivateKey:
		b, err := x509.MarshalPKCS8PrivateKey(key)
		return &pem.Block{
			Type:  blockTypePKCS8PrivateKey,
			Bytes: b,
		}, errors.Wrap(err, "failed to marshal ML-DSA private key")
	}
	return nil, errors.Wrap(errUnsupportedKey, "cannot encode private key")
}

// FromCertBytes wraps DER-encoded certificate bytes in a CERTIFICATE PEM block. The
// bytes are not parsed or validated.
func FromCertBytes(b []byte) *pem.Block {
	return &pem.Block{Type: blockTypeCertificate, Bytes: b}
}

// ToPrivateKey decodes the private key held in b. It accepts the block types written by
// FromPrivateKey, and returns validate.ErrValidation for any other block type.
func ToPrivateKey(b *pem.Block) (crypto.PrivateKey, error) {
	switch b.Type {
	case blockTypeRSAPrivateKey:
		return x509.ParsePKCS1PrivateKey(b.Bytes)
	case blockTypeECPrivateKey:
		return x509.ParseECPrivateKey(b.Bytes)
	case blockTypePKCS8PrivateKey:
		return x509.ParsePKCS8PrivateKey(b.Bytes)
	}
	return nil, errors.Wrapf(errUnsupportedKey, "cannot decode PEM block %q", b.Type)
}

// Write encodes blocks to w in order.
func Write(w io.Writer, blocks ...*pem.Block) error {
	for _, b := range blocks {
		if err := pem.Encode(w, b); err != nil {
			return err
		}
	}
	return nil
}

// Read reads r to completion and returns its first PEM block. It returns a nil block
// and a nil error when r holds no PEM data.
func Read(r io.Reader) (*pem.Block, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	p, _ := pem.Decode(b)
	return p, nil
}

// ReadMany reads r to completion and returns every PEM block in it, in order. Trailing
// data that is not a PEM block is ignored.
func ReadMany(r io.Reader) ([]*pem.Block, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var blocks []*pem.Block
	for {
		var p *pem.Block
		p, b = pem.Decode(b)
		if p == nil {
			break
		}
		blocks = append(blocks, p)
	}
	return blocks, nil
}
