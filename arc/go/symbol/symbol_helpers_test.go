// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol_test

import (
	"context"
	"strings"

	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/x/compare"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// staticResolver is a test-only Resolver backed by a map of bare names.
// Used only by tests that exercise the Resolver interface itself
// (CreateRoot/Resolve/Search behavior) and therefore need a concrete
// implementation. Production code uses real dynamic Resolvers; static
// symbols belong in the ambient prelude via (*Symbol).AttachToAmbient.
type staticResolver map[string]symbol.Symbol

func (r staticResolver) Resolve(_ context.Context, name string) (*symbol.Symbol, error) {
	if s, ok := r[name]; ok {
		sym := s
		return &sym, nil
	}
	return nil, errors.Wrapf(query.ErrNotFound, "symbol %s not found", name)
}

func (r staticResolver) Search(_ context.Context, term string) ([]*symbol.Symbol, error) {
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
