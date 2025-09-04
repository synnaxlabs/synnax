// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package expression

import (
	"github.com/synnaxlabs/slate/analyzer/symbol"
	"github.com/synnaxlabs/slate/types"
	"github.com/synnaxlabs/x/errors"
)

// compileIdentifier compiles variable references
func (e *Compiler) compileIdentifier(name string) (types.Type, error) {
	// First, look up the symbol in the symbol table to get its type
	scope, err := e.ctx.Symbols.Get(name)
	if err != nil {
		return nil, errors.Wrapf(err, "identifier '%s' not found", name)
	}
	
	if scope.Symbol == nil {
		return nil, errors.Newf("identifier '%s' has no symbol information", name)
	}
	
	sym := scope.Symbol
	
	switch sym.Kind {
	case symbol.KindVariable, symbol.KindStatefulVariable:
		// Local variable - should be in our local map
		if idx, ok := e.ctx.GetLocal(name); ok {
			e.encoder.WriteLocalGet(idx)
			return sym.Type, nil
		}
		// If not in locals, it might be a stateful variable
		if e.ctx.Current != nil && e.ctx.Current.IsTask {
			if key, ok := e.ctx.GetStateful(name); ok {
				// Load stateful variable
				e.emitStatefulLoad(key, sym.Type)
				return sym.Type, nil
			}
		}
		return nil, errors.Newf("variable '%s' not found in local context", name)
		
	case symbol.KindParam:
		// Function/task parameter - should be in locals
		if idx, ok := e.ctx.GetLocal(name); ok {
			e.encoder.WriteLocalGet(idx)
			return sym.Type, nil
		}
		return nil, errors.Newf("parameter '%s' not found in local context", name)
		
	case symbol.KindChannel:
		// Channel reference - for non-blocking read
		if idx, ok := e.ctx.GetLocal(name); ok {
			// Channel ID is stored as a local
			e.encoder.WriteLocalGet(idx)
			// Emit non-blocking channel read
			e.emitChannelRead(sym.Type)
			return sym.Type, nil
		}
		return nil, errors.Newf("channel '%s' not accessible", name)
		
	case symbol.KindFunction, symbol.KindTask:
		// Functions and tasks can't be used as values
		return nil, errors.Newf("'%s' is a %v and cannot be used as a value", 
			name, sym.Kind)
		
	default:
		return nil, errors.Newf("unsupported symbol kind: %v for '%s'", sym.Kind, name)
	}
}

// emitStatefulLoad emits code to load a stateful variable
func (e *Compiler) emitStatefulLoad(key uint32, t types.Type) {
	// Push task ID (0 for now - would be provided at runtime)
	e.encoder.WriteI32Const(0)
	// Push variable key
	e.encoder.WriteI32Const(int32(key))
	
	// Call appropriate state load function based on type
	importIdx := e.ctx.Imports.GetStateLoad(t)
	e.encoder.WriteCall(importIdx)
}

// emitChannelRead emits code for non-blocking channel read
func (e *Compiler) emitChannelRead(t types.Type) {
	// Stack has channel ID
	// Call appropriate channel read function based on type
	importIdx := e.ctx.Imports.GetChannelRead(t)
	e.encoder.WriteCall(importIdx)
}
