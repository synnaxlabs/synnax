// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lsp

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/lsp/doc"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"go.lsp.dev/protocol"
	"go.uber.org/zap"
)

func (s *Server) Hover(
	_ context.Context,
	params *protocol.HoverParams,
) (*protocol.Hover, error) {
	d, ok := s.getDocument(params.TextDocument.URI)
	if !ok {
		s.cfg.L.Debug(
			"hover: document not found",
			zap.String("uri", string(params.TextDocument.URI)),
		)
		return nil, nil
	}

	operator := s.getOperatorAtPosition(d.Content, params.Position)
	if operator != "" {
		contents := s.getOperatorHoverContents(operator)
		if contents != "" {
			return &protocol.Hover{
				Contents: protocol.MarkupContent{
					Kind:  protocol.Markdown,
					Value: contents,
				},
			}, nil
		}
	}

	word := getWordAtPosition(d.Content, params.Position)
	if word == "" {
		s.cfg.L.Debug(
			"hover: no word at position",
			zap.Uint32("line", params.Position.Line),
			zap.Uint32("char", params.Position.Character),
		)
		return nil, nil
	}

	contents := s.getHoverContents(word)
	if contents == "" && d.IR.Symbols != nil {
		scopeAtCursor := FindScopeAtPosition(d.IR.Symbols, FromProtocol(params.Position))
		contents = s.getUserSymbolHover(scopeAtCursor, word, d.Content)
	}

	if contents == "" {
		return nil, nil
	}

	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: contents,
		},
	}, nil
}

var operators = []string{
	parser.LiteralDECLARE, parser.LiteralSTATE_DECLARE, parser.LiteralTRANSITION, parser.LiteralARROW,
	parser.LiteralPLUS_ASSIGN, parser.LiteralMINUS_ASSIGN, parser.LiteralSTAR_ASSIGN, parser.LiteralSLASH_ASSIGN, parser.LiteralPERCENT_ASSIGN,
	parser.LiteralEQ, parser.LiteralNEQ, parser.LiteralLEQ, parser.LiteralGEQ,
}

// operatorDocs contains pre-computed documentation for operators.
var operatorDocs = map[string]string{
	parser.LiteralDECLARE: doc.New(
		doc.TitleWithKind(parser.LiteralDECLARE, "Operator"),
		doc.Paragraph("Declares and initializes a new local variable."),
		doc.Divider(),
		doc.ArcCode("x := 42\nname := \"hello\""),
		doc.Divider(),
		doc.Paragraph("The variable type is inferred from the right-hand side expression."),
	).Render(),
	parser.LiteralSTATE_DECLARE: doc.New(
		doc.TitleWithKind(parser.LiteralSTATE_DECLARE, "Operator"),
		doc.Paragraph("Declares a stateful variable that persists across executions."),
		doc.Divider(),
		doc.ArcCode("count $= 0\ncount = count + 1"),
		doc.Divider(),
		doc.Paragraph("Stateful variables retain their values between reactive stage executions, making them useful for counters, accumulators, and maintaining state."),
	).Render(),
	parser.LiteralTRANSITION: doc.New(
		doc.TitleWithKind(parser.LiteralTRANSITION, "Operator"),
		doc.Paragraph("Transitions to another stage in a sequence."),
		doc.Divider(),
		doc.ArcCode("sequence main {\n    stage first {\n        if ready => second\n    }\n    stage second {}\n}"),
		doc.Divider(),
		doc.Paragraph("When the condition is true, execution transitions to the specified stage on the next cycle."),
	).Render(),
	parser.LiteralARROW: doc.New(
		doc.TitleWithKind(parser.LiteralARROW, "Operator"),
		doc.Paragraph("Writes a value to a channel."),
		doc.Divider(),
		doc.ArcCode("value -> outputChannel"),
		doc.Divider(),
		doc.Paragraph("Sends the left-hand value to the channel on the right."),
	).Render(),
	parser.LiteralPLUS_ASSIGN: doc.New(
		doc.TitleWithKind(parser.LiteralPLUS_ASSIGN, "Operator"),
		doc.Paragraph("Adds and assigns."),
		doc.Divider(),
		doc.ArcCode("x += 5  // equivalent to: x = x + 5"),
	).Render(),
	parser.LiteralMINUS_ASSIGN: doc.New(
		doc.TitleWithKind(parser.LiteralMINUS_ASSIGN, "Operator"),
		doc.Paragraph("Subtracts and assigns."),
		doc.Divider(),
		doc.ArcCode("x -= 5  // equivalent to: x = x - 5"),
	).Render(),
	parser.LiteralSTAR_ASSIGN: doc.New(
		doc.TitleWithKind(parser.LiteralSTAR_ASSIGN, "Operator"),
		doc.Paragraph("Multiplies and assigns."),
		doc.Divider(),
		doc.ArcCode("x *= 2  // equivalent to: x = x * 2"),
	).Render(),
	parser.LiteralSLASH_ASSIGN: doc.New(
		doc.TitleWithKind(parser.LiteralSLASH_ASSIGN, "Operator"),
		doc.Paragraph("Divides and assigns."),
		doc.Divider(),
		doc.ArcCode("x /= 2  // equivalent to: x = x / 2"),
	).Render(),
	parser.LiteralPERCENT_ASSIGN: doc.New(
		doc.TitleWithKind(parser.LiteralPERCENT_ASSIGN, "Operator"),
		doc.Paragraph("Computes modulo and assigns."),
		doc.Divider(),
		doc.ArcCode("x %= 3  // equivalent to: x = x % 3"),
	).Render(),
	parser.LiteralEQ: doc.New(
		doc.TitleWithKind(parser.LiteralEQ, "Operator"),
		doc.Paragraph("Tests equality between two values."),
		doc.Divider(),
		doc.ArcCode("if x == 10 { ... }"),
	).Render(),
	parser.LiteralNEQ: doc.New(
		doc.TitleWithKind(parser.LiteralNEQ, "Operator"),
		doc.Paragraph("Tests inequality between two values."),
		doc.Divider(),
		doc.ArcCode("if x != 0 { ... }"),
	).Render(),
	parser.LiteralLEQ: doc.New(
		doc.TitleWithKind(parser.LiteralLEQ, "Operator"),
		doc.Paragraph("Tests if left value is less than or equal to right value."),
		doc.Divider(),
		doc.ArcCode("if x <= 100 { ... }"),
	).Render(),
	parser.LiteralGEQ: doc.New(
		doc.TitleWithKind(parser.LiteralGEQ, "Operator"),
		doc.Paragraph("Tests if left value is greater than or equal to right value."),
		doc.Divider(),
		doc.ArcCode("if x >= 0 { ... }"),
	).Render(),
}

// keywordDocs contains pre-computed documentation for keywords, types, and built-in functions.
var keywordDocs = map[string]string{
	parser.LiteralFUNC: doc.New(
		doc.TitleWithKind(parser.LiteralFUNC, "Keyword"),
		doc.Paragraph("Declares a function."),
		doc.Divider(),
		doc.ArcCode("func name(param type) returnType {\n    // body\n}"),
	).Render(),
	parser.LiteralSTAGE: doc.New(
		doc.TitleWithKind(parser.LiteralSTAGE, "Keyword"),
		doc.Paragraph("Declares a stage within a sequence."),
		doc.Divider(),
		doc.ArcCode("sequence name {\n    stage stageName {\n        // body\n    }\n}"),
	).Render(),
	parser.LiteralSEQUENCE: doc.New(
		doc.TitleWithKind(parser.LiteralSEQUENCE, "Keyword"),
		doc.Paragraph("Declares a sequence (state machine)."),
		doc.Divider(),
		doc.ArcCode("sequence name {\n    stage first {\n        // initial stage\n    }\n}"),
	).Render(),
	parser.LiteralIF: doc.New(
		doc.TitleWithKind(parser.LiteralIF, "Keyword"),
		doc.Paragraph("Conditional statement."),
		doc.Divider(),
		doc.ArcCode("if condition {\n    // body\n}"),
	).Render(),
	parser.LiteralELSE: doc.New(
		doc.TitleWithKind(parser.LiteralELSE, "Keyword"),
		doc.Paragraph("Alternative branch for if statement."),
		doc.Divider(),
		doc.ArcCode("if condition {\n    // body\n} else {\n    // alternative\n}"),
	).Render(),
	parser.LiteralRETURN: doc.New(
		doc.TitleWithKind(parser.LiteralRETURN, "Keyword"),
		doc.Paragraph("Returns a value from a function."),
	).Render(),
	parser.LiteralNEXT: doc.New(
		doc.TitleWithKind(parser.LiteralNEXT, "Keyword"),
		doc.Paragraph("Transitions to a stage unconditionally."),
		doc.Divider(),
		doc.ArcCode("stage first {\n    next second\n}"),
	).Render(),
	parser.LiteralI8: doc.New(
		doc.TitleWithKind(parser.LiteralI8, "Type"),
		doc.Paragraph("Signed 8-bit integer."),
		doc.Detail("Range", "-128 to 127", false),
	).Render(),
	parser.LiteralI16: doc.New(
		doc.TitleWithKind(parser.LiteralI16, "Type"),
		doc.Paragraph("Signed 16-bit integer."),
		doc.Detail("Range", "-32768 to 32767", false),
	).Render(),
	parser.LiteralI32: doc.New(
		doc.TitleWithKind(parser.LiteralI32, "Type"),
		doc.Paragraph("Signed 32-bit integer."),
		doc.Detail("Range", "-2147483648 to 2147483647", false),
	).Render(),
	parser.LiteralI64: doc.New(
		doc.TitleWithKind(parser.LiteralI64, "Type"),
		doc.Paragraph("Signed 64-bit integer."),
		doc.Detail("Range", "-9223372036854775808 to 9223372036854775807", false),
	).Render(),
	parser.LiteralU8: doc.New(
		doc.TitleWithKind(parser.LiteralU8, "Type"),
		doc.Paragraph("Unsigned 8-bit integer."),
		doc.Detail("Range", "0 to 255", false),
	).Render(),
	parser.LiteralU16: doc.New(
		doc.TitleWithKind(parser.LiteralU16, "Type"),
		doc.Paragraph("Unsigned 16-bit integer."),
		doc.Detail("Range", "0 to 65535", false),
	).Render(),
	parser.LiteralU32: doc.New(
		doc.TitleWithKind(parser.LiteralU32, "Type"),
		doc.Paragraph("Unsigned 32-bit integer."),
		doc.Detail("Range", "0 to 4294967295", false),
	).Render(),
	parser.LiteralU64: doc.New(
		doc.TitleWithKind(parser.LiteralU64, "Type"),
		doc.Paragraph("Unsigned 64-bit integer."),
		doc.Detail("Range", "0 to 18446744073709551615", false),
	).Render(),
	parser.LiteralF32: doc.New(
		doc.TitleWithKind(parser.LiteralF32, "Type"),
		doc.Paragraph("32-bit floating point number (single precision)."),
	).Render(),
	parser.LiteralF64: doc.New(
		doc.TitleWithKind(parser.LiteralF64, "Type"),
		doc.Paragraph("64-bit floating point number (double precision)."),
	).Render(),
	"string": doc.New(
		doc.TitleWithKind("string", "Type"),
		doc.Paragraph("Immutable UTF-8 encoded string."),
	).Render(),
	"timestamp": doc.New(
		doc.TitleWithKind("timestamp", "Type"),
		doc.Paragraph("Point in time represented as nanoseconds since Unix epoch."),
	).Render(),
	"timespan": doc.New(
		doc.TitleWithKind("timespan", "Type"),
		doc.Paragraph("Duration represented as nanoseconds."),
	).Render(),
	parser.LiteralSERIES: doc.New(
		doc.TitleWithKind(parser.LiteralSERIES, "Type"),
		doc.Paragraph("Homogeneous array of values."),
		doc.Divider(),
		doc.ArcCode("series f64"),
	).Render(),
	parser.LiteralCHAN: doc.New(
		doc.TitleWithKind(parser.LiteralCHAN, "Type"),
		doc.Paragraph("Bidirectional channel for communication."),
		doc.Divider(),
		doc.ArcCode("chan f64"),
	).Render(),
	"len": doc.New(
		doc.TitleWithKind("len", "Function"),
		doc.Paragraph("Returns the length of a series."),
		doc.Divider(),
		doc.ArcCode("length := len(data)"),
	).Render(),
	"now": doc.New(
		doc.TitleWithKind("now", "Function"),
		doc.Paragraph("Returns the current timestamp."),
		doc.Divider(),
		doc.ArcCode("time := now()"),
	).Render(),
}

func (s *Server) getOperatorAtPosition(content string, pos protocol.Position) string {
	line, ok := getLine(content, pos.Line)
	if !ok {
		return ""
	}
	col := int(pos.Character)
	if col >= len(line) {
		return ""
	}
	for _, op := range operators {
		opLen := len(op)
		for startOffset := 0; startOffset < opLen; startOffset++ {
			start := col - startOffset
			if start < 0 || start+opLen > len(line) {
				continue
			}
			if line[start:start+opLen] == op {
				return op
			}
		}
	}
	return ""
}

func (s *Server) getOperatorHoverContents(op string) string {
	return operatorDocs[op]
}

func (s *Server) getHoverContents(word string) string {
	return keywordDocs[word]
}

func (s *Server) extractDocComment(content string, sym *symbol.Scope) string {
	if sym.AST == nil {
		return ""
	}
	start := sym.AST.GetStart()
	if start == nil {
		return ""
	}

	symLine := start.GetLine()
	tokens := TokenizeContentWithComments(content)
	if len(tokens) == 0 {
		return ""
	}

	var commentTokens []string
	for i := len(tokens) - 1; i >= 0; i-- {
		t := tokens[i]
		tokenType := t.GetTokenType()
		tokenLine := t.GetLine()

		if tokenLine >= symLine {
			continue
		}

		if tokenType == parser.ArcLexerSINGLE_LINE_COMMENT ||
			tokenType == parser.ArcLexerMULTI_LINE_COMMENT {
			if hasCodeBetween(tokens, i, symLine) {
				break
			}
			commentTokens = append([]string{t.GetText()}, commentTokens...)
		} else if tokenType != parser.ArcLexerWS && tokenType != antlr.TokenEOF {
			break
		}
	}

	if len(commentTokens) == 0 {
		return ""
	}

	return cleanDocComment(commentTokens)
}

func hasCodeBetween(tokens []antlr.Token, fromIndex int, targetLine int) bool {
	startLine := tokens[fromIndex].GetLine()
	commentText := tokens[fromIndex].GetText()
	endLine := startLine
	for _, ch := range commentText {
		if ch == '\n' {
			endLine++
		}
	}

	for i := fromIndex + 1; i < len(tokens); i++ {
		t := tokens[i]
		tokenLine := t.GetLine()
		tokenType := t.GetTokenType()

		if tokenLine <= endLine {
			continue
		}
		if tokenLine >= targetLine {
			break
		}

		if tokenType == parser.ArcLexerWS ||
			tokenType == antlr.TokenEOF ||
			tokenType == parser.ArcLexerSINGLE_LINE_COMMENT ||
			tokenType == parser.ArcLexerMULTI_LINE_COMMENT {
			continue
		}

		return true
	}
	return false
}

func cleanDocComment(comments []string) string {
	var lines []string
	for _, comment := range comments {
		if strings.HasPrefix(comment, "//") {
			line := strings.TrimPrefix(comment, "//")
			line = strings.TrimPrefix(line, " ")
			lines = append(lines, line)
		} else if strings.HasPrefix(comment, "/*") {
			text := strings.TrimPrefix(comment, "/*")
			text = strings.TrimSuffix(text, "*/")
			text = strings.TrimSpace(text)
			for _, line := range strings.Split(text, "\n") {
				line = strings.TrimSpace(line)
				line = strings.TrimPrefix(line, "*")
				line = strings.TrimPrefix(line, " ")
				lines = append(lines, line)
			}
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func (s *Server) getUserSymbolHover(scope *symbol.Scope, name string, content string) string {
	sym, err := scope.Resolve(context.Background(), name)
	if err != nil {
		return ""
	}

	docComment := s.extractDocComment(content, sym)

	var d doc.Doc
	switch sym.Kind {
	case symbol.KindFunction:
		d = doc.New(doc.TitleWithKind(sym.Name, formatFunctionKindDescription(sym)))
		d.Add(doc.Divider())
		d.Add(doc.ArcCode(formatFunctionSignatureContent(sym)))
	case symbol.KindVariable:
		d = doc.New(doc.TitleWithKind(sym.Name, "Variable"))
		d.Add(doc.Detail("Type", sym.Type.String(), true))
	case symbol.KindStatefulVariable:
		d = doc.New(doc.TitleWithKind(sym.Name, "Stateful Variable"))
		d.Add(doc.Paragraph("Persists across executions"))
		d.Add(doc.Detail("Type", sym.Type.String(), true))
	case symbol.KindInput:
		d = doc.New(doc.TitleWithKind(sym.Name, "Input Parameter"))
		d.Add(doc.Detail("Type", sym.Type.String(), true))
	case symbol.KindOutput:
		d = doc.New(doc.TitleWithKind(sym.Name, "Output Parameter"))
		d.Add(doc.Detail("Type", sym.Type.String(), true))
	case symbol.KindConfig:
		d = doc.New(doc.TitleWithKind(sym.Name, "Configuration Parameter"))
		d.Add(doc.Detail("Type", sym.Type.String(), true))
	case symbol.KindChannel:
		d = doc.New(doc.TitleWithKind(sym.Name, "Channel"))
		d.Add(doc.Detail("Type", sym.Type.String(), true))
	case symbol.KindSequence:
		d = doc.New(doc.TitleWithKind(sym.Name, "Sequence"))
		if stages := formatSequenceStagesList(sym); len(stages) > 0 {
			d.Add(doc.Paragraph("Stages: " + strings.Join(stages, ", ")))
		}
	case symbol.KindStage:
		d = doc.New(doc.TitleWithKind(sym.Name, "Stage"))
	default:
		d = doc.New(doc.Title(sym.Name))
		d.Add(doc.Detail("Type", sym.Type.String(), true))
	}
	if docComment != "" {
		d.Add(doc.Divider())
		d.Add(doc.Paragraph(docComment))
	}
	return d.Render()
}

// formatFunctionSignatureContent returns the function signature without code fences.
func formatFunctionSignatureContent(sym *symbol.Scope) string {
	if sym.Type.Kind != types.KindFunction {
		return ""
	}
	var sig strings.Builder
	sig.WriteString("func ")
	sig.WriteString(sym.Name)
	if len(sym.Type.Config) > 0 {
		sig.WriteString("{")
		first := true
		for _, param := range sym.Type.Config {
			if !first {
				sig.WriteString(", ")
			}
			sig.WriteString(fmt.Sprintf("\n    %s %s", param.Name, param.Type))
			first = false
		}
		sig.WriteString("\n}")
	}
	sig.WriteString("(")
	if len(sym.Type.Inputs) > 0 {
		first := true
		for _, param := range sym.Type.Inputs {
			if !first {
				sig.WriteString(", ")
			}
			sig.WriteString(fmt.Sprintf("%s %s", param.Name, param.Type))
			first = false
		}
	}
	sig.WriteString(")")
	if len(sym.Type.Outputs) > 0 {
		sig.WriteString(" ")
		if len(sym.Type.Outputs) == 1 {
			outputType := sym.Type.Outputs[0].Type
			sig.WriteString(outputType.String())
		} else {
			sig.WriteString("{")
			for _, param := range sym.Type.Outputs {
				sig.WriteString(fmt.Sprintf("\n    %s %s", param.Name, param.Type))
			}
			sig.WriteString("\n}")
		}
	}
	return sig.String()
}

func formatFunctionKindDescription(sym *symbol.Scope) string {
	if sym.Type.Config != nil {
		return "Reactive stage with configuration"
	}
	return "Function"
}

// formatSequenceStagesList returns a list of formatted stage names.
func formatSequenceStagesList(sym *symbol.Scope) []string {
	var stages []string
	for _, child := range sym.Children {
		if child.Kind == symbol.KindStage {
			stages = append(stages, "`"+child.Name+"`")
		}
	}
	return stages
}

// symbolToLocation converts a symbol to an LSP Location pointing to its definition
func (s *Server) symbolToLocation(
	uri protocol.DocumentURI,
	sym *symbol.Scope,
) *protocol.Location {
	if sym.AST == nil {
		return nil
	}
	start := sym.AST.GetStart()
	if start == nil {
		return nil
	}
	line := uint32(start.GetLine() - 1)
	col := uint32(start.GetColumn())
	return &protocol.Location{
		URI: uri,
		Range: protocol.Range{
			Start: protocol.Position{Line: line, Character: col},
			End:   protocol.Position{Line: line, Character: col + uint32(len(sym.Name))},
		},
	}
}
