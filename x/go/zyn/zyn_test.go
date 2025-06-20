package zyn_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/zyn"
)

func dataTypeLiteralTest(literal zyn.Z, dt zyn.DataType) func() {
	return func() {

	}
}

var _ = Describe("Zyn", func() {
	Describe("DataType Enums", func() {
		Describe("Integer Types", func() {
			for _, t := range zyn.IntegerTypes {
				It(fmt.Sprintf("Should parse %s successfully", t), func() {
					var dest zyn.DataType
					Expect(zyn.IntegerTypeZ.Parse(t, &dest)).To(Succeed())
					Expect(dest).To(Equal(t))
				})

			}
			It("Should fail on a floating point type", func() {
				var dest zyn.DataType
				Expect(zyn.IntegerTypeZ.Parse(zyn.Float32T, &dest)).To(MatchError(ContainSubstring("invalid enum value")))
			})
		})

		Describe("Floating Point Types", func() {
			for _, t := range zyn.FloatingPointTypes {
				It(fmt.Sprintf("Should parse %s successfully", t), func() {
					var dest zyn.DataType
					Expect(zyn.FloatingPointTypeZ.Parse(t, &dest)).To(Succeed())
					Expect(dest).To(Equal(t))
				})
				It("Should fail on an integer type", func() {
					var dest zyn.DataType
					Expect(zyn.FloatingPointTypeZ.Parse(zyn.Int32T, &dest)).To(MatchError(ContainSubstring("invalid enum value")))
				})
			}
		})

		Describe("Numeric Types", func() {
			for _, t := range zyn.NumericTypes {
				It(fmt.Sprintf("Should parse %s successfully", t), func() {
					var dest zyn.DataType
					Expect(zyn.NumericTypeZ.Parse(t, &dest)).To(Succeed())
					Expect(dest).To(Equal(t))
				})
			}
			It("Should fail on a string type", func() {
				var dest zyn.DataType
				Expect(zyn.NumericTypeZ.Parse(zyn.StringT, &dest)).To(MatchError(ContainSubstring("invalid enum value")))
			})
		})

		Describe("Primitive Types", func() {
			for _, t := range zyn.PrimitiveTypes {
				It(fmt.Sprintf("Should parse %s successfully", t), func() {
					var dest zyn.DataType
					Expect(zyn.PrimitiveTypeZ.Parse(t, &dest)).To(Succeed())
				})
			}
			It("Should fail on an object type", func() {
				var dest zyn.DataType
				Expect(zyn.PrimitiveTypeZ.Parse(zyn.ObjectT, &dest)).To(MatchError(ContainSubstring("invalid enum value")))
			})
		})

		Describe("Types", func() {
			for _, t := range zyn.Types {
				It(fmt.Sprintf("Should parse %s successfully", t), func() {
					var dest zyn.DataType
					Expect(zyn.AnyTypeZ.Parse(t, &dest)).To(Succeed())
				})
			}
			It("Should fail on a random string", func() {
				var dest zyn.DataType
				Expect(zyn.AnyTypeZ.Parse("dog", &dest)).To(MatchError(ContainSubstring("invalid enum value")))
			})
		})
	})

	DescribeTable("DataType Literals", func(literal zyn.Z, dataType zyn.DataType) {
		var dest zyn.DataType
		Expect(literal.Parse(dataType, &dest)).Should(Succeed())
		Expect(dest).To(Equal(dataType))
		Expect(literal.Parse("cat", &dest)).To(HaveOccurred())
	},
		Entry("string", zyn.StringTypeZ, zyn.StringT),
		Entry("bool", zyn.BoolTypeZ, zyn.BoolT),
		Entry("object", zyn.ObjectTypeZ, zyn.ObjectT),
		Entry("number", zyn.NumberTypeZ, zyn.NumberT),
		Entry("UUID", zyn.UUIDTypeZ, zyn.UUIDT),
		Entry("float32", zyn.Float32TypeZ, zyn.Float32T),
		Entry("float64", zyn.Float64TypeZ, zyn.Float64T),
		Entry("uint8", zyn.Uint8TypeZ, zyn.Uint8T),
		Entry("uint16", zyn.Uint16TypeZ, zyn.Uint16T),
		Entry("uint32", zyn.Uint32TypeZ, zyn.Uint32T),
		Entry("uint64", zyn.Uint64TypeZ, zyn.Uint64T),
		Entry("int", zyn.IntTypeZ, zyn.IntT),
		Entry("int8", zyn.Int8TypeZ, zyn.Int8T),
		Entry("int16", zyn.Int16TypeZ, zyn.Int16T),
		Entry("int32", zyn.Int32TypeZ, zyn.Int32T),
		Entry("int64", zyn.Int64TypeZ, zyn.Int64T),
		Entry("uint", zyn.UintTypeZ, zyn.UintT),
		Entry("uint8", zyn.Uint8TypeZ, zyn.Uint8T),
		Entry("uint16", zyn.Uint16TypeZ, zyn.Uint16T),
		Entry("uint32", zyn.Uint32TypeZ, zyn.Uint32T),
		Entry("uint64", zyn.Uint64TypeZ, zyn.Uint64T),
	)
})
