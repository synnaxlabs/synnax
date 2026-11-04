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
	"encoding/json/v2"
	"io"

	"github.com/synnaxlabs/x/errors"
)

// pathNode is one step of the pointers an Extractor resolves.
type pathNode struct {
	// pointers are the positions of the pointers that end here.
	pointers []int
	// names holds the next step by object member name.
	names map[string]*pathNode
	// indexes holds the next step by array index.
	indexes map[int]*pathNode
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
		if n.indexes == nil {
			n.indexes = make(map[int]*pathNode)
		}
		n.indexes[i] = c
	}
	return c
}

// Extractor resolves a fixed set of pointers in one pass over a JSON document. It
// decodes only the values that the pointers address and skips the rest, so it is
// faster than Decode followed by Pointer.Get. It is not safe for concurrent use.
type Extractor struct {
	root   pathNode
	reader bytes.Reader
	dec    jsontext.Decoder
}

// NewExtractor returns an Extractor for pointers.
func NewExtractor(pointers []Pointer) *Extractor {
	e := &Extractor{}
	for i, p := range pointers {
		n := &e.root
		for _, token := range p {
			n = n.child(token)
		}
		n.pointers = append(n.pointers, i)
	}
	return e
}

// Extract resolves every pointer in data, a JSON document, into values and found,
// which hold one entry for each pointer. A value is a node of the kind that Decode
// returns, and found[i] is false when pointer i leads to no value. Extract returns an
// error when data is not one JSON document, or when a member name repeats on the path
// to an addressed value.
func (e *Extractor) Extract(data []byte, values []any, found []bool) error {
	clear(found)
	e.reader.Reset(data)
	e.dec.Reset(
		&e.reader,
		jsontext.AllowDuplicateNames(true),
		jsontext.AllowInvalidUTF8(true),
	)
	if err := walk(&e.dec, &e.root, values, found); err != nil {
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

func walk(dec *jsontext.Decoder, n *pathNode, values []any, found []bool) error {
	if len(n.pointers) > 0 {
		raw, err := dec.ReadValue()
		if err != nil {
			return err
		}
		v, err := decodeValue(raw)
		if err != nil {
			return err
		}
		for _, p := range n.pointers {
			if found[p] {
				return errors.Newf("duplicate value at %s", dec.StackPointer())
			}
			values[p], found[p] = v, true
		}
		if n.names == nil {
			return nil
		}
		sub := jsontext.NewDecoder(bytes.NewReader(raw))
		return walk(sub, &pathNode{names: n.names, indexes: n.indexes}, values, found)
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
				err = walk(dec, c, values, found)
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
			var err error
			if c, ok := n.indexes[i]; ok {
				err = walk(dec, c, values, found)
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

// decodeValue decodes one JSON value as Decode does.
func decodeValue(raw jsontext.Value) (any, error) {
	switch raw.Kind() {
	case '0':
		return NumberText(raw), nil
	case 't':
		return true, nil
	case 'f':
		return false, nil
	case 'n':
		return nil, nil
	case '"':
		if s, err := jsontext.AppendUnquote(nil, raw); err == nil {
			return string(s), nil
		}
		// AppendUnquote rejects invalid UTF-8, which Decode replaces.
		var s string
		err := json.Unmarshal(raw, &s, decodeOptions)
		return s, err
	default:
		var v any
		err := json.Unmarshal(raw, &v, decodeOptions)
		return v, err
	}
}
