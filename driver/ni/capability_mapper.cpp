// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <unordered_set>

#include "driver/ni/capability_mapper.h"

namespace ni { namespace capability {

std::vector<std::string>
convert_ai_types_to_synnax(const std::vector<int32> &daqmx_types) {
    std::vector<std::string> synnax_types;
    for (const auto type: daqmx_types) {
        switch (type) {
            case DAQmx_Val_Voltage: // 10322
                synnax_types.push_back("ai_voltage");
                break;
            case DAQmx_Val_Current: // 10134
                synnax_types.push_back("ai_current");
                break;
            case DAQmx_Val_Temp_TC: // 10303
                synnax_types.push_back("ai_thermocouple");
                break;
            case DAQmx_Val_Temp_RTD: // 10301
                synnax_types.push_back("ai_rtd");
                break;
            case DAQmx_Val_Accelerometer: // 10356
                synnax_types.push_back("ai_accel");
                break;
            case DAQmx_Val_Bridge: // 15908
                synnax_types.push_back("ai_bridge");
                break;
            case DAQmx_Val_Force_Bridge: // 15899
                // Could map to ai_force_bridge_table or ai_force_bridge_two_point_lin
                // Since device supports bridge-based force, expose both variants
                synnax_types.push_back("ai_force_bridge_table");
                synnax_types.push_back("ai_force_bridge_two_point_lin");
                break;
            case DAQmx_Val_Force_IEPESensor: // 15895
                synnax_types.push_back("ai_force_iepe");
                break;
            case DAQmx_Val_SoundPressure_Microphone: // 10354
                synnax_types.push_back("ai_microphone");
                break;
            case DAQmx_Val_Pressure_Bridge: // 15902
                // Similar to force bridge - expose both variants
                synnax_types.push_back("ai_pressure_bridge_table");
                break;
            case DAQmx_Val_Resistance: // 10278
                synnax_types.push_back("ai_resistance");
                break;
            case DAQmx_Val_Strain_Gage: // 10300
                synnax_types.push_back("ai_strain_gauge");
                break;
            case DAQmx_Val_Temp_BuiltInSensor: // 10311
                synnax_types.push_back("ai_temp_builtin");
                break;
            case DAQmx_Val_Torque_Bridge: // 15905
                // Expose both torque bridge variants
                synnax_types.push_back("ai_torque_bridge_table");
                synnax_types.push_back("ai_torque_bridge_two_point_lin");
                break;
            case DAQmx_Val_Velocity_IEPESensor: // 15966
                synnax_types.push_back("ai_velocity_iepe");
                break;
            // Note: Some AI types in Synnax may not have direct DAQmx equivalents
            // or may be specialized versions of bridge measurements
            default:
                // Unknown type - skip
                break;
        }
    }
    return synnax_types;
}

std::vector<std::string>
convert_ao_types_to_synnax(const std::vector<int32> &daqmx_types) {
    std::vector<std::string> synnax_types;
    for (const auto type: daqmx_types) {
        switch (type) {
            case DAQmx_Val_Voltage: // 10322
                synnax_types.push_back("ao_voltage");
                break;
            case DAQmx_Val_Current: // 10134
                synnax_types.push_back("ao_current");
                break;
            case DAQmx_Val_FuncGen: // 14750
                synnax_types.push_back("ao_func_gen");
                break;
            default:
                // Unknown type - skip
                break;
        }
    }
    return synnax_types;
}

std::vector<std::string>
convert_ci_types_to_synnax(const std::vector<int32> &daqmx_types) {
    std::vector<std::string> synnax_types;
    for (const auto type: daqmx_types) {
        switch (type) {
            case DAQmx_Val_Freq: // 10179
                synnax_types.push_back("ci_frequency");
                break;
            case DAQmx_Val_CountEdges: // 10125
                synnax_types.push_back("ci_edge_count");
                break;
            case DAQmx_Val_Period: // 10256
                synnax_types.push_back("ci_period");
                break;
            case DAQmx_Val_PulseWidth: // 10359
                synnax_types.push_back("ci_pulse_width");
                break;
            case DAQmx_Val_SemiPeriod: // 10289
                synnax_types.push_back("ci_semi_period");
                break;
            case DAQmx_Val_TwoEdgeSep: // 10267
                synnax_types.push_back("ci_two_edge_sep");
                break;
            case DAQmx_Val_Velocity_LinEncoder: // 16079
                synnax_types.push_back("ci_velocity_linear");
                break;
            case DAQmx_Val_Velocity_AngEncoder: // 16078
                synnax_types.push_back("ci_velocity_angular");
                break;
            case DAQmx_Val_Position_LinEncoder: // 10361
                synnax_types.push_back("ci_position_linear");
                break;
            case DAQmx_Val_Position_AngEncoder: // 10360
                synnax_types.push_back("ci_position_angular");
                break;
            case DAQmx_Val_DutyCycle: // 16070
                synnax_types.push_back("ci_duty_cycle");
                break;
            default:
                // Unknown type - skip
                break;
        }
    }
    return synnax_types;
}

std::vector<std::string>
convert_co_types_to_synnax(const std::vector<int32> &daqmx_types) {
    // Check if device supports ANY pulse output type
    // DAQmx has 3 types: Time, Freq, Ticks - Synnax uses Time-based
    std::unordered_set<int32> pulse_types = {
        DAQmx_Val_Pulse_Time, // 10269
        DAQmx_Val_Pulse_Freq, // 10119
        DAQmx_Val_Pulse_Ticks // 10268
    };

    for (const auto type: daqmx_types) {
        if (pulse_types.count(type) > 0) {
            // If device supports any pulse output, it supports co_pulse_output
            return {"co_pulse_output"};
        }
    }

    return {}; // No supported types
}

} // namespace capability
} // namespace ni
