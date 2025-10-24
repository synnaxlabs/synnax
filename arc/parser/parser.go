// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:generate antlr4 -Dlanguage=Go -o . -package parser ArcLexer.g4 ArcParser.g4
package parser

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
)

func Parse(source string) (IProgramContext, error) {
	input := antlr.NewInputStream(source)
	lexer := NewArcLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, 0)
	parser := NewArcParser(stream)
	errorListener := &ErrorListener{}
	parser.RemoveErrorListeners()
	parser.AddErrorListener(errorListener)
	tree := parser.Program()
	if errorListener.HasErrors() {
		return nil, fmt.Errorf("parse errors: %v", errorListener.Errors)
	}
	return tree, nil
}

type ErrorListener struct {
	*antlr.DefaultErrorListener
	Errors []string
}

func (e *ErrorListener) SyntaxError(recognizer antlr.Recognizer, offendingSymbol interface{}, line, column int, msg string, ex antlr.RecognitionException) {
	err := fmt.Sprintf("line %d:%d %s", line, column, msg)
	e.Errors = append(e.Errors, err)
}

func (e *ErrorListener) HasErrors() bool {
	return len(e.Errors) > 0
}

func ParseExpression(source string) (IExpressionContext, error) {
	input := antlr.NewInputStream(source)
	lexer := NewArcLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, 0)
	parser := NewArcParser(stream)
	errorListener := &ErrorListener{}
	parser.RemoveErrorListeners()
	parser.AddErrorListener(errorListener)
	tree := parser.Expression()
	if errorListener.HasErrors() {
		return nil, fmt.Errorf("parse errors: %v", errorListener.Errors)
	}
	return tree, nil
}

func ParseStatement(source string) (IStatementContext, error) {
	input := antlr.NewInputStream(source)
	lexer := NewArcLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, 0)
	parser := NewArcParser(stream)
	errorListener := &ErrorListener{}
	parser.RemoveErrorListeners()
	parser.AddErrorListener(errorListener)
	tree := parser.Statement()
	if errorListener.HasErrors() {
		return nil, fmt.Errorf("parse errors: %v", errorListener.Errors)
	}
	return tree, nil
}

func ParseBlock(source string) (IBlockContext, error) {
	input := antlr.NewInputStream(source)
	lexer := NewArcLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, 0)
	parser := NewArcParser(stream)
	errorListener := &ErrorListener{}
	parser.RemoveErrorListeners()
	parser.AddErrorListener(errorListener)
	tree := parser.Block()
	if errorListener.HasErrors() {
		return nil, fmt.Errorf("parse errors: %v", errorListener.Errors)
	}
	return tree, nil
}
