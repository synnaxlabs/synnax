// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions

import v0 "github.com/synnaxlabs/x/color/versions/v0"

// FromHex parses a hex color string into a Color. Supports 6 or 8 character hex
// strings with or without a leading '#'.
func FromHex(s string) (Color, error) { return v0.FromHex(s) }
