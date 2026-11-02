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
	"encoding/json/v2"
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
	// indexes holds the next steps by array index. A pointer token such as "01" and
	// "1" both address index 1, so one index can hold several steps.
	indexes map[int][]*pathNode
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
			if n.indexes == nil {
				n.indexes = make(map[int][]*pathNode)
			}
			n.indexes[i] = append(n.indexes[i], c)
		}
	}
	return c
}

// extractor resolves a fixed set of JSON pointers in one pass over a document. It
// decodes only the values the pointers address and skips the rest. It is not safe
// for concurrent use.
type extractor struct {
	root pathNode
	// errs[i] is the error of the malformed pointer of field i.
	errs   []error
	reader bytes.Reader
	dec    jsontext.Decoder
}

func newExtractor(pointers []string) *extractor {
	e := &extractor{errs: make([]error, len(pointers))}
	for i, p := range pointers {
		tokens, err := pointerTokens(p)
		if err != nil {
			e.errs[i] = err
			continue
		}
		n := &e.root
		for _, t := range tokens {
			n = n.child(t)
		}
		n.fields = append(n.fields, i)
	}
	return e
}

// extract resolves every pointer in doc into values and found, which must have one
// entry per pointer. A value is decoded as json.Unmarshal into an any would, with
// integers as int64 or uint64. It returns an error when doc is not one valid JSON
// document.
func (e *extractor) extract(doc []byte, values []any, found []bool) error {
	clear(found)
	e.reader.Reset(doc)
	// Names the pointers do not address may repeat; a repeated addressed value is
	// caught in walk.
	e.dec.Reset(&e.reader, jsontext.AllowDuplicateNames(true))
	if err := e.walk(&e.dec, &e.root, values, found); err != nil {
		return err
	}
	if _, err := e.dec.ReadToken(); err != io.EOF {
		if err == nil {
			err = errors.New("unexpected data after the top-level value")
		}
		return err
	}
	return nil
}

func (e *extractor) walk(
	dec *jsontext.Decoder,
	n *pathNode,
	values []any,
	found []bool,
) error {
	if len(n.fields) > 0 {
		raw, err := dec.ReadValue()
		if err != nil {
			return err
		}
		v, err := decodeValue(raw)
		if err != nil {
			return err
		}
		for _, f := range n.fields {
			if found[f] {
				return errors.Newf("duplicate value at %s", dec.StackPointer())
			}
			values[f], found[f] = v, true
		}
		if n.names == nil {
			return nil
		}
		sub := jsontext.NewDecoder(bytes.NewReader(raw))
		return e.walk(sub, &pathNode{names: n.names, indexes: n.indexes}, values, found)
	}
	switch dec.PeekKind() {
	case '{':
		if n.names == nil {
			return dec.SkipValue()
		}
		if _, err := dec.ReadToken(); err != nil {
			return err
		}
		for dec.PeekKind() != '}' {
			name, err := dec.ReadToken()
			if err != nil {
				return err
			}
			if c, ok := n.names[name.String()]; ok {
				err = e.walk(dec, c, values, found)
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
		if n.indexes == nil {
			return dec.SkipValue()
		}
		if _, err := dec.ReadToken(); err != nil {
			return err
		}
		for i := 0; dec.PeekKind() != ']'; i++ {
			cs := n.indexes[i]
			var err error
			switch len(cs) {
			case 0:
				err = dec.SkipValue()
			case 1:
				err = e.walk(dec, cs[0], values, found)
			default:
				err = e.walk(dec, merge(cs), values, found)
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
		for i, cs := range n.indexes {
			if m.indexes == nil {
				m.indexes = make(map[int][]*pathNode)
			}
			m.indexes[i] = append(m.indexes[i], cs...)
		}
	}
	return m
}

// decodeValue decodes one JSON value as json.Unmarshal into an any does, keeping
// integers as int64 or uint64.
func decodeValue(raw jsontext.Value) (any, error) {
	switch raw.Kind() {
	case '0':
		return parseNumber(string(raw))
	case 't':
		return true, nil
	case 'f':
		return false, nil
	case 'n':
		return nil, nil
	case '"':
		s, err := jsontext.AppendUnquote(nil, raw)
		return string(s), err
	default:
		var v any
		err := json.Unmarshal(raw, &v, preciseNumbers)
		return v, err
	}
}
