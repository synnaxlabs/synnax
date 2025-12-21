// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package units

import "github.com/synnaxlabs/arc/types"

// Registry maps unit names to their definitions.
// Each entry defines the dimensions, scale factor (to SI base unit), and display name.
var Registry = map[string]types.Unit{
	// Time units (base: seconds)
	"ns": {Dimensions: types.DimTime, Scale: 1e-9, Name: "ns"},
	"us": {Dimensions: types.DimTime, Scale: 1e-6, Name: "us"},
	"ms": {Dimensions: types.DimTime, Scale: 1e-3, Name: "ms"},
	"s":  {Dimensions: types.DimTime, Scale: 1, Name: "s"},
	"m":  {Dimensions: types.DimTime, Scale: 60, Name: "m"},
	"h":  {Dimensions: types.DimTime, Scale: 3600, Name: "h"},

	// Length units (base: meters)
	"nm":   {Dimensions: types.DimLength, Scale: 1e-9, Name: "nm"},
	"um":   {Dimensions: types.DimLength, Scale: 1e-6, Name: "um"},
	"mm":   {Dimensions: types.DimLength, Scale: 1e-3, Name: "mm"},
	"cm":   {Dimensions: types.DimLength, Scale: 1e-2, Name: "cm"},
	"dm":   {Dimensions: types.DimLength, Scale: 1e-1, Name: "dm"},
	"km":   {Dimensions: types.DimLength, Scale: 1e3, Name: "km"},
	"in":   {Dimensions: types.DimLength, Scale: 0.0254, Name: "in"},
	"ft":   {Dimensions: types.DimLength, Scale: 0.3048, Name: "ft"},
	"yd":   {Dimensions: types.DimLength, Scale: 0.9144, Name: "yd"},
	"mi":   {Dimensions: types.DimLength, Scale: 1609.344, Name: "mi"},
	"nmi":  {Dimensions: types.DimLength, Scale: 1852, Name: "nmi"},
	"meter": {Dimensions: types.DimLength, Scale: 1, Name: "meter"},

	// Mass units (base: kilograms)
	"ug":  {Dimensions: types.DimMass, Scale: 1e-9, Name: "ug"},
	"mg":  {Dimensions: types.DimMass, Scale: 1e-6, Name: "mg"},
	"g":   {Dimensions: types.DimMass, Scale: 1e-3, Name: "g"},
	"kg":  {Dimensions: types.DimMass, Scale: 1, Name: "kg"},
	"lb":  {Dimensions: types.DimMass, Scale: 0.453592, Name: "lb"},
	"oz":  {Dimensions: types.DimMass, Scale: 0.0283495, Name: "oz"},
	"ton": {Dimensions: types.DimMass, Scale: 1000, Name: "ton"},

	// Pressure units (base: Pascals)
	"Pa":   {Dimensions: types.DimPressure, Scale: 1, Name: "Pa"},
	"kPa":  {Dimensions: types.DimPressure, Scale: 1e3, Name: "kPa"},
	"MPa":  {Dimensions: types.DimPressure, Scale: 1e6, Name: "MPa"},
	"bar":  {Dimensions: types.DimPressure, Scale: 1e5, Name: "bar"},
	"mbar": {Dimensions: types.DimPressure, Scale: 100, Name: "mbar"},
	"psi":  {Dimensions: types.DimPressure, Scale: 6894.76, Name: "psi"},
	"atm":  {Dimensions: types.DimPressure, Scale: 101325, Name: "atm"},
	"torr": {Dimensions: types.DimPressure, Scale: 133.322, Name: "torr"},

	// Frequency units (base: Hertz)
	"Hz":  {Dimensions: types.DimFrequency, Scale: 1, Name: "Hz"},
	"hz":  {Dimensions: types.DimFrequency, Scale: 1, Name: "hz"},
	"kHz": {Dimensions: types.DimFrequency, Scale: 1e3, Name: "kHz"},
	"khz": {Dimensions: types.DimFrequency, Scale: 1e3, Name: "khz"},
	"MHz": {Dimensions: types.DimFrequency, Scale: 1e6, Name: "MHz"},
	"mhz": {Dimensions: types.DimFrequency, Scale: 1e6, Name: "mhz"},
	"GHz": {Dimensions: types.DimFrequency, Scale: 1e9, Name: "GHz"},

	// Voltage units (base: Volts)
	"uV": {Dimensions: types.DimVoltage, Scale: 1e-6, Name: "uV"},
	"mV": {Dimensions: types.DimVoltage, Scale: 1e-3, Name: "mV"},
	"V":  {Dimensions: types.DimVoltage, Scale: 1, Name: "V"},
	"kV": {Dimensions: types.DimVoltage, Scale: 1e3, Name: "kV"},

	// Current units (base: Amperes)
	"uA": {Dimensions: types.DimCurrent, Scale: 1e-6, Name: "uA"},
	"mA": {Dimensions: types.DimCurrent, Scale: 1e-3, Name: "mA"},
	"A":  {Dimensions: types.DimCurrent, Scale: 1, Name: "A"},
	"kA": {Dimensions: types.DimCurrent, Scale: 1e3, Name: "kA"},

	// Temperature units (treated as incompatible in Phase 1)
	// Using Temperature dimension with different "base" scales
	// C, F, K cannot be mixed - offset conversions not yet supported
	"C": {Dimensions: types.DimTemperature, Scale: 1, Name: "C"},
	"K": {Dimensions: types.DimTemperature, Scale: 1, Name: "K"},
	"F": {Dimensions: types.DimTemperature, Scale: 1, Name: "F"},

	// Angle units (base: radians)
	"rad": {Dimensions: types.DimAngle, Scale: 1, Name: "rad"},
	"deg": {Dimensions: types.DimAngle, Scale: 0.0174533, Name: "deg"},
	"rev": {Dimensions: types.DimAngle, Scale: 6.28319, Name: "rev"},

	// Data units (base: bits)
	"bit":  {Dimensions: types.DimData, Scale: 1, Name: "bit"},
	"byte": {Dimensions: types.DimData, Scale: 8, Name: "byte"},
	"KB":   {Dimensions: types.DimData, Scale: 8 * 1024, Name: "KB"},
	"MB":   {Dimensions: types.DimData, Scale: 8 * 1024 * 1024, Name: "MB"},
	"GB":   {Dimensions: types.DimData, Scale: 8 * 1024 * 1024 * 1024, Name: "GB"},
	"KiB":  {Dimensions: types.DimData, Scale: 8 * 1024, Name: "KiB"},
	"MiB":  {Dimensions: types.DimData, Scale: 8 * 1024 * 1024, Name: "MiB"},
	"GiB":  {Dimensions: types.DimData, Scale: 8 * 1024 * 1024 * 1024, Name: "GiB"},

	// Count units (dimensionless count)
	"count":   {Dimensions: types.DimCount, Scale: 1, Name: "count"},
	"samples": {Dimensions: types.DimCount, Scale: 1, Name: "samples"},
	"pct":     {Dimensions: types.DimNone, Scale: 0.01, Name: "pct"},
	"percent": {Dimensions: types.DimNone, Scale: 0.01, Name: "percent"},
}

// Lookup finds a unit by name. Returns the unit and true if found,
// or a zero Unit and false if not found.
func Lookup(name string) (types.Unit, bool) {
	u, ok := Registry[name]
	return u, ok
}

// IsValidUnit returns true if the given name is a valid unit.
func IsValidUnit(name string) bool {
	_, ok := Registry[name]
	return ok
}
