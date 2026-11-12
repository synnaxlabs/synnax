// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package vcs

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"slices"
	"strings"

	"github.com/synnaxlabs/x/errors"
)

// encode marshals v as canonical JSON: sorted object names, two-space indents, and a
// trailing newline. Numbers keep the text that json.Marshal gives them.
func encode(v any) ([]byte, error) {
	raw, err := json.Marshal(v, json.Deterministic(true))
	if err != nil {
		return nil, errors.Wrap(err, "encode file")
	}
	var buf bytes.Buffer
	enc := jsontext.NewEncoder(
		&buf,
		jsontext.WithIndent("  "),
		jsontext.SpaceAfterColon(true),
	)
	if err = writeSorted(enc, jsontext.Value(raw)); err != nil {
		return nil, err
	}
	out := bytes.TrimRight(buf.Bytes(), "\n")
	return append(out, '\n'), nil
}

type member struct {
	name  string
	value jsontext.Value
}

func writeSorted(enc *jsontext.Encoder, v jsontext.Value) error {
	switch v.Kind() {
	case '{':
		dec := jsontext.NewDecoder(bytes.NewReader(v))
		if _, err := dec.ReadToken(); err != nil {
			return err
		}
		var members []member
		for dec.PeekKind() != '}' {
			tok, err := dec.ReadToken()
			if err != nil {
				return err
			}
			// A later read voids the token, so take its text first.
			name := tok.String()
			value, err := dec.ReadValue()
			if err != nil {
				return err
			}
			members = append(members, member{name: name, value: value.Clone()})
		}
		slices.SortFunc(members, func(a, b member) int {
			return strings.Compare(a.name, b.name)
		})
		if err := enc.WriteToken(jsontext.BeginObject); err != nil {
			return err
		}
		for _, m := range members {
			if err := enc.WriteToken(jsontext.String(m.name)); err != nil {
				return err
			}
			if err := writeSorted(enc, m.value); err != nil {
				return err
			}
		}
		return enc.WriteToken(jsontext.EndObject)
	case '[':
		dec := jsontext.NewDecoder(bytes.NewReader(v))
		if _, err := dec.ReadToken(); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.BeginArray); err != nil {
			return err
		}
		for dec.PeekKind() != ']' {
			value, err := dec.ReadValue()
			if err != nil {
				return err
			}
			if err = writeSorted(enc, value.Clone()); err != nil {
				return err
			}
		}
		return enc.WriteToken(jsontext.EndArray)
	default:
		return enc.WriteValue(v)
	}
}

// decode unmarshals a file body into v.
func decode(data []byte, v any) error {
	if err := json.Unmarshal(data, v); err != nil {
		return errors.Wrap(err, "decode file")
	}
	return nil
}
