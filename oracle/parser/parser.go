// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:generate antlr4 -Dlanguage=Go -o . -package parser OracleLexer.g4 OracleParser.g4

// Package parser provides parsing functionality for the Oracle schema language.
// It uses ANTLR4-generated parsers to convert Oracle schema source code into
// abstract syntax trees.
//
// Basic usage:
//
//	tree, err := parser.Parse(`
//	    struct Range {
//	        field key uuid {
//	            domain id
//	        }
//	        field name string
//	    }
//	`)
//	if err != nil {
//	    // handle parse error
//	}
package parser

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/x/diagnostics"
)

// Parse parses a complete Oracle schema from source code.
//
// Returns an ISchemaContext representing the parsed schema's abstract syntax tree,
// or diagnostics containing syntax errors if the source is malformed.
//
// Example:
//
//	tree, diag := parser.Parse(`
//	    import "schema/core/label"
//
//	    struct Range {
//	        field key uuid { domain id }
//	        field name string {
//	            domain validate { required }
//	        }
//	    }
//	`)
func Parse(source string) (ISchemaContext, *diagnostics.Diagnostics) {
	return parseWithContext(source, (*OracleParser).Schema)
}

// parseWithContext executes the parsing with proper error handling.
// It sets up the lexer, parser, and error listener, then invokes the provided
// parse function to generate the appropriate parse tree node.
func parseWithContext[T any](source string, parseFn func(*OracleParser) T) (T, *diagnostics.Diagnostics) {
	var (
		input  = antlr.NewInputStream(source)
		lexer  = NewOracleLexer(input)
		stream = antlr.NewCommonTokenStream(lexer, 0)
		parser = NewOracleParser(stream)
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
