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
)

var _ = Describe("MatchOpcodes", func() {
	Describe("Basic matching", func() {
		It("should match identical opcode sequences", func() {
			bytecode := WASM(wasm.OpI32Const, int32(10), wasm.OpI32Add)
			Expect(bytecode).To(MatchOpcodes(wasm.OpI32Const, int32(10), wasm.OpI32Add))
		})

		It("should match when comparing byte slices", func() {
			expected := WASM(wasm.OpI64Const, int64(100), wasm.OpI64Sub)
			actual := WASM(wasm.OpI64Const, int64(100), wasm.OpI64Sub)
			Expect(actual).To(MatchOpcodes(expected))
		})

		It("should match when comparing OPCodes directly", func() {
			opcodes := wasm.OPCodesFromBytes(WASM(wasm.OpF32Add, wasm.OpF32Mul))
			Expect(opcodes).To(MatchOpcodes(wasm.OpF32Add, wasm.OpF32Mul))
		})

		It("should fail when opcodes differ", func() {
			bytecode := WASM(wasm.OpI32Const, int32(10), wasm.OpI32Add)
			Expect(bytecode).NotTo(MatchOpcodes(wasm.OpI32Const, int32(20), wasm.OpI32Add))
		})

		It("should fail when lengths differ", func() {
			bytecode := WASM(wasm.OpI32Add)
			Expect(bytecode).NotTo(MatchOpcodes(wasm.OpI32Add, wasm.OpI32Sub))
		})
	})

	Describe("Complex sequences", func() {
		It("should match complex arithmetic expressions", func() {
			bytecode := WASM(
				wasm.OpI64Const, int64(10),
				wasm.OpI32WrapI64,
				wasm.OpI32Const, int32(20),
				wasm.OpI32Add,
			)
			Expect(bytecode).To(MatchOpcodes(
				wasm.OpI64Const, int64(10),
				wasm.OpI32WrapI64,
				wasm.OpI32Const, int32(20),
				wasm.OpI32Add,
			))
		})

		It("should match local variable operations", func() {
			bytecode := WASM(
				wasm.OpLocalGet, 0,
				wasm.OpLocalGet, 1,
				wasm.OpI32Mul,
				wasm.OpLocalSet, 2,
			)
			Expect(bytecode).To(MatchOpcodes(
				wasm.OpLocalGet, 0,
				wasm.OpLocalGet, 1,
				wasm.OpI32Mul,
				wasm.OpLocalSet, 2,
			))
		})
	})

	Describe("Error handling", func() {
		It("should handle invalid input types gracefully", func() {
			_, err := MatchOpcodes(wasm.OpI32Add).Match("invalid")
			Expect(err).To(HaveOccurred())
		})

		It("should provide clear failure messages", func() {
			bytecode := WASM(wasm.OpI32Add, wasm.OpI32Sub)
			matcher := MatchOpcodes(wasm.OpI32Add, wasm.OpI32Mul)
			success, _ := matcher.Match(bytecode)
			Expect(success).To(BeFalse())

			message := matcher.FailureMessage(bytecode)
			Expect(message).To(ContainSubstring("Opcodes did not match"))
			Expect(message).To(ContainSubstring("OpI32Sub"))
			Expect(message).To(ContainSubstring("OpI32Mul"))
		})
	})
})
