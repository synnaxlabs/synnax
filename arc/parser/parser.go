//go:generate antlr4 -Dlanguage=Go -o . -package parser SlateLexer.g4 SlateParser.g4
package parser

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
)

func Parse(source string) (IProgramContext, error) {
	input := antlr.NewInputStream(source)
	lexer := NewSlateLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, 0)
	parser := NewSlateParser(stream)
	errorListener := &ErrorListener{}
	parser.RemoveErrorListeners()
	parser.AddErrorListener(errorListener)
	tree := parser.Program()
	if errorListener.HasErrors() {
		return nil, fmt.Errorf("parse errors: %v", errorListener.Errors)
	}
	return tree, nil
}

// ErrorListener collects parsing errors
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

// ParseExpression parses a single expression
func ParseExpression(source string) (IExpressionContext, error) {
	input := antlr.NewInputStream(source)
	lexer := NewSlateLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, 0)
	parser := NewSlateParser(stream)
	errorListener := &ErrorListener{}
	parser.RemoveErrorListeners()
	parser.AddErrorListener(errorListener)
	tree := parser.Expression()
	if errorListener.HasErrors() {
		return nil, fmt.Errorf("parse errors: %v", errorListener.Errors)
	}
	return tree, nil
}

// ParseStatement parses a single statement
func ParseStatement(source string) (IStatementContext, error) {
	input := antlr.NewInputStream(source)
	lexer := NewSlateLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, 0)
	parser := NewSlateParser(stream)
	errorListener := &ErrorListener{}
	parser.RemoveErrorListeners()
	parser.AddErrorListener(errorListener)
	tree := parser.Statement()
	if errorListener.HasErrors() {
		return nil, fmt.Errorf("parse errors: %v", errorListener.Errors)
	}
	return tree, nil
}

// ParseBlock parses a block of statements
func ParseBlock(source string) (IBlockContext, error) {
	input := antlr.NewInputStream(source)
	lexer := NewSlateLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, 0)
	parser := NewSlateParser(stream)
	errorListener := &ErrorListener{}
	parser.RemoveErrorListeners()
	parser.AddErrorListener(errorListener)
	tree := parser.Block()
	if errorListener.HasErrors() {
		return nil, fmt.Errorf("parse errors: %v", errorListener.Errors)
	}
	return tree, nil
}
