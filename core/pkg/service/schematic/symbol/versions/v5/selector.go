// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/synnaxlabs/x/errors"
)

// compound is one step of a selector that the v4 editor wrote: #id, .class.class,
// [data-region-id="..."], or tag:nth-of-type(n).
type compound struct {
	tag          string
	id           string
	classes      []string
	dataRegionID string
	nth          int
}

// parseSelector parses a selector of compounds joined by " > ".
func parseSelector(sel string) ([]compound, error) {
	var (
		out []compound
		p   = selectorParser{src: sel}
	)
	for {
		c, err := p.compound()
		if err != nil {
			return nil, errors.Wrapf(err, "selector %q", sel)
		}
		out = append(out, c)
		if p.done() {
			return out, nil
		}
		if !p.consume(" > ") {
			return nil, errors.Newf("selector %q: unexpected text at %d", sel, p.pos)
		}
	}
}

type selectorParser struct {
	src string
	pos int
}

func (p *selectorParser) done() bool { return p.pos >= len(p.src) }

func (p *selectorParser) consume(s string) bool {
	if !strings.HasPrefix(p.src[p.pos:], s) {
		return false
	}
	p.pos += len(s)
	return true
}

func (p *selectorParser) compound() (compound, error) {
	var c compound
	switch {
	case p.consume("#"):
		c.id = p.ident()
		if c.id == "" {
			return c, errors.New("empty id")
		}
	case p.consume(`[data-region-id="`):
		c.dataRegionID = p.until('"')
		if !p.consume(`"]`) {
			return c, errors.New("unclosed attribute")
		}
	case strings.HasPrefix(p.src[p.pos:], "."):
		for p.consume(".") {
			class := p.ident()
			if class == "" {
				return c, errors.New("empty class")
			}
			c.classes = append(c.classes, class)
		}
	default:
		c.tag = p.ident()
		if c.tag == "" {
			return c, errors.New("empty tag")
		}
		if p.consume(":nth-of-type(") {
			n, err := strconv.Atoi(p.until(')'))
			if err != nil || n < 1 || !p.consume(")") {
				return c, errors.New("bad nth-of-type")
			}
			c.nth = n
		}
	}
	return c, nil
}

// ident reads an escaped CSS identifier up to the next unescaped delimiter.
func (p *selectorParser) ident() string {
	var b strings.Builder
	for !p.done() {
		ch := p.src[p.pos]
		if ch == '\\' {
			b.WriteRune(p.escape())
			continue
		}
		if strings.IndexByte(".#[:( >)", ch) >= 0 {
			break
		}
		r, size := utf8.DecodeRuneInString(p.src[p.pos:])
		b.WriteRune(r)
		p.pos += size
	}
	return b.String()
}

// until reads escaped text up to the next unescaped end.
func (p *selectorParser) until(end byte) string {
	var b strings.Builder
	for !p.done() && p.src[p.pos] != end {
		if p.src[p.pos] == '\\' {
			b.WriteRune(p.escape())
			continue
		}
		r, size := utf8.DecodeRuneInString(p.src[p.pos:])
		b.WriteRune(r)
		p.pos += size
	}
	return b.String()
}

// escape reads one CSS escape: up to six hex digits and one optional space, or one
// literal character.
func (p *selectorParser) escape() rune {
	p.pos++
	start := p.pos
	for p.pos < len(p.src) && p.pos-start < 6 && isHex(p.src[p.pos]) {
		p.pos++
	}
	if p.pos == start {
		if p.done() {
			return utf8.RuneError
		}
		r, size := utf8.DecodeRuneInString(p.src[p.pos:])
		p.pos += size
		return r
	}
	code, err := strconv.ParseUint(p.src[start:p.pos], 16, 32)
	if !p.done() && p.src[p.pos] == ' ' {
		p.pos++
	}
	if err != nil || code == 0 || code > utf8.MaxRune ||
		(code >= 0xD800 && code <= 0xDFFF) {
		return utf8.RuneError
	}
	return rune(code)
}

func isHex(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

// cssEscape escapes s as the CSS.escape function of the CSS Object Model does.
func cssEscape(s string) string {
	var b strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		switch {
		case r == 0:
			b.WriteRune(utf8.RuneError)
		case (r >= 0x1 && r <= 0x1F) || r == 0x7F,
			i == 0 && r >= '0' && r <= '9',
			i == 1 && r >= '0' && r <= '9' && runes[0] == '-':
			b.WriteString(`\` + strconv.FormatInt(int64(r), 16) + " ")
		case i == 0 && r == '-' && len(runes) == 1:
			b.WriteString(`\-`)
		case r >= 0x80, r == '-', r == '_',
			r >= '0' && r <= '9', r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		default:
			b.WriteString(`\`)
			b.WriteRune(r)
		}
	}
	return b.String()
}
