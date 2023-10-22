// Copyright 2023 Synnax Labs, Inc.
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
	"strconv"

	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/x/stack"
)

// A dictionary mapping operators to their token
var operatorTokens map[string]token.Token = map[string]token.Token{
	"+": token.ADD,
	"-": token.SUB,
	"*": token.MUL,
	"/": token.QUO,
	"^": token.XOR,
}

var precedence map[string]int = map[string]int{
	"+": 1,
	"-": 1,
	"*": 2,
	"/": 2,
	"(": 0,
	"^": 3,
}

var InvalidExpressionError = errors.New("Invalid expression")

type Expression struct {
	exp ast.Expr
}

type Resolver interface {
	Resolve(string) (float64, error)
}

func findTokens(s string) (tokens []string) {
	currToken := ""
	for _, c := range s {
		previousToken := ""
		if len(tokens) > 0 {
			previousToken = tokens[len(tokens)-1]
		}
		_, previousTokenIsOperator := operatorTokens[previousToken]
		_, currTokenInOperator := operatorTokens[string(c)]
		if c == ' ' && currToken != "" {
			tokens = append(tokens, currToken)
			currToken = ""
		} else if c == '-' && currToken == "" && (len(tokens) == 0 || previousToken == "(" || previousTokenIsOperator) {
			//	Check if token is a negative number
			currToken += string(c)
		} else if currTokenInOperator || c == '(' || c == ')' {
			if currToken != "" {
				tokens = append(tokens, currToken)
				currToken = ""
			}
			tokens = append(tokens, string(c))
		} else if c != ' ' {
			currToken += string(c)
		}
	}
	if currToken != "" {
		tokens = append(tokens, currToken)
	}
	return
}

// Given a string representing a valid mathematical expression, sets exp to the
// parsed expression
func (e *Expression) Build(s string) error {
	tokens := findTokens(s)
	output := stack.Stack[interface{}]{}
	operators := stack.Stack[string]{}
	//	Use shunting-yard algorithm
	for _, t := range tokens {
		switch t {
		case "+", "-", "*", "/":
			for operators.Len() > 0 && precedence[*operators.Peek()] >= precedence[t] {
				op, err := operators.Pop()
				if err != nil {
					return err
				}
				Y, err1 := output.Pop()
				X, err2 := output.Pop()
				if err1 != nil || err2 != nil {
					return errors.Wrap(InvalidExpressionError, "Invalid expression: binary operator used with only one operand")
				}
				output.Push(&ast.BinaryExpr{X: X.(ast.Expr), Op: operatorTokens[op], Y: Y.(ast.Expr)})
			}
			operators.Push(t)
		case "^":
			for operators.Len() > 0 && precedence[*operators.Peek()] > precedence[t] {
				op, err := operators.Pop()
				if err != nil {
					return err
				}
				Y, err1 := output.Pop()
				X, err2 := output.Pop()
				if err1 != nil || err2 != nil {
					return errors.Wrap(InvalidExpressionError, "Invalid expression: binary operator used with only one operand")
				}
				output.Push(&ast.BinaryExpr{X: X.(ast.Expr), Op: operatorTokens[op], Y: Y.(ast.Expr)})
			}
			operators.Push(t)
		case "(":
			operators.Push(t)
		case ")":
			for operators.Len() > 0 && *operators.Peek() != "(" {
				op, err := operators.Pop()
				if err != nil {
					return err
				}
				Y, err1 := output.Pop()
				X, err2 := output.Pop()
				if err1 != nil || err2 != nil {
					return errors.Wrap(InvalidExpressionError, "Invalid expression: binary operator used with only one operand")
				}
				output.Push(&ast.BinaryExpr{X: X.(ast.Expr), Op: operatorTokens[op], Y: Y.(ast.Expr)})
			}
			_, err := operators.Pop()
			if err != nil {
				return errors.Wrap(InvalidExpressionError, "Invalid expression: mismatched parentheses")
			}
		default:
			_, err := strconv.ParseFloat(t, 64)
			if err != nil {
				output.Push(&ast.Ident{Name: t})
			} else {
				output.Push(&ast.BasicLit{Kind: token.FLOAT, Value: t})
			}

		}
	}
	for operators.Len() > 0 {
		op, _ := operators.Pop()
		Y, err1 := output.Pop()
		X, err2 := output.Pop()
		if err1 != nil || err2 != nil {
			return errors.Wrap(InvalidExpressionError, "Invalid expression: binary operator used with only one operand")
		}
		output.Push(&ast.BinaryExpr{X: X.(ast.Expr), Op: operatorTokens[op], Y: Y.(ast.Expr)})
	}
	exp, err := output.Pop()
	if err != nil {
		return err
	}
	e.exp = exp.(ast.Expr)
	return nil
}

func (e Expression) Tree() ast.Expr {
	return e.exp
}

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
	}

	return 0
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
	}

	return 0
}
