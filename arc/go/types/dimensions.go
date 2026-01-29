// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import "fmt"

// Common dimension constants for base dimensions.
var (
	// DimNone represents a dimensionless quantity.
	DimNone = Dimensions{}

	// Base dimensions
	DimLength      = Dimensions{Length: 1}
	DimMass        = Dimensions{Mass: 1}
	DimTime        = Dimensions{Time: 1}
	DimCurrent     = Dimensions{Current: 1}
	DimTemperature = Dimensions{Temperature: 1}
	DimAngle       = Dimensions{Angle: 1}
	DimCount       = Dimensions{Count: 1}
	DimData        = Dimensions{Data: 1}

	// Derived dimensions (computed via multiplication/division of base dimensions)
	DimVelocity  = Dimensions{Length: 1, Time: -1}                       // m/s
	DimAccel     = Dimensions{Length: 1, Time: -2}                       // m/s^2
	DimForce     = Dimensions{Mass: 1, Length: 1, Time: -2}              // kg*m/s^2 (N)
	DimPressure  = Dimensions{Mass: 1, Length: -1, Time: -2}             // kg/(m*s^2) (Pa)
	DimEnergy    = Dimensions{Mass: 1, Length: 2, Time: -2}              // kg*m^2/s^2 (J)
	DimPower     = Dimensions{Mass: 1, Length: 2, Time: -3}              // kg*m^2/s^3 (W)
	DimFrequency = Dimensions{Time: -1}                                  // 1/s (Hz)
	DimVoltage   = Dimensions{Mass: 1, Length: 2, Time: -3, Current: -1} // V = kg*m^2/(A*s^3)
)

// Mul adds dimension exponents (for multiplication of quantities).
// For example, length * time^-1 = velocity.
func (d Dimensions) Mul(other Dimensions) Dimensions {
	return Dimensions{
		Length:      d.Length + other.Length,
		Mass:        d.Mass + other.Mass,
		Time:        d.Time + other.Time,
		Current:     d.Current + other.Current,
		Temperature: d.Temperature + other.Temperature,
		Angle:       d.Angle + other.Angle,
		Count:       d.Count + other.Count,
		Data:        d.Data + other.Data,
	}
}

// Div subtracts dimension exponents (for division of quantities).
// For example, length / time = velocity.
func (d Dimensions) Div(other Dimensions) Dimensions {
	return Dimensions{
		Length:      d.Length - other.Length,
		Mass:        d.Mass - other.Mass,
		Time:        d.Time - other.Time,
		Current:     d.Current - other.Current,
		Temperature: d.Temperature - other.Temperature,
		Angle:       d.Angle - other.Angle,
		Count:       d.Count - other.Count,
		Data:        d.Data - other.Data,
	}
}

// Scale multiplies all dimension exponents by n (for power operations).
// For example, length.Scale(2) = length^2 (area).
func (d Dimensions) Scale(n int8) Dimensions {
	return Dimensions{
		Length:      d.Length * n,
		Mass:        d.Mass * n,
		Time:        d.Time * n,
		Current:     d.Current * n,
		Temperature: d.Temperature * n,
		Angle:       d.Angle * n,
		Count:       d.Count * n,
		Data:        d.Data * n,
	}
}

// Equal checks if two dimensions are identical.
func (d Dimensions) Equal(other Dimensions) bool {
	return d == other
}

// IsZero returns true if the dimensions represent a dimensionless quantity.
func (d Dimensions) IsZero() bool {
	return d == Dimensions{}
}

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

	result := ""
	for i, p := range parts {
		if i > 0 {
			result += " "
		}
		result += p
	}
	return result
}

// Equal checks if two units are identical.
func (u Unit) Equal(other Unit) bool {
	return u.Dimensions.Equal(other.Dimensions) &&
		u.Scale == other.Scale &&
		u.Name == other.Name
}
