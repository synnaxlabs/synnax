// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

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
