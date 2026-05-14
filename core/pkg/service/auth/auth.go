// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package auth provides the credential primitives used to authenticate entities in
// Synnax. It defines the [Authenticator] interface and the plaintext credential payload
// [Credentials] that flows through it. Password hashing and persistent storage are the
// responsibility of [Authenticator] implementations.
//
// Higher-level packages compose these primitives: the user service owns credential
// lifecycle alongside user records, and the rbac service drives startup-time root-user
// reconciliation. Direct callers of this package are expected to be
// infrastructure-level — most application code should reach for the user service
// instead.
package auth

import (
	"context"

	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// Credentials is a set of unencrypted username/password credentials used to
// authenticate an entity (user, client, etc.). These are NOT safe to store on disk;
// persistent storage of credentials is the responsibility of the [Authenticator]
// implementation.
type Credentials struct {
	// Username is the username of the credential entry.
	Username string `json:"username"  msgpack:"username" validate:"required"`
	// Password is the plaintext password of the credential entry. Hashing and
	// validation are the responsibility of the [Authenticator] implementation.
	Password string `json:"password"  msgpack:"password"`
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

// Authenticator validates the identity of a particular entity (i.e. they are who they
// say they are).
type Authenticator interface {
	// Authenticate validates the identity of the entity with the given credentials. If
	// the credentials are invalid, [ErrInvalidCredentials] is returned.
	Authenticate(context.Context, Credentials) error
	// NewWriter opens a new Writer using the provided write context.
	NewWriter(gorp.Tx) Writer
}

// Writer registers and mutates credentials within an authentication service. Every
// Writer method is a primitive: it performs no identity check itself and writes
// directly within the writer's transaction. Verification flows (e.g. "user proves they
// know the old password before rotating it") are composed by higher-level packages by
// combining [Authenticator.Authenticate] with these primitives.
type Writer interface {
	// Register stores new credentials. Returns [ErrRepeatedUsername] if the username is
	// already taken.
	Register(context.Context, Credentials) error
	// UpdateUsername renames the credential entry from oldUsername to newUsername. No
	// identity check; caller is responsible for authorization.
	UpdateUsername(_ context.Context, oldUsername, newUsername string) error
	// ChangePassword replaces the stored password for creds.Username with
	// creds.Password. No identity check; caller is responsible for authorization.
	ChangePassword(_ context.Context, _ Credentials) error
	// Deactivate removes credentials for the given usernames. No identity check; caller
	// is responsible for authorization.
	Deactivate(_ context.Context, usernames ...string) error
}
