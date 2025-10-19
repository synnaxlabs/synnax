// Copyright 2025 Synnax Labs, Inc.
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

type Resolver interface {
	Resolve(ctx context.Context, name string) (Symbol, error)
	// ResolvePrefix returns all symbols whose names start with the given prefix.
	// Used for completion and symbol browsing. Returns empty slice if no matches.
	ResolvePrefix(ctx context.Context, prefix string) ([]Symbol, error)
}

type MapResolver map[string]Symbol

var _ Resolver = (*MapResolver)(nil)

func (m MapResolver) Resolve(_ context.Context, name string) (Symbol, error) {
	if s, ok := m[name]; ok {
		return s, nil
	}
	return Symbol{}, errors.Wrapf(query.NotFound, "symbol %s not found", name)
}

func (m MapResolver) ResolvePrefix(_ context.Context, prefix string) ([]Symbol, error) {
	var symbols []Symbol
	for name, sym := range m {
		if strings.HasPrefix(name, prefix) {
			symbols = append(symbols, sym)
		}
	}
	return symbols, nil
}

type CompoundResolver []Resolver

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

func (c CompoundResolver) ResolvePrefix(ctx context.Context, prefix string) ([]Symbol, error) {
	seen := make(map[string]bool)
	var symbols []Symbol

	for _, resolver := range c {
		prefixSymbols, err := resolver.ResolvePrefix(ctx, prefix)
		if err != nil {
			continue // Skip resolvers that error
		}

		for _, sym := range prefixSymbols {
			// Deduplicate by name (first resolver wins)
			if !seen[sym.Name] {
				symbols = append(symbols, sym)
				seen[sym.Name] = true
			}
		}
	}

	return symbols, nil
}
