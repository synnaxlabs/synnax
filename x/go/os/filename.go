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

// MaxFileNameLength is the longest single path element ext4, APFS, and NTFS accept. It
// counts bytes, which bounds NTFS's UTF-16 limit too: a rune never takes more UTF-16
// code units than it takes UTF-8 bytes.
const MaxFileNameLength = 255

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

// SanitizeFileName turns a user-supplied name into one that writes to disk on any
// platform. It replaces every character a file name cannot hold with an underscore,
// drops trailing dots and spaces, prefixes an underscore to a Windows device name, and
// shortens the result to maxBytes bytes.
//
// Pass MaxFileNameLength as maxBytes for a whole file name. A caller that appends an
// extension must hold that many bytes back, because the extension counts against the
// same limit.
//
// It returns an empty string for a name that sanitizes to nothing, such as one holding
// dots and spaces alone. The caller decides what a nameless file means: substituting a
// placeholder here would invent a name the caller never gave.
//
// The result is a single path element, but it is not unique: two names can sanitize to
// one, and shortening makes that more likely. Callers that need distinct files must
// compare the results with FoldFileName.
func SanitizeFileName(name string, maxBytes int) string {
	name = fitFileName(unsafeFileNameChars.ReplaceAllString(name, "_"), maxBytes)
	if reservedFileNames.MatchString(name) {
		// Hold a byte back for the prefix so the result still fits.
		return "_" + fitFileName(name, maxBytes-1)
	}
	return name
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
