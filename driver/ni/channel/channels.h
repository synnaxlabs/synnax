// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
#include "x/cpp/xjson/xjson.h"
#include "client/cpp/synnax.h"

/// internal
#include "driver/ni/daqmx/sugared.h"
#include "driver/ni/channel/scale.h"
#include "driver/ni/channel/units.h"

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

struct ExcitationConfig {
    int32_t excit_source;
    double excit_val;
    double min_val_for_excitation; // optional
    double max_val_for_excitation; //optional
    bool32 use_excit_for_scaling; //optional

    explicit ExcitationConfig(xjson::Parser &cfg, const std::string &prefix)
        : excit_source(
              get_excitation_src(cfg.required<std::string>(prefix + "_excit_source"))),
          excit_val(cfg.required<double>(prefix + "_excit_val")),
          min_val_for_excitation(cfg.optional<double>("min_val_for_excitation", 0)),
          max_val_for_excitation(cfg.optional<double>("max_val_for_excitation", 0)),
          use_excit_for_scaling(cfg.optional<bool32>("use_excit_for_scaling", 0)) {
    }
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
            get_excitation_src(cfg.required<std::string>("voltage_excit_source"))),
        voltage_excit_val(cfg.required<double>("voltage_excit_val")),
        nominal_bridge_resistance(cfg.required<double>("nominal_bridge_resistance")) {
    }
};

struct PolynomialConfig {
    float64 *forward_coeffs;
    const uint32_t num_forward_coeffs;
    float64 *reverse_coeffs;
    const uint32_t num_reverse_coeffs;
    int32_t electrical_units;
    int32_t physical_units;

    explicit PolynomialConfig(xjson::Parser &cfg)
        : num_forward_coeffs(cfg.required<uint32_t>("num_forward_coeffs")),
          num_reverse_coeffs(cfg.required<uint32_t>("num_reverse_coeffs")) {
        const auto eu = cfg.required<std::string>("electrical_units");
        const auto pu = cfg.required<std::string>("physical_units");

        const auto ni_eu = channel::UNITS_MAP.find(eu);
        if (ni_eu == channel::UNITS_MAP.end()) electrical_units = DAQmx_Val_Volts;
        else electrical_units = ni_eu->second;

        const auto ni_pu = channel::UNITS_MAP.find(pu);
        if (ni_pu == channel::UNITS_MAP.end()) physical_units = DAQmx_Val_Volts;
        else physical_units = channel::UNITS_MAP.at(pu);
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

        // TODO: figure out why using vector and .data() throws exception when passed to
        // NI function
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

    explicit TwoPointLinConfig(xjson::Parser &cfg)
        : first_electrical_val(cfg.required<double>("first_electrical_val")),
          second_electrical_val(cfg.required<double>("second_electrical_val")),
          electrical_units(UNITS_MAP.at(cfg.required<std::string>("electrical_units"))),
          first_physical_val(cfg.required<double>("first_physical_val")),
          second_physical_val(cfg.required<double>("second_physical_val")),
          physical_units(UNITS_MAP.at(cfg.required<std::string>("physical_units"))) {
        const auto eu = cfg.required<std::string>("electrical_units");
    }
};

struct Base {
    const bool enabled;
    const std::string dev_key;
    const std::string cfg_path;
    std::string dev;

    virtual ~Base() = default;

    explicit Base(xjson::Parser &cfg):
        enabled(cfg.optional<bool>("enabled", true)),
        dev_key(cfg.optional<std::string>("device", "")),
        cfg_path(cfg.path_prefix) {
    }

    virtual xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle
    ) const = 0;
};

struct Input : virtual Base {
    const synnax::ChannelKey synnax_key;
    synnax::Channel ch;

    explicit Input(xjson::Parser &cfg):
        Base(cfg),
        synnax_key(cfg.required<synnax::ChannelKey>("channel")) {
    }

    void bind_remote_info(const synnax::Channel &ch, const std::string &dev) {
        this->ch = ch;
        this->dev = dev;
    }
};

struct Output : virtual Base {
    const synnax::ChannelKey cmd_ch_key;
    const synnax::ChannelKey state_ch_key;
    const size_t index = 0;
    synnax::Channel state_ch;

    explicit Output(xjson::Parser &cfg):
        Base(cfg),
        cmd_ch_key(cfg.required<synnax::ChannelKey>("cmd_channel")),
        state_ch_key(cfg.required<synnax::ChannelKey>("state_channel")) {
    }
};

struct Digital : virtual Base {
    const int port;
    const int line;

    explicit Digital(xjson::Parser &cfg):
        port(cfg.required<int>("port")),
        line(cfg.required<int>("line")) {
    }

    [[nodiscard]] std::string loc() const {
        return this->dev + "/port" + std::to_string(this->port) + "/line" +
               std::to_string(this->line);
    }
};

struct DI final : Digital, Input {
    explicit DI(xjson::Parser &cfg):
        Base(cfg),
        Digital(cfg),
        Input(cfg) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle
    ) const override {
        return dmx->CreateDIChan(
            task_handle,
            this->loc().c_str(),
            "",
            DAQmx_Val_ChanPerLine
        );
    }
};

struct DO final : Digital, Output {
    explicit DO(xjson::Parser &cfg): Base(cfg), Digital(cfg), Output(cfg) {
    }

    void bind_remote_info(const synnax::Channel &state_ch, const std::string &dev) {
        this->state_ch = state_ch;
        this->dev = dev;
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle
    ) const override {
        return dmx->CreateDOChan(
            task_handle,
            this->loc().c_str(),
            "",
            DAQmx_Val_ChanPerLine
        );
    }
};

class Analog : virtual Base {
public:
    const int port;
    const double min_val;
    const double max_val;
    int32_t units;

    int32_t static parse_units(xjson::Parser &cfg, const std::string &path) {
        const auto str_units = cfg.optional<std::string>(path, "Volts");
        const auto units = UNITS_MAP.find(str_units);
        if (units == UNITS_MAP.end())
            cfg.field_err(path, "invalid units: " + str_units);
        return units->second;
    }

    explicit Analog(xjson::Parser &cfg):
        port(cfg.required<int>("port")),
        min_val(cfg.optional<float>("min_val", 0)),
        max_val(cfg.optional<float>("max_val", 0)),
        units(channel::Analog::parse_units(cfg, "units")) {
    }
};

struct AnalogCustomScale : virtual Analog {
    std::unique_ptr<Scale> scale;

    explicit AnalogCustomScale(xjson::Parser &cfg):
        Analog(cfg),
        scale(parse_scale(cfg, "custom_scale")) {
        if (!this->scale->is_none()) units = DAQmx_Val_FromCustomScale;
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle
    ) const override {
        auto [scale_key, err] = this->scale->apply(dmx);
        if (err) return err;
        return this->apply(
            dmx,
            task_handle,
            scale_key.empty() ? nullptr : scale_key.c_str()
        );
    }

    virtual xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const = 0;
};

struct AI : virtual Analog, Input {
    explicit AI(xjson::Parser &cfg): Analog(cfg), Input(cfg) {
    }

    [[nodiscard]] std::string physical_channel() const {
        return this->dev + "/ai" + std::to_string(this->port);
    }
};

struct AO : virtual Analog, Output {
    explicit AO(xjson::Parser &cfg): Analog(cfg), Output(cfg) {
    }

    [[nodiscard]] std::string loc() const {
        return this->dev + "/ao" + std::to_string(this->port);
    }
};

struct AICustomScale : AI, AnalogCustomScale {
    explicit AICustomScale(xjson::Parser &cfg):
        AI(cfg),
        AnalogCustomScale(cfg) {
    }
};

struct AOCustomScale : AO, AnalogCustomScale {
    explicit AOCustomScale(xjson::Parser &cfg):
        AO(cfg),
        AnalogCustomScale(cfg) {
    }
};

struct AIVoltage : AICustomScale {
    const int32_t terminal_config = 0;

    explicit AIVoltage(xjson::Parser &cfg) :
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        terminal_config(parse_terminal_config(cfg)) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIVoltageChan(
            task_handle,
            this->physical_channel().c_str(),
            "", // name to assign channel
            this->terminal_config,
            this->min_val,
            this->max_val,
            this->units,
            scale_key
        );
    }
};

struct AIVoltageRMS final : AIVoltage {
    explicit AIVoltageRMS(xjson::Parser &cfg): Analog(cfg), Base(cfg), AIVoltage(cfg) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIVoltageRMSChan(
            task_handle,
            this->physical_channel().c_str(),
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

    explicit AIVoltageWithExcit(xjson::Parser &cfg) :
        Analog(cfg),
        Base(cfg),
        AIVoltage(cfg),
        bridge_config(parse_bridge_config(cfg)),
        excitation_config(cfg, VOLT_EXCIT_PREFIX) {
    }

    ~AIVoltageWithExcit() override = default;

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIVoltageChanWithExcit(
            task_handle,
            this->physical_channel().c_str(),
            this->cfg_path.c_str(),
            this->terminal_config,
            this->min_val,
            this->max_val,
            this->units,
            this->bridge_config,
            this->excitation_config.excit_source,
            this->excitation_config.excit_val,
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
            get_shunt_resistor_loc(cfg.required<std::string>("shunt_resistor_loc"))),
        ext_shunt_resistor_val(cfg.required<double>("ext_shunt_resistor_val")),
        terminal_config(parse_terminal_config(cfg)) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAICurrentChan(
            task_handle,
            this->physical_channel().c_str(),
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
    explicit AICurrentRMS(xjson::Parser &cfg) :
        Analog(cfg),
        Base(cfg),
        AICurrent(cfg) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAICurrentRMSChan(
            task_handle,
            this->physical_channel().c_str(),
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

    explicit AIRTD(xjson::Parser &cfg) :
        Analog(cfg),
        Base(cfg),
        AI(cfg),
        rtd_type(get_rtd_type(
            cfg.required<std::string>("rtd_type"))),
        resistance_config(
            parse_resistance_config(cfg)),
        excitation_config(cfg, CURR_EXCIT_PREFIX),
        r0(cfg.required<double>("r0")) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle
    ) const override {
        return dmx->CreateAIRTDChan(
            task_handle,
            this->physical_channel().c_str(),
            this->cfg_path.c_str(),
            this->min_val,
            this->max_val,
            this->units,
            this->rtd_type,
            this->resistance_config,
            this->excitation_config.excit_source,
            this->excitation_config.excit_val,
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

    explicit AIThermocouple(
        xjson::Parser &cfg,
        const std::map<std::int32_t, std::string> &cjc_sources
    ) : Analog(cfg),
        Base(cfg),
        AI(cfg),
        thermocouple_type(parse_type(cfg)),
        cjc_source(parse_cjc_source(cfg)),
        cjc_val(cfg.optional<double>("cjc_val", 0)) {
        const auto cjc_port = cfg.required<std::int32_t>("cjc_port");
        if (cjc_sources.find(cjc_port) == cjc_sources.end()) this->cjc_port = "";
        else this->cjc_port = cjc_sources.at(cjc_port);
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle
    ) const override {
        return dmx->CreateAIThrmcplChan(
            task_handle,
            this->physical_channel().c_str(),
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
    explicit AITempBuiltIn(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AI(cfg) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle
    ) const override {
        const auto i_name = this->dev + "/_boardTempSensor_vs_aignd";
        return dmx->CreateAITempBuiltInSensorChan(
            task_handle,
            i_name.c_str(),
            "",
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
        c(cfg.required<double>("c")) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle
    ) const override {
        return dmx->CreateAIThrmstrChanIex(
            task_handle,
            this->physical_channel().c_str(),
            this->cfg_path.c_str(),
            this->min_val,
            this->max_val,
            this->units,
            this->resistance_config,
            this->excitation_config.excit_source, // current excitation source FIXME
            this->excitation_config.excit_val, // current excitation val FIXME
            this->a,
            this->b,
            this->c
        );
    }
};

class AIThermistorVex final : public AI {
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
        r1(cfg.required<double>("r1")) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle
    ) const override {
        return dmx->CreateAIThrmstrChanVex(
            task_handle,
            this->physical_channel().c_str(),
            this->cfg_path.c_str(),
            this->min_val,
            this->max_val,
            this->units,
            this->resistance_config,
            this->excitation_config.excit_source, // current excitation source FIXME
            this->excitation_config.excit_val, // current excitation val FIXME
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
        terminal_config(parse_terminal_config(cfg)) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIAccelChan(
            task_handle,
            this->physical_channel().c_str(),
            this->cfg_path.c_str(),
            this->terminal_config,
            this->min_val,
            this->max_val,
            this->units,
            this->sensitivity,
            this->sensitivity_units,
            this->excitation_config.excit_source,
            this->excitation_config.excit_val,
            scale_key
        );
    }
};

struct AIAccel4WireDCVoltage final : AIAccel {
    explicit AIAccel4WireDCVoltage(xjson::Parser &cfg) :
        Analog(cfg),
        Base(cfg),
        AIAccel(cfg) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIAccel4WireDCVoltageChan(
            task_handle,
            this->physical_channel().c_str(),
            this->cfg_path.c_str(),
            this->terminal_config,
            this->min_val,
            this->max_val,
            this->units,
            this->sensitivity,
            this->sensitivity_units,
            this->excitation_config.excit_source,
            this->excitation_config.excit_val,
            this->excitation_config.use_excit_for_scaling,
            scale_key
        );
    }
};

class AIAccelCharge final : public AICustomScale {
    const double sensitivity;
    const int32_t sensitivity_units;
    const int32 terminal_config;

    explicit AIAccelCharge(xjson::Parser &cfg) :
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        sensitivity(cfg.required<double>("sensitivity")),
        sensitivity_units(UNITS_MAP.at(cfg.required<std::string>("sensitivity_units"))),
        terminal_config(parse_terminal_config(cfg)) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIAccelChargeChan(
            task_handle,
            this->physical_channel().c_str(),
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

class AIResistance final : public AICustomScale {
    const int32_t resistance_config;
    const ExcitationConfig excitation_config;

public:
    explicit AIResistance(xjson::Parser &cfg) :
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        resistance_config(parse_resistance_config(cfg)),
        excitation_config(cfg, CURR_EXCIT_PREFIX) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIResistanceChan(
            task_handle,
            this->physical_channel().c_str(),
            this->cfg_path.c_str(),
            this->min_val,
            this->max_val,
            this->units,
            this->resistance_config,
            this->excitation_config.excit_source,
            this->excitation_config.excit_val,
            scale_key
        );
    }
};

class AIBridge final : public AICustomScale {
public:
    BridgeConfig bridge_config;

    explicit AIBridge(xjson::Parser &cfg) :
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        bridge_config(cfg) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIBridgeChan(
            task_handle,
            this->physical_channel().c_str(),
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

    explicit AIStrainGauge(xjson::Parser &cfg) :
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        strain_config(get_strain_config(cfg.required<std::string>("strain_config"))),
        excitation_config(cfg, VOLT_EXCIT_PREFIX),
        gage_factor(cfg.required<double>("gage_factor")),
        initial_bridge_voltage(cfg.required<double>("initial_bridge_voltage")),
        nominal_gage_resistance(cfg.required<double>("nominal_gage_resistance")),
        poisson_ratio(cfg.required<double>("poisson_ratio")),
        lead_wire_resistance(cfg.required<double>("lead_wire_resistance")) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIStrainGageChan(
            task_handle,
            this->physical_channel().c_str(),
            this->cfg_path.c_str(),
            this->min_val,
            this->max_val,
            this->units,
            this->strain_config,
            this->excitation_config.excit_source,
            this->excitation_config.excit_val,
            this->gage_factor,
            this->initial_bridge_voltage,
            this->nominal_gage_resistance,
            this->poisson_ratio,
            this->lead_wire_resistance,
            scale_key
        );
    }
};

class AIRosetteStrainGauge final : public AI {
    const int32_t rosette_type;
    const double gage_orientation;
    const int32 rosette_meas_type;
    const int32 strain_config;
    const ExcitationConfig excitation_config;
    const double gage_factor;
    const double nominal_gage_resistance;
    const double poisson_ratio;
    const double lead_wire_resistance;

public:
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

    explicit AIRosetteStrainGauge(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AI(cfg),
        rosette_type(get_rosette_type(
            cfg.required<std::string>("rosette_type"))),
        gage_orientation(cfg.required<double>("gage_orientation")),
        rosette_meas_type(
            get_rosette_meas_type(cfg.required<std::string>("rosette_meas_type"))),
        strain_config(get_strain_config(cfg.required<std::string>("strain_config"))),
        excitation_config(cfg, VOLT_EXCIT_PREFIX),
        gage_factor(cfg.required<double>("gage_factor")),
        nominal_gage_resistance(cfg.required<double>("nominal_gage_resistance")),
        poisson_ratio(cfg.required<double>("poisson_ratio")),
        lead_wire_resistance(cfg.required<double>("lead_wire_resistance")) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle
    ) const override {
        return dmx->CreateAIRosetteStrainGageChan(
            task_handle,
            this->physical_channel().c_str(),
            this->cfg_path.c_str(),
            this->min_val,
            this->max_val,
            this->rosette_type,
            this->gage_orientation,
            &this->rosette_meas_type,
            1, // bynRosetteMeasTypes
            this->strain_config,
            this->excitation_config.excit_source,
            this->excitation_config.excit_val,
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

    explicit AIMicrophone(xjson::Parser &cfg) :
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        mic_sensitivity(cfg.required<double>("mic_sensitivity")),
        max_snd_press_level(cfg.required<double>("max_snd_press_level")),
        excitation_config(cfg, CURR_EXCIT_PREFIX),
        terminal_config(parse_terminal_config(cfg)) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIMicrophoneChan(
            task_handle,
            this->physical_channel().c_str(),
            this->cfg_path.c_str(),
            this->terminal_config,
            this->units,
            this->mic_sensitivity,
            this->max_snd_press_level,
            this->excitation_config.excit_source,
            this->excitation_config.excit_val,
            scale_key
        );
    }
};

class AIFrequencyVoltage final : public AICustomScale {
    double threshold_level;
    double hysteresis;

public:
    explicit AIFrequencyVoltage(xjson::Parser &cfg) :
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        threshold_level(cfg.required<double>("threshold_level")),
        hysteresis(cfg.required<double>("hysteresis")) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        const auto port = this->dev + "ctr" + std::to_string(this->port);
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

    explicit AIPressureBridgeTwoPointLin(xjson::Parser &cfg) :
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        bridge_config(cfg),
        two_point_lin_config(cfg) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIPressureBridgeTwoPointLinChan(
            task_handle,
            this->physical_channel().c_str(),
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

    explicit AIPressureBridgeTable(xjson::Parser &cfg) :
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        bridge_config(cfg),
        table_config(cfg) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIPressureBridgeTableChan(
            task_handle,
            this->physical_channel().c_str(),
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

class AIPressureBridgePolynomial final : public AICustomScale {
    const BridgeConfig bridge_config;
    const PolynomialConfig polynomial_config;

public:
    explicit AIPressureBridgePolynomial(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        bridge_config(cfg),
        polynomial_config(cfg) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIPressureBridgePolynomialChan(
            task_handle,
            this->physical_channel().c_str(),
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

class AIForceBridgePolynomial final : public AICustomScale {
    const BridgeConfig bridge_config;
    const PolynomialConfig polynomial_config;

public:
    explicit AIForceBridgePolynomial(xjson::Parser &cfg):
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        bridge_config(cfg),
        polynomial_config(cfg) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIForceBridgePolynomialChan(
            task_handle,
            this->physical_channel().c_str(),
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
        table_config(cfg) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIForceBridgeTableChan(
            task_handle,
            this->physical_channel().c_str(),
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
        two_point_lin_config(cfg) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle,

        const char *scale_key
    ) const override {
        return dmx->CreateAIForceBridgeTwoPointLinChan(
            task_handle,
            this->physical_channel().c_str(),
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
        sensitivity_units(
            channel::AIVelocityIEPE::parse_units(cfg, "sensitivity_units")),
        sensitivity(
            cfg.required<double>(
                "sensitivity")),
        excitation_config(
            cfg, CURR_EXCIT_PREFIX),
        terminal_config(
            parse_terminal_config(cfg)) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIVelocityIEPEChan(
            task_handle,
            this->physical_channel().c_str(),
            this->cfg_path.c_str(),
            this->terminal_config,
            this->min_val,
            this->max_val,
            this->units,
            this->sensitivity,
            this->sensitivity_units,
            this->excitation_config.excit_source,
            this->excitation_config.excit_val,
            scale_key
        );
    }
};

struct AITorqueBridgeTwoPointLin final : AICustomScale {
    const BridgeConfig bridge_config;
    const TwoPointLinConfig two_point_lin_config;

    explicit AITorqueBridgeTwoPointLin(xjson::Parser &cfg) :
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        bridge_config(cfg),
        two_point_lin_config(cfg) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAITorqueBridgeTwoPointLinChan(
            task_handle,
            this->physical_channel().c_str(),
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

    explicit AITorqueBridgePolynomial(xjson::Parser &cfg) :
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        bridge_config(cfg),
        polynomial_config(cfg) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAITorqueBridgePolynomialChan(
            task_handle,
            this->physical_channel().c_str(),
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

    explicit AITorqueBridgeTable(xjson::Parser &cfg) :
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        bridge_config(cfg),
        table_config(cfg) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAITorqueBridgeTableChan(
            task_handle,
            this->physical_channel().c_str(),
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

    explicit AIForceIEPE(xjson::Parser &cfg) :
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        sensitivity_units(channel::AIForceIEPE::parse_units(cfg, "sensitivity_units")),
        sensitivity(cfg.required<double>("sensitivity")),
        excitation_config(cfg, CURR_EXCIT_PREFIX),
        terminal_config(parse_terminal_config(cfg)) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIForceIEPEChan(
            task_handle,
            this->physical_channel().c_str(),
            this->cfg_path.c_str(),
            this->terminal_config,
            this->min_val,
            this->max_val,
            this->units,
            this->sensitivity,
            this->sensitivity_units,
            this->excitation_config.excit_source,
            this->excitation_config.excit_val,
            scale_key
        );
    }
};

struct AICharge final : AICustomScale {
    const int32 terminal_config;

    explicit AICharge(xjson::Parser &cfg) :
        Analog(cfg),
        Base(cfg),
        AICustomScale(cfg),
        terminal_config(parse_terminal_config(cfg)) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle,
        const char *scale_key
    ) const override {
        return dmx->CreateAIChargeChan(
            task_handle,
            this->physical_channel().c_str(),
            "",
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
        Analog(cfg),
        Base(cfg),
        AOCustomScale(cfg) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
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
    explicit AOCurrent(xjson::Parser &cfg) :
        Analog(cfg),
        Base(cfg),
        AOCustomScale(cfg) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
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


    explicit AOFunctionGenerator(xjson::Parser &cfg) :
        Analog(cfg),
        Base(cfg),
        AO(cfg),
        frequency(cfg.required<double>("frequency")),
        amplitude(cfg.required<double>("amplitude")),
        offset(cfg.required<double>("offset")),
        wave_type(get_type(cfg.required<std::string>("wave_type"), cfg)) {
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
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

using AIFactory = std::function<std::unique_ptr<AI>(
        xjson::Parser &cfg,
        const std::map<int32_t, std::string> &port_to_channel)
>;
using AOFactory = std::function<std::unique_ptr<AO>(xjson::Parser &cfg)>;

#define AO_FACTORY(type, class) \
    {type, [](xjson::Parser& cfg) { return std::make_unique<class>(cfg); }}

#define AI_FACTORY(type, class) \
    {type, [](xjson::Parser& cfg, const auto& ptc) { return std::make_unique<class>(cfg); }}

#define AI_FACTORY_WITH_PORT(type, class) \
    {type, [](xjson::Parser& cfg, const auto& ptc) { return std::make_unique<class>(cfg, ptc); }}

static const std::map<std::string, AOFactory> AO_CHANS = {
    AO_FACTORY("ao_current", AOCurrent),
    AO_FACTORY("ao_voltage", AOVoltage),
    AO_FACTORY("ao_func_gen", AOFunctionGenerator)
};

static const std::map<std::string, AIFactory> AI_CHANS = {
    AI_FACTORY("ai_accel", AIAccel),
    AI_FACTORY("ai_accel_4_wire_dc_voltage", AIAccel4WireDCVoltage),
    AI_FACTORY("ai_bridge", AIBridge),
    AI_FACTORY("ai_charge", AICharge),
    AI_FACTORY("ai_current", AICurrent),
    AI_FACTORY("ai_force_bridge_polynomial", AIForceBridgePolynomial),
    AI_FACTORY("ai_force_bridge_table", AIForceBridgeTable),
    AI_FACTORY("ai_force_bridge_two_point_lin", AIForceBridgeTwoPointLin),
    AI_FACTORY("ai_force_iepe", AIForceIEPE),
    AI_FACTORY("ai_microphone", AIMicrophone),
    AI_FACTORY("ai_pressure_bridge_polynomial", AIPressureBridgePolynomial),
    AI_FACTORY("ai_pressure_bridge_table", AIPressureBridgeTable),
    AI_FACTORY("ai_pressure_bridge_two_point_lin", AIPressureBridgeTwoPointLin),
    AI_FACTORY("ai_resistance", AIResistance),
    AI_FACTORY("ai_rtd", AIRTD),
    AI_FACTORY("ai_strain_gauge", AIStrainGauge),
    AI_FACTORY("ai_temp_builtin", AITempBuiltIn),
    AI_FACTORY_WITH_PORT("ai_thermocouple", AIThermocouple),
    AI_FACTORY("ai_torque_bridge_polynomial", AITorqueBridgePolynomial),
    AI_FACTORY("ai_torque_bridge_table", AITorqueBridgeTable),
    AI_FACTORY("ai_torque_bridge_two_point_lin", AITorqueBridgeTwoPointLin),
    AI_FACTORY("ai_velocity_iepe", AIVelocityIEPE),
    AI_FACTORY("ai_voltage", AIVoltage)
};

inline std::unique_ptr<Input> parse_input(
    xjson::Parser &cfg,
    const std::map<int32_t, std::string> &port_to_channel
) {
    const auto type = cfg.required<std::string>("type");
    if (AI_CHANS.count(type) == 0)
        cfg.field_err("type", "invalid analog input channel type: " + type);
    return AI_CHANS.at(type)(cfg, port_to_channel);
}

inline std::unique_ptr<Output> parse_output(xjson::Parser &cfg) {
    const auto type = cfg.required<std::string>("type");
    if (AO_CHANS.count(type) == 0)
        cfg.field_err("type", "invalid analog output channel type: " + type);
    return AO_CHANS.at(type)(cfg);
}

#undef AO_FACTORY
#undef AI_FACTORY
#undef AI_FACTORY_WITH_PORT
};
