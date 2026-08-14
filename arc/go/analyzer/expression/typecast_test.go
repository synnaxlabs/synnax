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
		func(ctx SpecContext, code string) { expectSuccess(ctx, code, nil) },
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
		func(ctx SpecContext, code string) { expectSuccess(ctx, code, nil) },
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
		func(ctx SpecContext, code string) { expectSuccess(ctx, code, nil) },
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
		func(ctx SpecContext, code string) { expectSuccess(ctx, code, nil) },
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
		func(ctx SpecContext, code string) { expectSuccess(ctx, code, nil) },
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
		func(ctx SpecContext, code string) { expectSuccess(ctx, code, nil) },
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
		func(ctx SpecContext, code string) { expectSuccess(ctx, code, nil) },
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
		func(ctx SpecContext, code string) { expectSuccess(ctx, code, nil) },
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
		func(ctx SpecContext, code string) { expectSuccess(ctx, code, nil) },
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

	DescribeTable("Numeric to String Casts",
		func(ctx SpecContext, code string) { expectSuccess(ctx, code, nil) },
		Entry("i8 to str", `
			func testFunc() {
				x i8 := -128
				y := str(x)
			}
		`),
		Entry("i16 to str", `
			func testFunc() {
				x i16 := -32768
				y := str(x)
			}
		`),
		Entry("i32 to str", `
			func testFunc() {
				x i32 := 42
				y := str(x)
			}
		`),
		Entry("i32 to str (negative)", `
			func testFunc() {
				x i32 := -2147483648
				y := str(x)
			}
		`),
		Entry("i64 to str", `
			func testFunc() {
				x i64 := 9223372036854775807
				y := str(x)
			}
		`),
		Entry("i64 to str (negative)", `
			func testFunc() {
				x i64 := -9223372036854775808
				y := str(x)
			}
		`),
		Entry("u8 to str", `
			func testFunc() {
				x u8 := 255
				y := str(x)
			}
		`),
		Entry("u16 to str", `
			func testFunc() {
				x u16 := 65535
				y := str(x)
			}
		`),
		Entry("u32 to str", `
			func testFunc() {
				x u32 := 4000000000
				y := str(x)
			}
		`),
		Entry("u64 to str", `
			func testFunc() {
				x u64 := 18000000000000000000
				y := str(x)
			}
		`),
		Entry("f32 to str", `
			func testFunc() {
				x f32 := 3.14
				y := str(x)
			}
		`),
		Entry("f32 to str (negative)", `
			func testFunc() {
				x f32 := -3.14
				y := str(x)
			}
		`),
		Entry("f32 to str (integer-valued, trailing zero)", `
			func testFunc() {
				x f32 := 1.0
				y := str(x)
			}
		`),
		Entry("f32 to str (single trailing zero)", `
			func testFunc() {
				x f32 := 3.10
				y := str(x)
			}
		`),
		Entry("f32 to str (multiple trailing zeros)", `
			func testFunc() {
				x f32 := 100.000
				y := str(x)
			}
		`),
		Entry("f64 to str", `
			func testFunc() {
				x f64 := 3.14159
				y := str(x)
			}
		`),
		Entry("f64 to str (negative)", `
			func testFunc() {
				x f64 := -3.14159
				y := str(x)
			}
		`),
		Entry("f64 to str (integer-valued, trailing zero)", `
			func testFunc() {
				x f64 := 1.0
				y := str(x)
			}
		`),
		Entry("f64 to str (single trailing zero)", `
			func testFunc() {
				x f64 := 3.10
				y := str(x)
			}
		`),
		Entry("f64 to str (multiple trailing zeros)", `
			func testFunc() {
				x f64 := 100.000
				y := str(x)
			}
		`),
		Entry("integer literal to str", `
			func testFunc() {
				y := str(42)
			}
		`),
		Entry("integer literal to str (negative)", `
			func testFunc() {
				y := str(-42)
			}
		`),
		Entry("float literal to str", `
			func testFunc() {
				y := str(3.14)
			}
		`),
		Entry("float literal to str (negative)", `
			func testFunc() {
				y := str(-3.14)
			}
		`),
		Entry("float literal to str (negative zero)", `
			func testFunc() {
				y := str(-0.0)
			}
		`),
		Entry("float literal to str (negative zero with trailing zeros)", `
			func testFunc() {
				y := str(-0.0000)
			}
		`),
		Entry("float literal to str (trailing zero)", `
			func testFunc() {
				y := str(1.0)
			}
		`),
		Entry("float literal to str (multiple trailing zeros)", `
			func testFunc() {
				y := str(100.000)
			}
		`),
		Entry("str cast in concat expression", `
			func testFunc() {
				x i32 := 42
				y := str(x) + " items"
			}
		`),
	)

	DescribeTable(
		"String to Numeric Casts (rejected)",
		func(ctx SpecContext, code string) { expectFailure(ctx, code, nil, "cannot cast") },
		Entry("str to i8", `
			func testFunc() {
				x str := "hello"
				y := i8(x)
			}
		`),
		Entry("str to i16", `
			func testFunc() {
				x str := "hello"
				y := i16(x)
			}
		`),
		Entry("str to i32", `
			func testFunc() {
				x str := "hello"
				y := i32(x)
			}
		`),
		Entry("str to i64", `
			func testFunc() {
				x str := "hello"
				y := i64(x)
			}
		`),
		Entry("str to u8", `
			func testFunc() {
				x str := "hello"
				y := u8(x)
			}
		`),
		Entry("str to u16", `
			func testFunc() {
				x str := "hello"
				y := u16(x)
			}
		`),
		Entry("str to u32", `
			func testFunc() {
				x str := "hello"
				y := u32(x)
			}
		`),
		Entry("str to u64", `
			func testFunc() {
				x str := "hello"
				y := u64(x)
			}
		`),
		Entry("str to f32", `
			func testFunc() {
				x str := "hello"
				y := f32(x)
			}
		`),
		Entry("str to f64", `
			func testFunc() {
				x str := "hello"
				y := f64(x)
			}
		`),
	)

	Describe("Type Cast Failures", func() {
		It("Should reject unknown type in cast", func(ctx SpecContext) {
			expectFailure(ctx, `
				func testFunc() {
					x i32 := 42
					y := unknownType(x)
				}
			`, nil, "undefined")
		})
	})
})
