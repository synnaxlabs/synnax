// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package constraints_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer/constraints"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Numeric Type Promotion", func() {
	var system *constraints.System
	BeforeEach(func() { system = constraints.New() })

	// Helper to test promotion
	testPromotion := func(constraint, value, expected types.Type) {
		tv := types.NewTypeVariable("T", &constraint)
		system.AddCompatible(tv, value, nil, "promotion test")
		Expect(system.Unify()).To(Succeed())
		result := MustBeOk(system.GetSubstitution("T"))
		Expect(result).To(Equal(expected),
			"Expected %v ~ %v → %v, got %v", constraint, value, expected, result)
	}

	Describe("Rule 1: Float Promotion", func() {
		Context("when either type is float", func() {
			It("f32 ~ i32 → f32 (both 32-bit)", func() {
				testPromotion(types.F32(), types.I32(), types.F32())
			})

			It("i32 ~ f32 → f32 (both 32-bit, reversed)", func() {
				testPromotion(types.I32(), types.F32(), types.F32())
			})

			It("f32 ~ i64 → f64 (one is 64-bit)", func() {
				testPromotion(types.F32(), types.I64(), types.F64())
			})

			It("f64 ~ i32 → f64 (one is 64-bit)", func() {
				testPromotion(types.F64(), types.I32(), types.F64())
			})

			It("f64 ~ i64 → f64 (both 64-bit)", func() {
				testPromotion(types.F64(), types.I64(), types.F64())
			})

			It("f32 ~ u32 → f32 (unsigned doesn't change float rule)", func() {
				testPromotion(types.F32(), types.U32(), types.F32())
			})
		})
	})

	Describe("Rule 2: 64-bit Integer Promotion", func() {
		Context("when both are integers and either is 64-bit", func() {
			It("u64 ~ u32 → u64 (both unsigned, one 64-bit)", func() {
				testPromotion(types.U64(), types.U32(), types.U64())
			})

			It("u64 ~ u64 → u64 (both unsigned 64-bit)", func() {
				testPromotion(types.U64(), types.U64(), types.U64())
			})

			// Key insight: Mixing signedness at 64-bit → F64
			It("i64 ~ u64 → f64 (mixed signedness at 64-bit)", func() {
				testPromotion(types.I64(), types.U64(), types.F64())
			})

			It("u64 ~ i64 → f64 (mixed signedness at 64-bit, reversed)", func() {
				testPromotion(types.U64(), types.I64(), types.F64())
			})

			It("i64 ~ u32 → f64 (64-bit signed with unsigned)", func() {
				testPromotion(types.I64(), types.U32(), types.F64())
			})

			It("u64 ~ i32 → f64 (64-bit unsigned with signed)", func() {
				testPromotion(types.U64(), types.I32(), types.F64())
			})

			It("i64 ~ i32 → f64 (both signed, different widths, one 64-bit)", func() {
				testPromotion(types.I64(), types.I32(), types.F64())
			})

			It("i32 ~ i64 → f64 (both signed, different widths, one 64-bit, reversed)", func() {
				testPromotion(types.I32(), types.I64(), types.F64())
			})
		})
	})

	Describe("Rule 3: 32-bit and Smaller Integer Promotion", func() {
		Context("when both are integers ≤32-bit", func() {
			It("i32 ~ u32 → i32 (mixed signedness at 32-bit)", func() {
				testPromotion(types.I32(), types.U32(), types.I32())
			})

			It("u32 ~ i32 → i32 (mixed signedness at 32-bit, reversed)", func() {
				testPromotion(types.U32(), types.I32(), types.I32())
			})

			It("i32 ~ i16 → i32 (widens to 32-bit)", func() {
				testPromotion(types.I32(), types.I16(), types.I32())
			})

			It("u32 ~ u16 → u32 (widens to 32-bit)", func() {
				testPromotion(types.U32(), types.U16(), types.U32())
			})

			It("i16 ~ u16 → i32 (sub-32-bit widens to 32-bit, signed wins)", func() {
				testPromotion(types.I16(), types.U16(), types.I32())
			})

			It("i16 ~ i8 → i32 (both signed, both widen to 32-bit)", func() {
				testPromotion(types.I16(), types.I8(), types.I32())
			})

			It("u16 ~ u8 → u32 (both unsigned, both widen to 32-bit)", func() {
				testPromotion(types.U16(), types.U8(), types.U32())
			})

			It("i8 ~ u8 → i32 (smallest types widen to 32-bit)", func() {
				testPromotion(types.I8(), types.U8(), types.I32())
			})

			It("u32 ~ u32 → u32 (both unsigned 32-bit)", func() {
				testPromotion(types.U32(), types.U32(), types.U32())
			})

			It("i32 ~ i32 → i32 (both signed 32-bit)", func() {
				testPromotion(types.I32(), types.I32(), types.I32())
			})
		})
	})

	Describe("Edge Cases and Rationale", func() {
		It("demonstrates why i64 ~ u64 → f64 (no safe integer type)", func() {
			// i64 range: -2^63 to 2^63-1
			// u64 range: 0 to 2^64-1
			// i64 can't hold all u64 values (loses top bit)
			// u64 can't hold negative i64 values
			// F64 can represent both (with precision loss >2^53)
			testPromotion(types.I64(), types.U64(), types.F64())
		})

		It("shows i32 ~ u32 → i32 is safe enough (32-bit)", func() {
			// At 32-bit, we accept potential overflow rather than F32
			// because F32 loses precision at 2^24, much earlier than overflow
			testPromotion(types.I32(), types.U32(), types.I32())
		})
	})
})