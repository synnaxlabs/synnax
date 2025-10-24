// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/types"
)

func TestTypes(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Types Suite")
}

var _ = Describe("Type.Unwrap", func() {
	Describe("Channel types", func() {
		It("should unwrap channel of i32 to i32", func() {
			chanType := types.Chan(types.I32())
			Expect(chanType.Unwrap()).To(Equal(types.I32()))
		})

		It("should unwrap channel of f64 to f64", func() {
			chanType := types.Chan(types.F64())
			Expect(chanType.Unwrap()).To(Equal(types.F64()))
		})

		It("should unwrap channel of u8 to u8", func() {
			chanType := types.Chan(types.U8())
			Expect(chanType.Unwrap()).To(Equal(types.U8()))
		})

		It("should handle nested channels (chan of chan)", func() {
			innerChan := types.Chan(types.I32())
			outerChan := types.Chan(innerChan)
			// First unwrap returns inner channel
			Expect(outerChan.Unwrap()).To(Equal(innerChan))
			// Second unwrap returns i32
			Expect(outerChan.Unwrap().Unwrap()).To(Equal(types.I32()))
		})
	})

	Describe("Series types", func() {
		It("should unwrap series of i32 to i32", func() {
			seriesType := types.Series(types.I32())
			Expect(seriesType.Unwrap()).To(Equal(types.I32()))
		})

		It("should unwrap series of f32 to f32", func() {
			seriesType := types.Series(types.F32())
			Expect(seriesType.Unwrap()).To(Equal(types.F32()))
		})

		It("should unwrap series of timestamp to timestamp", func() {
			seriesType := types.Series(types.TimeStamp())
			Expect(seriesType.Unwrap()).To(Equal(types.TimeStamp()))
		})

		It("should handle nested series (series of series)", func() {
			innerSeries := types.Series(types.F64())
			outerSeries := types.Series(innerSeries)
			// First unwrap returns inner series
			Expect(outerSeries.Unwrap()).To(Equal(innerSeries))
			// Second unwrap returns f64
			Expect(outerSeries.Unwrap().Unwrap()).To(Equal(types.F64()))
		})
	})

	Describe("Primitive types", func() {
		It("should return i32 unchanged", func() {
			t := types.I32()
			Expect(t.Unwrap()).To(Equal(t))
		})

		It("should return f64 unchanged", func() {
			t := types.F64()
			Expect(t.Unwrap()).To(Equal(t))
		})

		It("should return u8 unchanged", func() {
			t := types.U8()
			Expect(t.Unwrap()).To(Equal(t))
		})

		It("should return timestamp unchanged", func() {
			t := types.TimeStamp()
			Expect(t.Unwrap()).To(Equal(t))
		})

		It("should return timespan unchanged", func() {
			t := types.TimeSpan()
			Expect(t.Unwrap()).To(Equal(t))
		})
	})

	Describe("Type variables", func() {
		It("should return type variable unchanged", func() {
			tv := types.NewTypeVariable("T", nil)
			Expect(tv.Unwrap()).To(Equal(tv))
		})

		It("should return constrained type variable unchanged", func() {
			constraint := types.NumericConstraint()
			tv := types.NewTypeVariable("N", &constraint)
			Expect(tv.Unwrap()).To(Equal(tv))
		})
	})

	Describe("Function types", func() {
		It("should return function type unchanged", func() {
			props := types.NewFunctionProperties()
			props.Inputs.Put("x", types.I32())
			props.Outputs.Put("result", types.I32())
			fnType := types.Function(props)
			Expect(fnType.Unwrap()).To(Equal(fnType))
		})
	})

	Describe("Mixed channel and series", func() {
		It("should unwrap channel of series correctly", func() {
			seriesType := types.Series(types.F32())
			chanType := types.Chan(seriesType)
			// Unwrap channel returns series
			Expect(chanType.Unwrap()).To(Equal(seriesType))
			// Further unwrap returns f32
			Expect(chanType.Unwrap().Unwrap()).To(Equal(types.F32()))
		})

		It("should unwrap series of channel correctly", func() {
			chanType := types.Chan(types.I64())
			seriesType := types.Series(chanType)
			// Unwrap series returns channel
			Expect(seriesType.Unwrap()).To(Equal(chanType))
			// Further unwrap returns i64
			Expect(seriesType.Unwrap().Unwrap()).To(Equal(types.I64()))
		})
	})

	Describe("Edge cases", func() {
		It("should handle invalid/zero type", func() {
			var t types.Type
			// Zero type should return itself (no panic)
			Expect(func() { t.Unwrap() }).NotTo(Panic())
			Expect(t.Unwrap()).To(Equal(t))
		})

		It("should handle channel with nil ValueType", func() {
			chanType := types.Type{Kind: types.KindChan, ValueType: nil}
			// Should return itself, not panic
			Expect(chanType.Unwrap()).To(Equal(chanType))
		})

		It("should handle series with nil ValueType", func() {
			seriesType := types.Type{Kind: types.KindSeries, ValueType: nil}
			// Should return itself, not panic
			Expect(seriesType.Unwrap()).To(Equal(seriesType))
		})
	})

	Describe("Idempotence", func() {
		It("should be idempotent for primitives", func() {
			t := types.I32()
			Expect(t.Unwrap().Unwrap()).To(Equal(t.Unwrap()))
		})

		It("should be idempotent after unwrapping once", func() {
			chanType := types.Chan(types.F64())
			unwrapped := chanType.Unwrap()
			// Unwrapping again should return same result
			Expect(unwrapped.Unwrap()).To(Equal(unwrapped))
		})

		It("should fully unwrap nested types with repeated calls", func() {
			nested := types.Chan(types.Series(types.I32()))
			// First unwrap: chan -> series
			firstUnwrap := nested.Unwrap()
			Expect(firstUnwrap.Kind).To(Equal(types.KindSeries))
			// Second unwrap: series -> i32
			secondUnwrap := firstUnwrap.Unwrap()
			Expect(secondUnwrap).To(Equal(types.I32()))
			// Third unwrap: i32 -> i32 (idempotent)
			thirdUnwrap := secondUnwrap.Unwrap()
			Expect(thirdUnwrap).To(Equal(types.I32()))
		})
	})
})
