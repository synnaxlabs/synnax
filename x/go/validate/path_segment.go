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

// PathSeparator joins the segments of a path. A path segment never holds it.
const PathSeparator = '.'

// IsPathSegment reports whether s can name one segment of a path: valid UTF-8, not
// empty after trimming spaces, and with no PathSeparator and no character that shows as
// nothing or as a space other than U+0020. It does not check that s is in NFC; see
// NormalizeName.
func IsPathSegment(s string) bool {
	if !utf8.ValidString(s) || strings.Trim(s, " ") == "" {
		return false
	}
	for _, r := range s {
		if r == PathSeparator || isInvisible(r) {
			return false
		}
	}
	return true
}

// NormalizeName returns s in Unicode Normalization Form C, the one form in which names
// are stored and compared.
func NormalizeName(s string) string { return norm.NFC.String(s) }

// IsNormalName reports whether s is in Unicode Normalization Form C.
func IsNormalName(s string) bool { return norm.NFC.IsNormalString(s) }

// SanitizePathSegment returns s rewritten to pass IsPathSegment and to be in NFC: each
// invalid byte becomes U+FFFD, each PathSeparator becomes '_', and each refused
// character goes. It returns "" when nothing but spaces is left.
func SanitizePathSegment(s string) string {
	s = strings.ToValidUTF8(s, string(utf8.RuneError))
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == PathSeparator:
			b.WriteByte('_')
		case !isInvisible(r):
			b.WriteRune(r)
		}
	}
	out := NormalizeName(b.String())
	if strings.Trim(out, " ") == "" {
		return ""
	}
	return out
}

// isInvisible reports whether r shows as nothing or as a space other than U+0020: the
// categories Cc, Cf, Zl, Zp, and Zs, and the default ignorable code points.
func isInvisible(r rune) bool {
	if r == ' ' {
		return false
	}
	return unicode.In(
		r,
		unicode.Cc,
		unicode.Cf,
		unicode.Zl,
		unicode.Zp,
		unicode.Zs,
		unicode.Other_Default_Ignorable_Code_Point,
		unicode.Variation_Selector,
	)
}
