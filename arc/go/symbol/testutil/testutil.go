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

	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/x/compare"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// StaticResolver is a test-only map-backed implementation of
// symbol.Resolver. Tests use it to stand in for a real dynamic resolver
// (production cluster channels) when only a fixed snapshot is needed
// for analysis.
//
// Production cluster code implements its own symbol.Resolver against
// live channel state; StaticResolver is not part of that surface and
// must not be imported from production packages.
type StaticResolver map[string]symbol.Symbol

// Resolve looks up name in the underlying map. Returns query.ErrNotFound
// (wrapped) when the name is absent.
func (r StaticResolver) Resolve(_ context.Context, name string) (*symbol.Symbol, error) {
	if s, ok := r[name]; ok {
		sym := s
		return &sym, nil
	}
	return nil, errors.Wrapf(query.ErrNotFound, "symbol %s not found", name)
}

// Search returns entries whose name has the given prefix or is within a
// Levenshtein distance of 2 from the term (matching the prior
// MapResolver behavior that several test suites depend on).
func (r StaticResolver) Search(_ context.Context, term string) ([]*symbol.Symbol, error) {
	var results []*symbol.Symbol
	for name, sym := range r {
		if strings.HasPrefix(name, term) {
			s := sym
			results = append(results, &s)
			continue
		}
		if len(term) > 2 && compare.LevenshteinDistance(name, term) <= 2 {
			s := sym
			results = append(results, &s)
		}
	}
	return results, nil
}
