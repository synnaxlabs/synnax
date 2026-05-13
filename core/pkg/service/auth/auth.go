// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package auth provides the credential primitives used to authenticate entities in
// Synnax. It defines the [Authenticator] interface (and its KV-backed implementation in
// [KV]), the password types ([RawPassword], [HashedPassword]) and the credential
// payloads ([InsecureCredentials], [SecureCredentials]) that flow through them.
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
)

// Authenticator validates the identity of a particular entity (i.e. they are who they
// say they are).
type Authenticator interface {
	// Authenticate validates the identity of the entity with the given credentials. If
	// the credentials are invalid, [ErrInvalidCredentials] is returned.
	Authenticate(context.Context, InsecureCredentials) error
	// NewWriter opens a new Writer using the provided write context.
	NewWriter(gorp.Tx) Writer
}

// Writer registers and mutates credentials within an authentication service. Every
// Writer method is a primitive: it performs no identity check itself and writes
// directly within the writer's transaction. Verification flows (e.g. "user proves they
// know the old password before rotating it") are composed by higher-level packages by
// combining [Authenticator.Authenticate] with these primitives.
type Writer interface {
	// Register stores new credentials. Returns [ErrRepeatedUsername] if the username
	// is already taken.
	Register(context.Context, InsecureCredentials) error
	// UpdateUsername renames the credential entry from oldUsername to newUsername. No
	// identity check; caller is responsible for authorization.
	UpdateUsername(_ context.Context, oldUsername, newUsername string) error
	// ChangePassword verifies the supplied credentials and, on success, replaces the
	// password for the matching entry. Returns [ErrInvalidCredentials] when the
	// supplied credentials are not valid.
	ChangePassword(context.Context, InsecureCredentials, RawPassword) error
	// Deactivate removes credentials for the given usernames. No identity check;
	// caller is responsible for authorization.
	Deactivate(_ context.Context, usernames ...string) error
}
