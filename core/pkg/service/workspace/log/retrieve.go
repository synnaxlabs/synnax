// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log

import (
	"context"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/uuid"
)

// Retrieve is a query builder for retrieving logs. It should not be instantiated
// directly, and should instead be instantiated via the NewRetrieve method on
// log.Service.
type Retrieve struct {
	baseTX gorp.Tx
	gorp   gorp.Retrieve[uuid.UUID, Log]
}

// WhereKeys filters the logs by the given keys.
func (r Retrieve) WhereKeys(keys ...uuid.UUID) Retrieve {
	r.gorp = r.gorp.WhereKeys(keys...)
	return r
}

// Entry binds the given log to the query. This pointer is where the results of the
// query will be stored after Exec is called.
func (r Retrieve) Entry(log *Log) Retrieve {
	r.gorp = r.gorp.Entry(log)
	return r
}

// Entries binds the given slice of logs to the query. This pointer is where the results
// of the query will be stored after Exec is called.
func (r Retrieve) Entries(logs *[]Log) Retrieve {
	r.gorp = r.gorp.Entries(logs)
	return r
}

// Exec executes the query against the given transaction. The results of the query
// will be stored in the pointer given to the Entry or Entries method. If tx is nil,
// the query will be executed directly against the underlying gorp.DB provided to the
// log service.
func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	return r.gorp.Exec(ctx, gorp.OverrideTx(r.baseTX, tx))
}
