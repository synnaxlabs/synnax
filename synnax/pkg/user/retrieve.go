// Copyright 2024 Synnax Labs, Inc.
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

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/x/gorp"
)

type Retrieve struct {
	baseTX gorp.Tx
	gorp   gorp.Retrieve[uuid.UUID, User]
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

// WhereUsername filters the query to only include users with the given username.
func (r Retrieve) WhereUsername(username string) Retrieve {
	r.gorp = r.gorp.Where(func(u *User) bool {
		return u.Username == username
	})
	return r
}

// WhereUsernames filters the query to only include users with the given usernames.
func (r Retrieve) WhereUsernames(usernames ...string) Retrieve {
	r.gorp = r.gorp.Where(func(u *User) bool {
		return lo.Contains(usernames, u.Username)
	})
	return r
}

// Exec executes the query.
func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	return r.gorp.Exec(ctx, gorp.OverrideTx(r.baseTX, tx))
}
