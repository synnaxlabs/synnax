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
	"strings"

	"go.lsp.dev/protocol"
)

// PositionToOffset converts an LSP line/character position to a byte offset
// within the given content string.
func PositionToOffset(content string, pos protocol.Position) int {
	line := int(pos.Line)
	char := int(pos.Character)
	offset := 0
	for i := 0; i < line; i++ {
		idx := strings.IndexByte(content[offset:], '\n')
		if idx < 0 {
			return len(content)
		}
		offset += idx + 1
	}
	offset += char
	if offset > len(content) {
		return len(content)
	}
	return offset
}

// IsFullReplacement detects whether a content change event represents a
// full-document replacement (no range specified).
func IsFullReplacement(
	change protocol.TextDocumentContentChangeEvent,
) bool {
	return change.Range == (protocol.Range{}) && change.RangeLength == 0
}

// ApplyIncrementalChange splices a single incremental change into the
// document content and returns the updated string.
func ApplyIncrementalChange(
	content string,
	change protocol.TextDocumentContentChangeEvent,
) string {
	start := PositionToOffset(content, change.Range.Start)
	end := PositionToOffset(content, change.Range.End)
	var b strings.Builder
	b.Grow(start + len(change.Text) + len(content) - end)
	b.WriteString(content[:start])
	b.WriteString(change.Text)
	b.WriteString(content[end:])
	return b.String()
}

// SplitLines normalizes \r\n to \n and splits the content into lines.
func SplitLines(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	return strings.Split(content, "\n")
}

// GetLine returns the line at the given 0-indexed line number.
func GetLine(content string, line uint32) (string, bool) {
	lines := SplitLines(content)
	if int(line) >= len(lines) {
		return "", false
	}
	return lines[line], true
}

// IsWordChar reports whether a byte is a word character ([a-zA-Z0-9_]).
func IsWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_'
}

// GetWordAtPosition extracts the word at the given position in the content.
func GetWordAtPosition(content string, pos protocol.Position) string {
	line, ok := GetLine(content, pos.Line)
	if !ok || int(pos.Character) >= len(line) {
		return ""
	}
	start := int(pos.Character)
	end := int(pos.Character)
	for start > 0 && IsWordChar(line[start-1]) {
		start--
	}
	for end < len(line) && IsWordChar(line[end]) {
		end++
	}
	return line[start:end]
}

// GetWordRangeAtPosition returns the range of the word at the given position,
// or nil if there is no word at that position.
func GetWordRangeAtPosition(
	content string,
	pos protocol.Position,
) *protocol.Range {
	word := GetWordAtPosition(content, pos)
	if word == "" {
		return nil
	}
	line, ok := GetLine(content, pos.Line)
	if !ok || int(pos.Character) >= len(line) {
		return nil
	}
	start := int(pos.Character)
	for start > 0 && IsWordChar(line[start-1]) {
		start--
	}
	return &protocol.Range{
		Start: protocol.Position{Line: pos.Line, Character: uint32(start)},
		End:   protocol.Position{Line: pos.Line, Character: uint32(start + len(word))},
	}
}
