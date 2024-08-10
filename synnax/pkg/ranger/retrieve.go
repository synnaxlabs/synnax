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

// Retrieve is used to retrieve ranges from the cluster using a builder pattern.
type Retrieve struct {
	baseTX     gorp.Tx
	gorp       gorp.Retrieve[uuid.UUID, Range]
	otg        *ontology.Ontology
	searchTerm string
}

func newRetrieve(tx gorp.Tx, otg *ontology.Ontology) Retrieve {
	return Retrieve{gorp: gorp.NewRetrieve[uuid.UUID, Range](), baseTX: tx, otg: otg}
}

// Search sets a fuzzy search term that Retrieve will use to filter results.
func (r Retrieve) Search(term string) Retrieve { r.searchTerm = term; return r }

// Entry binds the Range that Retrieve will fill results into. If multiple results match
// the query, only the first result will be filled into the provided Range.
func (r Retrieve) Entry(rng *Range) Retrieve { r.gorp.Entry(rng); return r }

// Limit sets the maximum number of results that Retrieve will return.
func (r Retrieve) Limit(limit int) Retrieve { r.gorp.Limit(limit); return r }

// Offset sets the number of results that Retrieve will skip before returning results.
func (r Retrieve) Offset(offset int) Retrieve { r.gorp.Offset(offset); return r }

// Entries binds a slice that Retrieve will fill results into.
func (r Retrieve) Entries(rng *[]Range) Retrieve { r.gorp.Entries(rng); return r }

// WhereKeys filters for ranges whose Name attribute matches the provided key.
func (r Retrieve) WhereKeys(keys ...uuid.UUID) Retrieve { r.gorp.WhereKeys(keys...); return r }

// WhereNames filters for ranges whose Name attribute matches the provided name.
func (r Retrieve) WhereNames(names ...string) Retrieve {
	r.gorp.Where(func(rng *Range) bool { return lo.Contains(names, rng.Name) })
	return r
}

// WhereOverlapsWith filters for ranges whose TimeRange overlaps with the
func (r Retrieve) WhereOverlapsWith(tr telem.TimeRange) Retrieve {
	r.gorp.Where(func(rng *Range) bool { return rng.TimeRange.OverlapsWith(tr) })
	return r
}

func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	tx = gorp.OverrideTx(r.baseTX, tx)
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
	if err := r.gorp.Exec(ctx, tx); err != nil {
		return err
	}
	entries := gorp.GetEntries[uuid.UUID, Range](r.gorp.Params)
	for i, e := range entries.All() {
		entries.Set(i, e.UseTx(tx).setOntology(r.otg))
	}
	return nil
}
