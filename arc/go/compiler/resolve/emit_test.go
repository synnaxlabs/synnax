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
	. "github.com/synnaxlabs/arc/compiler/testutil"
	"github.com/synnaxlabs/arc/compiler/wasm"
	stlstrings "github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

var _ = Describe("Emit", func() {
	var (
		scope      *symbol.Symbol
		resolver   *resolve.Resolver
		writer     *wasm.Writer
		writerID   int
		wasmModule *wasm.Module
	)
	BeforeEach(func() {
		scope = symbol.NewRoot(nil, stlstrings.NewSymbols())
		resolver = resolve.NewResolver()
		writer = wasm.NewWriter()
		writerID = resolver.TrackWriter(writer)
		wasmModule = wasm.NewModule()
	})
	Describe("EmitNumericToString", func() {
		DescribeTable("Should dispatch to the host fn matching the source type",
			func(ctx SpecContext, from types.Type, wantWASMName string) {
				Expect(
					resolver.EmitNumericToString(ctx, writer, writerID, scope, from),
				).To(Succeed())
				resolver.Finalize(wasmModule)
				Expect(ImportNames(ctx, wasmModule)).To(ConsistOf(wantWASMName))
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

		It(
			"Should return an error for non-numeric source types",
			func(ctx SpecContext) {
				Expect(
					resolver.EmitNumericToString(
						ctx,
						writer,
						writerID,
						scope,
						types.String(),
					),
				).
					To(MatchError(ContainSubstring("cannot convert")))
			},
		)
	})

	Describe("EmitNumericFormat", func() {
		DescribeTable(
			"Should dispatch to the format_<type> host fn matching the source type",
			func(ctx SpecContext, from types.Type, wantWASMName string) {
				Expect(
					resolver.EmitNumericFormat(ctx, writer, writerID, scope, from),
				).To(Succeed())
				resolver.Finalize(wasmModule)
				Expect(ImportNames(ctx, wasmModule)).To(ConsistOf(wantWASMName))
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

		It(
			"Should return an error for non-numeric source types",
			func(ctx SpecContext) {
				Expect(
					resolver.EmitNumericFormat(
						ctx,
						writer,
						writerID,
						scope,
						types.String(),
					),
				).
					To(MatchError(ContainSubstring("cannot convert")))
			},
		)
	})

	Describe("EmitStringFormat", func() {
		It("Should emit an import for string.format_str", func(ctx SpecContext) {
			Expect(
				resolver.EmitStringFormat(ctx, writer, writerID, scope),
			).To(Succeed())
			resolver.Finalize(wasmModule)
			Expect(ImportNames(ctx, wasmModule)).To(ConsistOf("format_str"))
		})

		It("Should return an error when scope is nil", func(ctx SpecContext) {
			Expect(resolver.EmitStringFormat(ctx, writer, writerID, nil)).
				To(MatchError(ContainSubstring("no scope")))
		})
	})

	Describe("EmitFixedImportCall", func() {
		It(
			"Should resolve the signature from the scope and emit an import",
			func(ctx SpecContext) {
				Expect(
					resolver.EmitFixedImportCall(
						ctx,
						writer,
						writerID,
						scope,
						"strings",
						"from_i32",
					),
				).To(Succeed())
				resolver.Finalize(wasmModule)
				Expect(ImportNames(ctx, wasmModule)).To(ConsistOf("from_i32"))
			},
		)

		It(
			"Should return an error when the symbol does not exist",
			func(ctx SpecContext) {
				Expect(
					resolver.EmitFixedImportCall(
						ctx,
						writer,
						writerID,
						scope,
						"strings",
						"does_not_exist",
					),
				).
					To(MatchError(ContainSubstring("resolve strings.does_not_exist")))
			},
		)

		It("Should return an error when no scope is configured", func(ctx SpecContext) {
			Expect(
				resolver.EmitFixedImportCall(
					ctx,
					writer,
					writerID,
					nil,
					"strings",
					"from_i32",
				),
			).
				To(MatchError(ContainSubstring("no scope")))
		})
	})

	Describe("EmitSeriesComparison", func() {
		DescribeTable("Should emit a compare_<op>_<type> series import",
			func(ctx SpecContext, op, wantWASMName string) {
				Expect(
					resolver.EmitSeriesComparison(writer, writerID, op, types.I64()),
				).To(Succeed())
				resolver.Finalize(wasmModule)
				Expect(ImportNames(ctx, wasmModule)).To(ConsistOf(wantWASMName))
			},
			Entry("> emits compare_gt_i64", ">", "compare_gt_i64"),
			Entry("< emits compare_lt_i64", "<", "compare_lt_i64"),
			Entry("== emits compare_eq_i64", "==", "compare_eq_i64"),
			Entry("!= emits compare_ne_i64", "!=", "compare_ne_i64"),
		)

		It("Should return an error for an unknown operator", func() {
			Expect(resolver.EmitSeriesComparison(writer, writerID, "??", types.I64())).
				To(MatchError(ContainSubstring("unknown comparison operator")))
		})
	})

	DescribeTable("Should emit the host import matching the element type",
		func(ctx SpecContext, emit func(), wantWASMName string) {
			emit()
			resolver.Finalize(wasmModule)
			Expect(ImportNames(ctx, wasmModule)).To(ConsistOf(wantWASMName))
		},
		Entry("EmitChannelRead", func() {
			resolver.EmitChannelRead(writer, writerID, types.Chan(types.I64()))
		}, "read_i64"),
		Entry("EmitChannelWrite", func() {
			resolver.EmitChannelWrite(writer, writerID, types.I64())
		}, "write_i64"),
		Entry("EmitStateLoad", func() {
			resolver.EmitStateLoad(writer, writerID, types.I64())
		}, "load_i64"),
		Entry("EmitStateStore", func() {
			resolver.EmitStateStore(writer, writerID, types.I64())
		}, "store_i64"),
		Entry("EmitStateLoadSeries", func() {
			resolver.EmitStateLoadSeries(writer, writerID, types.I64())
		}, "load_series_i64"),
		Entry("EmitStateStoreSeries", func() {
			resolver.EmitStateStoreSeries(writer, writerID, types.I64())
		}, "store_series_i64"),
		Entry("EmitSeriesArithmetic with a scalar", func() {
			Expect(
				resolver.EmitSeriesArithmetic(writer, writerID, "+", types.I64(), true),
			).To(Succeed())
		}, "element_add_i64"),
		Entry("EmitSeriesArithmetic with a series", func() {
			Expect(
				resolver.EmitSeriesArithmetic(
					writer,
					writerID,
					"+",
					types.I64(),
					false,
				),
			).To(Succeed())
		}, "series_add_i64"),
		Entry("EmitSeriesReverseArithmetic", func() {
			Expect(
				resolver.EmitSeriesReverseArithmetic(
					writer,
					writerID,
					"-",
					types.I64(),
				),
			).To(Succeed())
		}, "element_rsub_i64"),
		Entry("EmitSeriesCreateEmpty", func() {
			resolver.EmitSeriesCreateEmpty(writer, writerID, types.I64())
		}, "create_empty_i64"),
		Entry("EmitSeriesSetElement", func() {
			resolver.EmitSeriesSetElement(writer, writerID, types.I64())
		}, "set_element_i64"),
		Entry("EmitSeriesIndex", func() {
			resolver.EmitSeriesIndex(writer, writerID, types.I64())
		}, "index_i64"),
		Entry("EmitSeriesNegate", func() {
			resolver.EmitSeriesNegate(writer, writerID, types.I64())
		}, "negate_i64"),
		Entry("EmitMathPow", func() {
			resolver.EmitMathPow(writer, writerID, types.F64())
		}, "pow_f64"),
	)

	Describe("EmitSeriesLogical", func() {
		It("Should emit and for a series right operand", func(ctx SpecContext) {
			Expect(
				resolver.EmitSeriesLogical(writer, writerID, "and", false),
			).To(Succeed())
			resolver.Finalize(wasmModule)
			Expect(ImportNames(ctx, wasmModule)).To(ConsistOf("and"))
		})

		It("Should emit or_scalar for a scalar right operand", func(ctx SpecContext) {
			Expect(
				resolver.EmitSeriesLogical(writer, writerID, "or", true),
			).To(Succeed())
			resolver.Finalize(wasmModule)
			Expect(ImportNames(ctx, wasmModule)).To(ConsistOf("or_scalar"))
		})

		It("Should return an error for an unknown operator", func() {
			Expect(
				resolver.EmitSeriesLogical(writer, writerID, "xor", false),
			).To(MatchError(ContainSubstring("unknown logical operator")))
		})
	})

	Describe("EmitSeriesNot", func() {
		It("Should emit a not series import", func(ctx SpecContext) {
			resolver.EmitSeriesNot(writer, writerID)
			resolver.Finalize(wasmModule)
			Expect(ImportNames(ctx, wasmModule)).To(ConsistOf("not"))
		})
	})

	Describe("EmitIndexFrom", func() {
		DescribeTable("should call the module function for the index type",
			func(t types.Type, name string) {
				Expect(resolve.IndexFromName(t)).To(Equal(name))
				resolver.EmitIndexFrom(writer, writerID, t)
				Expect(resolver.References(name)).To(BeTrue())
				resolver.RegisterLocal(name, 0)
				Expect(resolver.Finalize(wasmModule)).To(HaveLen(1))
				Expect(wasmModule.ImportCount()).To(BeZero())
			},
			Entry("i64", types.I64(), "$index_from_i64"),
			Entry("u64", types.U64(), "$index_from_u64"),
		)
	})

	Describe("EmitSeriesIndexFrom", func() {
		DescribeTable("should import the host function for the index type",
			func(ctx SpecContext, t types.Type, name string) {
				resolver.EmitSeriesIndexFrom(writer, writerID, t)
				resolver.Finalize(wasmModule)
				Expect(ImportNames(ctx, wasmModule)).To(ConsistOf(name))
			},
			Entry("i64", types.I64(), "index_from_i64"),
			Entry("u64", types.U64(), "index_from_u64"),
		)
	})
})
