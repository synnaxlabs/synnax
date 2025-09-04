// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package core

import (
	"github.com/synnaxlabs/slate/analyzer/symbol"
	"github.com/synnaxlabs/slate/compiler/wasm"
)

// Context maintains compilation state across all code generation
type Context struct {
	Imports *wasm.ImportIndex
	Scope   *symbol.Scope
	Writer  *wasm.Writer
}

// NewContext creates a new compilation context
func NewContext(imports *wasm.ImportIndex, symbols *symbol.Scope) *Context {
	return &Context{Imports: imports, Scope: symbols, Writer: wasm.NewWriter()}
}
