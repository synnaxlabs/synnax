// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package validate

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// pathSeparator joins the segments of a path. A path segment never holds it.
const pathSeparator = '.'

// IsPathSegment reports whether s can name one segment of a path: valid UTF-8 in NFC,
// not empty after trimming spaces, and with no '.' and no character that shows as
// nothing or as a space other than U+0020.
func IsPathSegment(s string) bool {
	if ascii, ok := asciiPathSegment(s); ascii {
		return ok
	}
	// String returns s itself when s is in NFC, so the check allocates nothing.
	if !utf8.ValidString(s) || strings.Trim(s, " ") == "" || norm.NFC.String(s) != s {
		return false
	}
	for _, r := range s {
		if r == pathSeparator || isInvisible(r) {
			return false
		}
	}
	return true
}

// asciiPathSegment reports whether s is all ASCII, which is always in NFC, and if so
// whether it passes the rule of IsPathSegment. It reports true for a refused byte
// before it sees the rest of s, since s then fails either way.
func asciiPathSegment(s string) (ascii, ok bool) {
	visible := false
	for i := range len(s) {
		c := s[i]
		switch {
		case c >= utf8.RuneSelf:
			return false, false
		case c == pathSeparator || c < ' ' || c == 0x7f:
			return true, false
		case c != ' ':
			visible = true
		}
	}
	return true, visible
}

// NormalizeName returns s in Unicode Normalization Form C, the one form in which names
// are stored and compared.
func NormalizeName(s string) string { return norm.NFC.String(s) }

// brailleBlank is U+2800, a symbol (So) that shows as nothing.
const brailleBlank = '\u2800'

// isInvisible reports whether r shows as nothing or as a space other than U+0020: the
// categories Cc, Cf, Zl, Zp, and Zs, the default ignorable code points, and U+2800.
func isInvisible(r rune) bool {
	if r == ' ' {
		return false
	}
	if r == brailleBlank {
		return true
	}
	return unicode.In(r, invisibleTables...)
}

// invisibleTables are the tables of the characters that isInvisible refuses.
var invisibleTables = []*unicode.RangeTable{
	unicode.Cc,
	unicode.Cf,
	unicode.Zl,
	unicode.Zp,
	unicode.Zs,
	unicode.Other_Default_Ignorable_Code_Point,
	unicode.Variation_Selector,
}
