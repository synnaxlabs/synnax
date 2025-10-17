// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>

#include "driver/ni/daqmx/nidaqmx.h"

namespace channel {
inline const std::map<std::string, int32_t> UNITS_MAP = {
    {"Volts", DAQmx_Val_Volts},
    {"Amps", DAQmx_Val_Amps},
    {"DegF", DAQmx_Val_DegF},
    {"F", DAQmx_Val_DegF},
    {"DegC", DAQmx_Val_DegC},
    {"C", DAQmx_Val_DegC},
    {"Celsius", DAQmx_Val_DegC},
    {"Farenheit", DAQmx_Val_DegF},
    {"DegR", DAQmx_Val_DegR},
    {"Rankine", DAQmx_Val_DegR},
    {"Kelvins", DAQmx_Val_Kelvins},
    {"K", DAQmx_Val_Kelvins},
    {"Strain", DAQmx_Val_Strain},
    {"Ohms", DAQmx_Val_Ohms},
    {"Hz", DAQmx_Val_Hz},
    {"Ticks", DAQmx_Val_Ticks},
    {"Seconds", DAQmx_Val_Seconds},
    {"FromCustomScale", DAQmx_Val_FromCustomScale},
    {"Meters", DAQmx_Val_Meters},
    {"Inches", DAQmx_Val_Inches},
    {"Degrees", DAQmx_Val_Degrees},
    {"Radians", DAQmx_Val_Radians},
    {"g", DAQmx_Val_g},
    {"MetersPerSecondSquared", DAQmx_Val_MetersPerSecondSquared},
    {"MetersPerSecond", DAQmx_Val_MetersPerSecond},
    {"m/s", DAQmx_Val_MetersPerSecond},
    {"InchesPerSecond", DAQmx_Val_InchesPerSecond},
    {"mV/m/s", DAQmx_Val_MillivoltsPerMillimeterPerSecond},
    {"MillivoltsPerMillimeterPerSecond", DAQmx_Val_MillivoltsPerMillimeterPerSecond},
    {"MilliVoltsPerInchPerSecond", DAQmx_Val_MilliVoltsPerInchPerSecond},
    {"mVoltsPerNewton", DAQmx_Val_mVoltsPerNewton},
    {"mVoltsPerPound", DAQmx_Val_mVoltsPerPound},
    {"Newtons", DAQmx_Val_Newtons},
    {"Pounds", DAQmx_Val_Pounds},
    {"KilogramForce", DAQmx_Val_KilogramForce},
    {"PoundsPerSquareInch", DAQmx_Val_PoundsPerSquareInch},
    {"Bar", DAQmx_Val_Bar},
    {"Pascals", DAQmx_Val_Pascals},
    {"VoltsPerVolt", DAQmx_Val_VoltsPerVolt},
    {"mVoltsPerVolt", DAQmx_Val_mVoltsPerVolt},
    {"NewtonMeters", DAQmx_Val_NewtonMeters},
    {"InchOunces", DAQmx_Val_InchOunces},
    {"InchPounds", DAQmx_Val_InchPounds},
    {"FootPounds", DAQmx_Val_FootPounds},
    {"Strain", DAQmx_Val_Strain},
    {"FromTEDS", DAQmx_Val_FromTEDS},
    {"VoltsPerG", DAQmx_Val_VoltsPerG},
    {"mVoltsPerG", DAQmx_Val_mVoltsPerG},
    {"AccelUnit_g", DAQmx_Val_AccelUnit_g}
};

int32_t inline parse_units(xjson::Parser &cfg, const std::string &path) {
    const auto str_units = cfg.optional<std::string>(path, "Volts");
    const auto units = UNITS_MAP.find(str_units);
    if (units == UNITS_MAP.end()) cfg.field_err(path, "invalid units: " + str_units);
    return units->second;
}
}
