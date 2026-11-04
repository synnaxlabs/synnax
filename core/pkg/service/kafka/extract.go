// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kafka

import (
	"bytes"
	"encoding/json/jsontext"
	"io"
	"strconv"

	"github.com/synnaxlabs/x/errors"
)

// pathNode is one step of the pointers an extractor resolves.
type pathNode struct {
	// fields are the positions of the fields whose pointer ends here.
	fields []int
	// names holds the next step by object member name.
	names map[string]*pathNode
	// tokens holds the steps array index tokens take. A pointer token such as "01" and
	// "1" both address index 1, so one index can hold several steps.
	tokens map[int][]*pathNode
	// elements holds the next step by array index, merged from tokens.
	elements map[int]*pathNode
	// children is this node without its fields, for walking a value already read.
	children *pathNode
}

func (n *pathNode) child(token string) *pathNode {
	if n.names == nil {
		n.names = make(map[string]*pathNode)
	}
	c, ok := n.names[token]
	if !ok {
		c = &pathNode{}
		n.names[token] = c
		if i, err := strconv.Atoi(token); err == nil && i >= 0 {
			if n.tokens == nil {
				n.tokens = make(map[int][]*pathNode)
			}
			n.tokens[i] = append(n.tokens[i], c)
		}
	}
	return c
}

// resolve computes the walk's lookups for n and every node below it.
func (n *pathNode) resolve() {
	for _, c := range n.names {
		c.resolve()
	}
	if n.tokens != nil {
		n.elements = make(map[int]*pathNode, len(n.tokens))
	}
	for i, cs := range n.tokens {
		c := cs[0]
		if len(cs) > 1 {
			c = merge(cs)
			c.resolve()
		}
		n.elements[i] = c
	}
	if len(n.fields) > 0 && n.names != nil {
		n.children = &pathNode{names: n.names, elements: n.elements}
	}
}

// merge joins the steps several tokens take into one array element.
func merge(nodes []*pathNode) *pathNode {
	m := &pathNode{}
	for _, n := range nodes {
		m.fields = append(m.fields, n.fields...)
		for name, c := range n.names {
			if m.names == nil {
				m.names = make(map[string]*pathNode)
			}
			if prev, ok := m.names[name]; ok {
				c = merge([]*pathNode{prev, c})
			}
			m.names[name] = c
		}
		for i, cs := range n.tokens {
			if m.tokens == nil {
				m.tokens = make(map[int][]*pathNode)
			}
			m.tokens[i] = append(m.tokens[i], cs...)
		}
	}
	return m
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

// extractor resolves a fixed set of JSON pointers in one pass over a document. It
// neither copies nor decodes the values the pointers address. It is not safe for
// concurrent use.
type extractor struct {
	root *pathNode
	// errs[i] is the error of the malformed pointer of field i.
	errs []error
	// decoders[i] decodes at nesting level i. A pointer that addresses a value inside
	// another pointer's value adds a level.
	decoders []*documentDecoder
	// opts allow repeats of the names no pointer addresses. A repeated addressed
	// value is caught in walk.
	opts []jsontext.Options
	// name is scratch space for an object member name holding an escape.
	name []byte
}

func newExtractor(pointers []string) *extractor {
	e := &extractor{
		root:     &pathNode{},
		errs:     make([]error, len(pointers)),
		decoders: []*documentDecoder{{}},
		opts:     []jsontext.Options{jsontext.AllowDuplicateNames(true)},
	}
	for i, p := range pointers {
		tokens, err := pointerTokens(p)
		if err != nil {
			e.errs[i] = err
			continue
		}
		n := e.root
		for _, t := range tokens {
			n = n.child(t)
		}
		n.fields = append(n.fields, i)
	}
	e.root.resolve()
	return e
}

// extract resolves every pointer in doc into values, which must have one zeroed entry
// per pointer. A pointer doc does not hold leaves its entry nil. Each value is a slice
// of doc. It returns an error when doc is not one valid JSON document.
func (e *extractor) extract(doc []byte, values []jsontext.Value) error {
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

func (e *extractor) walk(
	d *documentDecoder,
	n *pathNode,
	values []jsontext.Value,
	level int,
) error {
	if len(n.fields) > 0 {
		raw, err := d.readValue()
		if err != nil {
			return err
		}
		for _, f := range n.fields {
			if values[f] != nil {
				return errors.Newf("duplicate value at %s", d.dec.StackPointer())
			}
			values[f] = raw
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
func (e *extractor) member(n *pathNode, quoted jsontext.Value) (*pathNode, error) {
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
