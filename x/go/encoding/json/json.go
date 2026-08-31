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
	"context"
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"io"
	"strconv"

	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/http"
)

// Codec is a JSON implementation of http.FileCodec with compact encoding.
var Codec = NewCodec()

// legacyWireSemantics holds the codec to the v1 wire format that released clients and
// stored files depend on. v2 changes more than a dozen defaults, among them encoding a
// nil slice as [] rather than null, so the whole v1 set is pinned rather than a chosen
// few.
var legacyWireSemantics = jsonv1.DefaultOptionsV1()

type codec struct {
	// indent is the per-level indentation for encoded output; empty means compact.
	indent string
	// escapeHTML is whether <, >, and & are escaped in encoded string values.
	escapeHTML bool
	// opts configures both directions of the underlying codec.
	opts json.Options
}

// NewCodec returns a JSON implementation of http.FileCodec configured with the given
// options.
func NewCodec(opts ...Option) http.FileCodec {
	c := &codec{escapeHTML: true}
	for _, opt := range opts {
		opt(c)
	}
	all := []json.Options{legacyWireSemantics, jsontext.EscapeForHTML(c.escapeHTML)}
	if c.indent != "" {
		all = append(all, jsontext.WithIndent(c.indent))
	}
	c.opts = json.JoinOptions(all...)
	return c
}

// Option configures a codec built by NewCodec.
type Option func(*codec)

// WithIndent encodes each level of nesting with the given indentation and appends a
// trailing newline, for files a user reads. Decoding is unaffected.
func WithIndent(indent string) Option { return func(c *codec) { c.indent = indent } }

// WithoutHTMLEscaping writes <, >, and & literally rather than as \u003c, \u003e, and
// \u0026, for files a user reads: JSON holding markup or source is unreadable escaped.
// U+2028 and U+2029 stay escaped either way, so the output is safe to embed in a
// script. Decoding is unaffected, so output encoded either way reads back the same.
//
// The escape only guards bytes placed into an HTML document without a parse, so drop it
// only where that cannot happen.
func WithoutHTMLEscaping() Option { return func(c *codec) { c.escapeHTML = false } }

func (*codec) ContentType() string { return "application/json" }

func (c *codec) Decode(_ context.Context, data []byte, value any) error {
	if err := json.Unmarshal(data, value, c.opts); err != nil {
		return encoding.SugarDecodingError(data, value, err)
	}
	return nil
}

// DecodeStream reads one value from r and leaves anything after it unread, so a reader
// holding a stream of values, or a file with bytes past the value, still decodes.
func (c *codec) DecodeStream(_ context.Context, r io.Reader, value any) error {
	dec := jsontext.NewDecoder(r, c.opts)
	if err := json.UnmarshalDecode(dec, value, c.opts); err != nil {
		data, ioErr := io.ReadAll(r)
		return encoding.SugarDecodingError(data, value, errors.Combine(err, ioErr))
	}
	return nil
}

func (c *codec) Encode(_ context.Context, value any) ([]byte, error) {
	b, err := json.Marshal(value, c.opts)
	if err != nil {
		return nil, encoding.SugarEncodingError(value, err)
	}
	if c.indent != "" {
		b = append(b, '\n')
	}
	return b, nil
}

func (c *codec) EncodeStream(_ context.Context, w io.Writer, value any) error {
	if err := json.MarshalWrite(w, value, c.opts); err != nil {
		return encoding.SugarEncodingError(value, err)
	}
	if c.indent == "" {
		return nil
	}
	_, err := w.Write([]byte{'\n'})
	return err
}

func (*codec) Extension() string { return ".json" }

// Validate reports the first defect in data under strict JSON rules: malformed syntax,
// a duplicate object name, or invalid UTF-8. The codec accepts the last two for
// backward compatibility, so a caller reading a document from outside the Core runs
// this first rather than silently taking the last of a repeated name. The error names
// the offending location as a JSON Pointer.
func Validate(data []byte) error {
	return jsontext.NewDecoder(bytes.NewReader(data)).SkipValue()
}

// MarshalStringInt64To writes the int64 to enc as a JSON string.
func MarshalStringInt64To(enc *jsontext.Encoder, n int64) error {
	return enc.WriteToken(jsontext.String(strconv.FormatInt(n, 10)))
}

// MarshalStringUint64To writes the uint64 to enc as a JSON string.
func MarshalStringUint64To(enc *jsontext.Encoder, n uint64) error {
	return enc.WriteToken(jsontext.String(strconv.FormatUint(n, 10)))
}

// readDigits reads the next token from dec as its decimal digits. A JSON number and a
// JSON string both decode, so a value written as either form reads back.
func readDigits(dec *jsontext.Decoder) (string, error) {
	tok, err := dec.ReadToken()
	if err != nil {
		return "", err
	}
	if k := tok.Kind(); k != '0' && k != '"' {
		return "", errors.Newf("cannot decode a number from JSON %s", k.String())
	}
	return tok.String(), nil
}

// UnmarshalStringInt64From reads an int64 from dec, accepting a JSON number or a JSON
// string holding the decimal digits.
func UnmarshalStringInt64From(dec *jsontext.Decoder) (int64, error) {
	digits, err := readDigits(dec)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(digits, 10, 64)
}

// UnmarshalStringUint32From reads a uint32 from dec, accepting a JSON number or a JSON
// string holding the decimal digits.
func UnmarshalStringUint32From(dec *jsontext.Decoder) (uint32, error) {
	digits, err := readDigits(dec)
	if err != nil {
		return 0, err
	}
	n, err := strconv.ParseUint(digits, 10, 32)
	return uint32(n), err
}

// UnmarshalStringUint64From reads a uint64 from dec, accepting a JSON number or a JSON
// string holding the decimal digits.
func UnmarshalStringUint64From(dec *jsontext.Decoder) (uint64, error) {
	digits, err := readDigits(dec)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(digits, 10, 64)
}
