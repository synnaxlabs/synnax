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
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	lsp "github.com/synnaxlabs/x/lsp"
	"github.com/synnaxlabs/x/lsp/doc"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"go.uber.org/zap"
)

func compoundAssignDoc(sym, verb, op string) string {
	return doc.New(
		doc.TitleWithKind(sym, "Operator"),
		doc.Paragraph(verb+" and assigns."),
		doc.Divider(),
		doc.Code("arc", fmt.Sprintf("x %s 5  // equivalent to: x = x %s 5", sym, op)),
	).Render()
}

func intTypeDoc(name, desc, rng string) string {
	return doc.New(
		doc.TitleWithKind(name, "Type"),
		doc.Paragraph(desc),
		doc.Detail("Range", rng, false),
	).Render()
}

func (s *Server) Hover(
	ctx context.Context,
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

	displayContent := d.displayContent()

	operator := s.getOperatorAtPosition(displayContent, params.Position)
	if operator != "" {
		contents := s.getOperatorHoverContents(operator)
		if contents != "" {
			return &protocol.Hover{
				Contents: &protocol.MarkupContent{
					Kind:  protocol.MarkupKindMarkdown,
					Value: contents,
				},
			}, nil
		}
	}

	word := d.getWordAtPosition(params.Position)
	if word == "" {
		s.cfg.L.Debug(
			"hover: no word at position",
			zap.Uint32("line", params.Position.Line),
			zap.Uint32("char", params.Position.Character),
		)
		return nil, nil
	}

	qualifiedWord := lsp.GetQualifiedPrefixWordAtPosition(
		displayContent,
		params.Position,
	)
	var contents string
	if qualifiedWord != word {
		contents = s.getHoverContents(qualifiedWord)
	}
	if contents == "" {
		contents = s.getHoverContents(word)
	}
	if contents == "" {
		var scopeAtCursor *symbol.Symbol
		if d.IR.Symbols != nil {
			scopeAtCursor = d.findScopeAtPosition(params.Position)
		} else {
			scopeAtCursor = s.cfg.NewRoot()
		}
		contents = s.getUserSymbolHover(
			ctx,
			scopeAtCursor,
			qualifiedWord,
			displayContent,
		)
	}

	if contents == "" {
		return nil, nil
	}

	return &protocol.Hover{
		Contents: &protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: contents,
		},
	}, nil
}

var operators = []string{
	parser.LiteralDECLARE, parser.LiteralSTATEDECLARE, parser.LiteralTRANSITION, parser.LiteralARROW,
	parser.LiteralPLUSASSIGN, parser.LiteralMINUSASSIGN, parser.LiteralSTARASSIGN, parser.LiteralSLASHASSIGN, parser.LiteralPERCENTASSIGN,
	parser.LiteralEQ, parser.LiteralNEQ, parser.LiteralLEQ, parser.LiteralGEQ,
}

// operatorDocs contains pre-computed documentation for operators.
var operatorDocs = map[string]string{
	parser.LiteralDECLARE: doc.New(
		doc.TitleWithKind(parser.LiteralDECLARE, "Operator"),
		doc.Paragraph("Declares and initializes a new local variable."),
		doc.Divider(),
		doc.Code("arc", "x := 42\nname := \"hello\""),
		doc.Divider(),
		doc.Paragraph("The variable type is inferred from the right-hand side expression."),
	).Render(),
	parser.LiteralSTATEDECLARE: doc.New(
		doc.TitleWithKind(parser.LiteralSTATEDECLARE, "Operator"),
		doc.Paragraph("Declares a stateful variable that persists across executions."),
		doc.Divider(),
		doc.Code("arc", "count $= 0\ncount = count + 1"),
		doc.Divider(),
		doc.Paragraph("Stateful variables retain their values between reactive stage executions, making them useful for counters, accumulators, and maintaining state."),
	).Render(),
	parser.LiteralTRANSITION: doc.New(
		doc.TitleWithKind(parser.LiteralTRANSITION, "Operator"),
		doc.Paragraph("Transitions to another stage in a sequence."),
		doc.Divider(),
		doc.Code("arc", "sequence main {\n    stage first {\n        if ready => second\n    }\n    stage second {}\n}"),
		doc.Divider(),
		doc.Paragraph("When the condition is true, execution transitions to the specified stage on the next cycle."),
	).Render(),
	parser.LiteralARROW: doc.New(
		doc.TitleWithKind(parser.LiteralARROW, "Operator"),
		doc.Paragraph("Writes a value to a channel."),
		doc.Divider(),
		doc.Code("arc", "value -> outputChannel"),
		doc.Divider(),
		doc.Paragraph("Sends the left-hand value to the channel on the right."),
	).Render(),
	parser.LiteralPLUSASSIGN:    compoundAssignDoc(parser.LiteralPLUSASSIGN, "Adds", "+"),
	parser.LiteralMINUSASSIGN:   compoundAssignDoc(parser.LiteralMINUSASSIGN, "Subtracts", "-"),
	parser.LiteralSTARASSIGN:    compoundAssignDoc(parser.LiteralSTARASSIGN, "Multiplies", "*"),
	parser.LiteralSLASHASSIGN:   compoundAssignDoc(parser.LiteralSLASHASSIGN, "Divides", "/"),
	parser.LiteralPERCENTASSIGN: compoundAssignDoc(parser.LiteralPERCENTASSIGN, "Computes modulo", "%"),
	parser.LiteralEQ: doc.New(
		doc.TitleWithKind(parser.LiteralEQ, "Operator"),
		doc.Paragraph("Tests equality between two values."),
		doc.Divider(),
		doc.Code("arc", "if x == 10 { ... }"),
	).Render(),
	parser.LiteralNEQ: doc.New(
		doc.TitleWithKind(parser.LiteralNEQ, "Operator"),
		doc.Paragraph("Tests inequality between two values."),
		doc.Divider(),
		doc.Code("arc", "if x != 0 { ... }"),
	).Render(),
	parser.LiteralLEQ: doc.New(
		doc.TitleWithKind(parser.LiteralLEQ, "Operator"),
		doc.Paragraph("Tests if left value is less than or equal to right value."),
		doc.Divider(),
		doc.Code("arc", "if x <= 100 { ... }"),
	).Render(),
	parser.LiteralGEQ: doc.New(
		doc.TitleWithKind(parser.LiteralGEQ, "Operator"),
		doc.Paragraph("Tests if left value is greater than or equal to right value."),
		doc.Divider(),
		doc.Code("arc", "if x >= 0 { ... }"),
	).Render(),
}

// keywordDocs contains pre-computed documentation for keywords, types, and built-in functions.
var keywordDocs = map[string]string{
	parser.LiteralFUNC: doc.New(
		doc.TitleWithKind(parser.LiteralFUNC, "Keyword"),
		doc.Paragraph("Declares a function."),
		doc.Divider(),
		doc.Code("arc", "func name(param type) returnType {\n    // body\n}"),
	).Render(),
	parser.LiteralSTAGE: doc.New(
		doc.TitleWithKind(parser.LiteralSTAGE, "Keyword"),
		doc.Paragraph("Declares a stage within a sequence."),
		doc.Divider(),
		doc.Code("arc", "sequence name {\n    stage stageName {\n        // body\n    }\n}"),
	).Render(),
	parser.LiteralSEQUENCE: doc.New(
		doc.TitleWithKind(parser.LiteralSEQUENCE, "Keyword"),
		doc.Paragraph("Declares a sequence (state machine)."),
		doc.Divider(),
		doc.Code("arc", "sequence name {\n    stage first {\n        // initial stage\n    }\n}"),
	).Render(),
	parser.LiteralIF: doc.New(
		doc.TitleWithKind(parser.LiteralIF, "Keyword"),
		doc.Paragraph("Conditional statement."),
		doc.Divider(),
		doc.Code("arc", "if condition {\n    // body\n}"),
	).Render(),
	parser.LiteralELSE: doc.New(
		doc.TitleWithKind(parser.LiteralELSE, "Keyword"),
		doc.Paragraph("Alternative branch for if statement."),
		doc.Divider(),
		doc.Code("arc", "if condition {\n    // body\n} else {\n    // alternative\n}"),
	).Render(),
	parser.LiteralRETURN: doc.New(
		doc.TitleWithKind(parser.LiteralRETURN, "Keyword"),
		doc.Paragraph("Returns a value from a function."),
	).Render(),
	parser.LiteralNEXT: doc.New(
		doc.TitleWithKind(parser.LiteralNEXT, "Keyword"),
		doc.Paragraph("Transitions to a stage unconditionally."),
		doc.Divider(),
		doc.Code("arc", "stage first {\n    next second\n}"),
	).Render(),
	parser.LiteralI8:  intTypeDoc(parser.LiteralI8, "Signed 8-bit integer.", "-128 to 127"),
	parser.LiteralI16: intTypeDoc(parser.LiteralI16, "Signed 16-bit integer.", "-32768 to 32767"),
	parser.LiteralI32: intTypeDoc(parser.LiteralI32, "Signed 32-bit integer.", "-2147483648 to 2147483647"),
	parser.LiteralI64: intTypeDoc(parser.LiteralI64, "Signed 64-bit integer.", "-9223372036854775808 to 9223372036854775807"),
	parser.LiteralU8:  intTypeDoc(parser.LiteralU8, "Unsigned 8-bit integer.", "0 to 255"),
	parser.LiteralU16: intTypeDoc(parser.LiteralU16, "Unsigned 16-bit integer.", "0 to 65535"),
	parser.LiteralU32: intTypeDoc(parser.LiteralU32, "Unsigned 32-bit integer.", "0 to 4294967295"),
	parser.LiteralU64: intTypeDoc(parser.LiteralU64, "Unsigned 64-bit integer.", "0 to 18446744073709551615"),
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
		doc.Code("arc", "series f64"),
	).Render(),
	parser.LiteralCHAN: doc.New(
		doc.TitleWithKind(parser.LiteralCHAN, "Type"),
		doc.Paragraph("Bidirectional channel for communication."),
		doc.Divider(),
		doc.Code("arc", "chan f64"),
	).Render(),
	parser.LiteralAUTHORITY: doc.New(
		doc.TitleWithKind(parser.LiteralAUTHORITY, "Keyword"),
		doc.Paragraph("Declares the initial control authority for write channels. Authority determines which writer takes priority when multiple writers target the same channel. Higher values take precedence (range 0-255)."),
		doc.Divider(),
		doc.Code("arc", "authority 200"),
		doc.Divider(),
		doc.Paragraph("Use a grouped block to set per-channel authority:"),
		doc.Divider(),
		doc.Code("arc", "authority (\n    200\n    valve_cmd 255\n)"),
		doc.Divider(),
		doc.Paragraph("Must appear before all function, flow, and sequence declarations."),
	).Render(),
	parser.LiteralIMPORT: doc.New(
		doc.TitleWithKind(parser.LiteralIMPORT, "Keyword"),
		doc.Paragraph("Imports modules so their qualified members can be used. A module must be imported before its dotted members (e.g. time.now, control.set_authority) can be referenced."),
		doc.Divider(),
		doc.Code("arc", "import ( time control )"),
		doc.Divider(),
		doc.Paragraph("Aliases rename the qualifier:"),
		doc.Divider(),
		doc.Code("arc", "import ( time as t )"),
	).Render(),
}

func (s *Server) getOperatorAtPosition(content string, pos protocol.Position) string {
	line, ok := lsp.GetLine(content, pos.Line)
	if !ok {
		return ""
	}
	col := int(pos.Character)
	if col >= len(line) {
		return ""
	}
	for _, op := range operators {
		opLen := len(op)
		for startOffset := range opLen {
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

func (s *Server) extractDocComment(content string, sym *symbol.Symbol) string {
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
	for i, t := range slices.Backward(tokens) {
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

func hasCodeBetween(tokens []antlr.Token, fromIndex, targetLine int) bool {
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
		if after, ok := strings.CutPrefix(comment, "//"); ok {
			line := after
			line = strings.TrimPrefix(line, " ")
			lines = append(lines, line)
		} else if after, ok := strings.CutPrefix(comment, "/*"); ok {
			text := after
			text = strings.TrimSuffix(text, "*/")
			text = strings.TrimSpace(text)
			for line := range strings.SplitSeq(text, "\n") {
				line = strings.TrimSpace(line)
				line = strings.TrimPrefix(line, "*")
				line = strings.TrimPrefix(line, " ")
				lines = append(lines, line)
			}
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// resolveDotted resolves a possibly-qualified name like "time.now" by
// splitting on dots and walking: the head is resolved in scope, each
// subsequent segment is resolved against the previous result's children.
// Resolution mirrors user-code resolution (no IncludeInternal), so a
// module that is reachable only through the ambient prelude — i.e. one the
// document has not imported — does not resolve. Hovering a member of an
// unimported module therefore returns nothing, matching the analyzer's
// "undefined" diagnostic instead of rendering docs as if it were valid.
func resolveDotted(
	ctx context.Context,
	scope *symbol.Symbol,
	name string,
) (*symbol.Symbol, error) {
	head, tail, hasDot := strings.Cut(name, ".")
	sym, err := scope.Resolve(ctx, head, symbol.WithoutUsageTracking)
	if err != nil {
		return nil, err
	}
	if !hasDot {
		return sym, nil
	}
	return resolveDotted(ctx, sym, tail)
}

// variableTypeDetail renders a value variable's hover type, tagged by its kind.
func variableTypeDetail(sym *symbol.Symbol) string {
	switch {
	case sym.IsChannelReadWrite():
		return "chan read/write " + sym.Type.UnwrapChan().String()
	case sym.IsReactive():
		return "chan read " + sym.Type.UnwrapChan().String()
	default:
		return sym.Type.String()
	}
}

func (s *Server) getUserSymbolHover(
	ctx context.Context,
	scope *symbol.Symbol,
	name string,
	content string,
) string {
	sym, err := resolveDotted(ctx, scope, name)
	if err != nil {
		return ""
	}

	docComment := s.extractDocComment(content, sym)

	displayName := name
	if displayName == "" {
		displayName = sym.Name
	}

	var d doc.Doc
	switch sym.Kind {
	case symbol.KindFunction:
		kindDesc := formatFunctionKindDescription(sym)
		if sym.Deprecated != nil {
			kindDesc += " (deprecated)"
		}
		d = doc.New(doc.TitleWithKind(displayName, kindDesc))
		d.Add(doc.Divider())
		d.Add(doc.Code("arc", formatFunctionSignatureContent(sym)))
		if sym.Trigger.Target != "" {
			d.Add(doc.Detail("Trigger", sym.Trigger.Target, true))
		}
	case symbol.KindModule, symbol.KindModuleAlias:
		d = doc.New(doc.TitleWithKind(displayName, "Module"))
		if members := formatModuleMembersList(sym); len(members) > 0 {
			d.Add(doc.Paragraph("Members: " + strings.Join(members, ", ")))
		}
	case symbol.KindVariable:
		d = doc.New(doc.TitleWithKind(displayName, "Variable"))
		d.Add(doc.Detail("Type", variableTypeDetail(sym), true))
	case symbol.KindStatefulVariable:
		d = doc.New(doc.TitleWithKind(displayName, "Stateful Variable"))
		d.Add(doc.Paragraph("Persists across executions"))
		d.Add(doc.Detail("Type", sym.Type.String(), true))
	case symbol.KindInput:
		d = doc.New(doc.TitleWithKind(displayName, "Input Parameter"))
		d.Add(doc.Detail("Type", sym.Type.String(), true))
	case symbol.KindOutput:
		d = doc.New(doc.TitleWithKind(displayName, "Output Parameter"))
		d.Add(doc.Detail("Type", sym.Type.String(), true))
	case symbol.KindChannel:
		d = doc.New(doc.TitleWithKind(displayName, "Channel"))
		d.Add(doc.Detail("Type", sym.Type.String(), true))
	case symbol.KindSequence:
		d = doc.New(doc.TitleWithKind(displayName, "Sequence"))
		if stages := formatSequenceStagesList(sym); len(stages) > 0 {
			d.Add(doc.Paragraph("Stages: " + strings.Join(stages, ", ")))
		}
	case symbol.KindStage:
		d = doc.New(doc.TitleWithKind(displayName, "Stage"))
	default:
		d = doc.New(doc.Title(displayName))
		d.Add(doc.Detail("Type", sym.Type.String(), true))
	}
	docSrc := sym
	if sym.Kind == symbol.KindModuleAlias && sym.Target != nil {
		docSrc = sym.Target
	}
	if docSrc.Deprecated != nil {
		d.Add(doc.Divider())
		d.Add(doc.Paragraph(fmt.Sprintf(
			"**Deprecated.** Use `%s` instead.",
			docSrc.Deprecated.QualifiedName(),
		)))
		if blocks := docSrc.Deprecated.Doc.Blocks(); len(blocks) > 0 {
			d.Add(doc.Divider())
			d.Add(blocks...)
		}
	} else if blocks := docSrc.Doc.Blocks(); len(blocks) > 0 {
		d.Add(doc.Divider())
		d.Add(blocks...)
	}
	if docComment != "" {
		d.Add(doc.Divider())
		d.Add(doc.Paragraph(docComment))
	}
	return d.Render()
}

// formatFunctionSignatureContent returns the function signature without code fences.
func formatFunctionSignatureContent(sym *symbol.Symbol) string {
	if sym.Type.Kind != types.KindFunction {
		return ""
	}
	var sig strings.Builder
	sig.WriteString("func ")
	sig.WriteString(sym.Name)
	sig.WriteString("(")
	for i, param := range sym.Type.Inputs {
		if i > 0 {
			sig.WriteString(", ")
		}
		_, _ = fmt.Fprintf(&sig, "%s %s", param.Name, param.Type)
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
				_, _ = fmt.Fprintf(&sig, "\n    %s %s", param.Name, param.Type)
			}
			sig.WriteString("\n}")
		}
	}
	return sig.String()
}

func formatFunctionKindDescription(sym *symbol.Symbol) string {
	if sym.Exec == symbol.ExecFlow || sym.Exec == symbol.ExecBoth {
		return "Node"
	}
	return "Function"
}

// formatSequenceStagesList returns a list of formatted stage names.
func formatSequenceStagesList(sym *symbol.Symbol) []string {
	var stages []string
	for _, child := range sym.Children() {
		if child.Kind == symbol.KindStage {
			stages = append(stages, "`"+child.Name+"`")
		}
	}
	return stages
}

// formatModuleMembersList returns a list of formatted user-visible member
// names for a module symbol (or a module alias, which dispatches to its
// target's children). Internal members are skipped.
func formatModuleMembersList(sym *symbol.Symbol) []string {
	var members []string
	for _, child := range sym.Children() {
		if child.Internal || child.Name == "" {
			continue
		}
		members = append(members, "`"+child.Name+"`")
	}
	return members
}

// symbolToLocation converts a symbol to an LSP Location pointing to its definition
func (s *Server) symbolToLocation(
	docURI uri.URI,
	sym *symbol.Symbol,
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
		URI: docURI,
		Range: protocol.Range{
			Start: protocol.Position{Line: line, Character: col},
			End:   protocol.Position{Line: line, Character: col + uint32(len(sym.Name))},
		},
	}
}
