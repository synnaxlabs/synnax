// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package compiler

import (
	"github.com/synnaxlabs/slate/analyzer/symbol"
	"github.com/synnaxlabs/slate/compiler/wasm"
	"github.com/synnaxlabs/slate/types"
)

// Context maintains compilation state across all code generation
type Context struct {
	Module  *wasm.Module      // The WASM module being built
	Imports *wasm.ImportIndex // Imported host functions
	Symbols *symbol.Scope     // Symbol table from analyzer
	Current *FunctionContext  // Current function being compiled
}

// FunctionContext tracks state for the current function being compiled
type FunctionContext struct {
	Name       string            // Function/task name
	Locals     map[string]uint32 // Variable name → local index
	LocalTypes []wasm.ValueType  // Types of local variables (for WASM)
	NextLocal  uint32            // Next available local index
	TypeIdx    uint32            // Function type index in module
	ReturnType types.Type        // Return type (if any)

	// For tasks only
	IsTask    bool              // Whether this is a task (vs function)
	Stateful  map[string]uint32 // Stateful variable → storage key
	NextState uint32            // Next available state key
}

// NewContext creates a new compilation context
func NewContext(module *wasm.Module, symbols *symbol.Scope) *Context {
	return &Context{
		Module:  module,
		Imports: wasm.SetupTypedImports(module),
		Symbols: symbols,
	}
}

// EnterFunction prepares context for compiling a new function
func (c *Context) EnterFunction(name string, returnType types.Type) {
	c.Current = &FunctionContext{
		Name:       name,
		Locals:     make(map[string]uint32),
		LocalTypes: make([]wasm.ValueType, 0),
		NextLocal:  0,
		ReturnType: returnType,
		IsTask:     false,
	}
}

// EnterTask prepares context for compiling a new task
func (c *Context) EnterTask(name string, returnType types.Type) {
	c.Current = &FunctionContext{
		Name:       name,
		Locals:     make(map[string]uint32),
		LocalTypes: make([]wasm.ValueType, 0),
		NextLocal:  0,
		ReturnType: returnType,
		IsTask:     true,
		Stateful:   make(map[string]uint32),
		NextState:  0,
	}
}

// AllocateLocal allocates a new local variable slot
func (c *Context) AllocateLocal(name string, wasmType wasm.ValueType) uint32 {
	if c.Current == nil {
		panic("AllocateLocal called outside function context")
	}

	// Check if already allocated
	if idx, exists := c.Current.Locals[name]; exists {
		return idx
	}

	// Allocate new local
	idx := c.Current.NextLocal
	c.Current.Locals[name] = idx
	c.Current.LocalTypes = append(c.Current.LocalTypes, wasmType)
	c.Current.NextLocal++
	return idx
}

// GetLocal returns the local index for a variable
func (c *Context) GetLocal(name string) (uint32, bool) {
	if c.Current == nil {
		return 0, false
	}
	idx, exists := c.Current.Locals[name]
	return idx, exists
}

// AllocateStateful allocates a storage key for a stateful variable
func (c *Context) AllocateStateful(name string) uint32 {
	if c.Current == nil || !c.Current.IsTask {
		panic("AllocateStateful called outside task context")
	}

	// Check if already allocated
	if key, exists := c.Current.Stateful[name]; exists {
		return key
	}

	// Allocate new key
	key := c.Current.NextState
	c.Current.Stateful[name] = key
	c.Current.NextState++
	return key
}

// GetStateful returns the storage key for a stateful variable
func (c *Context) GetStateful(name string) (uint32, bool) {
	if c.Current == nil || !c.Current.IsTask {
		return 0, false
	}
	key, exists := c.Current.Stateful[name]
	return key, exists
}
