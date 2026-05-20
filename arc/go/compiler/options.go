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
	hostScope          *symbol.Symbol
}

type Option func(o *options)

func DisableHostImport() Option {
	return func(o *options) { o.disableHostImports = true }
}

// WithHostScope overrides the scope the compiler uses to look up host
// function type definitions. By default the compiler uses the analyzed
// program's root scope, which reaches STL modules via its ambient parent.
// Tests and specialized callers can pass an alternative scope here.
func WithHostScope(scope *symbol.Symbol) Option {
	return func(o *options) { o.hostScope = scope }
}
