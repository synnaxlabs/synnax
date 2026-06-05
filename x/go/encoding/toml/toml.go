// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package toml provides a TOML implementation of encoding.Codec and http.Codec for use
// as a portable text format (import/export bodies). It wraps github.com/pelletier/
// go-toml/v2, so types customize their wire shape with `toml` struct tags.
//
// go-toml/v2 exposes no general marshal-side interface (its only marshal hook is
// encoding.TextMarshaler, which is string-wrapped and cannot be the root) and its
// unmarshal hook surfaces only raw bytes. To let a type that needs more control over its
// TOML representation (for example, one that flattens promoted fields into the top-level
// table) participate, this codec defines its own Marshaler and Unmarshaler interfaces and
// checks for them before falling back to go-toml's struct reflection.
//
// TOML has no null value and requires the top-level document to be a table, so encoding a
// value that resolves to TOML null or to a non-table returns an error.
package toml

import (
	"context"
	"io"

	"github.com/pelletier/go-toml/v2"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/http"
)

// Marshaler is implemented by types that supply the value the codec encodes in their
// place. It exists because go-toml/v2 has no struct-level marshal interface: its only
// native marshal hook is encoding.TextMarshaler, which is unusable for a type whose wire
// form is a table (such as the import/export envelope, which flattens its promoted fields
// to the top level). TextMarshaler is a scalar→string hook — go-toml errors when the
// document root implements it ("cannot be a root element") and otherwise runs the result
// through encodeString, emitting a single quoted string rather than a table. TOMLValue
// instead returns the table as a map or struct, which go-toml renders as TOML proper.
type Marshaler interface {
	TOMLValue() (any, error)
}

// Unmarshaler is implemented by types that populate themselves from the codec's decoded
// TOML table. It mirrors Marshaler on the decode side: go-toml/v2's native hooks can't
// rebuild a value from a table — unstable.Unmarshaler hands back raw bytes, and
// encoding.TextUnmarshaler only fires on string nodes — so the codec decodes into a
// map[string]any and passes it here.
type Unmarshaler interface {
	FromTOMLValue(map[string]any) error
}

// Codec is a TOML implementation of encoding.Codec and http.Codec.
var Codec http.Codec = &codec{}

type codec struct{}

func (c *codec) ContentType() string { return "application/toml" }

// Encode implements the encoding.Encoder interface. If value implements Marshaler, the
// value it returns is encoded in its place.
func (c *codec) Encode(_ context.Context, value any) ([]byte, error) {
	if m, ok := value.(Marshaler); ok {
		v, err := m.TOMLValue()
		if err != nil {
			return nil, encoding.SugarEncodingErr(value, err)
		}
		value = v
	}
	b, err := toml.Marshal(value)
	return b, encoding.SugarEncodingErr(value, err)
}

// Decode implements the encoding.Decoder interface. If value implements Unmarshaler, the
// body is decoded into a table and handed to it; otherwise go-toml's struct reflection
// is used.
func (c *codec) Decode(_ context.Context, data []byte, value any) error {
	if u, ok := value.(Unmarshaler); ok {
		var m map[string]any
		if err := toml.Unmarshal(data, &m); err != nil {
			return encoding.SugarDecodingErr(data, value, err)
		}
		if err := u.FromTOMLValue(m); err != nil {
			return encoding.SugarDecodingErr(data, value, err)
		}
		return nil
	}
	if err := toml.Unmarshal(data, value); err != nil {
		return encoding.SugarDecodingErr(data, value, err)
	}
	return nil
}

// DecodeStream implements the encoding.Decoder interface.
func (c *codec) DecodeStream(_ context.Context, r io.Reader, value any) error {
	if u, ok := value.(Unmarshaler); ok {
		var m map[string]any
		if err := toml.NewDecoder(r).Decode(&m); err != nil {
			return encoding.SugarDecodingErr(nil, value, err)
		}
		if err := u.FromTOMLValue(m); err != nil {
			return encoding.SugarDecodingErr(nil, value, err)
		}
		return nil
	}
	if err := toml.NewDecoder(r).Decode(value); err != nil {
		return encoding.SugarDecodingErr(nil, value, err)
	}
	return nil
}

// EncodeStream implements the encoding.Encoder interface.
func (c *codec) EncodeStream(ctx context.Context, w io.Writer, value any) error {
	b, err := c.Encode(ctx, value)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return encoding.SugarEncodingErr(value, err)
}
