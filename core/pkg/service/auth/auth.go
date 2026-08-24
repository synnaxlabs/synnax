// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package auth provides the credential primitives used to authenticate entities in
// Synnax. It defines the plaintext credential payload [Credentials] and the gorp-backed
// [Service] that persists and validates them. Password hashing is an implementation
// detail of [Service].
//
// Higher-level packages compose these primitives: the user service owns credential
// lifecycle alongside user records and drives startup-time root-user reconciliation
// (creating, demoting, or rotating the root credential to match the configured root).
// Direct callers of this package are expected to be infrastructure-level — most
// application code should reach for the user service instead.
package auth

import (
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// Credentials is a set of unencrypted username/password credentials used to
// authenticate an entity (user, client, etc.). These are NOT safe to store on disk;
// persistent storage of credentials is the responsibility of [Service].
type Credentials struct {
	// Username is the username of the credential entry.
	Username string `json:"username" msgpack:"username" validate:"required"`
	// Password is the plaintext password of the credential entry. Hashing and
	// validation are the responsibility of [Service].
	Password string `json:"password" msgpack:"password" validate:"required"`
}

var _ override.Zeroable = Credentials{}

// IsZero implements the override.Zeroable interface.
func (c Credentials) IsZero() bool { return c == Credentials{} }

// Validate validates the Credentials.
func (c Credentials) Validate() error {
	v := validate.New("auth.credentials")
	validate.NotEmptyString(v, "username", c.Username)
	validate.NotEmptyString(v, "password", c.Password)
	return v.Error()
}
