// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package hooks declares optional analyzer hook signatures that Symbols may
// carry. Lives in its own package to keep symbol.Symbol free of any analyzer
// import cycle.
package hooks

import "github.com/synnaxlabs/arc/parser"

// CallHook runs after the generic func-form validation passes. The ctx
// argument is typed as any to avoid an import cycle; implementations
// type-assert to the analyzer context they need.
type CallHook func(ctx any, funcCall parser.IFunctionCallSuffixContext)

// FlowConfigHook runs after the generic flow-form config validation passes.
// Same any-typed ctx as CallHook for the same reason.
type FlowConfigHook func(ctx any, config parser.IConfigValuesContext)
