// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bindings_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/compiler/bindings"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ImportIndex Helpers", func() {
	var idx *bindings.ImportIndex

	BeforeEach(func() {
		idx = bindings.NewImportIndex()
		// Populate with test data
		idx.ChannelRead["i64"] = 1
		idx.ChannelRead["f64"] = 2
		idx.ChannelWrite["i64"] = 3
		idx.ChannelWrite["f64"] = 4
		idx.SeriesCreateEmpty["i64"] = 6
		idx.SeriesIndex["i64"] = 7
		idx.StateLoad["i64"] = 8
		idx.StateStore["i64"] = 9

		// Series arithmetic
		idx.SeriesElementAdd["i64"] = 10
		idx.SeriesElementSub["i64"] = 11
		idx.SeriesElementMul["i64"] = 12
		idx.SeriesElementDiv["i64"] = 13
		idx.SeriesSeriesAdd["i64"] = 14
		idx.SeriesSeriesSub["i64"] = 15
		idx.SeriesSeriesMul["i64"] = 16
		idx.SeriesSeriesDiv["i64"] = 17

		// Series comparison
		idx.SeriesCompareGT["i64"] = 20
		idx.SeriesCompareLT["i64"] = 21
		idx.SeriesCompareGE["i64"] = 22
		idx.SeriesCompareLE["i64"] = 23
		idx.SeriesCompareEQ["i64"] = 24
		idx.SeriesCompareNE["i64"] = 25

		// Series set element
		idx.SeriesSetElement["i64"] = 30

		// Series reverse arithmetic
		idx.SeriesElementRSub["i64"] = 31
		idx.SeriesElementRDiv["i64"] = 32

		// Series scalar comparison
		idx.SeriesCompareGTScalar["i64"] = 40
		idx.SeriesCompareLTScalar["i64"] = 41
		idx.SeriesCompareGEScalar["i64"] = 42
		idx.SeriesCompareLEScalar["i64"] = 43
		idx.SeriesCompareEQScalar["i64"] = 44
		idx.SeriesCompareNEScalar["i64"] = 45

		// State load/store series
		idx.StateLoadSeries["i64"] = 50
		idx.StateStoreSeries["i64"] = 51

		// Series negate
		idx.SeriesNegate["i64"] = 60

		// Series modulo
		idx.SeriesElementMod["i64"] = 70
		idx.SeriesSeriesMod["i64"] = 71
	})

	Describe("GetChannelRead", func() {
		It("Should return import index for valid type", func() {
			funcIdx := MustSucceed(idx.GetChannelRead(types.I64()))
			Expect(funcIdx).To(Equal(uint32(1)))
		})

		It("Should return import index for float type", func() {
			funcIdx := MustSucceed(idx.GetChannelRead(types.F64()))
			Expect(funcIdx).To(Equal(uint32(2)))
		})

		It("Should return error for unsupported type", func() {
			_, err := idx.GetChannelRead(types.U8())
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(ContainSubstring("no channel read function")))
			Expect(err).To(MatchError(ContainSubstring("u8")))
		})
	})

	Describe("GetChannelWrite", func() {
		It("Should return import index for valid type", func() {
			funcIdx := MustSucceed(idx.GetChannelWrite(types.I64()))
			Expect(funcIdx).To(Equal(uint32(3)))
		})

		It("Should return error for unsupported type", func() {
			_, err := idx.GetChannelWrite(types.U8())
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(ContainSubstring("no channel write function")))
		})
	})

	Describe("GetSeriesCreateEmpty", func() {
		It("Should return import index for valid type", func() {
			funcIdx := MustSucceed(idx.GetSeriesCreateEmpty(types.I64()))
			Expect(funcIdx).To(Equal(uint32(6)))
		})

		It("Should return error for unsupported type", func() {
			_, err := idx.GetSeriesCreateEmpty(types.U8())
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(ContainSubstring("no series create function")))
		})
	})

	Describe("GetSeriesIndex", func() {
		It("Should return import index for valid type", func() {
			funcIdx := MustSucceed(idx.GetSeriesIndex(types.I64()))
			Expect(funcIdx).To(Equal(uint32(7)))
		})

		It("Should return error for unsupported type", func() {
			_, err := idx.GetSeriesIndex(types.U8())
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(ContainSubstring("no series index function")))
		})
	})

	Describe("GetStateLoad", func() {
		It("Should return import index for valid type", func() {
			funcIdx := MustSucceed(idx.GetStateLoad(types.I64()))
			Expect(funcIdx).To(Equal(uint32(8)))
		})

		It("Should return error for unsupported type", func() {
			_, err := idx.GetStateLoad(types.U8())
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(ContainSubstring("no state load function")))
		})
	})

	Describe("GetStateStore", func() {
		It("Should return import index for valid type", func() {
			funcIdx := MustSucceed(idx.GetStateStore(types.I64()))
			Expect(funcIdx).To(Equal(uint32(9)))
		})

		It("Should return error for unsupported type", func() {
			_, err := idx.GetStateStore(types.U8())
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(ContainSubstring("no state store function")))
		})
	})

	Describe("GetSeriesArithmetic", func() {
		Context("Scalar operations (isScalar = true)", func() {
			It("Should return correct index for addition", func() {
				funcIdx := MustSucceed(idx.GetSeriesArithmetic("+", types.I64(), true))
				Expect(funcIdx).To(Equal(uint32(10)))
			})

			It("Should return correct index for subtraction", func() {
				funcIdx := MustSucceed(idx.GetSeriesArithmetic("-", types.I64(), true))
				Expect(funcIdx).To(Equal(uint32(11)))
			})

			It("Should return correct index for multiplication", func() {
				funcIdx := MustSucceed(idx.GetSeriesArithmetic("*", types.I64(), true))
				Expect(funcIdx).To(Equal(uint32(12)))
			})

			It("Should return correct index for division", func() {
				funcIdx := MustSucceed(idx.GetSeriesArithmetic("/", types.I64(), true))
				Expect(funcIdx).To(Equal(uint32(13)))
			})

			It("Should return error for unsupported type", func() {
				_, err := idx.GetSeriesArithmetic("+", types.U8(), true)
				Expect(err).NotTo(BeNil())
				Expect(err).To(MatchError(ContainSubstring("no series + function")))
			})
		})

		Context("Series-to-series operations (isScalar = false)", func() {
			It("Should return correct index for addition", func() {
				funcIdx := MustSucceed(idx.GetSeriesArithmetic("+", types.I64(), false))
				Expect(funcIdx).To(Equal(uint32(14)))
			})

			It("Should return correct index for subtraction", func() {
				funcIdx := MustSucceed(idx.GetSeriesArithmetic("-", types.I64(), false))
				Expect(funcIdx).To(Equal(uint32(15)))
			})

			It("Should return correct index for multiplication", func() {
				funcIdx := MustSucceed(idx.GetSeriesArithmetic("*", types.I64(), false))
				Expect(funcIdx).To(Equal(uint32(16)))
			})

			It("Should return correct index for division", func() {
				funcIdx := MustSucceed(idx.GetSeriesArithmetic("/", types.I64(), false))
				Expect(funcIdx).To(Equal(uint32(17)))
			})

		})
	})

	Describe("GetSeriesComparison", func() {
		It("Should return correct index for greater-than", func() {
			funcIdx := MustSucceed(idx.GetSeriesComparison(">", types.I64()))
			Expect(funcIdx).To(Equal(uint32(20)))
		})

		It("Should return correct index for less-than", func() {
			funcIdx := MustSucceed(idx.GetSeriesComparison("<", types.I64()))
			Expect(funcIdx).To(Equal(uint32(21)))
		})

		It("Should return correct index for greater-than-or-equal", func() {
			funcIdx := MustSucceed(idx.GetSeriesComparison(">=", types.I64()))
			Expect(funcIdx).To(Equal(uint32(22)))
		})

		It("Should return correct index for less-than-or-equal", func() {
			funcIdx := MustSucceed(idx.GetSeriesComparison("<=", types.I64()))
			Expect(funcIdx).To(Equal(uint32(23)))
		})

		It("Should return correct index for equals", func() {
			funcIdx := MustSucceed(idx.GetSeriesComparison("==", types.I64()))
			Expect(funcIdx).To(Equal(uint32(24)))
		})

		It("Should return correct index for not-equals", func() {
			funcIdx := MustSucceed(idx.GetSeriesComparison("!=", types.I64()))
			Expect(funcIdx).To(Equal(uint32(25)))
		})

		It("Should return error for unknown operator", func() {
			_, err := idx.GetSeriesComparison("===", types.I64())
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(ContainSubstring("unknown comparison operator")))
		})

		It("Should return error for unsupported type", func() {
			_, err := idx.GetSeriesComparison(">", types.U8())
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(ContainSubstring("no series comparison > function")))
		})
	})

	Describe("GetSeriesSetElement", func() {
		It("Should return import index for valid type", func() {
			funcIdx := MustSucceed(idx.GetSeriesSetElement(types.I64()))
			Expect(funcIdx).To(Equal(uint32(30)))
		})

		It("Should return error for unsupported type", func() {
			_, err := idx.GetSeriesSetElement(types.U8())
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(ContainSubstring("no series set element function")))
		})
	})

	Describe("GetSeriesReverseArithmetic", func() {
		It("Should return correct index for reverse subtraction", func() {
			funcIdx := MustSucceed(idx.GetSeriesReverseArithmetic("-", types.I64()))
			Expect(funcIdx).To(Equal(uint32(31)))
		})

		It("Should return correct index for reverse division", func() {
			funcIdx := MustSucceed(idx.GetSeriesReverseArithmetic("/", types.I64()))
			Expect(funcIdx).To(Equal(uint32(32)))
		})

		It("Should return error for unsupported operator", func() {
			_, err := idx.GetSeriesReverseArithmetic("+", types.I64())
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(ContainSubstring("reverse arithmetic only supported for - and /")))
		})

		It("Should return error for unsupported type", func() {
			_, err := idx.GetSeriesReverseArithmetic("-", types.U8())
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(ContainSubstring("no series reverse - function")))
		})
	})

	Describe("GetSeriesScalarComparison", func() {
		It("Should return correct index for greater-than", func() {
			funcIdx := MustSucceed(idx.GetSeriesScalarComparison(">", types.I64()))
			Expect(funcIdx).To(Equal(uint32(40)))
		})

		It("Should return correct index for less-than", func() {
			funcIdx := MustSucceed(idx.GetSeriesScalarComparison("<", types.I64()))
			Expect(funcIdx).To(Equal(uint32(41)))
		})

		It("Should return correct index for greater-than-or-equal", func() {
			funcIdx := MustSucceed(idx.GetSeriesScalarComparison(">=", types.I64()))
			Expect(funcIdx).To(Equal(uint32(42)))
		})

		It("Should return correct index for less-than-or-equal", func() {
			funcIdx := MustSucceed(idx.GetSeriesScalarComparison("<=", types.I64()))
			Expect(funcIdx).To(Equal(uint32(43)))
		})

		It("Should return correct index for equals", func() {
			funcIdx := MustSucceed(idx.GetSeriesScalarComparison("==", types.I64()))
			Expect(funcIdx).To(Equal(uint32(44)))
		})

		It("Should return correct index for not-equals", func() {
			funcIdx := MustSucceed(idx.GetSeriesScalarComparison("!=", types.I64()))
			Expect(funcIdx).To(Equal(uint32(45)))
		})

		It("Should return error for unknown operator", func() {
			_, err := idx.GetSeriesScalarComparison("===", types.I64())
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(ContainSubstring("unknown comparison operator")))
		})

		It("Should return error for unsupported type", func() {
			_, err := idx.GetSeriesScalarComparison(">", types.U8())
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(ContainSubstring("no series scalar comparison > function")))
		})
	})

	Describe("GetStateLoadSeries", func() {
		It("Should return import index for valid element type", func() {
			funcIdx := MustSucceed(idx.GetStateLoadSeries(types.I64()))
			Expect(funcIdx).To(Equal(uint32(50)))
		})

		It("Should return error for unsupported type", func() {
			_, err := idx.GetStateLoadSeries(types.U8())
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(ContainSubstring("no series state load function")))
		})
	})

	Describe("GetStateStoreSeries", func() {
		It("Should return import index for valid element type", func() {
			funcIdx := MustSucceed(idx.GetStateStoreSeries(types.I64()))
			Expect(funcIdx).To(Equal(uint32(51)))
		})

		It("Should return error for unsupported type", func() {
			_, err := idx.GetStateStoreSeries(types.U8())
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(ContainSubstring("no series state store function")))
		})
	})

	Describe("GetSeriesNegate", func() {
		It("Should return import index for valid type", func() {
			funcIdx := MustSucceed(idx.GetSeriesNegate(types.I64()))
			Expect(funcIdx).To(Equal(uint32(60)))
		})

		It("Should return error for unsupported type", func() {
			_, err := idx.GetSeriesNegate(types.U8())
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(ContainSubstring("no series negate function")))
		})
	})

	Describe("GetSeriesArithmetic modulo", func() {
		It("Should return correct index for modulo with scalar", func() {
			funcIdx := MustSucceed(idx.GetSeriesArithmetic("%", types.I64(), true))
			Expect(funcIdx).To(Equal(uint32(70)))
		})

		It("Should return correct index for modulo with series", func() {
			funcIdx := MustSucceed(idx.GetSeriesArithmetic("%", types.I64(), false))
			Expect(funcIdx).To(Equal(uint32(71)))
		})

		It("Should return error for unknown operator", func() {
			_, err := idx.GetSeriesArithmetic("^", types.I64(), true)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(ContainSubstring("unknown arithmetic operator")))
		})
	})
})
