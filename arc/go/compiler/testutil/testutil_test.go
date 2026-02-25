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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/arc/compiler/testutil"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/symbol"
)

var _ = Describe("WASM Helper", func() {
	Describe("Constant instructions", func() {
		It("Should encode i32.const", func() {
			bytes := WASM(wasm.OpI32Const, int32(42))
			Expect(bytes).ToNot(BeEmpty())
			Expect(bytes).To(MatchOpcodes(wasm.OpI32Const, int32(42)))
		})

		It("Should encode i64.const", func() {
			bytes := WASM(wasm.OpI64Const, int64(100))
			Expect(bytes).ToNot(BeEmpty())
			Expect(bytes).To(MatchOpcodes(wasm.OpI64Const, int64(100)))
		})

		It("Should encode f32.const", func() {
			bytes := WASM(wasm.OpF32Const, float32(3.14))
			Expect(bytes).ToNot(BeEmpty())
			Expect(bytes).To(MatchOpcodes(wasm.OpF32Const, float32(3.14)))
		})

		It("Should encode f64.const", func() {
			bytes := WASM(wasm.OpF64Const, float64(2.718))
			Expect(bytes).ToNot(BeEmpty())
			Expect(bytes).To(MatchOpcodes(wasm.OpF64Const, float64(2.718)))
		})
	})

	Describe("Local variable instructions", func() {
		It("Should encode local.get", func() {
			bytes := WASM(wasm.OpLocalGet, 0)
			Expect(bytes).To(MatchOpcodes(wasm.OpLocalGet, 0))
		})

		It("Should encode local.set", func() {
			bytes := WASM(wasm.OpLocalSet, 2)
			Expect(bytes).To(MatchOpcodes(wasm.OpLocalSet, 2))
		})
	})

	Describe("Call instruction", func() {
		It("Should encode call with uint32 operand", func() {
			bytes := WASM(wasm.OpCall, uint32(5))
			Expect(bytes).To(MatchOpcodes(wasm.OpCall, uint32(5)))
		})

		It("Should encode call with int operand", func() {
			bytes := WASM(wasm.OpCall, 3)
			Expect(bytes).To(MatchOpcodes(wasm.OpCall, uint32(3)))
		})
	})

	Describe("If instruction", func() {
		It("Should encode if with empty block type by default", func() {
			bytes := WASM(wasm.OpIf, wasm.BlockTypeEmpty)
			Expect(bytes).ToNot(BeEmpty())
		})

		It("Should encode if with i32 block type", func() {
			bytes := WASM(wasm.OpIf, wasm.BlockTypeI32)
			Expect(bytes).ToNot(BeEmpty())
		})
	})

	Describe("Simple opcodes", func() {
		It("Should encode opcodes without operands", func() {
			bytes := WASM(wasm.OpI32Add)
			Expect(bytes).To(MatchOpcodes(wasm.OpI32Add))
		})

		It("Should encode a sequence of simple opcodes", func() {
			bytes := WASM(wasm.OpI32Add, wasm.OpI32Mul, wasm.OpI32Sub)
			Expect(bytes).To(MatchOpcodes(wasm.OpI32Add, wasm.OpI32Mul, wasm.OpI32Sub))
		})
	})

	Describe("Complex sequences", func() {
		It("Should encode mixed constant and arithmetic instructions", func() {
			bytes := WASM(
				wasm.OpI32Const, int32(10),
				wasm.OpI32Const, int32(20),
				wasm.OpI32Add,
			)
			Expect(bytes).To(MatchOpcodes(
				wasm.OpI32Const, int32(10),
				wasm.OpI32Const, int32(20),
				wasm.OpI32Add,
			))
		})
	})
})

var _ = Describe("FunctionScope", func() {
	It("Should create a scope containing a function and block", func() {
		scope := FunctionScope(context.Background())
		Expect(scope).ToNot(BeNil())
		Expect(scope.Kind).To(Equal(symbol.KindBlock))
	})
})

var _ = Describe("NewContext", func() {
	It("Should create a root compilation context with initialized fields", func() {
		ctx := NewContext(context.Background())
		Expect(ctx.Scope).ToNot(BeNil())
		Expect(ctx.Module).ToNot(BeNil())
		Expect(ctx.Writer).ToNot(BeNil())
		Expect(ctx.TypeMap).ToNot(BeNil())
	})
})
