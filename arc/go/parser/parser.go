// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:generate antlr4 -Dlanguage=Go -visitor -o . -package parser ArcLexer.g4 ArcParser.g4
//go:generate go run gen_token_literals.go

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
	"github.com/synnaxlabs/x/diagnostics"
	"go.lsp.dev/protocol"
)

// Config carries per-parse language settings. Adding a field threads a new setting
// through every parse and analysis path without changing call signatures.
type Config struct {
	// AllowDashedNames permits '-' inside identifiers so "sensor-1" lexes as one token.
	// Off by default; enable only when Core runs with channel-name validation disabled.
	AllowDashedNames bool
}

// ConfigOf returns the effective config from a variadic list: the last entry, or the
// zero Config when none is given. The zero Config means validation-on (dashed names
// off), so omitting it preserves the default behavior.
func ConfigOf(cfgs ...Config) Config {
	if len(cfgs) == 0 {
		return Config{}
	}
	return cfgs[len(cfgs)-1]
}

// dashAwareStream carries per-parse dashed-name configuration to the lexer predicate
// without a package-level global.
type dashAwareStream struct {
	antlr.CharStream
	allowDashedNames bool
}

// dashJoinsIdentifier reports whether the lexer should absorb the upcoming '-' into the
// current IDENTIFIER token. Invoked from the generated IDENTIFIER predicate.
func (l *ArcLexer) dashJoinsIdentifier() bool {
	in := l.GetInputStream()
	s, ok := in.(*dashAwareStream)
	if !ok || !s.allowDashedNames {
		return false
	}
	return in.LA(2) != '>' && in.LA(2) != '='
}

// NewLexer constructs an ArcLexer over source configured with cfg.
func NewLexer(source string, cfg Config) *ArcLexer {
	var input antlr.CharStream = antlr.NewInputStream(source)
	if cfg.AllowDashedNames {
		input = &dashAwareStream{CharStream: input, allowDashedNames: true}
	}
	return NewArcLexer(input)
}

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
func Parse(source string, cfgs ...Config) (IProgramContext, *diagnostics.Diagnostics) {
	return parseWithContext(source, ConfigOf(cfgs...), (*ArcParser).Program)
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
func ParseExpression(source string, cfgs ...Config) (IExpressionContext, *diagnostics.Diagnostics) {
	return parseWithContext(source, ConfigOf(cfgs...), (*ArcParser).Expression)
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
func ParseStatement(source string, cfgs ...Config) (IStatementContext, *diagnostics.Diagnostics) {
	return parseWithContext(source, ConfigOf(cfgs...), (*ArcParser).Statement)
}

// ParseBlock parses an Arc block (sequence of statements enclosed in braces).
//
// Example:
//
//	block, err := parser.ParseBlock(`{
//	    x := 10
//	    y := x * 2
//	}`)
func ParseBlock(source string, cfgs ...Config) (IBlockContext, *diagnostics.Diagnostics) {
	return parseWithContext(source, ConfigOf(cfgs...), (*ArcParser).Block)
}

// parseWithContext executes the parsing with proper error handling.
// It sets up the lexer, parser, and error listener, then invokes the provided
// parse function to generate the appropriate parse tree node.
func parseWithContext[T any](
	source string,
	cfg Config,
	parseFn func(*ArcParser) T,
) (T, *diagnostics.Diagnostics) {
	var (
		lexer  = NewLexer(source, cfg)
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
	_ any,
	line,
	column int,
	msg string,
	_ antlr.RecognitionException,
) {
	e.Add(diagnostics.Diagnostic{
		Severity: protocol.DiagnosticSeverityError,
		Range: protocol.Range{
			Start: protocol.Position{Line: uint32(line - 1), Character: uint32(column)},
			End:   protocol.Position{Line: uint32(line - 1), Character: uint32(column + 1)},
		},
		Message: msg,
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
