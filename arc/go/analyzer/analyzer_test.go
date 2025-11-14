// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package analyzer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/diagnostics"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
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
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			diagnostic := (*ctx.Diagnostics)[0]
			Expect(diagnostic.Message).To(Equal("name dog conflicts with existing symbol at line 2, col 4"))
			Expect(diagnostic.Line).To(Equal(5))
			Expect(diagnostic.Severity).To(Equal(diagnostics.Error))
		})

		It("Should correctly diagnose a variable declaration that shadows a function", func() {
			prog := MustSucceed(parser.Parse(`
				func dog() {
					dog := 1
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			diagnostic := (*ctx.Diagnostics)[0]
			Expect(diagnostic.Message).To(Equal("name dog conflicts with existing symbol at line 2, col 4"))
		})

		It("Should correctly diagnose a function with duplicate parameter names", func() {
			prog := MustSucceed(parser.Parse(`
				func dog(age i32, age i32) {
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			diagnostic := (*ctx.Diagnostics)[0]
			Expect(diagnostic.Message).To(Equal("duplicate input age"))
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
				ctx := context.CreateRoot(bCtx, prog, nil)
				Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("type mismatch: cannot assign str to i32"))
			})

			It("Should allow compatible types in local variable declaration", func() {
				ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x i32 := 42
				}
				`))
				ctx := context.CreateRoot(bCtx, ast, nil)
				Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())
			})

			It("Should infer types from an int literal", func() {
				prog := MustSucceed(parser.Parse(`
				func testFunc() {
					x := 42
				}
				`))
				ctx := context.CreateRoot(bCtx, prog, nil)
				Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())
				Expect(*ctx.Diagnostics).To(BeEmpty())
				funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "testFunc"))
				Expect(funcScope.ID).To(Equal(0))
				Expect(funcScope.Name).To(Equal("testFunc"))
				blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
				varScope := MustSucceed(blockScope.Resolve(ctx, "x"))
				Expect(varScope.ID).To(Equal(0))
				Expect(varScope.Name).To(Equal("x"))
				// Integer literals default to i64 after unification
				Expect(varScope.Type).To(Equal(types.I64()))
			})

			It("Should infer types from a float literal", func() {
				prog := MustSucceed(parser.Parse(`
				func testFunc() {
					x := 42.0
				}
				`))
				ctx := context.CreateRoot(bCtx, prog, nil)
				Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())
				Expect(*ctx.Diagnostics).To(BeEmpty())
				funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "testFunc"))
				Expect(funcScope.ID).To(Equal(0))
				Expect(funcScope.Name).To(Equal("testFunc"))
				blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
				varScope := MustSucceed(blockScope.Resolve(ctx, "x"))
				Expect(varScope.ID).To(Equal(0))
				Expect(varScope.Name).To(Equal("x"))
				Expect(varScope.Type).To(Equal(types.F64()))
			})

			It("Should automatically cast an int literal to a floating point type", func() {
				prog := MustSucceed(parser.Parse(`
				func testFunc() {
					x f32 := 42
				}
				`))
				ctx := context.CreateRoot(bCtx, prog, nil)
				Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())
				Expect(*ctx.Diagnostics).To(BeEmpty())
				funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "testFunc"))
				Expect(funcScope.ID).To(Equal(0))
				Expect(funcScope.Name).To(Equal("testFunc"))
				blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
				varScope := MustSucceed(blockScope.Resolve(ctx, "x"))
				Expect(varScope.ID).To(Equal(0))
				Expect(varScope.Name).To(Equal("x"))
				Expect(varScope.Type).To(Equal(types.F32()))
			})

			It("Should not allow assignment of a float literal to an int type", func() {
				prog := MustSucceed(parser.Parse(`
				func testFunc() {
					x i32 := 42.0
				}
				`))
				ctx := context.CreateRoot(bCtx, prog, nil)
				Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				first := (*ctx.Diagnostics)[0]
				Expect(first.Message).To(ContainSubstring("does not satisfy float constraint"))
			})

			It("Should allow for variable declaration from a function parameter", func() {
				prog := MustSucceed(parser.Parse(`
					func testFunc(a i64) {
						b := a
					}
				`))
				ctx := context.CreateRoot(bCtx, prog, nil)
				Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())
			})
		})
	})

	Describe("Assignment", func() {
		It("Should return an error diagnostic when the variable being assigned to was not declared", func() {
			prog := MustSucceed(parser.Parse(`
				func dog() {
					my_var i32 := 1
					cat str := "abc"
					bob = cat
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			first := (*ctx.Diagnostics)[0]
			Expect(first.Message).To(ContainSubstring("undefined symbol: bob"))
			Expect(first.Line).To(Equal(5))
			Expect(first.Column).To(Equal(5))
		})

		It("Should return an error diagnostic when the variable on the right hand side is not declared", func() {
			prog := MustSucceed(parser.Parse(`
				func dog() {
					my_var i32 := 1
					cat str := "abc"
					cat = bob
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			first := (*ctx.Diagnostics)[0]
			Expect(first.Message).To(ContainSubstring("undefined symbol: bob"))
		})

		It("Should return an error when assignment is attempted between incompatible types", func() {
			prog := MustSucceed(parser.Parse(`
				func dog() {
					v1 i32 := 1
					v2 str := "abc"
					v2 = v1
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			first := (*ctx.Diagnostics)[0]
			Expect(first.Message).To(ContainSubstring("type mismatch: cannot assign i32 to variable of type str"))
		})
	})

	Describe("Functions", func() {
		Describe("Input, Output, and Config Binding", func() {
			It("Should bind function input and output types to the function signature", func() {
				prog := MustSucceed(parser.Parse(`
					func add(x f64, y f64) f64 {
						return x + y
					}
				`))
				ctx := context.CreateRoot(bCtx, prog, nil)
				Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())
				Expect(*ctx.Diagnostics).To(BeEmpty())
				funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "add"))
				Expect(funcScope.ID).To(Equal(0))
				Expect(funcScope.Name).To(Equal("add"))
				output := MustBeOk(funcScope.Type.Outputs.Get(ir.DefaultOutputParam))
				Expect(output.Type).To(Equal(types.F64()))
				Expect(funcScope.Type.Inputs).To(HaveLen(2))
				p0 := funcScope.Type.Inputs[0]
				Expect(p0.Name).To(Equal("x"))
				Expect(p0.Type).To(Equal(types.F64()))
				p1 := funcScope.Type.Inputs[1]
				Expect(p1.Name).To(Equal("y"))
				Expect(p1.Type).To(Equal(types.F64()))
				Expect(funcScope.Children).To(HaveLen(3))
				paramChildren := funcScope.FilterChildrenByKind(symbol.KindInput)
				Expect(paramChildren).To(HaveLen(2))
				Expect(paramChildren[0].Name).To(Equal("x"))
				Expect(paramChildren[1].Type).To(Equal(types.F64()))
				Expect(paramChildren[1].Name).To(Equal("y"))
				Expect(paramChildren[1].Type).To(Equal(types.F64()))
				blockChildren := funcScope.FilterChildrenByKind(symbol.KindBlock)
				Expect(blockChildren).To(HaveLen(1))
			})

			It("Should bind func config, runtime params and output types to the func signature", func() {
				prog := MustSucceed(parser.Parse(`
				func controller{
					setpoint f64
					sensor <-chan f64
					actuator ->chan f64
				} (enable u8) f64 {
					return 1.0
				}
				`))
				ctx := context.CreateRoot(bCtx, prog, nil)
				Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())
				Expect(*ctx.Diagnostics).To(BeEmpty())
				fScope := MustSucceed(ctx.Scope.Resolve(ctx, "controller"))
				Expect(fScope.ID).To(Equal(0))
				Expect(fScope.Name).To(Equal("controller"))
				output := MustBeOk(fScope.Type.Outputs.Get(ir.DefaultOutputParam))
				Expect(output.Type).To(Equal(types.F64()))

				By("Having the correct config parameters")
				Expect(fScope.Type.Config).To(HaveLen(3))
				p0 := fScope.Type.Config[0]
				Expect(p0.Name).To(Equal("setpoint"))
				Expect(p0.Type).To(Equal(types.F64()))
				p1 := fScope.Type.Config[1]
				Expect(p1.Name).To(Equal("sensor"))
				Expect(p1.Type).To(Equal(types.Chan(types.F64())))
				p2 := fScope.Type.Config[2]
				Expect(p2.Name).To(Equal("actuator"))
				Expect(p2.Type).To(Equal(types.Chan(types.F64())))

				By("Having the correct parameters")
				Expect(fScope.Type.Inputs).To(HaveLen(1))
				p0 = fScope.Type.Inputs[0]
				Expect(p0.Name).To(Equal("enable"))
				Expect(p0.Type).To(Equal(types.U8()))

				By("Having the correct symbols")
				Expect(len(fScope.Children)).To(Equal(5))
				configScopeParamScopes := fScope.FilterChildrenByKind(symbol.KindConfig)
				Expect(configScopeParamScopes).To(HaveLen(3))
				Expect(configScopeParamScopes[0].Name).To(Equal("setpoint"))
				Expect(configScopeParamScopes[0].ID).To(Equal(0))
				Expect(configScopeParamScopes[1].Name).To(Equal("sensor"))
				Expect(configScopeParamScopes[1].ID).To(Equal(1))
			})

			It("Should correctly analyze a PID function", func() {
				prog := MustSucceed(parser.Parse(`
					func pid{
						kp f32
						ki f32
						kd f32
						setpoint f32
					} (input u8) f64 {
						error := setpoint - measurement
						p := kp * error
						last_measurement_time $= measurement_time
						dt := measurement_time - last_measurement_time
						integral $= 0
						integral = integral + error * f32(dt)
						i := ki * integral
						last_error $= error
						derivative := (error - last_error) / f32(dt)
						d := kd * derivative
						return p + i + d
					}
				`))
				resolver := symbol.MapResolver{
					"measurement": symbol.Symbol{
						Name: "measurement",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F32()),
						ID:   5,
					},
					"measurement_time": symbol.Symbol{
						Name: "measurement_time",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I64()),
						ID:   5,
					},
				}
				ctx := context.CreateRoot(bCtx, prog, resolver)
				Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
				Expect(*ctx.Diagnostics).To(BeEmpty())
			})
		})

		Describe("Output Statements", func() {
			It("Should return true for a valid return type on a function", func() {
				prog := MustSucceed(parser.Parse(`
				func dog() i64 {
					return 12
				}
			`))
				ctx := context.CreateRoot(bCtx, prog, nil)
				Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())

			})

			It("Should correctly infer a literal return type", func() {
				prog := MustSucceed(parser.Parse(`
				func dog() i32 {
					return 12
				}
			`))
				ctx := context.CreateRoot(bCtx, prog, nil)
				Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())

			})

			It("Should correctly infer an expression literal return type", func() {
				prog := MustSucceed(parser.Parse(`
				func dog() i32 {
					return 1 + 1
				}
			`))
				ctx := context.CreateRoot(bCtx, prog, nil)
				Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())

			})

			It("Should return an error for a floating point literal on an integer return", func() {
				prog := MustSucceed(parser.Parse(`
				func dog() i32 {
					return 1.0
				}
			`))
				ctx := context.CreateRoot(bCtx, prog, nil)
				Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				first := (*ctx.Diagnostics)[0]
				Expect(first.Message).To(ContainSubstring("does not satisfy float constraint"))
			})

			It("Should not return an error for an integer literal on a floating point return", func() {
				prog := MustSucceed(parser.Parse(`
				func dog() f32 {
					return 12
				}
			`))
				ctx := context.CreateRoot(bCtx, prog, nil)
				Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())

			})

			It("Should return an error when there is a return statement on a void function", func() {
				prog := MustSucceed(parser.Parse(`
				func dog() {
					return 5
				}
			`))
				ctx := context.CreateRoot(bCtx, prog, nil)
				Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				first := (*ctx.Diagnostics)[0]
				Expect(first.Message).To(ContainSubstring("unexpected return value in function/func with void return type"))
			})

			It("Should return an error for a missing return with a function that has a return type", func() {
				prog := MustSucceed(parser.Parse(`
				func dog() f64 {
				}
			`))
				ctx := context.CreateRoot(bCtx, prog, nil)
				Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				first := (*ctx.Diagnostics)[0]
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
				ctx := context.CreateRoot(bCtx, prog, nil)
				Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				first := (*ctx.Diagnostics)[0]
				Expect(first.Message).To(Equal("function 'dog' must return a value of type f64 on all paths"))
			})

			It("Should return an error when the function doesn't have a return type on all code paths (deeply nested)", func() {
				prog := MustSucceed(parser.Parse(`
				func dog() f64 {
					if (5 > 3) {
						return 2.3
					} else {
						if (12 > 14) {
							if (5 < 7) {
								return 7
							}
						} else {
							return 5
						}
					}
				}
				`))
				ctx := context.CreateRoot(bCtx, prog, nil)
				Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				first := (*ctx.Diagnostics)[0]
				Expect(first.Message).To(Equal("function 'dog' must return a value of type f64 on all paths"))
			})
		})

		Describe("Channels", func() {
			It("Should correctly bind global channels used in function body", func() {
				prog := MustSucceed(parser.Parse(`
					func add() f32 {
						return (ox_pt_1 + ox_pt_2) / 2
					}
				`))
				resolver := symbol.MapResolver{
					"ox_pt_1": symbol.Symbol{
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F32()),
						ID:   12,
					},
					"ox_pt_2": symbol.Symbol{
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F32()),
						ID:   13,
					},
				}
				ctx := context.CreateRoot(bCtx, prog, resolver)
				Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
				f := MustSucceed(ctx.Scope.Resolve(ctx, "add"))
				Expect(f.Channels.Read).To(HaveLen(2))
				Expect(f.Channels.Write).To(BeEmpty())
				Expect(f.Channels.Read.Contains(12)).To(BeTrue())
				Expect(f.Channels.Read.Contains(13)).To(BeTrue())
			})
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
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())

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
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())
			funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "dog"))
			Expect(funcScope.ID).To(Equal(0))
			Expect(funcScope.Name).To(Equal("dog"))
			blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
			blocks := blockScope.FilterChildrenByKind(symbol.KindBlock)
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
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())
			funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "dog"))
			Expect(funcScope.ID).To(Equal(0))
			Expect(funcScope.Name).To(Equal("dog"))
			blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
			blocks := blockScope.FilterChildrenByKind(symbol.KindBlock)
			Expect(blocks).To(HaveLen(1))
			firstBlock := blocks[0]
			Expect(firstBlock.Children).To(HaveLen(1))
			firstChild := firstBlock.Children[0]
			Expect(firstChild).ToNot(BeNil())
			Expect(firstChild.Name).To(Equal("b"))
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
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())
			funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "dog"))
			Expect(funcScope.ID).To(Equal(0))
			Expect(funcScope.Name).To(Equal("dog"))
			blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
			blocks := blockScope.FilterChildrenByKind(symbol.KindBlock)
			Expect(blocks).To(HaveLen(3))
			firstBlock := blocks[0]
			Expect(firstBlock.Children).To(BeEmpty())
			secondBlock := blocks[1]
			Expect(secondBlock.Children).To(HaveLen(1))
			secondBlockFirstChild := secondBlock.Children[0]
			Expect(secondBlockFirstChild.Name).To(Equal("c"))
			thirdBlock := blocks[2]
			Expect(thirdBlock.Children).To(BeEmpty())
		})
	})

	Describe("Optional Parameters", func() {
		It("Should parse and store default values for optional input parameters", func() {
			prog := MustSucceed(parser.Parse(`
				func add(x i64, y i64 = 0) i64 {
					return x + y
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())

			// Find the function scope
			funcScope := ctx.Scope.FindChildByName("add")
			Expect(funcScope).ToNot(BeNil())
			Expect(funcScope.Type.Inputs).To(HaveLen(2))

			// Check that y has a default value
			p0 := funcScope.Type.Inputs[0]
			Expect(p0.Name).To(Equal("x"))
			Expect(p0.Type).To(Equal(types.I64()))
			p1 := funcScope.Type.Inputs[1]
			Expect(p1.Name).To(Equal("y"))
			Expect(p1.Type).To(Equal(types.I64()))
			Expect(p1.Value).To(Equal(int64(0)))
		})

		It("Should reject required parameters after optional parameters", func() {
			prog := MustSucceed(parser.Parse(`
				func add(x i64 = 0, y i64) i64 {
					return x + y
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			diagnostic := (*ctx.Diagnostics)[0]
			Expect(diagnostic.Message).To(ContainSubstring("required parameter y cannot follow optional parameters"))
		})

		It("Should handle multiple optional parameters", func() {
			prog := MustSucceed(parser.Parse(`
				func multi(a i32, b f64 = 1.5, c u8 = 10) f64 {
					return f64(a) + b + f64(c)
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())

			funcScope := ctx.Scope.FindChildByName("multi")
			Expect(funcScope).ToNot(BeNil())
			Expect(funcScope.Type.Inputs).To(HaveLen(3))
			p0 := funcScope.Type.Inputs[0]
			Expect(p0.Name).To(Equal("a"))
			Expect(p0.Type).To(Equal(types.I32()))
			p1 := funcScope.Type.Inputs[1]
			Expect(p1.Name).To(Equal("b"))
			Expect(p1.Type).To(Equal(types.F64()))
			Expect(p1.Value).To(Equal(1.5))
			p2 := funcScope.Type.Inputs[2]
			Expect(p2.Name).To(Equal("c"))
			Expect(p2.Type).To(Equal(types.U8()))
			Expect(p2.Value).To(Equal(uint8(10)))
		})

		It("Should handle functions with no optional parameters", func() {
			prog := MustSucceed(parser.Parse(`
				func multiply(x i64, y i64) i64 {
					return x * y
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())

			funcScope := ctx.Scope.FindChildByName("multiply")
			Expect(funcScope).ToNot(BeNil())
			for _, p := range funcScope.Type.Inputs {
				Expect(p.Value).To(BeNil())
			}
		})

		It("Should reject default values that overflow target type", func() {
			prog := MustSucceed(parser.Parse(`
				func foo(x i8 = 128) i8 {
					return x
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).ToNot(BeEmpty())
			diagnostic := (*ctx.Diagnostics)[0]
			Expect(diagnostic.Message).To(ContainSubstring("out of range for i8"))
		})

		It("Should reject default values with non-exact float-to-int conversion", func() {
			prog := MustSucceed(parser.Parse(`
				func foo(x i32 = 3.14) i32 {
					return x
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).ToNot(BeEmpty())
			diagnostic := (*ctx.Diagnostics)[0]
			Expect(diagnostic.Message).To(ContainSubstring("cannot convert non-integer float"))
		})
	})
})
