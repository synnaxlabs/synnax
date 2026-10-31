// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/arc/compiler/testutil"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/symbol"
)

var _ = Describe("MatchOpcodes", func() {
	DescribeTable(
		"Should match the bytes that a wasm.Writer encodes",
		func(write func(w *wasm.Writer), expected ...any) {
			w := wasm.NewWriter()
			write(w)
			Expect(w.Bytes()).To(MatchOpcodes(expected...))
		},
		Entry("i32.const",
			func(w *wasm.Writer) { w.WriteI32Const(42) },
			wasm.OpI32Const, int32(42),
		),
		Entry("i64.const",
			func(w *wasm.Writer) { w.WriteI64Const(100) },
			wasm.OpI64Const, int64(100),
		),
		Entry("f32.const",
			func(w *wasm.Writer) { w.WriteF32Const(3.14) },
			wasm.OpF32Const, float32(3.14),
		),
		Entry("f64.const",
			func(w *wasm.Writer) { w.WriteF64Const(2.718) },
			wasm.OpF64Const, float64(2.718),
		),
		Entry(
			"local.get",
			func(w *wasm.Writer) { w.WriteLocalGet(0) },
			wasm.OpLocalGet,
			0,
		),
		Entry(
			"local.set",
			func(w *wasm.Writer) { w.WriteLocalSet(2) },
			wasm.OpLocalSet,
			2,
		),
		Entry("call with a uint32 operand",
			func(w *wasm.Writer) {
				w.WriteOpcode(wasm.OpCall)
				w.WriteLEB128Fixed5(5)
			},
			wasm.OpCall, uint32(5),
		),
		Entry("call with an int operand",
			func(w *wasm.Writer) {
				w.WriteOpcode(wasm.OpCall)
				w.WriteLEB128Fixed5(3)
			},
			wasm.OpCall, 3,
		),
		Entry("if with no block type",
			func(w *wasm.Writer) { w.WriteIf(wasm.BlockTypeEmpty) },
			wasm.OpIf,
		),
		Entry("if with an i32 block type",
			func(w *wasm.Writer) { w.WriteIf(wasm.BlockTypeI32) },
			wasm.OpIf, wasm.BlockTypeI32,
		),
		Entry("if with an f64 block type byte",
			func(w *wasm.Writer) { w.WriteIf(wasm.BlockTypeF64) },
			wasm.OpIf, byte(wasm.F64),
		),
		Entry("block",
			func(w *wasm.Writer) { w.WriteBlock(wasm.BlockTypeEmpty) },
			wasm.OpBlock, wasm.BlockTypeEmpty,
		),
		Entry("loop",
			func(w *wasm.Writer) { w.WriteLoop(wasm.BlockTypeEmpty) },
			wasm.OpLoop, wasm.BlockTypeEmpty,
		),
		Entry("br", func(w *wasm.Writer) { w.WriteBr(1) }, wasm.OpBr, uint32(1)),
		Entry("br_if", func(w *wasm.Writer) { w.WriteBrIf(0) }, wasm.OpBrIf, uint32(0)),
		Entry("a sequence of simple opcodes",
			func(w *wasm.Writer) {
				w.WriteBinaryOp(wasm.OpI32Add)
				w.WriteBinaryOp(wasm.OpI32Mul)
				w.WriteBinaryOp(wasm.OpI32Sub)
			},
			wasm.OpI32Add, wasm.OpI32Mul, wasm.OpI32Sub,
		),
		Entry("mixed constant and arithmetic instructions",
			func(w *wasm.Writer) {
				w.WriteI32Const(10)
				w.WriteI32Const(20)
				w.WriteBinaryOp(wasm.OpI32Add)
			},
			wasm.OpI32Const, int32(10),
			wasm.OpI32Const, int32(20),
			wasm.OpI32Add,
		),
	)
})

var _ = Describe("NewContext", func() {
	It(
		"Should create a root compilation context with initialized fields",
		func(ctx SpecContext) {
			cCtx := NewContext(ctx)
			Expect(cCtx.Scope).ToNot(BeNil())
			Expect(cCtx.Scope.Kind).To(Equal(symbol.KindBlock))
			Expect(cCtx.Scope.Parent.Kind).To(Equal(symbol.KindFunction))
			Expect(cCtx.Module).ToNot(BeNil())
			Expect(cCtx.Writer).ToNot(BeNil())
			Expect(cCtx.TypeMap).ToNot(BeNil())
		},
	)
})

var _ = Describe("Compile", func() {
	It("Should compile the module that a wasm.Module generates", func(ctx SpecContext) {
		mod := wasm.NewModule()
		mod.AddImport("env", "host", wasm.FunctionType{})
		Expect(Compile(ctx, mod).ImportedFunctions()).To(HaveLen(1))
	})

	It("Should fail the spec when the module is not valid WASM", func(ctx SpecContext) {
		mod := wasm.NewModule()
		typeIdx := mod.AddType(wasm.FunctionType{Results: []wasm.ValueType{wasm.I32}})
		mod.AddFunction(typeIdx, nil, nil)
		Expect(InterceptGomegaFailures(func() { Compile(ctx, mod) })).
			To(ConsistOf(ContainSubstring("not enough results")))
	})
})

var _ = Describe("ImportNames", func() {
	It("Should return the imported function names in order", func(ctx SpecContext) {
		mod := wasm.NewModule()
		mod.AddImport("env", "first", wasm.FunctionType{})
		mod.AddImport("env", "second", wasm.FunctionType{})
		Expect(ImportNames(ctx, mod)).To(Equal([]string{"first", "second"}))
	})
})
