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
	"github.com/synnaxlabs/arc/compiler/runtime"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/symbol"
)

// Context maintains compilation state across all code generation
type Context struct {
	Imports *runtime.ImportIndex
	Scope   *symbol.Scope
	Writer  *wasm.Writer
	Module  *wasm.Module
}

func (c *Context) WithScope(scope *symbol.Scope) *Context {
	return &Context{Scope: scope, Module: c.Module, Writer: c.Writer, Imports: c.Imports}
}

func (c *Context) WithNewWriter() *Context {
	return &Context{Writer: wasm.NewWriter(), Module: c.Module, Scope: c.Scope, Imports: c.Imports}
}

func NewContext(
	symbols *symbol.Scope,
	disableHostImports bool,
) *Context {
	mod := wasm.NewModule()
	ctx := &Context{Module: mod, Scope: symbols, Writer: wasm.NewWriter()}
	if !disableHostImports {
		ctx.Imports = runtime.SetupImports(mod)
	}
	return ctx
}
