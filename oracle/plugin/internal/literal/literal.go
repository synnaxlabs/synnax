// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package literal renders schema values as source literals shared by the plugins.
package literal

import (
	"strconv"
)

// Float renders a float default as a decimal literal that reads back as v. It keeps
// the fixed six-digit form when that is exact, and falls back to the shortest exact
// form otherwise, so a small default such as 1e-7 does not become 0.000000.
func Float(v float64) string {
	fixed := strconv.FormatFloat(v, 'f', 6, 64)
	if parsed, err := strconv.ParseFloat(fixed, 64); err == nil && parsed == v {
		return fixed
	}
	return strconv.FormatFloat(v, 'g', -1, 64)
}
