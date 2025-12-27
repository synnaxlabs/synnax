// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
)

// Retrieve is a query builder for retrieving arcs. It should not be instantiated
// directly, and should instead be instantiated via the NewRetrieve method on
// arc.Service.
type Retrieve struct {
	baseTX     gorp.Tx
	gorp       gorp.Retrieve[uuid.UUID, Arc]
	otg        *ontology.Ontology
	searchTerm string
}

// WhereKeys filters the arcs by the given keys.
func (r Retrieve) WhereKeys(keys ...uuid.UUID) Retrieve {
	r.gorp = r.gorp.WhereKeys(keys...)
	return r
}

// WhereNames filters the arcs by the given names.
func (r Retrieve) WhereNames(names ...string) Retrieve {
	if len(names) == 0 {
		return r
	}
	r.gorp = r.gorp.Where(func(ctx gorp.Context, a *Arc) (bool, error) {
		return lo.Contains(names, a.Name), nil
	})
	return r
}

// Search sets a fuzzy search term that Retrieve will use to filter results.
func (r Retrieve) Search(term string) Retrieve {
	r.searchTerm = term
	return r
}

// Limit limits the number of results returned.
func (r Retrieve) Limit(limit int) Retrieve {
	r.gorp = r.gorp.Limit(limit)
	return r
}

// Offset offsets the results returned.
func (r Retrieve) Offset(offset int) Retrieve {
	r.gorp = r.gorp.Offset(offset)
	return r
}

// Entry binds the given Arc to the query. This pointer is where the results of the
// query will be stored after Exec is called.
func (r Retrieve) Entry(arc *Arc) Retrieve {
	r.gorp = r.gorp.Entry(arc)
	return r
}

// Entries binds the given slice of arcs to the query. This pointer is where the results
// of the query will be stored after Exec is called.
func (r Retrieve) Entries(arcs *[]Arc) Retrieve {
	r.gorp = r.gorp.Entries(arcs)
	return r
}

// Exec executes the query against the given translate. The results of the query
// will be stored in the pointer given to the Entry or Entries method. If tx is nil,
// the query will be executed directly against the underlying gorp.DB provided to the
// Arc service.
func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	tx = gorp.OverrideTx(r.baseTX, tx)
	if r.searchTerm != "" && r.otg != nil {
		ids, err := r.otg.SearchIDs(ctx, ontology.SearchRequest{
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
	return r.gorp.Exec(ctx, tx)
}
