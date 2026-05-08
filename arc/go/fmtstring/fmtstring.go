// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fmtstring

import "github.com/synnaxlabs/x/errors"

// Segment is one piece of a parsed format string. If IsPlaceholder is false,
// Text is literal output. If IsPlaceholder is true, Text is the source of an
// Arc expression to evaluate and splice in.
type Segment struct {
	Text          string
	IsPlaceholder bool
}

// Parse splits a format-string body into ordered segments. `{expr}` is a
// placeholder; `{{` and `}}` escape to literal `{` and `}`. body should be
// the unquoted body (no surrounding quotes or backticks).
func Parse(body string) ([]Segment, error) {
	var (
		segments []Segment
		lit      []byte
		i        int
	)
	for i < len(body) {
		c := body[i]
		if c == '{' {
			if i+1 < len(body) && body[i+1] == '{' {
				lit = append(lit, '{')
				i += 2
				continue
			}
			if len(lit) > 0 {
				segments = append(segments, Segment{Text: string(lit)})
				lit = lit[:0]
			}
			end := -1
			for j := i + 1; j < len(body); j++ {
				if body[j] == '}' {
					end = j
					break
				}
				if body[j] == '{' {
					return nil, errors.New("nested '{' inside placeholder")
				}
			}
			if end == -1 {
				return nil, errors.New("unterminated placeholder; expected closing '}'")
			}
			expr := body[i+1 : end]
			if expr == "" {
				return nil, errors.New("placeholder '{}' must contain an expression")
			}
			segments = append(segments, Segment{Text: expr, IsPlaceholder: true})
			i = end + 1
			continue
		}
		if c == '}' {
			if i+1 < len(body) && body[i+1] == '}' {
				lit = append(lit, '}')
				i += 2
				continue
			}
			return nil, errors.New("unmatched '}'; use '}}' to escape a literal '}'")
		}
		lit = append(lit, c)
		i++
	}
	if len(lit) > 0 {
		segments = append(segments, Segment{Text: string(lit)})
	}
	return segments, nil
}
