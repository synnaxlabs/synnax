// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package effect

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/search"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/x/gorp"
)

// Retrieve is a query builder for retrieving effects. It should not be instantiated
// directly and should instead be instantiated via the NewRetrieve method on
// effect.Service.
type Retrieve struct {
	baseTX                gorp.Tx
	gorp                  gorp.Retrieve[uuid.UUID, Effect]
	label                 *label.Service
	otg                   *ontology.Ontology
	effectService         *Service
	effectStateChannelKey channel.Key
	searchTerm            string
}

// WhereKeys filters the effects by the given keys.
func (r Retrieve) WhereKeys(keys ...uuid.UUID) Retrieve {
	r.gorp = r.gorp.WhereKeys(keys...)
	return r
}

// Search executes a fuzzy search for effects whose Name attribute matches the provided term.
func (r Retrieve) Search(term string) Retrieve { r.searchTerm = term; return r }

// Limit limits the number of results that Retrieve will return.
func (r Retrieve) Limit(limit int) Retrieve { r.gorp = r.gorp.Limit(limit); return r }

// Offset marks the starting index of results that Retrieve will return.
func (r Retrieve) Offset(offset int) Retrieve { r.gorp = r.gorp.Offset(offset); return r }

// Entry binds the given effect to the query. This pointer is where the results of the
// query will be stored after Exec is called.
func (r Retrieve) Entry(effect *Effect) Retrieve {
	r.gorp = r.gorp.Entry(effect)
	return r
}

// Entries binds the given slice of effects to the query. This pointer is where the results
// of the query will be stored after Exec is called.
func (r Retrieve) Entries(effects *[]Effect) Retrieve {
	r.gorp = r.gorp.Entries(effects)
	return r
}

// Exec executes the query against the given transaction. The results of the query
// will be stored in the pointer given to the Entry or Entries method. If tx is nil,
// the query will be executed directly against the underlying gorp.DB provided to the
// effect service.
func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	tx = gorp.OverrideTx(r.baseTX, tx)
	if r.searchTerm != "" {
		ids, err := r.otg.SearchIDs(ctx, search.Request{
			Type: ontologyType,
			Term: r.searchTerm,
		})
		if err != nil {
			return err
		}
		keys, err := KeysFromOntologyIDs(ids)
		if err != nil {
			return err
		}
		r.gorp = r.gorp.WhereKeys(keys...)
	}

	if err := r.gorp.Exec(ctx, tx); err != nil {
		return err
	}
	entries := gorp.GetEntries[uuid.UUID, Effect](r.gorp.Params)
	for i, e := range entries.All() {
		entries.Set(i, e.UseTx(tx).setLabel(r.label))
	}
	return nil
}
