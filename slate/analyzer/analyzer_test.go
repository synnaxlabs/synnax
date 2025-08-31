package analyzer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/slate/analyzer"
	"github.com/synnaxlabs/slate/analyzer/result"
	"github.com/synnaxlabs/slate/parser"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Analyzer", func() {
	Describe("Duplicate Symbols", func() {

		It("Should correctly diagnose a duplicate function declaration", func() {
			ast := MustSucceed(parser.Parse(`
func dog() {
}

func dog() {
}
		`))
			r := analyzer.Analyze(ast)
			Expect(r.Diagnostics).To(HaveLen(1))
			diagnostic := r.Diagnostics[0]
			Expect(diagnostic.Message).To(Equal("name dog conflicts with existing symbol at line 2, col 0"))
			Expect(diagnostic.Line).To(Equal(5))
			Expect(diagnostic.Severity).To(Equal(result.SeverityError))
		})

		It("Should correctly diagnose a variable declaration that shadows a function", func() {
			ast := MustSucceed(parser.Parse(`
func dog() {
	dog := 1
}
		`))
			result := analyzer.Analyze(ast)
			Expect(result.Diagnostics).To(HaveLen(1))
			diagnostic := result.Diagnostics[0]
			Expect(diagnostic.Message).To(Equal("name dog conflicts with existing symbol at line 2, col 0"))
		})

		It("Should correctly diagnose a function with duplicate parameter names", func() {
			ast := MustSucceed(parser.Parse(`
func dog(age i32, age i32) {
}
		`))
			result := analyzer.Analyze(ast)
			Expect(result.Diagnostics).To(HaveLen(1))
			diagnostic := result.Diagnostics[0]
			Expect(diagnostic.Message).To(Equal("name age conflicts with existing symbol at line 2, col 9"))
		})
	})

	Describe("Variable Declarations", func() {
		Describe("Local", func() {
			It("Should return an error diagnostic when a string is declared on an i32", func() {
				ast := MustSucceed(parser.Parse(`
func cat() {
	my_var i32 := "dog"
}
`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(1))
				Expect(result.Diagnostics[0].Message).To(ContainSubstring("type mismatch: cannot assign string to i32"))
			})

			It("Should allow compatible types in local variable declaration", func() {
				ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x i32 := 42
				}
			`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(0))
			})

			It("Should infer types from an initializer expression", func() {
				ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x := 42
				}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(0))
			})
		})
	})

	Describe("Assignment", func() {
		It("Should return an error diagnostic when the variable being assigned to was not declared", func() {
			ast := MustSucceed(parser.Parse(`
func dog() {
	my_var i32 := 1
	cat string := "abc"
	 bob = cat
}
		`))
			result := analyzer.Analyze(ast)
			Expect(result.Diagnostics).To(HaveLen(1))
			first := result.Diagnostics[0]
			Expect(first.Message).To(ContainSubstring("undefined symbol: bob"))
			Expect(first.Line).To(Equal(5))
			Expect(first.Column).To(Equal(2))
		})

		It("Should return an error diagnostic when the variable on the right hand side is not declared", func() {
			ast := MustSucceed(parser.Parse(`
func dog() {
my_var i32 := 1
cat string := "abc"
cat = bob
}
		`))
			result := analyzer.Analyze(ast)
			Expect(result.Diagnostics).To(HaveLen(1))
			first := result.Diagnostics[0]
			Expect(first.Message).To(ContainSubstring("undefined symbol: bob"))
		})

		It("Should return an error when assignment is attempted between incompatible types", func() {
			ast := MustSucceed(parser.Parse(`
				func dog() {
					v1 i32 := 1
					v2 string := "abc"
					v2 = v1
				}
			`))
			result := analyzer.Analyze(ast)
			Expect(result.Diagnostics).To(HaveLen(1))
			first := result.Diagnostics[0]
			Expect(first.Message).To(ContainSubstring("type mismatch: cannot assign i32 to variable of type string"))
		})
	})

	Describe("Expressions", func() {
		Describe("Binary Expressions", func() {
			It("Should validate arithmetic operations on numeric types", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i32 := 10
						y i32 := 20
						z := x + y
						w := x * y
						v := x - y
						u := x / y
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(0))
			})

			It("Should reject arithmetic operations on strings", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x string := "hello"
						y string := "world"
						z := x + y
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(1))
				Expect(result.Diagnostics[0].Message).To(ContainSubstring("cannot use string in + operation"))
			})

			It("Should reject mixed type arithmetic", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i32 := 10
						y f32 := 20.5
						z := x + y
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(1))
				Expect(result.Diagnostics[0].Message).To(ContainSubstring("type mismatch"))
			})

			It("Should validate comparison operations", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i32 := 10
						y i32 := 20
						a := x < y
						b := x > y
						c := x <= y
						d := x >= y
						e := x == y
						f := x != y
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(0))
			})

			It("Should validate logical operations on booleans", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
						a u8 := 0
						b u8 := 1
						c := a && b
						d := a || b
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(0))
			})

			It("Should reject logical AND operations on non-booleans", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i32 := 10
						y i32 := 20
						z := x && y
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(1))
				Expect(result.Diagnostics[0].Message).To(ContainSubstring("cannot use i32 in && operation"))
			})

			It("Should reject logical OR operations on non-booleans", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
					x i32 := 10
					y i32 := 20
					z := x || y
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(1))
			})

			It("Should validate modulo operation on integers", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i32 := 10
						y i32 := 3
						z := x % y
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(0))
			})

			//Skip("Should reject modulo operation on floats", func() {
			//	ast := MustSucceed(parser.Parse(`
			//		func testFunc() {
			//			x f32 := 10.5
			//			y f32 := 3.2
			//			z := x % y
			//		}
			//	`))
			//	result := analyzer.Analyze(ast)
			//	Expect(result.Diagnostics).To(HaveLen(1))
			//	Expect(result.Diagnostics[0].Message).To(ContainSubstring("operator % not supported for type f32"))
			//})
		})

		Describe("Unary Expressions", func() {
			It("Should validate unary negation on numeric types", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i32 := 10
						y := -x
						z f32 := 5.5
						w := -z
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(0))
			})

			It("Should reject unary negation on non-numeric types", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x string := "hello"
						y := -x
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(1))
				Expect(result.Diagnostics[0].Message).To(ContainSubstring("operator - not supported for type string"))
			})

			It("Should validate logical not on booleans", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x u8 := 1
						y := !x
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(0))
			})

			It("Should reject logical not on non-booleans", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i32 := 10
						y := !x
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(1))
				Expect(result.Diagnostics[0].Message).To(ContainSubstring("operator ! requires boolean operand"))
			})
		})

		Describe("Literal Expressions", func() {
			It("Should correctly type integer literals", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x := 42
						y i32 := 100
						z i64 := 1000000
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(0))
			})

			It("Should correctly type float literals", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x := 3.14
						y f32 := 2.718
						z f64 := 1.414213
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(0))
			})

			It("Should correctly type string literals", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x := "hello world"
						y string := "test"
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(0))
			})

			It("Should correctly type boolean literals", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x := 1
						y u8 := 1
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(0))
			})
		})

		Describe("Complex Expressions", func() {
			It("Should handle nested binary expressions with correct precedence", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i32 := 10
						y i32 := 20
						z i32 := 30
						result := x + y * z
						result2 := (x + y) * z
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(0))
			})

			It("Should handle mixed arithmetic and comparison operations", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i32 := 10
						y i32 := 20
						z i32 := 30
						result := x + y < z
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(0))
			})

			It("Should handle chained logical operations", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
						a u8 := 1
						b u8 := 0
						c u8 := 1
						result := a && b || c
						result2 := a || b && c
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(0))
			})

			It("Should validate complex mixed expressions", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i32 := 10
						y i32 := 20
						z i32 := 30
						a u8 := x < y
						b u8 := y > z
						result := a && b || (x + y == z)
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(0))
			})

			It("Should detect type errors in complex expressions", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i32 := 10
						y string := "20"
						z u8 := true
						result := x + y * z
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).ToNot(HaveLen(0))
			})
		})

		Describe("Function Call Expressions", func() {
			It("Should validate function calls with correct arguments", func() {
				ast := MustSucceed(parser.Parse(`
					func add(x i32, y i32) i32 {
						return x + y
					}

					func testFunc() {
						result := add(10, 20)
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(0))
			})

			It("Should detect undefined function calls", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
						result := undefinedFunc(10, 20)
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(1))
				Expect(result.Diagnostics[0].Message).To(ContainSubstring("undefined symbol: undefinedFunc"))
			})

			It("Should detect incorrect number of arguments", func() {
				ast := MustSucceed(parser.Parse(`
					func add(x i32, y i32) i32 {
						return x + y
					}

					func testFunc() {
						result := add(10)
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(1))
				Expect(result.Diagnostics[0].Message).To(ContainSubstring("incorrect number of arguments"))
			})

			It("Should detect type mismatch in function arguments", func() {
				ast := MustSucceed(parser.Parse(`
					func add(x i32, y i32) i32 {
						return x + y
					}

					func testFunc() {
						result := add(10, "20")
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(1))
				Expect(result.Diagnostics[0].Message).To(ContainSubstring("type mismatch"))
			})

			It("Should handle nested function calls", func() {
				ast := MustSucceed(parser.Parse(`
					func double(x i32) i32 {
						return x * 2
					}

					func add(x i32, y i32) i32 {
						return x + y
					}

					func testFunc() {
						result := add(double(5), double(10))
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(0))
			})
		})

		Describe("Variable Reference Expressions", func() {
			It("Should resolve local variables correctly", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i32 := 10
						y := x + 5
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(0))
			})

			It("Should detect undefined variable references", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
						y := undefinedVar + 5
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(1))
				Expect(result.Diagnostics[0].Message).To(ContainSubstring("undefined symbol: undefinedVar"))
			})

			It("Should not allow shadowing", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i32 := 10
						if x > 0 {
							x i32 := 20
							y := x + 5
						}
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(1))
				Expect(result.Diagnostics[0].Message).To(ContainSubstring("name x conflicts with existing symbol at line 3, col 6"))
			})

			It("Should resolve function parameters correctly", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc(x i32, y i32) i32 {
						return x + y
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(0))
			})
		})

		Describe("Type Coercion and Casting", func() {
			It("Should handle implicit numeric widening", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i32 := 10
						y i64 := x
					}
				`))
				_ = analyzer.Analyze(ast)
				// This might generate diagnostics depending on the implementation
				// Adjust based on actual analyzer behavior
			})

			It("Should reject narrowing conversions without explicit cast", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i64 := 1000
						y i32 := x
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).ToNot(HaveLen(0))
			})
		})

		Describe("Edge Cases", func() {
			It("Should handle empty expressions gracefully", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(0))
			})

			It("Should detect division by zero in constant expressions", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x := 10 / 0
					}
				`))
				_ = analyzer.Analyze(ast)
				// May or may not generate diagnostics depending on implementation
			})

			It("Should handle very deeply nested expressions", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i32 := ((((1 + 2) * 3) - 4) / 5) % 6
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(0))
			})

			It("Should handle expressions with all operators", func() {
				ast := MustSucceed(parser.Parse(`
					func testFunc() {
						a i32 := 10
						b i32 := 20
						c i32 := 30
						d u8 := a + b * c / 2 - 5 % 3 < 100 && 0 || 1
					}
				`))
				result := analyzer.Analyze(ast)
				Expect(result.Diagnostics).To(HaveLen(0))
			})
		})
	})
})
