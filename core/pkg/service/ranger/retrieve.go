// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
)

// Retrieve is used to retrieve ranges from the cluster using a builder pattern.
type Retrieve struct {
	baseTX     gorp.Tx
	gorp       gorp.Retrieve[uuid.UUID, Range]
	otg        *ontology.Ontology
	label      *label.Service
	searchTerm string
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
func (r Retrieve) WhereKeys(keys ...uuid.UUID) Retrieve {
	r.gorp.WhereKeys(keys...)
	return r
}

// WhereNames filters for ranges whose Name attribute matches the provided name.
func (r Retrieve) WhereNames(names ...string) Retrieve {
	r.gorp.Where(func(ctx gorp.Context, rng *Range) (bool, error) {
		return lo.Contains(names, rng.Name), nil
	})
	return r
}

// WhereOverlapsWith filters for ranges whose TimeRange overlaps with the
func (r Retrieve) WhereOverlapsWith(tr telem.TimeRange) Retrieve {
	r.gorp.Where(func(ctx gorp.Context, rng *Range) (bool, error) {
		return rng.TimeRange.OverlapsWith(tr), nil
	})
	return r
}

func (r Retrieve) WhereHasLabels(matchLabels ...uuid.UUID) Retrieve {
	r.gorp.Where(func(ctx gorp.Context, rng *Range) (bool, error) {
		oRng := rng.UseTx(ctx.Tx).setLabel(r.label).setOntology(r.otg)
		labels, err := oRng.RetrieveLabels(ctx)
		if err != nil {
			return false, err
		}
		labelKeys := lo.Map(labels, func(l label.Label, _ int) uuid.UUID { return l.Key })
		return lo.ContainsBy(labelKeys, func(l uuid.UUID) bool {
			return lo.Contains(matchLabels, l)
		}), nil
	})
	return r
}

// Exec executes the query and fills the results into the provided Range or slice of
// Ranges. It's important to note that fuzzy search will not be aware of any writes/
// deletes executed on the tx, and will only search the underlying database.
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
		keys, err := KeysFromOntologyIDs(ids)
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
		entries.Set(i, e.UseTx(tx).setOntology(r.otg).setLabel(r.label))
	}
	return nil
}
