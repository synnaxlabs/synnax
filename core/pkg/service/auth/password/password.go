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
	"github.com/synnaxlabs/synnax/pkg/service/auth/base"
	"github.com/synnaxlabs/x/errors"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalid     = errors.Wrap(base.ErrAuth, "invalid credentials")
	ErrInvalidHash = errors.Wrap(base.ErrAuth, "invalid hash")
)

// Raw represents a raw password. It is not safe to store the raw password on disk.
// The password should be hashed by calling Hash before saving it.
type Raw string

// Hash hashes the raw password using bcrypt at the default cost.
func (r Raw) Hash() (Hashed, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(r), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.Combine(ErrInvalidHash, err)
	}
	return h, nil
}

// Hashed represents an encrypted hash of a password. It is safe to store the hash on disk.
// The hash can be compared against a raw password by calling Validate.
type Hashed []byte

// Validate validates the hashed password against the raw password. Returns ErrInvalid
// if the password does not match the hash.
func (h Hashed) Validate(r Raw) error {
	if err := bcrypt.CompareHashAndPassword(h, []byte(r)); err != nil {
		return errors.Combine(ErrInvalid, err)
	}
	return nil
}
