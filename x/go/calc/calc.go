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
	"errors"
	"go/ast"
	"go/token"
	"math"
	"strconv"

	"github.com/synnaxlabs/x/stack"
)

type Expression struct {
	exp ast.Expr
}

type Resolver interface {
	Resolve(string) (float64, error)
}

func findTokens(s string) ([]string, error) {
	tokens := []string{}
	currToken := ""
	for i, c := range s {
		if c == ' ' && currToken != "" {
			tokens = append(tokens, currToken)
			currToken = ""
		} else if c == '-' && currToken == "" && (i == 0 || tokens[len(tokens)-1] == "(" || tokens[len(tokens)-1] == "+" || tokens[len(tokens)-1] == "-" || tokens[len(tokens)-1] == "*" || tokens[len(tokens)-1] == "/" || tokens[len(tokens)-1] == "^") {
			//	Check if token is a negative number
			currToken += string(c)
		} else if c == '+' || c == '-' || c == '*' || c == '/' || c == '(' || c == ')' || c == '^' {
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
	return tokens, nil
}

// Given a string representing a valid mathematical expression, sets exp to the
// parsed expression
func (e *Expression) Build(s string) error {
	tokens, err := findTokens(s)
	if err != nil {
		return err
	}
	//	Uses the shunting yard algorithm to create an ast
	precedence := map[string]int{
		"+": 1,
		"-": 1,
		"*": 2,
		"/": 2,
		"(": 0,
		"^": 3,
	}
	tokensDict := map[string]token.Token{
		"+": token.ADD,
		"-": token.SUB,
		"*": token.MUL,
		"/": token.QUO,
		"^": token.XOR,
	}
	output := stack.Stack[interface{}]{}
	operators := stack.Stack[string]{}

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
					return errors.New("Invalid expression")
				}
				output.Push(&ast.BinaryExpr{X: X.(ast.Expr), Op: tokensDict[op], Y: Y.(ast.Expr)})
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
					return errors.New("Invalid expression")
				}
				output.Push(&ast.BinaryExpr{X: X.(ast.Expr), Op: tokensDict[op], Y: Y.(ast.Expr)})
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
					return errors.New("Invalid expression")
				}
				output.Push(&ast.BinaryExpr{X: X.(ast.Expr), Op: tokensDict[op], Y: Y.(ast.Expr)})
			}
			_, err := operators.Pop()
			if err != nil {
				return err
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
		op, err := operators.Pop()
		if err != nil {
			return err
		}
		Y, err1 := output.Pop()
		X, err2 := output.Pop()
		if err1 != nil || err2 != nil {
			return errors.New("Invalid expression")
		}
		output.Push(&ast.BinaryExpr{X: X.(ast.Expr), Op: tokensDict[op], Y: Y.(ast.Expr)})
	}
	exp, err := output.Pop()
	if err != nil {
		return err
	}
	e.exp = exp.(ast.Expr)
	return nil
}

func (e Expression) GetTree() ast.Expr {
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
