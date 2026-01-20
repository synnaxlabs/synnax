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
	"strconv"
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

	word := s.getWordAtPosition(d.Content, params.Position)
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
		scopeAtCursor := s.findScopeAtPosition(d.IR.Symbols, params.Position)
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

func (s *Server) getWordAtPosition(content string, pos protocol.Position) string {
	lines := strings.Split(content, "\n")
	if int(pos.Line) >= len(lines) {
		return ""
	}
	line := lines[pos.Line]
	if int(pos.Character) >= len(line) {
		return ""
	}
	start := int(pos.Character)
	end := int(pos.Character)
	for start > 0 && isWordChar(line[start-1]) {
		start--
	}
	for end < len(line) && isWordChar(line[end]) {
		end++
	}
	return line[start:end]
}

func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

var operators = []string{
	":=", "$=", "=>", "->",
	"+=", "-=", "*=", "/=", "%=",
	"==", "!=", "<=", ">=",
}

func (s *Server) getOperatorAtPosition(content string, pos protocol.Position) string {
	lines := strings.Split(content, "\n")
	if int(pos.Line) >= len(lines) {
		return ""
	}
	line := lines[pos.Line]
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
	switch op {
	case ":=":
		return doc.New(
			doc.NewTitleWithKind(":=", "Operator"),
			doc.Paragraph("Declares and initializes a new local variable."),
			doc.NewDivider(),
			doc.NewArcCode("x := 42\nname := \"hello\""),
			doc.NewDivider(),
			doc.Paragraph("The variable type is inferred from the right-hand side expression."),
		).Render()
	case "$=":
		return doc.New(
			doc.NewTitleWithKind("$=", "Operator"),
			doc.Paragraph("Declares a stateful variable that persists across executions."),
			doc.NewDivider(),
			doc.NewArcCode("count $= 0\ncount = count + 1"),
			doc.NewDivider(),
			doc.Paragraph("Stateful variables retain their values between reactive stage executions, making them useful for counters, accumulators, and maintaining state."),
		).Render()
	case "=>":
		return doc.New(
			doc.NewTitleWithKind("=>", "Operator"),
			doc.Paragraph("Transitions to another stage in a sequence."),
			doc.NewDivider(),
			doc.NewArcCode("sequence main {\n    stage first {\n        if ready => second\n    }\n    stage second {}\n}"),
			doc.NewDivider(),
			doc.Paragraph("When the condition is true, execution transitions to the specified stage on the next cycle."),
		).Render()
	case "->":
		return doc.New(
			doc.NewTitleWithKind("->", "Operator"),
			doc.Paragraph("Writes a value to a channel."),
			doc.NewDivider(),
			doc.NewArcCode("value -> outputChannel"),
			doc.NewDivider(),
			doc.Paragraph("Sends the left-hand value to the channel on the right."),
		).Render()
	case "+=":
		return doc.New(
			doc.NewTitleWithKind("+=", "Operator"),
			doc.Paragraph("Adds and assigns."),
			doc.NewDivider(),
			doc.NewArcCode("x += 5  // equivalent to: x = x + 5"),
		).Render()
	case "-=":
		return doc.New(
			doc.NewTitleWithKind("-=", "Operator"),
			doc.Paragraph("Subtracts and assigns."),
			doc.NewDivider(),
			doc.NewArcCode("x -= 5  // equivalent to: x = x - 5"),
		).Render()
	case "*=":
		return doc.New(
			doc.NewTitleWithKind("*=", "Operator"),
			doc.Paragraph("Multiplies and assigns."),
			doc.NewDivider(),
			doc.NewArcCode("x *= 2  // equivalent to: x = x * 2"),
		).Render()
	case "/=":
		return doc.New(
			doc.NewTitleWithKind("/=", "Operator"),
			doc.Paragraph("Divides and assigns."),
			doc.NewDivider(),
			doc.NewArcCode("x /= 2  // equivalent to: x = x / 2"),
		).Render()
	case "%=":
		return doc.New(
			doc.NewTitleWithKind("%=", "Operator"),
			doc.Paragraph("Computes modulo and assigns."),
			doc.NewDivider(),
			doc.NewArcCode("x %= 3  // equivalent to: x = x % 3"),
		).Render()
	case "==":
		return doc.New(
			doc.NewTitleWithKind("==", "Operator"),
			doc.Paragraph("Tests equality between two values."),
			doc.NewDivider(),
			doc.NewArcCode("if x == 10 { ... }"),
		).Render()
	case "!=":
		return doc.New(
			doc.NewTitleWithKind("!=", "Operator"),
			doc.Paragraph("Tests inequality between two values."),
			doc.NewDivider(),
			doc.NewArcCode("if x != 0 { ... }"),
		).Render()
	case "<=":
		return doc.New(
			doc.NewTitleWithKind("<=", "Operator"),
			doc.Paragraph("Tests if left value is less than or equal to right value."),
			doc.NewDivider(),
			doc.NewArcCode("if x <= 100 { ... }"),
		).Render()
	case ">=":
		return doc.New(
			doc.NewTitleWithKind(">=", "Operator"),
			doc.Paragraph("Tests if left value is greater than or equal to right value."),
			doc.NewDivider(),
			doc.NewArcCode("if x >= 0 { ... }"),
		).Render()
	default:
		return ""
	}
}

func (s *Server) getHoverContents(word string) string {
	switch word {
	case "func":
		return doc.New(
			doc.NewTitleWithKind("func", "Keyword"),
			doc.Paragraph("Declares a function."),
			doc.NewDivider(),
			doc.NewArcCode("func name(param type) returnType {\n    // body\n}"),
		).Render()
	case "stage":
		return doc.New(
			doc.NewTitleWithKind("stage", "Keyword"),
			doc.Paragraph("Declares a stage within a sequence."),
			doc.NewDivider(),
			doc.NewArcCode("sequence name {\n    stage stageName {\n        // body\n    }\n}"),
		).Render()
	case "sequence":
		return doc.New(
			doc.NewTitleWithKind("sequence", "Keyword"),
			doc.Paragraph("Declares a sequence (state machine)."),
			doc.NewDivider(),
			doc.NewArcCode("sequence name {\n    stage first {\n        // initial stage\n    }\n}"),
		).Render()
	case "if":
		return doc.New(
			doc.NewTitleWithKind("if", "Keyword"),
			doc.Paragraph("Conditional statement."),
			doc.NewDivider(),
			doc.NewArcCode("if condition {\n    // body\n}"),
		).Render()
	case "else":
		return doc.New(
			doc.NewTitleWithKind("else", "Keyword"),
			doc.Paragraph("Alternative branch for if statement."),
			doc.NewDivider(),
			doc.NewArcCode("if condition {\n    // body\n} else {\n    // alternative\n}"),
		).Render()
	case "return":
		return doc.New(
			doc.NewTitleWithKind("return", "Keyword"),
			doc.Paragraph("Returns a value from a function."),
		).Render()
	case "next":
		return doc.New(
			doc.NewTitleWithKind("next", "Keyword"),
			doc.Paragraph("Transitions to a stage unconditionally."),
			doc.NewDivider(),
			doc.NewArcCode("stage first {\n    condition => next\n}"),
		).Render()
	case "i8", "i16", "i32", "i64":
		bits := word[1:]
		return doc.New(
			doc.NewTitleWithKind(word, "Type"),
			doc.Paragraph(fmt.Sprintf("Signed %s-bit integer.", bits)),
			doc.NewDetail("Range", fmt.Sprintf("-%d to %d", 1<<(parseInt(bits)-1), (1<<(parseInt(bits)-1))-1), false),
		).Render()
	case "u8", "u16", "u32", "u64":
		bits := word[1:]
		return doc.New(
			doc.NewTitleWithKind(word, "Type"),
			doc.Paragraph(fmt.Sprintf("Unsigned %s-bit integer.", bits)),
			doc.NewDetail("Range", fmt.Sprintf("0 to %d", (1<<parseInt(bits))-1), false),
		).Render()
	case "f32":
		return doc.New(
			doc.NewTitleWithKind("f32", "Type"),
			doc.Paragraph("32-bit floating point number (single precision)."),
		).Render()
	case "f64":
		return doc.New(
			doc.NewTitleWithKind("f64", "Type"),
			doc.Paragraph("64-bit floating point number (double precision)."),
		).Render()
	case "string":
		return doc.New(
			doc.NewTitleWithKind("string", "Type"),
			doc.Paragraph("Immutable UTF-8 encoded string."),
		).Render()
	case "timestamp":
		return doc.New(
			doc.NewTitleWithKind("timestamp", "Type"),
			doc.Paragraph("Point in time represented as nanoseconds since Unix epoch."),
		).Render()
	case "timespan":
		return doc.New(
			doc.NewTitleWithKind("timespan", "Type"),
			doc.Paragraph("Duration represented as nanoseconds."),
		).Render()
	case "series":
		return doc.New(
			doc.NewTitleWithKind("series", "Type"),
			doc.Paragraph("Homogeneous array of values."),
			doc.NewDivider(),
			doc.NewArcCode("series f64"),
		).Render()
	case "chan":
		return doc.New(
			doc.NewTitleWithKind("chan", "Type"),
			doc.Paragraph("Bidirectional channel for communication."),
			doc.NewDivider(),
			doc.NewArcCode("chan f64"),
		).Render()
	case "len":
		return doc.New(
			doc.NewTitleWithKind("len", "Function"),
			doc.Paragraph("Returns the length of a series."),
			doc.NewDivider(),
			doc.NewArcCode("length := len(data)"),
		).Render()
	case "now":
		return doc.New(
			doc.NewTitleWithKind("now", "Function"),
			doc.Paragraph("Returns the current timestamp."),
			doc.NewDivider(),
			doc.NewArcCode("time := now()"),
		).Render()
	default:
		return ""
	}
}

func parseInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
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
	tokens := tokenizeContentWithComments(content)
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

func tokenizeContentWithComments(content string) []antlr.Token {
	input := antlr.NewInputStream(content)
	lexer := parser.NewArcLexer(input)
	lexer.RemoveErrorListeners()
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	stream.Fill()
	return stream.GetAllTokens()
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
		d = doc.New(doc.NewTitleWithKind(sym.Name, formatFunctionKindDescription(sym)))
		d.Add(doc.NewDivider())
		d.Add(doc.Code{Language: "arc", Content: formatFunctionSignatureContent(sym)})
	case symbol.KindVariable:
		d = doc.New(doc.NewTitleWithKind(sym.Name, "Variable"))
		d.Add(doc.NewDetail("Type", sym.Type.String(), true))
	case symbol.KindStatefulVariable:
		d = doc.New(doc.NewTitleWithKind(sym.Name, "Stateful Variable"))
		d.Add(doc.Paragraph("Persists across executions"))
		d.Add(doc.NewDetail("Type", sym.Type.String(), true))
	case symbol.KindInput:
		d = doc.New(doc.NewTitleWithKind(sym.Name, "Input Parameter"))
		d.Add(doc.NewDetail("Type", sym.Type.String(), true))
	case symbol.KindOutput:
		d = doc.New(doc.NewTitleWithKind(sym.Name, "Output Parameter"))
		d.Add(doc.NewDetail("Type", sym.Type.String(), true))
	case symbol.KindConfig:
		d = doc.New(doc.NewTitleWithKind(sym.Name, "Configuration Parameter"))
		d.Add(doc.NewDetail("Type", sym.Type.String(), true))
	case symbol.KindChannel:
		d = doc.New(doc.NewTitleWithKind(sym.Name, "Channel"))
		d.Add(doc.NewDetail("Type", sym.Type.String(), true))
	case symbol.KindSequence:
		d = doc.New(doc.NewTitleWithKind(sym.Name, "Sequence"))
		if stages := formatSequenceStagesList(sym); len(stages) > 0 {
			d.Add(doc.Paragraph("Stages: " + strings.Join(stages, ", ")))
		}
	case symbol.KindStage:
		d = doc.New(doc.NewTitleWithKind(sym.Name, "Stage"))
	default:
		d = doc.New(doc.NewTitle(sym.Name))
		d.Add(doc.NewDetail("Type", sym.Type.String(), true))
	}
	if docComment != "" {
		d.Add(doc.NewDivider())
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

func (s *Server) findScopeAtPosition(
	rootScope *symbol.Scope,
	pos protocol.Position,
) *symbol.Scope {
	targetLine := int(pos.Line) + 1
	targetCol := int(pos.Character)
	deepest := rootScope
	s.findScopeAtPositionRecursive(rootScope, targetLine, targetCol, &deepest)
	return deepest
}

func (s *Server) findScopeAtPositionRecursive(
	scope *symbol.Scope,
	line, col int,
	deepest **symbol.Scope,
) {
	if scope.AST != nil {
		start := scope.AST.GetStart()
		stop := scope.AST.GetStop()
		if start != nil && stop != nil {
			startLine := start.GetLine()
			startCol := start.GetColumn()
			stopLine := stop.GetLine()
			stopCol := stop.GetColumn() + len(stop.GetText())
			inRange := false
			if line > startLine && line < stopLine {
				inRange = true
			} else if line == startLine && line == stopLine {
				inRange = col >= startCol && col <= stopCol
			} else if line == startLine {
				inRange = col >= startCol
			} else if line == stopLine {
				inRange = col <= stopCol
			}
			if inRange {
				*deepest = scope
			}
		}
	}
	for _, child := range scope.Children {
		s.findScopeAtPositionRecursive(child, line, col, deepest)
	}
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
