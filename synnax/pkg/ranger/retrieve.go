// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger

import (
	"context"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/search"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
)

type Retrieve struct {
	tx         gorp.Tx
	gorp       gorp.Retrieve[uuid.UUID, Range]
	otg        *ontology.Ontology
	searchTerm string
}

func newRetrieve(tx gorp.Tx, otg *ontology.Ontology) Retrieve {
	return Retrieve{gorp: gorp.NewRetrieve[uuid.UUID, Range](), tx: tx, otg: otg}
}

func (r Retrieve) Search(term string) Retrieve { r.searchTerm = term; return r }

func (r Retrieve) Entry(rng *Range) Retrieve { r.gorp.Entry(rng); return r }

func (r Retrieve) Entries(rng *[]Range) Retrieve { r.gorp.Entries(rng); return r }

func (r Retrieve) WhereKeys(keys ...uuid.UUID) Retrieve { r.gorp.WhereKeys(keys...); return r }

func (r Retrieve) WhereNames(names ...string) Retrieve {
	r.gorp.Where(func(rng *Range) bool { return lo.Contains(names, rng.Name) })
	return r
}

func (r Retrieve) OverlapsWith(tr telem.TimeRange) Retrieve {
	r.gorp.Where(func(rng *Range) bool { return rng.TimeRange.OverlapsWith(tr) })
	return r
}

func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	if r.searchTerm != "" {
		ids, err := r.otg.SearchIDs(ctx, search.Request{
			Type: ontologyType,
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
	return r.gorp.Exec(ctx, gorp.OverrideTx(r.tx, tx))
}
