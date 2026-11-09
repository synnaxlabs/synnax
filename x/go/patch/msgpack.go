// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package patch

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"strconv"

	"github.com/synnaxlabs/x/errors"
	"github.com/vmihailenco/msgpack/v5"
)

// EncodeMsgpack writes the JSON form of v to enc as msgpack. Integers keep every digit.
//
// msgpack never calls the decoder of a struct field that holds nil, so a Change or a
// Struct cannot see its own clear. A struct that holds one implements
// msgpack.CustomEncoder and msgpack.CustomDecoder with EncodeMsgpack and DecodeMsgpack,
// which carry its JSON form, where a clear is a null that the field decodes.
func EncodeMsgpack[T any](enc *msgpack.Encoder, v T) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	tree, err := readJSON(jsontext.NewDecoder(bytes.NewReader(b)))
	if err != nil {
		return err
	}
	return enc.Encode(tree)
}

// DecodeMsgpack reads a msgpack value from dec and decodes it into dst as JSON. It
// reads what EncodeMsgpack writes.
func DecodeMsgpack[T any](dec *msgpack.Decoder, dst *T) error {
	tree, err := dec.DecodeInterface()
	if err != nil {
		return err
	}
	b, err := json.Marshal(tree)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}

// readJSON reads one JSON value into maps, slices, and scalars. An integer that fits an
// int64 or a uint64 reads as one, so msgpack encodes it exactly.
func readJSON(dec *jsontext.Decoder) (any, error) {
	tok, err := dec.ReadToken()
	if err != nil {
		return nil, err
	}
	switch tok.Kind() {
	case 'n':
		return nil, nil
	case 't', 'f':
		return tok.Bool(), nil
	case '"':
		return tok.String(), nil
	case '0':
		return readNumber(tok.String())
	case '{':
		obj := make(map[string]any)
		for dec.PeekKind() != '}' {
			name, err := dec.ReadToken()
			if err != nil {
				return nil, err
			}
			if obj[name.String()], err = readJSON(dec); err != nil {
				return nil, err
			}
		}
		_, err = dec.ReadToken()
		return obj, err
	case '[':
		arr := make([]any, 0)
		for dec.PeekKind() != ']' {
			v, err := readJSON(dec)
			if err != nil {
				return nil, err
			}
			arr = append(arr, v)
		}
		_, err = dec.ReadToken()
		return arr, err
	default:
		return nil, errors.Newf("unexpected JSON token %s", tok.Kind())
	}
}

func readNumber(text string) (any, error) {
	if i, err := strconv.ParseInt(text, 10, 64); err == nil {
		return i, nil
	}
	if u, err := strconv.ParseUint(text, 10, 64); err == nil {
		return u, nil
	}
	return strconv.ParseFloat(text, 64)
}
