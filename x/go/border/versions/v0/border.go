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

	"github.com/synnaxlabs/x/spatial"
)

// UnmarshalJSON supports four formats:
//   - number: 4
//   - pair:   {"x": 4, "y": 8}
//   - per-corner numbers: {"top_left": 4, ...}
//   - per-corner pairs:   {"top_left": {"x": 4, "y": 8}, ...}
//
// Shorthand forms expand to the canonical per-corner shape.
func (r *Radius) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*r = Radius{}
		return nil
	}
	var n float64
	if err := json.Unmarshal(data, &n); err == nil {
		v := spatial.XY{X: n, Y: n}
		*r = Radius{TopLeft: v, TopRight: v, BottomLeft: v, BottomRight: v}
		return nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	if _, hasX := obj["x"]; hasX {
		var v spatial.XY
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*r = Radius{TopLeft: v, TopRight: v, BottomLeft: v, BottomRight: v}
		return nil
	}
	corners := []struct {
		key string
		dst *spatial.XY
	}{
		{"top_left", &r.TopLeft},
		{"top_right", &r.TopRight},
		{"bottom_left", &r.BottomLeft},
		{"bottom_right", &r.BottomRight},
	}
	for _, c := range corners {
		raw, ok := obj[c.key]
		if !ok {
			continue
		}
		var num float64
		if err := json.Unmarshal(raw, &num); err == nil {
			*c.dst = spatial.XY{X: num, Y: num}
			continue
		}
		if err := json.Unmarshal(raw, c.dst); err != nil {
			return err
		}
	}
	return nil
}
