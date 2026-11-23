// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package compiler implements the Arc language compiler, translating Arc intermediate
// representation (IR) into WebAssembly bytecode.
//
// The compiler follows a multi-stage pipeline:
//  1. Expression compilation (recursive descent with operator precedence)
//  2. Statement compilation (control flow, variables, assignments)
//  3. WebAssembly module generation (types, imports, functions, exports)
//  4. Bytecode emission (WASM instruction encoding)
//
// The compiler maintains type information through the compilation process and
// generates calls to host functions for channel operations, series manipulation,
// and state persistence.
//
// Example usage:
//
//	output, err := compiler.Compile(ctx, program)
//	if err != nil {
//	    return err
//	}
//	// output.WASM contains the compiled WebAssembly bytecode
//	// output.OutputMemoryBases contains memory addresses for multi-output functions
package compiler

import (
	"context"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	ccontext "github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/compiler/expression"
	"github.com/synnaxlabs/arc/compiler/resolve"
	"github.com/synnaxlabs/arc/compiler/statement"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

// ScaleSyntheticPrefix marks a synthesized function that converts its input to the
// unit of its output.
const ScaleSyntheticPrefix = "scale$"

type compiledFunction struct {
	scopeName string
	typeIdx   uint32
	locals    []wasm.ValueType
	writer    *wasm.Writer
}

// Compile translates an Arc program from intermediate representation (IR) to
// WebAssembly bytecode.
//
// The function processes each function and stage in the IR, allocating memory for
// multi-output stages, compiling expressions and statements into WASM instructions, and
// generating a complete WASM module with type signatures, imports, functions, and
// exports.
//
// Parameters:
//   - ctx_: Go context for cancellation
//   - program: Arc IR containing functions, stages, symbol table, and type information
//   - opts: Optional compiler configuration (e.g., DisableHostImport())
//
// Returns:
//   - Output: Contains WASM bytecode and memory base addresses for multi-output stages
//   - error: Compilation errors (type mismatches, undefined symbols,
//     invalid operations)
//
// The compiler maintains type safety by propagating type hints through expression
// compilation and emitting type conversions when necessary.
func Compile(ctx context.Context, program ir.IR, opts ...Option) (Output, error) {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	resolver := resolve.NewResolver()

	compCtx := ccontext.NewRoot(ctx, program.Symbols, program.TypeMap, resolver)
	compCtx.Config = o.config
	compCtx.FmtStrings = program.FmtStrings

	for i, f := range program.Functions {
		resolver.RegisterLocal(f.Key, uint32(i))
	}

	mem := &outputMemory{next: 0x1000, bases: make(map[string]uint32)}

	var (
		compiled     []compiledFunction
		batchTargets []batchTarget
	)
	for _, i := range program.Functions {
		if t, ok := batchTargetFor(i); ok {
			batchTargets = append(batchTargets, t)
		}
		if cf, ok, err := compileSynthetic(compCtx, i, program.Functions); ok {
			if err != nil {
				return Output{}, err
			}
			compiled = append(compiled, cf)
			continue
		}
		cf, err := compileUserFunction(compCtx, i, mem)
		if err != nil {
			return Output{}, err
		}
		compiled = append(compiled, cf)
	}

	for _, t := range batchTargets {
		compiled = append(compiled, compileBatchWrapper(compCtx, t))
	}
	indexFunctions := compileIndexFunctions(compCtx, len(compiled))

	resolver.FinalizeAndPatch(compCtx.Module)

	return Output{
		WASM:              generateModule(compCtx.Module, compiled, indexFunctions),
		OutputMemoryBases: mem.bases,
	}, nil
}

// generateModule adds the compiled functions to m, exports each one but the index
// functions, and returns the WASM bytes.
func generateModule(
	m *wasm.Module,
	compiled, indexFunctions []compiledFunction,
) []byte {
	for _, cf := range compiled {
		funcIdx := m.AddFunction(cf.typeIdx, cf.locals, cf.writer.Bytes())
		m.AddExport(cf.scopeName, wasm.ExportKindFunc, funcIdx)
	}
	for _, cf := range indexFunctions {
		m.AddFunction(cf.typeIdx, cf.locals, cf.writer.Bytes())
	}
	m.EnableMemory()
	m.AddExport("memory", wasm.ExportKindMemory, 0)
	return m.Generate()
}

// compileSynthetic compiles fn when the analyzer synthesized it. It returns false when
// fn is a user-written function.
func compileSynthetic(
	rootCtx ccontext.Context[antlr.ParserRuleContext],
	fn ir.Function,
	functions ir.Functions,
) (compiledFunction, bool, error) {
	switch {
	case strings.HasPrefix(fn.Key, ir.DispatcherSyntheticPrefix):
		cf, err := compileDispatcherSynthetic(rootCtx, fn, functions)
		return cf, true, err
	case strings.HasPrefix(fn.Key, ScaleSyntheticPrefix):
		cf, err := compileScaleSynthetic(rootCtx, fn)
		return cf, true, err
	default:
		return compiledFunction{}, false, nil
	}
}

// outputMemory hands out the memory regions that hold the named outputs of each
// function.
type outputMemory struct {
	next  uint32
	bases map[string]uint32
}

func compileUserFunction(
	rootCtx ccontext.Context[antlr.ParserRuleContext],
	fn ir.Function,
	mem *outputMemory,
) (compiledFunction, error) {
	defaultOutput, hasDefaultOutput := fn.Outputs.Get(ir.DefaultOutputParam)
	hasNamedOutputs := len(fn.Outputs) > 1 ||
		(len(fn.Outputs) == 1 && !hasDefaultOutput)
	if !hasNamedOutputs {
		return compileItem(rootCtx, fn, fn.Inputs, defaultOutput.Type, 0)
	}
	base := mem.next
	mem.bases[fn.Key] = base
	size, err := outputMemorySize(fn)
	if err != nil {
		return compiledFunction{}, err
	}
	mem.next += size
	return compileItem(rootCtx, fn, fn.Inputs, types.Type{}, base)
}

// outputMemorySize returns the bytes that the named outputs of fn take in memory,
// including the 8-byte dirty mask.
func outputMemorySize(fn ir.Function) (uint32, error) {
	var size uint32 = 8
	for _, oParam := range fn.Outputs {
		if !oParam.Type.IsValid() {
			return 0, errors.Newf(
				"function %s has output %q with unresolved type",
				fn.Key, oParam.Name,
			)
		}
		if oParam.Type.Kind == types.KindString {
			size += 4
		} else {
			size += uint32(oParam.Type.Density())
		}
	}
	return size, nil
}

func compileItem(
	rootCtx ccontext.Context[antlr.ParserRuleContext],
	fn ir.Function,
	params types.Params,
	results types.Type,
	outputMemoryBase uint32,
) (compiledFunction, error) {
	key, body, outputs := fn.Key, fn.Body.AST, fn.Outputs
	scope, err := functionScope(fn)
	if err != nil {
		return compiledFunction{}, err
	}
	wasmParams := make([]wasm.ValueType, 0, len(params))
	for _, param := range params {
		wasmParams = append(wasmParams, wasm.ConvertType(param.Type))
	}
	var wasmResults []wasm.ValueType
	if results.IsValid() {
		wasmResults = append(wasmResults, wasm.ConvertType(results))
	}
	ctx := rootCtx.Child(body).WithScope(scope).WithNewWriter()
	ctx.Outputs = outputs
	ctx.OutputMemoryBase = outputMemoryBase

	funcT := wasm.FunctionType{Params: wasmParams, Results: wasmResults}
	typeIdx := ctx.Module.AddType(funcT)
	// Recursion has no loop, so each call also takes from the budget.
	ctx.Writer.WriteLoopBudget(ctx.LoopBudget)

	if outputMemoryBase > 0 {
		ctx.Writer.WriteI32Const(int32(outputMemoryBase))
		ctx.Writer.WriteI64Const(0)
		ctx.Writer.WriteMemoryOp(wasm.OpI64Store, 3, 0)
	}

	if err = compileBody(ctx, key, body, results); err != nil {
		return compiledFunction{}, err
	}

	return compiledFunction{
		scopeName: key,
		typeIdx:   typeIdx,
		locals:    collectLocals(ctx.Scope),
		writer:    ctx.Writer,
	}, nil
}

func compileBody(
	ctx ccontext.Context[antlr.ParserRuleContext],
	key string,
	body antlr.ParserRuleContext,
	results types.Type,
) error {
	switch body := body.(type) {
	case parser.IBlockContext:
		if _, err := statement.CompileBlock(ctx.Child(body)); err != nil {
			return errors.Wrapf(
				err,
				"failed to compile function '%s' body",
				ctx.Scope.Name,
			)
		}
	case parser.IExpressionContext:
		t, err := expression.Compile(ctx.Child(body))
		if err == nil && types.UnitsConvertible(t.Unit, results.Unit) {
			err = expression.EmitCast(ctx, t, results)
		}
		if err != nil {
			return errors.Wrapf(
				err,
				"failed to compile expression '%s'",
				ctx.Scope.Name,
			)
		}
	case parser.IFunctionContext:
		var buf [2]string
		parts := parser.AppendFunctionNameParts(buf[:0], body)
		if _, err := expression.CompileCall(
			ctx.Child(body), parts, symbol.FunctionArguments(body),
		); err != nil {
			return errors.Wrapf(err, "failed to compile call '%s'", ctx.Scope.Name)
		}
	default:
		return errors.Newf("unsupported body type for '%s'", key)
	}
	return nil
}

// functionScope returns the symbol fn compiles in. A function lowered from a module
// keeps its own root, so resolving fn.Key against the program root would miss it.
func functionScope(fn ir.Function) (*symbol.Symbol, error) {
	if fn.Symbol == nil {
		return nil, errors.Newf("function %q has no symbol", fn.Key)
	}
	return fn.Symbol, nil
}

// compileScaleSynthetic emits a WASM body that returns its one input in the type and
// unit of its output. An integer truncates toward zero.
func compileScaleSynthetic(
	rootCtx ccontext.Context[antlr.ParserRuleContext],
	fn ir.Function,
) (compiledFunction, error) {
	if len(fn.Inputs) != 1 || len(fn.Outputs) != 1 {
		return compiledFunction{}, errors.Newf(
			"unit conversion %s takes one input and one output", fn.Key,
		)
	}
	in, out := fn.Inputs[0].Type, fn.Outputs[0].Type
	if in.Kind == types.KindVarRef && in.Elem != nil {
		in = *in.Elem
	}
	if !types.UnitsConvertible(in.Unit, out.Unit) {
		return compiledFunction{}, errors.Newf(
			"unit conversion %s cannot convert %s to %s", fn.Key, in, out,
		)
	}
	ctx := rootCtx.WithNewWriter()
	typeIdx := ctx.Module.AddType(wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.ConvertType(in)},
		Results: []wasm.ValueType{wasm.ConvertType(out)},
	})
	ctx.Writer.WriteLocalGet(0)
	if err := expression.EmitCast(ctx, in, out); err != nil {
		return compiledFunction{}, err
	}
	return compiledFunction{
		scopeName: fn.Key,
		typeIdx:   typeIdx,
		writer:    ctx.Writer,
	}, nil
}

// compileDispatcherSynthetic emits a $sel-keyed if/else chain that calls exactly
// one branch function per evaluation. Trigger params are gating-only, unused.
func compileDispatcherSynthetic(
	rootCtx ccontext.Context[antlr.ParserRuleContext],
	fn ir.Function,
	functions ir.Functions,
) (compiledFunction, error) {
	ctx := rootCtx.WithNewWriter()
	params := make([]wasm.ValueType, 0, len(fn.Inputs))
	slots := make(map[string]int, len(fn.Inputs))
	for i, p := range fn.Inputs {
		params = append(params, wasm.ConvertType(p.Type))
		slots[p.Name] = i
	}
	out, ok := fn.Outputs.Get(ir.DefaultOutputParam)
	if !ok {
		return compiledFunction{}, errors.Newf("dispatcher %s has no output", fn.Key)
	}
	selSlot, ok := slots["$sel"]
	if !ok {
		return compiledFunction{}, errors.Newf(
			"dispatcher %s has no $sel input",
			fn.Key,
		)
	}
	funcT := wasm.FunctionType{
		Params:  params,
		Results: []wasm.ValueType{wasm.ConvertType(out.Type)},
	}
	typeIdx := ctx.Module.AddType(funcT)
	blockT := blockTypeFor(wasm.ConvertType(out.Type))
	keys := strings.Split(fn.Body.Raw, ",")
	for bi, key := range keys {
		bfn, found := functions.Find(key)
		if !found {
			return compiledFunction{}, errors.Newf(
				"dispatcher %s references unknown branch %q", fn.Key, key,
			)
		}
		last := bi == len(keys)-1
		if !last {
			ctx.Writer.WriteLocalGet(selSlot)
			ctx.Writer.WriteI32Const(int32(bi))
			ctx.Writer.WriteBinaryOp(wasm.OpI32Eq)
			ctx.Writer.WriteIf(blockT)
		}
		if err := emitBranchCall(ctx, fn.Key, bfn, slots, out.Type); err != nil {
			return compiledFunction{}, err
		}
		if !last {
			ctx.Writer.WriteElse()
		}
	}
	for range len(keys) - 1 {
		ctx.Writer.WriteEnd()
	}
	return compiledFunction{
		scopeName: fn.Key,
		typeIdx:   typeIdx,
		writer:    ctx.Writer,
	}, nil
}

// emitBranchCall calls the dispatcher branch with the dispatcher params it reads and
// casts its result to the dispatcher output type.
func emitBranchCall(
	ctx ccontext.Context[antlr.ParserRuleContext],
	dispatcher string,
	branch ir.Function,
	slots map[string]int,
	outType types.Type,
) error {
	for _, p := range branch.Inputs {
		slot, ok := slots[p.Name]
		if !ok {
			return errors.Newf(
				"dispatcher %s missing input %q for branch %q",
				dispatcher, p.Name, branch.Key,
			)
		}
		ctx.Writer.WriteLocalGet(slot)
	}
	ctx.Resolver.EmitLocalCall(ctx.Writer, ctx.WriterID, branch.Key, types.Type{})
	if bout, ok := branch.Outputs.Get(ir.DefaultOutputParam); ok {
		return expression.EmitCast(ctx, bout.Type, outType)
	}
	return nil
}

// blockTypeFor returns the value-producing block type for vt.
func blockTypeFor(vt wasm.ValueType) wasm.BlockType {
	switch vt {
	case wasm.I64:
		return wasm.BlockTypeI64
	case wasm.F32:
		return wasm.BlockTypeF32
	case wasm.F64:
		return wasm.BlockTypeF64
	case wasm.I32:
		return wasm.BlockTypeI32
	default:
		panic(errors.Newf("unknown wasm value type %#x", byte(vt)))
	}
}

func collectLocals(scope *symbol.Symbol) []wasm.ValueType {
	var locals []wasm.ValueType
	for _, child := range scope.Children() {
		switch child.Kind {
		case symbol.KindVariable, symbol.KindStatefulVariable,
			symbol.KindOutput, symbol.KindLoopVariable:
			locals = append(locals, wasm.ConvertType(child.Type))
		case symbol.KindBlock, symbol.KindLoop:
			locals = append(locals, collectLocals(child)...)
		case symbol.KindChannel, symbol.KindFunction, symbol.KindInput,
			symbol.KindSequence, symbol.KindStage, symbol.KindConstant,
			symbol.KindModule, symbol.KindModuleAlias, symbol.KindAmbient:
		}
	}
	return locals
}
