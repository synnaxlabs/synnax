// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package format

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/synnaxlabs/x/errors"
)

// LicenseTemplatePath is the repo-relative path to the source-of-truth
// license header template, with `{{YEAR}}` as the year placeholder.
const LicenseTemplatePath = "licenses/headers/template.txt"

// License is a Formatter that prepends a per-extension license header.
// The header is read once from licenses/headers/template.txt at
// construction, year-substituted with time.Now().Year(), and rendered
// into the comment style appropriate to each file extension.
//
// Format is idempotent: if the file already starts with a header that
// matches the rendered template (any year, any whitespace following),
// the content is returned unchanged. Files with no header get the
// header prepended; files with a stale (different-year) header get the
// stale header replaced.
type License struct {
	// headersByExt is the rendered header (including trailing blank
	// line) for each supported extension.
	headersByExt map[string]string
	// year is the substituted year, used for staleness detection.
	year string
}

// NewLicense reads the template at <repoRoot>/licenses/headers/template.txt,
// substitutes the current year, and returns a License formatter ready to
// apply that header to files of any of the supported extensions.
func NewLicense(repoRoot string) (*License, error) {
	tmplPath := filepath.Join(repoRoot, LicenseTemplatePath)
	raw, err := os.ReadFile(tmplPath)
	if err != nil {
		return nil, errors.Wrapf(err, "read license template %s", tmplPath)
	}
	year := time.Now().Format("2006")
	body := strings.ReplaceAll(string(raw), "{{YEAR}}", year)
	body = strings.TrimRight(body, "\n")
	lineHash := renderLineComment(body, "#", 2)
	lineSlash := renderLineComment(body, "//", 1)
	block := renderBlockComment(body)
	return &License{
		year: year,
		headersByExt: map[string]string{
			".go":     lineSlash,
			".ts":     lineSlash,
			".tsx":    lineSlash,
			".js":     lineSlash,
			".jsx":    lineSlash,
			".cpp":    lineSlash,
			".cc":     lineSlash,
			".cxx":    lineSlash,
			".h":      lineSlash,
			".hpp":    lineSlash,
			".proto":  lineSlash,
			".oracle": lineSlash,
			".py":     lineHash,
			".css":    block,
		},
	}, nil
}

// Format prepends or refreshes the license header on content. Files with
// extensions outside the supported set pass through unchanged.
func (l *License) Format(content []byte, absPath string) ([]byte, error) {
	header, ok := l.headersByExt[filepath.Ext(absPath)]
	if !ok {
		return content, nil
	}
	if bytes.HasPrefix(content, []byte(header)) {
		return content, nil
	}
	stripped := stripExistingLicenseHeader(content, filepath.Ext(absPath))
	out := make([]byte, 0, len(header)+len(stripped))
	out = append(out, header...)
	out = append(out, stripped...)
	return out, nil
}

// renderLineComment renders body as a sequence of single-line comments
// using prefix and `pad` spaces between the prefix and the line text.
// Empty lines emit just the prefix. The result includes a trailing
// blank line so the next thing in the file is separated from the
// header.
func renderLineComment(body, prefix string, pad int) string {
	var b strings.Builder
	padding := strings.Repeat(" ", pad)
	for _, line := range strings.Split(body, "\n") {
		if line == "" {
			b.WriteString(prefix)
			b.WriteByte('\n')
			continue
		}
		b.WriteString(prefix)
		b.WriteString(padding)
		b.WriteString(line)
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	return b.String()
}

// renderBlockComment renders body as a single C-style block comment.
// Used for languages where line comments aren't appropriate at file
// top (CSS).
func renderBlockComment(body string) string {
	var b strings.Builder
	b.WriteString("/*\n")
	for _, line := range strings.Split(body, "\n") {
		if line == "" {
			b.WriteString(" *\n")
			continue
		}
		b.WriteString(" * ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	b.WriteString(" */\n\n")
	return b.String()
}

// stripExistingLicenseHeader removes a previously-applied (possibly
// stale-year) Synnax license header from the start of content, if one
// is present, returning the remainder. If no header is detected, the
// content is returned unchanged.
//
// Detection is conservative: we only strip when the first line is a
// comment of the appropriate style and contains "Copyright" and
// "Synnax Labs". This avoids ever stripping a non-header comment from
// the top of a file.
func stripExistingLicenseHeader(content []byte, ext string) []byte {
	switch ext {
	case ".css":
		return stripBlockHeader(content)
	case ".py":
		return stripLineHeader(content, "#")
	default:
		return stripLineHeader(content, "//")
	}
}

func stripLineHeader(content []byte, prefix string) []byte {
	lines := bytes.SplitN(content, []byte("\n"), 32)
	if len(lines) == 0 {
		return content
	}
	if !bytes.HasPrefix(lines[0], []byte(prefix)) {
		return content
	}
	if !bytes.Contains(lines[0], []byte("Copyright")) || !bytes.Contains(lines[0], []byte("Synnax Labs")) {
		return content
	}
	var end int
	for i, line := range lines {
		if !bytes.HasPrefix(line, []byte(prefix)) && len(line) != 0 {
			end = i
			break
		}
		if i == len(lines)-1 {
			end = i
		}
	}
	if end == 0 {
		return content
	}
	if end < len(lines) && len(lines[end]) == 0 {
		end++
	}
	rest := bytes.Join(lines[end:], []byte("\n"))
	return rest
}

func stripBlockHeader(content []byte) []byte {
	if !bytes.HasPrefix(content, []byte("/*")) {
		return content
	}
	idx := bytes.Index(content, []byte("*/"))
	if idx == -1 {
		return content
	}
	header := content[:idx+2]
	if !bytes.Contains(header, []byte("Copyright")) || !bytes.Contains(header, []byte("Synnax Labs")) {
		return content
	}
	rest := content[idx+2:]
	rest = bytes.TrimLeft(rest, "\n")
	return rest
}
