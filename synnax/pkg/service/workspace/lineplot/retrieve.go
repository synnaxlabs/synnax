// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot

import (
	"context"
	"github.com/google/uuid"
	"github.com/synnaxlabs/x/gorp"
)

type Retrieve struct {
	baseTX gorp.Tx
	gorp   gorp.Retrieve[uuid.UUID, LinePlot]
}

func (r Retrieve) WhereKeys(keys ...uuid.UUID) Retrieve {
	r.gorp = r.gorp.WhereKeys(keys...)
	return r
}

func (r Retrieve) Entry(plot *LinePlot) Retrieve {
	r.gorp = r.gorp.Entry(plot)
	return r
}

func (r Retrieve) Entries(plots *[]LinePlot) Retrieve {
	r.gorp = r.gorp.Entries(plots)
	return r
}

func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	return r.gorp.Exec(ctx, gorp.OverrideTx(r.baseTX, tx))
}
