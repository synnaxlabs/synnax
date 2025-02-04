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

func (r Retrieve) Search(term string) Retrieve {
	r.searchTerm = term
	return r
}

func (r Retrieve) WhereNames(names ...string) Retrieve {
	r.gorp = r.gorp.Where(func(m *Task) bool {
		ok := lo.Contains(names, m.Name)
		return ok
	})
	return r
}

func (r Retrieve) WhereKeys(keys ...Key) Retrieve {
	r.gorp = r.gorp.WhereKeys(keys...)
	return r
}

func (r Retrieve) WhereRack(key rack.Key) Retrieve {
	r.gorp = r.gorp.Where(func(m *Task) bool {
		return m.Rack() == key
	}, gorp.Required())
	return r
}

func (r Retrieve) WhereTypes(types ...string) Retrieve {
	r.gorp = r.gorp.Where(func(m *Task) bool {
		return lo.Contains(types, m.Type)
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

func (r Retrieve) WhereInternal(internal bool) Retrieve {
	r.gorp = r.gorp.Where(func(m *Task) bool { return m.Internal == internal })
	return r
}

func (r Retrieve) Offset(offset int) Retrieve {
	r.gorp = r.gorp.Offset(offset)
	return r
}

func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	if r.searchTerm != "" {
		ids, err := r.otg.SearchIDs(ctx, search.Request{
			Type: OntologyType,
			Term: r.searchTerm,
		})
		if err != nil {
			return err
		}
		keys, err := KeysFromOntologyIds(ids)
		if err != nil {
			return err
		}
		r = r.WhereKeys(keys...)
	}
	return r.gorp.Exec(ctx, gorp.OverrideTx(r.baseTX, tx))
}
