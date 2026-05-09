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
	"strings"

	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

// Segment is one piece of a parsed format string. If IsPlaceholder is false,
// Text is literal output. If IsPlaceholder is true, Text is the source of an
// Arc expression to evaluate and splice in.
type Segment struct {
	Text          string
	Spec          string
	IsPlaceholder bool
}

// Parse splits a format-string body into ordered segments at `{...}` placeholders.
func Parse(body string) ([]Segment, error) {
	var segments []Segment
	for len(body) > 0 {
		i := strings.IndexAny(body, "{}")
		if i == -1 {
			segments = append(segments, Segment{Text: body})
			return segments, nil
		}
		if body[i] == '}' {
			return nil, errors.New("unmatched '}'")
		}
		if i > 0 {
			segments = append(segments, Segment{Text: body[:i]})
		}
		rest := body[i+1:]
		j := strings.IndexAny(rest, "{}")
		if j == -1 || rest[j] == '{' {
			return nil, errors.New("unmatched '{'")
		}
		expr := rest[:j]
		if expr == "" {
			return nil, errors.New("placeholder '{}' must contain an expression")
		}
		exprPart, spec, err := SplitSpec(expr)
		if err != nil {
			return nil, err
		}
		segments = append(segments, Segment{Text: exprPart, Spec: spec, IsPlaceholder: true})
		body = rest[j+1:]
	}
	return segments, nil
}

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

// ValidateNumericSpec probes fmt.Sprintf with a typed dummy and reports an
// error if the spec is not a valid Go fmt verb for the given numeric type.
func ValidateNumericSpec(spec string, t types.Type) error {
	var dummy any
	switch t.Kind {
	case types.KindI8, types.KindI16, types.KindI32, types.KindI64:
		dummy = int64(0)
	case types.KindU8, types.KindU16, types.KindU32, types.KindU64:
		dummy = uint64(0)
	case types.KindF32, types.KindF64:
		dummy = float64(0)
	default:
		return errors.Newf("cannot format type %s", t)
	}
	if strings.Contains(fmt.Sprintf("%"+spec, dummy), "%!") {
		return errors.Newf("invalid format spec %q for type %s", spec, t)
	}
	return nil
}
