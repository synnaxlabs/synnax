// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package json

import (
	"encoding/json/jsontext"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// TemplateField is one value of the documents a Template renders.
type TemplateField struct {
	// Label names the field in errors, such as field "source".
	Label string
	// Pointer places the value. Its tokens name object members, never array indexes.
	Pointer Pointer
	// Literal is the JSON text of a value every document holds. A nil Literal makes the
	// field a slot that each render fills.
	Literal []byte
}

// Slots fills the slots of a Template.
type Slots interface {
	// AppendSlot appends the JSON value of the slot for the field at position field of
	// the Template's fields to dst.
	AppendSlot(dst []byte, field int) ([]byte, error)
}

// templateNode is one value of the JSON document a Template renders.
type templateNode struct {
	// names orders members, which is nil unless the node is an object.
	names   []string
	members map[string]*templateNode
	// field is the position of the field that sets the node, -1 for an object that
	// fields create.
	field int
}

// Template renders JSON documents whose members keep the order of the fields that set
// them. Everything but the slots is rendered once, so a render only costs its slots.
type Template struct {
	segments []segment
	// suffix is the JSON text after the last slot.
	suffix []byte
}

type segment struct {
	// prefix is the JSON text before the slot.
	prefix []byte
	field  int
}

// NewTemplate returns a Template for fields, which it creates the objects of. It
// returns an error wrapping validate.ErrValidation when two pointers are equal or one
// holds the other, since a document can hold only one of them. The error names both
// fields in the order of fields.
func NewTemplate(fields []TemplateField) (Template, error) {
	root := &templateNode{field: -1}
	for i := range fields {
		if err := root.set(fields, i); err != nil {
			return Template{}, err
		}
	}
	var t Template
	text, err := t.render(fields, root, nil)
	if err != nil {
		return Template{}, err
	}
	t.suffix = text
	return t, nil
}

func (n *templateNode) set(fields []TemplateField, i int) error {
	p := fields[i].Pointer
	for depth, token := range p {
		if n.field != -1 {
			return overlapError(fields, n.field, i)
		}
		c, ok := n.members[token]
		if !ok {
			c = &templateNode{field: -1}
			if depth == len(p)-1 {
				c.field = i
			}
			if n.members == nil {
				n.members = make(map[string]*templateNode)
			}
			n.names = append(n.names, token)
			n.members[token] = c
		} else if depth == len(p)-1 {
			return overlapError(fields, overlapped(c), i)
		}
		n = c
	}
	if len(p) == 0 {
		if n.field != -1 || n.members != nil {
			return overlapError(fields, overlapped(n), i)
		}
		n.field = i
	}
	return nil
}

// overlapped returns the position of a field that sets n or a value inside it.
func overlapped(n *templateNode) int {
	for n.field == -1 {
		n = n.members[n.names[0]]
	}
	return n.field
}

func overlapError(fields []TemplateField, a, b int) error {
	return errors.Wrapf(
		validate.ErrValidation,
		"%s %q and %s %q overlap",
		fields[a].Label, fields[a].Pointer, fields[b].Label, fields[b].Pointer,
	)
}

// render appends n to text, which holds the JSON text since the last slot. Each slot
// ends a segment. It returns the JSON text since the last slot.
func (t *Template) render(
	fields []TemplateField,
	n *templateNode,
	text []byte,
) ([]byte, error) {
	if n.field != -1 {
		if lit := fields[n.field].Literal; lit != nil {
			return append(text, lit...), nil
		}
		t.segments = append(t.segments, segment{prefix: text, field: n.field})
		return nil, nil
	}
	text = append(text, '{')
	var err error
	for i, name := range n.names {
		if i > 0 {
			text = append(text, ',')
		}
		if text, err = jsontext.AppendQuote(text, name); err != nil {
			return nil, err
		}
		text = append(text, ':')
		if text, err = t.render(fields, n.members[name], text); err != nil {
			return nil, err
		}
	}
	return append(text, '}'), nil
}

// Append appends a document to dst, with each slot filled by slots.
func (t Template) Append(dst []byte, slots Slots) ([]byte, error) {
	var err error
	for _, seg := range t.segments {
		dst = append(dst, seg.prefix...)
		if dst, err = slots.AppendSlot(dst, seg.field); err != nil {
			return nil, err
		}
	}
	return append(dst, t.suffix...), nil
}
