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
	"bytes"
	"fmt"
	"strings"

	"github.com/synnaxlabs/slate/analyzer"
	"github.com/synnaxlabs/slate/compiler/wasm"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/slate/resolver"
)

// Compiler compiles Slate source code to WASM and metadata.
type Compiler struct {
	resolver resolver.GlobalResolver

	// Code generation
	wasmModule *wasm.Module
	imports    *wasm.ImportIndex
}

// New creates a new compiler with the given resolver.
func New(r resolver.GlobalResolver) *Compiler {
	if r == nil {
		r = resolver.NewStubResolver()
	}
	return &Compiler{
		resolver: r,
	}
}

// Compile compiles Slate source code to a Module.
func (c *Compiler) Compile(source string) (*analyzer.Module, error) {
	// Parse
	ast, err := parser.Parse(source)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// Analyze with resolver - this extracts all metadata
	a := analyzer.NewAnalyzer(c.resolver)
	analysis := a.Analyze(ast)

	// Check for analysis errors
	for _, diag := range analysis.Diagnostics {
		if diag.Severity == analyzer.SeverityError {
			return nil, fmt.Errorf("analysis error at %d:%d: %s",
				diag.Line, diag.Column, diag.Message)
		}
	}

	// Generate WASM code from analysis results
	wasmBytes, err := c.generateCode(analysis)
	if err != nil {
		return nil, fmt.Errorf("code generation error: %w", err)
	}

	// Build final module
	return &analyzer.Module{
		Version:   "1.0.0",
		Module:    wasmBytes,
		Tasks:     analysis.Tasks,
		Functions: analysis.Functions,
		Nodes:     analysis.FlowGraph.Nodes,
		Edges:     analysis.FlowGraph.Edges,
	}, nil
}

// generateCode generates WASM code from analysis results.
func (c *Compiler) generateCode(analysis *analyzer.Result) ([]byte, error) {
	c.wasmModule = wasm.NewModule()

	// Setup imports for all typed host functions
	if err := c.setupImports(); err != nil {
		return nil, err
	}

	// Generate code for each task
	for _, task := range analysis.Tasks {
		if err := c.generateTaskCode(task); err != nil {
			return nil, err
		}
	}

	// Generate code for each function
	for _, function := range analysis.Functions {
		if err := c.generateFunctionCode(function); err != nil {
			return nil, err
		}
	}

	return c.wasmModule.Generate(), nil
}

// setupImports adds all typed host function imports.
func (c *Compiler) setupImports() error {
	// Use the typed imports setup from wasm package
	c.imports = wasm.SetupTypedImports(c.wasmModule)
	return nil
}

// generateTaskCode generates WASM code for a task.
func (c *Compiler) generateTaskCode(spec analyzer.TaskSpec) error {
	// Create function type for the task
	// Tasks take (task_id: i32, args...) and return result if any
	params := []wasm.ValueType{wasm.I32} // task ID

	// AddSymbol runtime argument types
	for _, arg := range spec.Args {
		params = append(params, slateTypeToWASM(arg.Type))
	}

	// AddSymbol return type if present
	var results []wasm.ValueType
	if spec.Return != nil {
		results = append(results, slateTypeToWASM(*spec.Return))
	}

	funcType := wasm.FunctionType{
		Params:  params,
		Results: results,
	}

	typeIdx := c.wasmModule.AddType(funcType)

	// Generate function body
	var body bytes.Buffer

	// 1. Load stateful variables at the start
	for _, stateVar := range spec.StatefulVars {
		// Get the appropriate load function for the type
		typeSuffix := getTypeSuffix(stateVar.Type)
		if loadFunc, ok := c.imports.StateLoad[typeSuffix]; ok {
			// local.get 0 (task ID)
			body.WriteByte(byte(wasm.OpLocalGet))
			wasm.WriteLEB128(&body, 0)

			// i32.const var_id
			wasm.WriteI32Const(&body, int32(stateVar.Key))

			// call state_load_<type>
			body.WriteByte(byte(wasm.OpCall))
			wasm.WriteLEB128(&body, uint64(loadFunc))

			// Store in local variable
			// TODO: Need to track local variable indices
		}
	}

	// 2. TODO: Generate task body code
	// This would involve walking the task's AST and generating
	// appropriate WASM instructions

	// 3. Store stateful variables at the end
	for _, stateVar := range spec.StatefulVars {
		typeSuffix := getTypeSuffix(stateVar.Type)
		if storeFunc, ok := c.imports.StateStore[typeSuffix]; ok {
			// local.get 0 (task ID)
			body.WriteByte(byte(wasm.OpLocalGet))
			wasm.WriteLEB128(&body, 0)

			// i32.const var_id
			wasm.WriteI32Const(&body, int32(stateVar.Key))

			// TODO: Get value from local variable

			// call state_store_<type>
			body.WriteByte(byte(wasm.OpCall))
			wasm.WriteLEB128(&body, uint64(storeFunc))
		}
	}

	// AddSymbol function to module
	function := wasm.Function{
		Name:    spec.Key,
		TypeIdx: typeIdx,
		Locals:  []wasm.ValueType{}, // TODO: Calculate needed locals
		Body:    body.Bytes(),
		Export:  true,
	}

	c.wasmModule.AddFunction(function)

	return nil
}

// generateFunctionCode generates WASM code for a function.
func (c *Compiler) generateFunctionCode(spec analyzer.FunctionSpec) error {
	// Create function type
	var params []wasm.ValueType
	for _, param := range spec.Params {
		params = append(params, slateTypeToWASM(param.Type))
	}

	var results []wasm.ValueType
	if spec.Return != nil {
		results = append(results, slateTypeToWASM(*spec.Return))
	}

	funcType := wasm.FunctionType{
		Params:  params,
		Results: results,
	}

	typeIdx := c.wasmModule.AddType(funcType)

	// Generate function body
	var body bytes.Buffer

	// TODO: Generate function body code
	// This would involve walking the function's AST and generating
	// appropriate WASM instructions

	// For now, just return a default value if needed
	if len(results) > 0 {
		switch results[0] {
		case wasm.I32:
			wasm.WriteI32Const(&body, 0)
		case wasm.I64:
			body.WriteByte(byte(wasm.OpI64Const))
			wasm.WriteLEB128(&body, 0)
		case wasm.F32:
			wasm.WriteF32Const(&body, 0.0)
		case wasm.F64:
			wasm.WriteF64Const(&body, 0.0)
		}
	}

	// AddSymbol function to module
	function := wasm.Function{
		Name:    spec.Key,
		TypeIdx: typeIdx,
		Locals:  []wasm.ValueType{}, // TODO: Calculate needed locals
		Body:    body.Bytes(),
		Export:  true,
	}

	c.wasmModule.AddFunction(function)

	return nil
}

// slateTypeToWASM converts a Slate type to a WASM value type.
func slateTypeToWASM(slateType string) wasm.ValueType {
	switch slateType {
	case "i8", "i16", "i32", "u8", "u16", "u32", "bool":
		return wasm.I32
	case "i64", "u64", "timestamp":
		return wasm.I64
	case "f32":
		return wasm.F32
	case "f64":
		return wasm.F64
	case "string":
		return wasm.I32 // String handle
	default:
		// For series and channels, return handle (i32)
		if strings.HasPrefix(slateType, "series") || strings.HasPrefix(slateType, "channel") {
			return wasm.I32
		}
		return wasm.I32 // Default to i32 for unknown types
	}
}

// getTypeSuffix extracts the type suffix for host function selection.
func getTypeSuffix(slateType string) string {
	// Handle basic types
	switch slateType {
	case "i8", "i16", "i32", "i64", "u8", "u16", "u32", "u64", "f32", "f64", "bool", "string":
		if slateType == "bool" {
			return "u8" // Booleans are stored as u8
		}
		return slateType
	}

	// Handle series types (e.g., "series<f64>" -> "f64")
	if strings.HasPrefix(slateType, "series<") && strings.HasSuffix(slateType, ">") {
		inner := slateType[7 : len(slateType)-1]
		return inner
	}

	// Handle channel types (e.g., "channel<f64>" -> "f64")
	if strings.HasPrefix(slateType, "channel<") && strings.HasSuffix(slateType, ">") {
		inner := slateType[8 : len(slateType)-1]
		return inner
	}

	// Default to f64 for unknown types
	return "f64"
}
