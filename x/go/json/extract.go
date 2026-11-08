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
	"bytes"
	"encoding/json/jsontext"
	"io"

	"github.com/synnaxlabs/x/errors"
)

// pathNode is one step of the pointers an Extractor resolves.
type pathNode struct {
	// pointers are the positions of the pointers that end here.
	pointers []int
	// names holds the next step by object member name.
	names map[string]*pathNode
	// elements holds the next step by array index.
	elements map[int]*pathNode
	// children is this node without its pointers, for walking a value already read.
	children *pathNode
}

func (n *pathNode) child(token string) *pathNode {
	if c, ok := n.names[token]; ok {
		return c
	}
	if n.names == nil {
		n.names = make(map[string]*pathNode)
	}
	c := &pathNode{}
	n.names[token] = c
	if i, ok := arrayIndex(token); ok {
		if n.elements == nil {
			n.elements = make(map[int]*pathNode)
		}
		n.elements[i] = c
	}
	return c
}

// resolve sets children on n and every node below it.
func (n *pathNode) resolve() {
	for _, c := range n.names {
		c.resolve()
	}
	if len(n.pointers) > 0 && n.names != nil {
		n.children = &pathNode{names: n.names, elements: n.elements}
	}
}

// documentDecoder decodes one JSON document held in memory.
type documentDecoder struct {
	doc []byte
	buf bytes.Buffer
	dec jsontext.Decoder
}

func (d *documentDecoder) reset(doc []byte, opts []jsontext.Options) {
	d.doc = doc
	d.buf = *bytes.NewBuffer(doc)
	d.dec.Reset(&d.buf, opts...)
}

// readValue reads the next value, returned as a slice of doc that stays valid across
// later reads, unlike the slice the decoder returns.
func (d *documentDecoder) readValue() (jsontext.Value, error) {
	raw, err := d.dec.ReadValue()
	if err != nil {
		return nil, err
	}
	end := d.dec.InputOffset()
	return jsontext.Value(d.doc[end-int64(len(raw)) : end]), nil
}

// Extractor resolves a fixed set of pointers in one pass over a JSON document. It
// neither copies nor decodes the values the pointers address, and it allocates nothing
// once it has seen a document as deep as the pointers. It is not safe for concurrent
// use.
type Extractor struct {
	root *pathNode
	// decoders[i] decodes at nesting level i. A pointer that addresses a value inside
	// another pointer's value adds a level.
	decoders []*documentDecoder
	// opts allow repeats of the names no pointer addresses. A repeated addressed value
	// is caught in walk.
	opts []jsontext.Options
	// name is scratch space for an object member name holding an escape.
	name []byte
}

// NewExtractor returns an Extractor for pointers.
func NewExtractor(pointers []Pointer) *Extractor {
	e := &Extractor{
		root:     &pathNode{},
		decoders: []*documentDecoder{{}},
		opts:     []jsontext.Options{jsontext.AllowDuplicateNames(true)},
	}
	for i, p := range pointers {
		n := e.root
		for _, token := range p {
			n = n.child(token)
		}
		n.pointers = append(n.pointers, i)
	}
	e.root.resolve()
	return e
}

// Extract resolves every pointer in doc into values, which holds one entry for each
// pointer. Each value is a slice of doc, or nil when doc holds no value at the pointer.
// Extract returns an error when doc is not one valid JSON document, or when a member
// name repeats on the path to an addressed value.
func (e *Extractor) Extract(doc []byte, values []jsontext.Value) error {
	clear(values)
	d := e.decoders[0]
	d.reset(doc, e.opts)
	if err := e.walk(d, e.root, values, 0); err != nil {
		return err
	}
	if _, err := d.dec.ReadToken(); err != io.EOF {
		if err == nil {
			err = errors.New("unexpected data after the top-level value")
		}
		return err
	}
	return nil
}

func (e *Extractor) walk(
	d *documentDecoder,
	n *pathNode,
	values []jsontext.Value,
	level int,
) error {
	if len(n.pointers) > 0 {
		raw, err := d.readValue()
		if err != nil {
			return err
		}
		for _, p := range n.pointers {
			if values[p] != nil {
				return errors.Newf("duplicate value at %s", d.dec.StackPointer())
			}
			values[p] = raw
		}
		if n.children == nil {
			return nil
		}
		level++
		if level == len(e.decoders) {
			e.decoders = append(e.decoders, &documentDecoder{})
		}
		sub := e.decoders[level]
		sub.reset(raw, e.opts)
		return e.walk(sub, n.children, values, level)
	}
	dec := &d.dec
	switch dec.PeekKind() {
	case '{':
		if n.names == nil {
			return dec.SkipValue()
		}
		if _, err := dec.ReadToken(); err != nil {
			return err
		}
		for dec.PeekKind() != '}' {
			name, err := dec.ReadValue()
			if err != nil {
				return err
			}
			c, err := e.member(n, name)
			if err != nil {
				return err
			}
			if c != nil {
				err = e.walk(d, c, values, level)
			} else {
				err = dec.SkipValue()
			}
			if err != nil {
				return err
			}
		}
		_, err := dec.ReadToken()
		return err
	case '[':
		if n.elements == nil {
			return dec.SkipValue()
		}
		if _, err := dec.ReadToken(); err != nil {
			return err
		}
		for i := 0; dec.PeekKind() != ']'; i++ {
			var err error
			if c, ok := n.elements[i]; ok {
				err = e.walk(d, c, values, level)
			} else {
				err = dec.SkipValue()
			}
			if err != nil {
				return err
			}
		}
		_, err := dec.ReadToken()
		return err
	default:
		return dec.SkipValue()
	}
}

// member returns the step n takes on the object member with the quoted name, or nil
// when no pointer addresses it.
func (e *Extractor) member(n *pathNode, quoted jsontext.Value) (*pathNode, error) {
	name := quoted[1 : len(quoted)-1]
	if bytes.IndexByte(name, '\\') >= 0 {
		var err error
		if e.name, err = jsontext.AppendUnquote(e.name[:0], quoted); err != nil {
			return nil, err
		}
		name = e.name
	}
	return n.names[string(name)], nil
}
