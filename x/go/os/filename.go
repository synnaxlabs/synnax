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

	"golang.org/x/text/unicode/norm"
)

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
// drops trailing dots and spaces, and prefixes an underscore to a Windows device name.
//
// It returns an empty string for a name that sanitizes to nothing, such as one holding
// dots and spaces alone. The caller decides what a nameless file means: substituting a
// placeholder here would invent a name the caller never gave.
//
// The result is a single path element, but it is not unique: two names can sanitize to
// one. Callers that need distinct files must compare the results with FoldFileName.
func SanitizeFileName(name string) string {
	name = unsafeFileNameChars.ReplaceAllString(name, "_")
	// Windows drops trailing dots and spaces, so a name keeping them addresses a
	// different file than the one the caller asked for.
	name = strings.TrimRight(name, ". ")
	if reservedFileNames.MatchString(name) {
		return "_" + name
	}
	return name
}

// FoldFileName reduces name to the form two file names must be compared in:
// Unicode-normalized and case-folded. Names differing only by case or by Unicode
// composition address the same file on macOS and Windows, so comparing the raw strings
// reports two files where the filesystem has one.
func FoldFileName(name string) string { return strings.ToLower(norm.NFC.String(name)) }
