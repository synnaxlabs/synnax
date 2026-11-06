// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pb

import "github.com/synnaxlabs/x/color"

// ColorToPB converts a color to its packed protobuf form. It never returns an error.
func ColorToPB(c color.Color) (*Color, error) { return &Color{Rgba: c.Uint32()}, nil }

// ColorFromPB converts a packed protobuf color to a color. A nil message is the zero
// color. It never returns an error.
func ColorFromPB(pb *Color) (color.Color, error) {
	return color.FromUint32(pb.GetRgba()), nil
}
