// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group

import (
	"context"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/x/gorp"
)

type Retrieve struct {
	tx   gorp.Tx
	gorp gorp.Retrieve[uuid.UUID, Group]
}

func newRetrieve(tx gorp.Tx) Retrieve {
	return Retrieve{
		tx:   tx,
		gorp: gorp.NewRetrieve[uuid.UUID, Group](),
	}
}

func (r Retrieve) WhereNames(names ...string) Retrieve {
	r.gorp = r.gorp.Where(func(ctx gorp.Context, grp *Group) (bool, error) {
		return lo.Contains(names, grp.Name), nil
	})
	return r
}

func (r Retrieve) WhereKeys(keys ...uuid.UUID) Retrieve {
	r.gorp = r.gorp.WhereKeys(keys...)
	return r
}

func (r Retrieve) Entry(grp *Group) Retrieve {
	r.gorp = r.gorp.Entry(grp)
	return r
}

func (r Retrieve) Entries(grps *[]Group) Retrieve {
	r.gorp = r.gorp.Entries(grps)
	return r
}

func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	return r.gorp.Exec(ctx, gorp.OverrideTx(r.tx, tx))
}

func (r Retrieve) Exists(ctx context.Context, tx gorp.Tx) (bool, error) {
	return r.gorp.Exists(ctx, gorp.OverrideTx(r.tx, tx))
}
