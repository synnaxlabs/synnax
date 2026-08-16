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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

// IsZero returns true if the color is the zero value for its type.
func (c Color) IsZero() bool { return c.R == 0 && c.G == 0 && c.B == 0 && c.A == 0 }

// FromHex parses a hex color string into a Color. Supports 6 or 8 character hex strings
// with or without a leading '#'.
func FromHex(s string) (Color, error) {
	s = strings.TrimPrefix(s, "#")
	var r, g, b, a uint8
	switch len(s) {
	case 6:
		_, err := fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b)
		if err != nil {
			return Color{}, errors.Wrapf(
				validate.ErrValidation, "invalid hex color: %q", s,
			)
		}
		return Color{R: r, G: g, B: b, A: 1}, nil
	case 8:
		_, err := fmt.Sscanf(s, "%02x%02x%02x%02x", &r, &g, &b, &a)
		if err != nil {
			return Color{}, errors.Wrapf(
				validate.ErrValidation, "invalid hex color: %q", s,
			)
		}
		return Color{R: r, G: g, B: b, A: float64(a) / 255}, nil
	default:
		return Color{}, errors.Wrapf(
			validate.ErrValidation, "invalid hex color length: %q", s,
		)
	}
}

// normalizeAlpha lifts a legacy 0-255 alpha onto the current 0-1 scale. Old Consoles
// persisted alpha as a fourth 0-255 channel; the current format caps alpha at 1, so
// any larger value is legacy. Values in (1, 2] clamp to 1 instead of dividing: a
// legacy alpha that small means a sub-1% opacity no user sets, while a current-scale
// value nudged past 1 by float error means opaque. An alpha above 255 fits neither
// scale and fails validation.
func normalizeAlpha(a float64) (float64, error) {
	if a <= 1 {
		return a, nil
	}
	if a <= 2 {
		return 1, nil
	}
	if a > 255 {
		return 0, errors.Wrapf(
			validate.ErrValidation, "alpha %v is above the 0-255 scale", a,
		)
	}
	return a / 255, nil
}

// fromNumbers parses an [R, G, B] or [R, G, B, A] number tuple into a Color.
func fromNumbers(arr []json.Number) (Color, error) {
	r, err := arr[0].Int64()
	if err != nil {
		return Color{}, err
	}
	g, err := arr[1].Int64()
	if err != nil {
		return Color{}, err
	}
	b, err := arr[2].Int64()
	if err != nil {
		return Color{}, err
	}
	a := 1.0
	if len(arr) == 4 {
		if a, err = arr[3].Float64(); err != nil {
			return Color{}, err
		}
		if a, err = normalizeAlpha(a); err != nil {
			return Color{}, err
		}
	}
	return Color{R: uint8(r), G: uint8(g), B: uint8(b), A: a}, nil
}

// UnmarshalJSON supports four formats:
//   - string: "#ff0000" or "#ff000080"
//   - array:  [255, 0, 0, 1.0]
//   - object: {"r": 255, "g": 0, "b": 0, "a": 1.0}
//   - legacy object: {"rgba255": [255, 0, 0, 1.0]}
//
// JSON null and the empty string both decode to the zero Color. An alpha above 1 is
// lifted from the legacy 0-255 scale onto 0-1.
func (c *Color) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*c = Color{}
		return nil
	}
	var s string
	if json.Unmarshal(data, &s) == nil {
		if s == "" {
			*c = Color{}
			return nil
		}
		parsed, err := FromHex(s)
		if err != nil {
			return err
		}
		*c = parsed
		return nil
	}

	// Try array [R, G, B, A]
	var arr []json.Number
	if json.Unmarshal(data, &arr) == nil && (len(arr) == 3 || len(arr) == 4) {
		parsed, err := fromNumbers(arr)
		if err != nil {
			return err
		}
		*c = parsed
		return nil
	}

	// Try legacy object {"rgba255": [R, G, B, A]}
	var legacy struct {
		RGBA255 []json.Number `json:"rgba255"`
	}
	if json.Unmarshal(data, &legacy) == nil &&
		(len(legacy.RGBA255) == 3 || len(legacy.RGBA255) == 4) {
		parsed, err := fromNumbers(legacy.RGBA255)
		if err != nil {
			return err
		}
		*c = parsed
		return nil
	}

	// Try object {"r": ..., "g": ..., "b": ..., "a": ...}
	type colorAlias Color
	var obj colorAlias
	if err := json.Unmarshal(data, &obj); err != nil {
		return errors.Newf("cannot unmarshal color from: %s", string(data))
	}
	a, err := normalizeAlpha(obj.A)
	if err != nil {
		return err
	}
	obj.A = a
	*c = Color(obj)
	return nil
}

// DecodeMsgpack supports backwards-compatible decoding from MessagePack. Old data
// stored as a string (hex) in Pebble will be decoded correctly. Also handles map
// (struct) and array formats.
func (c *Color) DecodeMsgpack(dec *msgpack.Decoder) error {
	code, err := dec.PeekCode()
	if err != nil {
		return err
	}

	if code == msgpcode.Nil {
		if err := dec.DecodeNil(); err != nil {
			return err
		}
		*c = Color{}
		return nil
	}

	if msgpcode.IsString(code) {
		s, err := dec.DecodeString()
		if err != nil {
			return err
		}
		if s == "" {
			*c = Color{}
			return nil
		}
		parsed, err := FromHex(s)
		if err != nil {
			return err
		}
		*c = parsed
		return nil
	}

	if msgpcode.IsFixedArray(code) || code == msgpcode.Array16 ||
		code == msgpcode.Array32 {
		arrLen, err := dec.DecodeArrayLen()
		if err != nil {
			return err
		}
		if arrLen < 3 || arrLen > 4 {
			return errors.Newf("invalid color array length: %d", arrLen)
		}
		r, err := dec.DecodeUint8()
		if err != nil {
			return err
		}
		g, err := dec.DecodeUint8()
		if err != nil {
			return err
		}
		b, err := dec.DecodeUint8()
		if err != nil {
			return err
		}
		a := 1.0
		if arrLen == 4 {
			a, err = dec.DecodeFloat64()
			if err != nil {
				return err
			}
			if a, err = normalizeAlpha(a); err != nil {
				return err
			}
		}
		*c = Color{R: r, G: g, B: b, A: a}
		return nil
	}

	// Default: map (struct) format
	type colorAlias Color
	var obj colorAlias
	if err := dec.Decode(&obj); err != nil {
		return err
	}
	a, err := normalizeAlpha(obj.A)
	if err != nil {
		return err
	}
	obj.A = a
	*c = Color(obj)
	return nil
}
