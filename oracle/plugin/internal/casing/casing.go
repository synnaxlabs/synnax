// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package casing provides shared snake/camel conversion helpers used across oracle
// plugins. Field-name conversions go through FieldSnake, which preserves the
// author's spelling when the field is already a valid snake_case identifier so
// domain names like axis keys (x1, y1) are not split on letter-digit boundaries.
package casing

import "github.com/samber/lo"

// FieldSnake converts a schema field name to its canonical snake_case wire form.
// When the input is already a valid snake_case identifier (starts with a lowercase
// letter, contains only lowercase letters, digits, and underscores) it is returned
// unchanged so domain names like x1 or y1 survive the conversion. Otherwise the
// input is delegated to TypeSnake so camelCase (clientX), PascalCase
// (PascalCaseField), and identifiers with embedded digit-letter boundaries
// (Int8Value) are split normally.
func FieldSnake(s string) string {
	if isValidSnake(s) {
		return s
	}
	return TypeSnake(s)
}

// TypeSnake converts a PascalCase or mixed-case identifier to snake_case,
// honoring acronym-followed-by-word boundaries that lo.SnakeCase misses. Inputs
// like "SetXChannel" or "URLValue" become "set_x_channel" / "url_value" rather
// than lo's collapsed "set_xchannel" / "urlvalue". Already-snake input is
// returned unchanged.
//
// Implementation note: lo.SnakeCase's underlying word splitter handles
// lower-then-upper boundaries (tX) and the [A-Z][A-Z][a-z] acronym-end
// boundary, but the second pattern only fires when the regex engine has
// already advanced past the cap-cap pair via an earlier split. Inserting a
// separator before any cap-cap-lower run nudges the splitter to see the
// boundary on the next pass.
func TypeSnake(s string) string {
	if isValidSnake(s) {
		return s
	}
	return lo.SnakeCase(insertAcronymBoundaries(s))
}

// insertAcronymBoundaries returns s with a space inserted before any uppercase
// letter that is preceded by another uppercase letter and followed by a
// lowercase letter, so lo.SnakeCase splits at the word the acronym is hugging.
func insertAcronymBoundaries(s string) string {
	if len(s) < 3 {
		return s
	}
	out := make([]byte, 0, len(s)+4)
	for i := 0; i < len(s); i++ {
		if i > 0 && i+1 < len(s) &&
			isUpper(s[i-1]) && isUpper(s[i]) && isLower(s[i+1]) {
			out = append(out, ' ')
		}
		out = append(out, s[i])
	}
	return string(out)
}

// isValidSnake reports whether s is a non-empty identifier composed of a leading
// lowercase letter followed by any mix of lowercase letters, digits, and
// underscores.
func isValidSnake(s string) bool {
	if s == "" {
		return false
	}
	if !isLower(s[0]) {
		return false
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		if !isLower(c) && !isDigit(c) && c != '_' {
			return false
		}
	}
	return true
}

func isLower(c byte) bool { return c >= 'a' && c <= 'z' }
func isUpper(c byte) bool { return c >= 'A' && c <= 'Z' }
func isDigit(c byte) bool { return c >= '0' && c <= '9' }
