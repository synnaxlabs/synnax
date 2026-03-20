// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pluralize

import "strings"

var irregulars = map[string]string{
	"child":     "children",
	"person":    "people",
	"mouse":     "mice",
	"goose":     "geese",
	"tooth":     "teeth",
	"foot":      "feet",
	"man":       "men",
	"woman":     "women",
	"ox":        "oxen",
	"datum":     "data",
	"criterion": "criteria",
	"index":     "indices",
	"matrix":    "matrices",
	"vertex":    "vertices",
	"appendix":  "appendices",
}

// irregularPlurals is the reverse map for already-plural detection.
var irregularPlurals map[string]bool

func init() {
	irregularPlurals = make(map[string]bool, len(irregulars))
	for _, v := range irregulars {
		irregularPlurals[v] = true
	}
}

var uncountable = map[string]bool{
	"sheep":    true,
	"fish":     true,
	"series":   true,
	"species":  true,
	"data":     true,
	"metadata": true,
	"info":     true,
	"software": true,
	"hardware": true,
	"firmware": true,
	"deer":     true,
	"moose":    true,
	"aircraft": true,
}

// words ending in consonant+o that take -es
var oEsWords = map[string]bool{
	"hero":    true,
	"potato":  true,
	"tomato":  true,
	"echo":    true,
	"torpedo": true,
	"veto":    true,
	"volcano": true,
}

// words ending in f that change to -ves
var fVesWords = map[string]bool{
	"leaf":  true,
	"shelf": true,
	"half":  true,
	"wolf":  true,
	"calf":  true,
	"loaf":  true,
	"thief": true,
	"self":  true,
	"elf":   true,
}

// String returns the plural form of the given word. It handles common English
// pluralization rules including irregular forms, uncountable words, and
// already-plural detection.
func String(name string) string {
	if len(name) == 0 {
		return name
	}

	lower := strings.ToLower(name)

	if uncountable[lower] {
		return name
	}

	if irregularPlurals[lower] {
		return name
	}

	if plural, ok := irregulars[lower]; ok {
		return matchCase(name, plural)
	}

	if isAlreadyPlural(lower) {
		return name
	}

	if strings.HasSuffix(lower, "fe") {
		base := name[:len(name)-2]
		return base + matchSuffixCase(name, "ves")
	}
	if fVesWords[lower] {
		base := name[:len(name)-1]
		return base + matchSuffixCase(name, "ves")
	}

	if strings.HasSuffix(lower, "z") {
		return name + matchSuffixCase(name, "zes")
	}

	if oEsWords[lower] {
		return name + matchSuffixCase(name, "es")
	}

	if len(name) > 1 && lower[len(lower)-1] == 'y' {
		// All-caps abbreviations (e.g. "XY", "ID") just get "s"
		if isAllUpper(name) {
			return name + "s"
		}
		if !isVowel(lower[len(lower)-2]) {
			return name[:len(name)-1] + matchSuffixCase(name, "ies")
		}
		return name + matchSuffixCase(name, "s")
	}

	if lower[len(lower)-1] == 's' ||
		lower[len(lower)-1] == 'x' ||
		strings.HasSuffix(lower, "ch") ||
		strings.HasSuffix(lower, "sh") {
		return name + matchSuffixCase(name, "es")
	}

	return name + matchSuffixCase(name, "s")
}

// isAlreadyPlural checks whether a lowercased word appears to already be plural
// by testing whether it could have been produced by our own pluralization rules.
func isAlreadyPlural(lower string) bool {
	n := len(lower)

	// words ending in "ies" could be plural of a consonant+y word (e.g. "authorities")
	if strings.HasSuffix(lower, "ies") && n > 3 {
		return true
	}

	// words ending in "ves" could be plural of f/fe words (e.g. "wolves")
	if strings.HasSuffix(lower, "ves") && n > 3 {
		return true
	}

	// words ending in "ses", "xes", "zes", "ches", "shes" are plural of sibilants
	if strings.HasSuffix(lower, "ses") ||
		strings.HasSuffix(lower, "xes") ||
		strings.HasSuffix(lower, "zes") ||
		strings.HasSuffix(lower, "ches") ||
		strings.HasSuffix(lower, "shes") {
		return true
	}

	// words ending in "s" but not "ss" could be regular plurals.
	// Check: if removing the trailing "s" gives a word ending in a consonant
	// (not s, x, z, ch, sh), it's likely already plural.
	if n > 2 && lower[n-1] == 's' && lower[n-2] != 's' {
		stem := lower[:n-1]
		stemEnd := stem[len(stem)-1]
		// If the stem doesn't end in a sibilant, this looks like a regular plural.
		if stemEnd != 's' && stemEnd != 'x' && stemEnd != 'z' &&
			!strings.HasSuffix(stem, "ch") && !strings.HasSuffix(stem, "sh") {
			// But singular words ending in 's' (like "status", "alias", "bus")
			// also end in 's'. We need to distinguish. A good heuristic: if the
			// last char before 's' is a vowel + consonant pattern that forms a
			// common singular ending, it's likely singular. For our use case,
			// we check if removing the 's' leaves a word ending in a common
			// singular suffix like "us", "is", "as".
			if strings.HasSuffix(stem, "u") || strings.HasSuffix(stem, "i") || strings.HasSuffix(stem, "a") {
				// e.g. "status" ends in "us", "alias" ends in "as" - these are singular
				return false
			}
			return true
		}
	}

	return false
}

func isVowel(b byte) bool {
	switch b {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	}
	return false
}

// matchCase transforms replacement to match the overall case pattern of original.
func matchCase(original, replacement string) string {
	if len(original) == 0 || len(replacement) == 0 {
		return replacement
	}
	if isAllUpper(original) {
		return strings.ToUpper(replacement)
	}
	if isUpper(original[0]) {
		return strings.ToUpper(replacement[:1]) + replacement[1:]
	}
	return replacement
}

// matchSuffixCase transforms a suffix to match the case pattern of the end of the word.
func matchSuffixCase(word, suffix string) string {
	if len(word) == 0 {
		return suffix
	}
	if isAllUpper(word) {
		return strings.ToUpper(suffix)
	}
	return suffix
}

func isUpper(b byte) bool {
	return b >= 'A' && b <= 'Z'
}

func isAllUpper(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 'a' && s[i] <= 'z' {
			return false
		}
	}
	return true
}
