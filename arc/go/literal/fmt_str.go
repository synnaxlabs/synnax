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
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

// FmtStrSegment is one piece of a parsed format string. Start, End are body
// byte offsets (End exclusive). SpecOffset is the body offset of `%`, or -1
// if no spec.
type FmtStrSegment struct {
	Text          string
	Spec          string
	IsPlaceholder bool
	Start         int
	End           int
	SpecOffset    int
}

// FmtStrHasPlaceholder reports whether any segment is a placeholder.
func FmtStrHasPlaceholder(segs []FmtStrSegment) bool {
	return slices.ContainsFunc(segs, func(s FmtStrSegment) bool { return s.IsPlaceholder })
}

// FmtStrStripDelimiters returns the inner body of a `...` raw string token,
// or ok=false if text isn't well-formed. \` escapes are left verbatim.
func FmtStrStripDelimiters(text string) (body string, ok bool) {
	if len(text) < 2 || text[0] != '`' || text[len(text)-1] != '`' {
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
		if expr == "" {
			return nil, errors.New("placeholder '{}' must contain an expression")
		}
		exprPart, spec, err := FmtStrSplitSpec(expr)
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

// FmtStrSplitSpec splits a placeholder body on the last `:`, yielding
// (expr, spec).
func FmtStrSplitSpec(body string) (expr, spec string, err error) {
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

// blacklistedVerbs lists fmt verbs that Go's fmt accepts for our scalar types
// but that we reject so they surface as invalid specs to the user. Reasons:
//
//	v: produces byte-identical output to the bare `{x}` form for every scalar
//	   type Arc allows in a placeholder, so it is a redundant alias.
//	T: leaks Go-runtime type names and duplicates type info already shown
//	   on hover in the Arc editor.
//	U: lacks a lowercase counterpart in Arc and has no current use case,
//	   so we reserve it until a concrete need appears.
const blacklistedVerbs = "vTU"

// stringBlockedVerbs lists verbs Go accepts on string values that Arc rejects.
// x/X dump bytes as hex and have no current use case on strings.
const stringBlockedVerbs = "xX"

// intBlockedVerbs lists verbs Go accepts on integer values that Arc rejects.
// q renders an integer as a quoted Unicode character, which is unrelated to
// the quoted-string meaning of q and has no current use case on integers.
const intBlockedVerbs = "q"

// specShape enforces the canonical anatomy [flags][width][.precision][verb].
// Anything trailing the verb (e.g., "f.2") fails this check; Go's fmt would
// otherwise treat it as literal text and silently accept the malformed spec.
var specShape = regexp.MustCompile(`^[#+\- 0]*\d*(\.\d+)?[a-zA-Z]$`)

// FmtStrValidateSpec probes fmt.Sprintf with a typed dummy and reports an
// error if the spec is not a valid Go fmt verb for the given type.
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
	var dummy any
	var isInt bool
	switch t.Kind {
	case types.KindString:
		dummy = ""
	case types.KindI8, types.KindI16, types.KindI32, types.KindI64,
		types.KindIntegerConstant:
		dummy = int64(0)
		isInt = true
	case types.KindU8, types.KindU16, types.KindU32, types.KindU64:
		dummy = uint64(0)
		isInt = true
	case types.KindF32, types.KindF64,
		types.KindFloatConstant, types.KindNumericConstant,
		types.KindExactIntegerFloatConstant:
		dummy = float64(0)
	default:
		return errors.Newf("cannot format type %s", t)
	}
	if !specShape.MatchString(spec) ||
		strings.ContainsAny(spec, blacklistedVerbs) ||
		(t.Kind == types.KindString && strings.ContainsAny(spec, stringBlockedVerbs)) ||
		(isInt && strings.ContainsAny(spec, intBlockedVerbs)) ||
		strings.Contains(fmt.Sprintf("%"+spec, dummy), "%!") {
		return errors.Newf("invalid format spec %q for type %s", spec, t)
	}
	return nil
}
