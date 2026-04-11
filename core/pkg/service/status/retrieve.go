// Copyright 2026 Synnax Labs, Inc.
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
	"slices"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/x/gorp"
	xlabel "github.com/synnaxlabs/x/label"
	"github.com/synnaxlabs/x/status"
)

// Retrieve is used to retrieve statuses from the cluster using a builder pattern.
type Retrieve[D any] struct {
	baseTX     gorp.Tx
	gorp       gorp.Retrieve[string, Status[D]]
	search     *search.Index
	label      *label.Service
	searchTerm string
}

// Filter is a per-service filter that is bound to the Retrieve when passed to
// Where. Pure filters ignore the Retrieve argument; service-bound filters read
// from it (e.g. r.label) to evaluate. Use Match to construct one from a closure.
type Filter[D any] func(r Retrieve[D]) gorp.Filter[string, Status[D]]

// Match wraps a closure that needs the Retrieve into a Filter.
func Match[D any](
	f func(ctx gorp.Context, r Retrieve[D], s *Status[D]) (bool, error),
) Filter[D] {
	return func(r Retrieve[D]) gorp.Filter[string, Status[D]] {
		return gorp.Match(func(ctx gorp.Context, s *Status[D]) (bool, error) {
			return f(ctx, r, s)
		})
	}
}

// And returns a filter that matches when all provided filters match.
func And[D any](fs ...Filter[D]) Filter[D] {
	return func(r Retrieve[D]) gorp.Filter[string, Status[D]] {
		inner := make([]gorp.Filter[string, Status[D]], len(fs))
		for i, f := range fs {
			inner[i] = f(r)
		}
		return gorp.And(inner...)
	}
}

// Or returns a filter that matches when any provided filter matches.
func Or[D any](fs ...Filter[D]) Filter[D] {
	return func(r Retrieve[D]) gorp.Filter[string, Status[D]] {
		inner := make([]gorp.Filter[string, Status[D]], len(fs))
		for i, f := range fs {
			inner[i] = f(r)
		}
		return gorp.Or(inner...)
	}
}

// Not returns a filter that inverts the provided filter.
func Not[D any](f Filter[D]) Filter[D] {
	return func(r Retrieve[D]) gorp.Filter[string, Status[D]] {
		return gorp.Not(f(r))
	}
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

// Where applies the provided filters to the query, binding each filter to the
// Retrieve so service-bound filters can read from r.label, r.search, etc.
func (r Retrieve[D]) Where(filters ...Filter[D]) Retrieve[D] {
	bound := make([]gorp.Filter[string, Status[D]], len(filters))
	for i, f := range filters {
		bound[i] = f(r)
	}
	r.gorp = r.gorp.Where(bound...)
	return r
}

// MatchKeyPrefix returns a filter for statuses whose key starts with the provided prefix.
func MatchKeyPrefix[D any](prefix string) Filter[D] {
	return func(_ Retrieve[D]) gorp.Filter[string, Status[D]] {
		return gorp.Match(func(_ gorp.Context, s *Status[D]) (bool, error) {
			return strings.HasPrefix(s.Key, prefix), nil
		})
	}
}

// MatchVariants returns a filter for statuses with the given variants.
func MatchVariants[D any](variants ...status.Variant) Filter[D] {
	return func(_ Retrieve[D]) gorp.Filter[string, Status[D]] {
		return gorp.Match(func(_ gorp.Context, s *Status[D]) (bool, error) {
			return slices.Contains(variants, s.Variant), nil
		})
	}
}

// MatchLabels returns a filter for statuses that have any of the provided labels.
func MatchLabels[D any](matchLabels ...xlabel.Key) Filter[D] {
	return Match(func(ctx gorp.Context, r Retrieve[D], s *Status[D]) (bool, error) {
		labels, err := r.label.RetrieveFor(ctx, OntologyID(s.Key), ctx.Tx)
		if err != nil {
			return false, err
		}
		labelKeys := lo.Map(labels, func(l xlabel.Label, _ int) xlabel.Key { return l.Key })
		return lo.ContainsBy(labelKeys, func(l xlabel.Key) bool {
			return lo.Contains(matchLabels, l)
		}), nil
	})
}

// Exec executes the query and fills the results into the provided Status or slice of
// Statuses. It's important to note that fuzzy search will not be aware of any writes/
// deletes executed on the tx, and will only search the underlying database.
func (r Retrieve[D]) Exec(ctx context.Context, tx gorp.Tx) error {
	tx = gorp.OverrideTx(r.baseTX, tx)
	if r.searchTerm != "" {
		ids, err := r.search.Search(ctx, search.Request{
			Type: ontology.ResourceTypeStatus,
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
