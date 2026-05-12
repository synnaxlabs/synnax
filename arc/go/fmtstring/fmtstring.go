// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fmtstring

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

// Segment is one piece of a parsed format string. Start, End are body byte
// offsets (End exclusive). SpecOffset is the body offset of `%`, or -1 if
// no spec.
type Segment struct {
	Text          string
	Spec          string
	IsPlaceholder bool
	Start         int
	End           int
	SpecOffset    int
}

// HasPlaceholder reports whether any segment is a placeholder.
func HasPlaceholder(segs []Segment) bool {
	return slices.ContainsFunc(segs, func(s Segment) bool { return s.IsPlaceholder })
}

// StripDelimiters returns the inner body of a `...` raw string token, or
// ok=false if text isn't well-formed. \` escapes are left verbatim.
func StripDelimiters(text string) (body string, ok bool) {
	if len(text) < 2 || text[0] != '`' || text[len(text)-1] != '`' {
		return "", false
	}
	return text[1 : len(text)-1], true
}

// Parse splits a format-string body into ordered segments at `{...}` placeholders.
// `\{` and `\}` in literal text escape to `{` and `}`.
func Parse(body string) ([]Segment, error) {
	var segments []Segment
	pos := 0
	for pos < len(body) {
		rel := indexUnescapedBrace(body[pos:])
		if rel == -1 {
			segments = append(segments, Segment{
				Text:       braceUnescaper.Replace(body[pos:]),
				Start:      pos,
				End:        len(body),
				SpecOffset: -1,
			})
			return segments, nil
		}
		i := pos + rel
		if body[i] == '}' {
			return nil, errors.New("unmatched '}'")
		}
		if i > pos {
			segments = append(segments, Segment{
				Text:       braceUnescaper.Replace(body[pos:i]),
				Start:      pos,
				End:        i,
				SpecOffset: -1,
			})
		}
		relR := strings.IndexAny(body[i+1:], "{}")
		if relR == -1 || body[i+1+relR] == '{' {
			return nil, errors.New("unmatched '{'")
		}
		rb := i + 1 + relR
		expr := body[i+1 : rb]
		if expr == "" {
			return nil, errors.New("placeholder '{}' must contain an expression")
		}
		exprPart, spec, err := SplitSpec(expr)
		if err != nil {
			return nil, err
		}
		specOffset := -1
		if spec != "" {
			specOffset = i + 1 + len(exprPart)
		}
		segments = append(segments, Segment{
			Text:          exprPart,
			Spec:          spec,
			IsPlaceholder: true,
			Start:         i,
			End:           rb + 1,
			SpecOffset:    specOffset,
		})
		pos = rb + 1
	}
	return segments, nil
}

func indexUnescapedBrace(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
			continue
		}
		if s[i] == '{' || s[i] == '}' {
			return i
		}
	}
	return -1
}

var braceUnescaper = strings.NewReplacer(`\{`, `{`, `\}`, `}`)

// SplitSpec splits a placeholder body on the last `%` not flanked by
// whitespace, leaving `a % b` as modulo and `x%.2f` as (x, .2f).
func SplitSpec(body string) (expr, spec string, err error) {
	idx := -1
	for i := len(body) - 1; i >= 0; i-- {
		if body[i] != '%' {
			continue
		}
		if i > 0 && isSpace(body[i-1]) {
			continue
		}
		if i+1 < len(body) && isSpace(body[i+1]) {
			continue
		}
		idx = i
		break
	}
	if idx < 0 {
		return body, "", nil
	}
	expr = body[:idx]
	spec = body[idx+1:]
	if expr == "" {
		return "", "", errors.New("placeholder must contain an expression before '%'")
	}
	if spec == "" {
		return "", "", errors.New("placeholder format spec after '%' is empty")
	}
	return expr, spec, nil
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// blacklistedVerbs lists fmt verbs that Go's fmt accepts for our scalar types
// but that we reject so they surface as invalid specs to the user. Reasons:
//
//	v: produces byte-identical output to the bare `{x}` form for every scalar
//	   type Arc allows in a placeholder, so it is a redundant alias.
//	T: leaks Go-runtime type names and duplicates type info already shown
//	   on hover in the Arc editor.
const blacklistedVerbs = "vT"

// specShape enforces the canonical anatomy [flags][width][.precision][verb].
// Anything trailing the verb (e.g., "f.2") fails this check; Go's fmt would
// otherwise treat it as literal text and silently accept the malformed spec.
var specShape = regexp.MustCompile(`^[#+\- 0]*\d*(\.\d+)?[a-zA-Z]$`)

// ValidateSpec probes fmt.Sprintf with a typed dummy and reports an error if
// the spec is not a valid Go fmt verb for the given type.
func ValidateSpec(spec string, t types.Type) error {
	if spec == "" {
		return nil
	}
	if t.Kind == types.KindVariable {
		if t.Constraint == nil {
			return errors.Newf("cannot format type %s", t)
		}
		return ValidateSpec(spec, *t.Constraint)
	}
	var dummy any
	switch t.Kind {
	case types.KindString:
		dummy = ""
	case types.KindI8, types.KindI16, types.KindI32, types.KindI64,
		types.KindIntegerConstant:
		dummy = int64(0)
	case types.KindU8, types.KindU16, types.KindU32, types.KindU64:
		dummy = uint64(0)
	case types.KindF32, types.KindF64,
		types.KindFloatConstant, types.KindNumericConstant,
		types.KindExactIntegerFloatConstant:
		dummy = float64(0)
	default:
		return errors.Newf("cannot format type %s", t)
	}
	if !specShape.MatchString(spec) ||
		strings.ContainsAny(spec, blacklistedVerbs) ||
		strings.Contains(fmt.Sprintf("%"+spec, dummy), "%!") {
		return errors.Newf("invalid format spec %q for type %s", spec, t)
	}
	return nil
}
