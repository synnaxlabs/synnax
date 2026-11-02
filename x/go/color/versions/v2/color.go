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
	"math"
	"strconv"
	"strings"

	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
	"github.com/vmihailenco/msgpack/v5/msgpcode"
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

// legacyAlpha converts an alpha from a legacy tuple or object form, written on the 0-1
// scale, to a byte. Old Consoles wrote a 0-255 alpha into the same slot, so a value
// above 2 is taken as a byte. A value in (1, 2] is a 0-1 alpha nudged past 1 by float
// error and means opaque.
func legacyAlpha(a float64) (uint8, error) {
	switch {
	case a < 0 || a > 255 || math.IsNaN(a):
		return 0, errors.Wrapf(
			validate.ErrValidation, "alpha %v is outside the 0-255 range", a,
		)
	case a <= 1:
		return uint8(math.Round(a * 255)), nil
	case a <= 2:
		return 255, nil
	default:
		return uint8(math.Round(a)), nil
	}
}

func channelFromJSON(n jsontext.Value) (uint8, error) {
	if n.Kind() != '0' {
		return 0, errors.Wrapf(
			validate.ErrValidation, "color channel %s is not a number", n,
		)
	}
	v, err := strconv.ParseInt(string(n), 10, 64)
	if err != nil || v < 0 || v > 255 {
		return 0, errors.Wrapf(
			validate.ErrValidation, "color channel %s is outside the 0-255 range", n,
		)
	}
	return uint8(v), nil
}

func alphaFromJSON(n jsontext.Value) (uint8, error) {
	if n.Kind() != '0' {
		return 0, errors.Wrapf(
			validate.ErrValidation,
			"color alpha %s is not a number",
			n,
		)
	}
	a, err := strconv.ParseFloat(string(n), 64)
	if err != nil {
		return 0, err
	}
	return legacyAlpha(a)
}

// fromJSONNumbers parses a legacy [R, G, B] or [R, G, B, A] tuple.
func fromJSONNumbers(arr []jsontext.Value) (Color, error) {
	if len(arr) != 3 && len(arr) != 4 {
		return Color{}, errors.Wrapf(
			validate.ErrValidation, "invalid color array length: %d", len(arr),
		)
	}
	var (
		c   = Color{A: 255}
		err error
	)
	for i, ch := range []*uint8{&c.R, &c.G, &c.B} {
		if *ch, err = channelFromJSON(arr[i]); err != nil {
			return Color{}, err
		}
	}
	if len(arr) == 4 {
		if c.A, err = alphaFromJSON(arr[3]); err != nil {
			return Color{}, err
		}
	}
	return c, nil
}

// fromJSONObject parses a legacy {"r", "g", "b", "a"} or {"rgba255": [...]} object. An
// empty object is the zero color.
func fromJSONObject(data jsontext.Value) (Color, error) {
	var obj struct {
		R       jsontext.Value   `json:"r"`
		G       jsontext.Value   `json:"g"`
		B       jsontext.Value   `json:"b"`
		A       jsontext.Value   `json:"a"`
		RGBA255 []jsontext.Value `json:"rgba255"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return Color{}, errors.Wrapf(
			validate.ErrValidation, "cannot unmarshal color from: %s", data,
		)
	}
	if obj.RGBA255 != nil {
		return fromJSONNumbers(obj.RGBA255)
	}
	if obj.R == nil && obj.G == nil && obj.B == nil && obj.A == nil {
		return Color{}, nil
	}
	numbers := []jsontext.Value{obj.R, obj.G, obj.B}
	for i, n := range numbers {
		if n == nil {
			numbers[i] = jsontext.Value("0")
		}
	}
	if obj.A != nil {
		numbers = append(numbers, obj.A)
	}
	return fromJSONNumbers(numbers)
}

// MarshalJSONTo writes the color as an 8 digit "#rrggbbaa" string.
func (c Color) MarshalJSONTo(enc *jsontext.Encoder) error {
	return enc.WriteToken(jsontext.String(c.Hex()))
}

// UnmarshalJSONFrom reads a hex string, or one of the legacy forms: an [R, G, B, A]
// tuple, an {"r", "g", "b", "a"} object, or an {"rgba255": [R, G, B, A]} object. A
// legacy alpha is on the 0-1 scale, and an absent alpha is 255. JSON null, the empty
// string, and the empty object decode to the zero color.
func (c *Color) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	data, err := dec.ReadValue()
	if err != nil {
		return err
	}
	switch data.Kind() {
	case 'n':
		*c = Color{}
		return nil
	case '"':
		var s string
		if err = json.Unmarshal(data, &s); err != nil {
			return err
		}
		if s == "" {
			*c = Color{}
			return nil
		}
		*c, err = FromHex(s)
		return err
	case '[':
		var arr []jsontext.Value
		if err = json.Unmarshal(data, &arr); err != nil {
			return err
		}
		*c, err = fromJSONNumbers(arr)
		return err
	case '{':
		*c, err = fromJSONObject(data)
		return err
	default:
		return errors.Wrapf(
			validate.ErrValidation, "cannot unmarshal color from: %s", data,
		)
	}
}

// EncodeMsgpack writes the color as an 8 digit "#rrggbbaa" string.
func (c Color) EncodeMsgpack(enc *msgpack.Encoder) error {
	return enc.EncodeString(c.Hex())
}

func alphaFromMsgpack(dec *msgpack.Decoder) (uint8, error) {
	a, err := dec.DecodeFloat64()
	if err != nil {
		return 0, err
	}
	return legacyAlpha(a)
}

// DecodeMsgpack reads the same forms as UnmarshalJSONFrom.
func (c *Color) DecodeMsgpack(dec *msgpack.Decoder) error {
	code, err := dec.PeekCode()
	if err != nil {
		return err
	}
	switch {
	case code == msgpcode.Nil:
		*c = Color{}
		return dec.DecodeNil()
	case msgpcode.IsString(code):
		s, err := dec.DecodeString()
		if err != nil {
			return err
		}
		if s == "" {
			*c = Color{}
			return nil
		}
		*c, err = FromHex(s)
		return err
	case msgpcode.IsFixedArray(code) || code == msgpcode.Array16 ||
		code == msgpcode.Array32:
		n, err := dec.DecodeArrayLen()
		if err != nil {
			return err
		}
		*c, err = fromMsgpackTuple(dec, n)
		return err
	default:
		*c, err = fromMsgpackMap(dec)
		return err
	}
}

func fromMsgpackTuple(dec *msgpack.Decoder, n int) (Color, error) {
	if n != 3 && n != 4 {
		return Color{}, errors.Wrapf(
			validate.ErrValidation, "invalid color array length: %d", n,
		)
	}
	var (
		c   = Color{A: 255}
		err error
	)
	for _, ch := range []*uint8{&c.R, &c.G, &c.B} {
		if *ch, err = dec.DecodeUint8(); err != nil {
			return Color{}, err
		}
	}
	if n == 4 {
		if c.A, err = alphaFromMsgpack(dec); err != nil {
			return Color{}, err
		}
	}
	return c, nil
}

func fromMsgpackMap(dec *msgpack.Decoder) (Color, error) {
	n, err := dec.DecodeMapLen()
	if err != nil {
		return Color{}, err
	}
	c := Color{A: 255}
	for range n {
		key, err := dec.DecodeString()
		if err != nil {
			return Color{}, err
		}
		switch key {
		case "r":
			c.R, err = dec.DecodeUint8()
		case "g":
			c.G, err = dec.DecodeUint8()
		case "b":
			c.B, err = dec.DecodeUint8()
		case "a":
			c.A, err = alphaFromMsgpack(dec)
		case "rgba255":
			var l int
			if l, err = dec.DecodeArrayLen(); err != nil {
				return Color{}, err
			}
			if c, err = fromMsgpackTuple(dec, l); err != nil {
				return Color{}, err
			}
		default:
			err = dec.Skip()
		}
		if err != nil {
			return Color{}, err
		}
	}
	if n == 0 {
		return Color{}, nil
	}
	return c, nil
}

// MarshalText returns the color as an 8 digit "#rrggbbaa" string.
func (c Color) MarshalText() ([]byte, error) { return []byte(c.Hex()), nil }
