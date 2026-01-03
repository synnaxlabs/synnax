// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package user

import (
	"context"
	"slices"

	"github.com/google/uuid"
	"github.com/synnaxlabs/x/gorp"
)

// A Retrieve is used to retrieve users from the key-value store.
type Retrieve struct {
	// baseTX is the transaction that the Retrieve will use to atomically interact with
	// the key-value store.
	baseTX gorp.Tx
	// gorp is the underlying query that the Retrieve will use to get users from the
	// key-value store.
	gorp gorp.Retrieve[uuid.UUID, User]
}

// WhereKeys filters the query to only include users with the given keys.
func (r Retrieve) WhereKeys(keys ...uuid.UUID) Retrieve {
	r.gorp = r.gorp.WhereKeys(keys...)
	return r
}

// Entry binds the query to the given user.
func (r Retrieve) Entry(user *User) Retrieve {
	r.gorp = r.gorp.Entry(user)
	return r
}

// Entries binds the query to the given users.
func (r Retrieve) Entries(users *[]User) Retrieve {
	r.gorp = r.gorp.Entries(users)
	return r
}

// WhereUsernames filters the query to only include users with the given usernames.
func (r Retrieve) WhereUsernames(usernames ...string) Retrieve {
	r.gorp = r.gorp.Where(func(ctx gorp.Context, u *User) (bool, error) {
		return slices.Contains(usernames, u.Username), nil
	})
	return r
}

// Exec executes the query.
func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	return r.gorp.Exec(ctx, gorp.OverrideTx(r.baseTX, tx))
}
