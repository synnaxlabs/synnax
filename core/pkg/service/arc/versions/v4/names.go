// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4

import (
	"strconv"
	"strings"
)

// QuoteDashedNames returns text, an Arc program or calculation expression that a v0.62
// Core stored, with each dashed name that held reports true for written as a quoted
// channel token. A dashed name is an identifier that holds '-', as v0.62 lexed it with
// dashed names on. QuoteDashedNames reads code and the placeholders of format strings,
// and leaves comments, the rest of strings, and quoted tokens as they are. A name that
// a placeholder writes with an escape keeps the escape inside the quotes.
func QuoteDashedNames(text string, held func(string) bool) string {
	var edits []span
	scanCode(text, offsets{}, func(s span, name string) {
		if strings.IndexByte(name, '-') >= 0 && held(name) {
			edits = append(edits, s)
		}
	})
	if len(edits) == 0 {
		return text
	}
	var b strings.Builder
	b.Grow(len(text) + 2*len(edits))
	last := 0
	for _, e := range edits {
		// A name written in code holds no quote or backslash, and one in a placeholder
		// keeps the escapes the format string removes before it lexes the name.
		b.WriteString(text[last:e.start])
		b.WriteByte('\'')
		b.WriteString(text[e.start:e.end])
		b.WriteByte('\'')
		last = e.end
	}
	b.WriteString(text[last:])
	return b.String()
}

// span is the byte range [start, end) of an identifier.
type span struct{ start, end int }

// offsets maps a byte offset in a source to its offset in the enclosing text. A nil at
// maps j to base + j. Otherwise at holds the offset of each byte and of the end.
type offsets struct {
	at   []int
	base int
}

func (o offsets) pos(j int) int {
	if o.at != nil {
		return o.at[j]
	}
	return o.base + j
}

// sub returns the offsets of the source that starts at i in the source of o.
func (o offsets) sub(i int) offsets {
	if o.at != nil {
		return offsets{at: o.at[i:]}
	}
	return offsets{base: o.base + i}
}

// scanCode calls found with the span in the enclosing text of each identifier in the
// code src, in order, and with the name that the identifier reads. off maps src into
// the enclosing text. It follows the v0.62 lexer with dashed names on, and the quoted
// channel token of v63.
func scanCode(src string, off offsets, found func(span, string)) {
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case strings.HasPrefix(src[i:], "//"):
			if end := strings.IndexAny(src[i:], "\r\n"); end >= 0 {
				i += end
			} else {
				i = len(src)
			}
		case strings.HasPrefix(src[i:], "/*"):
			if end := strings.Index(src[i+2:], "*/"); end >= 0 {
				i += end + 4
			} else {
				i++
			}
		case c == '\'':
			if n, ok := quotedLen(src[i:]); ok {
				i += n
			} else {
				i += skip(src[i:], n)
			}
		case c == '"' || c == '`' || c == 'r' || c == 'f':
			n, ok := stringLen(src[i:])
			switch {
			case ok:
				scanFormatString(src[i:i+n], off.sub(i), found)
				i += n
			case c == 'r' || c == 'f':
				i = scanIdentifier(src, i, off, found)
			default:
				i += skip(src[i:], n)
			}
		case c == '$' || c == '!':
			// Each starts a token only before '=', so the error recovery of a lone one
			// drops the character after it too.
			i = min(i+2, len(src))
		case isIdentifierByte(c) && (c < '0' || c > '9'):
			i = scanIdentifier(src, i, off, found)
		default:
			i++
		}
	}
}

// scanIdentifier calls found with the identifier that starts at src[i], and returns its
// end. A '-' joins an identifier unless '>' or '=' comes after it.
func scanIdentifier(src string, i int, off offsets, found func(span, string)) int {
	end := i + 1
	for end < len(src) {
		c := src[end]
		if c == '-' {
			if end+1 < len(src) && (src[end+1] == '>' || src[end+1] == '=') {
				break
			}
		} else if !isIdentifierByte(c) {
			break
		}
		end++
	}
	found(span{start: off.pos(i), end: off.pos(end)}, src[i:end])
	return end
}

func isIdentifierByte(c byte) bool {
	return c == '_' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' ||
		'0' <= c && c <= '9'
}

// quotedLen returns the length of the quoted channel token at the start of src, and
// true. When none starts there, it returns where the token stopped matching, and false.
func quotedLen(src string) (int, bool) {
	for i := 1; i < len(src); i++ {
		switch src[i] {
		case '\'':
			if i == 1 {
				return i, false
			}
			return i + 1, true
		case '\r', '\n':
			return i, false
		case '\\':
			if i+1 < len(src) && src[i+1] != '\'' && src[i+1] != '\\' {
				return i + 1, false
			}
			i++
		}
	}
	return len(src), false
}

// stringLen returns the length of the string literal at the start of src, with its r,
// f, rf, or fr prefix, and true. When none starts there, it returns where the literal
// stopped matching, and false.
func stringLen(src string) (int, bool) {
	prefix := 0
	switch {
	case strings.HasPrefix(src, "rf"), strings.HasPrefix(src, "fr"):
		prefix = 2
	case strings.HasPrefix(src, "r"), strings.HasPrefix(src, "f"):
		prefix = 1
	}
	if !isQuote(src, prefix) {
		return prefix, false
	}
	quote := src[prefix]
	for i := prefix + 1; i < len(src); i++ {
		switch c := src[i]; {
		case c == quote:
			return i + 1, true
		case c == '\\':
			i++
		case quote == '"' && (c == '\r' || c == '\n'):
			return i, false
		}
	}
	return len(src), false
}

// skip returns how far the v0.62 lexer moved past a token that stopped matching at stop
// in src: past the character at stop, which its error recovery drops.
func skip(src string, stop int) int { return min(stop+1, len(src)) }

func isQuote(src string, i int) bool {
	return i < len(src) && (src[i] == '"' || src[i] == '`')
}

// scanFormatString calls found with each identifier in the placeholders of the string
// literal lit, when lit is a format string. off maps lit into the enclosing text. A
// string that v0.62 could not split holds no identifiers.
func scanFormatString(lit string, off offsets, found func(span, string)) {
	prefix := strings.IndexAny(lit, "\"`")
	if strings.IndexByte(lit[:prefix], 'f') < 0 {
		return
	}
	var (
		multi = lit[prefix] == '`'
		start = prefix + 1
		body  = lit[start : len(lit)-1]
		value = body
		from  = make([]int, len(body)+1)
	)
	for i := range from {
		from[i] = i
	}
	if strings.IndexByte(lit[:prefix], 'r') < 0 {
		var ok bool
		if value, from, ok = unescape(body, multi); !ok {
			return
		}
	}
	placeholders, ok := splitPlaceholders(value)
	if !ok {
		return
	}
	for _, p := range placeholders {
		at := make([]int, p.end-p.start+1)
		for i := range at {
			at[i] = off.pos(start + from[p.start+i])
		}
		scanCode(value[p.start:p.end], offsets{at: at}, found)
	}
}

// unescape returns body with the escapes of a v0.62 string applied, and the offset in
// body of each byte of the result and of its end. It reports false for an escape that
// v0.62 refused.
func unescape(body string, multi bool) (string, []int, bool) {
	var (
		b    strings.Builder
		from = make([]int, 0, len(body)+1)
	)
	b.Grow(len(body))
	for i := 0; i < len(body); {
		if body[i] != '\\' {
			b.WriteByte(body[i])
			from = append(from, i)
			i++
			continue
		}
		if i+1 >= len(body) {
			return "", nil, false
		}
		width := 2
		switch next := body[i+1]; {
		case next == 'b':
			b.WriteByte('\b')
		case next == 't':
			b.WriteByte('\t')
		case next == 'n':
			b.WriteByte('\n')
		case next == 'f':
			b.WriteByte('\f')
		case next == 'r':
			b.WriteByte('\r')
		case next == '\\', next == '"' && !multi, next == '`' && multi:
			b.WriteByte(next)
		case next == 'u':
			if i+6 > len(body) {
				return "", nil, false
			}
			cp, err := strconv.ParseUint(body[i+2:i+6], 16, 16)
			if err != nil {
				return "", nil, false
			}
			b.WriteRune(rune(cp))
			width = 6
		default:
			// An unknown escape passes through, so each byte keeps its own offset.
			b.WriteString(body[i : i+2])
			from = append(from, i, i+1)
		}
		for len(from) < b.Len() {
			from = append(from, i)
		}
		i += width
	}
	return b.String(), append(from, len(body)), true
}

// placeholder is the byte range [start, end) of the expression of a placeholder.
type placeholder struct{ start, end int }

// splitPlaceholders returns the expression of each placeholder in the unescaped body of
// a format string, as v0.62 split it, and false when v0.62 refused the body.
func splitPlaceholders(body string) ([]placeholder, bool) {
	var out []placeholder
	for i := 0; i < len(body); i++ {
		switch {
		case strings.HasPrefix(body[i:], "{{"), strings.HasPrefix(body[i:], "}}"):
			i++
		case body[i] == '{':
			end, ok := placeholderEnd(body, i)
			if !ok {
				return nil, false
			}
			expr, ok := exprEnd(body[i+1 : end])
			if !ok {
				return nil, false
			}
			out = append(out, placeholder{start: i + 1, end: i + 1 + expr})
			i = end
		}
	}
	return out, true
}

// placeholderEnd returns the offset of the brace that closes the placeholder opened at
// body[open].
func placeholderEnd(body string, open int) (int, bool) {
	depth := 1
	for i := open + 1; i < len(body); i++ {
		switch body[i] {
		case '{':
			depth++
		case '}':
			if depth--; depth == 0 {
				return i, true
			}
		}
	}
	return 0, false
}

// exprEnd returns the length of the expression of the placeholder body, which ends at
// the last ':' outside braces and brackets, and false when v0.62 refused an empty
// expression or spec.
func exprEnd(body string) (int, bool) {
	var braces, brackets int
	for i := len(body) - 1; i >= 0; i-- {
		switch body[i] {
		case '}':
			braces++
		case '{':
			braces--
		case ']':
			brackets++
		case '[':
			brackets--
		case ':':
			if braces == 0 && brackets == 0 {
				return i, i > 0 && i < len(body)-1
			}
		}
	}
	return len(body), true
}
