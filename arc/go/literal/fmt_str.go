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
	var b strings.Builder
	b.Grow(len(body))
	for i := 0; i < len(body); {
		c := body[i]
		if c != '\\' {
			b.WriteByte(c)
			i++
			continue
		}
		if i+1 >= len(body) {
			return "", errors.New("trailing backslash in string literal")
		}
		next := body[i+1]
		switch {
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
		case next == '"' && !multi:
			b.WriteByte('"')
		case next == '`' && multi:
			b.WriteByte('`')
		case next == '\\':
			b.WriteByte('\\')
		case next == 'u':
			if i+6 > len(body) {
				return "", errors.New(`incomplete \u escape in string literal`)
			}
			cp, err := strconv.ParseUint(body[i+2:i+6], 16, 32)
			if err != nil {
				return "", errors.Newf(
					`invalid \u escape %q in string literal`,
					body[i:i+6],
				)
			}
			b.WriteRune(rune(cp))
			i += 6
			continue
		default:
			b.WriteByte('\\')
			b.WriteByte(next)
		}
		i += 2
	}
	return b.String(), nil
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
		close, err := findPlaceholderClose(body, open)
		if err != nil {
			return nil, err
		}
		expr := body[open+1 : close]
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
			End:           close + 1,
			SpecOffset:    specOffset,
		})
		pos = close + 1
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

// splitSpec splits a placeholder body on the last `:` at brace and bracket
// depth 0, yielding (expr, spec). Colons inside nested `{...}` (struct
// literals) or `[...]` (index/slice expressions) are skipped.
func splitSpec(body string) (expr, spec string, err error) {
	idx := -1
	braceDepth := 0
	bracketDepth := 0
	for i := len(body) - 1; i >= 0; i-- {
		switch body[i] {
		case '}':
			braceDepth++
		case '{':
			braceDepth--
		case ']':
			bracketDepth++
		case '[':
			bracketDepth--
		case ':':
			if braceDepth == 0 && bracketDepth == 0 {
				idx = i
			}
		}
		if idx >= 0 {
			break
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
