// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package color

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vmihailenco/msgpack/v5"
	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

// Color represents an RGBA color with 8-bit RGB channels and a float64 alpha.
type Color struct {
	R uint8   `json:"r" msgpack:"r"`
	G uint8   `json:"g" msgpack:"g"`
	B uint8   `json:"b" msgpack:"b"`
	A float64 `json:"a" msgpack:"a"`
}

// IsZero returns true if the color is the zero value for its type.
func (c Color) IsZero() bool { return c.R == 0 && c.G == 0 && c.B == 0 && c.A == 0 }

// Hex returns the hex string representation of the color (e.g. "#ff0000" or
// "#ff000080" if alpha is not 1).
func (c Color) Hex() string {
	alphaByte := uint8(c.A * 255)
	if alphaByte == 255 {
		return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
	}
	return fmt.Sprintf("#%02x%02x%02x%02x", c.R, c.G, c.B, alphaByte)
}

// FromHex parses a hex color string into a Color. Supports 6 or 8 character hex
// strings with or without a leading '#'.
func FromHex(s string) (Color, error) {
	s = strings.TrimPrefix(s, "#")
	var r, g, b, a uint8
	switch len(s) {
	case 6:
		_, err := fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b)
		if err != nil {
			return Color{}, fmt.Errorf("invalid hex color: %q", s)
		}
		return Color{R: r, G: g, B: b, A: 1}, nil
	case 8:
		_, err := fmt.Sscanf(s, "%02x%02x%02x%02x", &r, &g, &b, &a)
		if err != nil {
			return Color{}, fmt.Errorf("invalid hex color: %q", s)
		}
		return Color{R: r, G: g, B: b, A: float64(a) / 255}, nil
	default:
		return Color{}, fmt.Errorf("invalid hex color length: %q", s)
	}
}

// MustFromHex parses a hex color string into a Color, panicking on error.
func MustFromHex(s string) Color {
	c, err := FromHex(s)
	if err != nil {
		panic(err)
	}
	return c
}

// UnmarshalJSON supports three formats:
//   - string: "#ff0000" or "#ff000080"
//   - array:  [255, 0, 0, 1.0]
//   - object: {"r": 255, "g": 0, "b": 0, "a": 1.0}
func (c *Color) UnmarshalJSON(data []byte) error {
	// Try string first
	var s string
	if json.Unmarshal(data, &s) == nil {
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
		r, err := arr[0].Int64()
		if err != nil {
			return err
		}
		g, err := arr[1].Int64()
		if err != nil {
			return err
		}
		b, err := arr[2].Int64()
		if err != nil {
			return err
		}
		a := 1.0
		if len(arr) == 4 {
			a, err = arr[3].Float64()
			if err != nil {
				return err
			}
		}
		*c = Color{R: uint8(r), G: uint8(g), B: uint8(b), A: a}
		return nil
	}

	// Try object {"r": ..., "g": ..., "b": ..., "a": ...}
	type colorAlias Color
	var obj colorAlias
	if err := json.Unmarshal(data, &obj); err != nil {
		return fmt.Errorf("cannot unmarshal color from: %s", string(data))
	}
	*c = Color(obj)
	return nil
}

// DecodeMsgpack supports backwards-compatible decoding from msgpack. Old data stored
// as a string (hex) in Pebble will be decoded correctly. Also handles map (struct)
// and array formats.
func (c *Color) DecodeMsgpack(dec *msgpack.Decoder) error {
	code, err := dec.PeekCode()
	if err != nil {
		return err
	}

	if msgpcode.IsString(code) {
		s, err := dec.DecodeString()
		if err != nil {
			return err
		}
		parsed, err := FromHex(s)
		if err != nil {
			return err
		}
		*c = parsed
		return nil
	}

	if msgpcode.IsFixedArray(code) || code == msgpcode.Array16 || code == msgpcode.Array32 {
		arrLen, err := dec.DecodeArrayLen()
		if err != nil {
			return err
		}
		if arrLen < 3 || arrLen > 4 {
			return fmt.Errorf("invalid color array length: %d", arrLen)
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
	*c = Color(obj)
	return nil
}
