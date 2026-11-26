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
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// NeedsRename reports whether the upgrade renames s, because a path cannot hold it: an
// empty name, a name with a '.', or a name with invalid UTF-8. Other stored names stay
// as they are.
func NeedsRename(s string) bool {
	return s == "" || strings.ContainsRune(s, '.') || !utf8.ValidString(s)
}

// NormalizeName returns s in Unicode Normalization Form C.
func NormalizeName(s string) string { return norm.NFC.String(s) }

// SanitizeName returns s rewritten so that NeedsRename reports false for it: each
// invalid byte becomes U+FFFD and each '.' becomes '_', in NFC. It returns "" when s is
// empty.
func SanitizeName(s string) string {
	valid := strings.ToValidUTF8(s, string(utf8.RuneError))
	return NormalizeName(strings.ReplaceAll(valid, ".", "_"))
}
