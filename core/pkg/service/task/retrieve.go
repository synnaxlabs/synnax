// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/x/gorp"
)

type Retrieve struct {
	baseTX     gorp.Tx
	otg        *ontology.Ontology
	gorp       gorp.Retrieve[Key, Task]
	searchTerm string
}

func (r Retrieve) Search(term string) Retrieve {
	r.searchTerm = term
	return r
}

func (r Retrieve) WhereNames(names ...string) Retrieve {
	r.gorp = r.gorp.Where(func(ctx gorp.Context, t *Task) (bool, error) {
		return lo.Contains(names, t.Name), nil
	}, gorp.Required())
	return r
}

func (r Retrieve) WhereKeys(keys ...Key) Retrieve {
	r.gorp = r.gorp.WhereKeys(keys...)
	return r
}

func (r Retrieve) WhereRacks(key ...rack.Key) Retrieve {
	r.gorp = r.gorp.Where(func(ctx gorp.Context, t *Task) (bool, error) {
		return lo.Contains(key, t.Rack()), nil
	}, gorp.Required())
	return r
}

func (r Retrieve) WhereTypes(types ...string) Retrieve {
	r.gorp = r.gorp.Where(func(ctx gorp.Context, t *Task) (bool, error) {
		return lo.Contains(types, t.Type), nil
	}, gorp.Required())
	return r
}

func (r Retrieve) Entry(rack *Task) Retrieve {
	r.gorp = r.gorp.Entry(rack)
	return r
}

func (r Retrieve) Entries(racks *[]Task) Retrieve {
	r.gorp = r.gorp.Entries(racks)
	return r
}

func (r Retrieve) Limit(limit int) Retrieve {
	r.gorp = r.gorp.Limit(limit)
	return r
}

func (r Retrieve) WhereInternal(internal bool, opts ...gorp.FilterOption) Retrieve {
	r.gorp = r.gorp.Where(func(ctx gorp.Context, t *Task) (bool, error) {
		return t.Internal == internal, nil
	}, opts...)
	return r
}

func (r Retrieve) WhereSnapshot(snapshot bool, opts ...gorp.FilterOption) Retrieve {
	r.gorp = r.gorp.Where(func(ctx gorp.Context, t *Task) (bool, error) { return t.Snapshot == snapshot, nil }, opts...)
	return r
}

func (r Retrieve) Offset(offset int) Retrieve {
	r.gorp = r.gorp.Offset(offset)
	return r
}

func (r Retrieve) execSearch(ctx context.Context) (Retrieve, error) {
	if r.searchTerm == "" {
		return r, nil
	}
	ids, err := r.otg.SearchIDs(ctx, ontology.SearchRequest{
		Type: OntologyType,
		Term: r.searchTerm,
	})
	if err != nil {
		return r, err
	}
	keys, err := KeysFromOntologyIDs(ids)
	if err != nil {
		return r, err
	}
	r = r.WhereKeys(keys...)
	return r, nil
}

func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) (err error) {
	if r, err = r.execSearch(ctx); err != nil {
		return
	}
	return r.gorp.Exec(ctx, gorp.OverrideTx(r.baseTX, tx))
}

func (r Retrieve) Exists(ctx context.Context, tx gorp.Tx) (bool, error) {
	var err error
	if r, err = r.execSearch(ctx); err != nil {
		return false, err
	}
	return r.gorp.Exists(ctx, gorp.OverrideTx(r.baseTX, tx))
}
