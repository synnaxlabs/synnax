// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package units

import (
	"fmt"

	"github.com/synnaxlabs/arc/types"
)

// Registry maps unit names to their definitions.
// Each entry defines the dimensions, scale factor (to SI base unit), and display name.
var registry = map[string]types.Unit{
	// Time units (base: nanoseconds - matches Synnax's canonical representation)
	"ns":  {Dimensions: types.DimTime, Scale: 1, Name: "ns"},
	"us":  {Dimensions: types.DimTime, Scale: 1e3, Name: "us"},
	"ms":  {Dimensions: types.DimTime, Scale: 1e6, Name: "ms"},
	"s":   {Dimensions: types.DimTime, Scale: 1e9, Name: "s"},
	"min": {Dimensions: types.DimTime, Scale: 60e9, Name: "min"},
	"h":   {Dimensions: types.DimTime, Scale: 3600e9, Name: "h"},

	// Length units (base: meters)
	"fm":    {Dimensions: types.DimLength, Scale: 1e-15, Name: "fm"},
	"pm":    {Dimensions: types.DimLength, Scale: 1e-12, Name: "pm"},
	"nm":    {Dimensions: types.DimLength, Scale: 1e-9, Name: "nm"},
	"um":    {Dimensions: types.DimLength, Scale: 1e-6, Name: "um"},
	"mm":    {Dimensions: types.DimLength, Scale: 1e-3, Name: "mm"},
	"cm":    {Dimensions: types.DimLength, Scale: 1e-2, Name: "cm"},
	"dm":    {Dimensions: types.DimLength, Scale: 1e-1, Name: "dm"},
	"m":     {Dimensions: types.DimLength, Scale: 1, Name: "m"},
	"km":    {Dimensions: types.DimLength, Scale: 1e3, Name: "km"},
	"in":    {Dimensions: types.DimLength, Scale: 0.0254, Name: "in"},
	"ft":    {Dimensions: types.DimLength, Scale: 0.3048, Name: "ft"},
	"yd":    {Dimensions: types.DimLength, Scale: 0.9144, Name: "yd"},
	"mi":    {Dimensions: types.DimLength, Scale: 1609.344, Name: "mi"},
	"nmi":   {Dimensions: types.DimLength, Scale: 1852, Name: "nmi"},
	"meter": {Dimensions: types.DimLength, Scale: 1, Name: "meter"},

	// Mass units (base: kilograms)
	"fg":  {Dimensions: types.DimMass, Scale: 1e-18, Name: "fg"},
	"pg":  {Dimensions: types.DimMass, Scale: 1e-15, Name: "pg"},
	"ng":  {Dimensions: types.DimMass, Scale: 1e-12, Name: "ng"},
	"ug":  {Dimensions: types.DimMass, Scale: 1e-9, Name: "ug"},
	"mg":  {Dimensions: types.DimMass, Scale: 1e-6, Name: "mg"},
	"g":   {Dimensions: types.DimMass, Scale: 1e-3, Name: "g"},
	"kg":  {Dimensions: types.DimMass, Scale: 1, Name: "kg"},
	"lb":  {Dimensions: types.DimMass, Scale: 0.453592, Name: "lb"},
	"oz":  {Dimensions: types.DimMass, Scale: 0.0283495, Name: "oz"},
	"ton": {Dimensions: types.DimMass, Scale: 1000, Name: "ton"},

	// Pressure units (base: Pascals)
	"pPa":  {Dimensions: types.DimPressure, Scale: 1e-12, Name: "pPa"},
	"nPa":  {Dimensions: types.DimPressure, Scale: 1e-9, Name: "nPa"},
	"uPa":  {Dimensions: types.DimPressure, Scale: 1e-6, Name: "uPa"},
	"mPa":  {Dimensions: types.DimPressure, Scale: 1e-3, Name: "mPa"},
	"Pa":   {Dimensions: types.DimPressure, Scale: 1, Name: "Pa"},
	"kPa":  {Dimensions: types.DimPressure, Scale: 1e3, Name: "kPa"},
	"MPa":  {Dimensions: types.DimPressure, Scale: 1e6, Name: "MPa"},
	"GPa":  {Dimensions: types.DimPressure, Scale: 1e9, Name: "GPa"},
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
	"fV": {Dimensions: types.DimVoltage, Scale: 1e-15, Name: "fV"},
	"pV": {Dimensions: types.DimVoltage, Scale: 1e-12, Name: "pV"},
	"nV": {Dimensions: types.DimVoltage, Scale: 1e-9, Name: "nV"},
	"uV": {Dimensions: types.DimVoltage, Scale: 1e-6, Name: "uV"},
	"mV": {Dimensions: types.DimVoltage, Scale: 1e-3, Name: "mV"},
	"V":  {Dimensions: types.DimVoltage, Scale: 1, Name: "V"},
	"kV": {Dimensions: types.DimVoltage, Scale: 1e3, Name: "kV"},
	"MV": {Dimensions: types.DimVoltage, Scale: 1e6, Name: "MV"},

	// Current units (base: Amperes)
	"fA": {Dimensions: types.DimCurrent, Scale: 1e-15, Name: "fA"},
	"pA": {Dimensions: types.DimCurrent, Scale: 1e-12, Name: "pA"},
	"nA": {Dimensions: types.DimCurrent, Scale: 1e-9, Name: "nA"},
	"uA": {Dimensions: types.DimCurrent, Scale: 1e-6, Name: "uA"},
	"mA": {Dimensions: types.DimCurrent, Scale: 1e-3, Name: "mA"},
	"A":  {Dimensions: types.DimCurrent, Scale: 1, Name: "A"},
	"kA": {Dimensions: types.DimCurrent, Scale: 1e3, Name: "kA"},
	"MA": {Dimensions: types.DimCurrent, Scale: 1e6, Name: "MA"},

	// Temperature units (base: Kelvin)
	// Only Kelvin is supported - it's the SI unit and has a true zero (absolute zero).
	// Celsius and Fahrenheit require affine conversions (offset + scale) which
	// are not yet supported. Users needing C/F should convert externally.
	"fK": {Dimensions: types.DimTemperature, Scale: 1e-15, Name: "fK"},
	"pK": {Dimensions: types.DimTemperature, Scale: 1e-12, Name: "pK"},
	"nK": {Dimensions: types.DimTemperature, Scale: 1e-9, Name: "nK"},
	"uK": {Dimensions: types.DimTemperature, Scale: 1e-6, Name: "uK"},
	"mK": {Dimensions: types.DimTemperature, Scale: 1e-3, Name: "mK"},
	"K":  {Dimensions: types.DimTemperature, Scale: 1, Name: "K"},
	"kK": {Dimensions: types.DimTemperature, Scale: 1e3, Name: "kK"},

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

// Resolve finds a unit by name. Returns the unit and true if found,
// or a zero Unit and false if not found.
func Resolve(name string) (*types.Unit, bool) {
	u, ok := registry[name]
	return &u, ok
}

func MustResolve(name string) *types.Unit {
	u, ok := Resolve(name)
	if !ok {
		panic(fmt.Sprintf("units: failed to resolve %s", name))
	}
	return u
}

// IsValid returns true if the given name is a valid unit.
func IsValid(name string) bool {
	_, ok := registry[name]
	return ok
}
