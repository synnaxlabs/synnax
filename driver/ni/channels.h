// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <map>
#include <string>

/// module
#include "x/cpp/xjson/xjson.h"
#include "client/cpp/synnax.h"

/// internal
#include "driver/ni/daqmx/sugared.h"
#include "driver/ni/scale.h"
#include "driver/ni/util.h"

using string = std::string;

namespace ni {
static int32_t parse_terminal_config(xjson::Parser &p) {
    const auto s = p.required<string>("terminal_config");
    if (s == "PseudoDiff") return DAQmx_Val_PseudoDiff;
    if (s == "Diff") return DAQmx_Val_Diff;
    if (s == "NRSE") return DAQmx_Val_NRSE;
    if (s == "RSE") return DAQmx_Val_RSE;
    return DAQmx_Val_Cfg_Default;
}

static int32_t parse_bridge_config(xjson::Parser &p) {
    const auto s = p.required<string>("bridge_config");
    if (s == "FullBridge") return DAQmx_Val_FullBridge;
    if (s == "HalfBridge") return DAQmx_Val_HalfBridge;
    if (s == "QuarterBridge") return DAQmx_Val_QuarterBridge;
    return DAQmx_Val_FullBridge;
}

static int32_t parse_resistance_config(xjson::Parser &p) {
    const auto s = p.required<string>("resistance_config");
    if (s == "2Wire") return DAQmx_Val_2Wire;
    if (s == "3Wire") return DAQmx_Val_3Wire;
    if (s == "4Wire") return DAQmx_Val_4Wire;
    return DAQmx_Val_2Wire;
}

static int32_t get_excitation_src(const string &s) {
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

    explicit ExcitationConfig(xjson::Parser &cfg, const string &prefix)
        : excit_source(
              get_excitation_src(cfg.required<string>(prefix + "_excit_source"))),
          excit_val(cfg.required<double>(prefix + "_excit_val")),
          min_val_for_excitation(cfg.optional<double>("min_val_for_excitation", 0)),
          max_val_for_excitation(cfg.optional<double>("max_val_for_excitation", 0)),
          use_excit_for_scaling(cfg.optional<bool32>("use_excit_for_scaling", 0)) {
    }
};

const string CURR_EXCIT_PREFIX = "current";
const string VOLT_EXCIT_PREFIX = "voltage";

struct BridgeConfig {
    int32_t ni_bridge_config;
    int32_t voltage_excit_source;
    double voltage_excit_val;
    double nominal_bridge_resistance;

    BridgeConfig() = default;

    explicit BridgeConfig(xjson::Parser &cfg)
        : ni_bridge_config(parse_bridge_config(cfg)),
          voltage_excit_source(
              get_excitation_src(cfg.required<string>("voltage_excit_source"))),
          voltage_excit_val(cfg.required<double>("voltage_excit_val")),
          nominal_bridge_resistance(cfg.required<double>("nominal_bridge_resistance")) {
    }
};

struct PolynomialConfig {
    float64 *forward_coeffs;
    uint32_t num_forward_coeffs;
    float64 *reverse_coeffs;
    uint32_t num_reverse_coeffs;
    int32_t electrical_units;
    int32_t physical_units;

    PolynomialConfig() = default;

    explicit PolynomialConfig(xjson::Parser &cfg)
        : num_forward_coeffs(cfg.required<uint32_t>("num_forward_coeffs")),
          num_reverse_coeffs(cfg.required<uint32_t>("num_reverse_coeffs")) {
        const auto eu = cfg.required<string>("electrical_units");
        const auto pu = cfg.required<string>("physical_units");

        if (ni::UNITS_MAP.find(eu) == ni::UNITS_MAP.end())
            electrical_units = DAQmx_Val_Volts;
        else electrical_units = ni::UNITS_MAP.at(eu);

        if (ni::UNITS_MAP.find(pu) == ni::UNITS_MAP.end())
            physical_units = DAQmx_Val_Volts;
        else physical_units = ni::UNITS_MAP.at(pu);

        json j = cfg.get_json();

        forward_coeffs = new double[num_forward_coeffs];
        reverse_coeffs = new double[num_reverse_coeffs];

        const auto f = cfg.required_vec<double>("forward_coeffs");

        //get forward coeffs (prescale -> scale)
        for (uint32_t i = 0; i < num_forward_coeffs; i++)
            forward_coeffs[i] = f[i];

        // dmx->CalculateReversePolyCo eff(
        //     forward_coeffs,
        //     num_forward_coeffs,
        //     -1000, //FIXME don't hard code
        //     1000, //FIXME don't hard code
        //     num_reverse_coeffs,
        //     -1,
        //     reverse_coeffs
        // ); // FIXME: reversePoly order should be user inputted?
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
        const auto eu = cfg.required<string>("electrical_units");
        const auto pu = cfg.required<string>("physical_units");

        electrical_units = ni::UNITS_MAP.at(eu);
        physical_units = ni::UNITS_MAP.at(pu);

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
          electrical_units(UNITS_MAP.at(cfg.required<string>("electrical_units"))),
          first_physical_val(cfg.required<double>("first_physical_val")),
          second_physical_val(cfg.required<double>("second_physical_val")),
          physical_units(UNITS_MAP.at(cfg.required<string>("physical_units"))) {
        const auto eu = cfg.required<string>("electrical_units");
    }
};

struct Chan {
    const bool enabled;
    const std::string dev_key;
    const std::string cfg_path;
    std::string dev;

    virtual ~Chan() = default;

    explicit Chan(xjson::Parser &cfg):
        enabled(cfg.optional<bool>("enabled", true)),
        dev_key(cfg.optional<std::string>("device", "")),
        cfg_path(cfg.path_prefix) {
    }

    virtual xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle
    ) const = 0;
};

struct InputChan : virtual Chan {
    const synnax::ChannelKey synnax_key;
    synnax::Channel ch;

    explicit InputChan(xjson::Parser &cfg):
        Chan(cfg),
        synnax_key(cfg.required<synnax::ChannelKey>("channel")) {
    }

    void bind_remote_info(const synnax::Channel &ch, const std::string &dev) {
        this->ch = ch;
        this->dev = dev;
    }
};

struct OutputChan : virtual Chan {
    const synnax::ChannelKey cmd_ch_key;
    const synnax::ChannelKey state_ch_key;
    const size_t index = 0;
    synnax::Channel state_ch;

    explicit OutputChan(xjson::Parser &cfg):
        Chan(cfg),
        cmd_ch_key(cfg.required<synnax::ChannelKey>("cmd_channel")),
        state_ch_key(cfg.required<synnax::ChannelKey>("state_channel")) {
    }
};

struct DigitalChan : virtual Chan {
    const int port;
    const int line;

    explicit DigitalChan(xjson::Parser &cfg):
        port(cfg.required<int>("port")),
        line(cfg.required<int>("line")) {
    }

    [[nodiscard]] std::string loc() const {
        return this->dev + "/port" + std::to_string(this->port) + "/line" +
               std::to_string(this->line);
    }
};

struct DIChan final : DigitalChan, InputChan {
    explicit DIChan(xjson::Parser &cfg):
        Chan(cfg),
        DigitalChan(cfg),
        InputChan(cfg) {
    }

    void bind_remote_info(const synnax::Channel &ch, const std::string &dev) {
        this->ch = ch;
        this->dev = dev;
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

struct DOChan final : DigitalChan, OutputChan {
    explicit DOChan(xjson::Parser &cfg): Chan(cfg), DigitalChan(cfg), OutputChan(cfg) {
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

class AnalogChan : virtual Chan {
public:
    const int port;
    const double min_val;
    const double max_val;
    int32_t units;

    int32_t static parse_units(xjson::Parser &cfg, const std::string &path) {
        const auto str_units = cfg.optional<string>(path, "Volts");
        const auto units = UNITS_MAP.find(str_units);
        if (units == UNITS_MAP.end())
            cfg.field_err(path, "invalid units: " + str_units);
        return units->second;
    }

    explicit AnalogChan(xjson::Parser &cfg):
        port(cfg.required<int>("port")),
        min_val(cfg.optional<float>("min_val", 0)),
        max_val(cfg.optional<float>("max_val", 0)),
        units(ni::AnalogChan::parse_units(cfg, "units")) {
    }
};

struct AnalogChanCustomScale : virtual AnalogChan {
    std::unique_ptr<Scale> scale;

    explicit AnalogChanCustomScale(xjson::Parser &cfg):
        AnalogChan(cfg),
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

struct AIChan : virtual AnalogChan, InputChan {
    explicit AIChan(xjson::Parser &cfg):
        AnalogChan(cfg),
        InputChan(cfg) {
    }

    [[nodiscard]] std::string physical_channel() const {
        return this->dev + "/ai" + std::to_string(this->port);
    }
};

struct AOChan : virtual AnalogChan, OutputChan {
    explicit AOChan(xjson::Parser &cfg):
        AnalogChan(cfg),
        OutputChan(cfg) {
    }

    [[nodiscard]] std::string loc() const {
        return this->dev + "/ao" + std::to_string(this->port);
    }
};

struct AIChanCustomScale : AIChan, AnalogChanCustomScale {
    explicit AIChanCustomScale(xjson::Parser &cfg):
        AIChan(cfg),
        AnalogChanCustomScale(cfg) {
    }
};

struct AOChanCustomScale : AOChan, AnalogChanCustomScale {
    explicit AOChanCustomScale(xjson::Parser &cfg):
        AOChan(cfg),
        AnalogChanCustomScale(cfg) {
    }
};

struct AIVoltageChan : AIChanCustomScale {
    const int32_t terminal_config = 0;

    explicit AIVoltageChan(xjson::Parser &cfg) :
        AnalogChan(cfg),
        Chan(cfg),
        AIChanCustomScale(cfg),
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

struct AIVoltageRMSChan final : AIVoltageChan {
    explicit AIVoltageRMSChan(xjson::Parser &cfg) :
        AnalogChan(cfg),
        Chan(cfg),
        AIVoltageChan(cfg) {
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

struct AIVoltageWithExcitChan final : AIVoltageChan {
    int32_t bridge_config = 0;
    ExcitationConfig excitation_config;

    explicit AIVoltageWithExcitChan(xjson::Parser &cfg) :
        AnalogChan(cfg),
        Chan(cfg),
        AIVoltageChan(cfg),
        bridge_config(parse_bridge_config(cfg)),
        excitation_config(cfg, VOLT_EXCIT_PREFIX) {
    }

    ~AIVoltageWithExcitChan() override = default;

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
            this->excitation_config.min_val_for_excitation,
            scale_key
        );
    }
};

struct AICurrentChan : AIChanCustomScale {
    static int32_t get_shunt_resistor_loc(const string &loc) {
        if (loc == "External") return DAQmx_Val_External;
        if (loc == "Internal") return DAQmx_Val_Internal;
        return DAQmx_Val_Default;
    }

    explicit AICurrentChan(xjson::Parser &cfg):
        AnalogChan(cfg),
        Chan(cfg),
        AIChanCustomScale(cfg),
        shunt_resistor_loc(
            get_shunt_resistor_loc(cfg.required<string>("shunt_resistor_loc"))),
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

    int32_t shunt_resistor_loc;
    double ext_shunt_resistor_val;
    int32 terminal_config = 0;
};

struct AICurrentRMSChan final : AICurrentChan {
    explicit AICurrentRMSChan(xjson::Parser &cfg) : AnalogChan(cfg), Chan(cfg),
                                                    AICurrentChan(cfg) {
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

struct AIRTDChan final : public AIChan {
    const int32_t rtd_type;
    const int32_t resistance_config;
    const ExcitationConfig excitation_config;
    const double r0;

    static int32_t get_rtd_type(const string &type) {
        if (type == "Pt3750") return DAQmx_Val_Pt3750;
        if (type == "PT3851") return DAQmx_Val_Pt3851;
        if (type == "PT3911") return DAQmx_Val_Pt3911;
        if (type == "PT3916") return DAQmx_Val_Pt3916;
        if (type == "PT3920") return DAQmx_Val_Pt3920;
        if (type == "PT3928") return DAQmx_Val_Pt3928;
        if (type == "Custom") return DAQmx_Val_Custom;
        return DAQmx_Val_Pt3750;
    }

public:
    explicit AIRTDChan(xjson::Parser &cfg) :
        AnalogChan(cfg),
        Chan(cfg),
        AIChan(cfg),
        rtd_type(get_rtd_type(
            cfg.required<string>("rtd_type"))),
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

struct AIThermocoupleChan final : public AIChan {
    int32_t thermocouple_type;
    int32_t cjc_source;
    string cjc_port;
    double cjc_val;

public:
    [[nodiscard]] int32_t static parse_type(xjson::Parser &cfg) {
        const auto type = cfg.required<string>("thermocouple_type");
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
        const auto source = cfg.required<string>("cjc_source");
        if (source == "BuiltIn") return DAQmx_Val_BuiltIn;
        if (source == "ConstVal") return DAQmx_Val_ConstVal;
        if (source == "Chan") return DAQmx_Val_Chan;
        cfg.field_err("cjc_source", "invalid thermocouple cjc source: " + source);
        return DAQmx_Val_BuiltIn;
    }

    explicit AIThermocoupleChan(
        xjson::Parser &cfg,
        const std::map<std::int32_t, string> &cjc_sources
    ) : AnalogChan(cfg),
        Chan(cfg),
        AIChan(cfg),
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

struct AITempBuiltInChan final : public AIChan {
    explicit AITempBuiltInChan(xjson::Parser &cfg):
        AnalogChan(cfg),
        Chan(cfg),
        AIChan(cfg) {
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

class AIThermistorIEXChan final : public AIChan {
    int32_t resistance_config;
    ExcitationConfig excitation_config;
    double a;
    double b;
    double c;

public:
    explicit AIThermistorIEXChan(xjson::Parser &cfg):
        AnalogChan(cfg),
        Chan(cfg),
        AIChan(cfg),
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

class AIThermistorVexChan final : public AIChan {
    int32_t resistance_config;
    ExcitationConfig excitation_config;
    double a;
    double b;
    double c;
    double r1;

public:
    explicit AIThermistorVexChan(xjson::Parser &cfg):
        AnalogChan(cfg),
        Chan(cfg),
        AIChan(cfg),
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

struct AIAccelChan : public AIChanCustomScale {
    double sensitivity;
    int32_t sensitivity_units;
    ExcitationConfig excitation_config;
    int32 terminal_config = 0;

    explicit AIAccelChan(xjson::Parser &cfg) :
        AnalogChan(cfg),
        Chan(cfg),
        AIChanCustomScale(cfg),
        sensitivity(cfg.required<double>("sensitivity")),
        excitation_config(cfg, CURR_EXCIT_PREFIX),
        terminal_config(parse_terminal_config(cfg)) {
        const auto su = cfg.optional<string>("sensitivity_units", "mVoltsPerG");
        this->sensitivity_units = ni::UNITS_MAP.at(su);
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

struct AIAccel4WireDCVoltageChan final : public AIAccelChan {
    explicit AIAccel4WireDCVoltageChan(xjson::Parser &cfg) :
        AnalogChan(cfg),
        Chan(cfg),
        AIAccelChan(cfg) {
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

class AIAccelChargeChan final : public AIChanCustomScale {
    double sensitivity;
    int32_t sensitivity_units = 0;
    int32 terminal_config = 0;

public:
    explicit AIAccelChargeChan(xjson::Parser &cfg) :
        AnalogChan(cfg),
        Chan(cfg),
        AIChanCustomScale(cfg),
        sensitivity(cfg.required<double>("sensitivity")),
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

class AIResistanceChan final : public AIChanCustomScale {
    int32_t resistance_config;
    ExcitationConfig excitation_config;

public:
    explicit AIResistanceChan(xjson::Parser &cfg) :
        AnalogChan(cfg),
        Chan(cfg),
        AIChanCustomScale(cfg),
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

class AIBridgeChan final : public AIChanCustomScale {
public:
    BridgeConfig bridge_config;

    explicit AIBridgeChan(xjson::Parser &cfg) :
        AnalogChan(cfg),
        Chan(cfg),
        AIChanCustomScale(cfg),
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

struct AIStrainGaugeChan final : public AIChanCustomScale {
    const int32_t strain_config;
    const ExcitationConfig excitation_config;
    const double gage_factor;
    const double initial_bridge_voltage;
    const double nominal_gage_resistance;
    const double poisson_ratio;
    const double lead_wire_resistance;

    static int32_t get_strain_config(const string &s) {
        if (s == "FullBridgeI") return DAQmx_Val_FullBridgeI;
        if (s == "FullBridgeII") return DAQmx_Val_FullBridgeII;
        if (s == "FullBridgeIII") return DAQmx_Val_FullBridgeIII;
        if (s == "HalfBridgeI") return DAQmx_Val_HalfBridgeI;
        if (s == "HalfBridgeII") return DAQmx_Val_HalfBridgeII;
        if (s == "QuarterBridgeI") return DAQmx_Val_QuarterBridgeI;
        if (s == "QuarterBridgeII") return DAQmx_Val_QuarterBridgeII;
        return DAQmx_Val_FullBridgeI;
    }

    explicit AIStrainGaugeChan(xjson::Parser &cfg) :
        AnalogChan(cfg),
        Chan(cfg),
        AIChanCustomScale(cfg),
        strain_config(get_strain_config(cfg.required<string>("strain_config"))),
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

class AIRosetteStrainGaugeChan final : public AIChan {
    int32_t rosette_type;
    double gage_orientation;
    int32 rosette_meas_type;
    int32 strain_config;
    ExcitationConfig excitation_config;
    double gage_factor;
    double nominal_gage_resistance;
    double poisson_ratio;
    double lead_wire_resistance;

public:
    static int32_t get_strain_config(const string &s) {
        if (s == "FullBridgeI") return DAQmx_Val_FullBridgeI;
        if (s == "FullBridgeII") return DAQmx_Val_FullBridgeII;
        if (s == "FullBridgeIII") return DAQmx_Val_FullBridgeIII;
        if (s == "HalfBridgeI") return DAQmx_Val_HalfBridgeI;
        if (s == "HalfBridgeII") return DAQmx_Val_HalfBridgeII;
        if (s == "QuarterBridgeI") return DAQmx_Val_QuarterBridgeI;
        if (s == "QuarterBridgeII") return DAQmx_Val_QuarterBridgeII;
        return DAQmx_Val_FullBridgeI;
    }

    static int32_t get_rosette_type(const string &s) {
        if (s == "RectangularRosette") return DAQmx_Val_RectangularRosette;
        if (s == "DeltaRosette") return DAQmx_Val_DeltaRosette;
        if (s == "TeeRosette") return DAQmx_Val_TeeRosette;
        return DAQmx_Val_RectangularRosette;
    }

    static int32_t get_rosette_meas_type(const string &s) {
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

    explicit AIRosetteStrainGaugeChan(xjson::Parser &cfg):
        AnalogChan(cfg),
        Chan(cfg),
        AIChan(cfg),
        rosette_type(get_rosette_type(
            cfg.required<string>("rosette_type"))),
        gage_orientation(cfg.required<double>("gage_orientation")),
        rosette_meas_type(
            get_rosette_meas_type(cfg.required<string>("rosette_meas_type"))),
        strain_config(get_strain_config(cfg.required<string>("strain_config"))),
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

struct AIMicrophoneChan final : AIChanCustomScale {
    const double mic_sensitivity;
    const double max_snd_press_level;
    const ExcitationConfig excitation_config;
    const int32 terminal_config = 0;

    explicit AIMicrophoneChan(xjson::Parser &cfg) :
        AnalogChan(cfg),
        Chan(cfg),
        AIChanCustomScale(cfg),
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

class AIFrequencyVoltageChan final : public AIChanCustomScale {
    double threshold_level;
    double hysteresis;

public:
    explicit AIFrequencyVoltageChan(xjson::Parser &cfg) :
        AnalogChan(cfg),
        Chan(cfg),
        AIChanCustomScale(cfg),
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

struct AIPressureBridgeTwoPointLinChan final : public AIChanCustomScale {
    const BridgeConfig bridge_config;
    const TwoPointLinConfig two_point_lin_config;

    explicit AIPressureBridgeTwoPointLinChan(xjson::Parser &cfg) :
        AnalogChan(cfg),
        Chan(cfg),
        AIChanCustomScale(cfg),
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

struct AIPressureBridgeTableChan final : AIChanCustomScale {
    const BridgeConfig bridge_config;
    const TableConfig table_config;

    explicit AIPressureBridgeTableChan(xjson::Parser &cfg) :
        AnalogChan(cfg),
        Chan(cfg),
        AIChanCustomScale(cfg),
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

class AIPressureBridgePolynomialChan final : public AIChanCustomScale {
    BridgeConfig bridge_config;
    PolynomialConfig polynomial_config;

public:
    explicit AIPressureBridgePolynomialChan(xjson::Parser &cfg):
        AnalogChan(cfg),
        Chan(cfg),
        AIChanCustomScale(cfg),
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

class AIForceBridgePolynomialChan final : public AIChanCustomScale {
    BridgeConfig bridge_config;
    PolynomialConfig polynomial_config;

public:
    explicit AIForceBridgePolynomialChan(xjson::Parser &cfg):
        AnalogChan(cfg),
        Chan(cfg),
        AIChanCustomScale(cfg),
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

struct AIForceBridgeTableChan final : public AIChanCustomScale {
    BridgeConfig bridge_config;
    TableConfig table_config;

    explicit AIForceBridgeTableChan(xjson::Parser &cfg):
        AnalogChan(cfg),
        Chan(cfg),
        AIChanCustomScale(cfg),
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

struct AIForceBridgeTwoPointLinChan final : AIChanCustomScale {
    BridgeConfig bridge_config;
    TwoPointLinConfig two_point_lin_config;

    explicit AIForceBridgeTwoPointLinChan(xjson::Parser &cfg):
        AnalogChan(cfg),
        Chan(cfg),
        AIChanCustomScale(cfg),
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

struct AIVelocityIEPEChan final : public AIChanCustomScale {
    const int32_t sensitivity_units;
    const double sensitivity;
    const ExcitationConfig excitation_config;
    const int32_t terminal_config;

    explicit AIVelocityIEPEChan(xjson::Parser &cfg):
        AnalogChan(cfg),
        Chan(cfg),
        AIChanCustomScale(cfg),
        sensitivity_units(
            ni::AIVelocityIEPEChan::parse_units(cfg, "sensitivity_units")),
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

struct AITorqueBridgeTwoPointLinChan final : public AIChanCustomScale {
    const BridgeConfig bridge_config;
    const TwoPointLinConfig two_point_lin_config;

    explicit AITorqueBridgeTwoPointLinChan(xjson::Parser &cfg) :
        AnalogChan(cfg),
        Chan(cfg),
        AIChanCustomScale(cfg),
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

class AITorqueBridgePolynomialChan final : public AIChanCustomScale {
    BridgeConfig bridge_config;
    PolynomialConfig polynomial_config;

public:
    explicit AITorqueBridgePolynomialChan(xjson::Parser &cfg) :
        AnalogChan(cfg),
        Chan(cfg),
        AIChanCustomScale(cfg),
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

struct AITorqueBridgeTableChan final : public AIChanCustomScale {
    BridgeConfig bridge_config;
    TableConfig table_config;

    explicit AITorqueBridgeTableChan(xjson::Parser &cfg) :
        AnalogChan(cfg),
        Chan(cfg),
        AIChanCustomScale(cfg),
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

struct AIForceIEPEChan final : public AIChanCustomScale {
    const int32_t sensitivity_units;
    const double sensitivity;
    const ExcitationConfig excitation_config;
    const int32 terminal_config;

    explicit AIForceIEPEChan(xjson::Parser &cfg) :
        AnalogChan(cfg),
        Chan(cfg),
        AIChanCustomScale(cfg),
        sensitivity_units(ni::AIForceIEPEChan::parse_units(cfg, "sensitivity_units")),
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

class AIChargeChan final : public AIChanCustomScale {
public:
    explicit AIChargeChan(xjson::Parser &cfg) :
        AnalogChan(cfg),
        Chan(cfg),
        AIChanCustomScale(cfg),
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

private:
    int32 terminal_config = 0;
};

struct AOVoltageChan final : AOChanCustomScale {
    explicit AOVoltageChan(xjson::Parser &cfg):
        AnalogChan(cfg),
        Chan(cfg),
        AOChanCustomScale(cfg) {
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

struct AOCurrent final : AOChanCustomScale {
    explicit AOCurrent(xjson::Parser &cfg) :
        AnalogChan(cfg),
        Chan(cfg),
        AOChanCustomScale(cfg) {
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

struct AOFunctionGeneratorChan final : AOChan {
    const double frequency;
    const double amplitude;
    const double offset;
    const int32 wave_type;

    int32_t static get_type(const string &type, const xjson::Parser &cfg) {
        if (type == "Sine") return DAQmx_Val_Sine;
        if (type == "Triangle") return DAQmx_Val_Triangle;
        if (type == "Square") return DAQmx_Val_Square;
        if (type == "Sawtooth") return DAQmx_Val_Sawtooth;
        cfg.field_err("", "invalid wave type: " + type);
        return DAQmx_Val_Sine;
    }


    explicit AOFunctionGeneratorChan(xjson::Parser &cfg) :
        AnalogChan(cfg),
        Chan(cfg),
        AOChan(cfg),
        frequency(cfg.required<double>("frequency")),
        amplitude(cfg.required<double>("amplitude")),
        offset(cfg.required<double>("offset")),
        wave_type(get_type(cfg.required<string>("wave_type"), cfg)) {
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

using AIChanFactory = std::function<std::unique_ptr<AIChan>(
        xjson::Parser &cfg,
        const std::map<int32_t, string> &port_to_channel)
>;
using AOChanFactory = std::function<std::unique_ptr<AOChan>(xjson::Parser &cfg)>;

#define AO_CHAN_FACTORY(type, class) \
    {type, [](xjson::Parser& cfg) { return std::make_unique<class>(cfg); }}

#define AI_CHAN_FACTORY(type, class) \
    {type, [](xjson::Parser& cfg, const auto& ptc) { return std::make_unique<class>(cfg); }}

#define AI_CHAN_ENTRY_WITH_PORT(type, class) \
    {type, [](xjson::Parser& cfg, const auto& ptc) { return std::make_unique<class>(cfg, ptc); }}

static const std::map<string, AOChanFactory> AO_CHANS = {
    AO_CHAN_FACTORY("ao_current", AOCurrent),
    AO_CHAN_FACTORY("ao_voltage", AOVoltageChan),
    AO_CHAN_FACTORY("ao_func_gen", AOFunctionGeneratorChan)
};

static const std::map<string, AIChanFactory> AI_CHANS = {
    AI_CHAN_FACTORY("ai_accel", AIAccelChan),
    AI_CHAN_FACTORY("ai_accel_4_wire_dc_voltage", AIAccel4WireDCVoltageChan),
    AI_CHAN_FACTORY("ai_bridge", AIBridgeChan),
    AI_CHAN_FACTORY("ai_charge", AIChargeChan),
    AI_CHAN_FACTORY("ai_current", AICurrentChan),
    AI_CHAN_FACTORY("ai_force_bridge_polynomial", AIForceBridgePolynomialChan),
    AI_CHAN_FACTORY("ai_force_bridge_table", AIForceBridgeTableChan),
    AI_CHAN_FACTORY("ai_force_bridge_two_point_lin", AIForceBridgeTwoPointLinChan),
    AI_CHAN_FACTORY("ai_force_iepe", AIForceIEPEChan),
    AI_CHAN_FACTORY("ai_microphone", AIMicrophoneChan),
    AI_CHAN_FACTORY("ai_pressure_bridge_polynomial", AIPressureBridgePolynomialChan),
    AI_CHAN_FACTORY("ai_pressure_bridge_table", AIPressureBridgeTableChan),
    AI_CHAN_FACTORY("ai_pressure_bridge_two_point_lin",
                    AIPressureBridgeTwoPointLinChan),
    AI_CHAN_FACTORY("ai_resistance", AIResistanceChan),
    AI_CHAN_FACTORY("ai_rtd", AIRTDChan),
    AI_CHAN_FACTORY("ai_strain_gauge", AIStrainGaugeChan),
    AI_CHAN_FACTORY("ai_temp_builtin", AITempBuiltInChan),
    AI_CHAN_ENTRY_WITH_PORT("ai_thermocouple", AIThermocoupleChan),
    AI_CHAN_FACTORY("ai_torque_bridge_polynomial", AITorqueBridgePolynomialChan),
    AI_CHAN_FACTORY("ai_torque_bridge_table", AITorqueBridgeTableChan),
    AI_CHAN_FACTORY("ai_torque_bridge_two_point_lin", AITorqueBridgeTwoPointLinChan),
    AI_CHAN_FACTORY("ai_velocity_iepe", AIVelocityIEPEChan),
    AI_CHAN_FACTORY("ai_voltage", AIVoltageChan)
};

inline std::unique_ptr<InputChan> parse_input_chan(
    xjson::Parser &cfg,
    const std::map<int32_t, string> &port_to_channel
) {
    const auto type = cfg.required<string>("type");
    if (AI_CHANS.count(type) == 0)
        cfg.field_err("type", "invalid analog input channel type: " + type);
    return AI_CHANS.at(type)(cfg, port_to_channel);
}

inline std::unique_ptr<OutputChan> parse_output_chan(xjson::Parser &cfg) {
    const auto type = cfg.required<string>("type");
    if (AO_CHANS.count(type) == 0)
        cfg.field_err("type", "invalid analog output channel type: " + type);
    return AO_CHANS.at(type)(cfg);
}

#undef AO_CHAN_FACTORY
#undef AI_CHAN_FACTORY
#undef AI_CHAN_ENTRY_WITH_PORT
};
