// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <map>
#include <string>

#include "client/cpp/ni/json.gen.h"
#include "client/cpp/ni/types.gen.h"
#include "client/cpp/synnax.h"
#include "x/cpp/json/json.h"

#include "driver/ni/channel/scale.h"
#include "driver/ni/channel/units.h"
#include "driver/ni/daqmx/sugared.h"

namespace driver::ni::channel {
static int32_t parse_terminal_config(const std::string &s) {
    if (s == "PseudoDiff") return DAQmx_Val_PseudoDiff;
    if (s == "Diff") return DAQmx_Val_Diff;
    if (s == "NRSE") return DAQmx_Val_NRSE;
    if (s == "RSE") return DAQmx_Val_RSE;
    return DAQmx_Val_Cfg_Default;
}

static int32_t parse_bridge_config(const std::string &s) {
    if (s == "FullBridge") return DAQmx_Val_FullBridge;
    if (s == "HalfBridge") return DAQmx_Val_HalfBridge;
    if (s == "QuarterBridge") return DAQmx_Val_QuarterBridge;
    return DAQmx_Val_FullBridge;
}

static int32_t parse_resistance_config(const std::string &s) {
    if (s == "2Wire") return DAQmx_Val_2Wire;
    if (s == "3Wire") return DAQmx_Val_3Wire;
    if (s == "4Wire") return DAQmx_Val_4Wire;
    return DAQmx_Val_2Wire;
}

static int32_t get_excitation_src(const std::string &s) {
    if (s == "Internal") return DAQmx_Val_Internal;
    if (s == "External") return DAQmx_Val_External;
    if (s == "None") return DAQmx_Val_None;
    return DAQmx_Val_None;
}

static int32_t get_strain_config(const std::string &s) {
    if (s == "FullBridgeI") return DAQmx_Val_FullBridgeI;
    if (s == "FullBridgeII") return DAQmx_Val_FullBridgeII;
    if (s == "FullBridgeIII") return DAQmx_Val_FullBridgeIII;
    if (s == "HalfBridgeI") return DAQmx_Val_HalfBridgeI;
    if (s == "HalfBridgeII") return DAQmx_Val_HalfBridgeII;
    if (s == "QuarterBridgeI") return DAQmx_Val_QuarterBridgeI;
    if (s == "QuarterBridgeII") return DAQmx_Val_QuarterBridgeII;
    return DAQmx_Val_FullBridgeI;
}

static int32_t get_rosette_type(const std::string &s) {
    if (s == "RectangularRosette") return DAQmx_Val_RectangularRosette;
    if (s == "DeltaRosette") return DAQmx_Val_DeltaRosette;
    if (s == "TeeRosette") return DAQmx_Val_TeeRosette;
    return DAQmx_Val_RectangularRosette;
}

static int32_t get_rosette_meas_type(const std::string &s) {
    if (s == "PrincipalStrain1") return DAQmx_Val_PrincipalStrain1;
    if (s == "PrincipalStrain2") return DAQmx_Val_PrincipalStrain2;
    if (s == "PrincipalStrainAngle") return DAQmx_Val_PrincipalStrainAngle;
    if (s == "CartesianStrainX") return DAQmx_Val_CartesianStrainX;
    if (s == "CartesianStrainY") return DAQmx_Val_CartesianStrainY;
    if (s == "CartesianShearStrainXY") return DAQmx_Val_CartesianShearStrainXY;
    if (s == "MaxShearStrain") return DAQmx_Val_MaxShearStrain;
    if (s == "MaxShearStrainAngle") return DAQmx_Val_MaxShearStrainAngle;
    return DAQmx_Val_PrincipalStrain1;
}

static int32_t get_ci_edge(const std::string &s) {
    if (s == "Rising") return DAQmx_Val_Rising;
    if (s == "Falling") return DAQmx_Val_Falling;
    return DAQmx_Val_Rising;
}

static int32_t get_ci_meas_method(const std::string &s) {
    if (s == "LowFreq1Ctr") return DAQmx_Val_LowFreq1Ctr;
    if (s == "HighFreq2Ctr") return DAQmx_Val_HighFreq2Ctr;
    if (s == "LargeRng2Ctr") return DAQmx_Val_LargeRng2Ctr;
    if (s == "DynamicAvg") return DAQmx_Val_DynAvg;
    return DAQmx_Val_LowFreq1Ctr;
}

static int32_t get_ci_count_direction(const std::string &s) {
    if (s == "CountUp") return DAQmx_Val_CountUp;
    if (s == "CountDown") return DAQmx_Val_CountDown;
    if (s == "ExternallyControlled") return DAQmx_Val_ExtControlled;
    return DAQmx_Val_CountUp;
}

static int32_t get_ci_decoding_type(const std::string &s) {
    if (s == "X1") return DAQmx_Val_X1;
    if (s == "X2") return DAQmx_Val_X2;
    if (s == "X4") return DAQmx_Val_X4;
    if (s == "TwoPulse") return DAQmx_Val_TwoPulseCounting;
    return DAQmx_Val_X4;
}

static int32_t get_ci_z_index_phase(const std::string &s) {
    if (s == "AHighBHigh") return DAQmx_Val_AHighBHigh;
    if (s == "AHighBLow") return DAQmx_Val_AHighBLow;
    if (s == "ALowBHigh") return DAQmx_Val_ALowBHigh;
    if (s == "ALowBLow") return DAQmx_Val_ALowBLow;
    return DAQmx_Val_AHighBHigh;
}

static int32_t get_shunt_resistor_loc(const std::string &loc) {
    if (loc == "External") return DAQmx_Val_External;
    if (loc == "Internal") return DAQmx_Val_Internal;
    return DAQmx_Val_Default;
}

static int32_t get_rtd_type(const std::string &type) {
    if (type == "Pt3750") return DAQmx_Val_Pt3750;
    if (type == "PT3851") return DAQmx_Val_Pt3851;
    if (type == "PT3911") return DAQmx_Val_Pt3911;
    if (type == "PT3916") return DAQmx_Val_Pt3916;
    if (type == "PT3920") return DAQmx_Val_Pt3920;
    if (type == "PT3928") return DAQmx_Val_Pt3928;
    if (type == "Custom") return DAQmx_Val_Custom;
    return DAQmx_Val_Pt3750;
}

struct ExcitationConfig {
    const int32_t source;
    const double val;
    const double min_val_for_excitation; // optional
    const double max_val_for_excitation; // optional
    const bool32 use_excit_for_scaling; // optional

    explicit ExcitationConfig(const ::synnax::ni::CurrentExcitation &e):
        source(get_excitation_src(e.current_excit_source)),
        val(e.current_excit_val),
        min_val_for_excitation(0),
        max_val_for_excitation(0),
        use_excit_for_scaling(0) {}

    ExcitationConfig(
        const ::synnax::ni::VoltageExcitation &e,
        const bool use_excit_for_scaling = false
    ):
        source(get_excitation_src(e.voltage_excit_source)),
        val(e.voltage_excit_val),
        min_val_for_excitation(0),
        max_val_for_excitation(0),
        use_excit_for_scaling(use_excit_for_scaling ? 1 : 0) {}
};

struct BridgeConfig {
    const int32_t ni_bridge_config;
    const int32_t voltage_excit_source;
    const double voltage_excit_val;
    const double nominal_bridge_resistance;

    BridgeConfig(
        const ::synnax::ni::Bridge &b,
        const ::synnax::ni::VoltageExcitation &e
    ):
        ni_bridge_config(parse_bridge_config(b.bridge_config)),
        voltage_excit_source(get_excitation_src(e.voltage_excit_source)),
        voltage_excit_val(e.voltage_excit_val),
        nominal_bridge_resistance(b.nominal_bridge_resistance) {}
};

struct PolynomialConfig {
    std::vector<double> forward_coeffs;
    std::vector<double> reverse_coeffs;
    int32_t electrical_units;
    int32_t physical_units;

    PolynomialConfig(
        const ::synnax::ni::BridgePolynomial &b,
        const std::string &physical_units_str
    ):
        forward_coeffs(b.forward_coeffs), reverse_coeffs(b.reverse_coeffs) {
        const auto eu = channel::UNITS_MAP.find(b.electrical_units);
        electrical_units = eu == channel::UNITS_MAP.end() ? DAQmx_Val_Volts
                                                          : eu->second;
        const auto pu = channel::UNITS_MAP.find(physical_units_str);
        physical_units = pu == channel::UNITS_MAP.end() ? DAQmx_Val_Volts : pu->second;
    }
};

struct TableConfig {
    std::vector<double> electrical_vals;
    std::vector<double> physical_vals;
    int32_t electrical_units;
    int32_t physical_units;

    TableConfig(const ::synnax::ni::Table &t, const std::string &physical_units_str):
        electrical_vals(t.electrical_vals), physical_vals(t.physical_vals) {
        electrical_units = units_or_volts(t.electrical_units);
        physical_units = units_or_volts(physical_units_str);
    }
};

struct TwoPointLinConfig {
    double first_electrical_val;
    double second_electrical_val;
    int32_t electrical_units;
    double first_physical_val;
    double second_physical_val;
    int32_t physical_units;

    TwoPointLinConfig() = default;

    TwoPointLinConfig(
        const ::synnax::ni::TwoPointLin &t,
        const std::string &physical_units_str
    ):
        first_electrical_val(t.first_electrical_val),
        second_electrical_val(t.second_electrical_val),
        electrical_units(units_or_volts(t.electrical_units)),
        first_physical_val(t.first_physical_val),
        second_physical_val(t.second_physical_val),
        physical_units(units_or_volts(physical_units_str)) {}
};

inline std::string format_cfg_path(const std::string &path) {
    auto formatted = path;
    std::replace(formatted.begin(), formatted.end(), '.', '_');
    if (formatted.empty()) return formatted;
    formatted.pop_back();
    return formatted;
}

inline std::string format_cjc_port(const std::string &path, const std::int32_t port) {
    const auto last_underscore = path.find_last_of('_');
    if (last_underscore == std::string::npos) return path;
    return path.substr(0, last_underscore) + "_" + std::to_string(port);
}

inline int32_t parse_wave_type(const std::string &type) {
    if (type == "Triangle") return DAQmx_Val_Triangle;
    if (type == "Square") return DAQmx_Val_Square;
    if (type == "Sawtooth") return DAQmx_Val_Sawtooth;
    return DAQmx_Val_Sine;
}

inline int32_t parse_thermocouple_type(const std::string &type) {
    if (type == "K") return DAQmx_Val_K_Type_TC;
    if (type == "N") return DAQmx_Val_N_Type_TC;
    if (type == "R") return DAQmx_Val_R_Type_TC;
    if (type == "S") return DAQmx_Val_S_Type_TC;
    if (type == "T") return DAQmx_Val_T_Type_TC;
    if (type == "B") return DAQmx_Val_B_Type_TC;
    if (type == "E") return DAQmx_Val_E_Type_TC;
    return DAQmx_Val_J_Type_TC;
}

/// @brief the DAQmx cold-junction arguments a thermocouple channel is created with.
struct CJCArgs {
    /// @brief the DAQmx source constant.
    int32_t source = DAQmx_Val_BuiltIn;
    /// @brief the reference temperature, used only when the source is a constant.
    double val = 0;
    /// @brief the reference channel's port, used only when the source is a channel.
    int32_t port = 0;
};

inline CJCArgs parse_cjc(const ::synnax::ni::CJC &cjc) {
    if (const auto *c = std::get_if<::synnax::ni::CJCConstVal>(&cjc))
        return {DAQmx_Val_ConstVal, c->val, 0};
    if (const auto *c = std::get_if<::synnax::ni::CJCChan>(&cjc))
        return {DAQmx_Val_Chan, 0, c->port};
    return {};
}

/// @brief per-channel context the owning task binds before configuration is
/// applied to the hardware.
struct ApplyContext {
    /// @brief the physical device location, e.g. "cDAQ1Mod1".
    std::string dev_loc;
    /// @brief the channel's path within the task config, used for error
    /// propagation and as the DAQmx channel name.
    std::string cfg_path;
};

inline x::errors::Error apply(
    const ::synnax::ni::AIAccelChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ai" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const ExcitationConfig excitation_config(
        static_cast<const ::synnax::ni::CurrentExcitation &>(ch)
    );
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const auto sensitivity = ch.sensitivity;
    const auto [sensitivity_units, sensitivity_units_err] = parse_units(
        ch.sensitivity_units
    );
    if (sensitivity_units_err) return sensitivity_units_err;
    const auto terminal_config = parse_terminal_config(ch.terminal_config);
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    return dmx->CreateAIAccelChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        terminal_config,
        min_val,
        max_val,
        units,
        sensitivity,
        sensitivity_units,
        excitation_config.source,
        excitation_config.val,
        scale_key
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AIAccel4WireDCVoltageChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ai" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const ExcitationConfig excitation_config = ExcitationConfig(
        ch,
        ch.use_excit_for_scaling
    );
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const auto sensitivity = ch.sensitivity;
    const auto [sensitivity_units, sensitivity_units_err] = parse_units(
        ch.sensitivity_units
    );
    if (sensitivity_units_err) return sensitivity_units_err;
    const auto terminal_config = parse_terminal_config(ch.terminal_config);
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    return dmx->CreateAIAccel4WireDCVoltageChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        terminal_config,
        min_val,
        max_val,
        units,
        sensitivity,
        sensitivity_units,
        excitation_config.source,
        excitation_config.val,
        excitation_config.use_excit_for_scaling,
        scale_key
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AIAccelChargeChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ai" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const auto sensitivity = ch.sensitivity;
    const auto [sensitivity_units, sensitivity_units_err] = parse_units(
        ch.sensitivity_units
    );
    if (sensitivity_units_err) return sensitivity_units_err;
    const auto terminal_config = DAQmx_Val_Cfg_Default;
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    return dmx->CreateAIAccelChargeChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        terminal_config,
        min_val,
        max_val,
        units,
        sensitivity,
        sensitivity_units,
        scale_key
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AIBridgeChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ai" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const BridgeConfig bridge_config(ch, ch);
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    return dmx->CreateAIBridgeChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        min_val,
        max_val,
        units,
        bridge_config.ni_bridge_config,
        bridge_config.voltage_excit_source,
        bridge_config.voltage_excit_val,
        bridge_config.nominal_bridge_resistance,
        scale_key
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AIChargeChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ai" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const auto terminal_config = parse_terminal_config(ch.terminal_config);
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    return dmx->CreateAIChargeChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        terminal_config,
        min_val,
        max_val,
        units,
        scale_key
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AICurrentChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ai" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const auto ext_shunt_resistor_val = ch.ext_shunt_resistor_val;
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const auto shunt_resistor_loc = get_shunt_resistor_loc(ch.shunt_resistor_loc);
    const auto terminal_config = parse_terminal_config(ch.terminal_config);
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    return dmx->CreateAICurrentChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        terminal_config,
        min_val,
        max_val,
        units,
        shunt_resistor_loc,
        ext_shunt_resistor_val,
        scale_key
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AICurrentRMSChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ai" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const auto ext_shunt_resistor_val = ch.ext_shunt_resistor_val;
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const auto shunt_resistor_loc = get_shunt_resistor_loc(ch.shunt_resistor_loc);
    const auto terminal_config = parse_terminal_config(ch.terminal_config);
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    return dmx->CreateAICurrentRMSChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        terminal_config,
        min_val,
        max_val,
        units,
        shunt_resistor_loc,
        ext_shunt_resistor_val,
        scale_key
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AIForceBridgePolynomialChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ai" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const BridgeConfig bridge_config(ch, ch);
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const PolynomialConfig polynomial_config(ch, ch.physical_units);
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    return dmx->CreateAIForceBridgePolynomialChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        min_val,
        max_val,
        units,
        bridge_config.ni_bridge_config,
        bridge_config.voltage_excit_source,
        bridge_config.voltage_excit_val,
        bridge_config.nominal_bridge_resistance,
        polynomial_config.forward_coeffs.data(),
        polynomial_config.forward_coeffs.size(),
        polynomial_config.reverse_coeffs.data(),
        polynomial_config.reverse_coeffs.size(),
        polynomial_config.electrical_units,
        polynomial_config.physical_units,
        scale_key
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AIForceBridgeTableChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ai" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const BridgeConfig bridge_config(ch, ch);
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const TableConfig table_config(ch, ch.physical_units);
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    return dmx->CreateAIForceBridgeTableChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        min_val,
        max_val,
        units,
        bridge_config.ni_bridge_config,
        bridge_config.voltage_excit_source,
        bridge_config.voltage_excit_val,
        bridge_config.nominal_bridge_resistance,
        table_config.electrical_vals.data(),
        table_config.electrical_vals.size(),
        table_config.electrical_units,
        table_config.physical_vals.data(),
        table_config.physical_vals.size(),
        table_config.physical_units,
        scale_key
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AIForceBridgeTwoPointLinChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ai" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const BridgeConfig bridge_config(ch, ch);
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const TwoPointLinConfig two_point_lin_config(ch, ch.physical_units);
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    return dmx->CreateAIForceBridgeTwoPointLinChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        min_val,
        max_val,
        units,
        bridge_config.ni_bridge_config,
        bridge_config.voltage_excit_source,
        bridge_config.voltage_excit_val,
        bridge_config.nominal_bridge_resistance,
        two_point_lin_config.first_electrical_val,
        two_point_lin_config.second_electrical_val,
        two_point_lin_config.electrical_units,
        two_point_lin_config.first_physical_val,
        two_point_lin_config.second_physical_val,
        two_point_lin_config.physical_units,
        scale_key
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AIForceIEPEChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ai" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const ExcitationConfig excitation_config(
        static_cast<const ::synnax::ni::CurrentExcitation &>(ch)
    );
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const auto sensitivity = ch.sensitivity;
    const auto [sensitivity_units, sensitivity_units_err] = parse_units(
        ch.sensitivity_units
    );
    if (sensitivity_units_err) return sensitivity_units_err;
    const auto terminal_config = parse_terminal_config(ch.terminal_config);
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    return dmx->CreateAIForceIEPEChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        terminal_config,
        min_val,
        max_val,
        units,
        sensitivity,
        sensitivity_units,
        excitation_config.source,
        excitation_config.val,
        scale_key
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AIFreqVoltageChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const auto hysteresis = ch.hysteresis;
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const auto threshold_level = ch.threshold_level;
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    const auto freq_loc = ctx.dev_loc + "ctr" + std::to_string(ch.port);
    return dmx->CreateAIFreqVoltageChan(
        task_handle,
        freq_loc.c_str(),
        ctx.cfg_path.c_str(),
        min_val,
        max_val,
        units,
        threshold_level,
        hysteresis,
        scale_key
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AIMicrophoneChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ai" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const ExcitationConfig excitation_config(
        static_cast<const ::synnax::ni::CurrentExcitation &>(ch)
    );
    const auto max_snd_press_level = ch.max_snd_press_level;
    const auto mic_sensitivity = ch.mic_sensitivity;
    const auto terminal_config = parse_terminal_config(ch.terminal_config);
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    return dmx->CreateAIMicrophoneChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        terminal_config,
        units,
        mic_sensitivity,
        max_snd_press_level,
        excitation_config.source,
        excitation_config.val,
        scale_key
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AIPressureBridgePolynomialChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ai" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const BridgeConfig bridge_config(ch, ch);
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const PolynomialConfig polynomial_config(ch, ch.physical_units);
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    return dmx->CreateAIPressureBridgePolynomialChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        min_val,
        max_val,
        units,
        bridge_config.ni_bridge_config,
        bridge_config.voltage_excit_source,
        bridge_config.voltage_excit_val,
        bridge_config.nominal_bridge_resistance,
        polynomial_config.forward_coeffs.data(),
        polynomial_config.forward_coeffs.size(),
        polynomial_config.reverse_coeffs.data(),
        polynomial_config.reverse_coeffs.size(),
        polynomial_config.electrical_units,
        polynomial_config.physical_units,
        scale_key
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AIPressureBridgeTableChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ai" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const BridgeConfig bridge_config(ch, ch);
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const TableConfig table_config(ch, ch.physical_units);
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    return dmx->CreateAIPressureBridgeTableChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        min_val,
        max_val,
        units,
        bridge_config.ni_bridge_config,
        bridge_config.voltage_excit_source,
        bridge_config.voltage_excit_val,
        bridge_config.nominal_bridge_resistance,
        table_config.electrical_vals.data(),
        table_config.electrical_vals.size(),
        table_config.electrical_units,
        table_config.physical_vals.data(),
        table_config.physical_vals.size(),
        table_config.physical_units,
        scale_key
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AIPressureBridgeTwoPointLinChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ai" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const BridgeConfig bridge_config(ch, ch);
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const TwoPointLinConfig two_point_lin_config(ch, ch.physical_units);
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    return dmx->CreateAIPressureBridgeTwoPointLinChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        min_val,
        max_val,
        units,
        bridge_config.ni_bridge_config,
        bridge_config.voltage_excit_source,
        bridge_config.voltage_excit_val,
        bridge_config.nominal_bridge_resistance,
        two_point_lin_config.first_electrical_val,
        two_point_lin_config.second_electrical_val,
        two_point_lin_config.electrical_units,
        two_point_lin_config.first_physical_val,
        two_point_lin_config.second_physical_val,
        two_point_lin_config.physical_units,
        scale_key
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AIRTDChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ai" + std::to_string(ch.port);
    const ExcitationConfig excitation_config(
        static_cast<const ::synnax::ni::CurrentExcitation &>(ch)
    );
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const auto r0 = ch.r_0;
    const auto resistance_config = parse_resistance_config(ch.resistance_config);
    const auto rtd_type = get_rtd_type(ch.rtd_type);
    const auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    return dmx->CreateAIRTDChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        min_val,
        max_val,
        units,
        rtd_type,
        resistance_config,
        excitation_config.source,
        excitation_config.val,
        r0
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AIResistanceChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ai" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const ExcitationConfig excitation_config(
        static_cast<const ::synnax::ni::CurrentExcitation &>(ch)
    );
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const auto resistance_config = parse_resistance_config(ch.resistance_config);
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    return dmx->CreateAIResistanceChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        min_val,
        max_val,
        units,
        resistance_config,
        excitation_config.source,
        excitation_config.val,
        scale_key
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AIRosetteStrainGageChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ai" + std::to_string(ch.port);
    const ExcitationConfig excitation_config(
        static_cast<const ::synnax::ni::VoltageExcitation &>(ch)
    );
    const auto gage_factor = ch.gage_factor;
    const auto gage_orientation = ch.gage_orientation;
    const auto lead_wire_resistance = ch.lead_wire_resistance;
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const auto nominal_gage_resistance = ch.nominal_gage_resistance;
    const auto poisson_ratio = ch.poisson_ratio;
    const auto rosette_meas_types = [&ch] {
        std::vector<int32> types;
        types.reserve(ch.rosette_meas_types.size());
        for (const auto &t: ch.rosette_meas_types)
            types.push_back(get_rosette_meas_type(t));
        return types;
    }();
    const auto rosette_type = get_rosette_type(ch.rosette_type);
    const auto strain_config = get_strain_config(ch.strain_config);
    return dmx->CreateAIRosetteStrainGageChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        min_val,
        max_val,
        rosette_type,
        gage_orientation,
        rosette_meas_types.data(),
        static_cast<uint32_t>(rosette_meas_types.size()),
        strain_config,
        excitation_config.source,
        excitation_config.val,
        gage_factor,
        nominal_gage_resistance,
        poisson_ratio,
        lead_wire_resistance
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AIStrainGaugeChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ai" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const ExcitationConfig excitation_config(
        static_cast<const ::synnax::ni::VoltageExcitation &>(ch)
    );
    const auto gage_factor = ch.gage_factor;
    const auto initial_bridge_voltage = ch.initial_bridge_voltage;
    const auto lead_wire_resistance = ch.lead_wire_resistance;
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const auto nominal_gage_resistance = ch.nominal_gage_resistance;
    const auto poisson_ratio = ch.poisson_ratio;
    const auto strain_config = get_strain_config(ch.strain_config);
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    return dmx->CreateAIStrainGageChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        min_val,
        max_val,
        units,
        strain_config,
        excitation_config.source,
        excitation_config.val,
        gage_factor,
        initial_bridge_voltage,
        nominal_gage_resistance,
        poisson_ratio,
        lead_wire_resistance,
        scale_key
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AITempBuiltinChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    const auto i_name = ctx.dev_loc + "/_boardTempSensor_vs_aignd";
    return dmx->CreateAITempBuiltInSensorChan(
        task_handle,
        i_name.c_str(),
        ctx.cfg_path.c_str(),
        units
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AIThermistorIexChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ai" + std::to_string(ch.port);
    const auto a = ch.a;
    const auto b = ch.b;
    const auto c = ch.c;
    const ExcitationConfig excitation_config(
        static_cast<const ::synnax::ni::CurrentExcitation &>(ch)
    );
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const auto resistance_config = parse_resistance_config(ch.resistance_config);
    const auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    return dmx->CreateAIThrmstrChanIex(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        min_val,
        max_val,
        units,
        resistance_config,
        excitation_config.source,
        excitation_config.val,
        a,
        b,
        c
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AIThermistorVexChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ai" + std::to_string(ch.port);
    const auto a = ch.a;
    const auto b = ch.b;
    const auto c = ch.c;
    const ExcitationConfig excitation_config(
        static_cast<const ::synnax::ni::VoltageExcitation &>(ch)
    );
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const auto r1 = ch.r_1;
    const auto resistance_config = parse_resistance_config(ch.resistance_config);
    const auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    return dmx->CreateAIThrmstrChanVex(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        min_val,
        max_val,
        units,
        resistance_config,
        excitation_config.source,
        excitation_config.val,
        a,
        b,
        c,
        r1
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AITorqueBridgePolynomialChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ai" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const BridgeConfig bridge_config(ch, ch);
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const PolynomialConfig polynomial_config(ch, ch.physical_units);
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    return dmx->CreateAITorqueBridgePolynomialChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        min_val,
        max_val,
        units,
        bridge_config.ni_bridge_config,
        bridge_config.voltage_excit_source,
        bridge_config.voltage_excit_val,
        bridge_config.nominal_bridge_resistance,
        polynomial_config.forward_coeffs.data(),
        polynomial_config.forward_coeffs.size(),
        polynomial_config.reverse_coeffs.data(),
        polynomial_config.reverse_coeffs.size(),
        polynomial_config.electrical_units,
        polynomial_config.physical_units,
        scale_key
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AITorqueBridgeTableChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ai" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const BridgeConfig bridge_config(ch, ch);
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const TableConfig table_config(ch, ch.physical_units);
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    return dmx->CreateAITorqueBridgeTableChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        min_val,
        max_val,
        units,
        bridge_config.ni_bridge_config,
        bridge_config.voltage_excit_source,
        bridge_config.voltage_excit_val,
        bridge_config.nominal_bridge_resistance,
        table_config.electrical_vals.data(),
        table_config.electrical_vals.size(),
        table_config.electrical_units,
        table_config.physical_vals.data(),
        table_config.physical_vals.size(),
        table_config.physical_units,
        scale_key
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AITorqueBridgeTwoPointLinChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ai" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const BridgeConfig bridge_config(ch, ch);
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const TwoPointLinConfig two_point_lin_config(ch, ch.physical_units);
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    return dmx->CreateAITorqueBridgeTwoPointLinChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        min_val,
        max_val,
        units,
        bridge_config.ni_bridge_config,
        bridge_config.voltage_excit_source,
        bridge_config.voltage_excit_val,
        bridge_config.nominal_bridge_resistance,
        two_point_lin_config.first_electrical_val,
        two_point_lin_config.second_electrical_val,
        two_point_lin_config.electrical_units,
        two_point_lin_config.first_physical_val,
        two_point_lin_config.second_physical_val,
        two_point_lin_config.physical_units,
        scale_key
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AIVelocityIEPEChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ai" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const ExcitationConfig excitation_config(
        static_cast<const ::synnax::ni::CurrentExcitation &>(ch)
    );
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const auto sensitivity = ch.sensitivity;
    const auto [sensitivity_units, sensitivity_units_err] = parse_units(
        ch.sensitivity_units
    );
    if (sensitivity_units_err) return sensitivity_units_err;
    const auto terminal_config = parse_terminal_config(ch.terminal_config);
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    return dmx->CreateAIVelocityIEPEChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        terminal_config,
        min_val,
        max_val,
        units,
        sensitivity,
        sensitivity_units,
        excitation_config.source,
        excitation_config.val,
        scale_key
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AIVoltageChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ai" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const auto terminal_config = parse_terminal_config(ch.terminal_config);
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    return dmx->CreateAIVoltageChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        terminal_config,
        min_val,
        max_val,
        units,
        scale_key
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AIVoltageRMSChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ai" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const auto terminal_config = parse_terminal_config(ch.terminal_config);
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    return dmx->CreateAIVoltageRMSChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        terminal_config,
        min_val,
        max_val,
        units,
        scale_key
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AIVoltageWithExcitChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ai" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const auto bridge_config = parse_bridge_config(ch.bridge_config);
    const ExcitationConfig excitation_config(ch, ch.use_excit_for_scaling);
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const auto terminal_config = parse_terminal_config(ch.terminal_config);
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    return dmx->CreateAIVoltageChanWithExcit(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        terminal_config,
        min_val,
        max_val,
        units,
        bridge_config,
        excitation_config.source,
        excitation_config.val,
        static_cast<bool32>(excitation_config.min_val_for_excitation),
        scale_key
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AOCurrentChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ao" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    return dmx->CreateAOCurrentChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        min_val,
        max_val,
        units,
        scale_key
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AOFuncGenChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ao" + std::to_string(ch.port);
    const auto amplitude = ch.amplitude;
    const auto frequency = ch.frequency;
    const auto offset = ch.offset;
    const auto wave_type = parse_wave_type(ch.wave_type);
    return dmx->CreateAOFuncGenChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        wave_type,
        frequency,
        amplitude,
        offset
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AOVoltageChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ao" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    return dmx->CreateAOVoltageChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        min_val,
        max_val,
        units,
        scale_key
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::CIPositionAngularChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ctr" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const auto decoding_type = get_ci_decoding_type(ch.decoding_type);
    const auto initial_angle = ch.initial_angle;
    const auto pulses_per_rev = static_cast<uint32_t>(ch.pulses_per_rev);
    const auto terminal_a = ch.terminal_a;
    const auto terminal_b = ch.terminal_b;
    const auto terminal_z = ch.terminal_z;
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    const auto z_index_enable = ch.z_index_enable;
    const auto z_index_phase = get_ci_z_index_phase(ch.z_index_phase);
    const auto z_index_val = ch.z_index_val;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    auto err = dmx->CreateCIAngEncoderChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        decoding_type,
        z_index_enable,
        z_index_val,
        z_index_phase,
        units,
        pulses_per_rev,
        initial_angle,
        scale_key
    );
    if (err) return err;

    // Set terminal A if specified (empty string uses DAQmx default)
    if (!terminal_a.empty()) {
        err = dmx->SetChanAttributeString(
            task_handle,
            ctx.cfg_path.c_str(),
            DAQmx_CI_Encoder_AInputTerm,
            terminal_a.c_str()
        );
        if (err) return err;
    }

    // Set terminal B if specified (empty string uses DAQmx default)
    if (!terminal_b.empty()) {
        err = dmx->SetChanAttributeString(
            task_handle,
            ctx.cfg_path.c_str(),
            DAQmx_CI_Encoder_BInputTerm,
            terminal_b.c_str()
        );
        if (err) return err;
    }

    // Set terminal Z if specified (empty string uses DAQmx default)
    if (!terminal_z.empty()) {
        err = dmx->SetChanAttributeString(
            task_handle,
            ctx.cfg_path.c_str(),
            DAQmx_CI_Encoder_ZInputTerm,
            terminal_z.c_str()
        );
    }

    return err;
}

inline x::errors::Error apply(
    const ::synnax::ni::CIVelocityAngularChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ctr" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const auto decoding_type = get_ci_decoding_type(ch.decoding_type);
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const auto pulses_per_rev = static_cast<uint32_t>(ch.pulses_per_rev);
    const auto terminal_a = ch.terminal_a;
    const auto terminal_b = ch.terminal_b;
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    auto err = dmx->CreateCIAngVelocityChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        min_val,
        max_val,
        decoding_type,
        units,
        pulses_per_rev,
        scale_key
    );
    if (err) return err;

    // Set terminal A if specified (empty string uses DAQmx default)
    if (!terminal_a.empty()) {
        err = dmx->SetChanAttributeString(
            task_handle,
            ctx.cfg_path.c_str(),
            DAQmx_CI_Velocity_Encoder_AInputTerm,
            terminal_a.c_str()
        );
        if (err) return err;
    }

    // Set terminal B if specified (empty string uses DAQmx default)
    if (!terminal_b.empty()) {
        err = dmx->SetChanAttributeString(
            task_handle,
            ctx.cfg_path.c_str(),
            DAQmx_CI_Velocity_Encoder_BInputTerm,
            terminal_b.c_str()
        );
    }

    return err;
}

inline x::errors::Error apply(
    const ::synnax::ni::CIDutyCycleChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ctr" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const auto edge = get_ci_edge(ch.active_edge);
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const auto terminal = ch.terminal;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    auto err = dmx->CreateCIDutyCycleChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        min_val,
        max_val,
        edge,
        scale_key
    );
    if (err) return err;

    // Set the input terminal if specified (empty string uses DAQmx default)
    if (!terminal.empty()) {
        err = dmx->SetChanAttributeString(
            task_handle,
            ctx.cfg_path.c_str(),
            DAQmx_CI_DutyCycle_Term,
            terminal.c_str()
        );
    }

    return err;
}

inline x::errors::Error apply(
    const ::synnax::ni::CIEdgeCountChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ctr" + std::to_string(ch.port);
    const auto count_direction = get_ci_count_direction(ch.count_direction);
    const auto edge = get_ci_edge(ch.active_edge);
    const auto initial_count = static_cast<uint32_t>(ch.initial_count);
    const auto terminal = ch.terminal;
    auto err = dmx->CreateCICountEdgesChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        edge,
        initial_count,
        count_direction
    );
    if (err) return err;

    // Set the PFI terminal if specified (empty string uses DAQmx default)
    if (!terminal.empty()) {
        err = dmx->SetChanAttributeString(
            task_handle,
            ctx.cfg_path.c_str(),
            DAQmx_CI_CountEdges_Term,
            terminal.c_str()
        );
    }

    return err;
}

inline x::errors::Error apply(
    const ::synnax::ni::CIFrequencyChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ctr" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const auto divisor = ch.divisor;
    const auto edge = get_ci_edge(ch.edge);
    const auto max_val = ch.max_val;
    const auto meas_method = get_ci_meas_method(ch.meas_method);
    const auto meas_time = ch.meas_time;
    const auto min_val = ch.min_val;
    const auto terminal = ch.terminal;
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    auto err = dmx->CreateCIFreqChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        min_val,
        max_val,
        units,
        edge,
        meas_method,
        meas_time,
        divisor,
        scale_key
    );
    if (err) return err;

    // Set the PFI terminal if specified (empty string uses DAQmx default)
    if (!terminal.empty()) {
        err = dmx->SetChanAttributeString(
            task_handle,
            ctx.cfg_path.c_str(),
            DAQmx_CI_Freq_Term,
            terminal.c_str()
        );
    }

    return err;
}

inline x::errors::Error apply(
    const ::synnax::ni::CIPositionLinearChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ctr" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const auto decoding_type = get_ci_decoding_type(ch.decoding_type);
    const auto dist_per_pulse = ch.dist_per_pulse;
    const auto initial_pos = ch.initial_pos;
    const auto terminal_a = ch.terminal_a;
    const auto terminal_b = ch.terminal_b;
    const auto terminal_z = ch.terminal_z;
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    const auto z_index_enable = ch.z_index_enable;
    const auto z_index_phase = get_ci_z_index_phase(ch.z_index_phase);
    const auto z_index_val = ch.z_index_val;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    auto err = dmx->CreateCILinEncoderChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        decoding_type,
        z_index_enable,
        z_index_val,
        z_index_phase,
        units,
        dist_per_pulse,
        initial_pos,
        scale_key
    );
    if (err) return err;

    // Set terminal A if specified (empty string uses DAQmx default)
    if (!terminal_a.empty()) {
        err = dmx->SetChanAttributeString(
            task_handle,
            ctx.cfg_path.c_str(),
            DAQmx_CI_Encoder_AInputTerm,
            terminal_a.c_str()
        );
        if (err) return err;
    }

    // Set terminal B if specified (empty string uses DAQmx default)
    if (!terminal_b.empty()) {
        err = dmx->SetChanAttributeString(
            task_handle,
            ctx.cfg_path.c_str(),
            DAQmx_CI_Encoder_BInputTerm,
            terminal_b.c_str()
        );
        if (err) return err;
    }

    // Set terminal Z if specified (empty string uses DAQmx default)
    if (!terminal_z.empty()) {
        err = dmx->SetChanAttributeString(
            task_handle,
            ctx.cfg_path.c_str(),
            DAQmx_CI_Encoder_ZInputTerm,
            terminal_z.c_str()
        );
    }

    return err;
}

inline x::errors::Error apply(
    const ::synnax::ni::CIVelocityLinearChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ctr" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const auto decoding_type = get_ci_decoding_type(ch.decoding_type);
    const auto dist_per_pulse = ch.dist_per_pulse;
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const auto terminal_a = ch.terminal_a;
    const auto terminal_b = ch.terminal_b;
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    auto err = dmx->CreateCILinVelocityChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        min_val,
        max_val,
        decoding_type,
        units,
        dist_per_pulse,
        scale_key
    );
    if (err) return err;

    // Set terminal A if specified (empty string uses DAQmx default)
    if (!terminal_a.empty()) {
        err = dmx->SetChanAttributeString(
            task_handle,
            ctx.cfg_path.c_str(),
            DAQmx_CI_Velocity_Encoder_AInputTerm,
            terminal_a.c_str()
        );
        if (err) return err;
    }

    // Set terminal B if specified (empty string uses DAQmx default)
    if (!terminal_b.empty()) {
        err = dmx->SetChanAttributeString(
            task_handle,
            ctx.cfg_path.c_str(),
            DAQmx_CI_Velocity_Encoder_BInputTerm,
            terminal_b.c_str()
        );
    }

    return err;
}

inline x::errors::Error apply(
    const ::synnax::ni::CIPeriodChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ctr" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const auto divisor = ch.divisor;
    const auto edge = get_ci_edge(ch.starting_edge);
    const auto max_val = ch.max_val;
    const auto meas_method = get_ci_meas_method(ch.meas_method);
    const auto meas_time = ch.meas_time;
    const auto min_val = ch.min_val;
    const auto terminal = ch.terminal;
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    auto err = dmx->CreateCIPeriodChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        min_val,
        max_val,
        units,
        edge,
        meas_method,
        meas_time,
        divisor,
        scale_key
    );
    if (err) return err;

    // Set the PFI terminal if specified (empty string uses DAQmx default)
    if (!terminal.empty()) {
        err = dmx->SetChanAttributeString(
            task_handle,
            ctx.cfg_path.c_str(),
            DAQmx_CI_Period_Term,
            terminal.c_str()
        );
    }

    return err;
}

inline x::errors::Error apply(
    const ::synnax::ni::CIPulseWidthChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ctr" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const auto edge = get_ci_edge(ch.starting_edge);
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const auto terminal = ch.terminal;
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    auto err = dmx->CreateCIPulseWidthChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        min_val,
        max_val,
        units,
        edge,
        scale_key
    );
    if (err) return err;

    // Set the PFI terminal if specified (empty string uses DAQmx default)
    if (!terminal.empty()) {
        err = dmx->SetChanAttributeString(
            task_handle,
            ctx.cfg_path.c_str(),
            DAQmx_CI_PulseWidth_Term,
            terminal.c_str()
        );
    }

    return err;
}

inline x::errors::Error apply(
    const ::synnax::ni::CISemiPeriodChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ctr" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const auto terminal = ch.terminal;
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    auto err = dmx->CreateCISemiPeriodChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        min_val,
        max_val,
        units,
        scale_key
    );
    if (err) return err;

    // Set the PFI terminal if specified (empty string uses DAQmx default)
    if (!terminal.empty()) {
        err = dmx->SetChanAttributeString(
            task_handle,
            ctx.cfg_path.c_str(),
            DAQmx_CI_SemiPeriod_Term,
            terminal.c_str()
        );
    }

    return err;
}

inline x::errors::Error apply(
    const ::synnax::ni::CITwoEdgeSepChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ctr" + std::to_string(ch.port);
    auto [scale, scale_err] = make_scale(ch.custom_scale);
    if (scale_err) return scale_err;
    const auto first_edge = get_ci_edge(ch.first_edge);
    const auto first_terminal = ch.first_terminal;
    const auto max_val = ch.max_val;
    const auto min_val = ch.min_val;
    const auto second_edge = get_ci_edge(ch.second_edge);
    const auto second_terminal = ch.second_terminal;
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    if (!scale->is_none()) units = DAQmx_Val_FromCustomScale;
    auto [key, key_err] = scale->apply(dmx);
    if (key_err) return key_err;
    const char *scale_key = key.empty() ? nullptr : key.c_str();
    auto err = dmx->CreateCITwoEdgeSepChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        min_val,
        max_val,
        units,
        first_edge,
        second_edge,
        scale_key
    );
    if (err) return err;

    // Set the first terminal if specified (empty string uses DAQmx default)
    if (!first_terminal.empty()) {
        err = dmx->SetChanAttributeString(
            task_handle,
            ctx.cfg_path.c_str(),
            DAQmx_CI_TwoEdgeSep_FirstTerm,
            first_terminal.c_str()
        );
        if (err) return err;
    }

    // Set the second terminal if specified (empty string uses DAQmx default)
    if (!second_terminal.empty()) {
        err = dmx->SetChanAttributeString(
            task_handle,
            ctx.cfg_path.c_str(),
            DAQmx_CI_TwoEdgeSep_SecondTerm,
            second_terminal.c_str()
        );
    }

    return err;
}

inline x::errors::Error apply(
    const ::synnax::ni::DigitalInputChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/port" + std::to_string(ch.port) + "/line" +
                     std::to_string(ch.line);
    return dmx->CreateDIChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        DAQmx_Val_ChanPerLine
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::DOChannelDigitalOutput &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/port" + std::to_string(ch.port) + "/line" +
                     std::to_string(ch.line);
    return dmx->CreateDOChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        DAQmx_Val_ChanPerLine
    );
}

inline x::errors::Error apply(
    const ::synnax::ni::AIThermocoupleChannel &ch,
    const ApplyContext &ctx,
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    const TaskHandle task_handle
) {
    const auto loc = ctx.dev_loc + "/ai" + std::to_string(ch.port);
    auto [units, units_err] = parse_units(ch.units);
    if (units_err) return units_err;
    const auto cjc = parse_cjc(ch.cjc);
    const auto cjc_port = format_cjc_port(ctx.cfg_path, cjc.port);
    return dmx->CreateAIThrmcplChan(
        task_handle,
        loc.c_str(),
        ctx.cfg_path.c_str(),
        ch.min_val,
        ch.max_val,
        units,
        parse_thermocouple_type(ch.thermocouple_type),
        cjc.source,
        cjc.val,
        cjc_port.c_str()
    );
}
/// @brief the generated wire representation of any input channel.
using InputUnion = std::
    variant<::synnax::ni::AIChannel, ::synnax::ni::CIChannel, ::synnax::ni::DIChannel>;
/// @brief the generated wire representation of any output channel.
using OutputUnion = std::variant<::synnax::ni::AOChannel, ::synnax::ni::DOChannel>;

/// @brief an input channel: the parsed wire configuration plus the runtime
/// state the task binds before applying it to the hardware.
struct Input {
    /// @brief whether data acquisition is enabled for the channel.
    bool enabled = true;
    /// @brief the key of the device the channel belongs to. Empty when the
    /// owning task supplies a single device.
    std::string dev_key;
    /// @brief the channel's path within the task config, used for error
    /// propagation and as the DAQmx channel name.
    std::string cfg_path;
    /// @brief the key of the Synnax channel acquired data is written to.
    synnax::channel::Key synnax_key = 0;
    /// @brief the resolved Synnax channel, bound by the owning task.
    synnax::channel::Channel ch;
    /// @brief the physical device location, bound by the owning task.
    std::string dev_loc;
    /// @brief the parsed wire configuration.
    InputUnion channel;

    void bind_remote_info(const synnax::channel::Channel &c, const std::string &loc) {
        this->ch = c;
        this->dev_loc = loc;
    }

    [[nodiscard]] x::errors::Error
    apply(const std::shared_ptr<daqmx::SugaredAPI> &dmx, TaskHandle task_handle) const {
        const ApplyContext ctx{this->dev_loc, this->cfg_path};
        return std::visit(
            [&](const auto &u) {
                return std::visit(
                    [&](const auto &v) {
                        return channel::apply(v, ctx, dmx, task_handle);
                    },
                    u
                );
            },
            this->channel
        );
    }
};

/// @brief an output channel: the parsed wire configuration plus the runtime
/// state the task binds before applying it to the hardware.
struct Output {
    /// @brief whether the channel is enabled for the task.
    bool enabled = true;
    /// @brief the channel's path within the task config.
    std::string cfg_path;
    /// @brief the key of the Synnax channel commands are read from.
    synnax::channel::Key cmd_ch_key = 0;
    /// @brief the key of the Synnax channel output state is written to.
    synnax::channel::Key state_ch_key = 0;
    /// @brief the resolved state channel, bound by the owning task.
    synnax::channel::Channel state_ch;
    /// @brief the physical device location, bound by the owning task.
    std::string dev_loc;
    /// @brief the parsed wire configuration.
    OutputUnion channel;

    void bind_remote_info(const synnax::channel::Channel &c, const std::string &loc) {
        this->state_ch = c;
        this->dev_loc = loc;
    }

    [[nodiscard]] x::errors::Error
    apply(const std::shared_ptr<daqmx::SugaredAPI> &dmx, TaskHandle task_handle) const {
        const ApplyContext ctx{this->dev_loc, this->cfg_path};
        return std::visit(
            [&](const auto &u) {
                return std::visit(
                    [&](const auto &v) {
                        return channel::apply(v, ctx, dmx, task_handle);
                    },
                    u
                );
            },
            this->channel
        );
    }
};

/// @brief parses an input channel for the given read task type, dispatching to
/// the channel family the task accepts.
inline std::unique_ptr<Input>
parse_input(x::json::Parser &cfg, const std::string &task_type) {
    auto out = std::make_unique<Input>();
    out->cfg_path = format_cfg_path(cfg.path_prefix);
    if (task_type == "ni_analog_read") {
        out->channel = ::synnax::ni::parse_ai_channel(cfg);
    } else if (task_type == "ni_counter_read") {
        out->channel = ::synnax::ni::parse_ci_channel(cfg);
    } else if (task_type == "ni_digital_read") {
        out->channel = ::synnax::ni::parse_di_channel(cfg);
    } else {
        cfg.field_err("type", "unknown read task type: " + task_type);
        return nullptr;
    }
    std::visit(
        [&](const auto &u) {
            std::visit(
                [&](const auto &v) {
                    out->enabled = !v.disabled;
                    out->synnax_key = v.channel;
                    if constexpr (requires { v.device; }) out->dev_key = v.device;
                },
                u
            );
        },
        out->channel
    );
    return out;
}

/// @brief parses an output channel for the given write task type, dispatching
/// to the channel family the task accepts.
inline std::unique_ptr<Output>
parse_output(x::json::Parser &cfg, const std::string &task_type) {
    auto out = std::make_unique<Output>();
    out->cfg_path = format_cfg_path(cfg.path_prefix);
    if (task_type == "ni_analog_write")
        out->channel = ::synnax::ni::parse_ao_channel(cfg);
    else if (task_type == "ni_digital_write")
        out->channel = ::synnax::ni::parse_do_channel(cfg);
    else {
        cfg.field_err("type", "unknown write task type: " + task_type);
        return nullptr;
    }
    std::visit(
        [&](const auto &u) {
            std::visit(
                [&](const auto &v) {
                    out->enabled = !v.disabled;
                    out->cmd_ch_key = v.cmd_channel;
                    out->state_ch_key = v.state_channel;
                },
                u
            );
        },
        out->channel
    );
    return out;
}
}
