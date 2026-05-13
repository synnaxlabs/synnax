// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package auth

import (
	"github.com/synnaxlabs/x/errors"
	"golang.org/x/crypto/bcrypt"
)

// ErrInvalidHash is returned by [RawPassword.Hash] when the underlying hashing
// primitive rejects the input (for example, when the password exceeds bcrypt's 72-byte
// limit).
var ErrInvalidHash = errors.Wrap(ErrAuth, "invalid hash")

// RawPassword is a plaintext password. It is not safe to store on disk; call
// [RawPassword.Hash] to produce a [HashedPassword] suitable for persistence.
type RawPassword string

// Hash hashes the password using bcrypt at the default cost. Returns [ErrInvalidHash]
// (wrapped) when bcrypt rejects the input.
func (r RawPassword) Hash() (HashedPassword, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(r), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.Combine(ErrInvalidHash, err)
	}
	return h, nil
}

// HashedPassword is the bcrypt-hashed form of a password. It is safe to persist. Call
// [HashedPassword.Validate] to compare it against a plaintext candidate.
type HashedPassword []byte

// Validate reports whether r matches the hash. Returns [ErrInvalidCredentials]
// (wrapped) when the candidate does not match.
func (h HashedPassword) Validate(r RawPassword) error {
	if err := bcrypt.CompareHashAndPassword(h, []byte(r)); err != nil {
		return errors.Combine(ErrInvalidCredentials, err)
	}
	return nil
}
