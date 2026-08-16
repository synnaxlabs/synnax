// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package filename

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

const (
	// maxLength is the longest single path element ext4, APFS, and NTFS accept. It
	// counts bytes, which bounds NTFS's UTF-16 limit too: a rune never takes more
	// UTF-16 code units than it takes UTF-8 bytes.
	maxLength = 255
	// placeholder names a file whose name sanitizes to nothing.
	placeholder = "_"
)

var (
	// unsafeChars matches the path separators, the characters Windows reserves, and the
	// control characters Windows forbids.
	unsafeChars = regexp.MustCompile(`[/\\<>:"|?*\x00-\x1f]`)
	// reservedNames matches the device names Windows refuses to open a file under, bare
	// or carrying an extension.
	reservedNames = regexp.MustCompile(
		`(?i)^(con|prn|aux|nul|com[1-9]|lpt[1-9])(\.|$)`,
	)
)

// Sanitize turns a user-supplied name into a file name that writes to disk on any
// platform, carrying extension. It replaces every character a file name cannot hold
// with an underscore, drops trailing dots and spaces, prefixes an underscore to a
// Windows device name, and shortens the name until it and extension together fit the
// longest path element a filesystem takes. Pass an empty extension for a name that
// carries none.
//
// A name that sanitizes to nothing, such as one holding dots and spaces alone, comes
// back as a single underscore.
//
// It returns a validation error for an extension that fills a file name by itself,
// which no name can rescue.
//
// The result is a single path element, but it is not unique: two names can sanitize to
// one, and shortening makes that more likely. Callers that need distinct files must
// compare the results with Fold.
func Sanitize(name, extension string) (string, error) {
	budget := maxLength - len(extension)
	if budget <= 0 {
		return "", errors.Wrapf(
			validate.ErrValidation,
			"extension %q leaves no room for a file name",
			extension,
		)
	}
	name = fit(unsafeChars.ReplaceAllString(name, "_"), budget)
	if reservedNames.MatchString(name) {
		// Hold a byte back for the prefix so the whole name still fits.
		name = "_" + fit(name, budget-1)
	}
	if name == "" {
		name = placeholder
	}
	return name + extension, nil
}

// WithSuffix inserts suffix between name's base and extension, shortening the base
// until the result fits the longest path element a filesystem takes. name must carry
// extension, as Sanitize returns it; pass an empty extension for a name without one.
func WithSuffix(name, suffix, extension string) string {
	base := strings.TrimSuffix(name, extension)
	base = fit(base, maxLength-len(suffix)-len(extension))
	return base + suffix + extension
}

// fit shortens name to maxBytes bytes, cutting on a rune boundary, and drops the
// trailing dots and spaces Windows drops. It trims after cutting because the cut can
// expose a dot or a space the original name buried.
func fit(name string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	for len(name) > maxBytes {
		_, size := utf8.DecodeLastRuneInString(name)
		name = name[:len(name)-size]
	}
	return strings.TrimRight(name, ". ")
}

// Fold reduces name to the form two file names must be compared in: case-folded and
// Unicode-normalized. Names differing only by case or by Unicode composition address
// the same file on macOS and Windows, so comparing the raw strings reports two files
// where the filesystem has one.
func Fold(name string) string {
	// Folding is stateful, so it cannot be shared between goroutines. It can also
	// denormalize, so the result is normalized again.
	return norm.NFC.String(cases.Fold().String(norm.NFC.String(name)))
}
