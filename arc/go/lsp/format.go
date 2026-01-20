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

	"github.com/synnaxlabs/arc/formatter"
	"go.lsp.dev/protocol"
)

func (s *Server) Formatting(
	_ context.Context,
	params *protocol.DocumentFormattingParams,
) ([]protocol.TextEdit, error) {
	doc, ok := s.getDocument(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}

	cfg := formatter.Config{
		IndentWidth:       4,
		MaxLineLength:     88,
		MaxBlankLines:     2,
		TrailingCommas:    true,
		AlignDeclarations: true,
	}

	if params.Options.TabSize > 0 {
		cfg.IndentWidth = int(params.Options.TabSize)
	}

	formatted := formatter.Format(doc.Content, cfg)

	if formatted == doc.Content {
		return nil, nil
	}

	lines := splitLines(doc.Content)
	endLine := len(lines) - 1
	endChar := 0
	if endLine >= 0 && len(lines) > 0 {
		endChar = len(lines[endLine])
	}

	return []protocol.TextEdit{
		{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: uint32(endLine), Character: uint32(endChar)},
			},
			NewText: formatted,
		},
	}, nil
}

func splitLines(content string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(content); i++ {
		if content[i] == '\n' {
			if i > 0 && content[i-1] == '\r' {
				lines = append(lines, content[start:i-1])
			} else {
				lines = append(lines, content[start:i])
			}
			start = i + 1
		}
	}
	if start <= len(content) {
		lines = append(lines, content[start:])
	}
	return lines
}

func (s *Server) RangeFormatting(
	_ context.Context,
	params *protocol.DocumentRangeFormattingParams,
) ([]protocol.TextEdit, error) {
	doc, ok := s.getDocument(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}

	cfg := formatter.Config{
		IndentWidth:       4,
		MaxLineLength:     88,
		MaxBlankLines:     2,
		TrailingCommas:    true,
		AlignDeclarations: true,
	}

	if params.Options.TabSize > 0 {
		cfg.IndentWidth = int(params.Options.TabSize)
	}

	startLine := int(params.Range.Start.Line)
	endLine := int(params.Range.End.Line)

	formatted := formatter.FormatRange(doc.Content, startLine, endLine, cfg)

	if formatted == doc.Content {
		return nil, nil
	}

	lines := splitLines(doc.Content)
	docEndLine := len(lines) - 1
	docEndChar := 0
	if docEndLine >= 0 && len(lines) > 0 {
		docEndChar = len(lines[docEndLine])
	}

	return []protocol.TextEdit{
		{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: uint32(docEndLine), Character: uint32(docEndChar)},
			},
			NewText: formatted,
		},
	}, nil
}
