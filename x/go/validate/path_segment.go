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
	if !utf8.ValidString(s) || strings.Trim(s, " ") == "" ||
		!norm.NFC.IsNormalString(s) {
		return false
	}
	for _, r := range s {
		if r == pathSeparator || isInvisible(r) {
			return false
		}
	}
	return true
}

// NormalizeName returns s in Unicode Normalization Form C, the one form in which names
// are stored and compared.
func NormalizeName(s string) string { return norm.NFC.String(s) }

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
