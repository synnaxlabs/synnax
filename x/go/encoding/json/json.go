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
	"encoding/json/v2"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/http"
)

// Codec is a JSON implementation of http.FileCodec with compact encoding.
var Codec = NewCodec()

type codec struct {
	trailingNewline bool
	opts            json.Options
}

// NewCodec returns a JSON implementation of http.FileCodec configured with the given
// options. Indented output also ends in a newline.
func NewCodec(opts ...json.Options) http.FileCodec {
	joined := json.JoinOptions(opts...)
	indent, _ := json.GetOption(joined, jsontext.WithIndent)
	return &codec{opts: joined, trailingNewline: indent != ""}
}

func (*codec) ContentType() string { return "application/json" }

func (c *codec) Decode(data []byte, value any) error {
	if err := json.Unmarshal(data, value, c.opts); err != nil {
		return encoding.SugarDecodingError(value, describe(err))
	}
	return nil
}

func (c *codec) DecodeStream(r io.Reader, value any) error {
	dec := jsontext.NewDecoder(r, c.opts)
	if err := json.UnmarshalDecode(dec, value, c.opts); err != nil {
		return encoding.SugarDecodingError(value, describe(err))
	}
	return nil
}

// fieldError restates a json.SemanticError by the field it failed on.
type fieldError struct {
	msg string
	err *json.SemanticError
}

func (e fieldError) Error() string { return e.msg }

func (e fieldError) Unwrap() error { return e.err }

// describe restates a json.SemanticError as the field that failed and the type that
// the field expects. It returns any other error as is.
func describe(err error) error {
	semErr, ok := errors.AsType[*json.SemanticError](err)
	if !ok {
		return err
	}
	field := strings.Join(slices.Collect(semErr.JSONPointer.Tokens()), ".")
	switch {
	case errors.Is(semErr.Err, json.ErrUnknownName):
		return fieldError{msg: fmt.Sprintf("unknown field %q", field), err: semErr}
	case field == "" && semErr.Err != nil:
		return fieldError{msg: semErr.Err.Error(), err: semErr}
	case field == "":
		return fieldError{msg: "got JSON " + kindName(semErr.JSONKind), err: semErr}
	}
	msg := fmt.Sprintf("field %q expects %s", field, semErr.GoType)
	if semErr.Err != nil {
		msg += ": " + semErr.Err.Error()
	} else {
		msg += ", got JSON " + kindName(semErr.JSONKind)
	}
	return fieldError{msg: msg, err: semErr}
}

func kindName(k jsontext.Kind) string {
	switch k {
	case 't', 'f':
		return "boolean"
	case '{', '}':
		return "object"
	case '[', ']':
		return "array"
	default:
		return k.String()
	}
}

func (c *codec) Encode(value any) ([]byte, error) {
	b, err := json.Marshal(value, c.opts)
	if err != nil {
		return nil, encoding.SugarEncodingError(value, err)
	}
	if c.trailingNewline {
		b = append(b, '\n')
	}
	return b, nil
}

func (c *codec) EncodeStream(w io.Writer, value any) error {
	if err := json.MarshalWrite(w, value, c.opts); err != nil {
		return encoding.SugarEncodingError(value, err)
	}
	if !c.trailingNewline {
		return nil
	}
	_, err := w.Write([]byte{'\n'})
	return err
}

func (*codec) Extension() string { return ".json" }

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
	if err != nil {
		return 0, err
	}
	return uint32(n), nil
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
