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
	"fmt"
	"strings"
)

// Equal checks if two dimensions are identical.
func (d Dimensions) Equal(other Dimensions) bool { return d == other }

// IsZero returns true if the dimensions represent a dimensionless quantity.
func (d Dimensions) IsZero() bool { return d == Dimensions{} }

// String returns a human-readable representation of the dimensions.
// For example: "length^1 time^-1" for velocity.
func (d Dimensions) String() string {
	if d.IsZero() {
		return "dimensionless"
	}

	var parts []string
	if d.Length != 0 {
		parts = append(parts, fmt.Sprintf("length^%d", d.Length))
	}
	if d.Mass != 0 {
		parts = append(parts, fmt.Sprintf("mass^%d", d.Mass))
	}
	if d.Time != 0 {
		parts = append(parts, fmt.Sprintf("time^%d", d.Time))
	}
	if d.Current != 0 {
		parts = append(parts, fmt.Sprintf("current^%d", d.Current))
	}
	if d.Temperature != 0 {
		parts = append(parts, fmt.Sprintf("temperature^%d", d.Temperature))
	}
	if d.Angle != 0 {
		parts = append(parts, fmt.Sprintf("angle^%d", d.Angle))
	}
	if d.Count != 0 {
		parts = append(parts, fmt.Sprintf("count^%d", d.Count))
	}
	if d.Data != 0 {
		parts = append(parts, fmt.Sprintf("data^%d", d.Data))
	}

	var result strings.Builder
	for i, p := range parts {
		if i > 0 {
			result.WriteString(" ")
		}
		result.WriteString(p)
	}
	return result.String()
}
