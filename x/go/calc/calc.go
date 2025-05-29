// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calc

import (
	"go/ast"
	"go/token"
	"math"
	"regexp"
	"strconv"

	"github.com/synnaxlabs/x/deque"
	"github.com/synnaxlabs/x/errors"
)

// A dictionary mapping operators to their token
var operatorTokens map[string]token.Token = map[string]token.Token{
	"+":  token.ADD,
	"-":  token.SUB,
	"*":  token.MUL,
	"/":  token.QUO,
	"^":  token.XOR,
	"==": token.EQL,
	"!=": token.NEQ,
	">":  token.GTR,
	"<":  token.LSS,
	">=": token.GEQ,
	"<=": token.LEQ,
	"&&": token.LAND,
	"||": token.LOR,
}

var precedence map[string]int = map[string]int{
	"+":  1,
	"-":  1,
	"*":  2,
	"/":  2,
	"(":  -3,
	"^":  3,
	"==": -1,
	"!=": -1,
	">":  -1,
	"<":  -1,
	">=": -1,
	"<=": -1,
	"&&": -2,
	"||": -2,
	"!":  0,
}

var InvalidExpressionError = errors.New("Invalid expression")

// Expression is a datatype for representing and evaluating mathematical expressions
type Expression struct {
	exp ast.Expr
}

// Resolver is an interface for resolving identifiers in an expression
type Resolver interface {
	Resolve(string) (float64, error)
}

func findTokens(s string) (tokens []string, err error) {
	re, err := regexp.Compile("[0-9]+(\\.[0-9]*)?|[\\w]+|[+\\-*\\/^()]|[><!=]=|[<>]|&&|\\|\\||!")
	return re.FindAllString(s, -1), err
}

func popOperators(output *deque.Deque[interface{}], operators *deque.Deque[string]) error {
	op := operators.PopBack()
	if op == "!" {
		if output.Len() == 0 {
			return errors.Wrap(InvalidExpressionError, "Invalid expression: unary operator used with no operand")
		}
		X := output.PopBack()
		output.PushBack(&ast.UnaryExpr{Op: token.NOT, X: X.(ast.Expr)})
		return nil
	}
	if output.Len() < 2 {
		return errors.Wrap(InvalidExpressionError, "Invalid expression: binary operator used with only one operand")
	}
	Y := output.PopBack()
	X := output.PopBack()
	output.PushBack(&ast.BinaryExpr{X: X.(ast.Expr), Op: operatorTokens[op], Y: Y.(ast.Expr)})
	return nil
}

// Build builds an AST from a string
func (e *Expression) Build(s string) error {
	tokens, tokenError := findTokens(s)
	if tokenError != nil {
		return errors.Wrap(InvalidExpressionError, "Invalid expression: invalid token")
	}
	output := deque.Deque[interface{}]{}
	operators := deque.Deque[string]{}
	//	Use shunting-yard algorithm
	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		switch t {
		case "-":
			var tokenInDict bool
			if i > 0 {
				_, tokenInDict = precedence[tokens[i-1]]
			}
			if i == 0 || (tokenInDict) {
				output.PushBack(&ast.BasicLit{Kind: token.FLOAT, Value: "-1"})
				operators.PushBack("*")
			} else {
				for operators.Len() > 0 && precedence[operators.Back()] >= precedence[t] {
					if err := popOperators(&output, &operators); err != nil {
						return err
					}
				}
				operators.PushBack(t)
			}
		case "+", "*", "/", "==", "!=", ">", "<", ">=", "<=", "&&", "||":
			for operators.Len() > 0 && precedence[operators.Back()] >= precedence[t] {
				if err := popOperators(&output, &operators); err != nil {
					return err
				}
			}
			operators.PushBack(t)
		case "^":
			for operators.Len() > 0 && precedence[operators.Back()] > precedence[t] {
				if err := popOperators(&output, &operators); err != nil {
					return err
				}
			}
			operators.PushBack(t)
		case "(", "!":
			operators.PushBack(t)
		case ")":
			for operators.Len() > 0 && operators.Back() != "(" {
				if err := popOperators(&output, &operators); err != nil {
					return err
				}
			}
			if operators.Len() == 0 {
				return errors.Wrap(InvalidExpressionError, "Invalid expression: mismatched parentheses")
			}
			operators.PopBack()
		default:
			_, err := strconv.ParseFloat(t, 64)
			if err != nil {
				output.PushBack(&ast.Ident{Name: t})
			} else {
				output.PushBack(&ast.BasicLit{Kind: token.FLOAT, Value: t})
			}

		}
	}
	for operators.Len() > 0 {
		err := popOperators(&output, &operators)
		if err != nil {
			return err
		}
	}
	if output.Len() == 0 {
		return errors.Wrap(InvalidExpressionError, "Invalid expression: invalid number of operands")
	}
	exp := output.PopBack()
	e.exp = exp.(ast.Expr)
	return nil
}

// Tree returns the AST of the expression
func (e Expression) Tree() ast.Expr {
	return e.exp
}

// Evaluate evaluates the expression
func (e Expression) Evaluate(r Resolver) float64 {
	return eval(e.exp, r)
}

func eval(exp ast.Expr, r Resolver) float64 {
	switch exp := exp.(type) {
	case *ast.BinaryExpr:
		return evalBinaryExpr(exp, r)
	case *ast.BasicLit:
		switch exp.Kind {
		case token.FLOAT:
			i, _ := strconv.ParseFloat(exp.Value, 64)
			return i
		}
	case *ast.Ident:
		i, _ := r.Resolve(exp.Name)
		return i
	case *ast.UnaryExpr:
		switch exp.Op {
		case token.NOT:
			return boolToFloat(eval(exp.X, r) == 0)
		}
	}

	return 0
}

func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

func evalBinaryExpr(exp *ast.BinaryExpr, r Resolver) float64 {
	left := eval(exp.X, r)
	right := eval(exp.Y, r)

	switch exp.Op {
	case token.ADD:
		return left + right
	case token.SUB:
		return left - right
	case token.MUL:
		return left * right
	case token.QUO:
		return left / right
	case token.XOR:
		return math.Pow(left, right)
	case token.EQL:
		return boolToFloat(left == right)
	case token.NEQ:
		return boolToFloat(left != right)
	case token.GTR:
		return boolToFloat(left > right)
	case token.LSS:
		return boolToFloat(left < right)
	case token.GEQ:
		return boolToFloat(left >= right)
	case token.LEQ:
		return boolToFloat(left <= right)
	case token.LAND:
		return boolToFloat(left == 1 && right == 1)
	case token.LOR:
		return boolToFloat(left == 1 || right == 1)
	}

	return 0
}
