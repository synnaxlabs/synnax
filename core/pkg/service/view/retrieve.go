// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package view

import (
	"context"
	"slices"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
)

// Retrieve is used to retrieve views from the cluster using a builder pattern.
type Retrieve struct {
	baseTX     gorp.Tx
	gorp       gorp.Retrieve[uuid.UUID, View]
	otg        *ontology.Ontology
	searchTerm string
}

// Search sets a fuzzy search term that Retrieve will use to filter results.
func (r Retrieve) Search(term string) Retrieve { r.searchTerm = term; return r }

// Entry binds the View that Retrieve will fill results into. If multiple results match
// the query, only the first result will be filled into the provided View.
func (r Retrieve) Entry(view *View) Retrieve { r.gorp.Entry(view); return r }

// Limit sets the maximum number of results that Retrieve will return.
func (r Retrieve) Limit(limit int) Retrieve { r.gorp.Limit(limit); return r }

// Offset sets the number of results that Retrieve will skip before returning results.
func (r Retrieve) Offset(offset int) Retrieve { r.gorp.Offset(offset); return r }

// Entries binds a slice that Retrieve will fill results into.
func (r Retrieve) Entries(views *[]View) Retrieve { r.gorp.Entries(views); return r }

// WhereKeys filters for views whose Key attribute matches the provided key.
func (r Retrieve) WhereKeys(keys ...uuid.UUID) Retrieve {
	r.gorp = r.gorp.WhereKeys(keys...)
	return r
}

// WhereTypes filters for views whose Type attribute matches the provided type.
func (r Retrieve) WhereTypes(types ...string) Retrieve {
	r.gorp = r.gorp.Where(func(ctx gorp.Context, v *View) (bool, error) {
		return slices.Contains(types, v.Type), nil
	})
	return r
}

// Exec executes the query and fills the results into the provided View or slice of
// views. It's important to note that fuzzy search will not be aware of any writes/
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
