#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Console labels of the generated NI values."""

_LABELS = {
    "Cfg_Default": "Default",
    "Diff": "Differential",
    "PseudoDiff": "Pseudo-Differential",
    "RSE": "Referenced single ended",
    "NRSE": "Non-referenced single ended",
    "none": "None",
    "linear": "Linear",
    "map": "Map",
    "table": "Table",
    "FullBridge": "Full bridge",
    "HalfBridge": "Half bridge",
    "QuarterBridge": "Quarter bridge",
    "mVoltsPerVolt": "mV/V",
    "VoltsPerVolt": "V/V",
    "TwoPulse": "Two pulse",
    "PoundsPerSquareInch": "PSI",
    "NewtonMeters": "Newton meters",
    "InchOunces": "Inch ounces",
    "FootPounds": "Foot pounds",
    "KilogramForce": "Kilograms",
    "DegC": "Celsius",
    "DegF": "Fahrenheit",
    "Kelvins": "Kelvin",
    "DegR": "Rankine",
    "LowFreq1Ctr": "One counter (low frequency)",
    "HighFreq2Ctr": "Two counters (high frequency)",
    "LargeRng2Ctr": "Two counters (large range)",
    "DynamicAvg": "Dynamic averaging",
    "AHighBHigh": "A high B high",
    "AHighBLow": "A high B low",
    "ALowBHigh": "A low B high",
    "ALowBLow": "A low B low",
    "MetersPerSecond": "m/s",
    "InchesPerSecond": "in/s",
    "2Wire": "2-Wire",
    "3Wire": "3-Wire",
    "4Wire": "4-Wire",
    "CountUp": "Count up",
    "CountDown": "Count down",
    "ExternallyControlled": "Externally controlled",
    "MillivoltsPerMillimeterPerSecond": "mV/mm/s",
    "MilliVoltsPerInchPerSecond": "mV/in/s",
    "mVoltsPerG": "mV/g",
    "VoltsPerG": "V/g",
    "mVoltsPerNewton": "mV/N",
    "mVoltsPerPound": "mV/lb",
    "built_in": "Built in",
    "const_val": "Constant value",
    "chan": "Channel",
    "FullBridgeI": "Full bridge I",
    "FullBridgeII": "Full bridge II",
    "FullBridgeIII": "Full bridge III",
    "HalfBridgeI": "Half bridge I",
    "HalfBridgeII": "Half bridge II",
    "QuarterBridgeI": "Quarter bridge I",
}


def label(value: str) -> str:
    """Return the label a Console form shows for a generated value. A value with no
    entry is its own label."""
    return _LABELS.get(value, value)
