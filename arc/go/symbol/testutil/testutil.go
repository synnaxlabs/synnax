// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package testutil provides test-only helpers around the symbol
// package. It is excluded from production code by convention: the only
// callers are *_test.go files and ginkgo suites.
package testutil

import (
	"context"
	"strings"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/x/compare"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// NewRoot builds a test root with all STL symbols plus extras attached
// to the ambient prelude. extras are passed by value (taken by reference
// internally) so tests can pass their per-test []symbol.Symbol resolver
// slices directly without a conversion loop.
func NewRoot(resolver symbol.Resolver, extras ...symbol.Symbol) *symbol.Symbol {
	stlSyms := stl.NewSymbols()
	syms := make([]*symbol.Symbol, 0, len(stlSyms)+len(extras))
	syms = append(syms, stlSyms...)
	for i := range extras {
		s := extras[i]
		syms = append(syms, &s)
	}
	return symbol.NewRoot(resolver, syms)
}

// BeAValidationPathError matches a validate.PathError wrapping validate.ErrValidation,
// which PathError has no Unwrap for, so MatchError(validate.ErrValidation) misses it.
func BeAValidationPathError() types.GomegaMatcher {
	return gomega.MatchError(func(err error) bool {
		var pathErr validate.PathError
		return errors.As(err, &pathErr) && errors.Is(pathErr.Err, validate.ErrValidation)
	}, "be a validation path error")
}

// NewGraphRoot is NewRoot plus symbol.AutoImportModules. Graph-mode tests
// use this to install KindModuleAlias children on the root so qualified
// names like "math.avg" resolve without an explicit import.
func NewGraphRoot(resolver symbol.Resolver, extras ...symbol.Symbol) *symbol.Symbol {
	root := NewRoot(resolver, extras...)
	symbol.AutoImportModules(root)
	return root
}

// FirstChildOfKind returns the first direct child of s with the given kind.
// Returns query.ErrNotFound when no matching child exists. Used by tests
// that need to drill into a known scope shape (e.g. a function's block
// child); production code resolves by name or kind-filter instead.
func FirstChildOfKind(s *symbol.Symbol, kind symbol.Kind) (*symbol.Symbol, error) {
	for _, child := range s.Children() {
		if child.Kind == kind {
			return child, nil
		}
	}
	return nil, errors.Wrap(query.ErrNotFound, "undefined symbol")
}

// StaticResolver is a test-only slice-backed implementation of
// symbol.Resolver. Tests use it to stand in for a real dynamic resolver
// (production cluster channels) when only a fixed snapshot is needed
// for analysis. Mutate via Add and Remove when a test needs to simulate
// external state changes.
//
// Production cluster code implements its own symbol.Resolver against
// live channel state; StaticResolver is not part of that surface and
// must not be imported from production packages.
type StaticResolver []symbol.Symbol

// Add appends s to the resolver. Uses a pointer receiver so the mutation
// is visible to callers that later read the resolver via a captured
// variable (e.g., LSP tests that simulate external changes).
func (r *StaticResolver) Add(s symbol.Symbol) { *r = append(*r, s) }

// Remove deletes the first entry whose Name matches name. No-op when no
// entry matches.
func (r *StaticResolver) Remove(name string) {
	for i := range *r {
		if (*r)[i].Name == name {
			*r = append((*r)[:i], (*r)[i+1:]...)
			return
		}
	}
}

// Resolve looks up name by scanning the slice. Returns query.ErrNotFound
// (wrapped) when the name is absent.
func (r StaticResolver) Resolve(_ context.Context, name string) (*symbol.Symbol, error) {
	for i := range r {
		if r[i].Name == name {
			sym := r[i]
			return &sym, nil
		}
	}
	return nil, errors.Wrapf(query.ErrNotFound, "symbol %s not found", name)
}

// Search returns entries whose name has the given prefix or is within a
// Levenshtein distance of 2 from the term.
func (r StaticResolver) Search(_ context.Context, term string) ([]*symbol.Symbol, error) {
	var results []*symbol.Symbol
	for i := range r {
		name := r[i].Name
		if strings.HasPrefix(name, term) {
			s := r[i]
			results = append(results, &s)
			continue
		}
		if len(term) > 2 && compare.LevenshteinDistance(name, term) <= 2 {
			s := r[i]
			results = append(results, &s)
		}
	}
	return results, nil
}
