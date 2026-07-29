// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package doc provides utilities for extracting documentation from oracle schemas.
package doc

import (
	"strings"
	"unicode"

	"github.com/synnaxlabs/oracle/resolution"
)

// Get extracts documentation from a domain map.
// It looks for a "doc" domain and returns the first expression's value or name.
// Returns an empty string if no documentation is defined.
func Get(domains map[string]resolution.Domain) string {
	if domain, ok := domains["doc"]; ok {
		if len(domain.Expressions) > 0 {
			expr := domain.Expressions[0]
			if len(expr.Values) > 0 {
				return expr.Values[0].StringValue
			}
			return expr.Name
		}
	}
	return ""
}

const maxLineWidth = 88

// wrapText takes documentation text and wraps it to fit within the given width.
// It normalizes internal newlines (single \n) into spaces, but preserves paragraph
// breaks (double \n\n). Returns a slice of wrapped lines.
func wrapText(text string, firstLineWidth, subsequentLineWidth int) []string {
	if text == "" {
		return nil
	}

	// Split by paragraph breaks (double newline)
	paragraphs := strings.Split(text, "\n\n")
	var allLines []string

	for pIdx, para := range paragraphs {
		// Normalize single newlines to spaces within the paragraph
		para = strings.ReplaceAll(para, "\n", " ")
		// Collapse multiple spaces
		words := strings.Fields(para)
		if len(words) == 0 {
			if pIdx > 0 {
				allLines = append(allLines, "")
			}
			continue
		}

		var lines []string
		var currentLine strings.Builder

		for _, word := range words {
			width := firstLineWidth
			if len(allLines)+len(lines) > 0 {
				width = subsequentLineWidth
			}

			if currentLine.Len() == 0 {
				currentLine.WriteString(word)
			} else if currentLine.Len()+1+len(word) <= width {
				currentLine.WriteString(" ")
				currentLine.WriteString(word)
			} else {
				lines = append(lines, currentLine.String())
				currentLine.Reset()
				currentLine.WriteString(word)
			}
		}
		if currentLine.Len() > 0 {
			lines = append(lines, currentLine.String())
		}

		// Add paragraph separator if not the first paragraph
		if pIdx > 0 && len(lines) > 0 {
			allLines = append(allLines, "")
		}
		allLines = append(allLines, lines...)
	}

	return allLines
}

// FormatGo formats documentation for Go comments.
// Single-line: "// Name doc text"
// Multi-line: "// Name line1\n// line2\n// line3"
// Text is wrapped to 88 columns including the comment prefix and the display width
// of the indentation the comment is emitted at (gofmt indents every comment line).
func FormatGo(name, doc string, indent ...int) string {
	if doc == "" {
		return ""
	}

	width := maxLineWidth
	if len(indent) > 0 {
		width -= indent[0]
	}

	// Calculate available width: "// Name " for first line, "// " for subsequent
	firstPrefix := "// " + name + " "
	subsequentPrefix := "// "
	firstLineWidth := width - len(firstPrefix)
	subsequentLineWidth := width - len(subsequentPrefix)

	lines := wrapText(doc, firstLineWidth, subsequentLineWidth)
	if len(lines) == 0 {
		return ""
	}

	var result []string
	for i, line := range lines {
		if i == 0 {
			result = append(result, firstPrefix+line)
		} else if line == "" {
			result = append(result, "//")
		} else {
			result = append(result, subsequentPrefix+line)
		}
	}
	return strings.Join(result, "\n")
}

// FormatTS formats documentation for TypeScript JSDoc comments.
//
// Single-line: "/** Name doc text */"
//
// Multi-line: "/**\n * Name line1\n * line2\n */"
//
// Text is wrapped to 88 characters including the comment prefix and the indentation the
// comment is emitted at (Prettier re-indents continuation lines to match).
func FormatTS(name, doc string, indent ...int) string {
	if doc == "" {
		return ""
	}

	width := maxLineWidth
	if len(indent) > 0 {
		width -= indent[0]
	}
	firstPrefix := " * " + name + " "
	subsequentPrefix := " * "
	lines := wrapText(doc, width-len(firstPrefix), width-len(subsequentPrefix))
	if len(lines) == 0 {
		return ""
	}

	if len(lines) == 1 {
		if single := "/** " + name + " " + lines[0] + " */"; len(single) <= width {
			return single
		}
	}

	result := []string{"/**"}
	for i, line := range lines {
		if i == 0 {
			result = append(result, firstPrefix+line)
		} else if line == "" {
			result = append(result, " *")
		} else {
			result = append(result, subsequentPrefix+line)
		}
	}
	result = append(result, " */")
	return strings.Join(result, "\n")
}

// FormatPyDocstring formats documentation for Python class/function docstrings.
//
// Single-line: `"""Name doc text"""`
//
// Multi-line: `"""Name line1\nline2\nline3"""`
//
// Text is wrapped to 88 characters including the docstring markers.
func FormatPyDocstring(name, doc string) string {
	if doc == "" {
		return ""
	}
	// Reserve room for the closing quotes, which land on the last wrapped line.
	firstPrefix := `"""` + name + " "
	closing := len(`"""`)
	lines := wrapText(doc, maxLineWidth-len(firstPrefix)-closing, maxLineWidth-closing)
	if len(lines) == 0 {
		return ""
	}
	result := make([]string, len(lines))
	for i, line := range lines {
		if i == 0 {
			result[i] = firstPrefix + line
		} else {
			result[i] = line
		}
	}
	result[len(result)-1] = result[len(result)-1] + `"""`
	return strings.Join(result, "\n")
}

// FormatPyComment formats documentation for Python line comments.
//
// Single-line: "# Name doc text"
//
// Multi-line: "# Name line1\n# line2\n# line3"
//
// Text is wrapped to 88 characters including the comment prefix.
func FormatPyComment(name, doc string) string {
	if doc == "" {
		return ""
	}
	firstPrefix := "# " + name + " "
	subsequentPrefix := "# "
	lines := wrapText(
		doc,
		maxLineWidth-len(firstPrefix),
		maxLineWidth-len(subsequentPrefix),
	)
	var result []string
	for i, line := range lines {
		if i == 0 {
			result = append(result, firstPrefix+line)
		} else if line == "" {
			result = append(result, "#")
		} else {
			result = append(result, subsequentPrefix+line)
		}
	}
	return strings.Join(result, "\n")
}

// FormatCpp formats documentation for C++ Doxygen-style comments.
//
// Single-line: "/// @brief Name doc text"
//
// Multi-line: "/// @brief Name line1\n/// line2\n/// line3"
//
// Text is wrapped to 88 characters including the comment prefix.
func FormatCpp(name, doc string) string {
	if doc == "" {
		return ""
	}

	// Calculate available width: "/// @brief Name " for first line, "/// " for subsequent
	firstPrefix := "/// @brief " + name + " "
	subsequentPrefix := "/// "
	firstLineWidth := maxLineWidth - len(firstPrefix)
	subsequentLineWidth := maxLineWidth - len(subsequentPrefix)

	lines := wrapText(doc, firstLineWidth, subsequentLineWidth)
	if len(lines) == 0 {
		return ""
	}

	var result []string
	for i, line := range lines {
		if i == 0 {
			result = append(result, firstPrefix+line)
		} else if line == "" {
			result = append(result, "///")
		} else {
			result = append(result, subsequentPrefix+line)
		}
	}
	return strings.Join(result, "\n")
}

// FormatProto formats documentation for Protobuf comments (same as Go style).
//
// Single-line: "// Name doc text"
//
// Multi-line: "// Name line1\n// line2\n// line3"
//
// Text is wrapped to 88 characters including the comment prefix and the indentation the
// comment is emitted at (buf format re-indents continuation lines to match).
func FormatProto(name, doc string, indent ...int) string {
	if doc == "" {
		return ""
	}

	width := maxLineWidth
	if len(indent) > 0 {
		width -= indent[0]
	}
	firstPrefix := "// " + name + " "
	subsequentPrefix := "// "
	lines := wrapText(doc, width-len(firstPrefix), width-len(subsequentPrefix))
	if len(lines) == 0 {
		return ""
	}

	var result []string
	for i, line := range lines {
		if i == 0 {
			result = append(result, firstPrefix+line)
		} else if line == "" {
			result = append(result, "//")
		} else {
			result = append(result, subsequentPrefix+line)
		}
	}
	return strings.Join(result, "\n")
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

type FieldDoc struct {
	Name string
	Doc  string
}

// FormatPyDocstringGoogle formats a Google-style class docstring with an Attributes
// section, indented for a class body. Text is wrapped to 88 characters including
// indentation.
func FormatPyDocstringGoogle(classDoc string, fields []FieldDoc) string {
	var fieldsWithDocs []FieldDoc
	for _, f := range fields {
		if f.Doc != "" {
			fieldsWithDocs = append(fieldsWithDocs, f)
		}
	}

	hasClassDoc := classDoc != ""
	hasFieldDocs := len(fieldsWithDocs) > 0

	if !hasClassDoc && !hasFieldDocs {
		return ""
	}

	const bodyIndent = "    "
	var lines []string

	if hasClassDoc {
		firstPrefix := bodyIndent + `"""`
		classLines := wrapText(
			capitalize(classDoc),
			maxLineWidth-len(firstPrefix),
			maxLineWidth-len(bodyIndent),
		)
		for i, line := range classLines {
			if i == 0 {
				lines = append(lines, firstPrefix+line)
			} else if line == "" {
				lines = append(lines, "")
			} else {
				lines = append(lines, bodyIndent+line)
			}
		}
	} else {
		lines = append(lines, bodyIndent+`"""`)
	}

	if hasFieldDocs {
		if hasClassDoc {
			lines = append(lines, "")
		}
		const fieldIndent = "        "
		const contIndent = "            "
		lines = append(lines, bodyIndent+"Attributes:")
		for _, f := range fieldsWithDocs {
			firstPrefix := fieldIndent + f.Name + ": "
			fieldLines := wrapText(
				capitalize(f.Doc),
				maxLineWidth-len(firstPrefix),
				maxLineWidth-len(contIndent),
			)
			for i, line := range fieldLines {
				if i == 0 {
					lines = append(lines, firstPrefix+line)
				} else if line == "" {
					lines = append(lines, "")
				} else {
					lines = append(lines, contIndent+line)
				}
			}
		}
	}

	lines = append(lines, bodyIndent+`"""`)

	return strings.Join(lines, "\n")
}
