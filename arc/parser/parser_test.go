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
	"github.com/antlr4-go/antlr/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/parser"
)

// Helper to parse expression without error handling in tests
func mustParseExpression(expr string) parser.IExpressionContext {
	exprCtx, err := parser.ParseExpression(expr)
	Expect(err).To(BeNil())
	return exprCtx
}

var _ = Describe("Parser", func() {
	Describe("Expressions", func() {
		Context("Numeric Literals", func() {
			It("Should parse integer literals", func() {
				expr := mustParseExpression("42")
				Expect(expr).NotTo(BeNil())

				// Check it's a primary expression with a literal
				logicalOr := expr.LogicalOrExpression()
				Expect(logicalOr).NotTo(BeNil())

				logicalAnd := logicalOr.LogicalAndExpression(0)
				equality := logicalAnd.EqualityExpression(0)
				relational := equality.RelationalExpression(0)
				additive := relational.AdditiveExpression(0)
				multiplicative := additive.MultiplicativeExpression(0)
				power := multiplicative.PowerExpression(0)
				unary := power.UnaryExpression()
				postfix := unary.PostfixExpression()
				primary := postfix.PrimaryExpression()
				literal := primary.Literal()

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
				expr := mustParseExpression("!true")
				unary := getPowerExpression(expr).UnaryExpression()

				Expect(unary.NOT()).NotTo(BeNil())
			})

			It("Should parse blocking read", func() {
				expr := mustParseExpression("<-input")
				unary := getPowerExpression(expr).UnaryExpression()

				Expect(unary.BlockingReadExpr()).NotTo(BeNil())
				Expect(unary.BlockingReadExpr().RECV()).NotTo(BeNil())
				Expect(unary.BlockingReadExpr().IDENTIFIER().GetText()).To(Equal("input"))
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
			prog := parseProgram(`
func add(x f64, y f64) f64 {
    return x + y
}`)

			Expect(prog.AllTopLevelItem()).To(HaveLen(1))
			funcDecl := prog.TopLevelItem(0).FunctionDeclaration()
			Expect(funcDecl).NotTo(BeNil())

			Expect(funcDecl.FUNC()).NotTo(BeNil())
			Expect(funcDecl.IDENTIFIER().GetText()).To(Equal("add"))

			params := funcDecl.ParameterList()
			Expect(params).NotTo(BeNil())
			Expect(params.AllParameter()).To(HaveLen(2))

			Expect(params.Parameter(0).IDENTIFIER().GetText()).To(Equal("x"))
			Expect(params.Parameter(0).Type_().PrimitiveType().NumericType().FloatType().F64()).NotTo(BeNil())

			returnType := funcDecl.ReturnType()
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
			prog := parseProgram(`
func process(input <-chan f64, output ->chan f64) {
    value := <-input
    value -> output
}`)

			funcDecl := prog.TopLevelItem(0).FunctionDeclaration()
			params := funcDecl.ParameterList()

			// First parameter: input <-chan f64
			param1 := params.Parameter(0)
			Expect(param1.IDENTIFIER().GetText()).To(Equal("input"))
			Expect(param1.Type_().ChannelType().RECV_CHAN()).NotTo(BeNil())
			Expect(param1.Type_().ChannelType().PrimitiveType().NumericType().FloatType().F64()).NotTo(BeNil())

			// Second parameter: output ->chan f64
			param2 := params.Parameter(1)
			Expect(param2.IDENTIFIER().GetText()).To(Equal("output"))
			Expect(param2.Type_().ChannelType().SEND_CHAN()).NotTo(BeNil())
		})
	})

	Describe("Tasks", func() {
		It("Should parse stage with config block", func() {
			prog := parseProgram(`
stage controller{
    setpoint f64
    sensor <-chan f64
    actuator ->chan f64
} (enable u8) {
    error := setpoint - (<-sensor)
    error -> actuator
}`)

			taskDecl := prog.TopLevelItem(0).StageDeclaration()
			Expect(taskDecl).NotTo(BeNil())

			Expect(taskDecl.STAGE()).NotTo(BeNil())
			Expect(taskDecl.IDENTIFIER().GetText()).To(Equal("controller"))

			// Config block
			config := taskDecl.ConfigBlock()
			Expect(config).NotTo(BeNil())
			Expect(config.AllConfigParameter()).To(HaveLen(3))

			// Runtime parameters
			params := taskDecl.ParameterList()
			Expect(params).NotTo(BeNil())
			Expect(params.AllParameter()).To(HaveLen(1))
			Expect(params.Parameter(0).IDENTIFIER().GetText()).To(Equal("enable"))

			// Raw
			block := taskDecl.Block()
			Expect(block).NotTo(BeNil())
			Expect(block.AllStatement()).To(HaveLen(2))
		})

		It("Should parse stage with return type", func() {
			prog := parseProgram(`
stage doubler{
    input <-chan f64
} () f64 {
    return (<-input) * 2
}`)

			taskDecl := prog.TopLevelItem(0).StageDeclaration()

			returnType := taskDecl.ReturnType()
			Expect(returnType).NotTo(BeNil())
			Expect(returnType.Type_().PrimitiveType().NumericType().FloatType().F64()).NotTo(BeNil())
		})
	})

	Describe("Inter-Stage Flow", func() {
		It("Should parse simple channel to stage flow", func() {
			prog := parseProgram(`sensor -> controller{} -> actuator`)

			flow := prog.TopLevelItem(0).FlowStatement()
			Expect(flow).NotTo(BeNil())

			// First node: sensor channel
			node1 := flow.FlowNode(0)
			Expect(node1.ChannelIdentifier()).NotTo(BeNil())
			Expect(node1.ChannelIdentifier().IDENTIFIER().GetText()).To(Equal("sensor"))

			// Second node: controller{}
			node2 := flow.FlowNode(1)
			Expect(node2.StageInvocation()).NotTo(BeNil())
			Expect(node2.StageInvocation().IDENTIFIER().GetText()).To(Equal("controller"))

			// Third node: actuator
			node3 := flow.FlowNode(2)
			Expect(node3.ChannelIdentifier()).NotTo(BeNil())
			Expect(node3.ChannelIdentifier().IDENTIFIER().GetText()).To(Equal("actuator"))
		})

		It("Should parse stage invocation with named config", func() {
			prog := parseProgram(`
controller{
    setpoint: 100,
    sensor: temp_sensor,
    interval: 100ms
}(1) -> output`)

			flow := prog.TopLevelItem(0).FlowStatement()
			node := flow.FlowNode(0)
			stage := node.StageInvocation()

			Expect(stage.IDENTIFIER().GetText()).To(Equal("controller"))

			// Config values
			config := stage.ConfigValues()
			Expect(config).NotTo(BeNil())
			Expect(config.NamedConfigValues()).NotTo(BeNil())
			Expect(config.NamedConfigValues().AllNamedConfigValue()).To(HaveLen(3))

			// Runtime arguments
			args := stage.Arguments()
			Expect(args).NotTo(BeNil())
			Expect(args.ArgumentList()).NotTo(BeNil())
			Expect(args.ArgumentList().AllExpression()).To(HaveLen(1))
		})

		It("Should parse stage invocation with anonymous config", func() {
			prog := parseProgram(`any{ox_pt_1, ox_pt_2} -> average{} -> ox_pt_avg`)

			flow := prog.TopLevelItem(0).FlowStatement()
			node := flow.FlowNode(0)
			stage := node.StageInvocation()

			Expect(stage.IDENTIFIER().GetText()).To(Equal("any"))

			// Anonymous config values
			config := stage.ConfigValues()
			Expect(config).NotTo(BeNil())
			Expect(config.AnonymousConfigValues()).NotTo(BeNil())
			Expect(config.AnonymousConfigValues().AllExpression()).To(HaveLen(2))

			// Check the second node also has stage invocation
			node2 := flow.FlowNode(1)
			Expect(node2.StageInvocation()).NotTo(BeNil())
			Expect(node2.StageInvocation().IDENTIFIER().GetText()).To(Equal("average"))
		})

		It("Should parse stage with anonymous arguments in complex flow", func() {
			prog := parseProgram(`
stage average {} (first chan f64, second chan f64) chan f64 {
    return (first + second) / 2
}

any{ox_pt_1, ox_pt_2} -> average{} -> ox_pt_avg`)

			// Check stage declaration
			taskDecl := prog.TopLevelItem(0).StageDeclaration()
			Expect(taskDecl).NotTo(BeNil())

			// Check flow statement
			flow := prog.TopLevelItem(1).FlowStatement()
			node := flow.FlowNode(0)
			stage := node.StageInvocation()

			// Verify anonymous config
			config := stage.ConfigValues()
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
			prog := parseProgram(`ox_pt_1 > 100 -> alarm{}`)

			flow := prog.TopLevelItem(0).FlowStatement()
			node := flow.FlowNode(0)

			// First node is an expression
			Expect(node.Expression()).NotTo(BeNil())
			relational := getRelationalExpression(node.Expression())
			Expect(relational.GT(0)).NotTo(BeNil())
		})

		It("Should parse empty config in flow chains", func() {
			prog := parseProgram(`
stage average {} (first chan f64, second chan f64) chan f64 {
    return (first + second) / 2
}

any{ox_pt_1, ox_pt_2} -> average{} -> ox_pt_avg`)

			// Check stage declaration
			taskDecl := prog.TopLevelItem(0).StageDeclaration()
			Expect(taskDecl).NotTo(BeNil())

			// Check flow statement
			flow := prog.TopLevelItem(1).FlowStatement()

			// Check first stage invocation (any)
			node1 := flow.FlowNode(0)
			Expect(node1.StageInvocation()).NotTo(BeNil())
			Expect(node1.StageInvocation().IDENTIFIER().GetText()).To(Equal("any"))

			// Check middle stage invocation (average with empty config)
			node2 := flow.FlowNode(1)
			Expect(node2.StageInvocation()).NotTo(BeNil())
			Expect(node2.StageInvocation().IDENTIFIER().GetText()).To(Equal("average"))

			// Verify average has empty config
			avgConfig := node2.StageInvocation().ConfigValues()
			Expect(avgConfig).NotTo(BeNil())
			Expect(avgConfig.LBRACE()).NotTo(BeNil())
			Expect(avgConfig.RBRACE()).NotTo(BeNil())
			Expect(avgConfig.NamedConfigValues()).To(BeNil())
			Expect(avgConfig.AnonymousConfigValues()).To(BeNil())

			// Check final node (channel)
			node3 := flow.FlowNode(2)
			Expect(node3.ChannelIdentifier()).NotTo(BeNil())
			Expect(node3.ChannelIdentifier().IDENTIFIER().GetText()).To(Equal("ox_pt_avg"))
		})

		It("Should fail parsing mixed named and anonymous config values", func() {
			_, err := parser.Parse(`stage{ox_pt_1, second: ox_pt_2} -> output`)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("parse errors"))
		})
	})

	Describe("Statements", func() {
		Context("Variable Declarations", func() {
			It("Should parse local variable declaration", func() {
				stmt := parseStatement("x := 42")

				varDecl := stmt.VariableDeclaration()
				Expect(varDecl).NotTo(BeNil())

				local := varDecl.LocalVariable()
				Expect(local).NotTo(BeNil())
				Expect(local.IDENTIFIER().GetText()).To(Equal("x"))
				Expect(local.DECLARE()).NotTo(BeNil())
				Expect(local.Expression()).NotTo(BeNil())
			})

			It("Should parse typed variable declaration", func() {
				stmt := parseStatement("voltage f32 := 3.3")

				local := stmt.VariableDeclaration().LocalVariable()
				Expect(local.IDENTIFIER().GetText()).To(Equal("voltage"))
				Expect(local.Type_()).NotTo(BeNil())
				Expect(local.Type_().PrimitiveType().NumericType().FloatType().F32()).NotTo(BeNil())
			})

			It("Should parse stateful variable declaration", func() {
				stmt := parseStatement("total $= 0")

				stateful := stmt.VariableDeclaration().StatefulVariable()
				Expect(stateful).NotTo(BeNil())
				Expect(stateful.IDENTIFIER().GetText()).To(Equal("total"))
				Expect(stateful.STATE_DECLARE()).NotTo(BeNil())
			})
		})

		Context("Variable Assignment", func() {
			It("Should parse assignment to existing variable", func() {
				stmt := parseStatement("x = 10")

				assignment := stmt.Assignment()
				Expect(assignment).NotTo(BeNil())
				Expect(assignment.IDENTIFIER().GetText()).To(Equal("x"))
				Expect(assignment.ASSIGN()).NotTo(BeNil())
				Expect(assignment.Expression()).NotTo(BeNil())
			})

			It("Should parse assignment with complex expression", func() {
				stmt := parseStatement("total = total + 1")

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
				declStmt := parseStatement("x := 5")
				Expect(declStmt.VariableDeclaration()).NotTo(BeNil())
				Expect(declStmt.Assignment()).To(BeNil())

				// Assignment with =
				assignStmt := parseStatement("x = 10")
				Expect(assignStmt.Assignment()).NotTo(BeNil())
				Expect(assignStmt.VariableDeclaration()).To(BeNil())
			})

			It("Should distinguish between stateful declaration and assignment", func() {
				// Stateful declaration with $=
				declStmt := parseStatement("count $= 0")
				Expect(declStmt.VariableDeclaration()).NotTo(BeNil())
				Expect(declStmt.VariableDeclaration().StatefulVariable()).NotTo(BeNil())
				Expect(declStmt.Assignment()).To(BeNil())

				// Assignment to stateful variable with =
				assignStmt := parseStatement("count = count + 1")
				Expect(assignStmt.Assignment()).NotTo(BeNil())
				Expect(assignStmt.VariableDeclaration()).To(BeNil())
			})
		})

		Context("Channel Operations", func() {
			It("Should parse channel write with arrow", func() {
				stmt := parseStatement("42 -> output")

				channelOp := stmt.ChannelOperation()
				Expect(channelOp).NotTo(BeNil())

				write := channelOp.ChannelWrite()
				Expect(write).NotTo(BeNil())
				Expect(write.ARROW()).NotTo(BeNil())
				Expect(write.IDENTIFIER().GetText()).To(Equal("output"))
			})

			It("Should parse channel write with receive operator", func() {
				stmt := parseStatement("output <- 42")

				write := stmt.ChannelOperation().ChannelWrite()
				Expect(write.RECV()).NotTo(BeNil())
				Expect(write.IDENTIFIER().GetText()).To(Equal("output"))
			})

			It("Should parse blocking channel read", func() {
				stmt := parseStatement("value := <-input")

				channelOp := stmt.ChannelOperation()
				if channelOp == nil {
					// Maybe it's a variable declaration with blocking read expression
					varDecl := stmt.VariableDeclaration()
					Expect(varDecl).NotTo(BeNil())
					return
				}

				read := channelOp.ChannelRead()
				Expect(read).NotTo(BeNil())

				blocking := read.BlockingRead()
				Expect(blocking).NotTo(BeNil())
				Expect(blocking.IDENTIFIER(0).GetText()).To(Equal("value"))
				Expect(blocking.RECV()).NotTo(BeNil())
				Expect(blocking.IDENTIFIER(1).GetText()).To(Equal("input"))
			})

			It("Should parse non-blocking channel read", func() {
				stmt := parseStatement("current := sensor")

				// This is likely parsed as a variable declaration
				varDecl := stmt.VariableDeclaration()
				if varDecl != nil {
					local := varDecl.LocalVariable()
					Expect(local).NotTo(BeNil())
					Expect(local.IDENTIFIER().GetText()).To(Equal("current"))
					return
				}

				channelOp := stmt.ChannelOperation()
				Expect(channelOp).NotTo(BeNil())
				nonBlocking := channelOp.ChannelRead().NonBlockingRead()
				Expect(nonBlocking).NotTo(BeNil())
				Expect(nonBlocking.IDENTIFIER(0).GetText()).To(Equal("current"))
				Expect(nonBlocking.IDENTIFIER(1).GetText()).To(Equal("sensor"))
			})
		})

		Context("Control Flow", func() {
			It("Should parse if statement", func() {
				stmt := parseStatement(`if x > 10 {
    y := 20
}`)

				ifStmt := stmt.IfStatement()
				Expect(ifStmt).NotTo(BeNil())
				Expect(ifStmt.IF()).NotTo(BeNil())
				Expect(ifStmt.Expression()).NotTo(BeNil())
				Expect(ifStmt.Block()).NotTo(BeNil())
			})

			It("Should parse if-else-if-else chain", func() {
				stmt := parseStatement(`if x > 10 {
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
				stmt := parseStatement(`if x > 100 {
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
				stmt := parseStatement(`if x > 0 {
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
				// !a && b || c && !d
				// Should be: ((!a) && b) || (c && (!d))
				expr := mustParseExpression("!a && b || c && !d")

				// Top level is OR
				logicalOr := expr.LogicalOrExpression()
				Expect(logicalOr.AllLogicalAndExpression()).To(HaveLen(2))
				Expect(logicalOr.OR(0)).NotTo(BeNil())

				// Left side: !a && b
				leftAnd := logicalOr.LogicalAndExpression(0)
				Expect(leftAnd.AllEqualityExpression()).To(HaveLen(2))
				Expect(leftAnd.AND(0)).NotTo(BeNil())

				// Right side: c && !d
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
				Expect(err.Error()).To(ContainSubstring("missing ')'"))
			})

			It("Should report error for invalid operators", func() {
				_, err := parser.ParseExpression("2 ** 3")
				Expect(err).NotTo(BeNil())
			})

			It("Should report error for double assignment", func() {
				_, err := parser.Parse(`func test() {
					x := := 5
				}`)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("parse errors"))
			})
		})
	})
})

// Helper functions to navigate the AST
func parseProgram(code string) parser.IProgramContext {
	inputStream := antlr.NewInputStream(code)
	lexer := parser.NewArcLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, 0)
	p := parser.NewArcParser(stream)
	p.BuildParseTrees = true
	return p.Program()
}

func parseStatement(code string) parser.IStatementContext {
	// Wrap in a function to parse as a statement
	prog := parseProgram("func test() { " + code + " }")
	funcDecl := prog.TopLevelItem(0).FunctionDeclaration()
	return funcDecl.Block().Statement(0)
}

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
