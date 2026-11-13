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

// Package parser provides parsing functionality for the Arc programming language. It
// uses ANTLR4-generated parsers to convert Arc source code into abstract syntax trees.
//
// The parser supports parsing complete programs as well as individual expressions,
// statements, and blocks for interactive use cases like REPLs or inline evaluation.
//
// Basic usage:
//
//	tree, err := parser.Parse(`
//	    func add() {x f64, y f64} f64 {
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

// dashJoinsIdentifier reports whether the lexer should absorb the '-' it just matched
// into the current IDENTIFIER token. Invoked from the generated IDENTIFIER predicate.
func (l *ArcLexer) dashJoinsIdentifier() bool {
	in := l.GetInputStream()
	s, ok := in.(*dashAwareStream)
	if !ok || !s.allowDashedNames {
		return false
	}
	return in.LA(1) != '>' && in.LA(1) != '='
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
//	    func double() {x f64} f64 {
//	        return x * 2
//	    }
//	`)
func Parse(source string, cfgs ...Config) (IProgramContext, *diagnostics.Diagnostics) {
	return parseWithContext(source, ConfigOf(cfgs...), nil, (*ArcParser).Program)
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
func ParseExpression(
	source string,
	cfgs ...Config,
) (IExpressionContext, *diagnostics.Diagnostics) {
	return parseWithContext(source, ConfigOf(cfgs...), nil, (*ArcParser).Expression)
}

// Position maps the index of a rune in a source to its 1-based line and its column in
// an enclosing source.
type Position func(index int) (line, column int)

// ParseExpressionAt parses an Arc expression that sits inside an enclosing source. Its
// tokens and diagnostics carry the positions that position gives.
func ParseExpressionAt(
	source string,
	position Position,
	cfgs ...Config,
) (IExpressionContext, *diagnostics.Diagnostics) {
	return parseWithContext(
		source,
		ConfigOf(cfgs...),
		position,
		(*ArcParser).Expression,
	)
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
func ParseStatement(
	source string,
	cfgs ...Config,
) (IStatementContext, *diagnostics.Diagnostics) {
	return parseWithContext(source, ConfigOf(cfgs...), nil, (*ArcParser).Statement)
}

// ParseBlock parses an Arc block (sequence of statements enclosed in braces).
//
// Example:
//
//	block, err := parser.ParseBlock(`{
//	    x := 10
//	    y := x * 2
//	}`)
func ParseBlock(
	source string,
	cfgs ...Config,
) (IBlockContext, *diagnostics.Diagnostics) {
	return parseWithContext(source, ConfigOf(cfgs...), nil, (*ArcParser).Block)
}

// parseWithContext executes the parsing with proper error handling.
// It sets up the lexer, parser, and error listener, then invokes the provided
// parse function to generate the appropriate parse tree node.
func parseWithContext[T any](
	source string,
	cfg Config,
	position Position,
	parseFn func(*ArcParser) T,
) (result T, _ *diagnostics.Diagnostics) {
	diag := &diagnostics.Diagnostics{}
	if len(source) > maxSourceSize {
		diag.Add(sizeDiagnostic(source))
		return result, diag
	}
	var (
		lexer              = NewLexer(source, cfg)
		tokens antlr.Lexer = lexer
		errLis             = &errorListener{Diagnostics: diag}
		lexLis             = errLis
	)
	if position != nil {
		tokens = &positionedTokens{ArcLexer: lexer, position: position}
		lexLis = &errorListener{
			Diagnostics: diag,
			position:    lexerPosition([]rune(source), position),
		}
	}
	var (
		stream = antlr.NewCommonTokenStream(tokens, 0)
		parser = NewArcParser(stream)
		sllErr = &syntaxErrorFlag{}
		limit  = &depthLimit{}
	)
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexLis)
	parser.RemoveErrorListeners()
	parser.AddParseListener(limit)
	parse := func() (exceeded *depthExceeded) {
		defer func() {
			if r := recover(); r != nil {
				d, ok := r.(depthExceeded)
				if !ok {
					panic(r)
				}
				exceeded = &d
			}
		}()
		limit.depth = 0
		result = parseFn(parser)
		return nil
	}
	// SLL prediction is much faster than LL, and when it reports no syntax error it
	// returns the tree LL would. Only source that fails SLL is parsed again with LL.
	parser.AddErrorListener(sllErr)
	parser.GetInterpreter().SetPredictionMode(antlr.PredictionModeSLL)
	exceeded := parse()
	if exceeded == nil && sllErr.reported {
		stream.Seek(0)
		parser.SetTokenStream(stream)
		parser.RemoveErrorListeners()
		parser.AddErrorListener(errLis)
		parser.GetInterpreter().SetPredictionMode(antlr.PredictionModeLL)
		exceeded = parse()
	}
	if exceeded != nil {
		var zeroT T
		return zeroT, &diagnostics.Diagnostics{exceeded.diagnostic()}
	}
	if !diag.Ok() {
		var zeroT T
		return zeroT, diag
	}
	return result, nil
}

// syntaxErrorFlag records whether a parse reported a syntax error.
type syntaxErrorFlag struct {
	*antlr.DefaultErrorListener
	reported bool
}

func (f *syntaxErrorFlag) SyntaxError(
	antlr.Recognizer,
	any,
	int,
	int,
	string,
	antlr.RecognitionException,
) {
	f.reported = true
}

// errorListener collects syntax errors encountered during parsing.
// It implements antlr.ErrorListener to capture all parse errors with
// position information.
type errorListener struct {
	*antlr.DefaultErrorListener
	*diagnostics.Diagnostics
	// position, when set, maps a line and column the lexer counts to the enclosing
	// source.
	position func(line, column int) (int, int)
}

// positionedTokens gives each token the position of its first rune in an enclosing
// source.
type positionedTokens struct {
	*ArcLexer
	position Position
}

func (p *positionedTokens) NextToken() antlr.Token {
	t := p.ArcLexer.NextToken()
	line, column := p.position(t.GetStart())
	return antlr.CommonTokenFactoryDEFAULT.Create(
		t.GetSource(),
		t.GetTokenType(),
		"",
		t.GetChannel(),
		t.GetStart(),
		t.GetStop(),
		line,
		column,
	)
}

// lexerPosition converts the line and column the lexer counts in source to the
// position in the enclosing source.
func lexerPosition(source []rune, position Position) func(line, column int) (int, int) {
	return func(line, column int) (int, int) {
		index := 0
		for l := 1; l < line && index < len(source); index++ {
			if source[index] == '\n' {
				l++
			}
		}
		return position(index + column)
	}
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
	if e.position != nil {
		line, column = e.position(line, column)
	}
	e.Add(diagnostics.Diagnostic{
		Severity: protocol.DiagnosticSeverityError,
		Range: protocol.Range{
			Start: protocol.Position{Line: uint32(line - 1), Character: uint32(column)},
			End: protocol.Position{
				Line:      uint32(line - 1),
				Character: uint32(column + 1),
			},
		},
		Message: msg,
	})
}

// TokensAdjacent returns true if two tokens are adjacent with no whitespace between
// them. This is used by the numericLiteral grammar rule to determine if an IDENTIFIER
// immediately follows a numeric literal (making it a unit suffix like "300ms") versus
// being separated by whitespace (making them separate tokens).
//
// The check uses character positions: if prev token ends at position X, and next token
// starts at position X+1, they are adjacent.
func (p *ArcParser) TokensAdjacent(prev, next antlr.Token) bool {
	if prev == nil || next == nil {
		return false
	}
	// GetStop() returns the last character index of the token (inclusive)
	// GetStart() returns the first character index of the token
	// Adjacent means next starts exactly where prev ends + 1
	return prev.GetStop()+1 == next.GetStart()
}

// CalleeTouchesParen reports whether the upcoming name, bare or dotted, is followed by
// '(' with no whitespace between them. A flow node instantiates a function only in that
// case; `name (x)` is a bare name and a separate parenthesized item.
func (p *ArcParser) CalleeTouchesParen() bool {
	stream := p.GetTokenStream()
	nameIndex := 1
	if dot := stream.LT(2); dot != nil && dot.GetTokenType() == ArcLexerDOT {
		nameIndex = 3
	}
	paren := stream.LT(nameIndex + 1)
	return paren != nil && paren.GetTokenType() == ArcLexerLPAREN &&
		p.TokensAdjacent(stream.LT(nameIndex), paren)
}

// FmtSegment is one piece of a format string: literal text, or a placeholder.
type FmtSegment struct {
	// Text is the literal text with its escapes applied, or the placeholder's source.
	Text string
	// Spec is the placeholder's format spec, or empty.
	Spec string
	// Expr is the placeholder's parsed expression, or nil for literal text.
	Expr IExpressionContext
}

// FmtStrings maps each analyzed format string token to its segments, in order.
type FmtStrings map[antlr.TerminalNode][]FmtSegment
