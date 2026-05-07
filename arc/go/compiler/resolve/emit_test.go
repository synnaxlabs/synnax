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

// stringSymbolResolver wraps the strings stdlib's "string" module so that
// qualified names like "string.from_i32" resolve through the same path the
// real compiler uses.
var stringSymbolResolver symbol.Resolver = stlstrings.SymbolResolver

var _ = Describe("EmitStringFrom* helpers", func() {
	DescribeTable("Should emit a call placeholder and register the matching import",
		func(
			emit func(r *resolve.Resolver, w *wasm.Writer, wID int),
			wantWASMName string,
		) {
			r := resolve.NewResolver(stringSymbolResolver)
			w := wasm.NewWriter()
			wID := r.TrackWriter(w)

			startLen := w.Len()
			emit(r, w, wID)

			Expect(w.Len() - startLen).To(Equal(6))
			Expect(w.Bytes()[startLen]).To(Equal(byte(wasm.OpCall)))

			m := wasm.NewModule()
			r.Finalize(m)
			Expect(m.ImportCount()).To(Equal(uint32(1)))
			Expect(m.ImportNames()).To(ConsistOf(wantWASMName))
		},
		Entry("from_i32",
			func(r *resolve.Resolver, w *wasm.Writer, wID int) { r.EmitStringFromI32(w, wID) },
			"from_i32"),
		Entry("from_u32",
			func(r *resolve.Resolver, w *wasm.Writer, wID int) { r.EmitStringFromU32(w, wID) },
			"from_u32"),
		Entry("from_i64",
			func(r *resolve.Resolver, w *wasm.Writer, wID int) { r.EmitStringFromI64(w, wID) },
			"from_i64"),
		Entry("from_u64",
			func(r *resolve.Resolver, w *wasm.Writer, wID int) { r.EmitStringFromU64(w, wID) },
			"from_u64"),
		Entry("from_f32",
			func(r *resolve.Resolver, w *wasm.Writer, wID int) { r.EmitStringFromF32(w, wID) },
			"from_f32"),
		Entry("from_f64",
			func(r *resolve.Resolver, w *wasm.Writer, wID int) { r.EmitStringFromF64(w, wID) },
			"from_f64"),
	)

	It("Should patch the recorded placeholder with the resolved import index", func() {
		r := resolve.NewResolver(stringSymbolResolver)
		w := wasm.NewWriter()
		wID := r.TrackWriter(w)
		r.EmitStringFromI32(w, wID)

		m := wasm.NewModule()
		r.FinalizeAndPatch(m)

		bytes := w.Bytes()
		Expect(bytes[0]).To(Equal(byte(wasm.OpCall)))
		Expect(bytes[1] & 0x7f).To(Equal(byte(0)))
	})
})

var _ = Describe("EmitNumericToString", func() {
	DescribeTable("Should dispatch to the host fn matching the source type",
		func(from types.Type, wantWASMName string) {
			r := resolve.NewResolver(stringSymbolResolver)
			w := wasm.NewWriter()
			wID := r.TrackWriter(w)

			Expect(r.EmitNumericToString(w, wID, from)).To(Succeed())

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

	It("Should return an error for non-numeric source types", func() {
		r := resolve.NewResolver(stringSymbolResolver)
		w := wasm.NewWriter()
		wID := r.TrackWriter(w)

		Expect(r.EmitNumericToString(w, wID, types.String())).
			To(MatchError(ContainSubstring("cannot convert")))
	})
})
