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
	return slices.ContainsFunc(segs, func(s FmtStrSegment) bool { return s.IsPlaceholder })
}

// FmtStrStripDelimiters returns the inner body of a `...` raw string token,
// or ok=false if text isn't well-formed. \` escapes are left verbatim.
func FmtStrStripDelimiters(text string) (string, bool) {
	if len(text) < 2 || text[0] != '`' || text[len(text)-1] != '`' {
		return "", false
	}
	backslashes := 0
	for i := len(text) - 2; i > 0 && text[i] == '\\'; i-- {
		backslashes++
	}
	if backslashes%2 == 1 {
		return "", false
	}
	return text[1 : len(text)-1], true
}

// FmtStrParse splits a format-string body into ordered segments at `{...}`
// placeholders. `\{` escapes to a literal `{`; a bare `}` outside a placeholder
// is plain text.
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
		if body[i] == '\\' && i+1 < len(body) && body[i+1] == '{' {
			b.WriteByte('{')
			i++
			continue
		}
		if body[i] == '{' {
			return i, b.String()
		}
		b.WriteByte(body[i])
	}
	return len(body), b.String()
}

func findPlaceholderClose(body string, open int) (int, error) {
	for i := open + 1; i < len(body); i++ {
		switch body[i] {
		case '{':
			return 0, errors.New("unmatched '{'")
		case '}':
			return i, nil
		}
	}
	return 0, errors.New("unmatched '{'")
}

// splitSpec splits a placeholder body on the last `:`, yielding (expr, spec).
func splitSpec(body string) (expr, spec string, err error) {
	idx := strings.LastIndexByte(body, ':')
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
