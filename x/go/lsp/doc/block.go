// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package doc

import (
	"fmt"
	"strings"
)

// title renders a heading with an optional kind annotation.
type title struct {
	name string
	kind string
}

// Title creates a title block with just a name.
func Title(name string) Block {
	return title{name: name}
}

// TitleWithKind creates a title block with a name and kind annotation.
func TitleWithKind(name, kind string) Block {
	return title{name: name, kind: kind}
}

// Render returns the markdown representation of the title.
func (t title) Render() string {
	if t.kind != "" {
		return fmt.Sprintf("#### %s\n##### %s", t.name, t.kind)
	}
	return fmt.Sprintf("#### %s", t.name)
}

// Paragraph is a simple text block.
type Paragraph string

// Render returns the paragraph text.
func (p Paragraph) Render() string {
	return string(p)
}

// code renders a fenced code block.
type code struct {
	language string
	content  string
}

// Code creates a code block with a specified language.
func Code(language, content string) Block {
	return code{language: language, content: content}
}

// Render returns the markdown fenced code block.
func (c code) Render() string {
	return fmt.Sprintf("```%s\n%s\n```", c.language, c.content)
}

// detail renders a key-value line, optionally with code formatting for the value.
type detail struct {
	key   string
	value string
	code  bool
}

// Detail creates a detail with the given key, value, and code formatting flag.
func Detail(key, value string, asCode bool) Block {
	return detail{key: key, value: value, code: asCode}
}

// Render returns the markdown representation of the detail.
func (d detail) Render() string {
	if d.code {
		return fmt.Sprintf("%s: `%s`", d.key, d.value)
	}
	return fmt.Sprintf("%s: %s", d.key, d.value)
}

// errorBlock renders an error display with optional code.
type errorBlock struct {
	code    string
	message string
}

// Error creates an error block with just a message.
func Error(message string) Block {
	return errorBlock{message: message}
}

// ErrorWithCode creates an error block with a code and message.
func ErrorWithCode(code, message string) Block {
	return errorBlock{code: code, message: message}
}

// Render returns the markdown representation of the error.
func (e errorBlock) Render() string {
	if e.code != "" {
		return fmt.Sprintf("**Error %s**: %s", e.code, e.message)
	}
	return fmt.Sprintf("**Error**: %s", e.message)
}

// Hint renders a suggestion hint.
type Hint string

// Render returns the markdown representation of the hint.
func (h Hint) Render() string {
	return fmt.Sprintf("_Hint_: %s", string(h))
}

// fix renders a fix suggestion with optional code.
type fix struct {
	description string
	code        string
	language    string
}

// Fix creates a fix block with a description, code example, and language for syntax highlighting.
func Fix(description, codeExample, language string) Block {
	return fix{description: description, code: codeExample, language: language}
}

// Render returns the markdown representation of the fix.
func (f fix) Render() string {
	if f.code != "" {
		return fmt.Sprintf("**Fix**: %s\n\n```%s\n%s\n```", f.description, f.language, f.code)
	}
	return fmt.Sprintf("**Fix**: %s", f.description)
}

// list renders a bullet or numbered list.
type list struct {
	items   []string
	ordered bool
}

// List creates an unordered list.
func List(items ...string) Block {
	return list{items: items, ordered: false}
}

// OrderedList creates an ordered list.
func OrderedList(items ...string) Block {
	return list{items: items, ordered: true}
}

// Render returns the markdown representation of the list.
func (l list) Render() string {
	if len(l.items) == 0 {
		return ""
	}
	parts := make([]string, len(l.items))
	for i, item := range l.items {
		if l.ordered {
			parts[i] = fmt.Sprintf("%d. %s", i+1, item)
		} else {
			parts[i] = fmt.Sprintf("- %s", item)
		}
	}
	return strings.Join(parts, "\n")
}

// Bold renders bold text.
type Bold string

// Render returns the markdown bold text.
func (b Bold) Render() string {
	return fmt.Sprintf("**%s**", string(b))
}

// Italic renders italic text.
type Italic string

// Render returns the markdown italic text.
func (i Italic) Render() string {
	return fmt.Sprintf("_%s_", string(i))
}

// InlineCode renders inline code.
type InlineCode string

// Render returns the markdown inline code.
func (c InlineCode) Render() string {
	return fmt.Sprintf("`%s`", string(c))
}

// divider renders a horizontal rule.
type divider struct{}

// Divider creates a divider block.
func Divider() Block { return divider{} }

// Render returns the markdown horizontal rule.
func (d divider) Render() string { return "---" }
