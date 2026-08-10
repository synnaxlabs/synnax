// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package os

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// maxFileNameLength is the longest single path element ext4, APFS, and NTFS accept. It
// counts bytes, which bounds NTFS's UTF-16 limit too: a rune never takes more UTF-16
// code units than it takes UTF-8 bytes.
const maxFileNameLength = 255

var (
	// unsafeFileNameChars matches the path separators, the control characters, and the
	// characters Windows reserves.
	unsafeFileNameChars = regexp.MustCompile(`[/\\<>:"|?*\x00-\x1f\x7f]`)
	// reservedFileNames matches the device names Windows refuses to open a file under,
	// bare or carrying an extension.
	reservedFileNames = regexp.MustCompile(
		`(?i)^(con|prn|aux|nul|com[1-9]|lpt[1-9])(\.|$)`,
	)
)

// SanitizeFileName turns a user-supplied name into a file name that writes to disk on
// any platform, carrying extension. It replaces every character a file name cannot hold
// with an underscore, drops trailing dots and spaces, prefixes an underscore to a
// Windows device name, and shortens the name until it and extension together fit the
// longest path element a filesystem takes. Pass an empty extension for a name that
// carries none.
//
// It returns an empty string for a name that sanitizes to nothing, such as one holding
// dots and spaces alone, rather than a file named by its extension. The caller decides
// what a nameless file means: substituting a placeholder here would invent a name the
// caller never gave.
//
// It panics on an extension that fills a file name by itself, which no name can rescue.
// Extensions come from codecs, not from users, so a caller cannot answer for one and an
// empty return would blame the name instead.
//
// The result is a single path element, but it is not unique: two names can sanitize to
// one, and shortening makes that more likely. Callers that need distinct files must
// compare the results with FoldFileName.
func SanitizeFileName(name, extension string) string {
	budget := maxFileNameLength - len(extension)
	if budget <= 0 {
		panic("[x/os] - extension leaves no room for a file name: " + extension)
	}
	name = fitFileName(unsafeFileNameChars.ReplaceAllString(name, "_"), budget)
	if reservedFileNames.MatchString(name) {
		// Hold a byte back for the prefix so the whole name still fits.
		name = "_" + fitFileName(name, budget-1)
	}
	if name == "" {
		return ""
	}
	return name + extension
}

// fitFileName shortens name to maxBytes bytes, cutting on a rune boundary, and drops
// the trailing dots and spaces Windows drops. It trims after cutting because the cut
// can expose a dot or a space the original name buried.
func fitFileName(name string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	for len(name) > maxBytes {
		_, size := utf8.DecodeLastRuneInString(name)
		name = name[:len(name)-size]
	}
	return strings.TrimRight(name, ". ")
}

// FoldFileName reduces name to the form two file names must be compared in:
// Unicode-normalized and case-folded. Names differing only by case or by Unicode
// composition address the same file on macOS and Windows, so comparing the raw strings
// reports two files where the filesystem has one.
func FoldFileName(name string) string { return strings.ToLower(norm.NFC.String(name)) }
