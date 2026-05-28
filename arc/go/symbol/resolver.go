// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import "context"

// Resolver provides on-demand lookups for symbols that live outside the
// in-memory symbol tree. Implementations are consulted by Symbol.Resolve
// and Symbol.Search when a key cannot be found among Children. The
// canonical production use is cluster channels: name and ID both stable,
// but the set changes at runtime. Symbols whose identity is fixed at
// compile time are passed to CreateRoot as ambient globals instead of
// behind a Resolver.
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
