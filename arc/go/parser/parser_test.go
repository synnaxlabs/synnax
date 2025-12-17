// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package parser_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/parser"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Parser", func() {
	Describe("Expressions", func() {
		Context("Numeric Literals", func() {
			It("Should parse integer literals", func() {
				expr := mustParseExpression("42")
				Expect(expr).NotTo(BeNil())
				logicalOr := expr.LogicalOrExpression()
				Expect(logicalOr).NotTo(BeNil())
				var (
					logicalAnd     = logicalOr.LogicalAndExpression(0)
					equality       = logicalAnd.EqualityExpression(0)
					relational     = equality.RelationalExpression(0)
					additive       = relational.AdditiveExpression(0)
					multiplicative = additive.MultiplicativeExpression(0)
					power          = multiplicative.PowerExpression(0)
					unary          = power.UnaryExpression()
					postfix        = unary.PostfixExpression()
					primary        = postfix.PrimaryExpression()
					literal        = primary.Literal()
				)
				Expect(literal).NotTo(BeNil())
				Expect(literal.NumericLiteral()).NotTo(BeNil())
				Expect(literal.NumericLiteral().INTEGER_LITERAL()).NotTo(BeNil())
				Expect(literal.NumericLiteral().INTEGER_LITERAL().GetText()).To(Equal("42"))
			})

			It("Should parse float literals", func() {
				expr := mustParseExpression("3.14")
				literal := getPrimaryLiteral(expr)
				Expect(literal.NumericLiteral().FLOAT_LITERAL().GetText()).To(Equal("3.14"))
			})
		})

		Context("Temporal Literals", func() {
			It("Should parse millisecond literals", func() {
				expr := mustParseExpression("100ms")
				literal := getPrimaryLiteral(expr)
				Expect(literal.TemporalLiteral()).NotTo(BeNil())
				Expect(literal.TemporalLiteral().TEMPORAL_LITERAL().GetText()).To(Equal("100ms"))
			})

			It("Should parse frequency literals", func() {
				expr := mustParseExpression("10hz")
				literal := getPrimaryLiteral(expr)
				Expect(literal.TemporalLiteral()).NotTo(BeNil())
				Expect(literal.TemporalLiteral().FREQUENCY_LITERAL().GetText()).To(Equal("10hz"))
			})
		})

		Context("Binary Operations", func() {
			It("Should parse addition", func() {
				expr := mustParseExpression("2 + 3")
				additive := getAdditiveExpression(expr)

				Expect(additive.AllMultiplicativeExpression()).To(HaveLen(2))
				Expect(additive.PLUS(0)).NotTo(BeNil())
			})

			It("Should parse multiplication with correct precedence", func() {
				expr := mustParseExpression("2 + 3 * 4")
				additive := getAdditiveExpression(expr)
				// Should be parsed as 2 + (3 * 4)
				Expect(additive.AllMultiplicativeExpression()).To(HaveLen(2))
				// First term is just "2"
				first := additive.MultiplicativeExpression(0)
				Expect(first.AllPowerExpression()).To(HaveLen(1))
				// Second term is "3 * 4"
				second := additive.MultiplicativeExpression(1)
				Expect(second.AllPowerExpression()).To(HaveLen(2))
				Expect(second.STAR(0)).NotTo(BeNil())
			})

			It("Should parse exponentiation with right associativity", func() {
				expr := mustParseExpression("2 ^ 3 ^ 2")
				// Should be parsed as 2 ^ (3 ^ 2)
				power := getMultiplicativeExpression(expr).PowerExpression(0)
				Expect(power).NotTo(BeNil())
				Expect(power.CARET()).NotTo(BeNil())
				// The right side should be another power expression
				rightPower := power.PowerExpression()
				Expect(rightPower).NotTo(BeNil())
				Expect(rightPower.CARET()).NotTo(BeNil())
			})
		})

		Context("Unary Operations", func() {
			It("Should parse unary minus", func() {
				expr := mustParseExpression("-42")
				unary := getPowerExpression(expr).UnaryExpression()
				Expect(unary.MINUS()).NotTo(BeNil())
				Expect(unary.UnaryExpression()).NotTo(BeNil())
			})

			It("Should parse logical NOT", func() {
				expr := mustParseExpression("not true")
				unary := getPowerExpression(expr).UnaryExpression()
				Expect(unary.NOT()).NotTo(BeNil())
			})

		})

		Context("Series", func() {
			It("Should parse series literals", func() {
				expr := mustParseExpression("[1, 2, 3]")
				literal := getPrimaryLiteral(expr)
				Expect(literal.SeriesLiteral()).NotTo(BeNil())
				series := literal.SeriesLiteral()
				Expect(series.LBRACKET()).NotTo(BeNil())
				Expect(series.RBRACKET()).NotTo(BeNil())
				Expect(series.ExpressionList()).NotTo(BeNil())
				Expect(series.ExpressionList().AllExpression()).To(HaveLen(3))
			})

			It("Should parse array indexing", func() {
				expr := mustParseExpression("data[0]")
				postfix := getPostfixExpression(expr)
				Expect(postfix.PrimaryExpression().IDENTIFIER().GetText()).To(Equal("data"))
				Expect(postfix.AllIndexOrSlice()).To(HaveLen(1))
				index := postfix.IndexOrSlice(0)
				Expect(index.LBRACKET()).NotTo(BeNil())
				Expect(index.RBRACKET()).NotTo(BeNil())
				Expect(index.AllExpression()).To(HaveLen(1))
			})

			It("Should parse array slicing", func() {
				expr := mustParseExpression("data[1:3]")
				postfix := getPostfixExpression(expr)
				index := postfix.IndexOrSlice(0)
				Expect(index.COLON()).NotTo(BeNil())
				Expect(index.AllExpression()).To(HaveLen(2))
			})
		})

		Context("Type Casting", func() {
			It("Should parse type casts", func() {
				expr := mustParseExpression("f32(42)")
				primary := getPrimaryExpression(expr)
				Expect(primary.TypeCast()).NotTo(BeNil())
				cast := primary.TypeCast()
				Expect(cast.Type_().PrimitiveType().NumericType().FloatType().F32()).NotTo(BeNil())
				Expect(cast.Expression()).NotTo(BeNil())
			})
		})
	})

	Describe("Functions", func() {
		It("Should parse basic function declaration", func() {
			prog := mustParseProgram(`
func add(x f64, y f64) f64 {
    return x + y
}`)

			Expect(prog.AllTopLevelItem()).To(HaveLen(1))
			funcDecl := prog.TopLevelItem(0).FunctionDeclaration()
			Expect(funcDecl).NotTo(BeNil())

			Expect(funcDecl.FUNC()).NotTo(BeNil())
			Expect(funcDecl.IDENTIFIER().GetText()).To(Equal("add"))

			params := funcDecl.InputList()
			Expect(params).NotTo(BeNil())
			Expect(params.AllInput()).To(HaveLen(2))

			Expect(params.Input(0).IDENTIFIER().GetText()).To(Equal("x"))
			Expect(params.Input(0).Type_().PrimitiveType().NumericType().FloatType().F64()).NotTo(BeNil())

			returnType := funcDecl.OutputType()
			Expect(returnType).NotTo(BeNil())
			Expect(returnType.Type_().PrimitiveType().NumericType().FloatType().F64()).NotTo(BeNil())

			block := funcDecl.Block()
			Expect(block).NotTo(BeNil())
			Expect(block.AllStatement()).To(HaveLen(1))

			returnStmt := block.Statement(0).ReturnStatement()
			Expect(returnStmt).NotTo(BeNil())
			Expect(returnStmt.RETURN()).NotTo(BeNil())
			Expect(returnStmt.Expression()).NotTo(BeNil())
		})

		It("Should parse function with channel parameters", func() {
			prog := mustParseProgram(`
func process(input chan f64, output chan f64) {
    value := input
    output = value
}`)

			funcDecl := prog.TopLevelItem(0).FunctionDeclaration()
			params := funcDecl.InputList()

			// First parameter: input chan f64
			param1 := params.Input(0)
			Expect(param1.IDENTIFIER().GetText()).To(Equal("input"))
			Expect(param1.Type_().ChannelType().CHAN()).NotTo(BeNil())
			Expect(param1.Type_().ChannelType().PrimitiveType().NumericType().FloatType().F64()).NotTo(BeNil())

			// Second parameter: output chan f64
			param2 := params.Input(1)
			Expect(param2.IDENTIFIER().GetText()).To(Equal("output"))
			Expect(param2.Type_().ChannelType().CHAN()).NotTo(BeNil())
		})
	})

	Describe("Tasks", func() {
		It("Should parse function with config block", func() {
			prog := mustParseProgram(`
func controller{
    setpoint f64
    sensor chan f64
    actuator chan f64
} (enable u8) {
    error := setpoint - sensor
    actuator = error
}`)

			taskDecl := prog.TopLevelItem(0).FunctionDeclaration()
			Expect(taskDecl).NotTo(BeNil())

			Expect(taskDecl.FUNC()).NotTo(BeNil())
			Expect(taskDecl.IDENTIFIER().GetText()).To(Equal("controller"))

			// Config block
			config := taskDecl.ConfigBlock()
			Expect(config).NotTo(BeNil())
			Expect(config.AllConfig()).To(HaveLen(3))

			// Runtime parameters
			params := taskDecl.InputList()
			Expect(params).NotTo(BeNil())
			Expect(params.AllInput()).To(HaveLen(1))
			Expect(params.Input(0).IDENTIFIER().GetText()).To(Equal("enable"))

			// Raw
			block := taskDecl.Block()
			Expect(block).NotTo(BeNil())
			Expect(block.AllStatement()).To(HaveLen(2))
		})

		It("Should parse function with return type", func() {
			prog := mustParseProgram(`
func doubler{
    input chan f64
} () f64 {
    return (input) * 2
}`)

			taskDecl := prog.TopLevelItem(0).FunctionDeclaration()

			returnType := taskDecl.OutputType()
			Expect(returnType).NotTo(BeNil())
			Expect(returnType.Type_().PrimitiveType().NumericType().FloatType().F64()).NotTo(BeNil())
		})
	})

	Describe("Inter-func Flow", func() {
		It("Should parse simple channel to funcflow", func() {
			prog := mustParseProgram(`sensor -> controller{} -> actuator`)

			flow := prog.TopLevelItem(0).FlowStatement()
			Expect(flow).NotTo(BeNil())

			// First node: sensor channel
			node1 := flow.FlowNode(0)
			Expect(node1.Identifier()).NotTo(BeNil())
			Expect(node1.Identifier().IDENTIFIER().GetText()).To(Equal("sensor"))

			// Second node: controller{}
			node2 := flow.FlowNode(1)
			Expect(node2.Function()).NotTo(BeNil())
			Expect(node2.Function().IDENTIFIER().GetText()).To(Equal("controller"))

			// Third node: actuator
			node3 := flow.FlowNode(2)
			Expect(node3.Identifier()).NotTo(BeNil())
			Expect(node3.Identifier().IDENTIFIER().GetText()).To(Equal("actuator"))
		})

		It("Should parse func invocation with named config", func() {
			prog := mustParseProgram(`
controller{
    setpoint=100,
    sensor=temp_sensor,
    period=100ms
} -> output`)

			flow := prog.TopLevelItem(0).FlowStatement()
			node := flow.FlowNode(0)
			invocation := node.Function()

			Expect(invocation.IDENTIFIER().GetText()).To(Equal("controller"))

			// Config values
			config := invocation.ConfigValues()
			Expect(config).NotTo(BeNil())
			Expect(config.NamedConfigValues()).NotTo(BeNil())
			Expect(config.NamedConfigValues().AllNamedConfigValue()).To(HaveLen(3))
		})

		It("Should parse func invocation with anonymous config", func() {
			prog := mustParseProgram(`any{ox_pt_1, ox_pt_2} -> average{} -> ox_pt_avg`)

			flow := prog.TopLevelItem(0).FlowStatement()
			node := flow.FlowNode(0)
			invocation := node.Function()

			Expect(invocation.IDENTIFIER().GetText()).To(Equal("any"))

			// Anonymous config values
			config := invocation.ConfigValues()
			Expect(config).NotTo(BeNil())
			Expect(config.AnonymousConfigValues()).NotTo(BeNil())
			Expect(config.AnonymousConfigValues().AllExpression()).To(HaveLen(2))

			// Check the second node also has func invocation
			node2 := flow.FlowNode(1)
			Expect(node2.Function()).NotTo(BeNil())
			Expect(node2.Function().IDENTIFIER().GetText()).To(Equal("average"))
		})

		It("Should parse func invocation with anonymous config in flow", func() {
			prog := mustParseProgram(`
func average {} (first chan f64, second chan f64) chan f64 {
    return (first + second) / 2
}

any{ox_pt_1, ox_pt_2} -> average{} -> ox_pt_avg`)

			// Check func declaration
			taskDecl := prog.TopLevelItem(0).FunctionDeclaration()
			Expect(taskDecl).NotTo(BeNil())

			// Check flow statement
			flow := prog.TopLevelItem(1).FlowStatement()
			node := flow.FlowNode(0)
			invocation := node.Function()

			// Verify anonymous config
			config := invocation.ConfigValues()
			Expect(config).NotTo(BeNil())
			Expect(config.AnonymousConfigValues()).NotTo(BeNil())

			exprs := config.AnonymousConfigValues().AllExpression()
			Expect(exprs).To(HaveLen(2))

			// First expression should be ox_pt_1
			expr1 := getPrimaryExpression(exprs[0])
			Expect(expr1.IDENTIFIER().GetText()).To(Equal("ox_pt_1"))

			// Second expression should be ox_pt_2
			expr2 := getPrimaryExpression(exprs[1])
			Expect(expr2.IDENTIFIER().GetText()).To(Equal("ox_pt_2"))
		})

		It("Should parse expression in flow", func() {
			prog := mustParseProgram(`ox_pt_1 > 100 -> alarm{}`)

			flow := prog.TopLevelItem(0).FlowStatement()
			node := flow.FlowNode(0)

			// First node is an expression
			Expect(node.Expression()).NotTo(BeNil())
			relational := getRelationalExpression(node.Expression())
			Expect(relational.GT(0)).NotTo(BeNil())
		})

		It("Should parse empty config in flow chains", func() {
			prog := mustParseProgram(`
func average {} (first chan f64, second chan f64) chan f64 {
    return (first + second) / 2
}

any{ox_pt_1, ox_pt_2} -> average{} -> ox_pt_avg`)

			// Check func declaration
			taskDecl := prog.TopLevelItem(0).FunctionDeclaration()
			Expect(taskDecl).NotTo(BeNil())

			// Check flow statement
			flow := prog.TopLevelItem(1).FlowStatement()

			// Check first func invocation (any)
			node1 := flow.FlowNode(0)
			Expect(node1.Function()).NotTo(BeNil())
			Expect(node1.Function().IDENTIFIER().GetText()).To(Equal("any"))

			// Check middle func invocation (average with empty config)
			node2 := flow.FlowNode(1)
			Expect(node2.Function()).NotTo(BeNil())
			Expect(node2.Function().IDENTIFIER().GetText()).To(Equal("average"))

			// Verify average has empty config
			avgConfig := node2.Function().ConfigValues()
			Expect(avgConfig).NotTo(BeNil())
			Expect(avgConfig.LBRACE()).NotTo(BeNil())
			Expect(avgConfig.RBRACE()).NotTo(BeNil())
			Expect(avgConfig.NamedConfigValues()).To(BeNil())
			Expect(avgConfig.AnonymousConfigValues()).To(BeNil())

			// Check final node (channel)
			node3 := flow.FlowNode(2)
			Expect(node3.Identifier()).NotTo(BeNil())
			Expect(node3.Identifier().IDENTIFIER().GetText()).To(Equal("ox_pt_avg"))
		})

		It("Should fail parsing mixed named and anonymous config values", func() {
			// Note: 'stage' is now a reserved keyword, so we use a different function name
			_, err := parser.Parse(`myfunc{ox_pt_1, second: ox_pt_2} -> output`)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("1:22 error: mismatched input")))
		})
	})

	Describe("Statements", func() {
		Context("Variable Declarations", func() {
			It("Should parse local variable declaration", func() {
				stmt := mustParseStatement("x := 42")

				varDecl := stmt.VariableDeclaration()
				Expect(varDecl).NotTo(BeNil())

				local := varDecl.LocalVariable()
				Expect(local).NotTo(BeNil())
				Expect(local.IDENTIFIER().GetText()).To(Equal("x"))
				Expect(local.DECLARE()).NotTo(BeNil())
				Expect(local.Expression()).NotTo(BeNil())
			})

			It("Should parse typed variable declaration", func() {
				stmt := mustParseStatement("voltage f32 := 3.3")

				local := stmt.VariableDeclaration().LocalVariable()
				Expect(local.IDENTIFIER().GetText()).To(Equal("voltage"))
				Expect(local.Type_()).NotTo(BeNil())
				Expect(local.Type_().PrimitiveType().NumericType().FloatType().F32()).NotTo(BeNil())
			})

			It("Should parse stateful variable declaration", func() {
				stmt := mustParseStatement("total $= 0")

				stateful := stmt.VariableDeclaration().StatefulVariable()
				Expect(stateful).NotTo(BeNil())
				Expect(stateful.IDENTIFIER().GetText()).To(Equal("total"))
				Expect(stateful.STATE_DECLARE()).NotTo(BeNil())
			})
		})

		Context("Variable Assignment", func() {
			It("Should parse assignment to existing variable", func() {
				stmt := mustParseStatement("x = 10")

				assignment := stmt.Assignment()
				Expect(assignment).NotTo(BeNil())
				Expect(assignment.IDENTIFIER().GetText()).To(Equal("x"))
				Expect(assignment.ASSIGN()).NotTo(BeNil())
				Expect(assignment.Expression()).NotTo(BeNil())
			})

			It("Should parse assignment with complex expression", func() {
				stmt := mustParseStatement("total = total + 1")

				assignment := stmt.Assignment()
				Expect(assignment).NotTo(BeNil())
				Expect(assignment.IDENTIFIER().GetText()).To(Equal("total"))
				Expect(assignment.ASSIGN()).NotTo(BeNil())

				// Check the expression is an addition
				expr := assignment.Expression()
				additive := getAdditiveExpression(expr)
				Expect(additive.PLUS(0)).NotTo(BeNil())
			})

			It("Should distinguish between declaration and assignment", func() {
				// Declaration with :=
				declStmt := mustParseStatement("x := 5")
				Expect(declStmt.VariableDeclaration()).NotTo(BeNil())
				Expect(declStmt.Assignment()).To(BeNil())

				// Assignment with =
				assignStmt := mustParseStatement("x = 10")
				Expect(assignStmt.Assignment()).NotTo(BeNil())
				Expect(assignStmt.VariableDeclaration()).To(BeNil())
			})

			It("Should distinguish between stateful declaration and assignment", func() {
				// Stateful declaration with $=
				declStmt := mustParseStatement("count $= 0")
				Expect(declStmt.VariableDeclaration()).NotTo(BeNil())
				Expect(declStmt.VariableDeclaration().StatefulVariable()).NotTo(BeNil())
				Expect(declStmt.Assignment()).To(BeNil())

				// Assignment to stateful variable with =
				assignStmt := mustParseStatement("count = count + 1")
				Expect(assignStmt.Assignment()).NotTo(BeNil())
				Expect(assignStmt.VariableDeclaration()).To(BeNil())
			})
		})

		Context("Control Flow", func() {
			It("Should parse if statement", func() {
				stmt := mustParseStatement(`if x > 10 {
    y := 20
}`)

				ifStmt := stmt.IfStatement()
				Expect(ifStmt).NotTo(BeNil())
				Expect(ifStmt.IF()).NotTo(BeNil())
				Expect(ifStmt.Expression()).NotTo(BeNil())
				Expect(ifStmt.Block()).NotTo(BeNil())
			})

			It("Should parse if-else-if-else chain", func() {
				stmt := mustParseStatement(`if x > 10 {
    y := 20
} else if x > 5 {
    y := 10
} else {
    y := 0
}`)

				ifStmt := stmt.IfStatement()
				Expect(ifStmt.AllElseIfClause()).To(HaveLen(1))
				Expect(ifStmt.ElseClause()).NotTo(BeNil())
			})
		})
	})

	Describe("Comprehensive Tests", func() {
		Context("Complex if-else chains", func() {
			It("Should parse multiple else-if chain", func() {
				stmt := mustParseStatement(`if x > 100 {
					high := true
				} else if x > 75 {
					medium_high := true
				} else if x > 50 {
					medium := true
				} else if x > 25 {
					medium_low := true
				} else if x > 0 {
					low := true
				} else {
					zero := true
				}`)

				ifStmt := stmt.IfStatement()
				Expect(ifStmt).NotTo(BeNil())

				// Check main if condition
				Expect(ifStmt.IF()).NotTo(BeNil())
				mainCond := ifStmt.Expression()
				Expect(mainCond).NotTo(BeNil())
				relational := getRelationalExpression(mainCond)
				Expect(relational.GT(0)).NotTo(BeNil())

				// Verify all else-if clauses
				elseIfClauses := ifStmt.AllElseIfClause()
				Expect(elseIfClauses).To(HaveLen(4)) // There are 4 else-if clauses (last one is just else)

				// Check first else-if: x > 75
				firstElseIf := elseIfClauses[0]
				Expect(firstElseIf.ELSE()).NotTo(BeNil())
				Expect(firstElseIf.IF()).NotTo(BeNil())
				cond1 := getRelationalExpression(firstElseIf.Expression())
				Expect(cond1.GT(0)).NotTo(BeNil())

				// Check each block has statements
				Expect(ifStmt.Block().AllStatement()).To(HaveLen(1))
				Expect(firstElseIf.Block().AllStatement()).To(HaveLen(1))

				// Verify else clause exists
				Expect(ifStmt.ElseClause()).NotTo(BeNil())
				Expect(ifStmt.ElseClause().ELSE()).NotTo(BeNil())
				Expect(ifStmt.ElseClause().Block().AllStatement()).To(HaveLen(1))
			})

			It("Should parse nested if statements", func() {
				stmt := mustParseStatement(`if x > 0 {
					if y > 0 {
						if z > 0 {
							positive := true
						} else {
							z_negative := true
						}
					} else {
						y_negative := true
					}
				} else {
					x_negative := true
				}`)

				// Outer if
				outerIf := stmt.IfStatement()
				Expect(outerIf).NotTo(BeNil())

				// First nested if (y > 0)
				outerBlock := outerIf.Block()
				Expect(outerBlock.AllStatement()).To(HaveLen(1))
				middleIf := outerBlock.Statement(0).IfStatement()
				Expect(middleIf).NotTo(BeNil())

				// Second nested if (z > 0)
				middleBlock := middleIf.Block()
				Expect(middleBlock.AllStatement()).To(HaveLen(1))
				innerIf := middleBlock.Statement(0).IfStatement()
				Expect(innerIf).NotTo(BeNil())

				// Verify all have else clauses
				Expect(outerIf.ElseClause()).NotTo(BeNil())
				Expect(middleIf.ElseClause()).NotTo(BeNil())
				Expect(innerIf.ElseClause()).NotTo(BeNil())
			})
		})

		Context("Complex operator precedence", func() {
			It("Should parse chained exponentials right-to-left", func() {
				// 2 ^ 3 ^ 2 ^ 1 should be 2 ^ (3 ^ (2 ^ 1))
				expr := mustParseExpression("2 ^ 3 ^ 2 ^ 1")

				power := getMultiplicativeExpression(expr).PowerExpression(0)
				Expect(power.CARET()).NotTo(BeNil())

				// Right side should be another power expression
				rightPower := power.PowerExpression()
				Expect(rightPower).NotTo(BeNil())
				Expect(rightPower.CARET()).NotTo(BeNil())

				// And that should have another power expression
				rightRightPower := rightPower.PowerExpression()
				Expect(rightRightPower).NotTo(BeNil())
				Expect(rightRightPower.CARET()).NotTo(BeNil())
			})

			It("Should parse complex logical expressions", func() {
				// not a and b or c and not d
				// Should be: ((not a) and b) or (c and (not d))
				expr := mustParseExpression("not a and b or c and not d")

				// Top level is OR
				logicalOr := expr.LogicalOrExpression()
				Expect(logicalOr.AllLogicalAndExpression()).To(HaveLen(2))
				Expect(logicalOr.OR(0)).NotTo(BeNil())

				// Left side: not a and b
				leftAnd := logicalOr.LogicalAndExpression(0)
				Expect(leftAnd.AllEqualityExpression()).To(HaveLen(2))
				Expect(leftAnd.AND(0)).NotTo(BeNil())

				// Right side: c and not d
				rightAnd := logicalOr.LogicalAndExpression(1)
				Expect(rightAnd.AllEqualityExpression()).To(HaveLen(2))
				Expect(rightAnd.AND(0)).NotTo(BeNil())
			})
		})

		Context("Complex series operations", func() {
			It("Should parse chained indexing and slicing", func() {
				expr := mustParseExpression("data[1:5][2]")

				postfix := getPostfixExpression(expr)
				Expect(postfix.AllIndexOrSlice()).To(HaveLen(2))

				// First operation: slice [1:5]
				slice := postfix.IndexOrSlice(0)
				Expect(slice.COLON()).NotTo(BeNil())
				Expect(slice.AllExpression()).To(HaveLen(2))

				// Second operation: index [2]
				index := postfix.IndexOrSlice(1)
				Expect(index.COLON()).To(BeNil())
				Expect(index.AllExpression()).To(HaveLen(1))
			})

			It("Should parse open-ended slices", func() {
				// data[:5]
				expr1 := mustParseExpression("data[:5]")
				slice1 := getPostfixExpression(expr1).IndexOrSlice(0)
				Expect(slice1.COLON()).NotTo(BeNil())
				Expect(slice1.AllExpression()).To(HaveLen(1)) // Only end expression

				// data[5:]
				expr2 := mustParseExpression("data[5:]")
				slice2 := getPostfixExpression(expr2).IndexOrSlice(0)
				Expect(slice2.COLON()).NotTo(BeNil())
				Expect(slice2.AllExpression()).To(HaveLen(1)) // Only start expression

				// data[:]
				expr3 := mustParseExpression("data[:]")
				slice3 := getPostfixExpression(expr3).IndexOrSlice(0)
				Expect(slice3.COLON()).NotTo(BeNil())
				Expect(slice3.AllExpression()).To(HaveLen(0)) // No expressions
			})
		})

		Context("Edge cases", func() {
			It("Should parse empty series literal", func() {
				expr := mustParseExpression("[]")
				literal := getPrimaryLiteral(expr)
				series := literal.SeriesLiteral()
				Expect(series).NotTo(BeNil())
				Expect(series.ExpressionList()).To(BeNil())
			})

			It("Should parse deeply nested expressions", func() {
				// Create a deeply nested expression
				expr := mustParseExpression("((((((1)))))))")

				// Navigate through all the parentheses
				primary := getPrimaryExpression(expr)
				Expect(primary.LPAREN()).NotTo(BeNil())
				Expect(primary.RPAREN()).NotTo(BeNil())

				inner1 := getPrimaryExpression(primary.Expression())
				Expect(inner1.LPAREN()).NotTo(BeNil())

				inner2 := getPrimaryExpression(inner1.Expression())
				Expect(inner2.LPAREN()).NotTo(BeNil())
			})
		})

		Context("Error cases", func() {
			It("Should report error for unclosed parenthesis", func() {
				_, err := parser.ParseExpression("(2 + 3")
				Expect(err).NotTo(BeNil())
				Expect(err).To(MatchError(ContainSubstring("missing ')'")))
			})

			It("Should report error for invalid operators", func() {
				_, err := parser.ParseExpression("2 ** 3")
				Expect(err).NotTo(BeNil())
			})

			It("Should capture lexer errors for invalid tokens (regression)", func() {
				// Regression test: lexer errors were not being captured properly
				// Using && instead of 'and' should produce lexer token recognition errors
				_, err := parser.ParseExpression("a > 5 && b < 10")
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("token recognition error at: '&'")))
			})

			It("Should report error for double assignment", func() {
				_, err := parser.Parse(`func test() {
					x := := 5
				}`)
				Expect(err).NotTo(BeNil())
				Expect(err).To(MatchError(ContainSubstring("error: extraneous input")))
			})

			It("Should report multiple errors with line information", func() {
				// Invalid syntax that should produce multiple parse errors
				_, err := parser.Parse(`
func broken() {
    x := := 5
    y = = 10
}`)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("3:9 error")))
				Expect(err).To(MatchError(ContainSubstring("4:8 error")))
			})

			It("Should handle empty input gracefully", func() {
				_, err := parser.Parse("")
				// Empty input should either succeed with empty program or fail gracefully
				if err != nil {
					Expect(err.Error()).NotTo(BeEmpty())
				}
			})

			It("Should handle whitespace-only input", func() {
				_, err := parser.Parse("   \n\t  \n  ")
				// Whitespace-only should either succeed or fail gracefully
				if err != nil {
					Expect(err.Error()).NotTo(BeEmpty())
				}
			})

			It("Should report error for unclosed brace", func() {
				_, err := parser.Parse(`func test() {
					x := 5
				`)
				Expect(err).NotTo(BeNil())
				Expect(err).To(MatchError(ContainSubstring("3:4 error: extraneous input")))
			})

			It("Should report error for missing function body", func() {
				_, err := parser.Parse(`func test()`)
				Expect(err).NotTo(BeNil())
			})
		})
	})

	Describe("Wrapper Functions", func() {
		Context("ParseExpression", func() {
			It("Should parse valid expression and return nil error", func() {
				expr, err := parser.ParseExpression("2 + 3")
				Expect(err).To(BeNil())
				Expect(expr).NotTo(BeNil())
			})

			It("Should return error for invalid expression", func() {
				_, err := parser.ParseExpression("2 + + 3")
				Expect(err).NotTo(BeNil())
			})

			It("Should handle empty expression", func() {
				_, err := parser.ParseExpression("")
				Expect(err).To(MatchError(ContainSubstring("mismatched input")))
			})
		})

		Context("ParseStatement", func() {
			It("Should parse valid statement and return nil error", func() {
				stmt, err := parser.ParseStatement("x := 42")
				Expect(err).To(BeNil())
				Expect(stmt).NotTo(BeNil())
			})

			It("Should return error for invalid statement", func() {
				_, err := parser.ParseStatement("x := := 5")
				Expect(err).NotTo(BeNil())
			})

			It("Should handle empty statement", func() {
				_, err := parser.ParseStatement("")
				// Empty statement should produce an error or handle gracefully
				if err != nil {
					Expect(err.Error()).NotTo(BeEmpty())
				}
			})
		})

		Context("ParseBlock", func() {
			It("Should parse valid block and return nil error", func() {
				block := MustSucceed(parser.ParseBlock("{ x := 42\n y := 10 }"))
				Expect(block).NotTo(BeNil())
			})

			It("Should return error for invalid block", func() {
				block, err := parser.ParseBlock("{ x := := 5 }")
				Expect(err).To(MatchError(ContainSubstring("1:7 error: extraneous input")))
				Expect(block).To(BeNil())
			})

			It("Should handle empty block", func() {
				block := MustSucceed(parser.ParseBlock("{}"))
				Expect(block).NotTo(BeNil())
			})

			It("Should handle block without braces", func() {
				block, err := parser.ParseBlock("x := 42")
				Expect(err).To(MatchError(ContainSubstring("missing '{'")))
				Expect(block).To(BeNil())
			})
		})

		Context("Parse (full program)", func() {
			It("Should parse valid program and return nil error", func() {
				prog := MustSucceed(parser.Parse(`func test() { x := 42 }`))
				Expect(prog).NotTo(BeNil())
			})

			It("Should return error for invalid program", func() {
				_, err := parser.Parse(`func test() { x := := 5 }`)
				Expect(err).NotTo(BeNil())
			})

			It("Should handle program with multiple top-level items", func() {
				prog, err := parser.Parse(`
func test1() { x := 1 }
func test2() { y := 2 }
sensor -> controller{}`)
				Expect(err).To(BeNil())
				Expect(prog).NotTo(BeNil())
				Expect(prog.AllTopLevelItem()).To(HaveLen(3))
			})
		})
	})

	Describe("Unicode and Special Characters", func() {
		Context("Unicode identifiers", func() {
			It("Should handle ASCII identifiers", func() {
				expr := mustParseExpression("sensor_1")
				primary := getPrimaryExpression(expr)
				Expect(primary.IDENTIFIER().GetText()).To(Equal("sensor_1"))
			})

			It("Should handle identifiers with underscores", func() {
				expr := mustParseExpression("temp_sensor_value")
				primary := getPrimaryExpression(expr)
				Expect(primary.IDENTIFIER().GetText()).To(Equal("temp_sensor_value"))
			})
		})

		Context("Comments handling", func() {
			It("Should parse code with comments", func() {
				prog := mustParseProgram(`
// This is a comment
func test() {
    x := 42  // inline comment
}`)
				Expect(prog).NotTo(BeNil())
				Expect(prog.AllTopLevelItem()).To(HaveLen(1))
			})
		})
	})

	Describe("Named Output Routing", func() {
		Context("Multi-Output func Declarations", func() {
			It("Should parse func with multiple named outputs", func() {
				prog := mustParseProgram(`
func demux{
    threshold f64
} (value f32) (high f32, low f32) {
    if (value > f32(threshold)) {
        high = value
    } else {
        low = value
    }
}`)

				stageDecl := prog.TopLevelItem(0).FunctionDeclaration()
				Expect(stageDecl).NotTo(BeNil())
				Expect(stageDecl.IDENTIFIER().GetText()).To(Equal("demux"))

				// Check multi-output block
				returnType := stageDecl.OutputType()
				Expect(returnType).NotTo(BeNil())

				multiOutput := returnType.MultiOutputBlock()
				Expect(multiOutput).NotTo(BeNil())
				Expect(multiOutput.LPAREN()).NotTo(BeNil())
				Expect(multiOutput.RPAREN()).NotTo(BeNil())

				// Check named outputs
				outputs := multiOutput.AllNamedOutput()
				Expect(outputs).To(HaveLen(2))

				// First output: high f32
				Expect(outputs[0].IDENTIFIER().GetText()).To(Equal("high"))
				Expect(outputs[0].Type_().PrimitiveType().NumericType().FloatType().F32()).NotTo(BeNil())

				// Second output: low f32
				Expect(outputs[1].IDENTIFIER().GetText()).To(Equal("low"))
				Expect(outputs[1].Type_().PrimitiveType().NumericType().FloatType().F32()).NotTo(BeNil())
			})

			It("Should parse func with three named outputs", func() {
				prog := mustParseProgram(`
func range_classifier{
    low f64
    high f64
} (value f32) (below_range f32, in_range f32, above_range f32) {
    // Logic
}`)

				stageDecl := prog.TopLevelItem(0).FunctionDeclaration()
				returnType := stageDecl.OutputType()
				multiOutput := returnType.MultiOutputBlock()

				outputs := multiOutput.AllNamedOutput()
				Expect(outputs).To(HaveLen(3))
				Expect(outputs[0].IDENTIFIER().GetText()).To(Equal("below_range"))
				Expect(outputs[1].IDENTIFIER().GetText()).To(Equal("in_range"))
				Expect(outputs[2].IDENTIFIER().GetText()).To(Equal("above_range"))
			})

			It("Should still parse stages with single return type", func() {
				prog := mustParseProgram(`
func simple{} (value f32) f32 {
    return value * 2.0
}`)

				stageDecl := prog.TopLevelItem(0).FunctionDeclaration()
				returnType := stageDecl.OutputType()
				Expect(returnType).NotTo(BeNil())

				// Should have Type, not MultiOutputBlock
				Expect(returnType.Type_()).NotTo(BeNil())
				Expect(returnType.MultiOutputBlock()).To(BeNil())
			})
		})

		Context("Routing Tables", func() {
			It("Should parse simple routing table", func() {
				prog := mustParseProgram(`
sensor -> demux{threshold=100} -> {
    high: alarm{},
    low: logger{}
}`)

				flow := prog.TopLevelItem(0).FlowStatement()
				Expect(flow).NotTo(BeNil())

				// Check routing table exists (should be one routing table)
				allRoutingTables := flow.AllRoutingTable()
				Expect(allRoutingTables).To(HaveLen(1))

				routingTable := allRoutingTables[0]
				Expect(routingTable).NotTo(BeNil())
				Expect(routingTable.LBRACE()).NotTo(BeNil())
				Expect(routingTable.RBRACE()).NotTo(BeNil())

				// Check routing entries
				entries := routingTable.AllRoutingEntry()
				Expect(entries).To(HaveLen(2))

				// First entry: high -> alarm{}
				Expect(entries[0].IDENTIFIER(0).GetText()).To(Equal("high"))
				Expect(entries[0].AllARROW()).To(HaveLen(0))
				highTargets := entries[0].AllFlowNode()
				Expect(highTargets).To(HaveLen(1))
				Expect(highTargets[0].Function()).NotTo(BeNil())
				Expect(highTargets[0].Function().IDENTIFIER().GetText()).To(Equal("alarm"))

				// Second entry: low -> logger{}
				Expect(entries[1].IDENTIFIER(0).GetText()).To(Equal("low"))
				lowTargets := entries[1].AllFlowNode()
				Expect(lowTargets).To(HaveLen(1))
				Expect(lowTargets[0].Function()).NotTo(BeNil())
				Expect(lowTargets[0].Function().IDENTIFIER().GetText()).To(Equal("logger"))
			})

			It("Should parse routing table with three outputs", func() {
				prog := mustParseProgram(`
sensor -> range_classifier{} -> {
    below_range: low_alarm{},
    in_range: controller{},
    above_range: high_alarm{}
}`)

				flow := prog.TopLevelItem(0).FlowStatement()
				allRoutingTables := flow.AllRoutingTable()
				Expect(allRoutingTables).To(HaveLen(1))

				routingTable := allRoutingTables[0]
				entries := routingTable.AllRoutingEntry()

				Expect(entries).To(HaveLen(3))
				Expect(entries[0].IDENTIFIER(0).GetText()).To(Equal("below_range"))
				Expect(entries[1].IDENTIFIER(0).GetText()).To(Equal("in_range"))
				Expect(entries[2].IDENTIFIER(0).GetText()).To(Equal("above_range"))
			})

			It("Should parse routing table to channels", func() {
				prog := mustParseProgram(`
processor -> splitter{} -> {
    output_a: channel_a,
    output_b: channel_b
}`)

				flow := prog.TopLevelItem(0).FlowStatement()
				allRoutingTables := flow.AllRoutingTable()
				Expect(allRoutingTables).To(HaveLen(1))

				routingTable := allRoutingTables[0]
				entries := routingTable.AllRoutingEntry()

				Expect(entries).To(HaveLen(2))

				// Target can be channel identifier
				targets := entries[0].AllFlowNode()
				Expect(targets).To(HaveLen(1))
				Expect(targets[0].Identifier()).NotTo(BeNil())
				Expect(targets[0].Identifier().IDENTIFIER().GetText()).To(Equal("channel_a"))
			})

			It("Should parse flow without routing table", func() {
				prog := mustParseProgram(`sensor -> controller{} -> actuator`)

				flow := prog.TopLevelItem(0).FlowStatement()
				Expect(flow).NotTo(BeNil())

				// Routing table should be optional (no routing tables)
				Expect(flow.AllRoutingTable()).To(HaveLen(0))
			})

			It("Should parse routing table with chained nodes", func() {
				prog := mustParseProgram(`
sensor -> state_router{} -> {
    idle_out: processor{} -> idle_display{},
    active_out: controller{} -> actuator
}`)

				flow := prog.TopLevelItem(0).FlowStatement()
				allRoutingTables := flow.AllRoutingTable()
				Expect(allRoutingTables).To(HaveLen(1))

				routingTable := allRoutingTables[0]
				entries := routingTable.AllRoutingEntry()

				Expect(entries).To(HaveLen(2))

				// First entry: idle_out: processor{} -> idle_display{}
				Expect(entries[0].IDENTIFIER(0).GetText()).To(Equal("idle_out"))
				entry0Nodes := entries[0].AllFlowNode()
				Expect(entry0Nodes).To(HaveLen(2))
				Expect(entries[0].AllARROW()).To(HaveLen(1))
				Expect(entry0Nodes[0].Function().IDENTIFIER().GetText()).To(Equal("processor"))
				Expect(entry0Nodes[1].Function().IDENTIFIER().GetText()).To(Equal("idle_display"))

				// Second entry: active_out: controller{} -> actuator
				Expect(entries[1].IDENTIFIER(0).GetText()).To(Equal("active_out"))
				entry1Nodes := entries[1].AllFlowNode()
				Expect(entry1Nodes).To(HaveLen(2))
				Expect(entries[1].AllARROW()).To(HaveLen(1))
				Expect(entry1Nodes[0].Function().IDENTIFIER().GetText()).To(Equal("controller"))
				Expect(entry1Nodes[1].Identifier().IDENTIFIER().GetText()).To(Equal("actuator"))
			})

			It("Should parse routing table with parameter mapping", func() {
				prog := mustParseProgram(`
first{} -> {
    outputA: processor{}: paramC,
    outputB: paramD
} -> second{}`)

				flow := prog.TopLevelItem(0).FlowStatement()
				allRoutingTables := flow.AllRoutingTable()
				Expect(allRoutingTables).To(HaveLen(1))

				routingTable := allRoutingTables[0]
				entries := routingTable.AllRoutingEntry()
				Expect(entries).To(HaveLen(2))

				// First entry: outputA: processor{}: paramC
				entry0 := entries[0]
				Expect(entry0.IDENTIFIER(0).GetText()).To(Equal("outputA"))
				entry0Nodes := entry0.AllFlowNode()
				Expect(entry0Nodes).To(HaveLen(1))
				Expect(entry0Nodes[0].Function().IDENTIFIER().GetText()).To(Equal("processor"))
				// Check trailing parameter name
				Expect(entry0.AllIDENTIFIER()).To(HaveLen(2))
				Expect(entry0.IDENTIFIER(1).GetText()).To(Equal("paramC"))

				// Second entry: outputB: paramD (no trailing parameter)
				entry1 := entries[1]
				Expect(entry1.IDENTIFIER(0).GetText()).To(Equal("outputB"))
				entry1Nodes := entry1.AllFlowNode()
				Expect(entry1Nodes).To(HaveLen(1))
				Expect(entry1Nodes[0].Identifier().IDENTIFIER().GetText()).To(Equal("paramD"))
				// No trailing parameter
				Expect(entry1.AllIDENTIFIER()).To(HaveLen(1))
			})

			It("Should parse routing table with chained processing and parameter mapping", func() {
				prog := mustParseProgram(`
stage1{} -> {
    out1: filter{} -> amplifier{}: input,
    out2: processor{} -> converter{}: value
} -> stage2{}`)

				flow := prog.TopLevelItem(0).FlowStatement()
				routingTable := flow.AllRoutingTable()[0]
				entries := routingTable.AllRoutingEntry()
				Expect(entries).To(HaveLen(2))

				// First entry: out1: filter{} -> amplifier{}: input
				entry0 := entries[0]
				Expect(entry0.IDENTIFIER(0).GetText()).To(Equal("out1"))
				Expect(entry0.AllFlowNode()).To(HaveLen(2))
				Expect(entry0.AllARROW()).To(HaveLen(1))
				Expect(entry0.AllIDENTIFIER()).To(HaveLen(2))
				Expect(entry0.IDENTIFIER(1).GetText()).To(Equal("input"))

				// Second entry: out2: processor{} -> converter{}: value
				entry1 := entries[1]
				Expect(entry1.IDENTIFIER(0).GetText()).To(Equal("out2"))
				Expect(entry1.AllFlowNode()).To(HaveLen(2))
				Expect(entry1.AllARROW()).To(HaveLen(1))
				Expect(entry1.AllIDENTIFIER()).To(HaveLen(2))
				Expect(entry1.IDENTIFIER(1).GetText()).To(Equal("value"))
			})
		})

		Context("Combined Multi-Output and Routing", func() {
			It("Should parse complete example with multi-output func and routing", func() {
				prog := mustParseProgram(`
func demux{
    threshold f64
} (value f32) (high f32, low f32) {
    if (value > f32(threshold)) {
        high = value
    } else {
        low = value
    }
}

sensor -> demux{threshold=100.0} -> {
    high: alarm{},
    low: logger{}
}`)

				// Check func declaration
				stageDecl := prog.TopLevelItem(0).FunctionDeclaration()
				Expect(stageDecl).NotTo(BeNil())

				multiOutput := stageDecl.OutputType().MultiOutputBlock()
				Expect(multiOutput.AllNamedOutput()).To(HaveLen(2))

				// Check flow statement
				flow := prog.TopLevelItem(1).FlowStatement()
				Expect(flow).NotTo(BeNil())

				allRoutingTables := flow.AllRoutingTable()
				Expect(allRoutingTables).To(HaveLen(1))
				Expect(allRoutingTables[0].AllRoutingEntry()).To(HaveLen(2))
			})
		})

		Context("Input Routing Tables", func() {
			It("Should parse simple input routing table", func() {
				prog := mustParseProgram(`
{
    sensor1: a,
    sensor2: b
} -> add{}`)

				flow := prog.TopLevelItem(0).FlowStatement()
				Expect(flow).NotTo(BeNil())

				// Check routing table exists at start
				allRoutingTables := flow.AllRoutingTable()
				Expect(allRoutingTables).To(HaveLen(1))

				routingTable := allRoutingTables[0]
				entries := routingTable.AllRoutingEntry()
				Expect(entries).To(HaveLen(2))

				// First entry: sensor1 -> a
				Expect(entries[0].IDENTIFIER(0).GetText()).To(Equal("sensor1"))
				entry0Targets := entries[0].AllFlowNode()
				Expect(entry0Targets).To(HaveLen(1))
				Expect(entry0Targets[0].Identifier()).NotTo(BeNil())
				Expect(entry0Targets[0].Identifier().IDENTIFIER().GetText()).To(Equal("a"))

				// Second entry: sensor2 -> b
				Expect(entries[1].IDENTIFIER(0).GetText()).To(Equal("sensor2"))
				entry1Targets := entries[1].AllFlowNode()
				Expect(entry1Targets).To(HaveLen(1))
				Expect(entry1Targets[0].Identifier()).NotTo(BeNil())
				Expect(entry1Targets[0].Identifier().IDENTIFIER().GetText()).To(Equal("b"))
			})

			It("Should parse input routing with flow chains", func() {
				prog := mustParseProgram(`
{
    sensor1: lowpass{cutoff=0.5} -> a,
    sensor2: scale{factor=2.0} -> b
} -> add{}`)

				flow := prog.TopLevelItem(0).FlowStatement()
				allRoutingTables := flow.AllRoutingTable()
				Expect(allRoutingTables).To(HaveLen(1))

				routingTable := allRoutingTables[0]
				entries := routingTable.AllRoutingEntry()
				Expect(entries).To(HaveLen(2))

				// First entry: sensor1 -> lowpass{cutoff=0.5} -> a
				entry0 := entries[0]
				Expect(entry0.IDENTIFIER(0).GetText()).To(Equal("sensor1"))
				Expect(entry0.AllARROW()).To(HaveLen(1)) // sensor1 -> lowpass{} -> a
				entry0Nodes := entry0.AllFlowNode()
				Expect(entry0Nodes).To(HaveLen(2)) // lowpass{}, a
				Expect(entry0Nodes[0].Function().IDENTIFIER().GetText()).To(Equal("lowpass"))
				Expect(entry0Nodes[1].Identifier().IDENTIFIER().GetText()).To(Equal("a"))

				// Second entry: sensor2 -> scale{factor=2.0} -> b
				entry1 := entries[1]
				Expect(entry1.IDENTIFIER(0).GetText()).To(Equal("sensor2"))
				Expect(entry1.AllARROW()).To(HaveLen(1))
				entry1Nodes := entry1.AllFlowNode()
				Expect(entry1Nodes).To(HaveLen(2))
				Expect(entry1Nodes[0].Function().IDENTIFIER().GetText()).To(Equal("scale"))
				Expect(entry1Nodes[1].Identifier().IDENTIFIER().GetText()).To(Equal("b"))
			})
		})
	})

	Describe("Sequences and Stages", func() {
		Context("Sequence Declarations", func() {
			It("Should parse a simple sequence with stages", func() {
				prog := mustParseProgram(`
sequence main {
    stage precheck { }
    stage pressurization { }
}`)
				seq := prog.TopLevelItem(0).SequenceDeclaration()
				Expect(seq).NotTo(BeNil())
				Expect(seq.IDENTIFIER().GetText()).To(Equal("main"))

				stages := seq.AllStageDeclaration()
				Expect(stages).To(HaveLen(2))

				// First stage: precheck
				Expect(stages[0].IDENTIFIER().GetText()).To(Equal("precheck"))

				// Second stage: pressurization
				Expect(stages[1].IDENTIFIER().GetText()).To(Equal("pressurization"))
			})
		})

		Context("Stage Items", func() {
			It("Should parse stage with multiple transitions", func() {
				prog := mustParseProgram(`
sequence seq {
    stage hold {
        condition1 => next,
        condition2 => next
    }
}`)
				seq := prog.TopLevelItem(0).SequenceDeclaration()
				stages := seq.AllStageDeclaration()
				Expect(stages).To(HaveLen(1))

				stage := stages[0]
				Expect(stage.IDENTIFIER().GetText()).To(Equal("hold"))
				Expect(stage.StageBody().AllStageItem()).To(HaveLen(2))
			})

			It("Should parse stage with single transition", func() {
				prog := mustParseProgram(`
sequence seq {
    stage test {
        condition => next
    }
}`)
				seq := prog.TopLevelItem(0).SequenceDeclaration()
				stages := seq.AllStageDeclaration()
				stage := stages[0]
				body := stage.StageBody()
				items := body.AllStageItem()
				Expect(items).To(HaveLen(1))

				// Check flow statement with transition operator
				flow := items[0].FlowStatement()
				Expect(flow).NotTo(BeNil())
				// The transition operator (=>) is used in the flow
				Expect(flow.AllFlowOperator()).To(HaveLen(1))
			})
		})

		Context("Transitions", func() {
			It("Should parse transitions in stage items", func() {
				prog := mustParseProgram(`
sequence seq {
    stage test {
        condition1 => next,
        condition2 => target,
        pressure > target => next
    }
}`)
				seq := prog.TopLevelItem(0).SequenceDeclaration()
				Expect(seq).NotTo(BeNil())

				stages := seq.AllStageDeclaration()
				Expect(stages).To(HaveLen(1))

				stage := stages[0]
				Expect(stage.IDENTIFIER().GetText()).To(Equal("test"))

				items := stage.StageBody().AllStageItem()
				Expect(items).To(HaveLen(3))
			})
		})

		Context("Complex Sequences", func() {
			It("Should parse sequence with multiple stages", func() {
				prog := mustParseProgram(`
sequence main {
    stage precheck {
        sensor_ok => next,
        abort_btn => abort
    }
    stage pressurize {
        pressure > target => next,
        timeout => abort
    }
    stage complete { }
}`)
				// Verify sequence
				seq := prog.TopLevelItem(0).SequenceDeclaration()
				Expect(seq.IDENTIFIER().GetText()).To(Equal("main"))
				stages := seq.AllStageDeclaration()
				Expect(stages).To(HaveLen(3))

				// Verify stage names
				Expect(stages[0].IDENTIFIER().GetText()).To(Equal("precheck"))
				Expect(stages[1].IDENTIFIER().GetText()).To(Equal("pressurize"))
				Expect(stages[2].IDENTIFIER().GetText()).To(Equal("complete"))
			})
		})
	})
})

func mustParseProgram(code string) parser.IProgramContext {
	return MustSucceed(parser.Parse(code))
}

func mustParseStatement(code string) parser.IStatementContext {
	return MustSucceed(parser.ParseStatement(code))
}

func mustParseExpression(expr string) parser.IExpressionContext {
	return MustSucceed(parser.ParseExpression(expr))
}

// AST Navigation Helpers - traverse the parse tree to access specific nodes
func getPrimaryLiteral(expr parser.IExpressionContext) parser.ILiteralContext {
	primary := getPrimaryExpression(expr)
	return primary.Literal()
}

func getPrimaryExpression(expr parser.IExpressionContext) parser.IPrimaryExpressionContext {
	return getPostfixExpression(expr).PrimaryExpression()
}

func getPostfixExpression(expr parser.IExpressionContext) parser.IPostfixExpressionContext {
	return getPowerExpression(expr).UnaryExpression().PostfixExpression()
}

func getPowerExpression(expr parser.IExpressionContext) parser.IPowerExpressionContext {
	return getMultiplicativeExpression(expr).PowerExpression(0)
}

func getMultiplicativeExpression(expr parser.IExpressionContext) parser.IMultiplicativeExpressionContext {
	return getAdditiveExpression(expr).MultiplicativeExpression(0)
}

func getAdditiveExpression(expr parser.IExpressionContext) parser.IAdditiveExpressionContext {
	return getRelationalExpression(expr).AdditiveExpression(0)
}

func getRelationalExpression(expr parser.IExpressionContext) parser.IRelationalExpressionContext {
	return getEqualityExpression(expr).RelationalExpression(0)
}

func getEqualityExpression(expr parser.IExpressionContext) parser.IEqualityExpressionContext {
	return getLogicalAndExpression(expr).EqualityExpression(0)
}

func getLogicalAndExpression(expr parser.IExpressionContext) parser.ILogicalAndExpressionContext {
	return expr.LogicalOrExpression().LogicalAndExpression(0)
}
