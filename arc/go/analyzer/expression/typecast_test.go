// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package expression_test

import (
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Type Casts", func() {
	DescribeTable("Integer to Float Casts",
		func(code string) { expectSuccess(code, nil) },
		Entry("i32 to f32", `
			func testFunc() {
				x i32 := 10
				y := f32(x)
			}
		`),
		Entry("i64 to f64", `
			func testFunc() {
				x i64 := 100
				y := f64(x)
			}
		`),
		Entry("integer literal to float", `
			func testFunc() {
				y := f32(42)
			}
		`),
	)

	DescribeTable("Float to Integer Casts",
		func(code string) { expectSuccess(code, nil) },
		Entry("f32 to i32", `
			func testFunc() {
				x f32 := 3.14
				y := i32(x)
			}
		`),
		Entry("f64 to i64", `
			func testFunc() {
				x f64 := 3.14159
				y := i64(x)
			}
		`),
		Entry("float literal to integer", `
			func testFunc() {
				y := i32(3.14)
			}
		`),
	)

	DescribeTable("Integer Width Conversions",
		func(code string) { expectSuccess(code, nil) },
		Entry("i32 to i64 (widening)", `
			func testFunc() {
				x i32 := 10
				y := i64(x)
			}
		`),
		Entry("i64 to i32 (narrowing)", `
			func testFunc() {
				x i64 := 10
				y := i32(x)
			}
		`),
		Entry("u32 to u64 (widening)", `
			func testFunc() {
				x u32 := 10
				y := u64(x)
			}
		`),
		Entry("i8 to i32", `
			func testFunc() {
				x i8 := 10
				y := i32(x)
			}
		`),
	)

	DescribeTable("Signed/Unsigned Conversions",
		func(code string) { expectSuccess(code, nil) },
		Entry("i32 to u32", `
			func testFunc() {
				x i32 := 10
				y := u32(x)
			}
		`),
		Entry("u32 to i32", `
			func testFunc() {
				x u32 := 10
				y := i32(x)
			}
		`),
	)

	DescribeTable("Float Width Conversions",
		func(code string) { expectSuccess(code, nil) },
		Entry("f32 to f64 (widening)", `
			func testFunc() {
				x f32 := 3.14
				y := f64(x)
			}
		`),
		Entry("f64 to f32 (narrowing)", `
			func testFunc() {
				x f64 := 3.14159
				y := f32(x)
			}
		`),
	)

	DescribeTable("Type Casts in Expressions",
		func(code string) { expectSuccess(code, nil) },
		Entry("cast in arithmetic expression", `
			func testFunc() {
				x i32 := 10
				y f32 := 3.14
				result := f32(x) + y
			}
		`),
		Entry("nested type casts", `
			func testFunc() {
				x f64 := 3.14159
				y := i32(f32(x))
			}
		`),
		Entry("cast of complex expression", `
			func testFunc() {
				x i32 := 10
				y i32 := 20
				result := f32(x + y)
			}
		`),
	)

	DescribeTable("Boolean and Edge Cases",
		func(code string) { expectSuccess(code, nil) },
		Entry("u8 in boolean context", `
			func testFunc() {
				x u8 := 1
				y u8 := 0
				result := x and y
			}
		`),
		Entry("cast of literal in function call", `
			func takeFloat(x f32) f32 {
				return x
			}

			func testFunc() {
				result := takeFloat(f32(10))
			}
		`),
		Entry("cast in comparison", `
			func testFunc() {
				x i32 := 10
				result := f32(x) > 5.0
			}
		`),
	)

	DescribeTable("Boolean/Numeric Conversions",
		func(code string) { expectSuccess(code, nil) },
		Entry("u8 (bool) to i32", `
			func testFunc() {
				x u8 := 1
				y := i32(x)
			}
		`),
		Entry("u8 (bool) to f32", `
			func testFunc() {
				x u8 := 1
				y := f32(x)
			}
		`),
		Entry("i32 to u8 (bool)", `
			func testFunc() {
				x i32 := 1
				y := u8(x)
			}
		`),
		Entry("f32 to u8 (bool)", `
			func testFunc() {
				x f32 := 1.0
				y := u8(x)
			}
		`),
	)

	DescribeTable("Same Type Casts",
		func(code string) { expectSuccess(code, nil) },
		Entry("i32 to i32", `
			func testFunc() {
				x i32 := 42
				y := i32(x)
			}
		`),
		Entry("f64 to f64", `
			func testFunc() {
				x f64 := 3.14
				y := f64(x)
			}
		`),
		Entry("str to str", `
			func testFunc() {
				x str := "hello"
				y := str(x)
			}
		`),
		Entry("string literal to str", `
			func testFunc() {
				y := str("hello")
			}
		`),
		Entry("integer literal to i32", `
			func testFunc() {
				y := i32(42)
			}
		`),
		Entry("float literal to f64", `
			func testFunc() {
				y := f64(3.14)
			}
		`),
	)

	Describe("Type Cast Failures", func() {
		It("Should reject casting string to numeric type", func() {
			expectFailure(`
				func testFunc() {
					x str := "hello"
					y := i32(x)
				}
			`, nil, "cannot cast")
		})

		It("Should reject casting numeric to string", func() {
			expectFailure(`
				func testFunc() {
					x i32 := 42
					y := str(x)
				}
			`, nil, "cannot cast")
		})

		It("Should reject unknown type in cast", func() {
			expectFailure(`
				func testFunc() {
					x i32 := 42
					y := unknownType(x)
				}
			`, nil, "undefined")
		})
	})
})
