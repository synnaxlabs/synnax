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

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// Retrieve is used to retrieve statuses from the cluster using a builder pattern.
type Retrieve[D any] struct {
	baseTX gorp.Tx
	gorp   gorp.Retrieve[Key, Status[D]]
	// indexes are on the table, whose entries hold details of any type.
	indexes indexes
}

// Filter is a per-service filter that is bound to the Retrieve when passed to Where.
// Use Match to construct one from a closure.
type Filter[D any] func(r Retrieve[D]) gorp.Filter[Key, Status[D]]

// Match wraps a closure that needs the Retrieve into a Filter.
func Match[D any](
	f func(gorp.Context, Retrieve[D], *Status[D]) (bool, error),
) Filter[D] {
	return func(r Retrieve[D]) gorp.Filter[Key, Status[D]] {
		return gorp.Match(func(ctx gorp.Context, s *Status[D]) (bool, error) {
			return f(ctx, r, s)
		})
	}
}

// Entry binds the Status that Retrieve will fill results into. If multiple results
// match the query, only the first result will be filled into the provided Status.
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

// Where applies the provided filter to the query. A status must match every chained
// Where.
func (r Retrieve[D]) Where(filter Filter[D]) Retrieve[D] {
	r.gorp = r.gorp.Where(filter(r))
	return r
}

// MatchKeys returns a filter that restricts results to statuses whose key
// matches any of the provided values.
func MatchKeys[D any](keys ...Key) Filter[D] {
	return func(Retrieve[D]) gorp.Filter[Key, Status[D]] {
		return gorp.MatchKeys[Key, Status[D]](keys...)
	}
}

// MatchNames returns a filter for statuses whose name matches any of the provided
// values.
func MatchNames[D any](names ...string) Filter[D] {
	return func(Retrieve[D]) gorp.Filter[Key, Status[D]] {
		return gorp.Match(func(_ gorp.Context, s *Status[D]) (bool, error) {
			return slices.Contains(names, s.Name), nil
		})
	}
}

// MatchSearch returns a filter for statuses whose name matches term, closest match
// first. An empty term matches no status. Only a Retrieve[any] can search, because the
// search index holds details of any type.
func MatchSearch(term string) Filter[any] {
	return func(r Retrieve[any]) gorp.Filter[Key, Status[any]] {
		return r.indexes.search.Filter(term)
	}
}

// MatchVariants returns a filter for statuses with the given variants.
func MatchVariants[D any](variants ...Variant) Filter[D] {
	return func(Retrieve[D]) gorp.Filter[Key, Status[D]] {
		return gorp.Match(func(_ gorp.Context, s *Status[D]) (bool, error) {
			return slices.Contains(variants, s.Variant), nil
		})
	}
}

// MatchLabels returns a filter for statuses that have any of the provided labels.
func MatchLabels[D any](labels ...label.Key) Filter[D] {
	return func(r Retrieve[D]) gorp.Filter[Key, Status[D]] {
		return r.indexes.labels.FilterAs[Status[D]](labels...)
	}
}

// MatchOwners returns a filter for the statuses that the provided resources own.
func MatchOwners[D any](owners ...resource.ID) Filter[D] {
	targets := make([]gorp.Target, len(owners))
	for i, o := range owners {
		targets[i] = resourceTarget(o)
	}
	return func(r Retrieve[D]) gorp.Filter[Key, Status[D]] {
		return r.indexes.owner.FilterAs[Status[D]](targets...)
	}
}

// Exec executes the query and fills the results into the provided Status or slice of
// Statuses.
func (r Retrieve[D]) Exec(ctx context.Context, tx gorp.Tx) error {
	return r.gorp.Exec(ctx, gorp.OverrideTx(r.baseTX, tx))
}
