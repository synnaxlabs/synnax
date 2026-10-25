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

	"github.com/synnaxlabs/gorp"
)

// Retrieve is a query builder for retrieving symbols. It should not be instantiated
// directly, and should instead be instantiated via the NewRetrieve method on
// symbol.Service.
type Retrieve struct {
	baseTX gorp.Tx
	gorp   gorp.Retrieve[Key, Symbol]
	search *gorp.SearchIndex[Key, Symbol]
}

// Where applies the provided filter to the query. To compose multiple filters,
// chain Where calls or pass a combined filter via gorp.And / gorp.Or.
func (r Retrieve) Where(filter gorp.Filter[Key, Symbol]) Retrieve {
	r.gorp = r.gorp.Where(filter)
	return r
}

// MatchKeys returns a filter that restricts results to symbols whose key
// matches any of the provided values.
func MatchKeys(keys ...Key) gorp.Filter[Key, Symbol] {
	return gorp.MatchKeys[Key, Symbol](keys...)
}

// Search filters the symbols to those whose name matches term, closest match first. An
// empty term adds no filter.
func (r Retrieve) Search(term string) Retrieve {
	if term == "" {
		return r
	}
	r.gorp = r.gorp.Where(r.search.Filter(term))
	return r
}

// Entry binds the given symbol to the query. This pointer is where the results of the
// query will be stored after Exec is called.
func (r Retrieve) Entry(symbol *Symbol) Retrieve {
	r.gorp = r.gorp.Entry(symbol)
	return r
}

// Entries binds the given slice of symbols to the query. This pointer is where the
// results of the query will be stored after Exec is called.
func (r Retrieve) Entries(symbols *[]Symbol) Retrieve {
	r.gorp = r.gorp.Entries(symbols)
	return r
}

// Exec executes the query against the given transaction. The results of the query will
// be stored in the pointer given to the Entry or Entries method. If tx is nil, the
// query will be executed directly against the underlying gorp.DB provided to the symbol
// service.
func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	return r.gorp.Exec(ctx, gorp.OverrideTx(r.baseTX, tx))
}
