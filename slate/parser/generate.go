//go:generate antlr4 -Dlanguage=Go -o generated -package parser SlateLexer.g4 SlateParser.g4

package parser

// This file contains the go:generate directive to regenerate the parser
// from the ANTLR grammar files.
//
// To regenerate the parser, run:
//   go generate ./...
// or
//   go generate ./parser
//
// Prerequisites:
//   - ANTLR 4 must be installed (brew install antlr4)
//   - The antlr4 command must be in your PATH