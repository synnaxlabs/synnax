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
)

// Resolver provides pluggable symbol resolution for global or built-in symbols.
//
// Resolvers are used to provide symbols that are not defined in the source code but
// are available globally (e.g., built-in functions, constants, or runtime symbols).
// They can be attached to the root scope via CreateRootScope to make symbols available
// throughout the program.
type Resolver interface {
	// Resolve looks up a symbol by exact name match. Returns query.NotFound error
	// if the symbol does not exist.
	Resolve(ctx context.Context, name string) (Symbol, error)
	// Search returns symbols matching the given search term. Implementations should
	// support fuzzy matching. Used for completion and "did you mean" suggestions.
	Search(ctx context.Context, term string) ([]Symbol, error)
}

// MapResolver is a simple map-based resolver for static symbol sets.
type MapResolver map[string]Symbol

var _ Resolver = (*MapResolver)(nil)

// Resolve looks up a symbol by name in the map. Returns query.NotFound if not found.
func (m MapResolver) Resolve(_ context.Context, name string) (Symbol, error) {
	if s, ok := m[name]; ok {
		return s, nil
	}
	return Symbol{}, errors.Wrapf(query.ErrNotFound, "symbol %s not found", name)
}

// Search returns symbols matching the search term using prefix matching and fuzzy matching.
func (m MapResolver) Search(_ context.Context, term string) ([]Symbol, error) {
	var symbols []Symbol
	for name, sym := range m {
		// Prefix match
		if strings.HasPrefix(name, term) {
			symbols = append(symbols, sym)
			continue
		}
		// Fuzzy match using Levenshtein distance
		if len(term) > 2 && compare.LevenshteinDistance(name, term) <= 2 {
			symbols = append(symbols, sym)
		}
	}
	return symbols, nil
}

// CompoundResolver chains multiple resolvers, returning the first successful match.
// Used to combine multiple symbol sources (e.g., built-ins + runtime symbols).
type CompoundResolver []Resolver

// Resolve attempts resolution with each resolver in order, returning the first match.
// If no resolver matches, returns the error from the last resolver.
func (c CompoundResolver) Resolve(ctx context.Context, name string) (Symbol, error) {
	var (
		symbol Symbol
		err    error
	)
	for _, s := range c {
		symbol, err = s.Resolve(ctx, name)
		if err == nil {
			return symbol, nil
		}
	}
	return symbol, err
}

func (c CompoundResolver) Search(ctx context.Context, term string) ([]Symbol, error) {
	var (
		seen           = make(map[string]bool)
		symbols        []Symbol
		accumulatedErr error
	)
	for _, resolver := range c {
		results, err := resolver.Search(ctx, term)
		if err != nil {
			accumulatedErr = err
			continue
		}
		for _, sym := range results {
			if !seen[sym.Name] {
				symbols = append(symbols, sym)
				seen[sym.Name] = true
			}
		}
	}
	return symbols, accumulatedErr
}

// ModuleResolver handles qualified name resolution for a named module. It strips
// the "Name." prefix before delegating to Members.
type ModuleResolver struct {
	// Name is the module namespace (e.g., "math").
	Name string
	// Members contains the module's symbols keyed by bare name.
	Members MapResolver
}

var _ Resolver = (*ModuleResolver)(nil)

// Resolve looks up a symbol by qualified name. If the name has the prefix
// "Name.", it strips the prefix and delegates to Members. Otherwise it returns
// query.ErrNotFound.
func (m *ModuleResolver) Resolve(ctx context.Context, name string) (Symbol, error) {
	bare, ok := strings.CutPrefix(name, m.Name+".")
	if !ok {
		return Symbol{}, errors.Wrapf(query.ErrNotFound, "symbol %s not found in module %s", name, m.Name)
	}
	return m.Members.Resolve(ctx, bare)
}

// Search returns symbols matching the given search term. If the term has the
// prefix "Name.", it strips the prefix and delegates. If the term is a prefix
// of "Name.", it returns all members. Otherwise it delegates with the raw term
// for fuzzy matching.
func (m *ModuleResolver) Search(ctx context.Context, term string) ([]Symbol, error) {
	prefix := m.Name + "."
	if bare, ok := strings.CutPrefix(term, prefix); ok {
		return m.Members.Search(ctx, bare)
	}
	if strings.HasPrefix(prefix, term) {
		return m.Members.Search(ctx, "")
	}
	return m.Members.Search(ctx, term)
}
