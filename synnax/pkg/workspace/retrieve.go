package workspace

import (
	"context"
	"github.com/google/uuid"
	"github.com/synnaxlabs/x/gorp"
)

type Retrieve struct {
	baseTX gorp.Tx
	gorp   gorp.Retrieve[uuid.UUID, Workspace]
}

func (r Retrieve) WhereKeys(keys ...uuid.UUID) Retrieve {
	r.gorp = r.gorp.WhereKeys(keys...)
	return r
}

func (r Retrieve) Entry(ws *Workspace) Retrieve {
	r.gorp = r.gorp.Entry(ws)
	return r
}

func (r Retrieve) Entries(wss *[]Workspace) Retrieve {
	r.gorp = r.gorp.Entries(wss)
	return r
}

func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	return r.gorp.Exec(ctx, tx)
}
