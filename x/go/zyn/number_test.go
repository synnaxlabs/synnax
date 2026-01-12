// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package zyn_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/x/zyn"
)

func numberTest[V any](data V) func() {
	return func() {
		var dest V
		ExpectWithOffset(1, zyn.Number().Parse(data, &dest)).To(Succeed())
		Expect(dest).To(Equal(data))
	}
}

var _ = Describe("Number", func() {
	Describe("Basic Parsing", func() {
		Specify("float64", numberTest[float64](12))
		Specify("float32", numberTest[float32](12))
		Specify("int", numberTest(12))
		Specify("int8", numberTest[int8](12))
		Specify("int16", numberTest[int16](12))
		Specify("int32", numberTest[int32](12))
		Specify("int64", numberTest[int64](12))
		Specify("uint", numberTest[uint](12))
		Specify("uint8", numberTest[uint8](12))
		Specify("uint16", numberTest[uint16](12))
		Specify("uint32", numberTest[uint32](12))
		Specify("uint64", numberTest[uint64](12))
	})

	Describe("Validate", func() {
		It("Should return nil if the value is a valid number", func() {
			Expect(zyn.Number().Validate(42)).To(Succeed())
		})
		It("Should return nil if the value is not a valid number", func() {
			Expect(zyn.Number().Validate("not a number")).To(HaveOccurred())
		})
	})

	Describe("DataType Validation", func() {
		Describe("Float64", func() {
			Specify("valid float64", func() {
				var dest float64
				Expect(zyn.Number().Float64().Parse(12.5, &dest)).To(Succeed())
				Expect(dest).To(Equal(12.5))
			})

			Specify("invalid type", func() {
				var dest float64
				Expect(zyn.Number().Float64().Parse(12, &dest)).To(MatchError(ContainSubstring("expected float64")))
			})

			Specify("custom type", func() {
				type MyFloat float64
				var dest float64
				Expect(zyn.Number().Float64().Parse(MyFloat(12.5), &dest)).To(MatchError(ContainSubstring("expected float64")))
			})

			Specify("coerce int to float64", func() {
				var dest float64
				Expect(zyn.Number().Float64().Coerce().Parse(12, &dest)).To(Succeed())
				Expect(dest).To(Equal(12.0))
			})

			Specify("coerce uint to float64", func() {
				var dest float64
				Expect(zyn.Number().Float64().Coerce().Parse(uint(12), &dest)).To(Succeed())
				Expect(dest).To(Equal(12.0))
			})

			Specify("coerce custom type to float64", func() {
				type MyInt int
				var dest float64
				Expect(zyn.Number().Float64().Coerce().Parse(MyInt(12), &dest)).To(Succeed())
				Expect(dest).To(Equal(12.0))
			})
		})

		Describe("Int", func() {
			Specify("valid int", func() {
				var dest int
				Expect(zyn.Number().Int().Parse(12, &dest)).To(Succeed())
				Expect(dest).To(Equal(12))
			})

			Specify("invalid type", func() {
				var dest int
				Expect(zyn.Number().Int().Parse(12.5, &dest)).To(MatchError(ContainSubstring("expected int")))
			})

			Specify("custom type", func() {
				type MyInt int
				var dest int
				Expect(zyn.Number().Int().Parse(MyInt(12), &dest)).To(MatchError(ContainSubstring("expected int")))
			})

			Specify("coerce float64 to int", func() {
				var dest int
				Expect(zyn.Number().Int().Coerce().Parse(12.0, &dest)).To(Succeed())
				Expect(dest).To(Equal(12))
			})

			Specify("coerce uint to int", func() {
				var dest int
				Expect(zyn.Number().Int().Coerce().Parse(uint(12), &dest)).To(Succeed())
				Expect(dest).To(Equal(12))
			})

			Specify("coerce custom type to int", func() {
				type MyFloat float64
				var dest int
				Expect(zyn.Number().Int().Coerce().Parse(MyFloat(12.0), &dest)).To(Succeed())
				Expect(dest).To(Equal(12))
			})

			Specify("coerce float with decimal to int fails", func() {
				var dest int
				Expect(zyn.Number().Int().Coerce().Parse(12.5, &dest)).To(MatchError(ContainSubstring("cannot convert float")))
			})
		})

		Describe("Uint64", func() {
			Specify("valid uint64", func() {
				var dest uint64
				Expect(zyn.Number().Uint64().Parse(uint64(12), &dest)).To(Succeed())
				Expect(dest).To(Equal(uint64(12)))
			})

			Specify("invalid type", func() {
				var dest uint64
				Expect(zyn.Number().Uint64().Parse(12, &dest)).To(MatchError(ContainSubstring("expected uint64")))
			})

			Specify("custom type", func() {
				type MyUint uint64
				var dest uint64
				Expect(zyn.Number().Uint64().Parse(MyUint(12), &dest)).To(MatchError(ContainSubstring("expected uint64")))
			})
		})

		Describe("Uint16", func() {
			Specify("valid uint16", func() {
				var dest uint16
				Expect(zyn.Number().Uint16().Parse(uint16(12), &dest)).To(Succeed())
				Expect(dest).To(Equal(uint16(12)))
			})

			Specify("invalid type", func() {
				var dest uint16
				Expect(zyn.Number().Uint16().Parse(12, &dest)).To(MatchError(ContainSubstring("expected uint16")))
			})

			Specify("custom type", func() {
				type MyUint uint16
				var dest uint16
				Expect(zyn.Number().Uint16().Parse(MyUint(12), &dest)).To(MatchError(ContainSubstring("expected uint16")))
			})

			Specify("coerce uint32 to uint16", func() {
				var dest uint16
				Expect(zyn.Number().Uint16().Coerce().Parse(uint32(12), &dest)).To(Succeed())
				Expect(dest).To(Equal(uint16(12)))
			})

			Specify("coerce int to uint16", func() {
				var dest uint16
				Expect(zyn.Number().Uint16().Coerce().Parse(12, &dest)).To(Succeed())
				Expect(dest).To(Equal(uint16(12)))
			})

			Specify("coerce float to uint16", func() {
				var dest uint16
				Expect(zyn.Number().Uint16().Coerce().Parse(12.0, &dest)).To(Succeed())
				Expect(dest).To(Equal(uint16(12)))
			})

			Specify("coerce negative int to uint16 fails", func() {
				var dest uint16
				Expect(zyn.Number().Uint16().Coerce().Parse(-12, &dest)).To(MatchError(ContainSubstring("cannot convert negative value")))
			})

			Specify("coerce large uint32 to uint16 fails", func() {
				var dest uint16
				Expect(zyn.Number().Uint16().Coerce().Parse(uint32(1<<32-1), &dest)).To(MatchError(ContainSubstring("out of range")))
			})
		})
	})

	Describe("Conversion", func() {
		Specify("int32 to float64", func() {
			var (
				data int32 = 12
				dest float64
			)
			Expect(zyn.Number().Parse(data, &dest)).To(Succeed())
		})
	})

	Describe("Edge Cases", func() {
		Describe("Large Values", func() {
			Specify("max int64", func() {
				var dest int64
				Expect(zyn.Number().Parse(int64(1<<63-1), &dest)).To(Succeed())
				Expect(dest).To(Equal(int64(1<<63 - 1)))
			})

			Specify("max uint64", func() {
				var dest uint64
				Expect(zyn.Number().Parse(uint64(1<<64-1), &dest)).To(Succeed())
				Expect(dest).To(Equal(uint64(1<<64 - 1)))
			})

			Specify("max float64", func() {
				var dest float64
				Expect(zyn.Number().Parse(1.7976931348623157e+308, &dest)).To(Succeed())
				Expect(dest).To(Equal(1.7976931348623157e+308))
			})
		})

		Describe("Precision", func() {
			Specify("float64 to int64 with decimal", func() {
				var dest int64
				Expect(zyn.Number().Parse(12.5, &dest)).To(MatchError(ContainSubstring("cannot convert float")))
			})

			Specify("float64 to uint64 with decimal", func() {
				var dest uint64
				Expect(zyn.Number().Parse(12.5, &dest)).To(MatchError(ContainSubstring("cannot convert float")))
			})

			Specify("float64 to uint64 with negative", func() {
				var dest uint64
				Expect(zyn.Number().Parse(float64(-12), &dest)).To(MatchError(ContainSubstring("cannot convert negative value")))
			})
		})

		Describe("Overflow", func() {
			Specify("int64 to int8", func() {
				var dest int8
				Expect(zyn.Number().Parse(int64(1<<63-1), &dest)).To(MatchError(ContainSubstring("value out of range")))
			})

			Specify("uint64 to int64", func() {
				var dest int64
				Expect(zyn.Number().Parse(uint64(1<<63), &dest)).To(MatchError(ContainSubstring("unsigned integer value too large")))
			})

			Specify("uint64 to uint8", func() {
				var dest uint8
				Expect(zyn.Number().Parse(uint64(1<<64-1), &dest)).To(MatchError(ContainSubstring("out of range")))
			})
		})

		Describe("Custom DataTypes", func() {
			type MyInt int
			type MyFloat float64
			type MyUint uint64

			Specify("custom int type", func() {
				var dest int
				Expect(zyn.Number().Parse(MyInt(12), &dest)).To(Succeed())
				Expect(dest).To(Equal(12))
			})

			Specify("custom float type", func() {
				var dest float64
				Expect(zyn.Number().Parse(MyFloat(12.5), &dest)).To(Succeed())
				Expect(dest).To(Equal(12.5))
			})

			Specify("custom uint type", func() {
				var dest uint64
				Expect(zyn.Number().Parse(MyUint(1<<63), &dest)).To(Succeed())
				Expect(dest).To(Equal(uint64(1 << 63)))
			})
		})

		Describe("Invalid Destination", func() {
			Specify("non-numeric type", func() {
				var dest string
				Expect(zyn.Number().Parse(12, &dest)).To(HaveOccurredAs(zyn.InvalidDestinationTypeError))
			})

			Specify("nil pointer", func() {
				var dest *int
				Expect(zyn.Number().Parse(12, dest)).To(HaveOccurredAs(zyn.InvalidDestinationTypeError))
			})

			Specify("non-pointer destination", func() {
				var dest int
				Expect(zyn.Number().Parse(12, dest)).To(HaveOccurredAs(zyn.InvalidDestinationTypeError))
			})

			Specify("nil interface", func() {
				var dest any
				Expect(zyn.Number().Parse(12, dest)).To(HaveOccurredAs(zyn.InvalidDestinationTypeError))
			})

			Specify("channel destination", func() {
				var dest chan int
				Expect(zyn.Number().Parse(12, &dest)).To(HaveOccurredAs(zyn.InvalidDestinationTypeError))
			})

			Specify("slice destination", func() {
				var dest []int
				Expect(zyn.Number().Parse(12, &dest)).To(HaveOccurredAs(zyn.InvalidDestinationTypeError))
			})

			Specify("map destination", func() {
				var dest map[string]int
				Expect(zyn.Number().Parse(12, &dest)).To(HaveOccurredAs(zyn.InvalidDestinationTypeError))
			})

			Specify("struct destination", func() {
				var dest struct{ Value int }
				Expect(zyn.Number().Parse(12, &dest)).To(HaveOccurredAs(zyn.InvalidDestinationTypeError))
			})

			Specify("bool destination", func() {
				var dest bool
				Expect(zyn.Number().Parse(12, &dest)).To(HaveOccurredAs(zyn.InvalidDestinationTypeError))
			})
		})
	})

	Describe("Optional Fields", func() {
		Specify("optional field with nil value", func() {
			var dest *int
			Expect(zyn.Number().Optional().Parse(nil, &dest)).To(Succeed())
			Expect(dest).To(BeNil())
		})

		Specify("required field with nil value", func() {
			var dest int
			Expect(zyn.Number().Parse(nil, &dest)).To(HaveOccurredAs(validate.RequiredError))
		})

		Specify("optional field with value", func() {
			var dest *int
			Expect(zyn.Number().Optional().Parse(42, &dest)).To(Succeed())
			Expect(*dest).To(Equal(42))
		})

		Specify("optional field with custom type", func() {
			type MyInt int
			var dest *MyInt
			Expect(zyn.Number().Optional().Parse(42, &dest)).To(Succeed())
			Expect(*dest).To(Equal(MyInt(42)))
		})
	})

	Describe("Dump", func() {
		Describe("Basic DataTypes", func() {
			Specify("float64", func() {
				result, err := zyn.Number().Dump(12.5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(12.5))
			})

			Specify("float32", func() {
				result, err := zyn.Number().Dump(float32(12.5))
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(float64(12.5)))
			})

			Specify("int", func() {
				result, err := zyn.Number().Dump(12)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(int64(12)))
			})

			Specify("uint", func() {
				result, err := zyn.Number().Dump(uint(12))
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(uint64(12)))
			})
		})

		Describe("DataType Validation", func() {
			Specify("valid float64", func() {
				result, err := zyn.Number().Float64().Dump(12.5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(12.5))
			})

			Specify("valid float32", func() {
				result, err := zyn.Number().Float32().Dump(float32(12.5))
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(float32(12.5)))
			})

			Specify("invalid type", func() {
				_, err := zyn.Number().Float64().Dump(12)
				Expect(err).To(MatchError(ContainSubstring("expected float64 but received int")))
			})

			Specify("valid int", func() {
				result, err := zyn.Number().Int().Dump(12)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(12))
			})

			Specify("invalid type for int", func() {
				_, err := zyn.Number().Int().Dump(12.5)
				Expect(err).To(MatchError(ContainSubstring("expected int but received float64")))
			})
		})

		Describe("Edge Cases", func() {
			Specify("max int64", func() {
				result, err := zyn.Number().Dump(int64(1<<63 - 1))
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(int64(1<<63 - 1)))
			})

			Specify("max uint64", func() {
				result, err := zyn.Number().Dump(uint64(1<<64 - 1))
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(uint64(1<<64 - 1)))
			})

			Specify("max float64", func() {
				result, err := zyn.Number().Dump(1.7976931348623157e+308)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(1.7976931348623157e+308))
			})
		})

		Describe("Custom DataTypes", func() {
			type MyInt int
			type MyFloat float64
			type MyUint uint64

			Specify("custom int type", func() {
				result, err := zyn.Number().Dump(MyInt(12))
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(int64(12)))
			})

			Specify("custom float type", func() {
				result, err := zyn.Number().Dump(MyFloat(12.5))
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(12.5))
			})

			Specify("custom uint type", func() {
				result, err := zyn.Number().Dump(MyUint(1 << 63))
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(uint64(1 << 63)))
			})
		})

		Describe("Invalid Inputs", func() {
			Specify("non-numeric type", func() {
				_, err := zyn.Number().Dump("not a number")
				Expect(err).To(MatchError(ContainSubstring("expected number or convertible to number")))
			})

			Specify("nil value", func() {
				_, err := zyn.Number().Dump(nil)
				Expect(err).To(HaveOccurredAs(validate.RequiredError))
			})

			Specify("nil pointer", func() {
				var n *int
				_, err := zyn.Number().Dump(n)
				Expect(err).To(HaveOccurredAs(validate.RequiredError))
			})

			Specify("optional nil value", func() {
				result, err := zyn.Number().Optional().Dump(nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(BeNil())
			})

			Specify("optional nil pointer", func() {
				var n *int
				result, err := zyn.Number().Optional().Dump(n)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(BeNil())
			})
		})
	})

	Describe("Dump with Coerce", func() {
		Specify("coerce int to float64", func() {
			result, err := zyn.Number().Float64().Coerce().Dump(12)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(12.0))
		})

		Specify("coerce uint to int64", func() {
			result, err := zyn.Number().Int64().Coerce().Dump(uint(12))
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(int64(12)))
		})

		Specify("coerce float64 to uint16", func() {
			result, err := zyn.Number().Uint16().Coerce().Dump(12.0)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(uint16(12)))
		})

		Specify("coerce custom type to float64", func() {
			type MyInt int
			result, err := zyn.Number().Float64().Coerce().Dump(MyInt(12))
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(12.0))
		})

		Specify("coerce float with decimal to int fails", func() {
			_, err := zyn.Number().Int().Coerce().Dump(12.5)
			Expect(err).To(MatchError(ContainSubstring("cannot convert float")))
		})

		Specify("coerce negative int to uint fails", func() {
			_, err := zyn.Number().Uint().Coerce().Dump(-12)
			Expect(err).To(MatchError(ContainSubstring("cannot convert negative value")))
		})

		Specify("coerce large uint64 to uint16 fails", func() {
			_, err := zyn.Number().Uint16().Coerce().Dump(uint64(1<<64 - 1))
			Expect(err).To(MatchError(ContainSubstring("out of range for destination type uint16")))
		})
	})
})
