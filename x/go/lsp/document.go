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
	"fmt"
	"strings"
	"unicode/utf16"

	"go.lsp.dev/protocol"
)

// PositionToOffset converts an LSP line/character position to a byte offset within
// content. Characters count UTF-16 code units. A character past the end of its line
// resolves to the line end.
func PositionToOffset(content string, pos protocol.Position) int {
	offset := 0
	for range pos.Line {
		idx := strings.IndexByte(content[offset:], '\n')
		if idx < 0 {
			return len(content)
		}
		offset += idx + 1
	}
	line := content[offset:]
	if idx := strings.IndexByte(line, '\n'); idx >= 0 {
		line = line[:idx]
	}
	return offset + CharacterToByte(strings.TrimSuffix(line, "\r"), pos.Character)
}

// CharacterToByte converts a UTF-16 column in line to a byte offset, clamped to the
// line length.
func CharacterToByte(line string, character uint32) int {
	units := 0
	for i, r := range line {
		if units >= int(character) {
			return i
		}
		units += utf16.RuneLen(r)
	}
	return len(line)
}

// byteToCharacter converts a byte offset in line to a UTF-16 column.
func byteToCharacter(line string, offset int) uint32 {
	units := 0
	for _, r := range line[:offset] {
		units += utf16.RuneLen(r)
	}
	return uint32(units)
}

// ApplyIncrementalChange splices a single incremental change into the document content
// and returns the updated string. Panics on a change variant outside the sealed union.
func ApplyIncrementalChange(
	content string,
	change protocol.TextDocumentContentChangeEvent,
) string {
	switch ev := change.(type) {
	case *protocol.TextDocumentContentChangeWholeDocument:
		return ev.Text
	case *protocol.TextDocumentContentChangePartial:
		start := PositionToOffset(content, ev.Range.Start)
		end := PositionToOffset(content, ev.Range.End)
		var b strings.Builder
		b.Grow(start + len(ev.Text) + len(content) - end)
		b.WriteString(content[:start])
		b.WriteString(ev.Text)
		b.WriteString(content[end:])
		return b.String()
	default:
		panic(
			fmt.Sprintf("unhandled TextDocumentContentChangeEvent variant %T", change),
		)
	}
}

// splitLines normalizes \r\n to \n and splits the content into lines.
func splitLines(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	return strings.Split(content, "\n")
}

// GetLine returns the line at the given 0-indexed line number.
func GetLine(content string, line uint32) (string, bool) {
	lines := splitLines(content)
	if int(line) >= len(lines) {
		return "", false
	}
	return lines[line], true
}

// WordChars reports, for each byte of line, whether the byte belongs to a word.
type WordChars func(line string) []bool

// DefaultWordChars marks the bytes [a-zA-Z0-9_].
func DefaultWordChars(line string) []bool {
	marks := make([]bool, len(line))
	for i := range len(line) {
		c := line[i]
		marks[i] = (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '_'
	}
	return marks
}

// GetWordAtPosition extracts the word at the given position in the content.
func GetWordAtPosition(
	content string,
	pos protocol.Position,
	wordChars WordChars,
) string {
	line, start, end := wordSpan(content, pos, wordChars, false)
	return line[start:end]
}

// GetQualifiedPrefixWordAtPosition extracts the qualifier chain ending at
// the cursor word. The left walk crosses dots so a qualifier prefix is
// included; the right walk does not, so the result stops at the end of the
// cursor word. For "time.now" with the cursor on "time", the result is
// "time"; with the cursor on "now", the result is "time.now". This is the
// shape hover and goto-definition want: identify which symbol the cursor
// is *on*, not the whole dotted span the cursor happens to sit inside.
func GetQualifiedPrefixWordAtPosition(
	content string,
	pos protocol.Position,
	wordChars WordChars,
) string {
	line, start, end := wordSpan(content, pos, wordChars, true)
	return line[start:end]
}

// GetWordRangeAtPosition returns the range of the word at the given position,
// or nil if there is no word at that position.
func GetWordRangeAtPosition(
	content string,
	pos protocol.Position,
	wordChars WordChars,
) *protocol.Range {
	line, start, end := wordSpan(content, pos, wordChars, false)
	if start == end {
		return nil
	}
	return &protocol.Range{
		Start: protocol.Position{
			Line:      pos.Line,
			Character: byteToCharacter(line, start),
		},
		End: protocol.Position{
			Line:      pos.Line,
			Character: byteToCharacter(line, end),
		},
	}
}

// wordSpan returns the line at pos and the byte bounds of the word under the cursor.
// When qualified is true, the left walk also crosses dots. The bounds are equal when
// the cursor is past the line end.
func wordSpan(
	content string,
	pos protocol.Position,
	wordChars WordChars,
	qualified bool,
) (line string, start, end int) {
	line, ok := GetLine(content, pos.Line)
	if !ok {
		return "", 0, 0
	}
	start = CharacterToByte(line, pos.Character)
	if start >= len(line) {
		return line, start, start
	}
	marks := wordChars(line)
	end = start
	for start > 0 && (marks[start-1] || (qualified && line[start-1] == '.')) {
		start--
	}
	for end < len(line) && marks[end] {
		end++
	}
	return line, start, end
}
