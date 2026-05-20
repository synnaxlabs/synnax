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
	"strings"

	"github.com/synnaxlabs/x/compare"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
)

// Resolver provides on-demand lookups for symbols that live outside the
// in-memory symbol tree. Implementations are consulted by Symbol.Resolve
// and Symbol.Search when a name cannot be found among Children.
type Resolver interface {
	// Resolve looks up a symbol by exact name match. Returns a non-nil
	// error when the symbol does not exist.
	Resolve(ctx context.Context, name string) (*Symbol, error)
	// Search returns symbols matching the given search term. Implementations
	// should support fuzzy matching. Used for completion and "did you mean"
	// suggestions.
	Search(ctx context.Context, term string) ([]*Symbol, error)
}

// MapResolver is a simple map-based resolver for static symbol sets.
// Construction sites use this for built-in declarations. Each Resolve call
// returns a fresh *Symbol so callers may attach the result into a tree.
type MapResolver map[string]Symbol

var _ Resolver = (MapResolver)(nil)

// Resolve looks up a symbol by name in the map.
func (m MapResolver) Resolve(_ context.Context, name string) (*Symbol, error) {
	if s, ok := m[name]; ok {
		sym := s
		return &sym, nil
	}
	return nil, errors.Wrapf(query.ErrNotFound, "symbol %s not found", name)
}

// Search returns symbols matching the search term using prefix and fuzzy match.
func (m MapResolver) Search(_ context.Context, term string) ([]*Symbol, error) {
	var symbols []*Symbol
	for name, sym := range m {
		if strings.HasPrefix(name, term) {
			s := sym
			symbols = append(symbols, &s)
			continue
		}
		if len(term) > 2 && compare.LevenshteinDistance(name, term) <= 2 {
			s := sym
			symbols = append(symbols, &s)
		}
	}
	return symbols, nil
}

// Values returns the map's symbols as a flat slice. Useful for STL module
// builders that already have a MapResolver of members and want to install
// them under a parent.
func (m MapResolver) Values() []Symbol {
	out := make([]Symbol, 0, len(m))
	for _, sym := range m {
		out = append(out, sym)
	}
	return out
}

// CompoundResolver chains multiple resolvers, returning the first successful match.
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

// ModuleResolver handles qualified name resolution for a named module. It
// strips the "Name." prefix before delegating to Members.
type ModuleResolver struct {
	// Name is the module namespace (e.g., "math").
	Name string
	// Members contains the module's symbols keyed by bare name.
	Members MapResolver
}

var _ Resolver = (*ModuleResolver)(nil)

// Resolve looks up a symbol by qualified name. If the name has the prefix
// "Name.", it strips the prefix and delegates to Members.
func (m *ModuleResolver) Resolve(ctx context.Context, name string) (*Symbol, error) {
	bare, ok := strings.CutPrefix(name, m.Name+".")
	if !ok {
		return nil, errors.Wrapf(query.ErrNotFound, "symbol %s not found in module %s", name, m.Name)
	}
	return m.Members.Resolve(ctx, bare)
}

// Search returns symbols matching the given search term.
func (m *ModuleResolver) Search(ctx context.Context, term string) ([]*Symbol, error) {
	prefix := m.Name + "."
	if bare, ok := strings.CutPrefix(term, prefix); ok {
		return m.Members.Search(ctx, bare)
	}
	if strings.HasPrefix(prefix, term) {
		return m.Members.Search(ctx, "")
	}
	return m.Members.Search(ctx, term)
}
