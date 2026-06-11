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

import (
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/set"
)

// acronyms are initialisms that stay fully upper-cased in PascalCase identifiers
// derived from snake_case (e.g. "ai_voltage" -> "AIVoltage"). The set covers the
// hardware and signal initialisms used in Synnax schemas.
var acronyms = set.New(
	"ai", "ao", "ci", "co", "di", "do",
	"rtd", "iepe", "rms", "dc", "ac",
)

// PascalAcronym converts a snake_case (or mixed-case) identifier to PascalCase,
// fully upper-casing any whole underscore-delimited segment that is a known
// acronym. Whole-segment matching means a word that merely contains an acronym
// (e.g. "email", "accel") is title-cased normally rather than mangled.
func PascalAcronym(s string) string {
	segs := strings.Split(TypeSnake(s), "_")
	for i, seg := range segs {
		if seg == "" {
			continue
		}
		if acronyms.Contains(strings.ToLower(seg)) {
			segs[i] = strings.ToUpper(seg)
		} else {
			segs[i] = lo.Capitalize(seg)
		}
	}
	return strings.Join(segs, "")
}

// VariantTypeName derives a discriminated-union variant's type name from the
// union's type name and the variant's discriminator value. When the variant
// repeats the union's leading acronym (e.g. value "ai_voltage" under union
// "AIChannel"), that acronym is factored out so the name reads naturally as
// "AIVoltageChannel" rather than the doubled "AIChannelAIVoltage". Otherwise the
// union name prefixes the PascalCased variant (e.g. "linear" under "Scale" ->
// "ScaleLinear"), which keeps every variant unique and avoids colliding with a
// field struct that shares the variant's bare name.
func VariantTypeName(unionName, variantValue string) string {
	variant := PascalAcronym(variantValue)
	if acr := leadingAcronym(unionName); acr != "" &&
		strings.HasPrefix(strings.ToLower(variant), strings.ToLower(acr)) {
		return variant + unionName[len(acr):]
	}
	return unionName + variant
}

// leadingAcronym returns the run of leading uppercase letters of s that form an
// acronym, excluding the final uppercase letter when it begins the next
// PascalCase word ("AIChannel" -> "AI", "RTDConfig" -> "RTD", "Scale" -> "").
func leadingAcronym(s string) string {
	n := 0
	for n < len(s) && isUpper(s[n]) {
		n++
	}
	if n < len(s) && n > 0 && isLower(s[n]) {
		n--
	}
	if n < 2 {
		return ""
	}
	return s[:n]
}

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

// TypePascal converts an identifier to PascalCase, honoring the same
// acronym-followed-by-word boundaries TypeSnake recovers. SetXChannel stays
// SetXChannel rather than collapsing into SetXchannel. Use this instead of
// lo.PascalCase whenever the input might already be PascalCase with adjacent
// capital letters.
func TypePascal(s string) string {
	return lo.PascalCase(TypeSnake(s))
}

// TypeCamel converts an identifier to camelCase, honoring the same boundaries
// as TypePascal. setXChannel rather than setXchannel.
func TypeCamel(s string) string {
	return lo.CamelCase(TypeSnake(s))
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
