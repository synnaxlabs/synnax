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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/errors"
	"github.com/vmihailenco/msgpack/v5"
)

func (c Color) IsZero() bool {
	return c.R == 0 && c.G == 0 && c.B == 0 && c.A == 0
}

func FromHex(s string) (Color, error) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 && len(s) != 8 {
		return Color{}, errors.Newf("invalid hex color: %q", s)
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return Color{}, errors.Newf("invalid hex color: %q: %w", s, err)
	}
	alpha := 1.0
	if len(b) == 4 {
		alpha = float64(b[3]) / 255.0
	}
	return Color{R: b[0], G: b[1], B: b[2], A: alpha}, nil
}

func MustFromHex(s string) Color { return lo.Must(FromHex(s)) }

func (c Color) Hex() string {
	alphaByte := uint8(c.A * 255)
	if alphaByte == 255 {
		return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
	}
	return fmt.Sprintf("#%02x%02x%02x%02x", c.R, c.G, c.B, alphaByte)
}

func (c *Color) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	switch data[0] {
	case 'n':
		return nil
	case '{':
		type alias Color
		var obj alias
		if err := json.Unmarshal(data, &obj); err != nil {
			return err
		}
		*c = Color(obj)
		return nil
	case '[':
		var arr []float64
		if err := json.Unmarshal(data, &arr); err != nil {
			return err
		}
		if len(arr) != 4 {
			return errors.Newf("color array must have 4 elements, got %d", len(arr))
		}
		c.R, c.G, c.B, c.A = uint8(arr[0]), uint8(arr[1]), uint8(arr[2]), arr[3]
		return nil
	case '"':
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		parsed, err := FromHex(s)
		if err != nil {
			return err
		}
		*c = parsed
		return nil
	}
	return errors.Newf("invalid color format")
}

func (c *Color) DecodeMsgpack(dec *msgpack.Decoder) error {
	code, err := dec.PeekCode()
	if err != nil {
		return err
	}

	switch {
	case code == 0xc0:
		return dec.DecodeNil()

	case code >= 0x80 && code <= 0x8f || code == 0xde || code == 0xdf:
		type alias Color
		var obj alias
		if err := dec.Decode(&obj); err != nil {
			return err
		}
		*c = Color(obj)
		return nil

	case code >= 0x90 && code <= 0x9f || code == 0xdc || code == 0xdd:
		var arr []float64
		if err := dec.Decode(&arr); err != nil {
			return err
		}
		if len(arr) != 4 {
			return errors.Newf("color array must have 4 elements, got %d", len(arr))
		}
		c.R, c.G, c.B, c.A = uint8(arr[0]), uint8(arr[1]), uint8(arr[2]), arr[3]
		return nil

	case code >= 0xa0 && code <= 0xbf || code >= 0xd9 && code <= 0xdb:
		var s string
		if err := dec.Decode(&s); err != nil {
			return err
		}
		parsed, err := FromHex(s)
		if err != nil {
			return err
		}
		*c = parsed
		return nil
	}

	return errors.Newf("invalid color format")
}
