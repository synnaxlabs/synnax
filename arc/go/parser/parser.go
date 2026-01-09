// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:generate antlr4 -Dlanguage=Go -o . -package parser ArcLexer.g4 ArcParser.g4

// Package parser provides parsing functionality for the Arc programming language.
// It uses ANTLR4-generated parsers to convert Arc source code into abstract syntax trees.
//
// The parser supports parsing complete programs as well as individual expressions,
// statements, and blocks for interactive use cases like REPLs or inline evaluation.
//
// Basic usage:
//
//	tree, err := parser.Parse(`
//	    func add(x f64, y f64) f64 {
//	        return x + y
//	    }
//	`)
//	if err != nil {
//	    // handle parse error
//	}
//
// For parsing individual constructs:
//
//	expr, err := parser.ParseExpression("2 + 3 * 4")
//	stmt, err := parser.ParseStatement("x := 42")
//	block, err := parser.ParseBlock("{ x := 1\n y := 2 }")
package parser

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/diagnostics"
)

// Parse parses a complete Arc program from source code.
//
// Returns an IProgramContext representing the parsed program's abstract syntax tree,
// or an error if the source contains syntax errors. The error will contain detailed
// position information for all syntax errors encountered.
//
// Example:
//
//	tree, err := parser.Parse(`
//	    func double(x f64) f64 {
//	        return x * 2
//	    }
//	`)
func Parse(source string) (IProgramContext, *diagnostics.Diagnostics) {
	return parseWithContext(source, (*ArcParser).Program)
}

// ParseExpression parses a single Arc expression.
//
// This is useful for evaluating expressions in isolation, such as in a REPL
// or configuration file. The expression is parsed with the same precedence
// and associativity rules as expressions within a program.
//
// Example:
//
//	expr, err := parser.ParseExpression("(2 + 3) * 4")
func ParseExpression(source string) (IExpressionContext, *diagnostics.Diagnostics) {
	return parseWithContext(source, (*ArcParser).Expression)
}

// ParseStatement parses a single Arc statement.
//
// Useful for line-by-line parsing in interactive environments. Note that
// some statements (like function declarations) are only valid at the top level
// of a program and will fail when parsed as standalone statements.
//
// Example:
//
//	stmt, err := parser.ParseStatement("total := total + 1")
func ParseStatement(source string) (IStatementContext, *diagnostics.Diagnostics) {
	return parseWithContext(source, (*ArcParser).Statement)
}

// ParseBlock parses an Arc block (sequence of statements enclosed in braces).
//
// Example:
//
//	block, err := parser.ParseBlock(`{
//	    x := 10
//	    y := x * 2
//	}`)
func ParseBlock(source string) (IBlockContext, *diagnostics.Diagnostics) {
	return parseWithContext(source, (*ArcParser).Block)
}

// parseWithContext executes the parsing with proper error handling.
// It sets up the lexer, parser, and error listener, then invokes the provided
// parse function to generate the appropriate parse tree node.
func parseWithContext[T any](source string, parseFn func(*ArcParser) T) (T, *diagnostics.Diagnostics) {
	var (
		input  = antlr.NewInputStream(source)
		lexer  = NewArcLexer(input)
		stream = antlr.NewCommonTokenStream(lexer, 0)
		parser = NewArcParser(stream)
		diag   = &diagnostics.Diagnostics{}
		errLis = &errorListener{Diagnostics: diag}
	)
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(errLis)
	parser.RemoveErrorListeners()
	parser.AddErrorListener(errLis)
	result := parseFn(parser)
	if !diag.Ok() {
		var zeroT T
		return zeroT, diag
	}
	return result, nil
}

// errorListener collects syntax errors encountered during parsing.
// It implements antlr.ErrorListener to capture all parse errors with
// position information.
type errorListener struct {
	*antlr.DefaultErrorListener
	*diagnostics.Diagnostics
}

// SyntaxError is called by ANTLR when a syntax error is encountered.
// It records the error along with its position in the source code.
func (e *errorListener) SyntaxError(
	_ antlr.Recognizer,
	_ interface{},
	line,
	column int,
	msg string,
	_ antlr.RecognitionException,
) {
	e.Add(diagnostics.Diagnostic{
		Severity: diagnostics.Error,
		Line:     line,
		Column:   column,
		Message:  msg,
	})
}

// TokensAdjacent returns true if two tokens are adjacent with no whitespace between them.
// This is used by the numericLiteral grammar rule to determine if an IDENTIFIER
// immediately follows a numeric literal (making it a unit suffix like "300ms")
// versus being separated by whitespace (making them separate tokens).
//
// The check uses character positions: if prev token ends at position X,
// and next token starts at position X+1, they are adjacent.
func (p *ArcParser) TokensAdjacent(prev, next antlr.Token) bool {
	if prev == nil || next == nil {
		return false
	}
	// GetStop() returns the last character index of the token (inclusive)
	// GetStart() returns the first character index of the token
	// Adjacent means next starts exactly where prev ends + 1
	return prev.GetStop()+1 == next.GetStart()
}
