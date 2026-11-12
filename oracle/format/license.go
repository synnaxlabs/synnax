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
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/synnaxlabs/x/errors"
)

// licenseTemplatePath is the repo-relative path to the source-of-truth
// license header template, with `{{YEAR}}` as the year placeholder.
const licenseTemplatePath = "licenses/headers/template.txt"

// headerYear matches the year in a Synnax license header.
var headerYear = regexp.MustCompile(`Copyright (\d{4}) Synnax Labs`)

// License is a formatter that prepends a per-extension license header rendered from
// licenses/headers/template.txt.
//
// The header keeps the year of the file on disk at the formatted path when that file
// starts with a valid header. Otherwise the header uses the year passed to NewLicense.
// Format replaces any other Synnax header on content and leaves a valid one unchanged.
type License struct {
	template string
	year     string
	prior    map[string]string
}

// NewLicense reads the template at <repoRoot>/licenses/headers/template.txt and
// returns a License formatter that gives new files the header for year. It reads the
// header years of the absolute paths in prior now, so Format keeps them after a tool
// such as buf rewrites those files without a header.
func NewLicense(repoRoot string, year int, prior []string) (*License, error) {
	tmplPath := filepath.Join(repoRoot, licenseTemplatePath)
	raw, err := os.ReadFile(tmplPath)
	if err != nil {
		return nil, errors.Wrapf(err, "read license template %s", tmplPath)
	}
	l := &License{
		template: strings.TrimRight(string(raw), "\n"),
		year:     strconv.Itoa(year),
		prior:    make(map[string]string, len(prior)),
	}
	for _, p := range prior {
		ext := filepath.Ext(p)
		if _, ok := renderers[ext]; !ok {
			continue
		}
		if l.prior[p], err = l.diskYear(p, ext); err != nil {
			return nil, err
		}
	}
	return l, nil
}

// Format prepends or refreshes the license header on content. Files with extensions
// outside the supported set pass through unchanged. Format reads the file at absPath,
// if one exists and NewLicense did not read it, to find the year to keep.
func (l *License) Format(
	_ context.Context,
	content []byte,
	absPath string,
) ([]byte, error) {
	ext := filepath.Ext(absPath)
	if _, ok := renderers[ext]; !ok {
		return content, nil
	}
	year, ok := l.prior[absPath]
	if !ok {
		var err error
		if year, err = l.diskYear(absPath, ext); err != nil {
			return nil, err
		}
	}
	header := l.render(ext, year)
	if bytes.HasPrefix(content, []byte(header)) {
		return content, nil
	}
	stripped := stripExistingLicenseHeader(content, ext)
	out := make([]byte, 0, len(header)+len(stripped))
	out = append(out, header...)
	out = append(out, stripped...)
	return out, nil
}

// diskYear returns the year of the valid header at the start of the file at absPath,
// or l.year when the file does not exist or has no valid header.
func (l *License) diskYear(absPath, ext string) (string, error) {
	existing, err := os.ReadFile(absPath)
	if errors.Is(err, fs.ErrNotExist) {
		return l.year, nil
	}
	if err != nil {
		return "", errors.Wrapf(err, "read %s", absPath)
	}
	m := headerYear.FindSubmatch(existing)
	if m == nil || !bytes.HasPrefix(existing, []byte(l.render(ext, string(m[1])))) {
		return l.year, nil
	}
	return string(m[1]), nil
}

func (l *License) render(ext, year string) string {
	return renderers[ext](strings.ReplaceAll(l.template, "{{YEAR}}", year))
}

var (
	lineSlash = func(body string) string { return renderLineComment(body, "//", 1) }
	lineHash  = func(body string) string { return renderLineComment(body, "#", 2) }
	renderers = map[string]func(string) string{
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
		".css":    renderBlockComment,
	}
)

// renderLineComment renders body as a sequence of single-line comments
// using prefix and `pad` spaces between the prefix and the line text.
// Empty lines emit just the prefix. The result includes a trailing
// blank line so the next thing in the file is separated from the
// header.
func renderLineComment(body, prefix string, pad int) string {
	var b strings.Builder
	padding := strings.Repeat(" ", pad)
	for line := range strings.SplitSeq(body, "\n") {
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
	for line := range strings.SplitSeq(body, "\n") {
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
	lines := bytes.Split(content, []byte("\n"))
	if len(lines) == 0 {
		return content
	}
	if !bytes.HasPrefix(lines[0], []byte(prefix)) {
		return content
	}
	if !bytes.Contains(lines[0], []byte("Copyright")) ||
		!bytes.Contains(lines[0], []byte("Synnax Labs")) {
		return content
	}
	// Walk forward through the contiguous comment block. The license
	// header is one block of // lines (some possibly //-only blanks)
	// terminated by a non-comment line. We stop there, then consume a
	// single blank separator if one follows. Without the blank-line
	// stop, this loop would chew through any subsequent comment block
	// (e.g., a protoc-gen-go "Code generated" banner) and silently
	// strip it along with the stale header.
	end := len(lines)
	for i, line := range lines {
		if !bytes.HasPrefix(line, []byte(prefix)) {
			end = i
			break
		}
	}
	if end < len(lines) && len(lines[end]) == 0 {
		end++
	}
	return bytes.Join(lines[end:], []byte("\n"))
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
	if !bytes.Contains(header, []byte("Copyright")) ||
		!bytes.Contains(header, []byte("Synnax Labs")) {
		return content
	}
	rest := content[idx+2:]
	rest = bytes.TrimLeft(rest, "\n")
	return rest
}
