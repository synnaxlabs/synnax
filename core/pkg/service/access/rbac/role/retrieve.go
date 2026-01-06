// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package role

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/x/gorp"
)

// Retrieve is used to retrieve roles from the database.
type Retrieve struct {
	baseTx gorp.Tx
	gorp   gorp.Retrieve[uuid.UUID, Role]
}

// WhereKeys filters roles by their UUIDs.
func (r Retrieve) WhereKeys(keys ...uuid.UUID) Retrieve {
	r.gorp = r.gorp.WhereKeys(keys...)
	return r
}

// WhereInternal filters roles by their internal flag.
func (r Retrieve) WhereInternal(internal bool) Retrieve {
	r.gorp = r.gorp.Where(func(_ gorp.Context, role *Role) (bool, error) {
		return role.Internal == internal, nil
	})
	return r
}

// WhereName filters roles by their name.
func (r Retrieve) WhereName(name string) Retrieve {
	r.gorp = r.gorp.Where(func(_ gorp.Context, role *Role) (bool, error) {
		return role.Name == name, nil
	})
	return r
}

// Limit sets the maximum number of roles to retrieve.
func (r Retrieve) Limit(limit int) Retrieve {
	r.gorp = r.gorp.Limit(limit)
	return r
}

// Offset sets the number of roles to skip before retrieving.
func (r Retrieve) Offset(offset int) Retrieve {
	r.gorp = r.gorp.Offset(offset)
	return r
}

// Entry sets the target for a single role retrieval.
func (r Retrieve) Entry(role *Role) Retrieve {
	r.gorp = r.gorp.Entry(role)
	return r
}

// Entries sets the target for multiple role retrieval.
func (r Retrieve) Entries(roles *[]Role) Retrieve {
	r.gorp = r.gorp.Entries(roles)
	return r
}

// Exec executes the retrieval query.
func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	tx = gorp.OverrideTx(r.baseTx, tx)
	return r.gorp.Exec(ctx, tx)
}
