// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package literal

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

// FmtStrSegment is one piece of a parsed format string.
type FmtStrSegment struct {
	// Literal text, or for placeholders, the expression source.
	Text string
	// Optional format spec after `:` in a placeholder.
	Spec string
	// True if this segment was a `{...}` placeholder.
	IsPlaceholder bool
	// Start byte offset in the body.
	Start int
	// End byte offset in the body, exclusive.
	End int
	// Body byte offset of `:`, or -1 if no spec.
	SpecOffset int
	// Offsets holds the body byte offset of each byte of a placeholder's Text, and one
	// past its end. FmtStrSplit sets it.
	Offsets []int
}

// FmtStrHasPlaceholder reports whether any segment is a placeholder.
func FmtStrHasPlaceholder(segs []FmtStrSegment) bool {
	return slices.ContainsFunc(
		segs,
		func(s FmtStrSegment) bool { return s.IsPlaceholder },
	)
}

// StringFlags carries the optional r/f prefix and the quote style of a string
// literal. Raw skips standard escape processing; Format opts into {expr}
// placeholders; Multi means the literal was backtick-delimited.
type StringFlags struct {
	Raw    bool
	Format bool
	Multi  bool
}

// StripQuotes peels the optional r/f prefix and surrounding quote delimiters
// from a string literal token, returning the inner body. ok=false when text is
// malformed; well-formed tokens from the lexer should always succeed.
func StripQuotes(text string) (body string, flags StringFlags, ok bool) {
	rest := text
	for i := 0; i < 2 && len(rest) > 0 && (rest[0] == 'r' || rest[0] == 'f'); i++ {
		switch rest[0] {
		case 'r':
			if flags.Raw {
				return "", StringFlags{}, false
			}
			flags.Raw = true
		case 'f':
			if flags.Format {
				return "", StringFlags{}, false
			}
			flags.Format = true
		}
		rest = rest[1:]
	}
	if len(rest) >= 2 && rest[0] == '`' && rest[len(rest)-1] == '`' {
		flags.Multi = true
		return rest[1 : len(rest)-1], flags, true
	}
	if len(rest) >= 2 && rest[0] == '"' && rest[len(rest)-1] == '"' {
		return rest[1 : len(rest)-1], flags, true
	}
	return "", StringFlags{}, false
}

// UnescapeString applies the standard Arc escape table to body. The delimiter
// escape is asymmetric: single-line strings recognize \", multi-line strings
// recognize \`; the other passes through verbatim. Unrecognized escapes also
// pass through verbatim. Literal-brace escapes ({{ and }}) are handled by
// FmtStrParse. Errors only on a trailing backslash or an incomplete \uXXXX escape.
func UnescapeString(body string, multi bool) (string, error) {
	value, _, err := unescape(body, multi)
	return value, err
}

// unescape applies the escape table to body. It also returns, for each byte of the
// result and one past its end, the offset in body that byte came from.
func unescape(body string, multi bool) (string, []int, error) {
	var b strings.Builder
	b.Grow(len(body))
	offsets := make([]int, 0, len(body)+1)
	for i := 0; i < len(body); {
		c := body[i]
		if c != '\\' {
			b.WriteByte(c)
			offsets = append(offsets, i)
			i++
			continue
		}
		value, width, known, err := decodeEscape(body, i, multi)
		if err != nil {
			return "", nil, err
		}
		b.WriteString(value)
		if !known {
			// The escape passes through, so each byte keeps its own offset.
			offsets = append(offsets, i, i+1)
		}
		for len(offsets) < b.Len() {
			offsets = append(offsets, i)
		}
		i += width
	}
	return b.String(), append(offsets, len(body)), nil
}

// decodeEscape returns the text of the escape that starts at body[i] and its width in
// body. It returns false when the escape is unknown and passes through verbatim.
func decodeEscape(body string, i int, multi bool) (string, int, bool, error) {
	if i+1 >= len(body) {
		return "", 0, false, errors.New("trailing backslash in string literal")
	}
	switch next := body[i+1]; {
	case next == 'b':
		return "\b", 2, true, nil
	case next == 't':
		return "\t", 2, true, nil
	case next == 'n':
		return "\n", 2, true, nil
	case next == 'f':
		return "\f", 2, true, nil
	case next == 'r':
		return "\r", 2, true, nil
	case next == '"' && !multi:
		return `"`, 2, true, nil
	case next == '`' && multi:
		return "`", 2, true, nil
	case next == '\\':
		return `\`, 2, true, nil
	case next == 'u':
		if i+6 > len(body) {
			return "", 0, false, errors.New(`incomplete \u escape in string literal`)
		}
		cp, err := strconv.ParseUint(body[i+2:i+6], 16, 16)
		if err != nil {
			return "", 0, false, errors.Newf(
				`invalid \u escape %q in string literal`,
				body[i:i+6],
			)
		}
		return string(rune(cp)), 6, true, nil
	default:
		return body[i : i+2], 2, false, nil
	}
}

// FmtStrSplit applies the escapes in body, unless flags.Raw, and then splits the
// result into segments as FmtStrParse does. Each segment's Text has its escapes
// applied, and its Start, End, and SpecOffset index body as written.
func FmtStrSplit(body string, flags StringFlags) ([]FmtStrSegment, error) {
	value, offsets := body, make([]int, len(body)+1)
	for i := range offsets {
		offsets[i] = i
	}
	if !flags.Raw {
		var err error
		if value, offsets, err = unescape(body, flags.Multi); err != nil {
			return nil, err
		}
	}
	segments, err := FmtStrParse(value)
	if err != nil {
		return nil, err
	}
	for i := range segments {
		seg := &segments[i]
		if seg.IsPlaceholder {
			text := seg.Start + 1
			seg.Offsets = slices.Clone(offsets[text : text+len(seg.Text)+1])
		}
		seg.Start, seg.End = offsets[seg.Start], offsets[seg.End]
		if seg.SpecOffset >= 0 {
			seg.SpecOffset = offsets[seg.SpecOffset]
		}
	}
	return segments, nil
}

// FmtStrParse splits a format-string body into ordered segments at `{...}`
// placeholders. `{{` escapes to a literal `{` and `}}` to a literal `}`; a
// bare `}` outside a placeholder is plain text.
func FmtStrParse(body string) ([]FmtStrSegment, error) {
	var segments []FmtStrSegment
	pos := 0
	for pos < len(body) {
		open, text := scanText(body, pos)
		if text != "" {
			segments = append(segments, FmtStrSegment{
				Text:       text,
				Start:      pos,
				End:        open,
				SpecOffset: -1,
			})
		}
		if open == len(body) {
			return segments, nil
		}
		closeIdx, err := findPlaceholderClose(body, open)
		if err != nil {
			return nil, err
		}
		expr := body[open+1 : closeIdx]
		exprPart, spec, err := splitSpec(expr)
		if err != nil {
			return nil, err
		}
		specOffset := -1
		if spec != "" {
			specOffset = open + 1 + len(exprPart)
		}
		segments = append(segments, FmtStrSegment{
			Text:          exprPart,
			Spec:          spec,
			IsPlaceholder: true,
			Start:         open,
			End:           closeIdx + 1,
			SpecOffset:    specOffset,
		})
		pos = closeIdx + 1
	}
	return segments, nil
}

func scanText(body string, pos int) (int, string) {
	var b strings.Builder
	for i := pos; i < len(body); i++ {
		c := body[i]
		if c == '{' {
			if i+1 < len(body) && body[i+1] == '{' {
				b.WriteByte('{')
				i++
				continue
			}
			return i, b.String()
		}
		if c == '}' && i+1 < len(body) && body[i+1] == '}' {
			b.WriteByte('}')
			i++
			continue
		}
		b.WriteByte(c)
	}
	return len(body), b.String()
}

func findPlaceholderClose(body string, open int) (int, error) {
	depth := 1
	for i := open + 1; i < len(body); i++ {
		switch body[i] {
		case '\'':
			i = quotedEnd(body, i)
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return 0, errors.New("unmatched '{'")
}

// quotedEnd returns the index of the quote that closes the channel name opening at i,
// or the last index of body when no quote closes it.
func quotedEnd(body string, i int) int {
	for i++; i < len(body); i++ {
		switch body[i] {
		case '\\':
			i++
		case '\'':
			return i
		}
	}
	return len(body) - 1
}

// splitSpec splits a placeholder body on the last `:` at brace and bracket depth 0,
// yielding (expr, spec). Colons inside nested `{...}` (struct literals), `[...]`
// (index/slice expressions), or a quoted channel name are skipped.
func splitSpec(body string) (expr, spec string, err error) {
	idx := -1
	braceDepth := 0
	bracketDepth := 0
	for i := 0; i < len(body); i++ {
		switch body[i] {
		case '\'':
			i = quotedEnd(body, i)
		case '{':
			braceDepth++
		case '}':
			braceDepth--
		case '[':
			bracketDepth++
		case ']':
			bracketDepth--
		case ':':
			if braceDepth == 0 && bracketDepth == 0 {
				idx = i
			}
		}
	}
	if idx < 0 {
		return body, "", nil
	}
	expr = body[:idx]
	spec = body[idx+1:]
	if expr == "" {
		return "", "", errors.New("placeholder must contain an expression before ':'")
	}
	if spec == "" {
		return "", "", errors.New("placeholder format spec after ':' is empty")
	}
	return expr, spec, nil
}

// specShape enforces the canonical anatomy [flags][width][.precision][verb].
var specShape = regexp.MustCompile(`^[#+\- 0]*\d*(\.\d+)?[a-zA-Z]$`)

var (
	stringKinds = []types.Kind{types.KindString}
	intKinds    = []types.Kind{
		types.KindI8, types.KindI16, types.KindI32, types.KindI64,
		types.KindU8, types.KindU16, types.KindU32, types.KindU64,
		types.KindIntegerConstant,
	}
	floatKinds = []types.Kind{
		types.KindF32, types.KindF64,
		types.KindFloatConstant, types.KindNumericConstant,
		types.KindExactIntegerFloatConstant,
	}
	numericKinds     = slices.Concat(intKinds, floatKinds)
	formattableKinds = slices.Concat(stringKinds, numericKinds)
)

// verbAllowedKinds maps each supported format verb to the type kinds it can
// format. To add a verb later (e.g., a timestamp verb), add an entry here.
var verbAllowedKinds = map[byte][]types.Kind{
	's': stringKinds,
	'q': stringKinds,
	'b': intKinds,
	'c': intKinds,
	'd': intKinds,
	'o': intKinds,
	'O': intKinds,
	'x': intKinds,
	'X': intKinds,
	'e': floatKinds,
	'E': floatKinds,
	'f': floatKinds,
	'g': floatKinds,
	'G': floatKinds,
}

// FmtStrValidateSpec reports an error if spec is not a supported verb for t,
// or if t is not a formattable kind.
func FmtStrValidateSpec(spec string, t types.Type) error {
	if spec == "" {
		return nil
	}
	if t.Kind == types.KindVariable {
		if t.Constraint == nil {
			return errors.Newf("cannot format type %s", t)
		}
		return FmtStrValidateSpec(spec, *t.Constraint)
	}
	if !slices.Contains(formattableKinds, t.Kind) {
		return errors.Newf("cannot format type %s", t)
	}
	if !specShape.MatchString(spec) {
		return errors.Newf("invalid format spec %q for type %s", spec, t)
	}
	allowed, ok := verbAllowedKinds[spec[len(spec)-1]]
	if !ok || !slices.Contains(allowed, t.Kind) {
		return errors.Newf("invalid format spec %q for type %s", spec, t)
	}
	return nil
}
