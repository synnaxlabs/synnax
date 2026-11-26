// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"context"
	"strconv"
	"strings"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	resource "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// CalculationTextMigration writes the new name of each renamed channel into the
// expression of each calculated channel (TextRename). It runs after the name and clash
// passes, and before anything compiles a calculation.
var CalculationTextMigration = migrate.New(
	"v63_rename_calculation_channels",
	func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
		renames, err := resource.StagedRenames(tx)
		if err != nil || len(renames) == 0 {
			return err
		}
		var channels []Channel
		if err = gorp.NewRetrieve[Key, Channel]().
			Entries(&channels).
			Exec(ctx, tx); err != nil {
			return err
		}
		names := make(map[Key]string, len(channels))
		for _, c := range channels {
			names[c.Key()] = c.Name
		}
		rename, err := TextRename(names, renames)
		if err != nil {
			return err
		}
		var changed []Channel
		for _, c := range channels {
			if e := RenameInText(c.Expression, rename); e != c.Expression {
				c.Expression = e
				changed = append(changed, c)
			}
		}
		return gorp.WrapWriter[Key, Channel](tx).Set(ctx, changed...)
	},
)

// TextRename returns the rename that the text passes apply to each channel name in
// text that a v0.62 Core stored. names holds the name of each channel now, by key. A
// name that no channel holds now in NFC, and that a renamed channel held, becomes the
// new name of the channel that a v0.62 Core bound it to: of the renamed channels that
// held it, the one with the highest key.
func TextRename(
	names map[Key]string,
	renames []resource.Rename,
) (func(string) (string, bool), error) {
	held := make(set.Set[string], len(names))
	for _, n := range names {
		held.Add(resource.NormalizeName(n))
	}
	type target struct {
		key  Key
		name string
	}
	moved := make(map[string]target, len(renames))
	for _, r := range renames {
		if r.ID.Type != resource.TypeChannel ||
			held.Contains(resource.NormalizeName(r.From)) {
			continue
		}
		local, err := strconv.ParseUint(r.ID.Key, 10, 32)
		if err != nil {
			return nil, err
		}
		key := Key(local)
		name, ok := names[key]
		if !ok {
			return nil, errors.Newf("staged rename of %s names no record", r.ID)
		}
		if t, ok := moved[r.From]; !ok || key > t.key {
			moved[r.From] = target{key: key, name: name}
		}
	}
	return func(name string) (string, bool) {
		t, ok := moved[name]
		return t.name, ok
	}, nil
}

// RenameInText returns text, an Arc program or calculation expression that a v0.62
// Core stored, with each identifier that rename maps to a new name written as that
// name. The new name must lex as one identifier. An identifier is one that v0.62 lexed
// with dashed names on, in code or in a placeholder of a format string, where rename
// gets the name that the placeholder reads. Comments and the rest of strings stay as
// they are.
func RenameInText(text string, rename func(name string) (string, bool)) string {
	type edit struct {
		span
		text string
	}
	var edits []edit
	scanCode(text, place{}, func(s span, name string) {
		if to, ok := rename(name); ok {
			edits = append(edits, edit{span: s, text: to})
		}
	})
	if len(edits) == 0 {
		return text
	}
	var b strings.Builder
	last := 0
	for _, e := range edits {
		b.WriteString(text[last:e.start])
		b.WriteString(e.text)
		last = e.end
	}
	b.WriteString(text[last:])
	return b.String()
}

// span is the byte range [start, end) of an identifier.
type span struct{ start, end int }

// place maps a source into the enclosing text. A nil at maps byte j to base + j.
// Otherwise at holds the offset of each byte and of the end.
type place struct {
	at   []int
	base int
}

func (p place) pos(j int) int {
	if p.at != nil {
		return p.at[j]
	}
	return p.base + j
}

// sub returns the place of the source that starts at i in the source of p.
func (p place) sub(i int) place {
	if p.at != nil {
		return place{at: p.at[i:]}
	}
	return place{base: p.base + i}
}

// scanCode calls found with the span in the enclosing text of each identifier in the
// code src, in order, and the name that the identifier reads. at places src in the
// enclosing text. It follows the v0.62 lexer with dashed names on.
func scanCode(src string, at place, found func(span, string)) {
	// No "/*" after one that finds no "*/" can find one.
	unclosed := false
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
			end := -1
			if !unclosed {
				end = strings.Index(src[i+2:], "*/")
				unclosed = end < 0
			}
			if end >= 0 {
				i += end + 4
			} else {
				i++
			}
		case c == '"' || c == '`' || c == 'r' || c == 'f':
			n, ok := stringLen(src[i:])
			switch {
			case ok:
				scanFormatString(src[i:i+n], at.sub(i), found)
				i += n
			case c == 'r' || c == 'f':
				i = scanIdentifier(src, i, at, found)
			default:
				i += skip(src[i:], n)
			}
		case c == '$' || c == '!':
			// Each starts a token only before '=', so the error recovery of a lone one
			// drops the character after it too.
			i = min(i+2, len(src))
		case isIdentifierByte(c) && (c < '0' || c > '9'):
			i = scanIdentifier(src, i, at, found)
		default:
			i++
		}
	}
}

// scanIdentifier calls found with the identifier that starts at src[i], and returns its
// end. A '-' joins an identifier unless '>' or '=' comes after it.
func scanIdentifier(src string, i int, at place, found func(span, string)) int {
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
	found(span{start: at.pos(i), end: at.pos(end)}, src[i:end])
	return end
}

func isIdentifierByte(c byte) bool {
	return c == '_' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' ||
		'0' <= c && c <= '9'
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
// literal lit, when lit is a format string. at places lit in the enclosing text. A
// string that v0.62 could not split holds no identifiers.
func scanFormatString(lit string, at place, found func(span, string)) {
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
	raw := strings.IndexByte(lit[:prefix], 'r') >= 0
	if !raw {
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
		offsets := make([]int, p.end-p.start+1)
		for i := range offsets {
			offsets[i] = at.pos(start + from[p.start+i])
		}
		scanCode(value[p.start:p.end], place{at: offsets}, found)
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
