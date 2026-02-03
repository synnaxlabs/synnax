// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package password

import (
	"golang.org/x/crypto/bcrypt"
)

// hashers is a list of hashers that synnax uses to encrypt and validate passwords.
// The hashers are tried in order. The first Hasher that returns a non-nil error
// is used. This value should generally remain unmodified unless the cluster is tailored
// to specific user needs.
var hashers = []Hasher{BcryptHasher{}}

// Hasher hashes and compares passwords against a hash.
type Hasher interface {
	// Hash hashes a Raw password. Returns an error if the password cannot be hashed.
	Hash(pwd Raw) (Hashed, error)
	// Compare compares a Raw password against a Hashed password. Returns an error
	// if pwd does not match the hash.
	Compare(pwd Raw, hash Hashed) error
}

// BcryptHasher is a Hasher that uses the bcrypt library to hash and compare passwords.
type BcryptHasher struct{}

var BcryptHashCost = bcrypt.DefaultCost

// Hash implements the Hasher interface.
func (BcryptHasher) Hash(pwd Raw) (Hashed, error) {
	return bcrypt.GenerateFromPassword([]byte(pwd), BcryptHashCost)
}

// Compare implements the Hasher interface.
func (BcryptHasher) Compare(pwd Raw, hash Hashed) error {
	return bcrypt.CompareHashAndPassword(hash, []byte(pwd))
}
