// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package compiler

import "github.com/synnaxlabs/arc/symbol"

type options struct {
	disableHostImports bool
	hostSymbols        symbol.Resolver
}

type Option func(o *options)

func DisableHostImport() Option {
	return func(o *options) { o.disableHostImports = true }
}

// WithHostSymbols provides a custom symbol resolver for host function type
// definitions. When set, the compiler uses this resolver instead of the default
// stdlib resolver. This allows the STL modules to serve as the single source of
// truth for host function signatures.
func WithHostSymbols(r symbol.Resolver) Option {
	return func(o *options) { o.hostSymbols = r }
}
