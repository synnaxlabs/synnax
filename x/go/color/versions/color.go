// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions

import v2 "github.com/synnaxlabs/x/color/versions/v2"

// FromHex parses a 6 or 8 digit hex color string, with or without a leading '#'. The
// alpha of a 6 digit string is 255.
func FromHex(s string) (Color, error) { return v2.FromHex(s) }

// FromUint32 unpacks a color from its 0xRRGGBBAA form.
func FromUint32(v uint32) Color { return v2.FromUint32(v) }
