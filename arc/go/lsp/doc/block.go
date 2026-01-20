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

// Title renders a heading with an optional kind annotation.
type Title struct {
	Name string
	Kind string
}

// NewTitle creates a Title with just a name.
func NewTitle(name string) Title {
	return Title{Name: name}
}

// NewTitleWithKind creates a Title with a name and kind annotation.
func NewTitleWithKind(name, kind string) Title {
	return Title{Name: name, Kind: kind}
}

// Render returns the markdown representation of the title.
func (t Title) Render() string {
	if t.Kind != "" {
		return fmt.Sprintf("#### %s\n##### %s", t.Name, t.Kind)
	}
	return fmt.Sprintf("#### %s", t.Name)
}

// Paragraph is a simple text block.
type Paragraph string

// Render returns the paragraph text.
func (p Paragraph) Render() string {
	return string(p)
}

// Code renders a fenced code block.
type Code struct {
	Language string
	Content  string
}

// NewCode creates a Code block with a specified language.
func NewCode(language, content string) Code {
	return Code{Language: language, Content: content}
}

// NewArcCode creates a Code block with the arc language.
func NewArcCode(content string) Code {
	return Code{Language: "arc", Content: content}
}

// Render returns the markdown fenced code block.
func (c Code) Render() string {
	return fmt.Sprintf("```%s\n%s\n```", c.Language, c.Content)
}

// Detail renders a key-value line, optionally with code formatting for the value.
type Detail struct {
	Key   string
	Value string
	Code  bool
}

// NewDetail creates a Detail with the given key, value, and code formatting flag.
func NewDetail(key, value string, code bool) Detail {
	return Detail{Key: key, Value: value, Code: code}
}

// Render returns the markdown representation of the detail.
func (d Detail) Render() string {
	if d.Code {
		return fmt.Sprintf("%s: `%s`", d.Key, d.Value)
	}
	return fmt.Sprintf("%s: %s", d.Key, d.Value)
}

// Details renders multiple key-value lines.
type Details []Detail

// Render returns all details joined by newlines.
func (ds Details) Render() string {
	parts := make([]string, len(ds))
	for i, d := range ds {
		parts[i] = d.Render()
	}
	return strings.Join(parts, "\n")
}

// Error renders an error display with optional code.
type Error struct {
	Code    string
	Message string
}

// NewError creates an Error with just a message.
func NewError(message string) Error {
	return Error{Message: message}
}

// NewErrorWithCode creates an Error with a code and message.
func NewErrorWithCode(code, message string) Error {
	return Error{Code: code, Message: message}
}

// Render returns the markdown representation of the error.
func (e Error) Render() string {
	if e.Code != "" {
		return fmt.Sprintf("**Error %s**: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("**Error**: %s", e.Message)
}

// Hint renders a suggestion hint.
type Hint string

// NewHint creates a Hint with the given message.
func NewHint(message string) Hint {
	return Hint(message)
}

// Render returns the markdown representation of the hint.
func (h Hint) Render() string {
	return fmt.Sprintf("_Hint_: %s", string(h))
}

// Fix renders a fix suggestion with optional code.
type Fix struct {
	Description string
	Code        string
}

// NewFix creates a Fix with a description and code example.
func NewFix(description, code string) Fix {
	return Fix{Description: description, Code: code}
}

// Render returns the markdown representation of the fix.
func (f Fix) Render() string {
	if f.Code != "" {
		return fmt.Sprintf("**Fix**: %s\n\n```arc\n%s\n```", f.Description, f.Code)
	}
	return fmt.Sprintf("**Fix**: %s", f.Description)
}

// List renders a bullet or numbered list.
type List struct {
	Items   []string
	Ordered bool
}

// NewList creates an unordered list.
func NewList(items ...string) List {
	return List{Items: items, Ordered: false}
}

// NewOrderedList creates an ordered list.
func NewOrderedList(items ...string) List {
	return List{Items: items, Ordered: true}
}

// Render returns the markdown representation of the list.
func (l List) Render() string {
	if len(l.Items) == 0 {
		return ""
	}
	parts := make([]string, len(l.Items))
	for i, item := range l.Items {
		if l.Ordered {
			parts[i] = fmt.Sprintf("%d. %s", i+1, item)
		} else {
			parts[i] = fmt.Sprintf("- %s", item)
		}
	}
	return strings.Join(parts, "\n")
}

// Bold renders bold text.
type Bold string

// NewBold creates a Bold block.
func NewBold(text string) Bold {
	return Bold(text)
}

// Render returns the markdown bold text.
func (b Bold) Render() string {
	return fmt.Sprintf("**%s**", string(b))
}

// Italic renders italic text.
type Italic string

// NewItalic creates an Italic block.
func NewItalic(text string) Italic {
	return Italic(text)
}

// Render returns the markdown italic text.
func (i Italic) Render() string {
	return fmt.Sprintf("_%s_", string(i))
}

// InlineCode renders inline code.
type InlineCode string

// NewInlineCode creates an InlineCode block.
func NewInlineCode(text string) InlineCode {
	return InlineCode(text)
}

// Render returns the markdown inline code.
func (c InlineCode) Render() string {
	return fmt.Sprintf("`%s`", string(c))
}

// Divider renders a horizontal rule.
type Divider struct{}

// NewDivider creates a Divider block.
func NewDivider() Divider {
	return Divider{}
}

// Render returns the markdown horizontal rule.
func (d Divider) Render() string {
	return "---"
}
