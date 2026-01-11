// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package label

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/label"
	"github.com/synnaxlabs/x/uuid"
)

// Retrieve is a builder for querying labels.
type Retrieve struct {
	baseTx     gorp.Tx
	gorp       gorp.Retrieve[uuid.UUID, label.Label]
	otg        *ontology.Ontology
	searchTerm string
}

// Search executes a fuzzy search for labels whose Name attribute matches the provided term.
func (r Retrieve) Search(term string) Retrieve { r.searchTerm = term; return r }

// Limit limits the number of results that Retrieve will return.
func (r Retrieve) Limit(limit int) Retrieve { r.gorp.Limit(limit); return r }

// Offset marks the starting index of results that Retrieve will return.
func (r Retrieve) Offset(offset int) Retrieve { r.gorp.Offset(offset); return r }

// Entry binds the label.Label that Retrieve will fill results into. If multiple results match
// the query, only the first result will be filled into the provided label.Label.
func (r Retrieve) Entry(label *label.Label) Retrieve { r.gorp.Entry(label); return r }

// Entries binds a slice that Retrieve will fill results into.
func (r Retrieve) Entries(labels *[]label.Label) Retrieve { r.gorp.Entries(labels); return r }

// WhereKeys filters for labels whose Name attribute matches the provided key.
func (r Retrieve) WhereKeys(keys ...uuid.UUID) Retrieve { r.gorp.WhereKeys(keys...); return r }

// WhereNames filters for labels whose Name attribute matches the provided name.
func (r Retrieve) WhereNames(names ...string) Retrieve {
	r.gorp.Where(func(ctx gorp.Context, label *label.Label) (bool, error) {
		return lo.Contains(names, label.Name), nil
	})
	return r
}

// Exec executes the Retrieve query. If a tx is provided, Exec will use it to execute
// the query. Otherwise, it will execute against the underlying gorp.DB. It's important
// to note that fuzzy search will not be aware of any writes/deletes executed on the
// tx, and will only search the underlying database.
func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	tx = gorp.OverrideTx(r.baseTx, tx)
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
		r.gorp.WhereKeys(keys...)
	}
	return r.gorp.Exec(ctx, tx)
}

// RetrieveFor retrieves all labels that are associated with the provided ontology ID.
// If a tx is provided, RetrieveFor will use it to execute the query. Otherwise, it will
// execute against the underlying gorp.DB.
func (s *Service) RetrieveFor(
	ctx context.Context,
	id ontology.ID,
	tx gorp.Tx,
) ([]label.Label, error) {
	var labelResources []ontology.Resource
	tx = gorp.OverrideTx(s.cfg.DB, tx)
	if err := s.cfg.Ontology.NewRetrieve().
		WhereIDs(id).
		TraverseTo(Labels).
		Entries(&labelResources).
		Exec(ctx, tx); err != nil {
		return nil, err
	}
	keys, err := KeysFromOntologyIDs(ontology.ResourceIDs(labelResources))
	if err != nil {
		return nil, err
	}
	labels := make([]label.Label, 0, len(keys))
	return labels, s.NewRetrieve().
		WhereKeys(keys...).
		Entries(&labels).
		Exec(ctx, tx)
}
