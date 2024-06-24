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

static inline int32_t getTerminalConfig(std::string terminal_config) {
    if (terminal_config == "PseudoDiff") return DAQmx_Val_PseudoDiff;
    if (terminal_config == "Diff") return DAQmx_Val_Diff;
    if (terminal_config == "NRSE") return DAQmx_Val_NRSE;
    if (terminal_config == "RSE") return DAQmx_Val_RSE;
    return DAQmx_Val_Cfg_Default;
}

static inline int32_t get_bridge_config(std::string s){
    if(s == "FullBridge") return DAQmx_Val_FullBridge;
    if(s == "HalfBridge") return DAQmx_Val_HalfBridge;
    if(s == "QuarterBridge") return DAQmx_Val_QuarterBridge;
    return DAQmx_Val_FullBridge;
}

static inline int32_t getResistanceConfig(std::string s){
    if(s == "2Wire") return DAQmx_Val_2Wire;
    if(s == "3Wire") return DAQmx_Val_3Wire;
    if(s == "4Wire") return DAQmx_Val_4Wire;
    return DAQmx_Val_2Wire;
}

static inline int32_t getExcitationSrc(std::string s){
        if(s == "Internal") return DAQmx_Val_Internal;
        if(s == "External") return DAQmx_Val_External;
        if(s == "None") return DAQmx_Val_None;
        return DAQmx_Val_None;
}

// TODO: make one for current excitation for correct parsing
typedef struct ExcitationConfig {
    int32_t voltageExcitSource;
    double voltageExcitVal;
    double minValForExcitation; // optional
    double maxValForExcitation; //optional
    bool32 useExcitForScaling; //optional

    
    
    ExcitationConfig(config::Parser &parser)
        : voltageExcitSource(getExcitationSrc(parser.required<std::string>("voltage_excit_source"))),
          voltageExcitVal(parser.required<double>("voltage_excit_val")),
          minValForExcitation(parser.optional<double>("min_val_for_excitation", 0)),
          maxValForExcitation(parser.optional<double>("max_val_for_excitation", 0)),
          useExcitForScaling(parser.optional<bool32>("use_excit_for_scaling", 0)) {
    }
} ExcitationConfig;

typedef struct BridgeConfig {
    int32_t niBridgeConfig;
    int32_t voltageExcitSource;
    double voltageExcitVal;
    double nominalBridgeResistance;

    BridgeConfig() = default;

    BridgeConfig(config::Parser &parser)
        : niBridgeConfig(get_bridge_config(parser.required<std::string>("bridge_config"))),
          voltageExcitSource(getExcitationSrc(parser.required<std::string>("voltage_excit_source"))),
          voltageExcitVal(parser.required<double>("voltage_excit_val")),
          nominalBridgeResistance(parser.required<double>("nominal_bridge_resistance")) {
    }
} BridgeConfig;

typedef struct PolynomialConfig {
    float64 *forwardCoeffs;
    uint32_t numForwardCoeffs;
    float64 *reverseCoeffs;
    uint32_t numReverseCoeffs;
    int32_t electricalUnits;
    int32_t physicalUnits;

    PolynomialConfig() = default;

    PolynomialConfig(config::Parser &parser)
        : numForwardCoeffs(parser.required<uint32_t>("num_forward_coeffs")),
          numReverseCoeffs(parser.required<uint32_t>("num_reverse_coeffs")){
            
            auto eu = parser.required<std::string>("electrical_units");
            auto pu = parser.required<std::string>("physical_units");
            electricalUnits = ni::UNITS_MAP.at(eu);
            physicalUnits = ni::UNITS_MAP.at(pu);

        if (!parser.ok()) return; // TODO: handle error

        json j = parser.get_json();

        forwardCoeffs = new double[numForwardCoeffs];
        reverseCoeffs = new double[numReverseCoeffs];

        //get forward coeffs (prescale -> scale)
        if (j.contains("forward_coeffs")) {
            forwardCoeffs = new double[numForwardCoeffs];
            for (uint32_t i = 0; i < numForwardCoeffs; i++) {
                forwardCoeffs[i] = j["forward_coeffs"][i];
            }
        }

        ni::NiDAQmxInterface::CalculateReversePolyCoeff(
            forwardCoeffs,
            numForwardCoeffs,
            -1000, //FIXME dont hard code
            1000, //FIXME dont hard code
            numReverseCoeffs,
            -1,
            reverseCoeffs
        ); // FIXME: reversePoly order should be user inputted?
    }

    ~PolynomialConfig() {
        if (forwardCoeffs != nullptr) delete[] forwardCoeffs;
        if (reverseCoeffs != nullptr) delete[] reverseCoeffs;
    }
} PolynomialConfig;

typedef struct TableConfig {
    float64 *electricalVals;
    uint32_t numElectricalVals;
    float64 *physicalVals;
    uint32_t numPhysicalVals;
    int32_t electricalUnits;
    int32_t physicalUnits;

    TableConfig() = default;

    TableConfig(config::Parser &parser)
        : numElectricalVals(parser.required<uint32_t>("num_electrical_vals")),
          numPhysicalVals(parser.required<uint32_t>("num_physical_vals")){
        
        auto eu = parser.required<std::string>("electrical_units");
        auto pu = parser.required<std::string>("physical_units");

        electricalUnits = ni::UNITS_MAP.at(eu);
        physicalUnits = ni::UNITS_MAP.at(pu);

        if (!parser.ok()) return; // TODO: handle error

        json j = parser.get_json();

        //get electrical vals
        if (j.contains("electrical_vals")) {
            electricalVals = new double[numElectricalVals];
            for (uint32_t i = 0; i < numElectricalVals; i++) {
                electricalVals[i] = j["electrical_vals"][i];
            }
        }

        //get physical vals
        if (j.contains("physical_vals")) {
            physicalVals = new double[numPhysicalVals];
            for (uint32_t i = 0; i < numPhysicalVals; i++) {
                physicalVals[i] = j["physical_vals"][i];
            }
        }
    }
} TableConfig;

typedef struct TwoPointLinConfig {
    double firstElectricalVal;
    double secondElectricalVal;
    int32_t electricalUnits;
    double firstPhysicalVal;
    double secondPhysicalVal;
    int32_t physicalUnits;

    TwoPointLinConfig() = default;

    TwoPointLinConfig(config::Parser &parser)
        : firstElectricalVal(parser.required<double>("first_electrical_val")),
          secondElectricalVal(parser.required<double>("second_electrical_val")),
          firstPhysicalVal(parser.required<double>("first_physical_val")),
          secondPhysicalVal(parser.required<double>("second_physical_val")){
            auto eu = parser.required<std::string>("electrical_units");
            auto pu = parser.required<std::string>("physical_units");
            electricalUnits = ni::UNITS_MAP.at(eu);
            physicalUnits = ni::UNITS_MAP.at(pu);
    }
} TwoPointLinConfig;


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

    virtual int32 createNIChannel() {
        LOG(INFO) << "Creating Analog Channel";
        return 0;
    }

    static std::unique_ptr<ScaleConfig> getScaleConfig(config::Parser &parser) {
        // TODO check if custom scale and channel exist
        std::string scale_name = std::to_string(parser.required<uint32_t>("channel")) +
                                 "_scale";
        auto scale_parser = parser.child("custom_scale");
        return std::make_unique<ScaleConfig>(scale_parser, scale_name);
    }

    int32 createNIScale() {
        if (this->scale_config->type == "none") return 0;
        return this->scale_config->createNIScale();
    }

    explicit Analog(config::Parser &parser, TaskHandle task_handle, std::string name)
        : task_handle(task_handle),
          min_val(parser.optional<float_t>("min_val",0)),
          max_val(parser.optional<float_t>("max_val",0)),
          units(DAQmx_Val_Volts),
          sy_key(parser.required<uint32_t>("channel")),
          name(name),
          type(parser.required<std::string>("type")),
          scale_config(getScaleConfig(parser)) {
        // check name of channel
        if (this->scale_config->type != "none") {
            LOG(INFO) << "Scale type: " << this->scale_config->type;
            this->scale_name = this->scale_config->name;
            this->units = DAQmx_Val_FromCustomScale;
        }
    }

    TaskHandle task_handle = 0;
    std::string scale_name = "";
    double min_val = 0;
    double max_val = 0;
    int32_t units = DAQmx_Val_Volts;
    uint32_t sy_key = 0;
    std::string name = "";
    std::string type = "";

    std::unique_ptr<ScaleConfig> scale_config;
};

///////////////////////////////////////////////////////////////////////////////////
//                                      Voltage                                  //
///////////////////////////////////////////////////////////////////////////////////
/// @brief voltage channel.
class Voltage : public Analog {
public:
    int32_t terminal_config = 0;

    explicit Voltage(config::Parser &parser, TaskHandle task_handle, std::string name)
        : Analog(parser, task_handle, name),
          terminal_config(
              ni::getTerminalConfig(parser.required<std::string>("terminal_config"))) {
    }

    ~Voltage() = default;

    int32 createNIChannel() override {
        if (this->scale_config->type == "none") {
            return ni::NiDAQmxInterface::CreateAIVoltageChan(
                this->task_handle,
                this->name.c_str(),
                "", // name to assign channel
                this->terminal_config,
                this->min_val,
                this->max_val,
                DAQmx_Val_Volts,
                NULL
            );
        } else {
            return ni::NiDAQmxInterface::CreateAIVoltageChan(
                this->task_handle,
                this->name.c_str(),
                "", // name to assign channel
                this->terminal_config,
                this->min_val,
                this->max_val,
                DAQmx_Val_FromCustomScale,
                this->scale_config->name.c_str()
            );
        }
    }
};



/// @brief RMS voltage Channel
class VoltageRMS : public Voltage {
    public:
        explicit VoltageRMS(config::Parser &parser, TaskHandle task_handle, std::string name)
            : Voltage(parser, task_handle, name){}

        ~VoltageRMS() = default;

        int32 createNIChannel() override {
            // TODO: check if scale exists
            return ni::NiDAQmxInterface::CreateAIVoltageRMSChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->terminal_config,
                    this->min_val,
                    this->max_val,
                    DAQmx_Val_Volts,
                    NULL
            );
    }
};

    /// @brief voltage Channel with excitation reference
    class VoltageWithExcit : public Voltage {
        public:
            int32_t bridgeConfig = 0;
            ExcitationConfig excitationConfig;

            explicit VoltageWithExcit(config::Parser &parser, TaskHandle task_handle, std::string name)
                : Voltage(parser, task_handle, name),
                  bridgeConfig(get_bridge_config(parser.required<std::string>("bridge_config"))),
                  excitationConfig(parser){}

            ~VoltageWithExcit() = default;

            int32 createNIChannel() override {
                LOG(INFO) << "Creating Voltage Channel with Excitation Reference";
                if(this->scale_config->type == "none"){
                    return ni::NiDAQmxInterface::CreateAIVoltageChanWithExcit(
                            this->task_handle,
                            this->name.c_str(),
                            "",
                            this->terminal_config,
                            this->min_val,
                            this->max_val,
                            DAQmx_Val_Volts,
                            this->bridgeConfig,
                            this->excitationConfig.voltageExcitSource,
                            this->excitationConfig.voltageExcitVal,
                            this->excitationConfig.minValForExcitation,
                            NULL
                    );
                }
            }
    };


///////////////////////////////////////////////////////////////////////////////////
//                                      Current                                  //
///////////////////////////////////////////////////////////////////////////////////
class Current : public Analog {
public:
    int32_t shuntResistorLoc;
    double extShuntResistorval;
    int32 terminal_config = 0;

    static int32_t getShuntResistorLocation(std::string loc) {
        // TODO: cant find any other options in daqmx.h?
        return DAQmx_Val_Default;
    }

    explicit Current(config::Parser &parser, TaskHandle task_handle, std::string name)
        : Analog(parser, task_handle, name),
          terminal_config(
              ni::getTerminalConfig(parser.required<std::string>("terminal_config"))),
          shuntResistorLoc(
              getShuntResistorLocation(
                  parser.required<std::string>("shunt_resistor_loc"))),
          extShuntResistorval(parser.required<double>("ext_shunt_resistor_val")) {
        std::string u = parser.optional<std::string>("units", "Amps");
        this->units = ni::UNITS_MAP.at(u);
    }

    int32 createNIChannel() override {
        if (this->scale_config->type == "none") {
            return ni::NiDAQmxInterface::CreateAICurrentChan(
                this->task_handle,
                this->name.c_str(),
                "",
                this->terminal_config,
                this->min_val,
                this->max_val,
                this->units,
                this->shuntResistorLoc,
                this->extShuntResistorval,
                NULL
            );
        }
    }
};

class CurrentRMS : public Current{
public:
    explicit CurrentRMS(config::Parser &parser, TaskHandle task_handle, std::string name)
            : Current(parser, task_handle, name) {}

    int32 createNIChannel() override {
        if(this->scale_config->type == "none"){
            return ni::NiDAQmxInterface::CreateAICurrentRMSChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->terminal_config,
                    this->min_val,
                    this->max_val,
                    this->units,
                    this->shuntResistorLoc,
                    this->extShuntResistorval,
                    NULL
            );
        }
    }
};

    ///////////////////////////////////////////////////////////////////////////////////
    //                                       RTD                                     //
    ///////////////////////////////////////////////////////////////////////////////////
    class RTD : public Analog{
        public:
            int32_t rtdType;
            int32_t resistanceConfig;
            ExcitationConfig excitationConfig;
            double r0;

            static int32_t getRTDType(std::string type){
                if(type == "Pt3750") return DAQmx_Val_Pt3750;
                if(type == "PT3851") return DAQmx_Val_Pt3851;
                if(type == "PT3911") return DAQmx_Val_Pt3911;
                if(type == "PT3916") return DAQmx_Val_Pt3916;
                if(type == "PT3920") return DAQmx_Val_Pt3920;
                if(type == "PT3928") return DAQmx_Val_Pt3928;
                if(type == "Custom") return DAQmx_Val_Custom;
                return DAQmx_Val_Pt3750;
            } 

            explicit RTD(config::Parser &parser, TaskHandle task_handle, std::string name)
                    : Analog(parser, task_handle, name),
                      rtdType(getRTDType(parser.required<std::string>("rtd_type"))),
                      resistanceConfig(getResistanceConfig(parser.required<std::string>("resistance_config"))),
                      excitationConfig(parser),
                      r0(parser.required<double>("r0")) {
                        std::string u = parser.optional<std::string>("units", "Amps");
                        this->units = ni::UNITS_MAP.at(u); 
                      }
    
            int32 createNIChannel() override {
                return ni::NiDAQmxInterface::CreateAIRTDChan(
                        this->task_handle,
                        this->name.c_str(),
                        "",
                        this->min_val,
                        this->max_val,
                        this->units,
                        this->rtdType,
                        this->resistanceConfig,
                        this->excitationConfig.voltageExcitSource, //TODO change name to current
                        this->excitationConfig.voltageExcitVal, //TODO change name to current
                        this->r0
                );
            }
    };

///////////////////////////////////////////////////////////////////////////////////
//                                      Temperature                              //
///////////////////////////////////////////////////////////////////////////////////
class Thermocouple : public Analog {
public:
    int32_t thermocoupleType;
    int32_t cjcSource;
    double cjcVal;
    std::string cjcChannel;

    static int32_t getType(std::string type) {
        if (type == "J") return DAQmx_Val_J_Type_TC;
        if (type == "K") return DAQmx_Val_K_Type_TC;
        if (type == "N") return DAQmx_Val_N_Type_TC;
        if (type == "R") return DAQmx_Val_R_Type_TC;
        if (type == "S") return DAQmx_Val_S_Type_TC;
        if (type == "T") return DAQmx_Val_T_Type_TC;
        if (type == "B") return DAQmx_Val_B_Type_TC;
        if (type == "E") return DAQmx_Val_E_Type_TC;

        LOG(ERROR) << "Invalid TC Type";
        return DAQmx_Val_J_Type_TC;
    }

    static int32_t getCJCSource(std::string source) {
        if (source == "BuiltIn") return DAQmx_Val_BuiltIn;
        if (source == "ConstVal") return DAQmx_Val_ConstVal;
        if (source == "Chan") return DAQmx_Val_Chan;
        LOG(ERROR) << "Invalid cjc type";
        return DAQmx_Val_BuiltIn;
    }


    explicit Thermocouple(config::Parser &parser, TaskHandle task_handle,
                          std::string name)
        : Analog(parser, task_handle, name),
          thermocoupleType(getType(parser.required<std::string>("thermocouple_type"))),
          cjcSource(getCJCSource(parser.required<std::string>("cjc_source"))),
          cjcVal(parser.required<double>("cjc_val")) {
        std::string u = parser.optional<std::string>("units", "DegC");
        this->units = ni::UNITS_MAP.at(u); // TODO: make this optional and default to C?
    }

    //cjcChannel(parser.required<std::string>("cjc_channel")) {} FIXME: this property should be take form console


    ///	DAQmxErrChk (DAQmxCreateAIThrmcplChan(taskHandle,"","",0.0,100.0,DAQmx_Val_DegC,DAQmx_Val_J_Type_TC,DAQmx_Val_BuiltIn,25.0,""));

    int32 createNIChannel() override {
        if (this->scale_config->type == "none") {
            return ni::NiDAQmxInterface::CreateAIThrmcplChan(
                this->task_handle,
                this->name.c_str(),
                "",
                this->min_val,
                this->max_val,
                this->units,
                this->thermocoupleType,
                this->cjcSource,
                this->cjcVal,
                ""
            );
        }
    }
};

class TemperatureBuiltInSensor : public Analog{
    public:
        explicit TemperatureBuiltInSensor(config::Parser &parser, TaskHandle task_handle, std::string name){
            this->task_handle = task_handle;
            
            std::string u = parser.optional<std::string>("units", "Volts");
            this->units = ni::UNITS_MAP.at(u);

            size_t pos = name.find("/");

            this->name =  name.substr(0, pos) + "/_boardTempSensor_vs_aignd";
        }

        int32 createNIChannel() override {
                       LOG(INFO) << "Creating Temperature Built In Sensor Channel";
            return ni::NiDAQmxInterface::CreateAITempBuiltInSensorChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->units
            );
        }
};


class ThermistorIEX : public Analog{
    public:
        int32_t resistanceConfig;
        ExcitationConfig excitationConfig;
        double a;
        double b;
        double c;

        explicit ThermistorIEX(config::Parser &parser, TaskHandle task_handle, std::string name)
                : Analog(parser, task_handle, name),
                  resistanceConfig(getResistanceConfig(parser.required<std::string>("resistance_config"))),
                  excitationConfig(parser),
                  a(parser.required<double>("a")),
                  b(parser.required<double>("b")),
                  c(parser.required<double>("c")) {
                    std::string u = parser.optional<std::string>("units", "Volts");
                    this->units = ni::UNITS_MAP.at(u);
                  }

        int32 createNIChannel() override {
            if(this->scale_config->type == "none"){
                return ni::NiDAQmxInterface::CreateAIThrmstrChanIex(
                        this->task_handle,
                        this->name.c_str(),
                        "",
                        this->min_val,
                        this->max_val,
                        this->units,
                        this->resistanceConfig,
                        this->excitationConfig.voltageExcitSource, // current excitation source FIXME
                        this->excitationConfig.voltageExcitVal,    // current excitation val FIXME
                        this->a,
                        this->b,
                        this->c
                );
            }
        }
};


class ThermistorVex : public Analog{
    public:
        int32_t resistanceConfig;
        ExcitationConfig excitationConfig;
        double a;
        double b;
        double c;
        double r1;

        explicit ThermistorVex(config::Parser &parser, TaskHandle task_handle, std::string name)
                : Analog(parser, task_handle, name),
                  resistanceConfig(getResistanceConfig(parser.required<std::string>("resistance_config"))),
                  excitationConfig(parser),
                  a(parser.required<double>("a")),
                  b(parser.required<double>("b")),
                  c(parser.required<double>("c")),
                  r1(parser.required<double>("r1")) {
                    std::string u = parser.optional<std::string>("units", "Volts");
                    this->units = ni::UNITS_MAP.at(u);
                  }

        int32 createNIChannel() override {
            if(this->scale_config->type == "none"){
                return ni::NiDAQmxInterface::CreateAIThrmstrChanVex(
                        this->task_handle,
                        this->name.c_str(),
                        "",
                        this->min_val,
                        this->max_val,
                        this->units,
                        this->resistanceConfig,
                        this->excitationConfig.voltageExcitSource, // current excitation source FIXME
                        this->excitationConfig.voltageExcitVal,    // current excitation val FIXME
                        this->a,
                        this->b,
                        this->c,
                        this->r1
                );
            }
        }
};



///////////////////////////////////////////////////////////////////////////////////
//                                    Acceleration                               //
///////////////////////////////////////////////////////////////////////////////////
/// @brief acceleration channel
class Acceleration : public Analog {
    public:
        double sensitivity;
        int32_t sensitivityUnits;
        ExcitationConfig excitationConfig;
        int32 terminal_config = 0;
        explicit Acceleration(config::Parser &parser, TaskHandle task_handle, std::string name)
                : Analog(parser, task_handle, name),
                  terminal_config(ni::getTerminalConfig(parser.required<std::string>("terminal_config"))),
                  sensitivity(parser.required<double>("sensitivity")),
                  excitationConfig(parser) {
                    std::string u = parser.optional<std::string>("units", "Volts");
                    this->units = ni::UNITS_MAP.at(u);

                    std::string su = parser.optional<std::string>("sensitivity_units", "mVoltsPerG");
                    this->sensitivityUnits = ni::UNITS_MAP.at(su);
                  }

        int32 createNIChannel() override {
            if(this->scale_config->type == "none"){
                return ni::NiDAQmxInterface::CreateAIAccelChan(
                        this->task_handle,
                        this->name.c_str(),
                        "",
                        this->terminal_config,
                        this->min_val,
                        this->max_val,
                        this->units,
                        this->sensitivity,
                        this->sensitivityUnits,
                        this->excitationConfig.voltageExcitSource,
                        this->excitationConfig.voltageExcitVal,
                        NULL
                );
            }
        }

};


/// @brief acceleration channel with 4 wire DC voltage
class Acceleration4WireDCVoltage : public Acceleration {
public:

    explicit Acceleration4WireDCVoltage(config::Parser &parser, TaskHandle task_handle, std::string name)
            : Acceleration(parser, task_handle, name) {}

    int32 createNIChannel() override {
        if(this->scale_config->type == "none"){
            return ni::NiDAQmxInterface::CreateAIAccel4WireDCVoltageChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->terminal_config,
                    this->min_val,
                    this->max_val,
                    this->units,
                    this->sensitivity,
                    this->sensitivityUnits,
                    this->excitationConfig.voltageExcitSource,
                    this->excitationConfig.voltageExcitVal,
                    this->excitationConfig.useExcitForScaling,
                    NULL
            );
        }
    
    }
};

/// @brief acceleration channel with charge
class AccelerationCharge : public Analog {
    public:
        double sensitivity;
        int32_t sensitivityUnits;
        int32 terminal_config = 0;

        explicit AccelerationCharge(config::Parser &parser, TaskHandle task_handle, std::string name)
                : Analog(parser, task_handle, name),
                  terminal_config(ni::getTerminalConfig(parser.required<std::string>("terminal_config"))), 
                  sensitivity(parser.required<double>("sensitivity")) {
                    std::string u = parser.optional<std::string>("units", "Volts");
                    this->units = ni::UNITS_MAP.at(u);

                    std::string su = parser.optional<std::string>("sensitivity_units", "mVoltsPerG");
                    this->sensitivityUnits = ni::UNITS_MAP.at(su);
                  }

        int32 createNIChannel() override {
            if(this->scale_config->type == "none"){
                return ni::NiDAQmxInterface::CreateAIAccelChargeChan(
                        this->task_handle,
                        this->name.c_str(),
                        "",
                        this->terminal_config,
                        this->min_val,
                        this->max_val,
                        this->units,
                        this->sensitivity,
                        this->sensitivityUnits,
                        NULL
                );
            }
        }
};

///////////////////////////////////////////////////////////////////////////////////
//                                      Resistance                               //
///////////////////////////////////////////////////////////////////////////////////
class Resistance : public Analog{
    public:
    int32_t resistanceConfig;
    ExcitationConfig excitationConfig;

    explicit Resistance(config::Parser &parser, TaskHandle task_handle, std::string name)
            : Analog(parser, task_handle, name),
                resistanceConfig(getResistanceConfig(parser.required<std::string>("resistance_config"))),
                excitationConfig(parser) {
                    std::string u = parser.optional<std::string>("units", "Volts");
                    this->units = ni::UNITS_MAP.at(u);
                }

    int32 createNIChannel() override {
        if(this->scale_config->type == "none"){
            return ni::NiDAQmxInterface::CreateAIResistanceChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->min_val,
                    this->max_val,
                    this->units,
                    this->resistanceConfig,
                    this->excitationConfig.voltageExcitSource,
                    this->excitationConfig.voltageExcitVal,
                    NULL
            );
        }
    }
};

///////////////////////////////////////////////////////////////////////////////////
//                                      Bridge                                   //
///////////////////////////////////////////////////////////////////////////////////
class Bridge : public Analog {
    public:
        BridgeConfig bridgeConfig;

        explicit Bridge(config::Parser &parser, TaskHandle task_handle, std::string name)
            : Analog(parser, task_handle, name),
              bridgeConfig(parser) {
                std::string u = parser.optional<std::string>("units", "Volts");
                this->units = ni::UNITS_MAP.at(u);
            }

        int32 createNIChannel() override{
            if(this->scale_config->type == "none"){
                return ni::NiDAQmxInterface::CreateAIBridgeChan(
                        this->task_handle,
                        this->name.c_str(),
                        "",
                        this->min_val,
                        this->max_val,
                        this->units,
                        this->bridgeConfig.niBridgeConfig,
                        this->bridgeConfig.voltageExcitSource,
                        this->bridgeConfig.voltageExcitVal,
                        this->bridgeConfig.nominalBridgeResistance,
                        NULL
                );
            }
        }
};

///////////////////////////////////////////////////////////////////////////////////
//                              Strain Gage                                      //
///////////////////////////////////////////////////////////////////////////////////
class StrainGage : public Analog{
public:
    int32_t strainConfig;
    ExcitationConfig excitationConfig;
    double gageFactor;
    double initialBridgeVoltage;
    double nominalGageResistance;
    double poissonRatio;
    double leadWireResistance;

    static inline int32_t get_strain_config(std::string s){
        if(s == "FullBridgeI") return DAQmx_Val_FullBridgeI;
        if(s == "FullBridgeII") return DAQmx_Val_FullBridgeII;
        if(s == "FullBridgeIII") return DAQmx_Val_FullBridgeIII;
        if(s == "HalfBridgeI") return DAQmx_Val_HalfBridgeI;
        if(s == "HalfBridgeII") return DAQmx_Val_HalfBridgeII;
        if(s == "QuarterBridgeI") return DAQmx_Val_QuarterBridgeI;
        if(s == "QuarterBridgeII") return DAQmx_Val_QuarterBridgeII;
        return DAQmx_Val_FullBridgeI;
    }

    explicit StrainGage(config::Parser &parser, TaskHandle task_handle, std::string name)
            : Analog(parser, task_handle, name),
                strainConfig(get_strain_config(parser.required<std::string>("strain_config"))),
                excitationConfig(parser),
                gageFactor(parser.required<double>("gage_factor")),
                initialBridgeVoltage(parser.required<double>("initial_bridge_voltage")),
                nominalGageResistance(parser.required<double>("nominal_gage_resistance")),
                poissonRatio(parser.required<double>("poisson_ratio")),
                leadWireResistance(parser.required<double>("lead_wire_resistance")) {
                    std::string u = parser.optional<std::string>("units", "Volts");
                    this->units = ni::UNITS_MAP.at(u);
                }

    int32 createNIChannel() override {
        if(this->scale_config->type == "none"){
            return ni::NiDAQmxInterface::CreateAIStrainGageChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->min_val,
                    this->max_val,
                    this->units,
                    this->strainConfig,
                    this->excitationConfig.voltageExcitSource,
                    this->excitationConfig.voltageExcitVal,
                    this->gageFactor,
                    this->initialBridgeVoltage,
                    this->nominalGageResistance,
                    this->poissonRatio,
                    this->leadWireResistance,
                    NULL
            );
        }
    }
};

///////////////////////////////////////////////////////////////////////////////////
//                                      Rosette Strain Gage                      //
///////////////////////////////////////////////////////////////////////////////////
class RosetteStrainGage : public Analog{
public:
    int32_t rosetteType;
    double gageOrientation;
    int32 rosseteMeasType;
    int32 strainConfig;
    ExcitationConfig excitationConfig;
    double gageFactor;
    double nominalGageResistance;
    double poissonRatio;
    double leadWireResistance;

    static inline int32_t get_strain_config(std::string s){
        if(s == "FullBridgeI") return DAQmx_Val_FullBridgeI;
        if(s == "FullBridgeII") return DAQmx_Val_FullBridgeII;
        if(s == "FullBridgeIII") return DAQmx_Val_FullBridgeIII;
        if(s == "HalfBridgeI") return DAQmx_Val_HalfBridgeI;
        if(s == "HalfBridgeII") return DAQmx_Val_HalfBridgeII;
        if(s == "QuarterBridgeI") return DAQmx_Val_QuarterBridgeI;
        if(s == "QuarterBridgeII") return DAQmx_Val_QuarterBridgeII;
        return DAQmx_Val_FullBridgeI;
    }
    static inline int32_t get_rosette_type(std::string s){
        if(s == "RectangularRosette") return DAQmx_Val_RectangularRosette;
        if(s == "DeltaRosette") return DAQmx_Val_DeltaRosette;
        if(s == "TeeRosette") return DAQmx_Val_TeeRosette;
        return DAQmx_Val_RectangularRosette;
    }


    static inline int32_t get_rosette_meas_type(std::string s){
        if(s == "PrincipalStrain1") return DAQmx_Val_PrincipalStrain1;
        if(s == "PrincipalStrain2") return DAQmx_Val_PrincipalStrain2;
        if(s == "PrincipalStrainAngle") return DAQmx_Val_PrincipalStrainAngle;
        if(s == "CartesianStrainX") return DAQmx_Val_CartesianStrainX;
        if(s == "CartesianStrainY") return DAQmx_Val_CartesianStrainY;
        if(s == "CartesianShearStrainXY") return DAQmx_Val_CartesianShearStrainXY;
        if(s == "MaxShearStrain") return DAQmx_Val_MaxShearStrain;
        if(s == "MaxShearStrainAngle") return DAQmx_Val_MaxShearStrainAngle;
        return DAQmx_Val_PrincipalStrain1;
    }

    explicit RosetteStrainGage(config::Parser &parser, TaskHandle task_handle, std::string name)
            : Analog(parser, task_handle, name),
                rosetteType(get_rosette_type(parser.required<std::string>("rosette_type"))),
                gageOrientation(parser.required<double>("gage_orientation")),
                rosseteMeasType(get_rosette_meas_type(parser.required<std::string>("rosette_meas_type"))),
                strainConfig(get_strain_config(parser.required<std::string>("strain_config"))),
                excitationConfig(parser),
                gageFactor(parser.required<double>("gage_factor")),
                nominalGageResistance(parser.required<double>("nominal_gage_resistance")),
                poissonRatio(parser.required<double>("poisson_ratio")),
                leadWireResistance(parser.required<double>("lead_wire_resistance")) {
                }

    int32 createNIChannel() override {
        return ni::NiDAQmxInterface::CreateAIRosetteStrainGageChan(
                this->task_handle,
                this->name.c_str(),
                "",
                this->min_val,
                this->max_val,
                this->rosetteType,
                this->gageOrientation,
                &this->rosseteMeasType,
                1, // bynRosseteMeasTypes // TODO: what is this for
                this->strainConfig,
                this->excitationConfig.voltageExcitSource,
                this->excitationConfig.voltageExcitVal,
                this->gageFactor,
                this->nominalGageResistance,
                this->poissonRatio,
                this->leadWireResistance
        );
    }
};

///////////////////////////////////////////////////////////////////////////////////
//                                      Microphone                               //
///////////////////////////////////////////////////////////////////////////////////
class Microphone : public Analog{
    public:
        double micSensitivity;
        double maxSndPressLevel;
        ExcitationConfig excitationConfig;
        int32 terminal_config = 0;

        explicit Microphone(config::Parser &parser, TaskHandle task_handle, std::string name)
                : Analog(parser, task_handle, name),
                  terminal_config(ni::getTerminalConfig(parser.required<std::string>("terminal_config"))),  
                  micSensitivity(parser.required<double>("mic_sensitivity")),
                  maxSndPressLevel(parser.required<double>("max_snd_press_level")),
                  excitationConfig(parser) {
                    std::string u = parser.optional<std::string>("units", "Volts");
                    this->units = ni::UNITS_MAP.at(u);
                  }

        int32 createNIChannel() override {
            if(this->scale_config->type == "none"){
                return ni::NiDAQmxInterface::CreateAIMicrophoneChan(
                        this->task_handle,
                        this->name.c_str(),
                        "",
                        this->terminal_config,
                        this->units,
                        this->micSensitivity,
                        this->maxSndPressLevel,
                        this->excitationConfig.voltageExcitSource,
                        this->excitationConfig.voltageExcitVal,
                        NULL
                );
            }
        }
};

///////////////////////////////////////////////////////////////////////////////////
//                                      Frequency                                //
///////////////////////////////////////////////////////////////////////////////////
class FrequencyVoltage : public Analog{
public:
    double thresholdLevel;
    double hysteresis;

    explicit FrequencyVoltage(config::Parser &parser, TaskHandle task_handle, std::string name)
            : Analog(parser, task_handle, name),
                thresholdLevel(parser.required<double>("threshold_level")),
                hysteresis(parser.required<double>("hysteresis")) {
                    std::string u = parser.optional<std::string>("units", "Volts");
                    this->units = ni::UNITS_MAP.at(u);

                    // get the device name by reading up to delimitn / 
                    size_t pos = name.find("/");
                    this->name = name.substr(0, pos) + "/ctr" + std::to_string(parser.required<std::uint64_t>("port"));
                }
    int32 createNIChannel() override {
        if(this->scale_config->type == "none"){
            return ni::NiDAQmxInterface::CreateAIFreqVoltageChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->min_val,
                    this->max_val,
                    this->units,
                    this->thresholdLevel,
                    this->hysteresis,
                    NULL
            );
        }
    }
};

///////////////////////////////////////////////////////////////////////////////////
//                                      Pressure                                 //
///////////////////////////////////////////////////////////////////////////////////

class PressureBridgeTwoPointLin : public Analog{
    public:
        BridgeConfig bridgeConfig;
        TwoPointLinConfig twoPointLinConfig;

        explicit PressureBridgeTwoPointLin(config::Parser &parser, TaskHandle task_handle, std::string name)
                : Analog(parser, task_handle, name),
                  bridgeConfig(parser),
                  twoPointLinConfig(parser) {
                    std::string u = parser.optional<std::string>("units", "Volts");
                    this->units = ni::UNITS_MAP.at(u);
                  }

        int32 createNIChannel() override {
            if(this->scale_config->type == "none"){
                return ni::NiDAQmxInterface::CreateAIPressureBridgeTwoPointLinChan(
                        this->task_handle,
                        this->name.c_str(),
                        "",
                        this->min_val,
                        this->max_val,
                        this->units,
                        this->bridgeConfig.niBridgeConfig,
                        this->bridgeConfig.voltageExcitSource,
                        this->bridgeConfig.voltageExcitVal,
                        this->bridgeConfig.nominalBridgeResistance,
                        this->twoPointLinConfig.firstElectricalVal,
                        this->twoPointLinConfig.secondElectricalVal,
                        this->twoPointLinConfig.electricalUnits,
                        this->twoPointLinConfig.firstPhysicalVal,
                        this->twoPointLinConfig.secondPhysicalVal,
                        this->twoPointLinConfig.physicalUnits,
                        NULL
                );
            }
        }
};

class PressureBridgeTable : public Analog{
    public:
        BridgeConfig bridgeConfig;
        TableConfig tableConfig;

        explicit PressureBridgeTable(config::Parser &parser, TaskHandle task_handle, std::string name)
                : Analog(parser, task_handle, name),
                  bridgeConfig(parser),
                  tableConfig(parser) {
                    std::string u = parser.optional<std::string>("units", "Volts");
                    this->units = ni::UNITS_MAP.at(u);
                  }

        int32 createNIChannel() override {
            if(this->scale_config->type == "none"){
                return ni::NiDAQmxInterface::CreateAIPressureBridgeTableChan(
                        this->task_handle,
                        this->name.c_str(),
                        "",
                        this->min_val,
                        this->max_val,
                        this->units,
                        this->bridgeConfig.niBridgeConfig,
                        this->bridgeConfig.voltageExcitSource,
                        this->bridgeConfig.voltageExcitVal,
                        this->bridgeConfig.nominalBridgeResistance,
                        this->tableConfig.electricalVals,
                        this->tableConfig.numElectricalVals,
                        this->tableConfig.electricalUnits,
                        this->tableConfig.physicalVals,
                        this->tableConfig.numPhysicalVals,
                        this->tableConfig.physicalUnits,
                        NULL
                );
            }
        }
};

class PressureBridgePolynomial : public Analog{
public:
    BridgeConfig bridgeConfig;
    PolynomialConfig polynomialConfig;

    explicit PressureBridgePolynomial(config::Parser &parser, TaskHandle task_handle, std::string name)
            : Analog(parser, task_handle, name),
                bridgeConfig(parser),
                polynomialConfig(parser) {
                    std::string u = parser.optional<std::string>("units", "Volts");
                    this->units = ni::UNITS_MAP.at(u);
                }

    int32 createNIChannel() override {
        if(this->scale_config->type == "none"){
            return ni::NiDAQmxInterface::CreateAIPressureBridgePolynomialChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->min_val,
                    this->max_val,
                    this->units,
                    this->bridgeConfig.niBridgeConfig,
                    this->bridgeConfig.voltageExcitSource,
                    this->bridgeConfig.voltageExcitVal,
                    this->bridgeConfig.nominalBridgeResistance,
                    this->polynomialConfig.forwardCoeffs,
                    this->polynomialConfig.numForwardCoeffs,
                    this->polynomialConfig.reverseCoeffs,
                    this->polynomialConfig.numReverseCoeffs,
                    this->polynomialConfig.electricalUnits,
                    this->polynomialConfig.physicalUnits,
                    NULL
            );
        }
    }
};

///////////////////////////////////////////////////////////////////////////////////
//                                      Force                                    //
///////////////////////////////////////////////////////////////////////////////////
class ForceBridgePolynomial : public Analog{
public:
    BridgeConfig bridgeConfig;
    PolynomialConfig polynomialConfig;

    explicit ForceBridgePolynomial(config::Parser &parser, TaskHandle task_handle, std::string name)
            : Analog(parser, task_handle, name),
                bridgeConfig(parser),
                polynomialConfig(parser) {
                    std::string u = parser.optional<std::string>("units", "Volts");
                    this->units = ni::UNITS_MAP.at(u);
                }

    int32 createNIChannel() override {
        if(this->scale_config->type == "none"){
            return ni::NiDAQmxInterface::CreateAIForceBridgePolynomialChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->min_val,
                    this->max_val,
                    this->units,
                    this->bridgeConfig.niBridgeConfig,
                    this->bridgeConfig.voltageExcitSource,
                    this->bridgeConfig.voltageExcitVal,
                    this->bridgeConfig.nominalBridgeResistance,
                    this->polynomialConfig.forwardCoeffs,
                    this->polynomialConfig.numForwardCoeffs,
                    this->polynomialConfig.reverseCoeffs,
                    this->polynomialConfig.numReverseCoeffs,
                    this->polynomialConfig.electricalUnits,
                    this->polynomialConfig.physicalUnits,
                    NULL
            );
        }
    }
};

class ForceBridgeTable : public Analog{
    public:
        BridgeConfig bridgeConfig;
        TableConfig tableConfig;

        explicit ForceBridgeTable(config::Parser &parser, TaskHandle task_handle, std::string name)
                : Analog(parser, task_handle, name),
                  bridgeConfig(parser),
                  tableConfig(parser) {
                    std::string u = parser.optional<std::string>("units", "Volts");
                    this->units = ni::UNITS_MAP.at(u);
                  }

        int32 createNIChannel() override {
            if(this->scale_config->type == "none"){
                return ni::NiDAQmxInterface::CreateAIForceBridgeTableChan(
                        this->task_handle,
                        this->name.c_str(),
                        "",
                        this->min_val,
                        this->max_val,
                        this->units,
                        this->bridgeConfig.niBridgeConfig,
                        this->bridgeConfig.voltageExcitSource,
                        this->bridgeConfig.voltageExcitVal,
                        this->bridgeConfig.nominalBridgeResistance,
                        this->tableConfig.electricalVals,
                        this->tableConfig.numElectricalVals,
                        this->tableConfig.electricalUnits,
                        this->tableConfig.physicalVals,
                        this->tableConfig.numPhysicalVals,
                        this->tableConfig.physicalUnits,
                        NULL
                );
            }
        }
};

class ForceBridgeTwoPointLin : public Analog{
    public:
        BridgeConfig bridgeConfig;
        TwoPointLinConfig twoPointLinConfig;

        explicit ForceBridgeTwoPointLin(config::Parser &parser, TaskHandle task_handle, std::string name)
                : Analog(parser, task_handle, name),
                  bridgeConfig(parser),
                  twoPointLinConfig(parser) {
                    std::string u = parser.optional<std::string>("units", "Volts");
                    this->units = ni::UNITS_MAP.at(u);
                  }

        int32 createNIChannel() override {
            if(this->scale_config->type == "none"){
                return ni::NiDAQmxInterface::CreateAIForceBridgeTwoPointLinChan(
                        this->task_handle,
                        this->name.c_str(),
                        "",
                        this->min_val,
                        this->max_val,
                        this->units,
                        this->bridgeConfig.niBridgeConfig,
                        this->bridgeConfig.voltageExcitSource,
                        this->bridgeConfig.voltageExcitVal,
                        this->bridgeConfig.nominalBridgeResistance,
                        this->twoPointLinConfig.firstElectricalVal,
                        this->twoPointLinConfig.secondElectricalVal,
                        this->twoPointLinConfig.electricalUnits,
                        this->twoPointLinConfig.firstPhysicalVal,
                        this->twoPointLinConfig.secondPhysicalVal,
                        this->twoPointLinConfig.physicalUnits,
                        NULL
                );
            }
        }
};


///////////////////////////////////////////////////////////////////////////////////
//                                      Velocity                                 //
///////////////////////////////////////////////////////////////////////////////////
class VelocityIEPE : public Analog{
    public:
        int32_t sensitivityUnits;
        double sensitivity;
        ExcitationConfig excitationConfig;
        int32_t terminal_config = 0;

        explicit VelocityIEPE(config::Parser &parser, TaskHandle task_handle, std::string name)
                : Analog(parser, task_handle, name),
                  sensitivity(parser.required<double>("sensitivity")),
                  excitationConfig(parser),
                  terminal_config(ni::getTerminalConfig(parser.required<std::string>("terminal_config"))) {
                    std::string u = parser.optional<std::string>("units", "Volts");
                    this->units = ni::UNITS_MAP.at(u);

                    auto su = parser.optional<std::string>("sensitivity_units", "mVoltsPerG");
                    this->sensitivityUnits = ni::UNITS_MAP.at(su);
                  }

        int32 createNIChannel() override {
            if(this->scale_config->type == "none"){
                return ni::NiDAQmxInterface::CreateAIVelocityIEPEChan(
                        this->task_handle,
                        this->name.c_str(),
                        "",
                        this->terminal_config,
                        this->min_val,
                        this->max_val,
                        this->units,
                        this->sensitivity,
                        this->sensitivityUnits,
                        this->excitationConfig.voltageExcitSource,
                        this->excitationConfig.voltageExcitVal,
                        NULL
                );
            }
        }
};

///////////////////////////////////////////////////////////////////////////////////
//                                      Torque                                   //
///////////////////////////////////////////////////////////////////////////////////
class TorqueBridgeTwoPointLin : public Analog{
    public:
        BridgeConfig bridgeConfig;
        TwoPointLinConfig twoPointLinConfig;
        explicit TorqueBridgeTwoPointLin(config::Parser &parser, TaskHandle task_handle, std::string name)
                : Analog(parser, task_handle, name),
                  bridgeConfig(parser),
                  twoPointLinConfig(parser) {
                    std::string u = parser.optional<std::string>("units", "Volts");
                    this->units = ni::UNITS_MAP.at(u);
                }

        int32 createNIChannel() override {
            if(this->scale_config->type == "none"){
                return ni::NiDAQmxInterface::CreateAITorqueBridgeTwoPointLinChan(
                        this->task_handle,
                        this->name.c_str(),
                        "",
                        this->min_val,
                        this->max_val,
                        this->units,
                        this->bridgeConfig.niBridgeConfig,
                        this->bridgeConfig.voltageExcitSource,
                        this->bridgeConfig.voltageExcitVal,
                        this->bridgeConfig.nominalBridgeResistance,
                        this->twoPointLinConfig.firstElectricalVal,
                        this->twoPointLinConfig.secondElectricalVal,
                        this->twoPointLinConfig.electricalUnits,
                        this->twoPointLinConfig.firstPhysicalVal,
                        this->twoPointLinConfig.secondPhysicalVal,
                        this->twoPointLinConfig.physicalUnits,
                        NULL
                );
            }
        }
};

class TorqueBridgePolynomial : public Analog{
    public:
        BridgeConfig bridgeConfig;
        PolynomialConfig polynomialConfig;

        explicit TorqueBridgePolynomial(config::Parser &parser, TaskHandle task_handle, std::string name)
                : Analog(parser, task_handle, name),
                  bridgeConfig(parser),
                  polynomialConfig(parser) {
                    std::string u = parser.optional<std::string>("units", "Volts");
                    this->units = ni::UNITS_MAP.at(u);
                }

        int32 createNIChannel() override {
            if(this->scale_config->type == "none"){
                return ni::NiDAQmxInterface::CreateAITorqueBridgePolynomialChan(
                        this->task_handle,
                        this->name.c_str(),
                        "",
                        this->min_val,
                        this->max_val,
                        this->units,
                        this->bridgeConfig.niBridgeConfig,
                        this->bridgeConfig.voltageExcitSource,
                        this->bridgeConfig.voltageExcitVal,
                        this->bridgeConfig.nominalBridgeResistance,
                        this->polynomialConfig.forwardCoeffs,
                        this->polynomialConfig.numForwardCoeffs,
                        this->polynomialConfig.reverseCoeffs,
                        this->polynomialConfig.numReverseCoeffs,
                        this->polynomialConfig.electricalUnits,
                        this->polynomialConfig.physicalUnits,
                        NULL
                );
            }
        }
};


class TorqueBridgeTable : public Analog{
    public:
        BridgeConfig bridgeConfig;
        TableConfig tableConfig;

        explicit TorqueBridgeTable(config::Parser &parser, TaskHandle task_handle, std::string name)
                : Analog(parser, task_handle, name),
                  bridgeConfig(parser),
                  tableConfig(parser) {
                    std::string u = parser.optional<std::string>("units", "Volts");
                    this->units = ni::UNITS_MAP.at(u);
                  }
        int32 createNIChannel() override {
            if(this->scale_config->type == "none"){
                return ni::NiDAQmxInterface::CreateAITorqueBridgeTableChan(
                        this->task_handle,
                        this->name.c_str(),
                        "",
                        this->min_val,
                        this->max_val,
                        this->units,
                        this->bridgeConfig.niBridgeConfig,
                        this->bridgeConfig.voltageExcitSource,
                        this->bridgeConfig.voltageExcitVal,
                        this->bridgeConfig.nominalBridgeResistance,
                        this->tableConfig.electricalVals,
                        this->tableConfig.numElectricalVals,
                        this->tableConfig.electricalUnits,
                        this->tableConfig.physicalVals,
                        this->tableConfig.numPhysicalVals,
                        this->tableConfig.physicalUnits,
                        NULL
                );
            }
        }
};


class ForceIEPE : public Analog{
public:
    int32_t sensitivityUnits;
    double sensitivity;
    ExcitationConfig excitationConfig;
    int32 terminal_config = 0;

    explicit ForceIEPE(config::Parser &parser, TaskHandle task_handle, std::string name)
            :   Analog(parser, task_handle, name),
                sensitivity(parser.required<double>("sensitivity")),
                excitationConfig(parser),
                terminal_config(ni::getTerminalConfig(parser.required<std::string>("terminal_config"))) {
                    std::string u = parser.optional<std::string>("units", "Volts");
                    this->units = ni::UNITS_MAP.at(u);

                    auto su = parser.optional<std::string>("sensitivity_units", "mVoltsPerG");
                    this->sensitivityUnits = ni::UNITS_MAP.at(su);

                }

    int32 createNIChannel() override {
        if(this->scale_config->type == "none"){
            return ni::NiDAQmxInterface::CreateAIForceIEPEChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->terminal_config,
                    this->min_val,
                    this->max_val,
                    this->units,
                    this->sensitivity,
                    this->sensitivityUnits,
                    this->excitationConfig.voltageExcitSource,
                    this->excitationConfig.voltageExcitVal,
                    NULL
            );
        }
    }
};


///////////////////////////////////////////////////////////////////////////////////
//                                      Charge                                   //
///////////////////////////////////////////////////////////////////////////////////
class Charge : public Analog {
public:
    int32 terminal_config = 0;
    explicit Charge(config::Parser &parser, TaskHandle task_handle, std::string name)
            : Analog(parser, task_handle, name),
              terminal_config(ni::getTerminalConfig(parser.required<std::string>("terminal_config"))){
                std::string u = parser.optional<std::string>("units", "Coulombs");
                this->units = ni::UNITS_MAP.at(u);
            }

    int32 createNIChannel() override {
        if(this->scale_config->type == "none"){
            return ni::NiDAQmxInterface::CreateAIChargeChan(
                    this->task_handle,
                    this->name.c_str(),
                    "",
                    this->terminal_config,
                    this->min_val,
                    this->max_val,
                    this->units,
                    NULL
            );
        }
    }

};
} // namespace ni
