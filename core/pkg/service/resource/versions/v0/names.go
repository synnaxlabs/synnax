// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// IsLegalName reports whether s passes the v63 name rule: valid UTF-8 in NFC, not empty
// after trimming spaces, and with no '.' and no character that shows as nothing or as a
// space other than U+0020.
func IsLegalName(s string) bool {
	if !utf8.ValidString(s) || strings.Trim(s, " ") == "" || NormalizeName(s) != s {
		return false
	}
	return !strings.ContainsFunc(
		s,
		func(r rune) bool { return r == '.' || isInvisible(r) },
	)
}

// NormalizeName returns s in Unicode Normalization Form C.
func NormalizeName(s string) string { return norm.NFC.String(s) }

// SanitizeName returns s rewritten to pass IsLegalName: each invalid byte becomes
// U+FFFD, each '.' becomes '_', and each refused character goes. A run of combining
// characters that NFC breaks with U+034F loses its characters from the break on. It
// returns "" when only spaces are left.
func SanitizeName(s string) string {
	out := sanitize(s)
	// Each pass drops the joiner that NFC put in and the character after it.
	for {
		i := strings.IndexRune(out, combiningGraphemeJoiner)
		if i < 0 {
			break
		}
		rest := out[i+utf8.RuneLen(combiningGraphemeJoiner):]
		_, size := utf8.DecodeRuneInString(rest)
		out = sanitize(out[:i] + rest[size:])
	}
	if strings.Trim(out, " ") == "" {
		return ""
	}
	return out
}

// combiningGraphemeJoiner is U+034F, which NFC puts in a long run of characters that
// combine with the one before them.
const combiningGraphemeJoiner = '\u034F'

// sanitize returns s in NFC, with each invalid byte as U+FFFD, each '.' as '_', and no
// refused character before NFC.
func sanitize(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '.':
			b.WriteByte('_')
		case !isInvisible(r):
			b.WriteRune(r)
		}
	}
	return NormalizeName(b.String())
}

// isInvisible reports whether r shows as nothing or as a space other than U+0020: the
// categories Cc, Cf, Zl, Zp, and Zs, the default ignorable code points, the variation
// selectors, and U+2800.
func isInvisible(r rune) bool {
	return r != ' ' && (r == '\u2800' || unicode.In(
		r,
		unicode.Cc,
		unicode.Cf,
		unicode.Zl,
		unicode.Zp,
		unicode.Zs,
		unicode.Other_Default_Ignorable_Code_Point,
		unicode.Variation_Selector,
	))
}
