package parser_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/slate/parser"
)

var _ = Describe("Parser", func() {
	Describe("Expressions", func() {
		DescribeTable("Binary operators",
			func(expression string, expectedOperator string, leftOperand string, rightOperand string) {
				expr, err := parser.ParseExpression(expression)
				Expect(err).To(BeNil())
				Expect(expr).NotTo(BeNil())
				Expect(expr.GetText()).To(Equal(expression))

				// Navigate through the expression hierarchy to get to the actual operator level
				logicalOr := expr.LogicalOrExpr()
				Expect(logicalOr).NotTo(BeNil())

				// For logical operators
				if expectedOperator == "||" {
					andExprs := logicalOr.AllLogicalAndExpr()
					Expect(andExprs).To(HaveLen(2))
					orTokens := logicalOr.AllOR()
					Expect(orTokens).To(HaveLen(1))
					Expect(orTokens[0].GetText()).To(Equal("||"))
					return
				}

				logicalAnd := logicalOr.AllLogicalAndExpr()[0]
				Expect(logicalAnd).NotTo(BeNil())

				// For logical AND
				if expectedOperator == "&&" {
					eqExprs := logicalAnd.AllEqualityExpr()
					Expect(eqExprs).To(HaveLen(2))
					andTokens := logicalAnd.AllAND()
					Expect(andTokens).To(HaveLen(1))
					Expect(andTokens[0].GetText()).To(Equal("&&"))
					return
				}

				equality := logicalAnd.AllEqualityExpr()[0]
				Expect(equality).NotTo(BeNil())

				// For equality operators
				if expectedOperator == "==" || expectedOperator == "!=" {
					relExprs := equality.AllRelationalExpr()
					Expect(relExprs).To(HaveLen(2))
					if expectedOperator == "==" {
						eqTokens := equality.AllEQUAL()
						Expect(eqTokens).To(HaveLen(1))
						Expect(eqTokens[0].GetText()).To(Equal("=="))
					} else {
						neqTokens := equality.AllNOT_EQUAL()
						Expect(neqTokens).To(HaveLen(1))
						Expect(neqTokens[0].GetText()).To(Equal("!="))
					}
					return
				}

				relational := equality.AllRelationalExpr()[0]
				Expect(relational).NotTo(BeNil())

				// For relational operators
				if expectedOperator == "<" || expectedOperator == "<=" || expectedOperator == ">" || expectedOperator == ">=" {
					addExprs := relational.AllAdditiveExpr()
					Expect(addExprs).To(HaveLen(2))
					switch expectedOperator {
					case "<":
						ltTokens := relational.AllLESS_THAN()
						Expect(ltTokens).To(HaveLen(1))
						Expect(ltTokens[0].GetText()).To(Equal("<"))
					case "<=":
						leTokens := relational.AllLESS_EQUAL()
						Expect(leTokens).To(HaveLen(1))
						Expect(leTokens[0].GetText()).To(Equal("<="))
					case ">":
						gtTokens := relational.AllGREATER_THAN()
						Expect(gtTokens).To(HaveLen(1))
						Expect(gtTokens[0].GetText()).To(Equal(">"))
					case ">=":
						geTokens := relational.AllGREATER_EQUAL()
						Expect(geTokens).To(HaveLen(1))
						Expect(geTokens[0].GetText()).To(Equal(">="))
					}
					return
				}

				// For additive operators
				additive := relational.AllAdditiveExpr()[0]
				Expect(additive).NotTo(BeNil())

				if expectedOperator == "+" || expectedOperator == "-" {
					multExprs := additive.AllMultiplicativeExpr()
					Expect(multExprs).To(HaveLen(2))

					if expectedOperator == "+" {
						plusTokens := additive.AllPLUS()
						Expect(plusTokens).To(HaveLen(1))
						Expect(plusTokens[0].GetText()).To(Equal("+"))
					} else {
						minusTokens := additive.AllMINUS()
						Expect(minusTokens).To(HaveLen(1))
						Expect(minusTokens[0].GetText()).To(Equal("-"))
					}

					// Verify operands for arithmetic operators
					leftMult := multExprs[0]
					leftUnary := leftMult.AllUnaryExpr()[0]
					leftPrimary := leftUnary.PrimaryExpr()
					Expect(leftPrimary.IDENTIFIER().GetText()).To(Equal(leftOperand))

					rightMult := multExprs[1]
					rightUnary := rightMult.AllUnaryExpr()[0]
					rightPrimary := rightUnary.PrimaryExpr()
					Expect(rightPrimary.IDENTIFIER().GetText()).To(Equal(rightOperand))
					return
				}

				// For multiplicative operators
				multExpr := additive.AllMultiplicativeExpr()[0]
				Expect(multExpr).NotTo(BeNil())

				if expectedOperator == "*" || expectedOperator == "/" {
					unaryExprs := multExpr.AllUnaryExpr()
					Expect(unaryExprs).To(HaveLen(2))

					if expectedOperator == "*" {
						multTokens := multExpr.AllMULTIPLY()
						Expect(multTokens).To(HaveLen(1))
						Expect(multTokens[0].GetText()).To(Equal("*"))
					} else {
						divTokens := multExpr.AllDIVIDE()
						Expect(divTokens).To(HaveLen(1))
						Expect(divTokens[0].GetText()).To(Equal("/"))
					}

					// Verify operands
					leftUnary := unaryExprs[0]
					leftPrimary := leftUnary.PrimaryExpr()
					Expect(leftPrimary.IDENTIFIER().GetText()).To(Equal(leftOperand))

					rightUnary := unaryExprs[1]
					rightPrimary := rightUnary.PrimaryExpr()
					Expect(rightPrimary.IDENTIFIER().GetText()).To(Equal(rightOperand))
				}
			},
			Entry("addition", "a+b", "+", "a", "b"),
			Entry("subtraction", "x-y", "-", "x", "y"),
			Entry("multiplication", "foo*bar", "*", "foo", "bar"),
			Entry("division", "total/count", "/", "total", "count"),
			Entry("equality", "left==right", "==", "left", "right"),
			Entry("inequality", "old!=new", "!=", "old", "new"),
			Entry("less than", "min<max", "<", "min", "max"),
			Entry("less than or equal", "start<=end", "<=", "start", "end"),
			Entry("greater than", "high>low", ">", "high", "low"),
			Entry("greater than or equal", "value>=threshold", ">=", "value", "threshold"),
			Entry("logical AND", "active&&enabled", "&&", "active", "enabled"),
			Entry("logical OR", "ready||done", "||", "ready", "done"),
		)

		DescribeTable("Unary operators",
			func(expression string, expectedOperator string, operand string) {
				expr, err := parser.ParseExpression(expression)
				Expect(err).To(BeNil())
				Expect(expr).NotTo(BeNil())

				// Navigate to unary expression
				logicalOr := expr.LogicalOrExpr()
				logicalAnd := logicalOr.AllLogicalAndExpr()[0]
				equality := logicalAnd.AllEqualityExpr()[0]
				relational := equality.AllRelationalExpr()[0]
				additive := relational.AllAdditiveExpr()[0]
				multExpr := additive.AllMultiplicativeExpr()[0]
				unaryExpr := multExpr.AllUnaryExpr()[0]

				if expectedOperator == "-" {
					Expect(unaryExpr.MINUS()).NotTo(BeNil())
					Expect(unaryExpr.MINUS().GetText()).To(Equal("-"))
				} else if expectedOperator == "!" {
					Expect(unaryExpr.NOT()).NotTo(BeNil())
					Expect(unaryExpr.NOT().GetText()).To(Equal("!"))
				}

				// Check the operand
				innerUnary := unaryExpr.UnaryExpr()
				Expect(innerUnary).NotTo(BeNil())
				primary := innerUnary.PrimaryExpr()
				Expect(primary.IDENTIFIER().GetText()).To(Equal(operand))
			},
			Entry("negation", "-x", "-", "x"),
			Entry("logical NOT", "!flag", "!", "flag"),
		)

		Describe("Complex expressions", func() {
			It("should parse parenthesized expressions", func() {
				expr, err := parser.ParseExpression("(a+b)")
				Expect(err).To(BeNil())
				Expect(expr).NotTo(BeNil())

				// Navigate to the primary expression
				logicalOr := expr.LogicalOrExpr()
				logicalAnd := logicalOr.AllLogicalAndExpr()[0]
				equality := logicalAnd.AllEqualityExpr()[0]
				relational := equality.AllRelationalExpr()[0]
				additive := relational.AllAdditiveExpr()[0]
				multExpr := additive.AllMultiplicativeExpr()[0]
				unaryExpr := multExpr.AllUnaryExpr()[0]
				primary := unaryExpr.PrimaryExpr()

				// Verify parentheses are present
				Expect(primary.LPAREN()).NotTo(BeNil())
				Expect(primary.RPAREN()).NotTo(BeNil())

				// Verify inner expression is a+b
				innerExpr := primary.Expression()
				Expect(innerExpr).NotTo(BeNil())
				innerAdditive := innerExpr.LogicalOrExpr().AllLogicalAndExpr()[0].AllEqualityExpr()[0].AllRelationalExpr()[0].AllAdditiveExpr()[0]
				Expect(innerAdditive.AllPLUS()).To(HaveLen(1))

				leftOperand := innerAdditive.AllMultiplicativeExpr()[0].AllUnaryExpr()[0].PrimaryExpr()
				Expect(leftOperand.IDENTIFIER().GetText()).To(Equal("a"))

				rightOperand := innerAdditive.AllMultiplicativeExpr()[1].AllUnaryExpr()[0].PrimaryExpr()
				Expect(rightOperand.IDENTIFIER().GetText()).To(Equal("b"))
			})

			It("should parse nested parentheses", func() {
				expr, err := parser.ParseExpression("((x+y)*z)")
				Expect(err).To(BeNil())
				Expect(expr).NotTo(BeNil())

				// Navigate to outer parentheses
				logicalOr := expr.LogicalOrExpr()
				logicalAnd := logicalOr.AllLogicalAndExpr()[0]
				equality := logicalAnd.AllEqualityExpr()[0]
				relational := equality.AllRelationalExpr()[0]
				additive := relational.AllAdditiveExpr()[0]
				multExpr := additive.AllMultiplicativeExpr()[0]
				unaryExpr := multExpr.AllUnaryExpr()[0]
				outerPrimary := unaryExpr.PrimaryExpr()

				// Verify outer parentheses
				Expect(outerPrimary.LPAREN()).NotTo(BeNil())
				Expect(outerPrimary.RPAREN()).NotTo(BeNil())

				// Get the multiplication expression inside
				innerExpr := outerPrimary.Expression()
				innerMult := innerExpr.LogicalOrExpr().AllLogicalAndExpr()[0].AllEqualityExpr()[0].AllRelationalExpr()[0].AllAdditiveExpr()[0].AllMultiplicativeExpr()[0]

				// Verify it's a multiplication
				Expect(innerMult.AllMULTIPLY()).To(HaveLen(1))

				// Left side should be (x+y)
				leftUnary := innerMult.AllUnaryExpr()[0]
				leftPrimary := leftUnary.PrimaryExpr()
				Expect(leftPrimary.LPAREN()).NotTo(BeNil()) // Inner parentheses
				Expect(leftPrimary.RPAREN()).NotTo(BeNil())

				// Verify x+y inside inner parentheses
				xyExpr := leftPrimary.Expression()
				xyAdditive := xyExpr.LogicalOrExpr().AllLogicalAndExpr()[0].AllEqualityExpr()[0].AllRelationalExpr()[0].AllAdditiveExpr()[0]
				Expect(xyAdditive.AllPLUS()).To(HaveLen(1))

				// Right side should be z
				rightUnary := innerMult.AllUnaryExpr()[1]
				rightPrimary := rightUnary.PrimaryExpr()
				Expect(rightPrimary.IDENTIFIER().GetText()).To(Equal("z"))
			})

			It("should parse function calls", func() {
				expr, err := parser.ParseExpression("getValue()")
				Expect(err).To(BeNil())
				Expect(expr).NotTo(BeNil())

				// Navigate to primary expression
				logicalOr := expr.LogicalOrExpr()
				logicalAnd := logicalOr.AllLogicalAndExpr()[0]
				equality := logicalAnd.AllEqualityExpr()[0]
				relational := equality.AllRelationalExpr()[0]
				additive := relational.AllAdditiveExpr()[0]
				multExpr := additive.AllMultiplicativeExpr()[0]
				unaryExpr := multExpr.AllUnaryExpr()[0]
				primary := unaryExpr.PrimaryExpr()

				funcCall := primary.FunctionCall()
				Expect(funcCall).NotTo(BeNil())
				Expect(funcCall.IDENTIFIER().GetText()).To(Equal("getValue"))
				Expect(funcCall.LPAREN()).NotTo(BeNil())
				Expect(funcCall.RPAREN()).NotTo(BeNil())
				Expect(funcCall.ArgumentList()).To(BeNil()) // No arguments
			})

			It("should parse function calls with arguments", func() {
				expr, err := parser.ParseExpression("add(x,y)")
				Expect(err).To(BeNil())
				Expect(expr).NotTo(BeNil())

				// Navigate to function call
				logicalOr := expr.LogicalOrExpr()
				logicalAnd := logicalOr.AllLogicalAndExpr()[0]
				equality := logicalAnd.AllEqualityExpr()[0]
				relational := equality.AllRelationalExpr()[0]
				additive := relational.AllAdditiveExpr()[0]
				multExpr := additive.AllMultiplicativeExpr()[0]
				unaryExpr := multExpr.AllUnaryExpr()[0]
				primary := unaryExpr.PrimaryExpr()

				funcCall := primary.FunctionCall()
				Expect(funcCall).NotTo(BeNil())
				Expect(funcCall.IDENTIFIER().GetText()).To(Equal("add"))

				// Verify arguments
				argList := funcCall.ArgumentList()
				Expect(argList).NotTo(BeNil())
				args := argList.AllExpression()
				Expect(args).To(HaveLen(2))

				// First argument should be x
				firstArg := args[0].LogicalOrExpr().AllLogicalAndExpr()[0].AllEqualityExpr()[0].AllRelationalExpr()[0].AllAdditiveExpr()[0].AllMultiplicativeExpr()[0].AllUnaryExpr()[0].PrimaryExpr()
				Expect(firstArg.IDENTIFIER().GetText()).To(Equal("x"))

				// Second argument should be y
				secondArg := args[1].LogicalOrExpr().AllLogicalAndExpr()[0].AllEqualityExpr()[0].AllRelationalExpr()[0].AllAdditiveExpr()[0].AllMultiplicativeExpr()[0].AllUnaryExpr()[0].PrimaryExpr()
				Expect(secondArg.IDENTIFIER().GetText()).To(Equal("y"))
			})

			It("should parse number literals", func() {
				expr, err := parser.ParseExpression("42")
				Expect(err).To(BeNil())
				Expect(expr).NotTo(BeNil())

				// Navigate to primary and verify it's a number literal
				logicalOr := expr.LogicalOrExpr()
				logicalAnd := logicalOr.AllLogicalAndExpr()[0]
				equality := logicalAnd.AllEqualityExpr()[0]
				relational := equality.AllRelationalExpr()[0]
				additive := relational.AllAdditiveExpr()[0]
				multExpr := additive.AllMultiplicativeExpr()[0]
				unaryExpr := multExpr.AllUnaryExpr()[0]
				primary := unaryExpr.PrimaryExpr()

				Expect(primary.NUMBER_LITERAL()).NotTo(BeNil())
				Expect(primary.NUMBER_LITERAL().GetText()).To(Equal("42"))
			})

			It("should parse boolean literals", func() {
				// Test true
				expr, err := parser.ParseExpression("true")
				Expect(err).To(BeNil())
				Expect(expr).NotTo(BeNil())

				logicalOr := expr.LogicalOrExpr()
				logicalAnd := logicalOr.AllLogicalAndExpr()[0]
				equality := logicalAnd.AllEqualityExpr()[0]
				relational := equality.AllRelationalExpr()[0]
				additive := relational.AllAdditiveExpr()[0]
				multExpr := additive.AllMultiplicativeExpr()[0]
				unaryExpr := multExpr.AllUnaryExpr()[0]
				primary := unaryExpr.PrimaryExpr()

				Expect(primary.TRUE()).NotTo(BeNil())
				Expect(primary.TRUE().GetText()).To(Equal("true"))

				// Test false
				expr, err = parser.ParseExpression("false")
				Expect(err).To(BeNil())
				Expect(expr).NotTo(BeNil())

				logicalOr = expr.LogicalOrExpr()
				logicalAnd = logicalOr.AllLogicalAndExpr()[0]
				equality = logicalAnd.AllEqualityExpr()[0]
				relational = equality.AllRelationalExpr()[0]
				additive = relational.AllAdditiveExpr()[0]
				multExpr = additive.AllMultiplicativeExpr()[0]
				unaryExpr = multExpr.AllUnaryExpr()[0]
				primary = unaryExpr.PrimaryExpr()

				Expect(primary.FALSE()).NotTo(BeNil())
				Expect(primary.FALSE().GetText()).To(Equal("false"))
			})

			It("should parse string literals", func() {
				expr, err := parser.ParseExpression(`"hello world"`)
				Expect(err).To(BeNil())
				Expect(expr).NotTo(BeNil())

				// Navigate to primary and verify it's a string literal
				logicalOr := expr.LogicalOrExpr()
				logicalAnd := logicalOr.AllLogicalAndExpr()[0]
				equality := logicalAnd.AllEqualityExpr()[0]
				relational := equality.AllRelationalExpr()[0]
				additive := relational.AllAdditiveExpr()[0]
				multExpr := additive.AllMultiplicativeExpr()[0]
				unaryExpr := multExpr.AllUnaryExpr()[0]
				primary := unaryExpr.PrimaryExpr()

				Expect(primary.STRING()).NotTo(BeNil())
				Expect(primary.STRING().GetText()).To(Equal(`"hello world"`))
			})

			It("should parse channel read", func() {
				expr, err := parser.ParseExpression("<-channel")
				Expect(err).To(BeNil())
				Expect(expr).NotTo(BeNil())

				// Navigate to primary and verify it's a channel read
				logicalOr := expr.LogicalOrExpr()
				logicalAnd := logicalOr.AllLogicalAndExpr()[0]
				equality := logicalAnd.AllEqualityExpr()[0]
				relational := equality.AllRelationalExpr()[0]
				additive := relational.AllAdditiveExpr()[0]
				multExpr := additive.AllMultiplicativeExpr()[0]
				unaryExpr := multExpr.AllUnaryExpr()[0]
				primary := unaryExpr.PrimaryExpr()

				channelRead := primary.ChannelRead()
				Expect(channelRead).NotTo(BeNil())
				Expect(channelRead.CHANNEL_RECV()).NotTo(BeNil())
				Expect(channelRead.IDENTIFIER().GetText()).To(Equal("channel"))
			})
		})

		Describe("Operator precedence", func() {
			It("should parse multiplication before addition", func() {
				// a+b*c should be parsed as a + (b * c)
				expr, err := parser.ParseExpression("a+b*c")
				Expect(err).To(BeNil())
				Expect(expr).NotTo(BeNil())

				// Navigate to the additive expression
				logicalOr := expr.LogicalOrExpr()
				logicalAnd := logicalOr.AllLogicalAndExpr()[0]
				equality := logicalAnd.AllEqualityExpr()[0]
				relational := equality.AllRelationalExpr()[0]
				additive := relational.AllAdditiveExpr()[0]

				// The additive expression should have two multiplicative expressions
				multExprs := additive.AllMultiplicativeExpr()
				Expect(multExprs).To(HaveLen(2))

				// Left operand should be just 'a'
				leftMult := multExprs[0]
				leftUnary := leftMult.AllUnaryExpr()[0]
				leftPrimary := leftUnary.PrimaryExpr()
				Expect(leftPrimary.IDENTIFIER().GetText()).To(Equal("a"))

				// Right operand should be 'b*c' (a multiplicative expression)
				rightMult := multExprs[1]
				rightUnaryExprs := rightMult.AllUnaryExpr()
				Expect(rightUnaryExprs).To(HaveLen(2)) // b and c

				// Verify it's a multiplication
				multTokens := rightMult.AllMULTIPLY()
				Expect(multTokens).To(HaveLen(1))
				Expect(multTokens[0].GetText()).To(Equal("*"))

				// Verify the operands of the multiplication
				bExpr := rightUnaryExprs[0].PrimaryExpr()
				Expect(bExpr.IDENTIFIER().GetText()).To(Equal("b"))

				cExpr := rightUnaryExprs[1].PrimaryExpr()
				Expect(cExpr.IDENTIFIER().GetText()).To(Equal("c"))
			})

			It("should parse parentheses to override precedence", func() {
				// (a+b)*c should be parsed as (a + b) * c
				expr, err := parser.ParseExpression("(a+b)*c")
				Expect(err).To(BeNil())
				Expect(expr).NotTo(BeNil())

				// Navigate to the multiplicative expression (should be at the top level for this case)
				logicalOr := expr.LogicalOrExpr()
				logicalAnd := logicalOr.AllLogicalAndExpr()[0]
				equality := logicalAnd.AllEqualityExpr()[0]
				relational := equality.AllRelationalExpr()[0]
				additive := relational.AllAdditiveExpr()[0]
				multExpr := additive.AllMultiplicativeExpr()[0]

				// The multiplicative expression should have two unary expressions
				unaryExprs := multExpr.AllUnaryExpr()
				Expect(unaryExprs).To(HaveLen(2))

				// Verify it's a multiplication at the top level
				multTokens := multExpr.AllMULTIPLY()
				Expect(multTokens).To(HaveLen(1))
				Expect(multTokens[0].GetText()).To(Equal("*"))

				// Left operand should be a parenthesized addition (a+b)
				leftUnary := unaryExprs[0]
				leftPrimary := leftUnary.PrimaryExpr()

				// Check for parentheses
				Expect(leftPrimary.LPAREN()).NotTo(BeNil())
				Expect(leftPrimary.RPAREN()).NotTo(BeNil())

				// Get the expression inside parentheses
				innerExpr := leftPrimary.Expression()
				Expect(innerExpr).NotTo(BeNil())

				// Verify it's an addition inside
				innerLogicalOr := innerExpr.LogicalOrExpr()
				innerLogicalAnd := innerLogicalOr.AllLogicalAndExpr()[0]
				innerEquality := innerLogicalAnd.AllEqualityExpr()[0]
				innerRelational := innerEquality.AllRelationalExpr()[0]
				innerAdditive := innerRelational.AllAdditiveExpr()[0]

				// Check for the plus token
				plusTokens := innerAdditive.AllPLUS()
				Expect(plusTokens).To(HaveLen(1))
				Expect(plusTokens[0].GetText()).To(Equal("+"))

				// Verify operands of the addition
				innerMultExprs := innerAdditive.AllMultiplicativeExpr()
				Expect(innerMultExprs).To(HaveLen(2))

				aExpr := innerMultExprs[0].AllUnaryExpr()[0].PrimaryExpr()
				Expect(aExpr.IDENTIFIER().GetText()).To(Equal("a"))

				bExpr := innerMultExprs[1].AllUnaryExpr()[0].PrimaryExpr()
				Expect(bExpr.IDENTIFIER().GetText()).To(Equal("b"))

				// Right operand of multiplication should be just 'c'
				rightUnary := unaryExprs[1]
				rightPrimary := rightUnary.PrimaryExpr()
				Expect(rightPrimary.IDENTIFIER().GetText()).To(Equal("c"))
			})

			It("should correctly parse complex precedence", func() {
				// a||b&&c==d<e+f*g should parse with correct precedence:
				// || < && < == < < < + < *
				// So: a || (b && (c == (d < (e + (f * g)))))
				expr, err := parser.ParseExpression("a||b&&c==d<e+f*g")
				Expect(err).To(BeNil())
				Expect(expr).NotTo(BeNil())

				// Top level should be OR
				logicalOr := expr.LogicalOrExpr()
				orExprs := logicalOr.AllLogicalAndExpr()
				Expect(orExprs).To(HaveLen(2))
				orTokens := logicalOr.AllOR()
				Expect(orTokens).To(HaveLen(1))

				// Left of OR should be 'a'
				leftAnd := orExprs[0]
				leftEq := leftAnd.AllEqualityExpr()[0]
				leftRel := leftEq.AllRelationalExpr()[0]
				leftAdd := leftRel.AllAdditiveExpr()[0]
				leftMult := leftAdd.AllMultiplicativeExpr()[0]
				leftUnary := leftMult.AllUnaryExpr()[0]
				leftPrimary := leftUnary.PrimaryExpr()
				Expect(leftPrimary.IDENTIFIER().GetText()).To(Equal("a"))

				// Right of OR should be b&&c==d<e+f*g
				rightAnd := orExprs[1]
				andExprs := rightAnd.AllEqualityExpr()
				Expect(andExprs).To(HaveLen(2))
				andTokens := rightAnd.AllAND()
				Expect(andTokens).To(HaveLen(1))

				// Left of AND should be 'b'
				bEq := andExprs[0]
				bRel := bEq.AllRelationalExpr()[0]
				bAdd := bRel.AllAdditiveExpr()[0]
				bMult := bAdd.AllMultiplicativeExpr()[0]
				bUnary := bMult.AllUnaryExpr()[0]
				bPrimary := bUnary.PrimaryExpr()
				Expect(bPrimary.IDENTIFIER().GetText()).To(Equal("b"))

				// Right of AND should be c==d<e+f*g
				rightEq := andExprs[1]
				eqExprs := rightEq.AllRelationalExpr()
				Expect(eqExprs).To(HaveLen(2))
				eqTokens := rightEq.AllEQUAL()
				Expect(eqTokens).To(HaveLen(1))

				// Left of == should be 'c'
				cRel := eqExprs[0]
				cAdd := cRel.AllAdditiveExpr()[0]
				cMult := cAdd.AllMultiplicativeExpr()[0]
				cUnary := cMult.AllUnaryExpr()[0]
				cPrimary := cUnary.PrimaryExpr()
				Expect(cPrimary.IDENTIFIER().GetText()).To(Equal("c"))

				// Right of == should be d<e+f*g
				rightRel := eqExprs[1]
				relAddExprs := rightRel.AllAdditiveExpr()
				Expect(relAddExprs).To(HaveLen(2))
				ltTokens := rightRel.AllLESS_THAN()
				Expect(ltTokens).To(HaveLen(1))

				// Left of < should be 'd'
				dAdd := relAddExprs[0]
				dMult := dAdd.AllMultiplicativeExpr()[0]
				dUnary := dMult.AllUnaryExpr()[0]
				dPrimary := dUnary.PrimaryExpr()
				Expect(dPrimary.IDENTIFIER().GetText()).To(Equal("d"))

				// Right of < should be e+f*g
				rightAdd := relAddExprs[1]
				addMultExprs := rightAdd.AllMultiplicativeExpr()
				Expect(addMultExprs).To(HaveLen(2))
				plusTokens := rightAdd.AllPLUS()
				Expect(plusTokens).To(HaveLen(1))

				// Left of + should be 'e'
				eMult := addMultExprs[0]
				eUnary := eMult.AllUnaryExpr()[0]
				ePrimary := eUnary.PrimaryExpr()
				Expect(ePrimary.IDENTIFIER().GetText()).To(Equal("e"))

				// Right of + should be f*g
				rightMult := addMultExprs[1]
				fgUnaryExprs := rightMult.AllUnaryExpr()
				Expect(fgUnaryExprs).To(HaveLen(2))
				multTokens := rightMult.AllMULTIPLY()
				Expect(multTokens).To(HaveLen(1))

				// Left of * should be 'f'
				fUnary := fgUnaryExprs[0]
				fPrimary := fUnary.PrimaryExpr()
				Expect(fPrimary.IDENTIFIER().GetText()).To(Equal("f"))

				// Right of * should be 'g'
				gUnary := fgUnaryExprs[1]
				gPrimary := gUnary.PrimaryExpr()
				Expect(gPrimary.IDENTIFIER().GetText()).To(Equal("g"))
			})
		})
	})
	
	Describe("Function Declarations", func() {
		Describe("Basic functions", func() {
			It("should parse a simple function with no parameters", func() {
				source := `func hello() {
					return
				}`
				
				tree, err := parser.Parse(source)
				Expect(err).To(BeNil())
				Expect(tree).NotTo(BeNil())
				
				// Verify we have one top-level statement
				statements := tree.AllTopLevelStatement()
				Expect(statements).To(HaveLen(1))
				
				// Get the function declaration
				funcDecl := statements[0].FunctionDecl()
				Expect(funcDecl).NotTo(BeNil())
				
				// Verify function name
				Expect(funcDecl.IDENTIFIER().GetText()).To(Equal("hello"))
				
				// Verify no parameters
				Expect(funcDecl.ParameterList()).To(BeNil())
				
				// Verify no return type (void)
				Expect(funcDecl.ReturnType()).To(BeNil())
				
				// Verify block exists
				block := funcDecl.Block()
				Expect(block).NotTo(BeNil())
				
				// Verify return statement
				stmts := block.AllStatement()
				Expect(stmts).To(HaveLen(1))
				returnStmt := stmts[0].ReturnStatement()
				Expect(returnStmt).NotTo(BeNil())
				Expect(returnStmt.Expression()).To(BeNil()) // No return value
			})
			
			It("should parse a function with single parameter", func() {
				source := `func square(x number) number {
					return x * x
				}`
				
				tree, err := parser.Parse(source)
				Expect(err).To(BeNil())
				
				funcDecl := tree.AllTopLevelStatement()[0].FunctionDecl()
				
				// Verify parameter
				paramList := funcDecl.ParameterList()
				Expect(paramList).NotTo(BeNil())
				params := paramList.AllParameter()
				Expect(params).To(HaveLen(1))
				
				param := params[0]
				Expect(param.IDENTIFIER().GetText()).To(Equal("x"))
				Expect(param.Type_().NUMBER()).NotTo(BeNil())
				
				// Verify return type
				returnType := funcDecl.ReturnType()
				Expect(returnType).NotTo(BeNil())
				Expect(returnType.Type_().NUMBER()).NotTo(BeNil())
				
				// Verify return statement expression
				block := funcDecl.Block()
				returnStmt := block.AllStatement()[0].ReturnStatement()
				expr := returnStmt.Expression()
				Expect(expr).NotTo(BeNil())
				Expect(expr.GetText()).To(Equal("x*x"))
			})
			
			It("should parse a function with multiple parameters", func() {
				source := `func calculate(a number, b number, flag bool) number {
					return a + b
				}`
				
				tree, err := parser.Parse(source)
				Expect(err).To(BeNil())
				
				funcDecl := tree.AllTopLevelStatement()[0].FunctionDecl()
				
				// Verify all parameters
				params := funcDecl.ParameterList().AllParameter()
				Expect(params).To(HaveLen(3))
				
				// First parameter: a number
				Expect(params[0].IDENTIFIER().GetText()).To(Equal("a"))
				Expect(params[0].Type_().NUMBER()).NotTo(BeNil())
				
				// Second parameter: b number
				Expect(params[1].IDENTIFIER().GetText()).To(Equal("b"))
				Expect(params[1].Type_().NUMBER()).NotTo(BeNil())
				
				// Third parameter: flag bool
				Expect(params[2].IDENTIFIER().GetText()).To(Equal("flag"))
				Expect(params[2].Type_().BOOL()).NotTo(BeNil())
			})
		})
		
		Describe("Function types", func() {
			It("should parse void function", func() {
				source := `func doNothing() void {
					return
				}`
				
				tree, err := parser.Parse(source)
				Expect(err).To(BeNil())
				
				funcDecl := tree.AllTopLevelStatement()[0].FunctionDecl()
				returnType := funcDecl.ReturnType()
				Expect(returnType).NotTo(BeNil())
				Expect(returnType.Type_().VOID()).NotTo(BeNil())
			})
			
			It("should parse channel parameter types", func() {
				source := `func readChannel(input <-chan) number {
					return <-input
				}`
				
				tree, err := parser.Parse(source)
				Expect(err).To(BeNil())
				
				funcDecl := tree.AllTopLevelStatement()[0].FunctionDecl()
				param := funcDecl.ParameterList().AllParameter()[0]
				
				// Verify channel type
				paramType := param.Type_()
				Expect(paramType.CHANNEL_RECV()).NotTo(BeNil())
				Expect(paramType.CHAN()).NotTo(BeNil())
			})
			
			It("should parse send-only channel parameter", func() {
				source := `func writeChannel(output ->chan, value number) {
					value -> output
				}`
				
				tree, err := parser.Parse(source)
				Expect(err).To(BeNil())
				
				funcDecl := tree.AllTopLevelStatement()[0].FunctionDecl()
				param := funcDecl.ParameterList().AllParameter()[0]
				
				// Verify send-only channel type
				paramType := param.Type_()
				Expect(paramType.CHANNEL_SEND()).NotTo(BeNil())
				Expect(paramType.CHAN()).NotTo(BeNil())
			})
		})
		
		Describe("Function body statements", func() {
			It("should parse variable declarations", func() {
				source := `func compute() number {
					x := 10
					y $= 20
					return x + y
				}`
				
				tree, err := parser.Parse(source)
				Expect(err).To(BeNil())
				
				funcDecl := tree.AllTopLevelStatement()[0].FunctionDecl()
				stmts := funcDecl.Block().AllStatement()
				Expect(stmts).To(HaveLen(3))
				
				// First: local variable declaration
				varDecl1 := stmts[0].VariableDecl()
				Expect(varDecl1).NotTo(BeNil())
				Expect(varDecl1.IDENTIFIER().GetText()).To(Equal("x"))
				Expect(varDecl1.LOCAL_ASSIGN()).NotTo(BeNil())
				
				// Second: state variable declaration
				varDecl2 := stmts[1].VariableDecl()
				Expect(varDecl2).NotTo(BeNil())
				Expect(varDecl2.IDENTIFIER().GetText()).To(Equal("y"))
				Expect(varDecl2.STATE_ASSIGN()).NotTo(BeNil())
			})
			
			It("should parse assignments", func() {
				source := `func update(x number) number {
					result := x
					result = result * 2
					return result
				}`
				
				tree, err := parser.Parse(source)
				Expect(err).To(BeNil())
				
				funcDecl := tree.AllTopLevelStatement()[0].FunctionDecl()
				stmts := funcDecl.Block().AllStatement()
				
				// Second statement should be assignment
				assignment := stmts[1].Assignment()
				Expect(assignment).NotTo(BeNil())
				Expect(assignment.IDENTIFIER().GetText()).To(Equal("result"))
				Expect(assignment.ASSIGN()).NotTo(BeNil())
				Expect(assignment.Expression()).NotTo(BeNil())
			})
			
			It("should parse if statements", func() {
				source := `func max(a number, b number) number {
					if (a > b) return a else return b
				}`
				
				tree, err := parser.Parse(source)
				Expect(err).To(BeNil())
				
				funcDecl := tree.AllTopLevelStatement()[0].FunctionDecl()
				stmts := funcDecl.Block().AllStatement()
				
				// Should have one if statement
				ifStmt := stmts[0].IfStatement()
				Expect(ifStmt).NotTo(BeNil())
				
				// Verify condition
				condition := ifStmt.Expression()
				Expect(condition).NotTo(BeNil())
				Expect(condition.GetText()).To(Equal("a>b"))
				
				// Verify then and else branches
				thenStmt := ifStmt.Statement(0)
				Expect(thenStmt).NotTo(BeNil())
				
				elseStmt := ifStmt.Statement(1)
				Expect(elseStmt).NotTo(BeNil())
			})
			
			It("should parse channel operations", func() {
				source := `func process(input <-chan, output ->chan) {
					value := <-input
					result := value * 2
					result -> output
				}`
				
				tree, err := parser.Parse(source)
				Expect(err).To(BeNil())
				
				funcDecl := tree.AllTopLevelStatement()[0].FunctionDecl()
				stmts := funcDecl.Block().AllStatement()
				Expect(stmts).To(HaveLen(3))
				
				// First: channel read in variable declaration
				varDecl := stmts[0].VariableDecl()
				Expect(varDecl).NotTo(BeNil())
				
				// Third: channel write
				channelWrite := stmts[2].ChannelWrite()
				Expect(channelWrite).NotTo(BeNil())
				Expect(channelWrite.CHANNEL_SEND()).NotTo(BeNil())
			})
			
			It("should parse expression statements", func() {
				source := `func callOther() void {
					doSomething()
					process(42)
					return
				}`
				
				tree, err := parser.Parse(source)
				Expect(err).To(BeNil())
				
				funcDecl := tree.AllTopLevelStatement()[0].FunctionDecl()
				stmts := funcDecl.Block().AllStatement()
				// Check we have at least the expected statements
				Expect(len(stmts)).To(BeNumerically(">=", 3))
				
				// First two should be expression statements  
				expr1 := stmts[0].ExpressionStatement()
				Expect(expr1).NotTo(BeNil())
				
				expr2 := stmts[1].ExpressionStatement()
				Expect(expr2).NotTo(BeNil())
				
				// Last should be return (could be index 2 or 3 depending on parsing)
				lastIdx := len(stmts) - 1
				returnStmt := stmts[lastIdx].ReturnStatement()
				Expect(returnStmt).NotTo(BeNil())
			})
		})
		
		Describe("Multiple functions", func() {
			It("should parse multiple function declarations", func() {
				source := `
				func first() number {
					return 1
				}
				
				func second() number {
					return 2
				}
				
				func third() number {
					return 3
				}`
				
				tree, err := parser.Parse(source)
				Expect(err).To(BeNil())
				
				// Should have three top-level statements
				statements := tree.AllTopLevelStatement()
				Expect(statements).To(HaveLen(3))
				
				// Verify each is a function
				func1 := statements[0].FunctionDecl()
				Expect(func1).NotTo(BeNil())
				Expect(func1.IDENTIFIER().GetText()).To(Equal("first"))
				
				func2 := statements[1].FunctionDecl()
				Expect(func2).NotTo(BeNil())
				Expect(func2.IDENTIFIER().GetText()).To(Equal("second"))
				
				func3 := statements[2].FunctionDecl()
				Expect(func3).NotTo(BeNil())
				Expect(func3.IDENTIFIER().GetText()).To(Equal("third"))
			})
		})
	})
	
	Describe("Reactive Bindings", func() {
		It("should parse simple channel binding", func() {
			source := `channel -> process()`
			
			tree, err := parser.Parse(source)
			Expect(err).To(BeNil())
			
			statements := tree.AllTopLevelStatement()
			Expect(statements).To(HaveLen(1))
			
			// Get reactive binding
			binding := statements[0].ReactiveBinding()
			Expect(binding).NotTo(BeNil())
			
			// Verify channel identifier
			Expect(binding.IDENTIFIER()).NotTo(BeNil())
			Expect(binding.IDENTIFIER().GetText()).To(Equal("channel"))
			
			// Verify arrow
			Expect(binding.CHANNEL_SEND()).NotTo(BeNil())
			
			// Verify function call
			funcCall := binding.FunctionCall()
			Expect(funcCall).NotTo(BeNil())
			Expect(funcCall.IDENTIFIER().GetText()).To(Equal("process"))
		})
		
		It("should parse channel list binding", func() {
			source := `[ch1, ch2, ch3] -> handleMultiple()`
			
			tree, err := parser.Parse(source)
			Expect(err).To(BeNil())
			
			statements := tree.AllTopLevelStatement()
			binding := statements[0].ReactiveBinding()
			
			// Verify channel list
			channelList := binding.ChannelList()
			Expect(channelList).NotTo(BeNil())
			
			// Verify brackets
			Expect(channelList.LBRACKET()).NotTo(BeNil())
			Expect(channelList.RBRACKET()).NotTo(BeNil())
			
			// Verify channel identifiers
			identifiers := channelList.AllIDENTIFIER()
			Expect(identifiers).To(HaveLen(3))
			Expect(identifiers[0].GetText()).To(Equal("ch1"))
			Expect(identifiers[1].GetText()).To(Equal("ch2"))
			Expect(identifiers[2].GetText()).To(Equal("ch3"))
		})
		
		It("should parse interval binding", func() {
			source := `interval(1000) -> tick()`
			
			tree, err := parser.Parse(source)
			Expect(err).To(BeNil())
			
			statements := tree.AllTopLevelStatement()
			binding := statements[0].ReactiveBinding()
			
			// Get interval binding
			intervalBinding := binding.IntervalBinding()
			Expect(intervalBinding).NotTo(BeNil())
			
			// Verify interval keyword
			Expect(intervalBinding.INTERVAL()).NotTo(BeNil())
			
			// Verify number literal
			Expect(intervalBinding.NUMBER_LITERAL()).NotTo(BeNil())
			Expect(intervalBinding.NUMBER_LITERAL().GetText()).To(Equal("1000"))
			
			// Verify function call
			funcCall := intervalBinding.FunctionCall()
			Expect(funcCall).NotTo(BeNil())
			Expect(funcCall.IDENTIFIER().GetText()).To(Equal("tick"))
		})
		
		It("should parse mixed program with functions and bindings", func() {
			source := `
			func process() {
				return
			}
			
			channel -> process()
			
			func handleTick() {
				return
			}
			
			interval(500) -> handleTick()`
			
			tree, err := parser.Parse(source)
			Expect(err).To(BeNil())
			
			statements := tree.AllTopLevelStatement()
			Expect(statements).To(HaveLen(4))
			
			// First: function
			Expect(statements[0].FunctionDecl()).NotTo(BeNil())
			
			// Second: channel binding
			Expect(statements[1].ReactiveBinding()).NotTo(BeNil())
			
			// Third: function
			Expect(statements[2].FunctionDecl()).NotTo(BeNil())
			
			// Fourth: interval binding
			Expect(statements[3].ReactiveBinding()).NotTo(BeNil())
		})
	})
})
