// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package literal_test

import (
	"strings"
	"testing"

	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/x/errors"
)

// v062Segment is a segment of the format string split of core/v0.62.0.
type v062Segment struct {
	Text          string
	Spec          string
	IsPlaceholder bool
	Start         int
	End           int
	SpecOffset    int
}

// v062Parse is FmtStrParse of core/v0.62.0, frozen. It must not change.
func v062Parse(body string) ([]v062Segment, error) {
	var segments []v062Segment
	pos := 0
	for pos < len(body) {
		open, text := v062ScanText(body, pos)
		if text != "" {
			segments = append(segments, v062Segment{
				Text:       text,
				Start:      pos,
				End:        open,
				SpecOffset: -1,
			})
		}
		if open == len(body) {
			return segments, nil
		}
		closeIdx, err := v062FindPlaceholderClose(body, open)
		if err != nil {
			return nil, err
		}
		expr := body[open+1 : closeIdx]
		exprPart, spec, err := v062SplitSpec(expr)
		if err != nil {
			return nil, err
		}
		specOffset := -1
		if spec != "" {
			specOffset = open + 1 + len(exprPart)
		}
		segments = append(segments, v062Segment{
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

func v062ScanText(body string, pos int) (int, string) {
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

func v062FindPlaceholderClose(body string, open int) (int, error) {
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

func v062SplitSpec(body string) (expr, spec string, err error) {
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

// quotesInStringsOrComments reports whether each single quote of the placeholder body p
// sits in a double-quoted string or a comment. A line comment ends at \r or \n, and a
// block comment at */.
func quotesInStringsOrComments(p string) bool {
	for i := 0; i < len(p); i++ {
		var end int
		switch rest := p[i:]; {
		case rest[0] == '\'':
			return false
		case rest[0] == '"':
			end = strings.IndexByte(rest[1:], '"') + 1
		case strings.HasPrefix(rest, "//"):
			end = strings.IndexAny(rest, "\r\n")
		case strings.HasPrefix(rest, "/*"):
			if end = strings.Index(rest[2:], "*/"); end >= 0 {
				end += 3
			}
		default:
			continue
		}
		if end <= 0 {
			return true
		}
		i += end
	}
	return true
}

// FuzzFmtStrParseAsV062 checks that FmtStrParse splits each body that core/v0.62.0
// split as v0.62 did, when every single quote of each placeholder sits in a string or
// a comment. Stored text from v0.62 then keeps its meaning.
func FuzzFmtStrParseAsV062(f *testing.F) {
	for _, seed := range []string{
		"{x /* range [0, 100) */:.1f}",
		`{"}" + x}`,
		`{len("{") + x /* } */}`,
		`{"it's" + x:%d}`,
		"{x // it's\n}",
		"{x // it's\r + y:%d}",
		"{x // note}",
		"{x // a:%d}",
		"{x /* it's */:%d}",
		"{x /* a:b */}",
		`{"a // b" + x:.2f}`,
		`{"a /* b" + x} and {y /* "it's" */}`,
		"{m[1:2]:.1f}",
		"{{literal}} {x:%5d} }}",
		"{f({a: 1}):s}",
		"{x /* [ */ + y]:d}",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, body string) {
		if strings.ContainsAny(body, "\\`") {
			return
		}
		want, err := v062Parse(body)
		if err != nil {
			return
		}
		for _, seg := range want {
			if seg.IsPlaceholder &&
				!quotesInStringsOrComments(body[seg.Start+1:seg.End-1]) {
				return
			}
		}
		got, err := literal.FmtStrParse(body)
		if err != nil {
			t.Fatalf("v0.62 split %q, and FmtStrParse fails: %v", body, err)
		}
		if len(got) != len(want) {
			t.Fatalf("%q: %d segments, v0.62 gave %d", body, len(got), len(want))
		}
		for i, g := range got {
			w := want[i]
			if g.Text != w.Text || g.Spec != w.Spec ||
				g.IsPlaceholder != w.IsPlaceholder || g.Start != w.Start ||
				g.End != w.End || g.SpecOffset != w.SpecOffset {
				t.Fatalf("%q: segment %d is %+v, v0.62 gave %+v", body, i, g, w)
			}
		}
	})
}
