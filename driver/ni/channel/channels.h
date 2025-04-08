// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// ReSharper disable CppParameterMayBeConst
#pragma once

/// std
#include <map>
#include <string>

/// module
#include "client/cpp/synnax.h"
#include "x/cpp/xjson/xjson.h"

/// internal
#include "driver/ni/channel/scale.h"
#include "driver/ni/channel/units.h"
#include "driver/ni/daqmx/sugared.h"

namespace channel {
static int32_t parse_terminal_config(xjson::Parser &p) {
    const auto s = p.required<std::string>("terminal_config");
    if (s == "PseudoDiff") return DAQmx_Val_PseudoDiff;
    if (s == "Diff") return DAQmx_Val_Diff;
    if (s == "NRSE") return DAQmx_Val_NRSE;
    if (s == "RSE") return DAQmx_Val_RSE;
    return DAQmx_Val_Cfg_Default;
}

static int32_t parse_bridge_config(xjson::Parser &p) {
    const auto s = p.required<std::string>("bridge_config");
    if (s == "FullBridge") return DAQmx_Val_FullBridge;
    if (s == "HalfBridge") return DAQmx_Val_HalfBridge;
    if (s == "QuarterBridge") return DAQmx_Val_QuarterBridge;
    return DAQmx_Val_FullBridge;
}

static int32_t parse_resistance_config(xjson::Parser &p) {
    const auto s = p.required<std::string>("resistance_config");
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

struct ExcitationConfig {
    const int32_t source;
    const double val;
    const double min_val_for_excitation; // optional
    const double max_val_for_excitation; // optional
    const bool32 use_excit_for_scaling; // optional

    explicit ExcitationConfig(xjson::Parser &cfg, const std::string &prefix):
        source(get_excitation_src(cfg.required<std::string>(prefix + "_excit_source"))),
        val(cfg.required<double>(prefix + "_excit_val")),
        min_val_for_excitation(cfg.optional<double>("min_val_for_excitation", 0)),
        max_val_for_excitation(cfg.optional<double>("max_val_for_excitation", 0)),
        use_excit_for_scaling(cfg.optional<bool32>("use_excit_for_scaling", 0)) {}
};

const std::string CURR_EXCIT_PREFIX = "current";
const std::string VOLT_EXCIT_PREFIX = "voltage";

struct BridgeConfig {
    const int32_t ni_bridge_config;
    const int32_t voltage_excit_source;
    const double voltage_excit_val;
    const double nominal_bridge_resistance;

    explicit BridgeConfig(xjson::Parser &cfg):
        ni_bridge_config(parse_bridge_config(cfg)),
        voltage_excit_source(
            get_excitation_src(cfg.required<std::string>("voltage_excit_source"))
        ),
        voltage_excit_val(cfg.required<double>("voltage_excit_val")),
        nominal_bridge_resistance(cfg.required<double>("nominal_bridge_resistance")) {}
};

struct PolynomialConfig {
    float64 *forward_coeffs;
    const uint32_t num_forward_coeffs;
    float64 *reverse_coeffs;
    const uint32_t num_reverse_coeffs;
    int32_t electrical_units;
    int32_t physical_units;

    explicit PolynomialConfig(xjson::Parser &cfg):
        num_forward_coeffs(cfg.required<uint32_t>("num_forward_coeffs")),
        num_reverse_coeffs(cfg.required<uint32_t>("num_reverse_coeffs")) {
        const auto eu = cfg.required<std::string>("electrical_units");
        const auto pu = cfg.required<std::string>("physical_units");

        const auto ni_eu = channel::UNITS_MAP.find(eu);
        if (ni_eu == channel::UNITS_MAP.end())
            electrical_units = DAQmx_Val_Volts;
        else
            electrical_units = ni_eu->second;

        const auto ni_pu = channel::UNITS_MAP.find(pu);
        if (ni_pu == channel::UNITS_MAP.end())
            physical_units = DAQmx_Val_Volts;
        else
            physical_units = channel::UNITS_MAP.at(pu);
        forward_coeffs = new double[num_forward_coeffs];
        reverse_coeffs = new double[num_reverse_coeffs];
        const auto f = cfg.required_vec<double>("forward_coeffs");
        for (uint32_t i = 0; i < num_forward_coeffs; i++)
            forward_coeffs[i] = f[i];
    }

    ~PolynomialConfig() {
        delete[] forward_coeffs;
        delete[] reverse_coeffs;
    }
};

struct TableConfig {
    float64 *electrical_vals;
    uint32_t num_electrical_vals;
    float64 *physical_vals;
    uint32_t num_physical_vals;
    int32_t electrical_units;
    int32_t physical_units;

    TableConfig() = default;

    explicit TableConfig(xjson::Parser &cfg) {
        const auto eu = cfg.required<std::string>("electrical_units");
        const auto pu = cfg.required<std::string>("physical_units");

        electrical_units = channel::UNITS_MAP.at(eu);
        physical_units = channel::UNITS_MAP.at(pu);

        // TODO: figure out why using vector and .data() throws exception when
        // passed to NI function
        const auto ev = cfg.required_vec<double>("electrical_vals");
        num_electrical_vals = ev.size();
        electrical_vals = new double[num_electrical_vals];
        for (uint32_t i = 0; i < num_electrical_vals; i++)
            electrical_vals[i] = ev[i];

        const auto pv = cfg.required_vec<double>("physical_vals");
        num_physical_vals = pv.size();
        physical_vals = new double[num_physical_vals];
        for (uint32_t i = 0; i < num_physical_vals; i++)
            physical_vals[i] = pv[i];
    }

    ~TableConfig() {
        delete[] electrical_vals;
        delete[] physical_vals;
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

    explicit TwoPointLinConfig(xjson::Parser &cfg):
        first_electrical_val(cfg.required<double>("first_electrical_val")),
        second_electrical_val(cfg.required<double>("second_electrical_val")),
        electrical_units(UNITS_MAP.at(cfg.required<std::string>("electrical_units"))),
        first_physical_val(cfg.required<double>("first_physical_val")),
        second_physical_val(cfg.required<double>("second_physical_val")),
        physical_units(UNITS_MAP.at(cfg.required<std::string>("physical_units"))) {
        const auto eu = cfg.required<std::string>("electrical_units");
    }
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

/// @brief base channel class that all NI channels inherit from.
struct Base {
    /// @brief whether data acquisition/control is enabled.
    const bool enabled;
    /// @brief the device key that the channel is associated with. This key is
    /// optional, and can be ultimately overridden by the caller in bind_remote_info
    /// implementations.
    const std::string dev_key;
    /// @brief the path within the JSON configuration structure that the channel is
    /// defined within. This is used for error propagation.
    const std::string cfg_path;
    /// @brief the actual location of the device e.g. "cDAQ1Mod1". This is not
    /// constant, it gets bound by the caller after fetching all the devices for the
    /// task.
    std::string dev_loc;

    virtual ~Base() = default;

    explicit Base(xjson::Parser &cfg):
        enabled(cfg.optional<bool>("enabled", true)),
        dev_key(cfg.optional<std::string>("device", "")),
        cfg_path(format_cfg_path(cfg.path_prefix)) {}

    /// @brief applies the channel configuration to the DAQmx task.
    virtual xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle
    ) const = 0;
};

/// @brief base class for an input channel (AI, DI)
struct Input : virtual Base {
    /// @brief the key of the synnax channel that we'll write acquired data to.
    const synnax::ChannelKey synnax_key;
    /// @brief the properties of the synnax channel that we'll write acquired data
    /// to. This field is bound by the caller after fetching all the synnax channels
    /// for the task.
    synnax::Channel ch;

    explicit Input(xjson::Parser &cfg):
        Base(cfg), synnax_key(cfg.required<synnax::ChannelKey>("channel")) {}

    /// @brief binds remotely fetched information to the channel.
    void bind_remote_info(const synnax::Channel &ch, const std::string &dev_loc) {
        this->ch = ch;
        this->dev_loc = dev_loc;
    }
};

/// @brief base class for an output channel (AO, DO)
struct Output : virtual Base {
    /// @brief the key of the command channel that we'll receive commands from.
    const synnax::ChannelKey cmd_ch_key;
    /// @brief the key of the state channel that we'll write the state of the
    /// command channel to.
    const synnax::ChannelKey state_ch_key;
    /// @brief the properties of the command channel that we'll receive commands
    /// from. This field is bound by the caller after fetching all the synnax
    /// channels for the task.
    synnax::Channel state_ch;

    explicit Output(xjson::Parser &cfg):
        Base(cfg),
        cmd_ch_key(cfg.required<synnax::ChannelKey>("cmd_channel")),
        state_ch_key(cfg.required<synnax::ChannelKey>("state_channel")) {}

    /// @brief binds remotely fetched information to the channel.
    void bind_remote_info(const synnax::Channel &state_ch, const std::string &dev_loc) {
        this->state_ch = state_ch;
        this->dev_loc = dev_loc;
    }
};

/// @brief base class for a digital channel (DI, DO)
struct Digital : virtual Base {
    const int port;
    const int line;

    explicit Digital(xjson::Parser &cfg):
        port(cfg.required<int>("port")), line(cfg.required<int>("line")) {}

    [[nodiscard]] std::string loc() const {
        return this->dev_loc + "/port" + std::to_string(this->port) + "/line" +
               std::to_string(this->line);
    }
};

/// @brief configuration for a digital input channel.
struct DI final : Digital, Input {
    explicit DI(xjson::Parser &cfg): Base(cfg), Digital(cfg), Input(cfg) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle
    ) const override {
        return dmx->CreateDIChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            DAQmx_Val_ChanPerLine
        );
    }
};

/// @brief configuration for a digital output channel.
struct DO final : Digital, Output {
    explicit DO(xjson::Parser &cfg): Base(cfg), Digital(cfg), Output(cfg) {}


    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle
    ) const override {
        return dmx->CreateDOChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            DAQmx_Val_ChanPerLine
        );
    }
};

/// @brief base class for all analog channels (AO, DO)
struct Analog : virtual Base {
    const int port;
    const double min_val;
    const double max_val;
    int32_t units;


    explicit Analog(xjson::Parser &cfg):
        port(cfg.required<int>("port")),
        min_val(cfg.optional<float>("min_val", 0)),
        max_val(cfg.optional<float>("max_val", 0)),
        units(parse_units(cfg, "units")) {}
};

/// @brief base class for analog channels that can have a custom scale applied.
struct AnalogCustomScale : virtual Analog {
    const std::unique_ptr<Scale> scale;

    explicit AnalogCustomScale(xjson::Parser &cfg):
        Analog(cfg), scale(parse_scale(cfg, "custom_scale")) {
        if (!this->scale->is_none()) units = DAQmx_Val_FromCustomScale;
    }

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle
    ) const override {
        auto [scale_key, err] = this->scale->apply(dmx);
        if (err) return err;
        return this
            ->apply(dmx, task_handle, scale_key.empty() ? nullptr : scale_key.c_str());
    }

    virtual xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const = 0;
};

/// @brief base class for analog input channels.
struct AI : virtual Analog, Input {
    explicit AI(xjson::Parser &cfg): Analog(cfg), Input(cfg) {}

    [[nodiscard]] std::string loc() const {
        return this->dev_loc + "/ai" + std::to_string(this->port);
    }
};

/// @brief base class for analog output channels.
struct AO : virtual Analog, Output {
    explicit AO(xjson::Parser &cfg): Analog(cfg), Output(cfg) {}

    [[nodiscard]] std::string loc() const {
        return this->dev_loc + "/ao" + std::to_string(this->port);
    }
};

/// @brief base class for analog channels that can have a custom scale applied.
struct AICustomScale : AI, AnalogCustomScale {
    explicit AICustomScale(xjson::Parser &cfg): AI(cfg), AnalogCustomScale(cfg) {}
};

/// @brief base class for analog channels that can have a custom scale applied.
struct AOCustomScale : AO, AnalogCustomScale {
    explicit AOCustomScale(xjson::Parser &cfg): AO(cfg), AnalogCustomScale(cfg) {}
};

struct AIVoltage : AICustomScale {
    const int32_t terminal_config = 0;

    explicit AIVoltage(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        terminal_config(parse_terminal_config(cfg)) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIVoltageChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->terminal_config,
            this->min_val,
            this->max_val,
            this->units,
            scale_key
        );
    }
};

struct AIVoltageRMS final : AIVoltage {
    explicit AIVoltageRMS(xjson::Parser &cfg): Analog(cfg), Base(cfg), AIVoltage(cfg) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIVoltageRMSChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->terminal_config,
            this->min_val,
            this->max_val,
            this->units,
            scale_key
        );
    }
};

struct AIVoltageWithExcit final : AIVoltage {
    const int32_t bridge_config;
    const ExcitationConfig excitation_config;

    explicit AIVoltageWithExcit(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AIVoltage(cfg),
        bridge_config(parse_bridge_config(cfg)),
        excitation_config(cfg, VOLT_EXCIT_PREFIX) {}

    ~AIVoltageWithExcit() override = default;

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIVoltageChanWithExcit(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->terminal_config,
            this->min_val,
            this->max_val,
            this->units,
            this->bridge_config,
            this->excitation_config.source,
            this->excitation_config.val,
            static_cast<bool32>(this->excitation_config.min_val_for_excitation),
            scale_key
        );
    }
};

struct AICurrent : AICustomScale {
    const int32_t shunt_resistor_loc;
    const double ext_shunt_resistor_val;
    const int32 terminal_config;

    static int32_t get_shunt_resistor_loc(const std::string &loc) {
        if (loc == "External") return DAQmx_Val_External;
        if (loc == "Internal") return DAQmx_Val_Internal;
        return DAQmx_Val_Default;
    }

    explicit AICurrent(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        shunt_resistor_loc(
            get_shunt_resistor_loc(cfg.required<std::string>("shunt_resistor_loc"))
        ),
        ext_shunt_resistor_val(cfg.required<double>("ext_shunt_resistor_val")),
        terminal_config(parse_terminal_config(cfg)) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAICurrentChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->terminal_config,
            this->min_val,
            this->max_val,
            this->units,
            this->shunt_resistor_loc,
            this->ext_shunt_resistor_val,
            scale_key
        );
    }
};

struct AICurrentRMS final : AICurrent {
    explicit AICurrentRMS(xjson::Parser &cfg): Analog(cfg), Base(cfg), AICurrent(cfg) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAICurrentRMSChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->terminal_config,
            this->min_val,
            this->max_val,
            this->units,
            this->shunt_resistor_loc,
            this->ext_shunt_resistor_val,
            scale_key
        );
    }
};

struct AIRTD final : AI {
    const int32_t rtd_type;
    const int32_t resistance_config;
    const ExcitationConfig excitation_config;
    const double r0;

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

    explicit AIRTD(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AI(cfg),
        rtd_type(get_rtd_type(cfg.required<std::string>("rtd_type"))),
        resistance_config(parse_resistance_config(cfg)),
        excitation_config(cfg, CURR_EXCIT_PREFIX),
        r0(cfg.required<double>("r0")) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle
    ) const override {
        return dmx->CreateAIRTDChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->min_val,
            this->max_val,
            this->units,
            this->rtd_type,
            this->resistance_config,
            this->excitation_config.source,
            this->excitation_config.val,
            this->r0
        );
    }
};

struct AIThermocouple final : AI {
    const int32_t thermocouple_type;
    const int32_t cjc_source;
    const double cjc_val;
    std::string cjc_port;

    [[nodiscard]] int32_t static parse_type(xjson::Parser &cfg) {
        const auto type = cfg.required<std::string>("thermocouple_type");
        if (type == "J") return DAQmx_Val_J_Type_TC;
        if (type == "K") return DAQmx_Val_K_Type_TC;
        if (type == "N") return DAQmx_Val_N_Type_TC;
        if (type == "R") return DAQmx_Val_R_Type_TC;
        if (type == "S") return DAQmx_Val_S_Type_TC;
        if (type == "T") return DAQmx_Val_T_Type_TC;
        if (type == "B") return DAQmx_Val_B_Type_TC;
        if (type == "E") return DAQmx_Val_E_Type_TC;
        cfg.field_err("thermocouple_type", "invalid thermocouple type:" + type);
        return DAQmx_Val_J_Type_TC;
    }

    [[nodiscard]] int32_t static parse_cjc_source(xjson::Parser &cfg) {
        const auto source = cfg.required<std::string>("cjc_source");
        if (source == "BuiltIn") return DAQmx_Val_BuiltIn;
        if (source == "ConstVal") return DAQmx_Val_ConstVal;
        if (source == "Chan") return DAQmx_Val_Chan;
        cfg.field_err("cjc_source", "invalid thermocouple cjc source: " + source);
        return DAQmx_Val_BuiltIn;
    }

    explicit AIThermocouple(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AI(cfg),
        thermocouple_type(parse_type(cfg)),
        cjc_source(parse_cjc_source(cfg)),
        cjc_val(cfg.optional<double>("cjc_val", 0)),
        cjc_port(format_cjc_port(this->cfg_path, cfg.optional<int32_t>("cjc_port", 0))
        ) {
        this->cjc_port = format_cjc_port(
            this->cfg_path,
            cfg.optional<int32_t>("cjc_port", 0)
        );
    }

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle
    ) const override {
        return dmx->CreateAIThrmcplChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->min_val,
            this->max_val,
            this->units,
            this->thermocouple_type,
            this->cjc_source,
            this->cjc_val,
            this->cjc_port.c_str()
        );
    }
};

struct AITempBuiltIn final : AI {
    explicit AITempBuiltIn(xjson::Parser &cfg): Analog(cfg), Base(cfg), AI(cfg) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle
    ) const override {
        const auto i_name = this->dev_loc + "/_boardTempSensor_vs_aignd";
        return dmx->CreateAITempBuiltInSensorChan(
            task_handle,
            i_name.c_str(),
            this->cfg_path.c_str(),
            this->units
        );
    }
};

struct AIThermistorIEX final : AI {
    const int32_t resistance_config;
    const ExcitationConfig excitation_config;
    const double a;
    const double b;
    const double c;

    explicit AIThermistorIEX(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AI(cfg),
        resistance_config(parse_resistance_config(cfg)),
        excitation_config(cfg, CURR_EXCIT_PREFIX),
        a(cfg.required<double>("a")),
        b(cfg.required<double>("b")),
        c(cfg.required<double>("c")) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle
    ) const override {
        return dmx->CreateAIThrmstrChanIex(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->min_val,
            this->max_val,
            this->units,
            this->resistance_config,
            this->excitation_config.source, // current excitation source FIXME
            this->excitation_config.val, // current excitation val FIXME
            this->a,
            this->b,
            this->c
        );
    }
};

struct AIThermistorVex final : AI {
    const int32_t resistance_config;
    const ExcitationConfig excitation_config;
    const double a;
    const double b;
    const double c;
    const double r1;

    explicit AIThermistorVex(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AI(cfg),
        resistance_config(parse_resistance_config(cfg)),
        excitation_config(cfg, VOLT_EXCIT_PREFIX),
        a(cfg.required<double>("a")),
        b(cfg.required<double>("b")),
        c(cfg.required<double>("c")),
        r1(cfg.required<double>("r1")) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle
    ) const override {
        return dmx->CreateAIThrmstrChanVex(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->min_val,
            this->max_val,
            this->units,
            this->resistance_config,
            this->excitation_config.source, // current excitation source FIXME
            this->excitation_config.val, // current excitation val FIXME
            this->a,
            this->b,
            this->c,
            this->r1
        );
    }
};

struct AIAccel : AICustomScale {
    const double sensitivity;
    const int32_t sensitivity_units;
    const ExcitationConfig excitation_config;
    const int32 terminal_config;

    explicit AIAccel(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        sensitivity(cfg.required<double>("sensitivity")),
        sensitivity_units(
            UNITS_MAP.at(cfg.optional<std::string>("sensitivity_units", "mVoltsPerG"))
        ),
        excitation_config(cfg, CURR_EXCIT_PREFIX),
        terminal_config(parse_terminal_config(cfg)) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIAccelChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->terminal_config,
            this->min_val,
            this->max_val,
            this->units,
            this->sensitivity,
            this->sensitivity_units,
            this->excitation_config.source,
            this->excitation_config.val,
            scale_key
        );
    }
};

struct AIAccel4WireDCVoltage final : AIAccel {
    explicit AIAccel4WireDCVoltage(xjson::Parser &cfg):
        Analog(cfg), Base(cfg), AIAccel(cfg) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIAccel4WireDCVoltageChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->terminal_config,
            this->min_val,
            this->max_val,
            this->units,
            this->sensitivity,
            this->sensitivity_units,
            this->excitation_config.source,
            this->excitation_config.val,
            this->excitation_config.use_excit_for_scaling,
            scale_key
        );
    }
};

struct AIAccelCharge final : AICustomScale {
    const double sensitivity;
    const int32_t sensitivity_units;
    const int32 terminal_config;

    explicit AIAccelCharge(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        sensitivity(cfg.required<double>("sensitivity")),
        sensitivity_units(UNITS_MAP.at(cfg.required<std::string>("sensitivity_units"))),
        terminal_config(parse_terminal_config(cfg)) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIAccelChargeChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->terminal_config,
            this->min_val,
            this->max_val,
            this->units,
            this->sensitivity,
            this->sensitivity_units,
            scale_key
        );
    }
};

struct AIResistance final : AICustomScale {
    const int32_t resistance_config;
    const ExcitationConfig excitation_config;

    explicit AIResistance(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        resistance_config(parse_resistance_config(cfg)),
        excitation_config(cfg, CURR_EXCIT_PREFIX) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIResistanceChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->min_val,
            this->max_val,
            this->units,
            this->resistance_config,
            this->excitation_config.source,
            this->excitation_config.val,
            scale_key
        );
    }
};

struct AIBridge final : AICustomScale {
    const BridgeConfig bridge_config;

    explicit AIBridge(xjson::Parser &cfg):
        Analog(cfg), Base(cfg), AICustomScale(cfg), bridge_config(cfg) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIBridgeChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->min_val,
            this->max_val,
            this->units,
            this->bridge_config.ni_bridge_config,
            this->bridge_config.voltage_excit_source,
            this->bridge_config.voltage_excit_val,
            this->bridge_config.nominal_bridge_resistance,
            scale_key
        );
    }
};

struct AIStrainGauge final : AICustomScale {
    const int32_t strain_config;
    const ExcitationConfig excitation_config;
    const double gage_factor;
    const double initial_bridge_voltage;
    const double nominal_gage_resistance;
    const double poisson_ratio;
    const double lead_wire_resistance;

    explicit AIStrainGauge(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        strain_config(get_strain_config(cfg.required<std::string>("strain_config"))),
        excitation_config(cfg, VOLT_EXCIT_PREFIX),
        gage_factor(cfg.required<double>("gage_factor")),
        initial_bridge_voltage(cfg.required<double>("initial_bridge_voltage")),
        nominal_gage_resistance(cfg.required<double>("nominal_gage_resistance")),
        poisson_ratio(cfg.required<double>("poisson_ratio")),
        lead_wire_resistance(cfg.required<double>("lead_wire_resistance")) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIStrainGageChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->min_val,
            this->max_val,
            this->units,
            this->strain_config,
            this->excitation_config.source,
            this->excitation_config.val,
            this->gage_factor,
            this->initial_bridge_voltage,
            this->nominal_gage_resistance,
            this->poisson_ratio,
            this->lead_wire_resistance,
            scale_key
        );
    }
};

struct AIRosetteStrainGauge final : AI {
    const int32_t rosette_type;
    const double gage_orientation;
    const int32 rosette_meas_type;
    const int32 strain_config;
    const ExcitationConfig excitation_config;
    const double gage_factor;
    const double nominal_gage_resistance;
    const double poisson_ratio;
    const double lead_wire_resistance;

    explicit AIRosetteStrainGauge(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AI(cfg),
        rosette_type(get_rosette_type(cfg.required<std::string>("rosette_type"))),
        gage_orientation(cfg.required<double>("gage_orientation")),
        rosette_meas_type(
            get_rosette_meas_type(cfg.required<std::string>("rosette_meas_type"))
        ),
        strain_config(get_strain_config(cfg.required<std::string>("strain_config"))),
        excitation_config(cfg, VOLT_EXCIT_PREFIX),
        gage_factor(cfg.required<double>("gage_factor")),
        nominal_gage_resistance(cfg.required<double>("nominal_gage_resistance")),
        poisson_ratio(cfg.required<double>("poisson_ratio")),
        lead_wire_resistance(cfg.required<double>("lead_wire_resistance")) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle
    ) const override {
        return dmx->CreateAIRosetteStrainGageChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->min_val,
            this->max_val,
            this->rosette_type,
            this->gage_orientation,
            &this->rosette_meas_type,
            1, // bynRosetteMeasTypes
            this->strain_config,
            this->excitation_config.source,
            this->excitation_config.val,
            this->gage_factor,
            this->nominal_gage_resistance,
            this->poisson_ratio,
            this->lead_wire_resistance
        );
    }
};

struct AIMicrophone final : AICustomScale {
    const double mic_sensitivity;
    const double max_snd_press_level;
    const ExcitationConfig excitation_config;
    const int32 terminal_config = 0;

    explicit AIMicrophone(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        mic_sensitivity(cfg.required<double>("mic_sensitivity")),
        max_snd_press_level(cfg.required<double>("max_snd_press_level")),
        excitation_config(cfg, CURR_EXCIT_PREFIX),
        terminal_config(parse_terminal_config(cfg)) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIMicrophoneChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->terminal_config,
            this->units,
            this->mic_sensitivity,
            this->max_snd_press_level,
            this->excitation_config.source,
            this->excitation_config.val,
            scale_key
        );
    }
};

struct AIFrequencyVoltage final : AICustomScale {
    const double threshold_level;
    const double hysteresis;

    explicit AIFrequencyVoltage(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        threshold_level(cfg.required<double>("threshold_level")),
        hysteresis(cfg.required<double>("hysteresis")) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        const auto port = this->dev_loc + "ctr" + std::to_string(this->port);
        return dmx->CreateAIFreqVoltageChan(
            task_handle,
            port.c_str(),
            this->cfg_path.c_str(),
            this->min_val,
            this->max_val,
            this->units,
            this->threshold_level,
            this->hysteresis,
            scale_key
        );
    }
};

struct AIPressureBridgeTwoPointLin final : AICustomScale {
    const BridgeConfig bridge_config;
    const TwoPointLinConfig two_point_lin_config;

    explicit AIPressureBridgeTwoPointLin(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        bridge_config(cfg),
        two_point_lin_config(cfg) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIPressureBridgeTwoPointLinChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->min_val,
            this->max_val,
            this->units,
            this->bridge_config.ni_bridge_config,
            this->bridge_config.voltage_excit_source,
            this->bridge_config.voltage_excit_val,
            this->bridge_config.nominal_bridge_resistance,
            this->two_point_lin_config.first_electrical_val,
            this->two_point_lin_config.second_electrical_val,
            this->two_point_lin_config.electrical_units,
            this->two_point_lin_config.first_physical_val,
            this->two_point_lin_config.second_physical_val,
            this->two_point_lin_config.physical_units,
            scale_key
        );
    }
};

struct AIPressureBridgeTable final : AICustomScale {
    const BridgeConfig bridge_config;
    const TableConfig table_config;

    explicit AIPressureBridgeTable(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        bridge_config(cfg),
        table_config(cfg) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIPressureBridgeTableChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->min_val,
            this->max_val,
            this->units,
            this->bridge_config.ni_bridge_config,
            this->bridge_config.voltage_excit_source,
            this->bridge_config.voltage_excit_val,
            this->bridge_config.nominal_bridge_resistance,
            this->table_config.electrical_vals,
            this->table_config.num_electrical_vals,
            this->table_config.electrical_units,
            this->table_config.physical_vals,
            this->table_config.num_physical_vals,
            this->table_config.physical_units,
            scale_key
        );
    }
};

struct AIPressureBridgePolynomial final : AICustomScale {
    const BridgeConfig bridge_config;
    const PolynomialConfig polynomial_config;

    explicit AIPressureBridgePolynomial(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        bridge_config(cfg),
        polynomial_config(cfg) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIPressureBridgePolynomialChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->min_val,
            this->max_val,
            this->units,
            this->bridge_config.ni_bridge_config,
            this->bridge_config.voltage_excit_source,
            this->bridge_config.voltage_excit_val,
            this->bridge_config.nominal_bridge_resistance,
            this->polynomial_config.forward_coeffs,
            this->polynomial_config.num_forward_coeffs,
            this->polynomial_config.reverse_coeffs,
            this->polynomial_config.num_reverse_coeffs,
            this->polynomial_config.electrical_units,
            this->polynomial_config.physical_units,
            scale_key
        );
    }
};

struct AIForceBridgePolynomial final : AICustomScale {
    const BridgeConfig bridge_config;
    const PolynomialConfig polynomial_config;

    explicit AIForceBridgePolynomial(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        bridge_config(cfg),
        polynomial_config(cfg) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIForceBridgePolynomialChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->min_val,
            this->max_val,
            this->units,
            this->bridge_config.ni_bridge_config,
            this->bridge_config.voltage_excit_source,
            this->bridge_config.voltage_excit_val,
            this->bridge_config.nominal_bridge_resistance,
            this->polynomial_config.forward_coeffs,
            this->polynomial_config.num_forward_coeffs,
            this->polynomial_config.reverse_coeffs,
            this->polynomial_config.num_reverse_coeffs,
            this->polynomial_config.electrical_units,
            this->polynomial_config.physical_units,
            scale_key
        );
    }
};

struct AIForceBridgeTable final : AICustomScale {
    const BridgeConfig bridge_config;
    const TableConfig table_config;

    explicit AIForceBridgeTable(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        bridge_config(cfg),
        table_config(cfg) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIForceBridgeTableChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->min_val,
            this->max_val,
            this->units,
            this->bridge_config.ni_bridge_config,
            this->bridge_config.voltage_excit_source,
            this->bridge_config.voltage_excit_val,
            this->bridge_config.nominal_bridge_resistance,
            this->table_config.electrical_vals,
            this->table_config.num_electrical_vals,
            this->table_config.electrical_units,
            this->table_config.physical_vals,
            this->table_config.num_physical_vals,
            this->table_config.physical_units,
            scale_key
        );
    }
};

struct AIForceBridgeTwoPointLin final : AICustomScale {
    BridgeConfig bridge_config;
    TwoPointLinConfig two_point_lin_config;

    explicit AIForceBridgeTwoPointLin(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        bridge_config(cfg),
        two_point_lin_config(cfg) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle,

        const char *scale_key
    ) const override {
        return dmx->CreateAIForceBridgeTwoPointLinChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->min_val,
            this->max_val,
            this->units,
            this->bridge_config.ni_bridge_config,
            this->bridge_config.voltage_excit_source,
            this->bridge_config.voltage_excit_val,
            this->bridge_config.nominal_bridge_resistance,
            this->two_point_lin_config.first_electrical_val,
            this->two_point_lin_config.second_electrical_val,
            this->two_point_lin_config.electrical_units,
            this->two_point_lin_config.first_physical_val,
            this->two_point_lin_config.second_physical_val,
            this->two_point_lin_config.physical_units,
            scale_key
        );
    }
};

struct AIVelocityIEPE final : AICustomScale {
    const int32_t sensitivity_units;
    const double sensitivity;
    const ExcitationConfig excitation_config;
    const int32_t terminal_config;

    explicit AIVelocityIEPE(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        sensitivity_units(parse_units(cfg, "sensitivity_units")),
        sensitivity(cfg.required<double>("sensitivity")),
        excitation_config(cfg, CURR_EXCIT_PREFIX),
        terminal_config(parse_terminal_config(cfg)) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIVelocityIEPEChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->terminal_config,
            this->min_val,
            this->max_val,
            this->units,
            this->sensitivity,
            this->sensitivity_units,
            this->excitation_config.source,
            this->excitation_config.val,
            scale_key
        );
    }
};

struct AITorqueBridgeTwoPointLin final : AICustomScale {
    const BridgeConfig bridge_config;
    const TwoPointLinConfig two_point_lin_config;

    explicit AITorqueBridgeTwoPointLin(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        bridge_config(cfg),
        two_point_lin_config(cfg) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAITorqueBridgeTwoPointLinChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->min_val,
            this->max_val,
            this->units,
            this->bridge_config.ni_bridge_config,
            this->bridge_config.voltage_excit_source,
            this->bridge_config.voltage_excit_val,
            this->bridge_config.nominal_bridge_resistance,
            this->two_point_lin_config.first_electrical_val,
            this->two_point_lin_config.second_electrical_val,
            this->two_point_lin_config.electrical_units,
            this->two_point_lin_config.first_physical_val,
            this->two_point_lin_config.second_physical_val,
            this->two_point_lin_config.physical_units,
            scale_key
        );
    }
};

struct AITorqueBridgePolynomial final : AICustomScale {
    const BridgeConfig bridge_config;
    const PolynomialConfig polynomial_config;

    explicit AITorqueBridgePolynomial(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        bridge_config(cfg),
        polynomial_config(cfg) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAITorqueBridgePolynomialChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->min_val,
            this->max_val,
            this->units,
            this->bridge_config.ni_bridge_config,
            this->bridge_config.voltage_excit_source,
            this->bridge_config.voltage_excit_val,
            this->bridge_config.nominal_bridge_resistance,
            this->polynomial_config.forward_coeffs,
            this->polynomial_config.num_forward_coeffs,
            this->polynomial_config.reverse_coeffs,
            this->polynomial_config.num_reverse_coeffs,
            this->polynomial_config.electrical_units,
            this->polynomial_config.physical_units,
            scale_key
        );
    }
};

struct AITorqueBridgeTable final : AICustomScale {
    const BridgeConfig bridge_config;
    const TableConfig table_config;

    explicit AITorqueBridgeTable(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        bridge_config(cfg),
        table_config(cfg) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAITorqueBridgeTableChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->min_val,
            this->max_val,
            this->units,
            this->bridge_config.ni_bridge_config,
            this->bridge_config.voltage_excit_source,
            this->bridge_config.voltage_excit_val,
            this->bridge_config.nominal_bridge_resistance,
            this->table_config.electrical_vals,
            this->table_config.num_electrical_vals,
            this->table_config.electrical_units,
            this->table_config.physical_vals,
            this->table_config.num_physical_vals,
            this->table_config.physical_units,
            scale_key
        );
    }
};

struct AIForceIEPE final : AICustomScale {
    const int32_t sensitivity_units;
    const double sensitivity;
    const ExcitationConfig excitation_config;
    const int32 terminal_config;

    explicit AIForceIEPE(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        sensitivity_units(parse_units(cfg, "sensitivity_units")),
        sensitivity(cfg.required<double>("sensitivity")),
        excitation_config(cfg, CURR_EXCIT_PREFIX),
        terminal_config(parse_terminal_config(cfg)) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIForceIEPEChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->terminal_config,
            this->min_val,
            this->max_val,
            this->units,
            this->sensitivity,
            this->sensitivity_units,
            this->excitation_config.source,
            this->excitation_config.val,
            scale_key
        );
    }
};

struct AICharge final : AICustomScale {
    const int32 terminal_config;

    explicit AICharge(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        terminal_config(parse_terminal_config(cfg)) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIChargeChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->terminal_config,
            this->min_val,
            this->max_val,
            this->units,
            scale_key
        );
    }
};

struct AOVoltage final : AOCustomScale {
    explicit AOVoltage(xjson::Parser &cfg):
        Analog(cfg), Base(cfg), AOCustomScale(cfg) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAOVoltageChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->min_val,
            this->max_val,
            this->units,
            scale_key
        );
    }
};

struct AOCurrent final : AOCustomScale {
    explicit AOCurrent(xjson::Parser &cfg):
        Analog(cfg), Base(cfg), AOCustomScale(cfg) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAOCurrentChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->min_val,
            this->max_val,
            this->units,
            scale_key
        );
    }
};

struct AOFunctionGenerator final : AO {
    const double frequency;
    const double amplitude;
    const double offset;
    const int32 wave_type;

    int32_t static get_type(const std::string &type, const xjson::Parser &cfg) {
        if (type == "Sine") return DAQmx_Val_Sine;
        if (type == "Triangle") return DAQmx_Val_Triangle;
        if (type == "Square") return DAQmx_Val_Square;
        if (type == "Sawtooth") return DAQmx_Val_Sawtooth;
        cfg.field_err("", "invalid wave type: " + type);
        return DAQmx_Val_Sine;
    }


    explicit AOFunctionGenerator(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AO(cfg),
        frequency(cfg.required<double>("frequency")),
        amplitude(cfg.required<double>("amplitude")),
        offset(cfg.required<double>("offset")),
        wave_type(get_type(cfg.required<std::string>("wave_type"), cfg)) {}

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle
    ) const override {
        return dmx->CreateAOFuncGenChan(
            task_handle,
            this->loc().c_str(),
            this->cfg_path.c_str(),
            this->wave_type,
            this->frequency,
            this->amplitude,
            this->offset
        );
    }
};

template<typename T>
using Factory = std::function<std::unique_ptr<T>(xjson::Parser &cfg)>;

#define INPUT_CHAN_FACTORY(type, class)                                                \
    {                                                                                  \
        type, [](xjson::Parser &cfg) { return std::make_unique<class>(cfg); }          \
    }

static const std::map<std::string, Factory<Output>> OUTPUTS = {
    INPUT_CHAN_FACTORY("ao_current", AOCurrent),
    INPUT_CHAN_FACTORY("ao_voltage", AOVoltage),
    INPUT_CHAN_FACTORY("ao_func_gen", AOFunctionGenerator),
    INPUT_CHAN_FACTORY("digital_output", DO)
};

static const std::map<std::string, Factory<Input>> INPUTS = {
    INPUT_CHAN_FACTORY("ai_accel", AIAccel),
    INPUT_CHAN_FACTORY("ai_accel_4_wire_dc_voltage", AIAccel4WireDCVoltage),
    INPUT_CHAN_FACTORY("ai_bridge", AIBridge),
    INPUT_CHAN_FACTORY("ai_charge", AICharge),
    INPUT_CHAN_FACTORY("ai_current", AICurrent),
    INPUT_CHAN_FACTORY("ai_force_bridge_polynomial", AIForceBridgePolynomial),
    INPUT_CHAN_FACTORY("ai_force_bridge_table", AIForceBridgeTable),
    INPUT_CHAN_FACTORY("ai_force_bridge_two_point_lin", AIForceBridgeTwoPointLin),
    INPUT_CHAN_FACTORY("ai_force_iepe", AIForceIEPE),
    INPUT_CHAN_FACTORY("ai_microphone", AIMicrophone),
    INPUT_CHAN_FACTORY("ai_pressure_bridge_polynomial", AIPressureBridgePolynomial),
    INPUT_CHAN_FACTORY("ai_pressure_bridge_table", AIPressureBridgeTable),
    INPUT_CHAN_FACTORY("ai_pressure_bridge_two_point_lin", AIPressureBridgeTwoPointLin),
    INPUT_CHAN_FACTORY("ai_resistance", AIResistance),
    INPUT_CHAN_FACTORY("ai_rtd", AIRTD),
    INPUT_CHAN_FACTORY("ai_strain_gauge", AIStrainGauge),
    INPUT_CHAN_FACTORY("ai_temp_builtin", AITempBuiltIn),
    INPUT_CHAN_FACTORY("ai_thermocouple", AIThermocouple),
    INPUT_CHAN_FACTORY("ai_torque_bridge_polynomial", AITorqueBridgePolynomial),
    INPUT_CHAN_FACTORY("ai_torque_bridge_table", AITorqueBridgeTable),
    INPUT_CHAN_FACTORY("ai_torque_bridge_two_point_lin", AITorqueBridgeTwoPointLin),
    INPUT_CHAN_FACTORY("ai_velocity_iepe", AIVelocityIEPE),
    INPUT_CHAN_FACTORY("ai_voltage", AIVoltage),
    INPUT_CHAN_FACTORY("ai_frequency_voltage", AIFrequencyVoltage),
    INPUT_CHAN_FACTORY("digital_input", DI)
};

inline std::unique_ptr<Input> parse_input(xjson::Parser &cfg) {
    const auto type = cfg.required<std::string>("type");
    const auto input = INPUTS.find(type);
    if (input != INPUTS.end()) return input->second(cfg);
    cfg.field_err("type", "unknown channel type: " + type);
    return nullptr;
}

inline std::unique_ptr<Output> parse_output(xjson::Parser &cfg) {
    const auto type = cfg.required<std::string>("type");
    const auto output = OUTPUTS.find(type);
    if (output != OUTPUTS.end()) return output->second(cfg);
    return nullptr;
}

#undef INPUT_CHAN_FACTORY
#undef FACTORY
#undef FACTORY_WITH_CJC_SOURCES
}
