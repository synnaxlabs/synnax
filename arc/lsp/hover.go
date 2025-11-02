// Copyright 2025 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"go.lsp.dev/protocol"
	"go.uber.org/zap"
)

func (s *Server) Hover(
	_ context.Context,
	params *protocol.HoverParams,
) (*protocol.Hover, error) {
	doc, ok := s.getDocument(params.TextDocument.URI)
	if !ok {
		s.cfg.L.Debug(
			"hover: document not found",
			zap.String("uri", string(params.TextDocument.URI)),
		)
		return nil, nil
	}

	word := s.getWordAtPosition(doc.Content, params.Position)
	if word == "" {
		s.cfg.L.Debug(
			"hover: no word at position",
			zap.Uint32("line", params.Position.Line),
			zap.Uint32("char", params.Position.Character),
		)
		return nil, nil
	}

	contents := s.getHoverContents(word)
	if contents == "" && doc.IR.Symbols != nil {
		scopeAtCursor := s.findScopeAtPosition(doc.IR.Symbols, params.Position)
		contents = s.getUserSymbolHover(scopeAtCursor, word)
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

func (s *Server) getHoverContents(word string) string {
	switch word {
	case "func":
		return "#### func\nDeclares a function.\n\n```arc\nfunc name(param type) returnType {\n    // body\n}\n```"
	case "stage":
		return "#### stage\nDeclares a reactive stage.\n\n```arc\nfunc name{\n    config type\n} (runtime type) returnType {\n    // body\n}\n```"
	case "if":
		return "#### if\nConditional statement.\n\n```arc\nif condition {\n    // body\n}\n```"
	case "else":
		return "#### else\nAlternative branch for if statement.\n\n```arc\nif condition {\n    // body\n} else {\n    // alternative\n}\n```"
	case "return":
		return "#### return\nReturns a value from a function."
	case "i8", "i16", "i32", "i64":
		bits := word[1:]
		return fmt.Sprintf("#### %s\nSigned %s-bit integer.\n\nRange: -%d to %d", word, bits, 1<<(parseInt(bits)-1), (1<<(parseInt(bits)-1))-1)
	case "u8", "u16", "u32", "u64":
		bits := word[1:]
		return fmt.Sprintf("#### %s\nUnsigned %s-bit integer.\n\nRange: 0 to %d", word, bits, (1<<parseInt(bits))-1)
	case "f32":
		return "#### f32\n32-bit floating point number (single precision)."
	case "f64":
		return "#### f64\n64-bit floating point number (double precision)."
	case "string":
		return "#### string\nImmutable UTF-8 encoded string."
	case "timestamp":
		return "#### timestamp\nPoint in time represented as nanoseconds since Unix epoch."
	case "timespan":
		return "#### timespan\nDuration represented as nanoseconds."
	case "series":
		return "#### series\nHomogeneous array of values.\n\n```arc\nseries f64\n```"
	case "chan":
		return "#### chan\nBidirectional channel for communication.\n\n```arc\nchan f64\n```"
	case "len":
		return "#### len\nReturns the length of a series.\n\n```arc\nlength := len(data)\n```"
	case "now":
		return "#### now\nReturns the current timestamp.\n\n```arc\ntime := now()\n```"
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

// getUserSymbolHover returns hover documentation for user-defined symbols
func (s *Server) getUserSymbolHover(scope *symbol.Scope, name string) string {
	sym, err := scope.Resolve(context.Background(), name)
	if err != nil {
		return ""
	}
	var content strings.Builder
	content.WriteString(fmt.Sprintf("## %s\n\n", sym.Name))
	switch sym.Kind {
	case symbol.KindFunction:
		content.WriteString(formatFunctionSignature(sym))
		content.WriteString("\n\n")
		content.WriteString(formatFunctionKindDescription(sym))
	case symbol.KindVariable:
		content.WriteString("**Variable**\n\n")
		content.WriteString(fmt.Sprintf("Type: `%s`", sym.Type))
	case symbol.KindStatefulVariable:
		content.WriteString("**Stateful Variable** (persists across executions)\n\n")
		content.WriteString(fmt.Sprintf("Type: `%s`", sym.Type))
	case symbol.KindInput:
		content.WriteString("**Input Parameter**\n\n")
		content.WriteString(fmt.Sprintf("Type: `%s`", sym.Type))
	case symbol.KindOutput:
		content.WriteString("**Output Parameter**\n\n")
		content.WriteString(fmt.Sprintf("Type: `%s`", sym.Type))
	case symbol.KindConfig:
		content.WriteString("**Configuration Parameter**\n\n")
		content.WriteString(fmt.Sprintf("Type: `%s`", sym.Type))
	case symbol.KindChannel:
		content.WriteString("**Channel**\n\n")
		content.WriteString(fmt.Sprintf("Type: `%s`", sym.Type))
	default:
		content.WriteString(fmt.Sprintf("Type: `%s`", sym.Type))
	}
	return content.String()
}

func formatFunctionSignature(sym *symbol.Scope) string {
	if sym.Type.Kind != types.KindFunction {
		return ""
	}
	var sig strings.Builder
	sig.WriteString("```arc\n")
	sig.WriteString("func ")
	sig.WriteString(sym.Name)
	if sym.Type.Config != nil && len(sym.Type.Config) > 0 {
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
	if sym.Type.Inputs != nil && len(sym.Type.Inputs) > 0 {
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
	if sym.Type.Outputs != nil && len(sym.Type.Outputs) > 0 {
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

	sig.WriteString("\n```")
	return sig.String()
}

func formatFunctionKindDescription(sym *symbol.Scope) string {
	if sym.Type.Config != nil {
		return "_Reactive stage with configuration_"
	}
	return "_Function_"
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
