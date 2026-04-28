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
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/label"
)

// Retrieve is a builder for querying labels.
type Retrieve struct {
	baseTx     gorp.Tx
	gorp       gorp.Retrieve[label.Key, label.Label]
	search     *search.Index
	searchTerm string
}

// Search executes a fuzzy search for labels whose Name attribute matches the provided term.
func (r Retrieve) Search(term string) Retrieve { r.searchTerm = term; return r }

// Limit limits the number of results that Retrieve will return.
func (r Retrieve) Limit(limit int) Retrieve { r.gorp = r.gorp.Limit(limit); return r }

// Offset marks the starting index of results that Retrieve will return.
func (r Retrieve) Offset(offset int) Retrieve { r.gorp = r.gorp.Offset(offset); return r }

// Entry binds the label.Label that Retrieve will fill results into. If multiple results match
// the query, only the first result will be filled into the provided label.Label.
func (r Retrieve) Entry(label *label.Label) Retrieve { r.gorp = r.gorp.Entry(label); return r }

// Entries binds a slice that Retrieve will fill results into.
func (r Retrieve) Entries(labels *[]label.Label) Retrieve { r.gorp = r.gorp.Entries(labels); return r }

// Where applies the provided filters to the query.
func (r Retrieve) Where(filters ...gorp.Filter[label.Key, label.Label]) Retrieve {
	r.gorp = r.gorp.Where(filters...)
	return r
}

// MatchKeys returns a filter that restricts results to labels whose key matches
// any of the provided values.
func MatchKeys(keys ...label.Key) gorp.Filter[label.Key, label.Label] {
	return gorp.MatchKeys[label.Key, label.Label](keys...)
}

// MatchNames returns a filter for labels whose Name matches any of the provided values.
func MatchNames(names ...string) gorp.Filter[label.Key, label.Label] {
	return gorp.Match(func(_ gorp.Context, l *label.Label) (bool, error) {
		return lo.Contains(names, l.Name), nil
	})
}

// Exec executes the Retrieve query. If a tx is provided, Exec will use it to execute
// the query. Otherwise, it will execute against the underlying gorp.DB. It's important
// to note that fuzzy search will not be aware of any writes/deletes executed on the
// tx, and will only search the underlying database.
func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	tx = gorp.OverrideTx(r.baseTx, tx)
	if r.searchTerm != "" {
		ids, err := r.search.Search(ctx, search.Request{
			Type: ontology.ResourceTypeLabel,
			Term: r.searchTerm,
		})
		if err != nil {
			return err
		}
		keys, err := KeysFromOntologyIDs(ids)
		if err != nil {
			return err
		}
		r.gorp = r.gorp.Where(MatchKeys(keys...))
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
		TraverseTo(LabelsOntologyTraverser).
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
		Where(MatchKeys(keys...)).
		Entries(&labels).
		Exec(ctx, tx)
}
