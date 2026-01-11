// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/uuid"
)

// Retrieve is a query builder for retrieving symbols. It should not be instantiated
// directly, and should instead be instantiated via the NewRetrieve method on
// symbol.Service.
type Retrieve struct {
	baseTX     gorp.Tx
	gorp       gorp.Retrieve[uuid.UUID, Symbol]
	otg        *ontology.Ontology
	searchTerm string
}

// WhereKeys filters the symbols by the given keys.
func (r Retrieve) WhereKeys(keys ...uuid.UUID) Retrieve {
	r.gorp = r.gorp.WhereKeys(keys...)
	return r
}

// Search sets a fuzzy search term that Retrieve will use to filter results.
func (r Retrieve) Search(term string) Retrieve { r.searchTerm = term; return r }

// Entry binds the given symbol to the query. This pointer is where the results of the
// query will be stored after Exec is called.
func (r Retrieve) Entry(symbol *Symbol) Retrieve {
	r.gorp = r.gorp.Entry(symbol)
	return r
}

// Entries binds the given slice of symbols to the query. This pointer is where the results
// of the query will be stored after Exec is called.
func (r Retrieve) Entries(symbols *[]Symbol) Retrieve {
	r.gorp = r.gorp.Entries(symbols)
	return r
}

// Exec executes the query against the given transaction. The results of the query
// will be stored in the pointer given to the Entry or Entries method. If tx is nil,
// the query will be executed directly against the underlying gorp.DB provided to the
// symbol service. It's important to note that fuzzy search will not be aware of any writes/
// deletes executed on the tx, and will only search the underlying database.
func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	tx = gorp.OverrideTx(r.baseTX, tx)
	if r.searchTerm != "" {
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
