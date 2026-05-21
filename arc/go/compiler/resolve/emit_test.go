// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resolve_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/compiler/resolve"
	"github.com/synnaxlabs/arc/compiler/wasm"
	stlstrings "github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

// stringScope wraps the strings stdlib's "strings" module under a fresh
// root so qualified lookups like "string.from_i32" resolve through the
// same scope-tree walk the real compiler uses.
func stringScope() *symbol.Symbol {
	root := symbol.NewRoot(nil)
	for _, s := range stlstrings.Symbols {
		root.AddChild(s)
	}
	return root
}

var _ = Describe("EmitNumericToString", func() {
	DescribeTable("Should dispatch to the host fn matching the source type",
		func(ctx SpecContext, from types.Type, wantWASMName string) {
			scope := stringScope()
			r := resolve.NewResolver()
			w := wasm.NewWriter()
			wID := r.TrackWriter(w)

			Expect(r.EmitNumericToString(ctx, w, wID, scope, from)).To(Succeed())

			m := wasm.NewModule()
			r.Finalize(m)
			Expect(m.ImportNames()).To(ConsistOf(wantWASMName))
		},
		Entry("i8 -> from_i32", types.I8(), "from_i32"),
		Entry("i16 -> from_i32", types.I16(), "from_i32"),
		Entry("i32 -> from_i32", types.I32(), "from_i32"),
		Entry("u8 -> from_u32", types.U8(), "from_u32"),
		Entry("u16 -> from_u32", types.U16(), "from_u32"),
		Entry("u32 -> from_u32", types.U32(), "from_u32"),
		Entry("i64 -> from_i64", types.I64(), "from_i64"),
		Entry("u64 -> from_u64", types.U64(), "from_u64"),
		Entry("f32 -> from_f32", types.F32(), "from_f32"),
		Entry("f64 -> from_f64", types.F64(), "from_f64"),
	)

	It("Should return an error for non-numeric source types", func(ctx SpecContext) {
		r := resolve.NewResolver()
		w := wasm.NewWriter()
		wID := r.TrackWriter(w)

		Expect(r.EmitNumericToString(ctx, w, wID, stringScope(), types.String())).
			To(MatchError(ContainSubstring("cannot convert")))
	})
})

var _ = Describe("EmitNumericFormat", func() {
	DescribeTable("Should dispatch to the format_<type> host fn matching the source type",
		func(ctx SpecContext, from types.Type, wantWASMName string) {
			scope := stringScope()
			r := resolve.NewResolver()
			w := wasm.NewWriter()
			wID := r.TrackWriter(w)

			Expect(r.EmitNumericFormat(ctx, w, wID, scope, from)).To(Succeed())

			m := wasm.NewModule()
			r.Finalize(m)
			Expect(m.ImportNames()).To(ConsistOf(wantWASMName))
		},
		Entry("i8 -> format_i32", types.I8(), "format_i32"),
		Entry("i16 -> format_i32", types.I16(), "format_i32"),
		Entry("i32 -> format_i32", types.I32(), "format_i32"),
		Entry("u8 -> format_u32", types.U8(), "format_u32"),
		Entry("u16 -> format_u32", types.U16(), "format_u32"),
		Entry("u32 -> format_u32", types.U32(), "format_u32"),
		Entry("i64 -> format_i64", types.I64(), "format_i64"),
		Entry("u64 -> format_u64", types.U64(), "format_u64"),
		Entry("f32 -> format_f32", types.F32(), "format_f32"),
		Entry("f64 -> format_f64", types.F64(), "format_f64"),
	)

	It("Should return an error for non-numeric source types", func(ctx SpecContext) {
		r := resolve.NewResolver()
		w := wasm.NewWriter()
		wID := r.TrackWriter(w)

		Expect(r.EmitNumericFormat(ctx, w, wID, stringScope(), types.String())).
			To(MatchError(ContainSubstring("cannot convert")))
	})
})

var _ = Describe("EmitStringFormat", func() {
	It("Should emit an import for string.format_str", func(ctx SpecContext) {
		scope := stringScope()
		r := resolve.NewResolver()
		w := wasm.NewWriter()
		wID := r.TrackWriter(w)

		Expect(r.EmitStringFormat(ctx, w, wID, scope)).To(Succeed())

		m := wasm.NewModule()
		r.Finalize(m)
		Expect(m.ImportNames()).To(ConsistOf("format_str"))
	})

	It("Should return an error when scope is nil", func(ctx SpecContext) {
		r := resolve.NewResolver()
		w := wasm.NewWriter()
		wID := r.TrackWriter(w)

		Expect(r.EmitStringFormat(ctx, w, wID, nil)).
			To(MatchError(ContainSubstring("no scope")))
	})
})

var _ = Describe("EmitFixedImportCall", func() {
	It("Should resolve the signature from the scope and emit an import", func(ctx SpecContext) {
		r := resolve.NewResolver()
		w := wasm.NewWriter()
		wID := r.TrackWriter(w)

		Expect(r.EmitFixedImportCall(ctx, w, wID, stringScope(), "strings", "from_i32")).To(Succeed())

		m := wasm.NewModule()
		r.Finalize(m)
		Expect(m.ImportNames()).To(ConsistOf("from_i32"))
	})

	It("Should return an error when the symbol does not exist", func(ctx SpecContext) {
		r := resolve.NewResolver()
		w := wasm.NewWriter()
		wID := r.TrackWriter(w)

		Expect(r.EmitFixedImportCall(ctx, w, wID, stringScope(), "strings", "does_not_exist")).
			To(MatchError(ContainSubstring("resolve strings.does_not_exist")))
	})

	It("Should return an error when no scope is configured", func(ctx SpecContext) {
		r := resolve.NewResolver()
		w := wasm.NewWriter()
		wID := r.TrackWriter(w)

		Expect(r.EmitFixedImportCall(ctx, w, wID, nil, "strings", "from_i32")).
			To(MatchError(ContainSubstring("no scope")))
	})
})
