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

	"github.com/synnaxlabs/x/set"
)

// Resolver provides on-demand lookups for symbols that live outside the
// in-memory symbol tree. Implementations are consulted by Symbol.Resolve
// and Symbol.Search when a key cannot be found among Children. The
// canonical production use is cluster channels: name and ID both stable,
// but the set changes at runtime. Symbols whose identity is fixed at
// compile time go in the ambient prelude via (*Symbol).AttachToAmbient
// instead of behind a Resolver.
type Resolver interface {
	// Resolve looks up a symbol by name. Numeric-looking names are
	// equally valid keys — production resolvers typically index by both
	// name and ID and dispatch based on input format.
	Resolve(ctx context.Context, name string) (*Symbol, error)
	// Search returns symbols matching the given search term.
	// Implementations should support fuzzy matching. Used for completion
	// and "did you mean" suggestions.
	Search(ctx context.Context, term string) ([]*Symbol, error)
}

// CompoundResolver chains multiple resolvers, returning the first
// successful match. Used to compose multiple dynamic sources (e.g.
// cluster channels plus a temporary scratch set) behind one
// GlobalResolver.
type CompoundResolver []Resolver

var _ Resolver = (CompoundResolver)(nil)

// Resolve attempts resolution with each resolver in order, returning the first match.
func (c CompoundResolver) Resolve(ctx context.Context, name string) (*Symbol, error) {
	var (
		sym *Symbol
		err error
	)
	for _, r := range c {
		sym, err = r.Resolve(ctx, name)
		if err == nil {
			return sym, nil
		}
	}
	return sym, err
}

// Search aggregates results across all chained resolvers, deduplicating by name.
func (c CompoundResolver) Search(ctx context.Context, term string) ([]*Symbol, error) {
	var (
		seen           = make(set.Set[string])
		symbols        []*Symbol
		accumulatedErr error
	)
	for _, resolver := range c {
		results, err := resolver.Search(ctx, term)
		if err != nil {
			accumulatedErr = err
			continue
		}
		for _, sym := range results {
			if !seen.Contains(sym.Name) {
				symbols = append(symbols, sym)
				seen.Add(sym.Name)
			}
		}
	}
	return symbols, accumulatedErr
}
