package zyn_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/zyn"
)

var _ = Describe("Zyn", func() {
	Describe("Type Enums", func() {
		Describe("Integer Types", func() {
			for _, t := range zyn.IntegerTypes {
				It(fmt.Sprintf("Should parse %s successfully", t), func() {
					var dest zyn.Type
					Expect(zyn.IntegerTypeZ.Parse(t, &dest)).To(Succeed())
					Expect(dest).To(Equal(t))
				})

			}
			It("Should fail on a floating point type", func() {
				var dest zyn.Type
				Expect(zyn.IntegerTypeZ.Parse(zyn.Float32T, &dest)).To(MatchError(ContainSubstring("invalid enum value")))
			})
		})

		Describe("Floating Point Types", func() {
			for _, t := range zyn.FloatingPointTypes {
				It(fmt.Sprintf("Should parse %s successfully", t), func() {
					var dest zyn.Type
					Expect(zyn.FloatingPointTypeZ.Parse(t, &dest)).To(Succeed())
					Expect(dest).To(Equal(t))
				})
				It("Should fail on an integer type", func() {
					var dest zyn.Type
					Expect(zyn.FloatingPointTypeZ.Parse(zyn.Int32T, &dest)).To(MatchError(ContainSubstring("invalid enum value")))
				})
			}
		})

		Describe("Numeric Types", func() {
			for _, t := range zyn.NumericTypes {
				It(fmt.Sprintf("Should parse %s successfully", t), func() {
					var dest zyn.Type
					Expect(zyn.NumericTypeZ.Parse(t, &dest)).To(Succeed())
					Expect(dest).To(Equal(t))
				})
			}
			It("Should fail on a string type", func() {
				var dest zyn.Type
				Expect(zyn.NumericTypeZ.Parse(zyn.StringT, &dest)).To(MatchError(ContainSubstring("invalid enum value")))
			})
		})

		Describe("Primitive Types", func() {
			for _, t := range zyn.PrimitiveTypes {
				It(fmt.Sprintf("Should parse %s successfully", t), func() {
					var dest zyn.Type
					Expect(zyn.PrimitiveTypeZ.Parse(t, &dest)).To(Succeed())
				})
			}
			It("Should fail on an object type", func() {
				var dest zyn.Type
				Expect(zyn.PrimitiveTypeZ.Parse(zyn.ObjectT, &dest)).To(MatchError(ContainSubstring("invalid enum value")))
			})
		})

		Describe("Types", func() {
			for _, t := range zyn.Types {
				It(fmt.Sprintf("Should parse %s successfully", t), func() {
					var dest zyn.Type
					Expect(zyn.TypesZ.Parse(t, &dest)).To(Succeed())
				})
			}
			It("Should fail on a random string", func() {
				var dest zyn.Type
				Expect(zyn.TypesZ.Parse("dog", &dest)).To(MatchError(ContainSubstring("invalid enum value")))
			})
		})
	})
})
