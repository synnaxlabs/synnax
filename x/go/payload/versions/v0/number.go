// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"math"
	"strconv"
	"strings"

	xjson "github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Float is a 64-bit float. It is a JSON number when finite, and the text "NaN",
// "Infinity", or "-Infinity" when not.
type Float float64

var (
	_ json.MarshalerTo     = Float(0)
	_ json.UnmarshalerFrom = (*Float)(nil)
	_ json.MarshalerTo     = Integer(0)
	_ json.UnmarshalerFrom = (*Integer)(nil)
	_ json.MarshalerTo     = Unsigned(0)
	_ json.UnmarshalerFrom = (*Unsigned)(nil)
)

// MarshalJSONTo implements json.MarshalerTo.
func (f Float) MarshalJSONTo(enc *jsontext.Encoder) error {
	v := float64(f)
	switch {
	case math.IsNaN(v):
		return enc.WriteToken(jsontext.String("NaN"))
	case math.IsInf(v, 1):
		return enc.WriteToken(jsontext.String("Infinity"))
	case math.IsInf(v, -1):
		return enc.WriteToken(jsontext.String("-Infinity"))
	default:
		return enc.WriteToken(jsontext.Float(v))
	}
}

// UnmarshalJSONFrom implements json.UnmarshalerFrom. It accepts a JSON number or the
// text "NaN", "Infinity", or "-Infinity".
func (f *Float) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	tok, err := dec.ReadToken()
	if err != nil {
		return err
	}
	switch tok.Kind() {
	case '0':
		v, err := tok.Float()
		if err != nil {
			return errors.Wrapf(
				validate.ErrValidation,
				"cannot decode a float: %v",
				err,
			)
		}
		*f = Float(v)
		return nil
	case '"':
		switch s := tok.String(); s {
		case "NaN":
			*f = Float(math.NaN())
		case "Infinity":
			*f = Float(math.Inf(1))
		case "-Infinity":
			*f = Float(math.Inf(-1))
		default:
			return errors.Wrapf(
				validate.ErrValidation,
				"cannot decode a float from %q",
				s,
			)
		}
		return nil
	default:
		return errors.Wrapf(
			validate.ErrValidation,
			"cannot decode a float from JSON %s",
			tok.Kind().String(),
		)
	}
}

// MarshalJSONTo implements json.MarshalerTo.
func (i Integer) MarshalJSONTo(enc *jsontext.Encoder) error {
	return xjson.MarshalStringInt64To(enc, int64(i))
}

// UnmarshalJSONFrom implements json.UnmarshalerFrom. It accepts a JSON number or a JSON
// string of decimal digits.
func (i *Integer) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	digits, err := readDigits(dec, "an integer")
	if err != nil {
		return err
	}
	n, err := strconv.ParseInt(digits, 10, 64)
	if err != nil {
		return errors.Wrapf(validate.ErrValidation, "cannot decode an integer: %v", err)
	}
	*i = Integer(n)
	return nil
}

// MarshalJSONTo implements json.MarshalerTo.
func (u Unsigned) MarshalJSONTo(enc *jsontext.Encoder) error {
	return xjson.MarshalStringUint64To(enc, uint64(u))
}

// UnmarshalJSONFrom implements json.UnmarshalerFrom. It accepts a JSON number or a JSON
// string of decimal digits.
func (u *Unsigned) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	digits, err := readDigits(dec, "an unsigned integer")
	if err != nil {
		return err
	}
	n, err := strconv.ParseUint(digits, 10, 64)
	if err != nil {
		return errors.Wrapf(
			validate.ErrValidation,
			"cannot decode an unsigned integer: %v",
			err,
		)
	}
	*u = Unsigned(n)
	return nil
}

func readDigits(dec *jsontext.Decoder, name string) (string, error) {
	tok, err := dec.ReadToken()
	if err != nil {
		return "", err
	}
	if k := tok.Kind(); k != '0' && k != '"' {
		return "", errors.Wrapf(
			validate.ErrValidation,
			"cannot decode %s from JSON %s",
			name,
			k.String(),
		)
	}
	digits := tok.String()
	if strings.HasPrefix(digits, "+") {
		return "", errors.Wrapf(
			validate.ErrValidation,
			"cannot decode %s from %q",
			name,
			digits,
		)
	}
	return digits, nil
}
