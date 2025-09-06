// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/x/gorp"
)

// Retrieve is a query builder for retrieving slates. It should not be instantiated
// directly, and should instead be instantiated via the NewRetrieve method on
// arc.Service.
type Retrieve struct {
	baseTX gorp.Tx
	gorp   gorp.Retrieve[uuid.UUID, arc]
}

// WhereKeys filters the slates by the given keys.
func (r Retrieve) WhereKeys(keys ...uuid.UUID) Retrieve {
	r.gorp = r.gorp.WhereKeys(keys...)
	return r
}

// Entry binds the given arc to the query. This pointer is where the results of the
// query will be stored after Exec is called.
func (r Retrieve) Entry(arc *arc) Retrieve {
	r.gorp = r.gorp.Entry(arc)
	return r
}

// Entries binds the given slice of slates to the query. This pointer is where the results
// of the query will be stored after Exec is called.
func (r Retrieve) Entries(slates *[]arc) Retrieve {
	r.gorp = r.gorp.Entries(slates)
	return r
}

// Exec executes the query against the given transslate. The results of the query
// will be stored in the pointer given to the Entry or Entries method. If tx is nil,
// the query will be executed directly against the underlying gorp.DB provided to the
// arc service.
func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	return r.gorp.Exec(ctx, gorp.OverrideTx(r.baseTX, tx))
}
