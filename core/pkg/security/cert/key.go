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
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/mldsa"
	"crypto/rand"
	"crypto/rsa"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// KeyAlgorithm selects the algorithm a Factory generates private keys with.
type KeyAlgorithm string

const (
	// KeyAlgorithmRSA generates an RSA key sized by FactoryConfig.KeySize.
	KeyAlgorithmRSA KeyAlgorithm = "rsa"
	// KeyAlgorithmMLDSA44 generates an ML-DSA-44 key, NIST security category 2.
	KeyAlgorithmMLDSA44 KeyAlgorithm = "ml-dsa-44"
	// KeyAlgorithmMLDSA65 generates an ML-DSA-65 key, NIST security category 3.
	KeyAlgorithmMLDSA65 KeyAlgorithm = "ml-dsa-65"
	// KeyAlgorithmMLDSA87 generates an ML-DSA-87 key, NIST security category 5.
	KeyAlgorithmMLDSA87 KeyAlgorithm = "ml-dsa-87"
)

// ErrUnsupportedKeyAlgorithm is returned when a KeyAlgorithm is not one of the
// constants declared in this package.
var ErrUnsupportedKeyAlgorithm = errors.Wrap(
	validate.ErrValidation,
	"unsupported key algorithm",
)

// PostQuantum reports whether the algorithm is a post-quantum signature scheme. A
// certificate signed by a post-quantum key is rejected by clients whose TLS stack
// predates FIPS 204, browsers included.
func (a KeyAlgorithm) PostQuantum() bool {
	switch a {
	case KeyAlgorithmMLDSA44, KeyAlgorithmMLDSA65, KeyAlgorithmMLDSA87:
		return true
	}
	return false
}

// validate returns ErrUnsupportedKeyAlgorithm if the algorithm is not one of the
// constants declared in this package.
func (a KeyAlgorithm) validate() error {
	switch a {
	case KeyAlgorithmRSA, KeyAlgorithmMLDSA44, KeyAlgorithmMLDSA65, KeyAlgorithmMLDSA87:
		return nil
	}
	return errors.Wrapf(ErrUnsupportedKeyAlgorithm, "%q", a)
}

// GenerateKey generates a private key for the algorithm. keySize sizes an RSA key and
// is ignored by the ML-DSA parameter sets, whose key sizes are fixed by FIPS 204. It
// returns ErrUnsupportedKeyAlgorithm if the algorithm is not recognized.
func (a KeyAlgorithm) GenerateKey(keySize int) (crypto.Signer, error) {
	switch a {
	case KeyAlgorithmRSA:
		return rsa.GenerateKey(rand.Reader, keySize)
	case KeyAlgorithmMLDSA44:
		return mldsa.GenerateKey(mldsa.MLDSA44())
	case KeyAlgorithmMLDSA65:
		return mldsa.GenerateKey(mldsa.MLDSA65())
	case KeyAlgorithmMLDSA87:
		return mldsa.GenerateKey(mldsa.MLDSA87())
	}
	return nil, errors.Wrapf(ErrUnsupportedKeyAlgorithm, "%q", a)
}

// SignsJWT reports whether key's algorithm has a JWT signing method. ML-DSA does not:
// RFC 7518 defines no ML-DSA algorithm, so a Core serving an ML-DSA certificate signs
// authentication tokens with a dedicated key instead.
func SignsJWT(key crypto.PrivateKey) bool {
	switch key.(type) {
	case *rsa.PrivateKey, *ecdsa.PrivateKey, ed25519.PrivateKey:
		return true
	}
	return false
}

// generateTokenKey generates the key a Core signs authentication tokens with. Ed25519
// is fixed rather than configurable: the key never appears in a certificate, so it
// needs no agreement with a peer, and EdDSA keeps tokens small.
func generateTokenKey() (ed25519.PrivateKey, error) {
	_, key, err := ed25519.GenerateKey(rand.Reader)
	return key, err
}
