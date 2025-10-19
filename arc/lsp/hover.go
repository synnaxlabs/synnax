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
	"strings"

	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"go.lsp.dev/protocol"
	"go.uber.org/zap"
)

// Hover handles hover requests
func (s *Server) Hover(_ context.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	s.mu.RLock()
	doc, ok := s.documents[params.TextDocument.URI]
	s.mu.RUnlock()

	if !ok {
		s.cfg.L.Debug("Hover: document not found", zap.String("uri", string(params.TextDocument.URI)))
		return nil, nil
	}

	// Get the word at the cursor position
	word := s.getWordAtPosition(doc.Content, params.Position)
	if word == "" {
		s.cfg.L.Debug("Hover: no word at position", zap.Uint32("line", params.Position.Line), zap.Uint32("char", params.Position.Character))
		return nil, nil
	}

	// Try built-in keywords and types first
	contents := s.getHoverContents(word)

	// If not a built-in, try user-defined symbols from IR
	if contents == "" && doc.IR.Symbols != nil {
		// Find the scope at the cursor position for proper context
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

// getWordAtPosition extracts the word at the given position
func (s *Server) getWordAtPosition(content string, pos protocol.Position) string {
	lines := strings.Split(content, "\n")
	if int(pos.Line) >= len(lines) {
		return ""
	}

	line := lines[pos.Line]
	if int(pos.Character) >= len(line) {
		return ""
	}

	// Find word boundaries
	start := int(pos.Character)
	end := int(pos.Character)

	// Expand left
	for start > 0 && isWordChar(line[start-1]) {
		start--
	}

	// Expand right
	for end < len(line) && isWordChar(line[end]) {
		end++
	}

	return line[start:end]
}

func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

// getHoverContents returns hover documentation for known keywords and types
func (s *Server) getHoverContents(word string) string {
	switch word {
	// Keywords
	case "func":
		return "## func\nDeclares a function.\n\n```arc\nfunc name(param type) returnType {\n    // body\n}\n```"
	case "stage":
		return "## stage\nDeclares a reactive stage.\n\n```arc\nfunc name{\n    config type\n} (runtime type) returnType {\n    // body\n}\n```"
	case "if":
		return "## if\nConditional statement.\n\n```arc\nif condition {\n    // body\n}\n```"
	case "else":
		return "## else\nAlternative branch for if statement.\n\n```arc\nif condition {\n    // body\n} else {\n    // alternative\n}\n```"
	case "return":
		return "## return\nReturns a value from a function or stage."

	// Types
	case "i8", "i16", "i32", "i64":
		bits := word[1:]
		return fmt.Sprintf("## %s\nSigned %s-bit integer.\n\nRange: -%d to %d", word, bits, 1<<(parseInt(bits)-1), (1<<(parseInt(bits)-1))-1)
	case "u8", "u16", "u32", "u64":
		bits := word[1:]
		return fmt.Sprintf("## %s\nUnsigned %s-bit integer.\n\nRange: 0 to %d", word, bits, (1<<parseInt(bits))-1)
	case "f32":
		return "## f32\n32-bit floating point number (single precision)."
	case "f64":
		return "## f64\n64-bit floating point number (double precision)."
	case "string":
		return "## string\nImmutable UTF-8 encoded string."
	case "timestamp":
		return "## timestamp\nPoint in time represented as nanoseconds since Unix epoch."
	case "timespan":
		return "## timespan\nDuration represented as nanoseconds."
	case "series":
		return "## series\nHomogeneous array of values.\n\n```arc\nseries f64\n```"
	case "chan":
		return "## chan\nBidirectional channel for communication.\n\n```arc\nchan f64\n```"

	// Built-in functions
	case "len":
		return "## len\nReturns the length of a series.\n\n```arc\nlength := len(data)\n```"
	case "now":
		return "## now\nReturns the current timestamp.\n\n```arc\ntime := now()\n```"

	default:
		return ""
	}
}

func parseInt(s string) int {
	switch s {
	case "8":
		return 8
	case "16":
		return 16
	case "32":
		return 32
	case "64":
		return 64
	default:
		return 0
	}
}

// getUserSymbolHover returns hover documentation for user-defined symbols
func (s *Server) getUserSymbolHover(scope *symbol.Scope, name string) string {
	// Try to resolve the symbol in the symbol table from the given scope
	// This will search upward through parent scopes automatically
	sym, err := scope.Resolve(context.Background(), name)
	if err != nil {
		return ""
	}

	// Format based on symbol kind
	var content strings.Builder
	content.WriteString(fmt.Sprintf("## %s\n\n", sym.Name))
	switch sym.Kind {
	case symbol.KindFunction:
		content.WriteString(formatFunctionSignature(sym))
		content.WriteString("\n\n")
		content.WriteString(formatFunctionKindDescription(sym))

	case symbol.KindVariable:
		content.WriteString(fmt.Sprintf("**Variable**\n\n"))
		content.WriteString(fmt.Sprintf("Type: `%s`", sym.Type))
	case symbol.KindStatefulVariable:
		content.WriteString("**Stateful Variable** (persists across executions)\n\n")
		content.WriteString(fmt.Sprintf("Type: `%s`", sym.Type))
	case symbol.KindInput:
		content.WriteString("**Input Parameter**\n\n")
		content.WriteString(fmt.Sprintf("Type: `%s`", sym.Type))
	case symbol.KindOutput:
		content.WriteString("Output Parameter**\n\n")
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

// formatFunctionSignature formats a function's signature for hover display
func formatFunctionSignature(sym *symbol.Scope) string {
	if sym.Type.Kind != types.KindFunction {
		return ""
	}

	var sig strings.Builder
	sig.WriteString("```arc\n")

	// All functions use the "func" keyword
	// (stages are just functions with a config block)
	sig.WriteString("func ")
	sig.WriteString(sym.Name)

	// Config block for stages
	if sym.Type.Config != nil && sym.Type.Config.Count() > 0 {
		sig.WriteString("{")
		first := true
		for name, t := range sym.Type.Config.Iter() {
			if !first {
				sig.WriteString(", ")
			}
			sig.WriteString(fmt.Sprintf("\n    %s %s", name, t))
			first = false
		}
		sig.WriteString("\n}")
	}

	// Input parameters
	sig.WriteString("(")
	if sym.Type.Inputs != nil && sym.Type.Inputs.Count() > 0 {
		first := true
		for name, t := range sym.Type.Inputs.Iter() {
			if !first {
				sig.WriteString(", ")
			}
			sig.WriteString(fmt.Sprintf("%s %s", name, t))
			first = false
		}
	}
	sig.WriteString(")")

	// Output type
	if sym.Type.Outputs != nil && sym.Type.Outputs.Count() > 0 {
		sig.WriteString(" ")
		if sym.Type.Outputs.Count() == 1 {
			// Single output - show inline
			_, outputType := sym.Type.Outputs.At(0)
			sig.WriteString(outputType.String())
		} else {
			// Multiple outputs - show block
			sig.WriteString("{")
			for name, t := range sym.Type.Outputs.Iter() {
				sig.WriteString(fmt.Sprintf("\n    %s %s", name, t))
			}
			sig.WriteString("\n}")
		}
	}

	sig.WriteString("\n```")
	return sig.String()
}

// formatFunctionKindDescription returns a description of the function kind
func formatFunctionKindDescription(sym *symbol.Scope) string {
	// A stage is a function with a config block (even if empty)
	if sym.Type.Config != nil {
		return "_Reactive stage with configuration_"
	}
	return "_Function_"
}

// findScopeAtPosition finds the deepest scope that contains the given position
// This allows us to resolve symbols from the correct context (e.g., inside a function body)
func (s *Server) findScopeAtPosition(rootScope *symbol.Scope, pos protocol.Position) *symbol.Scope {
	// Convert LSP position to 1-based line number (LSP is 0-based, ANTLR is 1-based)
	targetLine := int(pos.Line) + 1
	targetCol := int(pos.Character)

	// Recursively find the deepest matching scope
	deepest := rootScope
	s.findScopeAtPositionRecursive(rootScope, targetLine, targetCol, &deepest)
	return deepest
}

// findScopeAtPositionRecursive recursively searches for the deepest scope containing the position
func (s *Server) findScopeAtPositionRecursive(scope *symbol.Scope, line, col int, deepest **symbol.Scope) {
	// Check if this scope's AST contains the position
	if scope.AST != nil {
		start := scope.AST.GetStart()
		stop := scope.AST.GetStop()

		if start != nil && stop != nil {
			startLine := start.GetLine()
			startCol := start.GetColumn()
			stopLine := stop.GetLine()
			stopCol := stop.GetColumn() + len(stop.GetText())

			// Check if position is within this scope's range
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

			// If this scope contains the position, it's a candidate
			if inRange {
				*deepest = scope
			}
		}
	}

	// Recursively check children (depth-first to find the deepest match)
	for _, child := range scope.Children {
		s.findScopeAtPositionRecursive(child, line, col, deepest)
	}
}

// symbolToLocation converts a symbol to an LSP Location pointing to its definition
func (s *Server) symbolToLocation(uri protocol.DocumentURI, sym *symbol.Scope) *protocol.Location {
	// Get the AST node for this symbol
	if sym.AST == nil {
		return nil
	}

	start := sym.AST.GetStart()
	if start == nil {
		return nil
	}

	// Convert ANTLR position (1-based line, 0-based column) to LSP position (0-based)
	line := uint32(start.GetLine() - 1)
	col := uint32(start.GetColumn())

	return &protocol.Location{
		URI: uri,
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      line,
				Character: col,
			},
			End: protocol.Position{
				Line:      line,
				Character: col + uint32(len(sym.Name)),
			},
		},
	}
}
