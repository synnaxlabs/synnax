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
#include <vector>

#include "driver/ni/daqmx/nidaqmx.h"

namespace ni { namespace capability {

/// @brief Converts DAQmx AI measurement type constants to Synnax channel type strings.
/// @param daqmx_types Vector of DAQmx measurement type constants.
/// @return Vector of Synnax channel type strings (e.g., "ai_voltage", "ai_current").
std::vector<std::string>
convert_ai_types_to_synnax(const std::vector<int32> &daqmx_types);

/// @brief Converts DAQmx AO output type constants to Synnax channel type strings.
/// @param daqmx_types Vector of DAQmx output type constants.
/// @return Vector of Synnax channel type strings (e.g., "ao_voltage", "ao_current").
std::vector<std::string>
convert_ao_types_to_synnax(const std::vector<int32> &daqmx_types);

/// @brief Converts DAQmx CI measurement type constants to Synnax channel type strings.
/// @param daqmx_types Vector of DAQmx measurement type constants.
/// @return Vector of Synnax channel type strings (e.g., "ci_frequency", "ci_period").
std::vector<std::string>
convert_ci_types_to_synnax(const std::vector<int32> &daqmx_types);

/// @brief Converts DAQmx CO output type constants to Synnax channel type strings.
/// @param daqmx_types Vector of DAQmx output type constants.
/// @return Vector of Synnax channel type strings (e.g., "co_pulse_output").
/// @note DAQmx has 3 pulse types (Time, Freq, Ticks) but Synnax only uses time-based,
///       so all three map to "co_pulse_output".
std::vector<std::string>
convert_co_types_to_synnax(const std::vector<int32> &daqmx_types);

} // namespace capability
} // namespace ni
