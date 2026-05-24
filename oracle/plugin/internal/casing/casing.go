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
// input is delegated to lo.SnakeCase so camelCase (clientX), PascalCase
// (PascalCaseField), and identifiers with embedded digit-letter boundaries
// (Int8Value) are split normally.
func FieldSnake(s string) string {
	if isValidSnake(s) {
		return s
	}
	return lo.SnakeCase(s)
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
func isDigit(c byte) bool { return c >= '0' && c <= '9' }
