// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package zyn_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/zyn"
)

var _ = Describe("Zyn", func() {
	Describe("DataType Enums", func() {
		Describe("Integer DataTypes", func() {
			for _, t := range zyn.IntegerTypes {
				It(fmt.Sprintf("Should parse %s successfully", t), func() {
					var dest zyn.DataType
					Expect(zyn.IntegerTypeSchema.Parse(t, &dest)).To(Succeed())
					Expect(dest).To(Equal(t))
				})
			}
			It("Should fail on a floating point type", func() {
				var dest zyn.DataType
				Expect(zyn.IntegerTypeSchema.Parse(zyn.Float32T, &dest)).
					To(MatchError(ContainSubstring("invalid enum value")))
			})
		})
		Describe("Floating Point DataTypes", func() {
			for _, t := range zyn.FloatingPointTypes {
				It(fmt.Sprintf("Should parse %s successfully", t), func() {
					var dest zyn.DataType
					Expect(zyn.FloatingPointTypeSchema.Parse(t, &dest)).To(Succeed())
					Expect(dest).To(Equal(t))
				})
				It("Should fail on an integer type", func() {
					var dest zyn.DataType
					Expect(zyn.FloatingPointTypeSchema.Parse(zyn.Int32T, &dest)).
						To(MatchError(ContainSubstring("invalid enum value")))
				})
			}
		})
		Describe("Numeric DataTypes", func() {
			for _, t := range zyn.NumericTypes {
				It(fmt.Sprintf("Should parse %s successfully", t), func() {
					var dest zyn.DataType
					Expect(zyn.NumericTypeSchema.Parse(t, &dest)).To(Succeed())
					Expect(dest).To(Equal(t))
				})
			}
			It("Should fail on a string type", func() {
				var dest zyn.DataType
				Expect(zyn.NumericTypeSchema.Parse(zyn.StringT, &dest)).
					To(MatchError(ContainSubstring("invalid enum value")))
			})
		})
		Describe("Primitive DataTypes", func() {
			for _, t := range zyn.PrimitiveTypes {
				It(fmt.Sprintf("Should parse %s successfully", t), func() {
					var dest zyn.DataType
					Expect(zyn.PrimitiveTypeSchema.Parse(t, &dest)).To(Succeed())
				})
			}
			It("Should fail on an object type", func() {
				var dest zyn.DataType
				Expect(zyn.PrimitiveTypeSchema.Parse(zyn.ObjectT, &dest)).
					To(MatchError(ContainSubstring("invalid enum value")))
			})
		})
		Describe("DataTypes", func() {
			for _, t := range zyn.DataTypes {
				It(fmt.Sprintf("Should parse %s successfully", t), func() {
					var dest zyn.DataType
					Expect(zyn.AnyDataTypeSchema.Parse(t, &dest)).To(Succeed())
				})
			}
			It("Should fail on a random string", func() {
				var dest zyn.DataType
				Expect(zyn.AnyDataTypeSchema.Parse("dog", &dest)).
					To(MatchError(ContainSubstring("invalid enum value")))
			})
		})
	})
	DescribeTable("DataType Literals", func(literal zyn.Schema, dataType zyn.DataType) {
		var dest zyn.DataType
		Expect(literal.Parse(dataType, &dest)).Should(Succeed())
		Expect(dest).To(Equal(dataType))
		Expect(literal.Parse("cat", &dest)).To(HaveOccurred())
	},
		Entry("string", zyn.StringTypeSchema, zyn.StringT),
		Entry("bool", zyn.BoolTypeSchema, zyn.BoolT),
		Entry("object", zyn.ObjectTypeSchema, zyn.ObjectT),
		Entry("number", zyn.NumberTypeSchema, zyn.NumberT),
		Entry("UUID", zyn.UUIDTypeSchema, zyn.UUIDT),
		Entry("float32", zyn.Float32TypeSchema, zyn.Float32T),
		Entry("float64", zyn.Float64TypeSchema, zyn.Float64T),
		Entry("uint8", zyn.Uint8TypeSchema, zyn.Uint8T),
		Entry("uint16", zyn.Uint16TypeSchema, zyn.Uint16T),
		Entry("uint32", zyn.Uint32TypeSchema, zyn.Uint32T),
		Entry("uint64", zyn.Uint64TypeSchema, zyn.Uint64T),
		Entry("int", zyn.IntTypeSchema, zyn.IntT),
		Entry("int8", zyn.Int8TypeSchema, zyn.Int8T),
		Entry("int16", zyn.Int16TypeSchema, zyn.Int16T),
		Entry("int32", zyn.Int32TypeSchema, zyn.Int32T),
		Entry("int64", zyn.Int64TypeSchema, zyn.Int64T),
		Entry("uint", zyn.UintTypeSchema, zyn.UintT),
		Entry("uint8", zyn.Uint8TypeSchema, zyn.Uint8T),
		Entry("uint16", zyn.Uint16TypeSchema, zyn.Uint16T),
		Entry("uint32", zyn.Uint32TypeSchema, zyn.Uint32T),
		Entry("uint64", zyn.Uint64TypeSchema, zyn.Uint64T),
	)
})
