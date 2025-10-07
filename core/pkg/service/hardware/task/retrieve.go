// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/search"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/rack"
	"github.com/synnaxlabs/x/gorp"
)

type Retrieve struct {
	baseTX     gorp.Tx
	otg        *ontology.Ontology
	gorp       gorp.Retrieve[Key, Task]
	searchTerm string
}

func (r Retrieve) Search(term string) Retrieve { r.searchTerm = term; return r }

func (r Retrieve) WhereNames(names ...string) Retrieve {
	r.gorp = r.gorp.Where(func(t *Task) bool {
		ok := lo.Contains(names, t.Name)
		return ok
	}, gorp.Required())
	return r
}

func (r Retrieve) WhereKeys(keys ...Key) Retrieve {
	r.gorp = r.gorp.WhereKeys(keys...)
	return r
}

func (r Retrieve) WhereRacks(key ...rack.Key) Retrieve {
	r.gorp = r.gorp.Where(func(t *Task) bool {
		return lo.Contains(key, t.Rack())
	}, gorp.Required())
	return r
}

func (r Retrieve) WhereTypes(types ...string) Retrieve {
	r.gorp = r.gorp.Where(func(t *Task) bool {
		return lo.Contains(types, t.Type)
	}, gorp.Required())
	return r
}

func (r Retrieve) Entry(task *Task) Retrieve { r.gorp = r.gorp.Entry(task); return r }

func (r Retrieve) Entries(tasks *[]Task) Retrieve {
	r.gorp = r.gorp.Entries(tasks)
	return r
}

func (r Retrieve) Limit(limit int) Retrieve {
	r.gorp = r.gorp.Limit(limit)
	return r
}

func (r Retrieve) WhereInternal(internal bool, opts ...gorp.FilterOption) Retrieve {
	r.gorp = r.gorp.Where(func(t *Task) bool { return t.Internal == internal }, opts...)
	return r
}

func (r Retrieve) WhereSnapshot(snapshot bool, opts ...gorp.FilterOption) Retrieve {
	r.gorp = r.gorp.Where(func(m *Task) bool { return m.Snapshot == snapshot }, opts...)
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
	ids, err := r.otg.SearchIDs(ctx, search.Request{
		Type: OntologyType,
		Term: r.searchTerm,
	})
	if err != nil {
		return Retrieve{}, err
	}
	keys, err := KeysFromOntologyIDs(ids)
	if err != nil {
		return Retrieve{}, err
	}
	r = r.WhereKeys(keys...)
	return r, nil
}

func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	var err error
	if r, err = r.execSearch(ctx); err != nil {
		return err
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
