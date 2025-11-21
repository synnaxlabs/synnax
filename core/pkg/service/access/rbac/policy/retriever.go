// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package policy

import (
	"context"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/x/gorp"
)

type Retriever struct {
	baseTx gorp.Tx
	gorp   gorp.Retrieve[uuid.UUID, Policy]
}

func (r Retriever) WhereKeys(keys ...uuid.UUID) Retriever {
	r.gorp = r.gorp.WhereKeys(keys...)
	return r
}

func (r Retriever) WhereNames(names ...string) Retriever {
	r.gorp = r.gorp.Where(func(ctx gorp.Context, e *Policy) (bool, error) {
		return lo.Contains(names, e.Name), nil
	})
	return r
}

func (r Retriever) Limit(limit int) Retriever {
	r.gorp = r.gorp.Limit(limit)
	return r
}

func (r Retriever) Offset(offset int) Retriever {
	r.gorp = r.gorp.Offset(offset)
	return r
}

func (r Retriever) Exec(ctx context.Context, tx gorp.Tx) error {
	tx = gorp.OverrideTx(r.baseTx, tx)
	return r.gorp.Exec(ctx, tx)
}

func (r Retriever) Entry(p *Policy) Retriever {
	r.gorp = r.gorp.Entry(p)
	return r
}

func (r Retriever) Entries(ps *[]Policy) Retriever {
	r.gorp = r.gorp.Entries(ps)
	return r
}
