package analyzer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/slate/analyzer"
	"github.com/synnaxlabs/slate/analyzer/result"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/slate/symbol"
	"github.com/synnaxlabs/slate/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Analyzer", func() {
	Describe("Duplicate Scope", func() {
		It("Should correctly diagnose a duplicate function declaration", func() {
			prog := MustSucceed(parser.Parse(`
				func dog() {
				}

				func dog() {
				}
			`))
			r := analyzer.Analyze(prog, analyzer.Options{})
			Expect(r.Diagnostics).To(HaveLen(1))
			diagnostic := r.Diagnostics[0]
			Expect(diagnostic.Message).To(Equal("name dog conflicts with existing symbol at line 2, col 4"))
			Expect(diagnostic.Line).To(Equal(5))
			Expect(diagnostic.Severity).To(Equal(result.Error))
		})

		It("Should correctly diagnose a variable declaration that shadows a function", func() {
			prog := MustSucceed(parser.Parse(`
				func dog() {
					dog := 1
				}
			`))
			result := analyzer.Analyze(prog, analyzer.Options{})
			Expect(result.Diagnostics).To(HaveLen(1))
			diagnostic := result.Diagnostics[0]
			Expect(diagnostic.Message).To(Equal("name dog conflicts with existing symbol at line 2, col 4"))
		})

		It("Should correctly diagnose a function with duplicate parameter names", func() {
			prog := MustSucceed(parser.Parse(`
				func dog(age i32, age i32) {
				}
		`))
			result := analyzer.Analyze(prog, analyzer.Options{})
			Expect(result.Diagnostics).To(HaveLen(1))
			diagnostic := result.Diagnostics[0]
			Expect(diagnostic.Message).To(Equal("duplicate parameter age"))
		})
	})

	Describe("Declaration", func() {
		Describe("Local", func() {
			It("Should return an error diagnostic when a string is declared on an i32", func() {
				prog := MustSucceed(parser.Parse(`
					func cat() {
						my_var i32 := "dog"
					}
				`))
				result := analyzer.Analyze(prog, analyzer.Options{})
				Expect(result.Diagnostics).To(HaveLen(1))
				Expect(result.Diagnostics[0].Message).To(ContainSubstring("type mismatch: cannot assign string to i32"))
			})

			It("Should allow compatible types in local variable declaration", func() {
				ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x i32 := 42
				}
				`))
				result := analyzer.Analyze(ast, analyzer.Options{})
				Expect(result.Diagnostics).To(HaveLen(0))
			})

			It("Should infer types from an int literal", func() {
				prog := MustSucceed(parser.Parse(`
				func testFunc() {
					x := 42
				}
				`))
				result := analyzer.Analyze(prog, analyzer.Options{})
				Expect(result.Diagnostics).To(HaveLen(0))
				funcScope := MustSucceed(result.Symbols.Get("testFunc"))
				Expect(funcScope.Symbol.ID).To(Equal(0))
				Expect(funcScope.Symbol).ToNot(BeNil())
				Expect(funcScope.Symbol).ToNot(BeNil())
				Expect(funcScope.Symbol.Name).To(Equal("testFunc"))
				blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
				varScope := MustSucceed(blockScope.Get("x"))
				Expect(varScope.Symbol.ID).To(Equal(0))
				Expect(varScope.Symbol).ToNot(BeNil())
				Expect(varScope.Symbol.Name).To(Equal("x"))
				Expect(varScope.Symbol.Type).To(Equal(types.I64{}))
			})

			It("Should infer types from a float literal", func() {
				prog := MustSucceed(parser.Parse(`
				func testFunc() {
					x := 42.0
				}
				`))
				result := analyzer.Analyze(prog, analyzer.Options{})
				Expect(result.Diagnostics).To(HaveLen(0))
				funcScope := MustSucceed(result.Symbols.Get("testFunc"))
				Expect(funcScope.Symbol.ID).To(Equal(0))
				Expect(funcScope.Symbol).ToNot(BeNil())
				Expect(funcScope.Symbol).ToNot(BeNil())
				Expect(funcScope.Symbol.Name).To(Equal("testFunc"))
				blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
				varScope := MustSucceed(blockScope.Get("x"))
				Expect(varScope.Symbol.ID).To(Equal(0))
				Expect(varScope.Symbol).ToNot(BeNil())
				Expect(varScope.Symbol.Name).To(Equal("x"))
				Expect(varScope.Symbol.Type).To(Equal(types.F64{}))
			})

			It("Should automatically cast an int literal to a floating point type", func() {
				prog := MustSucceed(parser.Parse(`
				func testFunc() {
					x f32 := 42
				}
				`))
				result := analyzer.Analyze(prog, analyzer.Options{})
				Expect(result.Diagnostics).To(HaveLen(0))
				funcScope := MustSucceed(result.Symbols.Get("testFunc"))
				Expect(funcScope.Symbol.ID).To(Equal(0))
				Expect(funcScope.Symbol).ToNot(BeNil())
				Expect(funcScope.Symbol.Name).To(Equal("testFunc"))
				blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
				varScope := MustSucceed(blockScope.Get("x"))
				Expect(varScope.Symbol.ID).To(Equal(0))
				Expect(varScope.Symbol).ToNot(BeNil())
				Expect(varScope.Symbol.Name).To(Equal("x"))
				Expect(varScope.Symbol.Type).To(Equal(types.F32{}))
			})

			It("Should not allow assignment of a float literal to an int type", func() {
				prog := MustSucceed(parser.Parse(`
				func testFunc() {
					x i32 := 42.0
				}
				`))
				result := analyzer.Analyze(prog, analyzer.Options{})
				Expect(result.Ok()).To(BeFalse())
				Expect(result.Diagnostics).To(HaveLen(1))
				first := result.Diagnostics[0]
				Expect(first.Message).To(Equal("type mismatch: cannot assign f64 to i32"))
			})

			It("Should allow for variable declaration from a function parameter", func() {
				prog := MustSucceed(parser.Parse(`
					func testFunc(a i64) {
						b := a
					}
				`))
				result := analyzer.Analyze(prog, analyzer.Options{})
				Expect(result.Ok()).To(BeTrue())
				Expect(result.Diagnostics).To(HaveLen(0))
			})
		})
	})

	Describe("Assignment", func() {
		It("Should return an error diagnostic when the variable being assigned to was not declared", func() {
			prog := MustSucceed(parser.Parse(`
				func dog() {
					my_var i32 := 1
					cat string := "abc"
					bob = cat
				}
			`))
			result := analyzer.Analyze(prog, analyzer.Options{})
			Expect(result.Diagnostics).To(HaveLen(1))
			first := result.Diagnostics[0]
			Expect(first.Message).To(ContainSubstring("undefined symbol: bob"))
			Expect(first.Line).To(Equal(5))
			Expect(first.Column).To(Equal(5))
		})

		It("Should return an error diagnostic when the variable on the right hand side is not declared", func() {
			prog := MustSucceed(parser.Parse(`
				func dog() {
					my_var i32 := 1
					cat string := "abc"
					cat = bob
				}
			`))
			result := analyzer.Analyze(prog, analyzer.Options{})
			Expect(result.Diagnostics).To(HaveLen(1))
			first := result.Diagnostics[0]
			Expect(first.Message).To(ContainSubstring("undefined symbol: bob"))
		})

		It("Should return an error when assignment is attempted between incompatible types", func() {
			prog := MustSucceed(parser.Parse(`
				func dog() {
					v1 i32 := 1
					v2 string := "abc"
					v2 = v1
				}
			`))
			result := analyzer.Analyze(prog, analyzer.Options{})
			Expect(result.Diagnostics).To(HaveLen(1))
			first := result.Diagnostics[0]
			Expect(first.Message).To(ContainSubstring("type mismatch: cannot assign i32 to variable of type string"))
		})
	})

	Describe("Type Signatures", func() {
		It("Should bind function parameter and return types to the function signature", func() {
			prog := MustSucceed(parser.Parse(`
				func add(x f64, y f64) f64 {
					return x + y
				}
			`))
			result := analyzer.Analyze(prog, analyzer.Options{})
			Expect(result.Diagnostics).To(HaveLen(0))
		})

		It("Should bind task config, runtime params and return types to the task signature", func() {
			prog := MustSucceed(parser.Parse(`
				task controller{
					setpoint f64
					sensor <-chan f64
					actuator ->chan f64
				} (enable u8) f64 {
					return 1.0
				}
			`))
			result := analyzer.Analyze(prog, analyzer.Options{})
			Expect(result.Diagnostics).To(HaveLen(0))
		})
	})

	Describe("Return", func() {
		It("Should return true for a valid return type on a function", func() {
			prog := MustSucceed(parser.Parse(`
				func dog() i64 {
					return 12
				}
			`))
			result := analyzer.Analyze(prog, analyzer.Options{})
			Expect(result.Ok()).To(BeTrue(), result.String())
		})

		It("Should correctly infer a literal return type", func() {
			prog := MustSucceed(parser.Parse(`
				func dog() i32 {
					return 12
				}
			`))
			result := analyzer.Analyze(prog, analyzer.Options{})
			Expect(result.Ok()).To(BeTrue(), result.String())
		})

		It("Should correctly infer an expression literal return type", func() {
			prog := MustSucceed(parser.Parse(`
				func dog() i32 {
					return 1 + 1
				}
			`))
			result := analyzer.Analyze(prog, analyzer.Options{})
			Expect(result.Ok()).To(BeTrue(), result.String())
		})

		It("Should return an error for a floating point literal on an integer return", func() {
			prog := MustSucceed(parser.Parse(`
				func dog() i32 {
					return 1.0
				}
			`))
			result := analyzer.Analyze(prog, analyzer.Options{})
			Expect(result.Diagnostics).To(HaveLen(1))
			first := result.Diagnostics[0]
			Expect(first.Message).To(ContainSubstring("cannot return f64, expected i32"))
		})

		It("Should not return an error for an integer literal on a floating point return", func() {
			prog := MustSucceed(parser.Parse(`
				func dog() f32 {
					return 12
				}
			`))
			result := analyzer.Analyze(prog, analyzer.Options{})
			Expect(result.Ok()).To(BeTrue(), result.String())
		})

		It("Should return an error when there is a return statement on a void function", func() {
			prog := MustSucceed(parser.Parse(`
				func dog() {
					return 5
				}
			`))
			result := analyzer.Analyze(prog, analyzer.Options{})
			Expect(result.Diagnostics).To(HaveLen(1))
			first := result.Diagnostics[0]
			Expect(first.Message).To(ContainSubstring("unexpected return value in function/task with void return type"))
		})

		It("Should return an error for a missing return with a function that has a return type", func() {
			prog := MustSucceed(parser.Parse(`
				func dog() f64 {
				}
			`))
			result := analyzer.Analyze(prog, analyzer.Options{})
			Expect(result.Diagnostics).To(HaveLen(1))
			first := result.Diagnostics[0]
			Expect(first.Message).To(Equal("function 'dog' must return a value of type f64 on all paths"))
		})

		It("Should return an error for a function that doesn't have a return type on all code paths", func() {
			prog := MustSucceed(parser.Parse(`
				func dog() f64 {
					if (5 > 3) {
						return 2.3
					}
				}
			`))
			result := analyzer.Analyze(prog, analyzer.Options{})
			Expect(result.Diagnostics).To(HaveLen(1))
			first := result.Diagnostics[0]
			Expect(first.Message).To(Equal("function 'dog' must return a value of type f64 on all paths"))
		})
	})

	Describe("Control Flow", func() {
		It("Should return no diagnostics for a valid if statement", func() {
			prog := MustSucceed(parser.Parse(`
				func dog() i64 {
					a f32 := 2.0
					if (a > 5) {
						return 1
					}
					return 2
				}
			`))
			result := analyzer.Analyze(prog, analyzer.Options{})
			Expect(result.Ok()).To(BeTrue(), result.String())
		})

		It("Should return the correct symbol table for an if-else statement", func() {
			prog := MustSucceed(parser.Parse(`
				func dog() i64 {
					if 3 > 5 {
						return 1
					} else {
						return 2
					}
				}
			`))
			result := analyzer.Analyze(prog, analyzer.Options{})
			Expect(result.Ok()).To(BeTrue(), result.String())
			funcScope := MustSucceed(result.Symbols.Get("dog"))
			Expect(funcScope.Symbol.ID).To(Equal(0))
			Expect(funcScope.Symbol).ToNot(BeNil())
			Expect(funcScope.Symbol.Name).To(Equal("dog"))
			blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
			blocks := blockScope.Blocks()
			Expect(blocks).To(HaveLen(2))
			Expect(blocks[0].Children).To(BeEmpty())
			Expect(blocks[1].Children).To(BeEmpty())
		})

		It("Should return the correct symbol table for variables declared inside blocks", func() {
			prog := MustSucceed(parser.Parse(`
				func dog() i64 {
					a f32 := 2.0
					if (a > 5) {
						b := 2
						return b
					}
					return 2
				}
			`))
			result := analyzer.Analyze(prog, analyzer.Options{})
			Expect(result.Ok()).To(BeTrue(), result.String())
			funcScope := MustSucceed(result.Symbols.Get("dog"))
			Expect(funcScope.Symbol.ID).To(Equal(0))
			Expect(funcScope.Symbol).ToNot(BeNil())
			Expect(funcScope.Symbol.Name).To(Equal("dog"))
			blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
			blocks := blockScope.Blocks()
			Expect(blocks).To(HaveLen(1))
			firstBlock := blockScope.Blocks()[0]
			Expect(firstBlock.Children).To(HaveLen(1))
			firstChild := firstBlock.Children[0]
			Expect(firstChild).ToNot(BeNil())
			Expect(firstChild.Symbol.Name).To(Equal("b"))
		})

		It("Should return the correct symbol table for variables declared in blocks", func() {
			prog := MustSucceed(parser.Parse(`
		func dog(b i64) i64 {
				a i64 := 2
			if b == a {
				return 2
			} else if a > b {
				c i64 := 5
				return c
			} else {
				return 3
			}
			}
			`))
			result := analyzer.Analyze(prog, analyzer.Options{})
			Expect(result.Ok()).To(BeTrue(), result.String())
			funcScope := MustSucceed(result.Symbols.Get("dog"))
			Expect(funcScope.Symbol.ID).To(Equal(0))
			Expect(funcScope.Symbol).ToNot(BeNil())
			Expect(funcScope.Symbol.Name).To(Equal("dog"))
			blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
			blocks := blockScope.Blocks()
			Expect(blocks).To(HaveLen(3))
			firstBlock := blocks[0]
			Expect(firstBlock.Children).To(HaveLen(0))
			secondBlock := blocks[1]
			Expect(secondBlock.Children).To(HaveLen(1))
			secondBlockFirstChild := secondBlock.Children[0]
			Expect(secondBlockFirstChild.Symbol.Name).To(Equal("c"))
			thirdBlock := blocks[2]
			Expect(thirdBlock.Children).To(HaveLen(0))
		})
	})
})
