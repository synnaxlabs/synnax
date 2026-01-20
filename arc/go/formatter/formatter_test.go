// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package formatter_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/formatter"
)

var _ = Describe("Formatter", func() {
	cfg := formatter.DefaultConfig

	Describe("Binary Operators", func() {
		It("should add spaces around :=", func() {
			input := "x:=42"
			expected := "x := 42\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should add spaces around $=", func() {
			input := "count$=0"
			expected := "count $= 0\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should add spaces around arithmetic operators", func() {
			input := "x:=a+b*c-d/e%f"
			expected := "x := a + b * c - d / e % f\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should add spaces around comparison operators", func() {
			input := "x==y"
			expected := "x == y\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should add spaces around logical operators", func() {
			input := "x and y or z"
			expected := "x and y or z\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should add spaces around flow operators", func() {
			input := "a->b=>c"
			expected := "a -> b => c\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should add spaces around compound assignment operators", func() {
			input := "x+=5"
			expected := "x += 5\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})
	})

	Describe("Unit Literals", func() {
		It("should not add space between number and unit suffix", func() {
			input := "delay := 100ms"
			expected := "delay := 100ms\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should not add space between float and unit suffix", func() {
			input := "pressure := 14.7psi"
			expected := "pressure := 14.7psi\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})
	})

	Describe("Functions", func() {
		It("should format simple function", func() {
			input := "func add(x i32,y i32)i32{return x+y}"
			expected := "func add(x i32, y i32) i32 {\n    return x + y\n}\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should format function with config block", func() {
			input := "func threshold{limit f64}(value f64)u8{return u8(0)}"
			expected := "func threshold {\n    limit f64\n} (value f64) u8 {\n    return u8(0)\n}\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should format empty function body", func() {
			input := "func noop(){}"
			expected := "func noop() {}\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})
	})

	Describe("Sequences", func() {
		It("should format sequence with stages", func() {
			input := "sequence main{stage first{}stage second{}}"
			expected := "sequence main {\n    stage first {}\n    stage second {}\n}\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})
	})

	Describe("Control Flow", func() {
		It("should format if statement", func() {
			input := "if x>0{return 1}"
			expected := "if x > 0 {\n    return 1\n}\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should format if-else statement", func() {
			input := "if x>0{return 1}else{return 0}"
			expected := "if x > 0 {\n    return 1\n} else {\n    return 0\n}\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})
	})

	Describe("Comments", func() {
		It("should preserve single-line comments", func() {
			input := "// comment\nx := 1"
			expected := "// comment\nx := 1\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should preserve trailing comments", func() {
			input := "x := 1 // comment"
			expected := "x := 1 // comment\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should preserve multi-line comments", func() {
			input := "/* multi\nline */ x := 1"
			expected := "/* multi\nline */\nx := 1\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})
	})

	Describe("Series Literals", func() {
		It("should format series literal", func() {
			input := "[1,2,3]"
			expected := "[1, 2, 3]\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})

		It("should format empty series", func() {
			input := "[]"
			expected := "[]\n"
			Expect(formatter.Format(input, cfg)).To(Equal(expected))
		})
	})

	Describe("Idempotency", func() {
		It("should be idempotent", func() {
			input := "func add(x i32, y i32) i32 {\n    return x + y\n}\n"
			firstPass := formatter.Format(input, cfg)
			secondPass := formatter.Format(firstPass, cfg)
			Expect(secondPass).To(Equal(firstPass))
		})

		It("should produce same output for messy input", func() {
			input := "func   add(x i32,y i32)i32{return   x+y}"
			firstPass := formatter.Format(input, cfg)
			secondPass := formatter.Format(firstPass, cfg)
			Expect(secondPass).To(Equal(firstPass))
		})
	})

	Describe("Edge Cases", func() {
		It("should handle empty input", func() {
			input := ""
			Expect(formatter.Format(input, cfg)).To(Equal(""))
		})

		It("should handle whitespace-only input", func() {
			input := "   \n\n   "
			Expect(formatter.Format(input, cfg)).To(Equal("   \n\n   "))
		})
	})
})
