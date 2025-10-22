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
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"

	"go.lsp.dev/protocol"
)

// DocumentMetadata contains metadata about a document
type DocumentMetadata struct {
	IsBlock bool `json:"is_block"` // If true, treat content as a function body block
}

// WrapperContext tracks the wrapping transformation for position mapping
type WrapperContext struct {
	OriginalContent string
	WrappedContent  string
	LineOffset      int // How many lines were added before expression
	ColumnOffset    int // Column offset (0 for no indentation)
}

// ExtractMetadataFromURI extracts metadata from a URI fragment
// Expected format: arc://block/id#base64(json)
func ExtractMetadataFromURI(uri protocol.DocumentURI) *DocumentMetadata {
	uriStr := string(uri)

	// Check if this is a block URI
	if !strings.HasPrefix(uriStr, "arc://block/") {
		return nil
	}

	// Extract fragment (metadata)
	parts := strings.Split(uriStr, "#")
	if len(parts) < 2 {
		return nil
	}

	// URL decode the fragment (Monaco URI-encodes it: = becomes %3D)
	fragment, err := url.QueryUnescape(parts[1])
	if err != nil {
		return nil
	}

	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(fragment)
	if err != nil {
		return nil
	}

	// Parse JSON
	var metadata DocumentMetadata
	if err := json.Unmarshal(decoded, &metadata); err != nil {
		return nil
	}

	return &metadata
}

// WrapExpression wraps a block expression in a function declaration for LSP
// parsing and analysis. Multi-line expressions are preserved as-is with NO
// indentation to simplify position mapping.
func WrapExpression(content string, metadata *DocumentMetadata) *WrapperContext {
	if metadata == nil || !metadata.IsBlock {
		// Not a block, no wrapping needed - treat as complete program
		return &WrapperContext{
			OriginalContent: content,
			WrappedContent:  content,
			LineOffset:      0,
			ColumnOffset:    0,
		}
	}

	// Build wrapped function WITHOUT parameters or return type or indentation
	// Variables/channels are accessed as globals from the GlobalResolver
	var sb strings.Builder
	sb.WriteString("func __block() {\n")
	sb.WriteString(content) // No indentation!
	if !strings.HasSuffix(content, "\n") {
		sb.WriteString("  \n")
	}
	sb.WriteString("}")

	return &WrapperContext{
		OriginalContent: content,
		WrappedContent:  sb.String(),
		LineOffset:      1, // Function declaration is line 0, expression starts at line 1
		ColumnOffset:    0, // No indentation offset
	}
}

// MapDiagnosticPosition translates a diagnostic position from wrapped content
// back to original expression coordinates
func (w *WrapperContext) MapDiagnosticPosition(pos protocol.Position) protocol.Position {
	// Subtract the wrapper line offset
	originalLine := int(pos.Line) - w.LineOffset

	// If diagnostic is before the expression (in function signature), map to line 0
	if originalLine < 0 {
		return protocol.Position{Line: 0, Character: 0}
	}

	// Column offset should be 0 since we don't indent
	originalColumn := int(pos.Character) - w.ColumnOffset
	if originalColumn < 0 {
		originalColumn = 0
	}

	return protocol.Position{
		Line:      uint32(originalLine),
		Character: uint32(originalColumn),
	}
}

// MapDiagnosticRange translates a diagnostic range from wrapped content back to original
func (w *WrapperContext) MapDiagnosticRange(r protocol.Range) protocol.Range {
	return protocol.Range{
		Start: w.MapDiagnosticPosition(r.Start),
		End:   w.MapDiagnosticPosition(r.End),
	}
}

// MapOriginalToWrapped maps a position from the original expression to the wrapped version
// This is useful for features like hover and completion
func (w *WrapperContext) MapOriginalToWrapped(pos protocol.Position) protocol.Position {
	return protocol.Position{
		Line:      uint32(int(pos.Line) + w.LineOffset),
		Character: uint32(int(pos.Character) + w.ColumnOffset),
	}
}
