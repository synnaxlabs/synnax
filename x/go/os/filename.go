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

// unsafeFileNameChars matches the path separators and the characters Windows reserves.
var unsafeFileNameChars = regexp.MustCompile(`[/\\<>:"|?*]`)

// SanitizeFileName replaces every character a file name cannot hold with an underscore,
// turning a user-supplied name into one that writes to disk on any platform.
func SanitizeFileName(name string) string {
	return unsafeFileNameChars.ReplaceAllString(name, "_")
}

// FoldFileName reduces name to the form two file names must be compared in:
// Unicode-normalized and case-folded. Names differing only by case or by Unicode
// composition address the same file on macOS and Windows, so comparing the raw strings
// reports two files where the filesystem has one.
func FoldFileName(name string) string { return strings.ToLower(norm.NFC.String(name)) }
