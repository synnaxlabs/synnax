// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <map>

#include "daqmx.h"
#include "nidaqmx_api.h"
#include "nisyscfg.h"
#include "driver/ni/ni.h"

#include "client/cpp/telem/telem.h"
#include "driver/config/config.h"
#include "driver/ni/scale.h"

#include "glog/logging.h"
#include "nlohmann/json.hpp"

namespace ni {
    static inline int32_t get_terminal_config(const std::string &terminal_config) {
        if (terminal_config == "PseudoDiff") return DAQmx_Val_PseudoDiff;
        if (terminal_config == "Diff") return DAQmx_Val_Diff;
        if (terminal_config == "NRSE") return DAQmx_Val_NRSE;
        if (terminal_config == "RSE") return DAQmx_Val_RSE;
        return DAQmx_Val_Cfg_Default;
    }

    static inline int32_t get_bridge_config(const std::string &s) {
        if (s == "FullBridge") return DAQmx_Val_FullBridge;
        if (s == "HalfBridge") return DAQmx_Val_HalfBridge;
        if (s == "QuarterBridge") return DAQmx_Val_QuarterBridge;
        return DAQmx_Val_FullBridge;
    }

    static inline int32_t get_resistance_config(const std::string &s) {
        if (s == "2Wire") return DAQmx_Val_2Wire;
        if (s == "3Wire") return DAQmx_Val_3Wire;
        if (s == "4Wire") return DAQmx_Val_4Wire;
        return DAQmx_Val_2Wire;
    }

    static inline int32_t get_excitation_src(const std::string &s) {
        if (s == "Internal") return DAQmx_Val_Internal;
        if (s == "External") return DAQmx_Val_External;
        if (s == "None") return DAQmx_Val_None;
        return DAQmx_Val_None;
    }

    struct VoltageExcitationConfig {
        int32_t excit_source;
        double excit_val;
        double min_val_for_excitation; // optional
        double max_val_for_excitation; //optional
        bool32 use_excit_for_scaling; //optional

        VoltageExcitationConfig(config::Parser &parser)
                : excit_source(
                get_excitation_src(
                        parser.required<std::string>("voltage_excit_source"))),
                  excit_val(parser.required<double>("voltage_excit_val")),
                  min_val_for_excitation(
                          parser.optional<double>("min_val_for_excitation", 0)),
                  max_val_for_excitation(
                          parser.optional<double>("max_val_for_excitation", 0)),
                  use_excit_for_scaling(
                          parser.optional<bool32>("use_excit_for_scaling", 0)) {
        }
    };

    struct CurrentExcitationConfig {
        int32_t excit_source;
        double excit_val;
        double min_val_for_excitation; // optional
        double max_val_for_excitation; //optional
        bool32 use_excit_for_scaling; //optional

        CurrentExcitationConfig(config::Parser &parser)
                : excit_source(
                get_excitation_src(
                        parser.required<std::string>("current_excit_source"))),
                  excit_val(parser.required<double>("current_excit_val")),
                  min_val_for_excitation(
                          parser.optional<double>("min_val_for_excitation", 0)),
                  max_val_for_excitation(
                          parser.optional<double>("max_val_for_excitation", 0)),
                  use_excit_for_scaling(
                          parser.optional<bool32>("use_excit_for_scaling", 0)) {
        }
    };


    struct BridgeConfig {
        int32_t ni_bridge_config;
        int32_t voltage_excit_source;
        double voltage_excit_val;
        double nominal_bridge_resistance;

        BridgeConfig() = default;

        BridgeConfig(config::Parser &parser)
                : ni_bridge_config(
                get_bridge_config(parser.required<std::string>("bridge_config"))),
                  voltage_excit_source(
                          get_excitation_src(parser.required<std::string>(
                                  "voltage_excit_source"))),
                  voltage_excit_val(parser.required<double>("voltage_excit_val")),
                  nominal_bridge_resistance(
                          parser.required<double>("nominal_bridge_resistance")) {
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

        PolynomialConfig(config::Parser &parser)
                : num_forward_coeffs(parser.required<uint32_t>("num_forward_coeffs")),
                  num_reverse_coeffs(parser.required<uint32_t>("num_reverse_coeffs")) {
            auto eu = parser.required<std::string>("electrical_units");
            auto pu = parser.required<std::string>("physical_units");

            if (ni::UNITS_MAP.find(eu) == ni::UNITS_MAP.end()) {
                LOG(WARNING) << "Invalid units: " << eu << ". Defaulting to Volts.";
                electrical_units = DAQmx_Val_Volts;
            } else {
                electrical_units = ni::UNITS_MAP.at(eu);
            }

            if (ni::UNITS_MAP.find(pu) == ni::UNITS_MAP.end()) {
                LOG(WARNING) << "Invalid units: " << pu << ". Defaulting to Volts.";
                physical_units = DAQmx_Val_Volts;
            } else {
                physical_units = ni::UNITS_MAP.at(pu);
            }

            json j = parser.get_json();

            forward_coeffs = new double[num_forward_coeffs];
            reverse_coeffs = new double[num_reverse_coeffs];

            auto f = parser.required_vector<double>("forward_coeffs");

            //get forward coeffs (prescale -> scale)
            for (uint32_t i = 0; i < num_forward_coeffs; i++)
                forward_coeffs[i] = f[i];

            ni::NiDAQmxInterface::CalculateReversePolyCoeff(
                    forward_coeffs,
                    num_forward_coeffs,
                    -1000, //FIXME dont hard code
                    1000, //FIXME dont hard code
                    num_reverse_coeffs,
                    -1,
                    reverse_coeffs
            ); // FIXME: reversePoly order should be user inputted?
        }

        ~PolynomialConfig() {
            if (forward_coeffs != nullptr) delete[] forward_coeffs;
            if (reverse_coeffs != nullptr) delete[] reverse_coeffs;
        }
    };

    struct TableConfig {
        float64 *electrical_vals;
        uint32_t num_eletrical_vals;
        float64 *physical_vals;
        uint32_t num_physical_vals;
        int32_t electrical_units;
        int32_t physical_units;

        TableConfig() = default;

        TableConfig(config::Parser &parser)
                : num_eletrical_vals(parser.required<uint32_t>("num_electrical_vals")),
                  num_physical_vals(parser.required<uint32_t>("num_physical_vals")) {
            auto eu = parser.required<std::string>("electrical_units");
            auto pu = parser.required<std::string>("physical_units");

            electrical_units = ni::UNITS_MAP.at(eu);
            physical_units = ni::UNITS_MAP.at(pu);

            // TODO: figure out why using vector and .data() throws exception when passed to
            // NI function
            electrical_vals = new double[num_eletrical_vals];
            auto e = parser.required_vector<double>("electrical_vals");
            for (uint32_t i = 0; i < num_eletrical_vals; i++)
                electrical_vals[i] = e[i];

            physical_vals = new double[num_physical_vals];
            auto p = parser.required_vector<double>("physical_vals");
            for (uint32_t i = 0; i < num_physical_vals; i++)
                physical_vals[i] = p[i];
        }

        ~TableConfig() {
            if (electrical_vals != nullptr) delete[] electrical_vals;
            if (physical_vals != nullptr) delete[] physical_vals;
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

        TwoPointLinConfig(config::Parser &parser)
                : first_electrical_val(parser.required<double>("first_electrical_val")),
                  second_electrical_val(
                          parser.required<double>("second_electrical_val")),
                  first_physical_val(parser.required<double>("first_physical_val")),
                  second_physical_val(parser.required<double>("second_physical_val")) {
            auto eu = parser.required<std::string>("electrical_units");
            auto pu = parser.required<std::string>("physical_units");
            electrical_units = ni::UNITS_MAP.at(eu);
            physical_units = ni::UNITS_MAP.at(pu);
        }
    };

///////////////////////////////////////////////////////////////////////////////////
//                                     ANALOG                                    //
///////////////////////////////////////////////////////////////////////////////////
/// @brief an object that represents and is responsible for the configuration of
/// a single analog channel on National Instruments hardware.
/// base class for all special analog channel types.
    class Analog {
    public:
        Analog() = default;

        ~Analog() = default;

        virtual int32 create_ni_channel() {
            return 0;
        }

        std::unique_ptr<ScaleConfig> getScaleConfig(config::Parser &parser) {
            std::string c = std::to_string(parser.required<uint32_t>("channel"));

            parser.get_json();
            if (!parser.get_json().contains("custom_scale")) return nullptr;
            auto scale_parser = parser.child("custom_scale");
            if (scale_parser.required<std::string>("type") == "none") return nullptr;
            this->scale_name = c + "_scale";
            return std::make_unique<ScaleConfig>(scale_parser, this->scale_name);
        }

        int32 create_ni_scale() {
            if (this->scale_name == "") return 0;
            return this->scale_config->create_ni_scale();
        }

        int32 get_units(const std::string &s, config::Parser &parser) {
            if (ni::UNITS_MAP.find(s) == ni::UNITS_MAP.end()) {
                LOG(WARNING) << "Invalid units: " << s << ". Defaulting to Volts.";
                return DAQmx_Val_Volts;
            }

            return ni::UNITS_MAP.at(s);
        }

        explicit Analog(config::Parser &parser, TaskHandle task_handle,
                        const std::string &name)
                : task_handle(task_handle),
                  min_val(parser.optional<float_t>("min_val", 0)),
                  max_val(parser.optional<float_t>("max_val", 0)),
                  units(get_units(parser.optional<std::string>("units", "Volts"),
                                  parser)),
                  sy_key(parser.required<uint32_t>("channel")),
                  name(name),
                  type(parser.required<std::string>("type")),
                  scale_config(getScaleConfig(parser)) {
            if (this->scale_name != "") this->units = DAQmx_Val_FromCustomScale;
        }

        TaskHandle task_handle = 0;
        double min_val = 0;
        double max_val = 0;
        int32_t units = DAQmx_Val_Volts;
        uint32_t sy_key = 0;
        std::string name = "";
        std::string type = "";
        std::string scale_name = "";

        std::unique_ptr<ScaleConfig> scale_config;
    };

///////////////////////////////////////////////////////////////////////////////////
//                                      Voltage                                  //
///////////////////////////////////////////////////////////////////////////////////
/// @brief voltage channel.
    class Voltage : public Analog {
    public:
        explicit Voltage(config::Parser &parser, TaskHandle task_handle,
                         const std::string &name)
                : Analog(parser, task_handle, name),
                  terminal_config(
                          ni::get_terminal_config(
                                  parser.required<std::string>("terminal_config"))) {
        }

        ~Voltage() = default;

        int32 create_ni_channel() override {
            std::string s = "";
            return ni::NiDAQmxInterface::CreateAIVoltageChan(
                    this->task_handle,
                    this->name.c_str(),
                    "", // name to assign channel
                    this->terminal_config,
                    this->min_val,
                    this->max_val,
                    this->units,
                    this->scale_name.c_str()
            );
        }

        int32_t terminal_config = 0;
    };

/// @brief RMS voltage Channel
    class VoltageRMS final : public Voltage {
    public:
        explicit VoltageRMS(config::Parser &parser, TaskHandle task_handle,
                            const std::string &name)
                : Voltage(parser, task_handle, name) {
        }

        ~VoltageRMS() = default;

        int32 create_ni_channel() override {
            return ni::NiDAQmxInterface::CreateAIVoltageRMSChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->terminal_config,
                    this->min_val,
                    this->max_val,
                    this->units,
                    this->scale_name.c_str()
            );
        }
    };

/// @brief voltage Channel with excitation reference
    class VoltageWithExcit final : public Voltage {
    public:
        int32_t bridge_config = 0;
        VoltageExcitationConfig excitation_config;

        explicit VoltageWithExcit(config::Parser &parser, TaskHandle task_handle,
                                  const std::string &name)
                : Voltage(parser, task_handle, name),
                  bridge_config(
                          get_bridge_config(
                                  parser.required<std::string>("bridge_config"))),
                  excitation_config(parser) {
        }

        ~VoltageWithExcit() = default;

        int32 create_ni_channel() override {
            return ni::NiDAQmxInterface::CreateAIVoltageChanWithExcit(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->terminal_config,
                    this->min_val,
                    this->max_val,
                    this->units,
                    this->bridge_config,
                    this->excitation_config.excit_source,
                    this->excitation_config.excit_val,
                    this->excitation_config.min_val_for_excitation,
                    this->scale_name.c_str()
            );
        }
    };

///////////////////////////////////////////////////////////////////////////////////
//                                      Current                                  //
///////////////////////////////////////////////////////////////////////////////////
    class Current : public Analog {
    public:
        static int32_t getShuntResistorLocation(const std::string &loc) {
            if (loc == "External") return DAQmx_Val_External;
            if (loc == "Internal") return DAQmx_Val_Internal;
            return DAQmx_Val_Default;
        }

        explicit Current(config::Parser &parser, TaskHandle task_handle,
                         const std::string &name)
                : Analog(parser, task_handle, name),
                  terminal_config(
                          ni::get_terminal_config(
                                  parser.required<std::string>("terminal_config"))),
                  shunt_resistor_loc(
                          getShuntResistorLocation(
                                  parser.required<std::string>("shunt_resistor_loc"))),
                  ext_shunt_resistor_val(
                          parser.required<double>("ext_shunt_resistor_val")) {
        }

        int32 create_ni_channel() override {
            return ni::NiDAQmxInterface::CreateAICurrentChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->terminal_config,
                    this->min_val,
                    this->max_val,
                    this->units,
                    this->shunt_resistor_loc,
                    this->ext_shunt_resistor_val,
                    this->scale_name.c_str()
            );
        }

        int32_t shunt_resistor_loc;
        double ext_shunt_resistor_val;
        int32 terminal_config = 0;
    };

    class CurrentRMS final : public Current {
    public:
        explicit CurrentRMS(config::Parser &parser, TaskHandle task_handle,
                            const std::string &name)
                : Current(parser, task_handle, name) {
        }

        int32 create_ni_channel() override {
            return ni::NiDAQmxInterface::CreateAICurrentRMSChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->terminal_config,
                    this->min_val,
                    this->max_val,
                    this->units,
                    this->shunt_resistor_loc,
                    this->ext_shunt_resistor_val,
                    this->scale_name.c_str()
            );
        }
    };

///////////////////////////////////////////////////////////////////////////////////
//                                       RTD                                     //
///////////////////////////////////////////////////////////////////////////////////
    class RTD final : public Analog {
    public:
        static int32_t get_rtd_type(std::string type) {
            if (type == "Pt3750") return DAQmx_Val_Pt3750;
            if (type == "PT3851") return DAQmx_Val_Pt3851;
            if (type == "PT3911") return DAQmx_Val_Pt3911;
            if (type == "PT3916") return DAQmx_Val_Pt3916;
            if (type == "PT3920") return DAQmx_Val_Pt3920;
            if (type == "PT3928") return DAQmx_Val_Pt3928;
            if (type == "Custom") return DAQmx_Val_Custom;
            return DAQmx_Val_Pt3750;
        }

        explicit RTD(config::Parser &parser, TaskHandle task_handle,
                     const std::string &name)
                : Analog(parser, task_handle, name),
                  rtd_type(get_rtd_type(parser.required<std::string>("rtd_type"))),
                  resistance_config(
                          get_resistance_config(
                                  parser.required<std::string>("resistance_config"))),
                  excitation_config(parser),
                  r0(parser.required<double>("r0")) {
        }

        int32 create_ni_channel() override {
            return ni::NiDAQmxInterface::CreateAIRTDChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
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

    private:
        int32_t rtd_type;
        int32_t resistance_config;
        CurrentExcitationConfig excitation_config;
        double r0;
    };

///////////////////////////////////////////////////////////////////////////////////
//                                      Temperature                              //
///////////////////////////////////////////////////////////////////////////////////
    class Thermocouple final : public Analog {
    public:
        int32_t get_type(const std::string &type, config::Parser &parser) {
            if (type == "J") return DAQmx_Val_J_Type_TC;
            if (type == "K") return DAQmx_Val_K_Type_TC;
            if (type == "N") return DAQmx_Val_N_Type_TC;
            if (type == "R") return DAQmx_Val_R_Type_TC;
            if (type == "S") return DAQmx_Val_S_Type_TC;
            if (type == "T") return DAQmx_Val_T_Type_TC;
            if (type == "B") return DAQmx_Val_B_Type_TC;
            if (type == "E") return DAQmx_Val_E_Type_TC;
            parser.field_err("channels." + name,
                             "Invalid thermocouple type:" + type +
                             ". Defaulting to J Type.");
            return DAQmx_Val_J_Type_TC;
        }

        int32_t get_cjc_source(const std::string &source, config::Parser &parser) {
            if (source == "BuiltIn") return DAQmx_Val_BuiltIn;
            if (source == "ConstVal") return DAQmx_Val_ConstVal;
            if (source == "Chan") return DAQmx_Val_Chan;
            parser.field_err("channels." + this->name,
                             "Invalid thermocouple cjc source:" + source +
                             ". Defaulting to BuiltIn.");
            return DAQmx_Val_BuiltIn;
        }

        explicit Thermocouple(config::Parser &parser, TaskHandle task_handle,
                              const std::string &name,
                              std::map<std::int32_t, std::string> &cjc_sources)
                : Analog(parser, task_handle, name),
                  thermocouple_type(
                          get_type(parser.required<std::string>("thermocouple_type"),
                                   parser)),
                  cjc_source(get_cjc_source(parser.required<std::string>("cjc_source"),
                                            parser)),
                  cjc_val(parser.optional<double>("cjc_val", 0)) {

            auto source = parser.required<std::int32_t>("cjc_port");
            if (cjc_sources.find(source) == cjc_sources.end()) this->cjcPort = "";
            else this->cjcPort = cjc_sources.at(source);

        }

        int32 create_ni_channel() override {
            return ni::NiDAQmxInterface::CreateAIThrmcplChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->min_val,
                    this->max_val,
                    this->units,
                    this->thermocouple_type,
                    this->cjc_source,
                    this->cjc_val,
                    this->cjcPort.c_str()
            );
        }

    private:
        int32_t thermocouple_type;
        int32_t cjc_source;
        std::string cjcPort;
        double cjc_val;
    };

    class TemperatureBuiltInSensor final : public Analog {
    public:
        explicit TemperatureBuiltInSensor(config::Parser &parser,
                                          TaskHandle task_handle,
                                          const std::string &name) {
            this->units = ni::UNITS_MAP.at(parser.required<std::string>("units"));
            this->task_handle = task_handle;
            size_t pos = name.find("/");
            this->name = name.substr(0, pos) + "/_boardTempSensor_vs_aignd";
        }

        int32 create_ni_channel() override {
            return ni::NiDAQmxInterface::CreateAITempBuiltInSensorChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->units
            );
        }
    };

    class ThermistorIEX final : public Analog {
    public:
        explicit ThermistorIEX(config::Parser &parser, TaskHandle task_handle,
                               const std::string &name)
                : Analog(parser, task_handle, name),
                  resistance_config(
                          get_resistance_config(
                                  parser.required<std::string>("resistance_config"))),
                  excitation_config(parser),
                  a(parser.required<double>("a")),
                  b(parser.required<double>("b")),
                  c(parser.required<double>("c")) {
        }

        int32 create_ni_channel() override {
            return ni::NiDAQmxInterface::CreateAIThrmstrChanIex(
                    this->task_handle,
                    this->name.c_str(),
                    "",
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

    private:
        int32_t resistance_config;
        CurrentExcitationConfig excitation_config;
        double a;
        double b;
        double c;
    };

    class ThermistorVex final : public Analog {
    public:
        explicit ThermistorVex(config::Parser &parser, TaskHandle task_handle,
                               const std::string &name)
                : Analog(parser, task_handle, name),
                  resistance_config(
                          get_resistance_config(
                                  parser.required<std::string>("resistance_config"))),
                  excitation_config(parser),
                  a(parser.required<double>("a")),
                  b(parser.required<double>("b")),
                  c(parser.required<double>("c")),
                  r1(parser.required<double>("r1")) {
        }

        int32 create_ni_channel() override {
            return ni::NiDAQmxInterface::CreateAIThrmstrChanVex(
                    this->task_handle,
                    this->name.c_str(),
                    "",
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

    private:
        int32_t resistance_config;
        VoltageExcitationConfig excitation_config;
        double a;
        double b;
        double c;
        double r1;
    };

///////////////////////////////////////////////////////////////////////////////////
//                                    Acceleration                               //
///////////////////////////////////////////////////////////////////////////////////
/// @brief acceleration channel
    class Acceleration : public Analog {
    public:
        explicit Acceleration(config::Parser &parser, TaskHandle task_handle,
                              const std::string &name)
                : Analog(parser, task_handle, name),
                  terminal_config(
                          ni::get_terminal_config(
                                  parser.required<std::string>("terminal_config"))),
                  sensitivity(parser.required<double>("sensitivity")),
                  excitation_config(parser) {
            std::string su = parser.optional<
                    std::string>("sensitivity_units", "mVoltsPerG");
            this->sensitivity_units = ni::UNITS_MAP.at(su);
        }

        int32 create_ni_channel() override {
            return ni::NiDAQmxInterface::CreateAIAccelChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->terminal_config,
                    this->min_val,
                    this->max_val,
                    this->units,
                    this->sensitivity,
                    this->sensitivity_units,
                    this->excitation_config.excit_source,
                    this->excitation_config.excit_val,
                    this->scale_name.c_str()
            );
        }

        double sensitivity;
        int32_t sensitivity_units;
        CurrentExcitationConfig excitation_config;
        int32 terminal_config = 0;
    };

/// @brief acceleration channel with 4 wire DC voltage
    class Acceleration4WireDCVoltage final : public Acceleration {
    public:
        explicit Acceleration4WireDCVoltage(config::Parser &parser,
                                            TaskHandle task_handle,
                                            const std::string &name)
                : Acceleration(parser, task_handle, name) {
        }

        int32 create_ni_channel() override {
            return ni::NiDAQmxInterface::CreateAIAccel4WireDCVoltageChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->terminal_config,
                    this->min_val,
                    this->max_val,
                    this->units,
                    this->sensitivity,
                    this->sensitivity_units,
                    this->excitation_config.excit_source,
                    this->excitation_config.excit_val,
                    this->excitation_config.use_excit_for_scaling,
                    this->scale_name.c_str()
            );
        }
    };

/// @brief acceleration channel with charge
    class AccelerationCharge final : public Analog {
    public:
        explicit AccelerationCharge(config::Parser &parser, TaskHandle task_handle,
                                    const std::string &name)
                : Analog(parser, task_handle, name),
                  terminal_config(
                          ni::get_terminal_config(
                                  parser.required<std::string>("terminal_config"))),
                  sensitivity(parser.required<double>("sensitivity")) {
        }

        int32 create_ni_channel() override {
            return ni::NiDAQmxInterface::CreateAIAccelChargeChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->terminal_config,
                    this->min_val,
                    this->max_val,
                    this->units,
                    this->sensitivity,
                    this->sensitivity_units,
                    this->scale_name.c_str()
            );
        }

    private:
        double sensitivity;
        int32_t sensitivity_units;
        int32 terminal_config = 0;
    };

///////////////////////////////////////////////////////////////////////////////////
//                                      Resistance                               //
///////////////////////////////////////////////////////////////////////////////////
    class Resistance final : public Analog {
    public:
        explicit Resistance(config::Parser &parser, TaskHandle task_handle,
                            const std::string &name)
                : Analog(parser, task_handle, name),
                  resistance_config(
                          get_resistance_config(
                                  parser.required<std::string>("resistance_config"))),
                  excitation_config(parser) {
        }

        int32 create_ni_channel() override {
            return ni::NiDAQmxInterface::CreateAIResistanceChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->min_val,
                    this->max_val,
                    this->units,
                    this->resistance_config,
                    this->excitation_config.excit_source,
                    this->excitation_config.excit_val,
                    this->scale_name.c_str()
            );
        }

    private:
        int32_t resistance_config;
        CurrentExcitationConfig excitation_config;
    };

///////////////////////////////////////////////////////////////////////////////////
//                                      Bridge                                   //
///////////////////////////////////////////////////////////////////////////////////
    class Bridge final : public Analog {
    public:
        explicit Bridge(config::Parser &parser, TaskHandle task_handle,
                        const std::string &name)
                : Analog(parser, task_handle, name),
                  bridge_config(parser) {
        }

        int32 create_ni_channel() override {
            return ni::NiDAQmxInterface::CreateAIBridgeChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->min_val,
                    this->max_val,
                    this->units,
                    this->bridge_config.ni_bridge_config,
                    this->bridge_config.voltage_excit_source,
                    this->bridge_config.voltage_excit_val,
                    this->bridge_config.nominal_bridge_resistance,
                    this->scale_name.c_str()
            );
        }

    private:
        BridgeConfig bridge_config;
    };

///////////////////////////////////////////////////////////////////////////////////
//                              Strain Gage                                      //
///////////////////////////////////////////////////////////////////////////////////
    class StrainGage final : public Analog {
    public:
        static inline int32_t get_strain_config(std::string s) {
            if (s == "FullBridgeI") return DAQmx_Val_FullBridgeI;
            if (s == "FullBridgeII") return DAQmx_Val_FullBridgeII;
            if (s == "FullBridgeIII") return DAQmx_Val_FullBridgeIII;
            if (s == "HalfBridgeI") return DAQmx_Val_HalfBridgeI;
            if (s == "HalfBridgeII") return DAQmx_Val_HalfBridgeII;
            if (s == "QuarterBridgeI") return DAQmx_Val_QuarterBridgeI;
            if (s == "QuarterBridgeII") return DAQmx_Val_QuarterBridgeII;
            return DAQmx_Val_FullBridgeI;
        }

        explicit StrainGage(config::Parser &parser, TaskHandle task_handle,
                            const std::string &name)
                : Analog(parser, task_handle, name),
                  strain_config(
                          get_strain_config(
                                  parser.required<std::string>("strain_config"))),
                  excitation_config(parser),
                  gage_factor(parser.required<double>("gage_factor")),
                  initial_bridge_voltage(
                          parser.required<double>("initial_bridge_voltage")),
                  nominal_gage_resistance(
                          parser.required<double>("nominal_gage_resistance")),
                  poisson_ratio(parser.required<double>("poisson_ratio")),
                  lead_wire_resistance(
                          parser.required<double>("lead_wire_resistance")) {
        }

        int32 create_ni_channel() override {
            return ni::NiDAQmxInterface::CreateAIStrainGageChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
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
                    this->scale_name.c_str()
            );
        }

    private:
        int32_t strain_config;
        VoltageExcitationConfig excitation_config;
        double gage_factor;
        double initial_bridge_voltage;
        double nominal_gage_resistance;
        double poisson_ratio;
        double lead_wire_resistance;
    };

///////////////////////////////////////////////////////////////////////////////////
//                                      Rosette Strain Gage                      //
///////////////////////////////////////////////////////////////////////////////////
    class RosetteStrainGage final : public Analog {
    public:
        static inline int32_t get_strain_config(std::string s) {
            if (s == "FullBridgeI") return DAQmx_Val_FullBridgeI;
            if (s == "FullBridgeII") return DAQmx_Val_FullBridgeII;
            if (s == "FullBridgeIII") return DAQmx_Val_FullBridgeIII;
            if (s == "HalfBridgeI") return DAQmx_Val_HalfBridgeI;
            if (s == "HalfBridgeII") return DAQmx_Val_HalfBridgeII;
            if (s == "QuarterBridgeI") return DAQmx_Val_QuarterBridgeI;
            if (s == "QuarterBridgeII") return DAQmx_Val_QuarterBridgeII;
            return DAQmx_Val_FullBridgeI;
        }

        static inline int32_t get_rosette_type(std::string s) {
            if (s == "RectangularRosette") return DAQmx_Val_RectangularRosette;
            if (s == "DeltaRosette") return DAQmx_Val_DeltaRosette;
            if (s == "TeeRosette") return DAQmx_Val_TeeRosette;
            return DAQmx_Val_RectangularRosette;
        }

        static inline int32_t get_rosette_meas_type(std::string s) {
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

        explicit RosetteStrainGage(config::Parser &parser, TaskHandle task_handle,
                                   const std::string &name)
                : Analog(parser, task_handle, name),
                  rosette_type(get_rosette_type(
                          parser.required<std::string>("rosette_type"))),
                  gage_orientation(parser.required<double>("gage_orientation")),
                  rosette_meas_type(
                          get_rosette_meas_type(
                                  parser.required<std::string>("rosette_meas_type"))),
                  strain_config(
                          get_strain_config(
                                  parser.required<std::string>("strain_config"))),
                  excitation_config(parser),
                  gage_factor(parser.required<double>("gage_factor")),
                  nominal_gage_resistance(
                          parser.required<double>("nominal_gage_resistance")),
                  poisson_ratio(parser.required<double>("poisson_ratio")),
                  lead_wire_resistance(
                          parser.required<double>("lead_wire_resistance")) {
        }

        int32 create_ni_channel() override {
            return ni::NiDAQmxInterface::CreateAIRosetteStrainGageChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->min_val,
                    this->max_val,
                    this->rosette_type,
                    this->gage_orientation,
                    &this->rosette_meas_type,
                    1, // bynRosseteMeasTypes // TODO: what is this for
                    this->strain_config,
                    this->excitation_config.excit_source,
                    this->excitation_config.excit_val,
                    this->gage_factor,
                    this->nominal_gage_resistance,
                    this->poisson_ratio,
                    this->lead_wire_resistance
            );
        }

    private:
        int32_t rosette_type;
        double gage_orientation;
        int32 rosette_meas_type;
        int32 strain_config;
        VoltageExcitationConfig excitation_config;
        double gage_factor;
        double nominal_gage_resistance;
        double poisson_ratio;
        double lead_wire_resistance;
    };

///////////////////////////////////////////////////////////////////////////////////
//                                      Microphone                               //
///////////////////////////////////////////////////////////////////////////////////
    class Microphone final : public Analog {
    public:
        explicit Microphone(config::Parser &parser, TaskHandle task_handle,
                            const std::string &name)
                : Analog(parser, task_handle, name),
                  terminal_config(
                          ni::get_terminal_config(
                                  parser.required<std::string>("terminal_config"))),
                  mic_sensitivity(parser.required<double>("mic_sensitivity")),
                  max_snd_press_level(parser.required<double>("max_snd_press_level")),
                  excitation_config(parser) {
        }

        int32 create_ni_channel() override {
            return ni::NiDAQmxInterface::CreateAIMicrophoneChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->terminal_config,
                    this->units,
                    this->mic_sensitivity,
                    this->max_snd_press_level,
                    this->excitation_config.excit_source,
                    this->excitation_config.excit_val,
                    this->scale_name.c_str()
            );
        }

    private:
        double mic_sensitivity;
        double max_snd_press_level;
        CurrentExcitationConfig excitation_config;
        int32 terminal_config = 0;
    };

///////////////////////////////////////////////////////////////////////////////////
//                                      Frequency                                //
///////////////////////////////////////////////////////////////////////////////////
    class FrequencyVoltage final : public Analog {
    public:
        explicit FrequencyVoltage(config::Parser &parser, TaskHandle task_handle,
                                  const std::string &name)
                : Analog(parser, task_handle, name),
                  threshold_level(parser.required<double>("threshold_level")),
                  hysteresis(parser.required<double>("hysteresis")) {
            // get the device name by reading up to delimiter
            size_t pos = name.find("/");
            this->name = name.substr(0, pos) + "/ctr" + std::to_string(
                    parser.required<std::uint64_t>("port"));
        }

        int32 create_ni_channel() override {
            return ni::NiDAQmxInterface::CreateAIFreqVoltageChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->min_val,
                    this->max_val,
                    this->units,
                    this->threshold_level,
                    this->hysteresis,
                    this->scale_name.c_str()
            );
        }

    private:
        double threshold_level;
        double hysteresis;
    };

///////////////////////////////////////////////////////////////////////////////////
//                                      Pressure                                 //
///////////////////////////////////////////////////////////////////////////////////
    class PressureBridgeTwoPointLin final : public Analog {
    public:
        explicit PressureBridgeTwoPointLin(config::Parser &parser,
                                           TaskHandle task_handle,
                                           const std::string &name)
                : Analog(parser, task_handle, name),
                  bridge_config(parser),
                  two_point_lin_config(parser) {
        }

        int32 create_ni_channel() override {
            return ni::NiDAQmxInterface::CreateAIPressureBridgeTwoPointLinChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
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
                    this->scale_name.c_str()
            );
        }

    private:
        BridgeConfig bridge_config;
        TwoPointLinConfig two_point_lin_config;
    };

    class PressureBridgeTable final : public Analog {
    public:
        explicit PressureBridgeTable(config::Parser &parser, TaskHandle task_handle,
                                     const std::string &name)
                : Analog(parser, task_handle, name),
                  bridge_config(parser),
                  table_config(parser) {
        }

        int32 create_ni_channel() override {
            return ni::NiDAQmxInterface::CreateAIPressureBridgeTableChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->min_val,
                    this->max_val,
                    this->units,
                    this->bridge_config.ni_bridge_config,
                    this->bridge_config.voltage_excit_source,
                    this->bridge_config.voltage_excit_val,
                    this->bridge_config.nominal_bridge_resistance,
                    this->table_config.electrical_vals,
                    this->table_config.num_eletrical_vals,
                    this->table_config.electrical_units,
                    this->table_config.physical_vals,
                    this->table_config.num_physical_vals,
                    this->table_config.physical_units,
                    this->scale_name.c_str()
            );
        }

    private:
        BridgeConfig bridge_config;
        TableConfig table_config;
    };

    class PressureBridgePolynomial final : public Analog {
    public:
        explicit PressureBridgePolynomial(config::Parser &parser,
                                          TaskHandle task_handle,
                                          const std::string &name)
                : Analog(parser, task_handle, name),
                  bridge_config(parser),
                  polynomial_config(parser) {
        }

        int32 create_ni_channel() override {
            return ni::NiDAQmxInterface::CreateAIPressureBridgePolynomialChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
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
                    this->scale_name.c_str()
            );
        }

    private:
        BridgeConfig bridge_config;
        PolynomialConfig polynomial_config;
    };

///////////////////////////////////////////////////////////////////////////////////
//                                      Force                                    //
///////////////////////////////////////////////////////////////////////////////////
    class ForceBridgePolynomial final : public Analog {
    public:
        explicit ForceBridgePolynomial(config::Parser &parser, TaskHandle task_handle,
                                       const std::string &name)
                : Analog(parser, task_handle, name),
                  bridge_config(parser),
                  polynomial_config(parser) {
        }

        int32 create_ni_channel() override {
            return ni::NiDAQmxInterface::CreateAIForceBridgePolynomialChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
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
                    this->scale_name.c_str()
            );
        }

    private:
        BridgeConfig bridge_config;
        PolynomialConfig polynomial_config;
    };

    class ForceBridgeTable final : public Analog {
    public:
        explicit ForceBridgeTable(config::Parser &parser, TaskHandle task_handle,
                                  const std::string &name)
                : Analog(parser, task_handle, name),
                  bridge_config(parser),
                  table_config(parser) {
        }

        int32 create_ni_channel() override {
            return ni::NiDAQmxInterface::CreateAIForceBridgeTableChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->min_val,
                    this->max_val,
                    this->units,
                    this->bridge_config.ni_bridge_config,
                    this->bridge_config.voltage_excit_source,
                    this->bridge_config.voltage_excit_val,
                    this->bridge_config.nominal_bridge_resistance,
                    this->table_config.electrical_vals,
                    this->table_config.num_eletrical_vals,
                    this->table_config.electrical_units,
                    this->table_config.physical_vals,
                    this->table_config.num_physical_vals,
                    this->table_config.physical_units,
                    this->scale_name.c_str()
            );
        }

    private:
        BridgeConfig bridge_config;
        TableConfig table_config;
    };

    class ForceBridgeTwoPointLin final : public Analog {
    public:
        explicit ForceBridgeTwoPointLin(config::Parser &parser, TaskHandle task_handle,
                                        const std::string &name)
                : Analog(parser, task_handle, name),
                  bridge_config(parser),
                  two_point_lin_config(parser) {
        }

        int32 create_ni_channel() override {
            return ni::NiDAQmxInterface::CreateAIForceBridgeTwoPointLinChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
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
                    this->scale_name.c_str()
            );
        }

    private:
        BridgeConfig bridge_config;
        TwoPointLinConfig two_point_lin_config;
    };

///////////////////////////////////////////////////////////////////////////////////
//                                      Velocity                                 //
///////////////////////////////////////////////////////////////////////////////////
    class VelocityIEPE final : public Analog {
    public:
        explicit VelocityIEPE(config::Parser &parser, TaskHandle task_handle,
                              const std::string &name)
                : Analog(parser, task_handle, name),
                  sensitivity(parser.required<double>("sensitivity")),
                  excitation_config(parser),
                  terminal_config(
                          ni::get_terminal_config(
                                  parser.required<std::string>("terminal_config"))),
                  sensitivity_units(
                          get_units(parser.required<std::string>("sensitivity_units"),
                                    parser)) {
        }

        int32 create_ni_channel() override {
            return ni::NiDAQmxInterface::CreateAIVelocityIEPEChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->terminal_config,
                    this->min_val,
                    this->max_val,
                    this->units,
                    this->sensitivity,
                    this->sensitivity_units,
                    this->excitation_config.excit_source,
                    this->excitation_config.excit_val,
                    this->scale_name.c_str()
            );
        }

    private:
        int32_t sensitivity_units;
        double sensitivity;
        CurrentExcitationConfig excitation_config;
        int32_t terminal_config = 0;
    };

///////////////////////////////////////////////////////////////////////////////////
//                                      Torque                                   //
///////////////////////////////////////////////////////////////////////////////////
    class TorqueBridgeTwoPointLin final : public Analog {
    public:
        explicit TorqueBridgeTwoPointLin(config::Parser &parser, TaskHandle task_handle,
                                         const std::string &name)
                : Analog(parser, task_handle, name),
                  bridge_config(parser),
                  two_point_lin_config(parser) {
        }

        int32 create_ni_channel() override {
            return ni::NiDAQmxInterface::CreateAITorqueBridgeTwoPointLinChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
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
                    this->scale_name.c_str()
            );
        }

    private:
        BridgeConfig bridge_config;
        TwoPointLinConfig two_point_lin_config;
    };

    class TorqueBridgePolynomial final : public Analog {
    public:
        explicit TorqueBridgePolynomial(config::Parser &parser, TaskHandle task_handle,
                                        const std::string &name)
                : Analog(parser, task_handle, name),
                  bridge_config(parser),
                  polynomial_config(parser) {
        }

        int32 create_ni_channel() override {
            return ni::NiDAQmxInterface::CreateAITorqueBridgePolynomialChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
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
                    this->scale_name.c_str()
            );
        }

    private:
        BridgeConfig bridge_config;
        PolynomialConfig polynomial_config;
    };

    class TorqueBridgeTable final : public Analog {
    public:
        explicit TorqueBridgeTable(config::Parser &parser, TaskHandle task_handle,
                                   const std::string &name)
                : Analog(parser, task_handle, name),
                  bridge_config(parser),
                  table_config(parser) {
        }

        int32 create_ni_channel() override {
            return ni::NiDAQmxInterface::CreateAITorqueBridgeTableChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->min_val,
                    this->max_val,
                    this->units,
                    this->bridge_config.ni_bridge_config,
                    this->bridge_config.voltage_excit_source,
                    this->bridge_config.voltage_excit_val,
                    this->bridge_config.nominal_bridge_resistance,
                    this->table_config.electrical_vals,
                    this->table_config.num_eletrical_vals,
                    this->table_config.electrical_units,
                    this->table_config.physical_vals,
                    this->table_config.num_physical_vals,
                    this->table_config.physical_units,
                    this->scale_name.c_str()
            );
        }

    private:
        BridgeConfig bridge_config;
        TableConfig table_config;
    };

    class ForceIEPE final : public Analog {
    public:
        explicit ForceIEPE(config::Parser &parser, TaskHandle task_handle,
                           const std::string &name)
                : Analog(parser, task_handle, name),
                  sensitivity(parser.required<double>("sensitivity")),
                  excitation_config(parser),
                  terminal_config(
                          ni::get_terminal_config(
                                  parser.required<std::string>("terminal_config"))),
                  sensitivity_units(
                          get_units(parser.required<std::string>("sensitivity_units"),
                                    parser)) {
        }

        int32 create_ni_channel() override {
            return ni::NiDAQmxInterface::CreateAIForceIEPEChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->terminal_config,
                    this->min_val,
                    this->max_val,
                    this->units,
                    this->sensitivity,
                    this->sensitivity_units,
                    this->excitation_config.excit_source,
                    this->excitation_config.excit_val,
                    this->scale_name.c_str()
            );
        }

    private:
        int32_t sensitivity_units;
        double sensitivity;
        CurrentExcitationConfig excitation_config;
        int32 terminal_config = 0;
    };

///////////////////////////////////////////////////////////////////////////////////
//                                      Charge                                   //
///////////////////////////////////////////////////////////////////////////////////
    class Charge final : public Analog {
    public:
        explicit Charge(config::Parser &parser, TaskHandle task_handle,
                        const std::string &name)
                : Analog(parser, task_handle, name),
                  terminal_config(
                          ni::get_terminal_config(
                                  parser.required<std::string>("terminal_config"))) {
        }

        int32 create_ni_channel() override {
            return ni::NiDAQmxInterface::CreateAIChargeChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->terminal_config,
                    this->min_val,
                    this->max_val,
                    this->units,
                    this->scale_name.c_str()
            );
        }

    private:
        int32 terminal_config = 0;
    };
} // namespace ni
