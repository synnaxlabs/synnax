// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/search"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/x/gorp"
)

// Retrieve is used to retrieve statuses from the cluster using a builder pattern.
type Retrieve[D any] struct {
	baseTX     gorp.Tx
	gorp       gorp.Retrieve[string, Status[D]]
	otg        *ontology.Ontology
	label      *label.Service
	searchTerm string
}

// Search sets a fuzzy search term that Retrieve will use to filter results.
func (r Retrieve[D]) Search(term string) Retrieve[D] { r.searchTerm = term; return r }

// Entry binds the Status that Retrieve will fill results into. If multiple results match
// the query, only the first result will be filled into the provided Status.
func (r Retrieve[D]) Entry(s *Status[D]) Retrieve[D] {
	r.gorp = r.gorp.Entry(s)
	return r
}

// Limit sets the maximum number of results that Retrieve will return.
func (r Retrieve[D]) Limit(limit int) Retrieve[D] {
	r.gorp = r.gorp.Limit(limit)
	return r
}

// Offset sets the number of results that Retrieve will skip before returning results.
func (r Retrieve[D]) Offset(offset int) Retrieve[D] {
	r.gorp = r.gorp.Offset(offset)
	return r
}

// Entries binds a slice that Retrieve will fill results into.
func (r Retrieve[D]) Entries(s *[]Status[D]) Retrieve[D] {
	r.gorp = r.gorp.Entries(s)
	return r
}

// WhereKeys filters for statuses whose Key attribute matches the provided key.
func (r Retrieve[D]) WhereKeys(keys ...string) Retrieve[D] {
	r.gorp = r.gorp.WhereKeys(keys...)
	return r
}

func (r Retrieve[D]) WhereKeyPrefix(prefix string) Retrieve[D] {
	r.gorp = r.gorp.Where(func(_ gorp.Context, s *Status[D]) (bool, error) {
		return strings.HasPrefix(s.Key, prefix), nil
	})
	return r
}

func (r Retrieve[D]) WhereHasLabels(matchLabels ...uuid.UUID) Retrieve[D] {
	r.gorp = r.gorp.Where(func(ctx gorp.Context, s *Status[D]) (bool, error) {
		labels, err := r.label.RetrieveFor(ctx, OntologyID(s.Key), ctx.Tx)
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

// Exec executes the query and fills the results into the provided Status or slice of
// Statuses. It's important to note that fuzzy search will not be aware of any writes/
// deletes executed on the tx, and will only search the underlying database.
func (r Retrieve[D]) Exec(ctx context.Context, tx gorp.Tx) error {
	tx = gorp.OverrideTx(r.baseTX, tx)
	if r.searchTerm != "" {
		ids, err := r.otg.SearchIDs(ctx, search.Request{
			Type: OntologyType,
			Term: r.searchTerm,
		})
		if err != nil {
			return err
		}
		keys := KeysFromOntologyIDs(ids)
		r = r.WhereKeys(keys...)
	}
	return r.gorp.Exec(ctx, tx)
}
