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

// Dimensions represents dimension exponents as a vector for dimensional analysis.
// Uses SI base dimensions plus pragmatic extensions for hardware telemetry.
// Exponents are typically small (-3 to +3), so int8 is sufficient.
type Dimensions struct {
	// SI base dimensions
	Length      int8 `json:"length,omitempty" msgpack:"length"`           // meters (m)
	Mass        int8 `json:"mass,omitempty" msgpack:"mass"`               // kilograms (kg)
	Time        int8 `json:"time,omitempty" msgpack:"time"`               // seconds (s)
	Current     int8 `json:"current,omitempty" msgpack:"current"`         // amperes (A)
	Temperature int8 `json:"temperature,omitempty" msgpack:"temperature"` // kelvin (K)

	// Pragmatic extensions (not SI, but useful for hardware telemetry)
	Angle int8 `json:"angle,omitempty" msgpack:"angle"` // radians/degrees - distinct from dimensionless
	Count int8 `json:"count,omitempty" msgpack:"count"` // samples, items, cycles, pixels
	Data  int8 `json:"data,omitempty" msgpack:"data"`   // bits, bytes
}

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

// Unit holds unit metadata for numeric types.
// A Unit is attached to a Type to indicate that the value has physical dimensions.
type Unit struct {
	// Name is the display name for this unit (e.g., "psi", "km", "ms").
	Name string `json:"name" msgpack:"name"`
	// Scale is the factor to convert this unit to SI base units.
	// For example, km has Scale=1000 (1 km = 1000 m).
	Scale float64 `json:"scale" msgpack:"scale"`
	// Dimensions contains the dimension exponents for this unit.
	Dimensions Dimensions `json:"dimensions" msgpack:"dimensions"`
}

// Equal checks if two units are identical.
func (u Unit) Equal(other Unit) bool {
	return u.Dimensions.Equal(other.Dimensions) &&
		u.Scale == other.Scale &&
		u.Name == other.Name
}
