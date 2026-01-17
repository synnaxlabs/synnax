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
	// ResolvePrefix returns all symbols whose names start with the given prefix.
	// Used for completion and symbol browsing. Returns empty slice if no matches.
	ResolvePrefix(ctx context.Context, prefix string) ([]Symbol, error)
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

// ResolvePrefix returns all symbols whose names start with the given prefix.
func (m MapResolver) ResolvePrefix(_ context.Context, prefix string) ([]Symbol, error) {
	var symbols []Symbol
	for name, sym := range m {
		if strings.HasPrefix(name, prefix) {
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

// ResolvePrefix collects symbols from all resolvers, deduplicating by name.
// The first resolver's symbol wins when multiple resolvers provide the same name.
// Returns the last error encountered, if any, even if some symbols were found.
func (c CompoundResolver) ResolvePrefix(ctx context.Context, prefix string) ([]Symbol, error) {
	var (
		seen           = make(map[string]bool)
		symbols        []Symbol
		accumulatedErr error
	)
	for _, resolver := range c {
		prefixSymbols, err := resolver.ResolvePrefix(ctx, prefix)
		if err != nil {
			accumulatedErr = err
			continue
		}
		for _, sym := range prefixSymbols {
			if !seen[sym.Name] {
				symbols = append(symbols, sym)
				seen[sym.Name] = true
			}
		}
	}
	return symbols, accumulatedErr
}
