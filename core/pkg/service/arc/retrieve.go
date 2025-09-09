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

// Retrieve is a query builder for retrieving arcs. It should not be instantiated
// directly, and should instead be instantiated via the NewRetrieve method on
// arc.Service.
type Retrieve struct {
	baseTX gorp.Tx
	gorp   gorp.Retrieve[uuid.UUID, Arc]
}

// WhereKeys filters the arcs by the given keys.
func (r Retrieve) WhereKeys(keys ...uuid.UUID) Retrieve {
	r.gorp = r.gorp.WhereKeys(keys...)
	return r
}

// Entry binds the given Arc to the query. This pointer is where the results of the
// query will be stored after Exec is called.
func (r Retrieve) Entry(arc *Arc) Retrieve {
	r.gorp = r.gorp.Entry(arc)
	return r
}

// Entries binds the given slice of arcs to the query. This pointer is where the results
// of the query will be stored after Exec is called.
func (r Retrieve) Entries(arcs *[]Arc) Retrieve {
	r.gorp = r.gorp.Entries(arcs)
	return r
}

// Exec executes the query against the given translate. The results of the query
// will be stored in the pointer given to the Entry or Entries method. If tx is nil,
// the query will be executed directly against the underlying gorp.DB provided to the
// Arc service.
func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	return r.gorp.Exec(ctx, gorp.OverrideTx(r.baseTX, tx))
}
