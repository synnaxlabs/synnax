// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"encoding/hex"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"strings"

	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
)

// FromHex parses a 6 or 8 digit hex color string, with or without a leading '#'. The
// alpha of a 6 digit string is 255.
func FromHex(s string) (Color, error) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 && len(s) != 8 {
		return Color{}, errors.Wrapf(
			validate.ErrValidation, "invalid hex color length: %q", s,
		)
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return Color{}, errors.Wrapf(validate.ErrValidation, "invalid hex color: %q", s)
	}
	c := Color{R: b[0], G: b[1], B: b[2], A: 255}
	if len(b) == 4 {
		c.A = b[3]
	}
	return c, nil
}

// FromUint32 unpacks a color from its 0xRRGGBBAA form.
func FromUint32(v uint32) Color {
	return Color{R: uint8(v >> 24), G: uint8(v >> 16), B: uint8(v >> 8), A: uint8(v)}
}

// Uint32 packs the color into its 0xRRGGBBAA form.
func (c Color) Uint32() uint32 {
	return uint32(c.R)<<24 | uint32(c.G)<<16 | uint32(c.B)<<8 | uint32(c.A)
}

// Hex returns the color as an 8 digit "#rrggbbaa" string.
func (c Color) Hex() string {
	return "#" + hex.EncodeToString([]byte{c.R, c.G, c.B, c.A})
}

// EncodeOrc writes the color as one packed 0xRRGGBBAA uint32.
func (c Color) EncodeOrc(w *orc.Writer) error {
	w.Uint32(c.Uint32())
	return nil
}

// DecodeOrc reads a color written by EncodeOrc.
func (c *Color) DecodeOrc(r *orc.Reader) error {
	v, err := r.Uint32()
	if err != nil {
		return err
	}
	*c = FromUint32(v)
	return nil
}

// MarshalJSONTo writes the color as an 8 digit "#rrggbbaa" string.
func (c Color) MarshalJSONTo(enc *jsontext.Encoder) error {
	return enc.WriteToken(jsontext.String(c.Hex()))
}

// UnmarshalJSONFrom reads a 6 or 8 digit hex string.
func (c *Color) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	var s string
	if err := json.UnmarshalDecode(dec, &s); err != nil {
		return errors.Wrapf(
			validate.ErrValidation,
			"color is not a hex string: %v",
			err,
		)
	}
	var err error
	*c, err = FromHex(s)
	return err
}

// EncodeMsgpack writes the color as an 8 digit "#rrggbbaa" string.
func (c Color) EncodeMsgpack(enc *msgpack.Encoder) error {
	return enc.EncodeString(c.Hex())
}

// DecodeMsgpack reads the same forms as UnmarshalJSONFrom.
func (c *Color) DecodeMsgpack(dec *msgpack.Decoder) error {
	s, err := dec.DecodeString()
	if err != nil {
		return errors.Wrapf(
			validate.ErrValidation,
			"color is not a hex string: %v",
			err,
		)
	}
	*c, err = FromHex(s)
	return err
}

// MarshalText returns the color as an 8 digit "#rrggbbaa" string.
func (c Color) MarshalText() ([]byte, error) { return []byte(c.Hex()), nil }
