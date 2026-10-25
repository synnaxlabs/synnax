// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package search

import (
	"strings"
	"unicode"
)

// tokenize splits s into lowercase tokens at whitespace, '_', '-', and '.'.
func tokenize(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), isSeparator)
}

func isSeparator(r rune) bool {
	return unicode.IsSpace(r) || r == '_' || r == '-' || r == '.'
}
