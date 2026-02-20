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
// Text is wrapped to 88 characters including the comment prefix.
func FormatGo(name, doc string) string {
	if doc == "" {
		return ""
	}

	// Calculate available width: "// Name " for first line, "// " for subsequent
	firstPrefix := "// " + name + " "
	subsequentPrefix := "// "
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
			result = append(result, "//")
		} else {
			result = append(result, subsequentPrefix+line)
		}
	}
	return strings.Join(result, "\n")
}

// FormatTS formats documentation for TypeScript JSDoc comments.
// Single-line: "/** Name doc text */"
// Multi-line: "/**\n * Name line1\n * line2\n */"
func FormatTS(name, doc string) string {
	if doc == "" {
		return ""
	}
	lines := strings.Split(doc, "\n")
	if len(lines) == 1 {
		return "/** " + name + " " + doc + " */"
	}
	var result []string
	result = append(result, "/**")
	for i, line := range lines {
		if i == 0 {
			result = append(result, " * "+name+" "+line)
		} else if line == "" {
			result = append(result, " *")
		} else {
			result = append(result, " * "+line)
		}
	}
	result = append(result, " */")
	return strings.Join(result, "\n")
}

// FormatPyDocstring formats documentation for Python class/function docstrings.
// Single-line: `"""Name doc text"""`
// Multi-line: `"""Name line1\nline2\nline3"""`
func FormatPyDocstring(name, doc string) string {
	if doc == "" {
		return ""
	}
	lines := strings.Split(doc, "\n")
	if len(lines) == 1 {
		return `"""` + name + " " + doc + `"""`
	}
	var result []string
	for i, line := range lines {
		if i == 0 {
			result = append(result, `"""`+name+" "+line)
		} else {
			result = append(result, line)
		}
	}
	result[len(result)-1] = result[len(result)-1] + `"""`
	return strings.Join(result, "\n")
}

// FormatPyComment formats documentation for Python line comments.
// Single-line: "# Name doc text"
// Multi-line: "# Name line1\n# line2\n# line3"
func FormatPyComment(name, doc string) string {
	if doc == "" {
		return ""
	}
	lines := strings.Split(doc, "\n")
	var result []string
	for i, line := range lines {
		if i == 0 {
			result = append(result, "# "+name+" "+line)
		} else {
			result = append(result, "# "+line)
		}
	}
	return strings.Join(result, "\n")
}

// FormatCpp formats documentation for C++ Doxygen-style comments.
// Single-line: "/// @brief Name doc text"
// Multi-line: "/// @brief Name line1\n/// line2\n/// line3"
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
// Single-line: "// Name doc text"
// Multi-line: "// Name line1\n// line2\n// line3"
func FormatProto(name, doc string) string {
	return FormatGo(name, doc)
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

	var lines []string

	if hasClassDoc {
		classDocCapitalized := capitalize(classDoc)
		classLines := strings.Split(classDocCapitalized, "\n")
		lines = append(lines, `    """`+classLines[0])
		for i := 1; i < len(classLines); i++ {
			lines = append(lines, "    "+classLines[i])
		}
	} else {
		lines = append(lines, `    """`)
	}

	if hasFieldDocs {
		if hasClassDoc {
			lines = append(lines, "")
		}
		lines = append(lines, "    Attributes:")
		for _, f := range fieldsWithDocs {
			fieldDocCapitalized := capitalize(f.Doc)
			fieldLines := strings.Split(fieldDocCapitalized, "\n")
			lines = append(lines, "        "+f.Name+": "+fieldLines[0])
			for i := 1; i < len(fieldLines); i++ {
				lines = append(lines, "            "+fieldLines[i])
			}
		}
	}

	lines = append(lines, `    """`)

	return strings.Join(lines, "\n")
}
