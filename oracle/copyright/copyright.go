// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package copyright provides copyright header management for Oracle schema files.
package copyright

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/synnaxlabs/oracle/paths"
)

const (
	templatePath   = "licenses/headers/template.txt"
	headerLines    = 8 // Number of lines in the copyright header (without trailing blank)
	commentPrefix  = "//"
	yearPlaceholder = "{{YEAR}}"
)

var (
	// Matches "// Copyright YYYY Synnax Labs" at the start of a line
	copyrightPattern = regexp.MustCompile(`^// Copyright (\d{4}) Synnax Labs`)
)

// Ensure ensures the content has a valid, up-to-date copyright header.
// Returns the content with the correct copyright header.
func Ensure(content string) (string, error) {
	header, err := generateHeader()
	if err != nil {
		return content, err
	}

	// Check if content already has the correct header
	if strings.HasPrefix(content, header) {
		return content, nil
	}

	// Check if content has any Synnax copyright header (possibly outdated year)
	if copyrightPattern.MatchString(content) {
		return replaceHeader(content, header), nil
	}

	// No copyright header - prepend one
	return header + "\n" + content, nil
}

// generateHeader creates the copyright header with the current year.
func generateHeader() (string, error) {
	repoRoot, err := paths.RepoRoot()
	if err != nil {
		return "", fmt.Errorf("failed to find repo root: %w", err)
	}

	templateFile := filepath.Join(repoRoot, templatePath)
	templateBytes, err := os.ReadFile(templateFile)
	if err != nil {
		return "", fmt.Errorf("failed to read template: %w", err)
	}

	currentYear := time.Now().Year()
	template := strings.ReplaceAll(string(templateBytes), yearPlaceholder, fmt.Sprintf("%d", currentYear))

	// Convert to // comment style
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(template), "\n") {
		if line == "" {
			lines = append(lines, commentPrefix)
		} else {
			lines = append(lines, commentPrefix+" "+line)
		}
	}

	return strings.Join(lines, "\n") + "\n", nil
}

// replaceHeader replaces an existing copyright header with the new one.
func replaceHeader(content, newHeader string) string {
	lines := strings.Split(content, "\n")

	// Find where the old header ends (first non-comment, non-blank line after copyright)
	headerEnd := 0
	inHeader := false
	for i, line := range lines {
		if copyrightPattern.MatchString(line) {
			inHeader = true
		}
		if inHeader {
			trimmed := strings.TrimSpace(line)
			// Header continues while we have comment lines or blank lines
			if strings.HasPrefix(trimmed, "//") || trimmed == "" {
				headerEnd = i + 1
			} else {
				break
			}
		}
	}

	// Skip any blank lines after the old header
	for headerEnd < len(lines) && strings.TrimSpace(lines[headerEnd]) == "" {
		headerEnd++
	}

	// Reconstruct: new header + blank line + rest of content
	rest := strings.Join(lines[headerEnd:], "\n")
	if rest == "" {
		return newHeader
	}
	return newHeader + "\n" + rest
}
