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
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	lsp "github.com/synnaxlabs/x/lsp"
	"github.com/synnaxlabs/x/lsp/doc"
	"github.com/synnaxlabs/x/lsp/protocol"
	"go.uber.org/zap"
)

func arcCode(content string) doc.Block { return doc.Code("arc", content) }

func compoundAssignDoc(sym, verb, op string) string {
	return doc.New(
		doc.TitleWithKind(sym, "Operator"),
		doc.Paragraph(verb+" and assigns."),
		doc.Divider(),
		arcCode(fmt.Sprintf("x %s 5  // equivalent to: x = x %s 5", sym, op)),
	).Render()
}

func intTypeDoc(name, desc, rng string) string {
	return doc.New(
		doc.TitleWithKind(name, "Type"),
		doc.Paragraph(desc),
		doc.Detail("Range", rng, false),
	).Render()
}

func runningStatDoc(name, stat string) string {
	return doc.New(
		doc.TitleWithKind(name, "Function"),
		doc.Paragraph(fmt.Sprintf(
			"Tracks the running %s of input values.", stat,
		)),
		doc.Divider(),
		arcCode(fmt.Sprintf("sensor -> %s{} -> output", name)),
		doc.Divider(),
		doc.Paragraph("Reset after a fixed number of samples or a time window:"),
		doc.Divider(),
		arcCode(fmt.Sprintf(
			"sensor -> %s{count=100} -> output\nsensor -> %s{duration=5s} -> output",
			name, name,
		)),
	).Render()
}

func simpleFuncDoc(name, desc, example string) string {
	return doc.New(
		doc.TitleWithKind(name, "Function"),
		doc.Paragraph(desc),
		doc.Divider(),
		arcCode(example),
	).Render()
}

func deprecatedDoc(old, replacement, example string) string {
	return doc.New(
		doc.TitleWithKind(old, "Function (deprecated)"),
		doc.Paragraph(fmt.Sprintf("Use %s instead.", replacement)),
		doc.Divider(),
		arcCode(example),
	).Render()
}

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

	displayContent := d.displayContent()

	operator := s.getOperatorAtPosition(displayContent, params.Position)
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

	word := d.getWordAtPosition(params.Position)
	if word == "" {
		s.cfg.L.Debug(
			"hover: no word at position",
			zap.Uint32("line", params.Position.Line),
			zap.Uint32("char", params.Position.Character),
		)
		return nil, nil
	}

	qualifiedWord := lsp.GetQualifiedWordAtPosition(
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
	if contents == "" && d.IR.Symbols != nil {
		scopeAtCursor := d.findScopeAtPosition(params.Position)
		contents = s.getUserSymbolHover(
			scopeAtCursor,
			qualifiedWord,
			displayContent,
		)
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
		arcCode("x := 42\nname := \"hello\""),
		doc.Divider(),
		doc.Paragraph("The variable type is inferred from the right-hand side expression."),
	).Render(),
	parser.LiteralSTATEDECLARE: doc.New(
		doc.TitleWithKind(parser.LiteralSTATEDECLARE, "Operator"),
		doc.Paragraph("Declares a stateful variable that persists across executions."),
		doc.Divider(),
		arcCode("count $= 0\ncount = count + 1"),
		doc.Divider(),
		doc.Paragraph("Stateful variables retain their values between reactive stage executions, making them useful for counters, accumulators, and maintaining state."),
	).Render(),
	parser.LiteralTRANSITION: doc.New(
		doc.TitleWithKind(parser.LiteralTRANSITION, "Operator"),
		doc.Paragraph("Transitions to another stage in a sequence."),
		doc.Divider(),
		arcCode("sequence main {\n    stage first {\n        if ready => second\n    }\n    stage second {}\n}"),
		doc.Divider(),
		doc.Paragraph("When the condition is true, execution transitions to the specified stage on the next cycle."),
	).Render(),
	parser.LiteralARROW: doc.New(
		doc.TitleWithKind(parser.LiteralARROW, "Operator"),
		doc.Paragraph("Writes a value to a channel."),
		doc.Divider(),
		arcCode("value -> outputChannel"),
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
		arcCode("if x == 10 { ... }"),
	).Render(),
	parser.LiteralNEQ: doc.New(
		doc.TitleWithKind(parser.LiteralNEQ, "Operator"),
		doc.Paragraph("Tests inequality between two values."),
		doc.Divider(),
		arcCode("if x != 0 { ... }"),
	).Render(),
	parser.LiteralLEQ: doc.New(
		doc.TitleWithKind(parser.LiteralLEQ, "Operator"),
		doc.Paragraph("Tests if left value is less than or equal to right value."),
		doc.Divider(),
		arcCode("if x <= 100 { ... }"),
	).Render(),
	parser.LiteralGEQ: doc.New(
		doc.TitleWithKind(parser.LiteralGEQ, "Operator"),
		doc.Paragraph("Tests if left value is greater than or equal to right value."),
		doc.Divider(),
		arcCode("if x >= 0 { ... }"),
	).Render(),
}

// keywordDocs contains pre-computed documentation for keywords, types, and built-in functions.
var keywordDocs = map[string]string{
	parser.LiteralFUNC: doc.New(
		doc.TitleWithKind(parser.LiteralFUNC, "Keyword"),
		doc.Paragraph("Declares a function."),
		doc.Divider(),
		arcCode("func name(param type) returnType {\n    // body\n}"),
	).Render(),
	parser.LiteralSTAGE: doc.New(
		doc.TitleWithKind(parser.LiteralSTAGE, "Keyword"),
		doc.Paragraph("Declares a stage within a sequence."),
		doc.Divider(),
		arcCode("sequence name {\n    stage stageName {\n        // body\n    }\n}"),
	).Render(),
	parser.LiteralSEQUENCE: doc.New(
		doc.TitleWithKind(parser.LiteralSEQUENCE, "Keyword"),
		doc.Paragraph("Declares a sequence (state machine)."),
		doc.Divider(),
		arcCode("sequence name {\n    stage first {\n        // initial stage\n    }\n}"),
	).Render(),
	parser.LiteralIF: doc.New(
		doc.TitleWithKind(parser.LiteralIF, "Keyword"),
		doc.Paragraph("Conditional statement."),
		doc.Divider(),
		arcCode("if condition {\n    // body\n}"),
	).Render(),
	parser.LiteralELSE: doc.New(
		doc.TitleWithKind(parser.LiteralELSE, "Keyword"),
		doc.Paragraph("Alternative branch for if statement."),
		doc.Divider(),
		arcCode("if condition {\n    // body\n} else {\n    // alternative\n}"),
	).Render(),
	parser.LiteralRETURN: doc.New(
		doc.TitleWithKind(parser.LiteralRETURN, "Keyword"),
		doc.Paragraph("Returns a value from a function."),
	).Render(),
	parser.LiteralNEXT: doc.New(
		doc.TitleWithKind(parser.LiteralNEXT, "Keyword"),
		doc.Paragraph("Transitions to a stage unconditionally."),
		doc.Divider(),
		arcCode("stage first {\n    next second\n}"),
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
		arcCode("series f64"),
	).Render(),
	parser.LiteralCHAN: doc.New(
		doc.TitleWithKind(parser.LiteralCHAN, "Type"),
		doc.Paragraph("Bidirectional channel for communication."),
		doc.Divider(),
		arcCode("chan f64"),
	).Render(),
	parser.LiteralAUTHORITY: doc.New(
		doc.TitleWithKind(parser.LiteralAUTHORITY, "Keyword"),
		doc.Paragraph("Declares the initial control authority for write channels. Authority determines which writer takes priority when multiple writers target the same channel. Higher values take precedence (range 0-255)."),
		doc.Divider(),
		arcCode("authority 200"),
		doc.Divider(),
		doc.Paragraph("Use a grouped block to set per-channel authority:"),
		doc.Divider(),
		arcCode("authority (\n    200\n    valve_cmd 255\n)"),
		doc.Divider(),
		doc.Paragraph("Must appear before all function, flow, and sequence declarations."),
	).Render(),
	"set_authority": deprecatedDoc("set_authority", "authority.set{}", "authority.set{value=255}"),
	"authority.set": doc.New(
		doc.TitleWithKind("authority.set", "Function"),
		doc.Paragraph("Dynamically changes the control authority of write channels at runtime."),
		doc.Divider(),
		arcCode("authority.set{value=255}"),
		doc.Divider(),
		doc.Paragraph("Set authority for a specific channel:"),
		doc.Divider(),
		arcCode("authority.set{value=255, channel=valve_cmd}"),
		doc.Divider(),
		doc.Paragraph("Authority is a u8 (0-255). Higher values take priority. Setting authority to 0 releases control of the channel."),
	).Render(),
	"set_status": deprecatedDoc("set_status", "status.set{}", "sensor -> status.set{status_key=\"ox_alarm\", variant=\"error\", message=\"Overpressure\"}"),
	"status.set": doc.New(
		doc.TitleWithKind("status.set", "Function"),
		doc.Paragraph("Sets a status notification on the cluster. Used to report alarms, warnings, or operational state."),
		doc.Divider(),
		arcCode("sensor -> status.set{status_key=\"ox_alarm\", variant=\"error\", message=\"Overpressure\"}"),
		doc.Divider(),
		doc.Paragraph("Accepted variants: success, error, warning, info."),
	).Render(),
	"math.avg": doc.New(
		doc.TitleWithKind("math.avg", "Function"),
		doc.Paragraph("Computes a running average of input values."),
		doc.Divider(),
		arcCode("sensor -> math.avg{} -> output"),
		doc.Divider(),
		doc.Paragraph("Reset after a fixed number of samples or a time window:"),
		doc.Divider(),
		arcCode("sensor -> math.avg{count=100} -> output\nsensor -> math.avg{duration=5s} -> output"),
		doc.Divider(),
		doc.Paragraph("An optional reset input clears the accumulated average:"),
		doc.Divider(),
		arcCode("sensor -> math.avg{} -> output\nreset_signal -> math.avg{}.reset"),
	).Render(),
	"math.min":        runningStatDoc("math.min", "minimum"),
	"math.max":        runningStatDoc("math.max", "maximum"),
	"math.derivative": simpleFuncDoc("math.derivative", "Computes the rate of change (derivative) of input values. Output is always f64.", "sensor -> math.derivative{} -> rate_output"),
	"selector.select": simpleFuncDoc("selector.select", "Routes input values to 'true' or 'false' outputs. Values equal to 1 are routed to the true output; all others to false.", "flag -> selector.select{} -> {\n    true: open_valve,\n    false: shut_valve\n}"),
	"stable.for":      simpleFuncDoc("stable.for", "Emits a value only after it has remained stable for a specified duration. Prevents spurious signals from transient fluctuations.", "sensor -> stable.for{duration=5s} -> output"),
	"stable_for":      deprecatedDoc("stable_for", "stable.for{}", "sensor -> stable.for{duration=5s} -> output"),
	"len":             simpleFuncDoc("len", "Returns the length of a series or string as i64.", "length := len(data)"),
	"time.now":        simpleFuncDoc("time.now", "Returns the current timestamp.", "t := time.now()"),
	"now":             deprecatedDoc("now", "time.now()", "t := time.now()"),
	"time.interval":   simpleFuncDoc("time.interval", "Fires repeatedly at a specified period.", "time.interval{period=1s} -> tick"),
	"interval":        deprecatedDoc("interval", "time.interval{}", "time.interval{period=1s} -> tick"),
	"time.wait":       simpleFuncDoc("time.wait", "Fires once after a specified duration.", "time.wait{duration=500ms} -> done"),
	"wait":            deprecatedDoc("wait", "time.wait{}", "time.wait{duration=500ms} -> done"),
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
		d.Add(arcCode(formatFunctionSignatureContent(sym)))
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
			_, _ = fmt.Fprintf(&sig, "\n    %s %s", param.Name, param.Type)
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
			_, _ = fmt.Fprintf(&sig, "%s %s", param.Name, param.Type)
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
				_, _ = fmt.Fprintf(&sig, "\n    %s %s", param.Name, param.Type)
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
