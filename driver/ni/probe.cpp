// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <charconv>
#include <cstdint>
#include <map>
#include <optional>
#include <ranges>
#include <string>
#include <system_error>
#include <unordered_map>
#include <utility>
#include <vector>

#include "absl/log/log.h"

#include "driver/ni/daqmx/nidaqmx.h"
#include "driver/ni/probe.h"

namespace driver::ni {
namespace {
using ::synnax::ni::AnalogInputPort;
using ::synnax::ni::AnalogOutputPort;
using ::synnax::ni::CounterPort;
using ::synnax::ni::DigitalPort;
using ::synnax::ni::Hardware;
using ::x::spatial::Bounds;

const std::unordered_map<int32, std::string> PRODUCT_CATEGORIES = {
    {DAQmx_Val_MSeriesDAQ, "m_series_daq"},
    {DAQmx_Val_XSeriesDAQ, "x_series_daq"},
    {DAQmx_Val_ESeriesDAQ, "e_series_daq"},
    {DAQmx_Val_SSeriesDAQ, "s_series_daq"},
    {DAQmx_Val_BSeriesDAQ, "b_series_daq"},
    {DAQmx_Val_SCSeriesDAQ, "sc_series_daq"},
    {DAQmx_Val_USBDAQ, "usb_daq"},
    {DAQmx_Val_AOSeries, "ao_series"},
    {DAQmx_Val_DigitalIO, "digital_io"},
    {DAQmx_Val_TIOSeries, "tio_series"},
    {DAQmx_Val_DynamicSignalAcquisition, "dynamic_signal_acquisition"},
    {DAQmx_Val_Switches, "switches"},
    {DAQmx_Val_CompactDAQChassis, "compact_daq_chassis"},
    {DAQmx_Val_CompactRIOChassis, "compact_rio_chassis"},
    {DAQmx_Val_CSeriesModule, "c_series_module"},
    {DAQmx_Val_SCXIModule, "scxi_module"},
    {DAQmx_Val_SCCConnectorBlock, "scc_connector_block"},
    {DAQmx_Val_SCCModule, "scc_module"},
    {DAQmx_Val_NIELVIS, "ni_elvis"},
    {DAQmx_Val_NetworkDAQ, "network_daq"},
    {DAQmx_Val_SCExpress, "sc_express"},
    {DAQmx_Val_FieldDAQ, "field_daq"},
};

/// @brief the channel types of each DAQmx measurement or output type. A DAQmx type with
/// no entry has no channel type, so no form can offer it.
using TypeTable = std::unordered_map<int32, std::vector<std::string>>;

const TypeTable AI_TYPES = {
    {DAQmx_Val_Voltage, {"ai_voltage"}},
    {DAQmx_Val_VoltageRMS, {"ai_voltage_rms"}},
    {DAQmx_Val_Current, {"ai_current"}},
    {DAQmx_Val_CurrentRMS, {"ai_current_rms"}},
    {DAQmx_Val_Voltage_CustomWithExcitation, {"ai_voltage_with_excit"}},
    {DAQmx_Val_Bridge, {"ai_bridge"}},
    {DAQmx_Val_Freq_Voltage, {"ai_freq_voltage"}},
    {DAQmx_Val_Resistance, {"ai_resistance"}},
    {DAQmx_Val_Temp_TC, {"ai_thermocouple"}},
    {DAQmx_Val_Temp_Thrmstr, {"ai_thermistor_iex", "ai_thermistor_vex"}},
    {DAQmx_Val_Temp_RTD, {"ai_rtd"}},
    {DAQmx_Val_Temp_BuiltInSensor, {"ai_temp_builtin"}},
    {DAQmx_Val_Strain_Gage, {"ai_strain_gauge"}},
    {DAQmx_Val_Accelerometer, {"ai_accel"}},
    {DAQmx_Val_Acceleration_Charge, {"ai_accel_charge"}},
    {DAQmx_Val_Acceleration_4WireDCVoltage, {"ai_accel_4_wire_dc_voltage"}},
    {DAQmx_Val_Velocity_IEPESensor, {"ai_velocity_iepe"}},
    {DAQmx_Val_Force_Bridge,
     {"ai_force_bridge_polynomial",
      "ai_force_bridge_table",
      "ai_force_bridge_two_point_lin"}},
    {DAQmx_Val_Force_IEPESensor, {"ai_force_iepe"}},
    {DAQmx_Val_Pressure_Bridge,
     {"ai_pressure_bridge_polynomial",
      "ai_pressure_bridge_table",
      "ai_pressure_bridge_two_point_lin"}},
    {DAQmx_Val_SoundPressure_Microphone, {"ai_microphone"}},
    {DAQmx_Val_Torque_Bridge,
     {"ai_torque_bridge_polynomial",
      "ai_torque_bridge_table",
      "ai_torque_bridge_two_point_lin"}},
    {DAQmx_Val_Charge, {"ai_charge"}},
};

const TypeTable AO_TYPES = {
    {DAQmx_Val_Voltage, {"ao_voltage"}},
    {DAQmx_Val_Current, {"ao_current"}},
    {DAQmx_Val_FuncGen, {"ao_func_gen"}},
};

const TypeTable CI_TYPES = {
    {DAQmx_Val_CountEdges, {"ci_edge_count"}},
    {DAQmx_Val_Freq, {"ci_frequency"}},
    {DAQmx_Val_Period, {"ci_period"}},
    {DAQmx_Val_PulseWidth, {"ci_pulse_width"}},
    {DAQmx_Val_SemiPeriod, {"ci_semi_period"}},
    {DAQmx_Val_DutyCycle, {"ci_duty_cycle"}},
    {DAQmx_Val_TwoEdgeSep, {"ci_two_edge_sep"}},
    {DAQmx_Val_Position_AngEncoder, {"ci_position_angular"}},
    {DAQmx_Val_Position_LinEncoder, {"ci_position_linear"}},
    {DAQmx_Val_Velocity_AngEncoder, {"ci_velocity_angular"}},
    {DAQmx_Val_Velocity_LinEncoder, {"ci_velocity_linear"}},
};

/// @brief the bits of DAQmx_PhysicalChan_AI_TermCfgs and
/// DAQmx_PhysicalChan_AO_TermCfgs.
const std::vector<std::pair<int32, std::string>> TERMINAL_CONFIG_BITS = {
    {DAQmx_Val_Bit_TermCfg_RSE, ::synnax::ni::TERMINAL_CONFIG_RSE},
    {DAQmx_Val_Bit_TermCfg_NRSE, ::synnax::ni::TERMINAL_CONFIG_NRSE},
    {DAQmx_Val_Bit_TermCfg_Diff, ::synnax::ni::TERMINAL_CONFIG_DIFF},
    {DAQmx_Val_Bit_TermCfg_PseudoDIFF, ::synnax::ni::TERMINAL_CONFIG_PSEUDO_DIFF},
};

/// @brief the values of DAQmx_AI_TermCfg.
const std::unordered_map<int32, std::string> TERMINAL_CONFIGS = {
    {DAQmx_Val_RSE, ::synnax::ni::TERMINAL_CONFIG_RSE},
    {DAQmx_Val_NRSE, ::synnax::ni::TERMINAL_CONFIG_NRSE},
    {DAQmx_Val_Diff, ::synnax::ni::TERMINAL_CONFIG_DIFF},
    {DAQmx_Val_PseudoDiff, ::synnax::ni::TERMINAL_CONFIG_PSEUDO_DIFF},
};

/// @brief a family of devices in which a differential channel on port n uses port n +
/// offset as its negative input, for each n where n % (2 * offset) < offset. No DAQmx
/// attribute names the negative port.
struct Pairing {
    /// @brief the DAQmx product categories of the family.
    std::vector<int32> categories;
    /// @brief the DAQmx product types of the family, without the "NI " prefix.
    std::vector<std::string> types;
    /// @brief the distance from a positive port to its negative port.
    std::int32_t offset = 0;
};

const std::vector<Pairing> PAIRINGS = {
    {
        .categories =
            {DAQmx_Val_ESeriesDAQ, DAQmx_Val_MSeriesDAQ, DAQmx_Val_XSeriesDAQ},
        .types = {"9204", "9205", "9206"},
        .offset = 8,
    },
    {
        .types = {"USB-6001", "USB-6002", "USB-6003", "USB-6008", "USB-6009"},
        .offset = 4,
    },
};

/// @brief the range of the task that reads back a default terminal configuration, for a
/// device that lists no voltage ranges.
const Bounds FALLBACK_VOLTAGE_RANGE = {.lower = -10, .upper = 10};
/// @brief like FALLBACK_VOLTAGE_RANGE, for a device that lists no current ranges.
const Bounds FALLBACK_CURRENT_RANGE = {.lower = -0.02, .upper = 0.02};
/// @brief the shunt resistance of a current readback channel, in ohms. DAQmx uses it
/// only when the default shunt of the port is external.
const float64 EXTERNAL_SHUNT_RESISTANCE = 249;

const std::string PRODUCT_TYPE_PREFIX = "NI ";
const std::string AI_VOLTAGE = "ai_voltage";
const std::string AI_CURRENT = "ai_current";
const std::string RSE = ::synnax::ni::TERMINAL_CONFIG_RSE;
const std::string DIFF = ::synnax::ni::TERMINAL_CONFIG_DIFF;

/// @brief a physical channel or port and the number at the end of its name.
struct Port {
    std::string name;
    std::int32_t number = 0;
};

template<typename T>
bool contains(const std::vector<T> &values, const T &value) {
    return std::ranges::find(values, value) != values.end();
}

/// @brief splits a DAQmx list such as "Dev1/ai0, Dev1/ai1" into its names.
std::vector<std::string> split(const std::string &list) {
    std::vector<std::string> names;
    std::size_t start = 0;
    while (start < list.size()) {
        auto end = list.find(',', start);
        if (end == std::string::npos) end = list.size();
        if (const auto first = list.find_first_not_of(' ', start); first < end) {
            const auto last = list.find_last_not_of(' ', end - 1);
            names.push_back(list.substr(first, last - first + 1));
        }
        start = end + 1;
    }
    return names;
}

/// @brief returns the number at the end of a name, such as 3 for Dev1/ai3.
std::optional<std::int32_t> number_of(const std::string &name) {
    const auto digits = name.find_last_not_of("0123456789");
    const auto start = digits == std::string::npos ? 0 : digits + 1;
    if (start == name.size()) return std::nullopt;
    std::int32_t number = 0;
    const char *end = name.data() + name.size();
    const auto [ptr, ec] = std::from_chars(name.data() + start, end, number);
    if (ec != std::errc{} || ptr != end) return std::nullopt;
    return number;
}

/// @brief reads a list attribute of the device. A device without the attribute has an
/// empty list.
std::pair<std::vector<std::string>, x::errors::Error>
read_list(daqmx::SugaredAPI &dmx, const std::string &device, const int32 attribute) {
    auto [list, err] = dmx.GetDeviceAttributeString(device.c_str(), attribute);
    if (err.matches(daqmx::ATTRIBUTE_NOT_SUPPORTED)) return {{}, x::errors::NIL};
    if (err) return {{}, err};
    return {split(list), x::errors::NIL};
}

std::pair<std::vector<Port>, x::errors::Error>
read_ports(daqmx::SugaredAPI &dmx, const std::string &device, const int32 attribute) {
    auto [names, err] = read_list(dmx, device, attribute);
    std::vector<Port> ports;
    for (auto &name: names) {
        const auto number = number_of(name);
        if (!number.has_value()) {
            VLOG(1) << "[ni] skipping physical channel with no number: " << name;
            continue;
        }
        ports.push_back({.name = std::move(name), .number = *number});
    }
    return {std::move(ports), err};
}

x::errors::Error read_types(
    daqmx::SugaredAPI &dmx,
    const Port &port,
    const int32 attribute,
    const TypeTable &table,
    std::vector<std::string> &types
) {
    auto [codes, err] = dmx.GetPhysicalChanAttributeInt32Array(
        port.name.c_str(),
        attribute
    );
    if (err.matches(daqmx::ATTRIBUTE_NOT_SUPPORTED)) return x::errors::NIL;
    if (err) return err;
    for (const auto code: codes)
        if (const auto it = table.find(code); it != table.end())
            types.insert(types.end(), it->second.begin(), it->second.end());
    return x::errors::NIL;
}

x::errors::Error read_terminal_configs(
    daqmx::SugaredAPI &dmx,
    const Port &port,
    const int32 attribute,
    std::vector<std::string> &configs
) {
    int32 bits = 0;
    const auto err = dmx.GetPhysicalChanAttributeInt32(
        port.name.c_str(),
        attribute,
        &bits
    );
    if (err.matches(daqmx::ATTRIBUTE_NOT_SUPPORTED)) return x::errors::NIL;
    if (err) return err;
    for (const auto &[bit, config]: TERMINAL_CONFIG_BITS)
        if ((bits & bit) != 0) configs.push_back(config);
    return x::errors::NIL;
}

x::errors::Error read_rate(
    daqmx::SugaredAPI &dmx,
    const std::string &device,
    const int32 attribute,
    std::optional<double> &rate
) {
    float64 value = 0;
    const auto err = dmx.GetDeviceAttributeDouble(device.c_str(), attribute, &value);
    if (err.matches(daqmx::ATTRIBUTE_NOT_SUPPORTED)) return x::errors::NIL;
    if (err) return err;
    rate = value;
    return x::errors::NIL;
}

x::errors::Error read_ranges(
    daqmx::SugaredAPI &dmx,
    const std::string &device,
    const int32 attribute,
    std::vector<Bounds> &ranges
) {
    auto [values, err] = dmx.GetDeviceAttributeDoubleArray(device.c_str(), attribute);
    if (err.matches(daqmx::ATTRIBUTE_NOT_SUPPORTED)) return x::errors::NIL;
    if (err) return err;
    for (std::size_t i = 0; i + 1 < values.size(); i += 2)
        ranges.push_back({.lower = values[i], .upper = values[i + 1]});
    return x::errors::NIL;
}

/// @brief whether probe reads the default terminal configuration of the port back from
/// a task: the port accepts more than one, and takes a voltage or current channel.
bool reads_back_default(const AnalogInputPort &ai) {
    return ai.terminal_configs.size() > 1 &&
           (contains(ai.types, AI_VOLTAGE) || contains(ai.types, AI_CURRENT));
}

/// @brief returns the narrowest of the ranges, which is valid in every terminal
/// configuration.
Bounds narrowest(const std::vector<Bounds> &ranges, const Bounds &fallback) {
    if (ranges.empty()) return fallback;
    return std::ranges::min(ranges, {}, [](const Bounds &r) {
        return r.upper - r.lower;
    });
}

/// @brief adds a channel with Cfg_Default on the port to the task: a voltage channel
/// when the port takes one, else a current channel.
x::errors::Error add_readback_channel(
    daqmx::SugaredAPI &dmx,
    TaskHandle task,
    const Port &port,
    const AnalogInputPort &ai,
    const Hardware &hw
) {
    if (contains(ai.types, AI_VOLTAGE)) {
        const auto range = narrowest(hw.ai_voltage_ranges, FALLBACK_VOLTAGE_RANGE);
        return dmx.CreateAIVoltageChan(
            task,
            port.name.c_str(),
            "",
            DAQmx_Val_Cfg_Default,
            range.lower,
            range.upper,
            DAQmx_Val_Volts,
            nullptr
        );
    }
    const auto range = narrowest(hw.ai_current_ranges, FALLBACK_CURRENT_RANGE);
    return dmx.CreateAICurrentChan(
        task,
        port.name.c_str(),
        "",
        DAQmx_Val_Cfg_Default,
        range.lower,
        range.upper,
        DAQmx_Val_Amps,
        DAQmx_Val_Default,
        EXTERNAL_SHUNT_RESISTANCE,
        nullptr
    );
}

/// @brief reads back what Cfg_Default gives on a port where reads_back_default holds. A
/// failure leaves the default unknown and does not fail the probe.
std::optional<std::string> read_default_terminal_config(
    daqmx::SugaredAPI &dmx,
    const Port &port,
    const AnalogInputPort &ai,
    const Hardware &hw
) {
    // The scanner warns about the missing default, so these failures log at VLOG(1).
    TaskHandle task = nullptr;
    if (const auto err = dmx.CreateTask("", &task)) {
        VLOG(1) << "[ni] failed to create a task to read the default terminal "
                << "configuration of " << port.name << ": " << err;
        return std::nullopt;
    }
    int32 config = 0;
    auto err = add_readback_channel(dmx, task, port, ai, hw);
    if (!err)
        err = dmx.GetChanAttributeInt32(
            task,
            port.name.c_str(),
            DAQmx_AI_TermCfg,
            &config
        );
    if (const auto clear_err = dmx.ClearTask(task))
        LOG(WARNING) << "[ni] failed to clear the task that read the default terminal "
                     << "configuration of " << port.name << ": " << clear_err;
    if (err) {
        VLOG(1) << "[ni] failed to read the default terminal configuration of "
                << port.name << ": " << err;
        return std::nullopt;
    }
    const auto it = TERMINAL_CONFIGS.find(config);
    if (it == TERMINAL_CONFIGS.end()) return std::nullopt;
    return it->second;
}

x::errors::Error
read_analog_inputs(daqmx::SugaredAPI &dmx, const std::string &device, Hardware &hw) {
    auto [ports, err] = read_ports(dmx, device, DAQmx_Dev_AI_PhysicalChans);
    if (err || ports.empty()) return err;
    for (const auto &[attribute, rate]: {
             std::pair{DAQmx_Dev_AI_MinRate, &hw.ai_min_rate},
             std::pair{DAQmx_Dev_AI_MaxSingleChanRate, &hw.ai_max_single_channel_rate},
             std::pair{DAQmx_Dev_AI_MaxMultiChanRate, &hw.ai_max_multi_channel_rate},
         })
        if (const auto rate_err = read_rate(dmx, device, attribute, *rate))
            return rate_err;
    for (const auto &[attribute, ranges]: {
             std::pair{DAQmx_Dev_AI_VoltageRngs, &hw.ai_voltage_ranges},
             std::pair{DAQmx_Dev_AI_CurrentRngs, &hw.ai_current_ranges},
         })
        if (const auto range_err = read_ranges(dmx, device, attribute, *ranges))
            return range_err;
    for (const auto &port: ports) {
        AnalogInputPort ai{.port = port.number};
        if (const auto cfg_err = read_terminal_configs(
                dmx,
                port,
                DAQmx_PhysicalChan_AI_TermCfgs,
                ai.terminal_configs
            ))
            return cfg_err;
        if (const auto type_err = read_types(
                dmx,
                port,
                DAQmx_PhysicalChan_AI_SupportedMeasTypes,
                AI_TYPES,
                ai.types
            ))
            return type_err;
        if (reads_back_default(ai))
            ai.default_terminal_config = read_default_terminal_config(
                dmx,
                port,
                ai,
                hw
            );
        else if (ai.terminal_configs.size() == 1)
            ai.default_terminal_config = ai.terminal_configs.front();
        hw.analog_inputs.push_back(std::move(ai));
    }
    return x::errors::NIL;
}

x::errors::Error
read_analog_outputs(daqmx::SugaredAPI &dmx, const std::string &device, Hardware &hw) {
    auto [ports, err] = read_ports(dmx, device, DAQmx_Dev_AO_PhysicalChans);
    if (err || ports.empty()) return err;
    if (const auto
            rate_err = read_rate(dmx, device, DAQmx_Dev_AO_MaxRate, hw.ao_max_rate))
        return rate_err;
    for (const auto &[attribute, ranges]: {
             std::pair{DAQmx_Dev_AO_VoltageRngs, &hw.ao_voltage_ranges},
             std::pair{DAQmx_Dev_AO_CurrentRngs, &hw.ao_current_ranges},
         })
        if (const auto range_err = read_ranges(dmx, device, attribute, *ranges))
            return range_err;
    for (const auto &port: ports) {
        AnalogOutputPort ao{.port = port.number};
        if (const auto cfg_err = read_terminal_configs(
                dmx,
                port,
                DAQmx_PhysicalChan_AO_TermCfgs,
                ao.terminal_configs
            ))
            return cfg_err;
        if (const auto type_err = read_types(
                dmx,
                port,
                DAQmx_PhysicalChan_AO_SupportedOutputTypes,
                AO_TYPES,
                ao.types
            ))
            return type_err;
        hw.analog_outputs.push_back(std::move(ao));
    }
    return x::errors::NIL;
}

x::errors::Error
read_counters(daqmx::SugaredAPI &dmx, const std::string &device, Hardware &hw) {
    auto [inputs, in_err] = read_ports(dmx, device, DAQmx_Dev_CI_PhysicalChans);
    if (in_err) return in_err;
    for (const auto &port: inputs) {
        CounterPort ci{.port = port.number};
        if (const auto type_err = read_types(
                dmx,
                port,
                DAQmx_PhysicalChan_CI_SupportedMeasTypes,
                CI_TYPES,
                ci.types
            ))
            return type_err;
        hw.counter_inputs.push_back(std::move(ci));
    }
    // No channel type writes counter output yet, so output types are not read.
    auto [outputs, out_err] = read_ports(dmx, device, DAQmx_Dev_CO_PhysicalChans);
    if (out_err) return out_err;
    for (const auto &port: outputs)
        hw.counter_outputs.push_back(CounterPort{.port = port.number});
    return x::errors::NIL;
}

x::errors::Error
read_digital_ports(daqmx::SugaredAPI &dmx, const std::string &device, Hardware &hw) {
    std::map<std::int32_t, DigitalPort> by_number;
    const auto read = [&](const int32 attribute,
                          const int32 width_attribute,
                          const bool input) -> x::errors::Error {
        auto [ports, err] = read_ports(dmx, device, attribute);
        if (err) return err;
        for (const auto &port: ports) {
            uInt32 lines = 0;
            if (const auto width_err = dmx.GetPhysicalChanAttributeUInt32(
                    port.name.c_str(),
                    width_attribute,
                    &lines
                );
                width_err && !width_err.matches(daqmx::ATTRIBUTE_NOT_SUPPORTED))
                return width_err;
            auto &digital = by_number[port.number];
            digital.port = port.number;
            digital.lines = std::max(digital.lines, static_cast<std::int32_t>(lines));
            if (input)
                digital.input = true;
            else
                digital.output = true;
        }
        return x::errors::NIL;
    };
    auto err = read(DAQmx_Dev_DI_Ports, DAQmx_PhysicalChan_DI_PortWidth, true);
    if (!err) err = read(DAQmx_Dev_DO_Ports, DAQmx_PhysicalChan_DO_PortWidth, false);
    if (err) return err;
    for (const auto &digital: by_number | std::views::values)
        hw.digital_ports.push_back(digital);
    return x::errors::NIL;
}

/// @brief sets the negative port of each positive port, for a device in a family of
/// PAIRINGS. A port pairs only when it accepts both RSE and Diff, which leaves out
/// devices with a separate negative pin for each input.
void pair_ports(Hardware &hw, const int32 category) {
    auto type = hw.product_type;
    if (type.starts_with(PRODUCT_TYPE_PREFIX))
        type = type.substr(PRODUCT_TYPE_PREFIX.size());
    const auto pairing = std::ranges::find_if(PAIRINGS, [&](const Pairing &p) {
        return contains(p.categories, category) || contains(p.types, type);
    });
    if (pairing == PAIRINGS.end()) return;
    const auto offset = pairing->offset;
    for (auto &ai: hw.analog_inputs) {
        if (ai.port % (2 * offset) >= offset) continue;
        if (!contains(ai.terminal_configs, RSE) || !contains(ai.terminal_configs, DIFF))
            continue;
        const auto negative = ai.port + offset;
        if (std::ranges::any_of(hw.analog_inputs, [&](const AnalogInputPort &other) {
                return other.port == negative;
            }))
            ai.negative = negative;
    }
}
}

bool is_complete(const Hardware &hw) {
    return std::ranges::none_of(hw.analog_inputs, [](const AnalogInputPort &ai) {
        return reads_back_default(ai) && !ai.default_terminal_config.has_value();
    });
}

std::pair<Hardware, x::errors::Error>
probe(daqmx::SugaredAPI &dmx, const std::string &device) {
    Hardware hw;
    int32 category = DAQmx_Val_Unknown;
    if (const auto err = dmx.GetDeviceAttributeInt32(
            device.c_str(),
            DAQmx_Dev_ProductCategory,
            &category
        );
        err && !err.matches(daqmx::ATTRIBUTE_NOT_SUPPORTED))
        return {std::move(hw), err};
    const auto category_name = PRODUCT_CATEGORIES.find(category);
    hw.product_category = category_name == PRODUCT_CATEGORIES.end()
                            ? "unknown"
                            : category_name->second;
    auto [type, type_err] = dmx.GetDeviceAttributeString(
        device.c_str(),
        DAQmx_Dev_ProductType
    );
    if (type_err && !type_err.matches(daqmx::ATTRIBUTE_NOT_SUPPORTED))
        return {std::move(hw), type_err};
    hw.product_type = std::move(type);
    if (const auto err = read_analog_inputs(dmx, device, hw))
        return {std::move(hw), err};
    if (const auto err = read_analog_outputs(dmx, device, hw))
        return {std::move(hw), err};
    if (const auto err = read_counters(dmx, device, hw)) return {std::move(hw), err};
    if (const auto err = read_digital_ports(dmx, device, hw))
        return {std::move(hw), err};
    auto [terminals, terminals_err] = read_list(dmx, device, DAQmx_Dev_Terminals);
    if (terminals_err) return {std::move(hw), terminals_err};
    const auto prefix = "/" + device + "/";
    for (auto &terminal: terminals)
        hw.terminals.push_back(
            terminal.starts_with(prefix) ? terminal.substr(prefix.size())
                                         : std::move(terminal)
        );
    pair_ports(hw, category);
    return {std::move(hw), x::errors::NIL};
}
}
