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


#include "nidaqmx_api.h"
#include "daqmx.h"
#include "nisyscfg.h"
#include "driver/config/config.h"
#include "nlohmann/json.hpp"
#include "glog/logging.h"
#include "client/cpp/telem/telem.h"

namespace ni {

    typedef struct ExcitationConfig{
        int32_t voltageExcitSource;
        double voltageExcitVal;    
        double minValForExcitation; // optional
        double maxValForExcitation; //optional
        bool32 useExcitForScaling;  //optional

        ExcitationConfig() = default;

        ExcitationConfig(config::Parser &parser)
            :   voltageExcitSource(parser.required<int32_t>("voltage_excit_source")),
                voltageExcitVal(parser.required<double>("voltage_excit_val")),
                minValForExcitation(parser.optional<double>("min_val_for_excitation", 0)),
                maxValForExcitation(parser.optional<double>("max_val_for_excitation", 0)),
                useExcitForScaling(parser.optional<bool32>("use_excit_for_scaling", 0)) {
                    
                }
    } ExcitationConfig; 

    typedef struct PolynomialConfig{
        float64* forwardCoeffs;
        uint32_t numForwardCoeffs;
        float64* reverseCoeffs;
        uint32_t numReverseCoeffs;
        int32_t electricalUnits;
        int32_t physicalUnits;

        PolynomialConfig() = default;

        PolynomialConfig(config::Parser &parser)
            :   numForwardCoeffs(parser.required<uint32_t>("num_forward_coeffs")),
                numReverseCoeffs(parser.required<uint32_t>("num_reverse_coeffs")),
                electricalUnits(parser.required<int32_t>("electrical_units")),
                physicalUnits(parser.required<int32_t>("physical_units")) {
                    if(!parser.ok()) return; // TODO: handle error

                    json j = parser.get_json();

                    forwardCoeffs = new double[numForwardCoeffs];
                    reverseCoeffs = new double[numReverseCoeffs];

                    //get forward coeffs (prescale -> scale)
                    if(j.contains("forward_coeffs")){
                        forwardCoeffs = new double[numForwardCoeffs];
                        for(uint32_t i = 0; i < numForwardCoeffs; i++){
                            forwardCoeffs[i] = j["forward_coeffs"][i];
                        }
                    }

                    ni::NiDAQmxInterface::CalculateReversePolyCoeff(
                            forwardCoeffs,
                            numForwardCoeffs,
                            -100, //FIXME dont hard code
                            100, //FIXME dont hard code
                            numReverseCoeffs,
                            -1,
                            reverseCoeffs
                    ); // FIXME: reversePoly order should be user inputted?
                }
        ~PolynomialConfig() {
            if(forwardCoeffs != nullptr) delete[] forwardCoeffs;
            if(reverseCoeffs != nullptr) delete[] reverseCoeffs;
        }

    } PolynomialConfig;

    typedef struct TableConfig{
        float64* electricalVals;
        uint32_t numElectricalVals;
        float64* physicalVals;
        uint32_t numPhysicalVals;
        int32_t electricalUnits;
        int32_t physicalUnits;
        
        TableConfig() = default;

        TableConfig(config::Parser &parser) 
            : numElectricalVals(parser.required<uint32_t>("num_electrical_vals")),
              numPhysicalVals(parser.required<uint32_t>("num_physical_vals")),
              electricalUnits(parser.required<int32_t>("electrical_units")),
              physicalUnits(parser.required<int32_t>("physical_units")) {
                  if(!parser.ok()) return; // TODO: handle error

                  json j = parser.get_json();

                  //get electrical vals
                  if(j.contains("electrical_vals")){
                      electricalVals = new double[numElectricalVals];
                      for(uint32_t i = 0; i < numElectricalVals; i++){
                          electricalVals[i] = j["electrical_vals"][i];
                      }
                  }

                  //get physical vals
                  if(j.contains("physical_vals")){
                      physicalVals = new double[numPhysicalVals];
                      for(uint32_t i = 0; i < numPhysicalVals; i++){
                          physicalVals[i] = j["physical_vals"][i];
                      }
                  }
              }
    } TableConfig;

    typedef struct TwoPointLinConfig{
        double firstElectricalVal;
        double secondElectricalVal;
        int32_t electricalUnits;
        double firstPhysicalVal;
        double secondPhysicalVal;
        int32_t physicalUnits;

        TwoPointLinConfig() = default;

        TwoPointLinConfig(config::Parser &parser)
            :   firstElectricalVal(parser.required<double>("first_electrical_val")),
                secondElectricalVal(parser.required<double>("second_electrical_val")),
                electricalUnits(parser.required<int32_t>("electrical_units")),
                firstPhysicalVal(parser.required<double>("first_physical_val")),
                secondPhysicalVal(parser.required<double>("second_physical_val")),
                physicalUnits(parser.required<int32_t>("physical_units")) {
        }

    } TwoPointLinConfig;

    typedef struct BridgeConfig{
        int32_t niBridgeConfig;
        int32_t voltageExcitSource;
        double voltageExcitVal;
        double nominalBridgeResistance;

        BridgeConfig() = default;

        BridgeConfig(config::Parser &parser)
            :   niBridgeConfig(parser.required<int32_t>("bridge_config")),
                voltageExcitSource(parser.required<int32_t>("voltage_excit_source")),
                voltageExcitVal(parser.required<double>("voltage_excit_val")),
                nominalBridgeResistance(parser.required<double>("nominal_bridge_resistance")) {
                }
        
    } BridgeConfig;

    static inline int32_t getTerminalConfig(std::string terminal_config) {
        if (terminal_config == "PseudoDiff") return DAQmx_Val_PseudoDiff;
        if (terminal_config == "Diff") return DAQmx_Val_Diff;
        if (terminal_config == "NRSE") return DAQmx_Val_NRSE;
        if (terminal_config == "RSE") return DAQmx_Val_RSE;
        return DAQmx_Val_Cfg_Default;
    }

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

       

        static ScaleConfig getScaleConfig(config::Parser &parser) {
            // TODO check if custom scale and channel exist
            std::string scale_name = std::to_string(parser.required<uint32_t>("channel")) + "_scale";
            auto scale_parser = parser.child("custom_scale");
            return ScaleConfig(scale_parser, scale_name);
        }

        int32 createNIScale() {
            if (this->scale_config.type == "none") return 0;
            return this->scale_config.createNIScale();
        }

        explicit Analog(config::Parser &parser, TaskHandle task_handle, std::string name)
                : task_handle(task_handle),
                  min_val(parser.required<float_t>("min_val")),
                  max_val(parser.required<float_t>("max_val")),
                  units(DAQmx_Val_Volts),
                  sy_key(parser.required<uint32_t>("channel")),
                  name(name),
                  type(parser.required<std::string>("type")),
                  scale_config(getScaleConfig(parser)) {
            LOG(INFO) << "Analog Channel constructor ";
            // check name of channel
            if (this->scale_config.type != "none") {
                LOG(INFO) << "Scale type: " << this->scale_config.type;
                this->scale_name = this->scale_config.name;
                this->units = DAQmx_Val_FromCustomScale;
            }
            LOG(INFO) << "Analog Channel constructor end";
        }

        TaskHandle task_handle = 0;
        std::string scale_name = "";
        double min_val = 0;
        double max_val = 0;
        int32_t units = DAQmx_Val_Volts;
        uint32_t sy_key = 0;
        std::string name = "";
        std::string type = "";

        ScaleConfig scale_config;
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
                  terminal_config(ni::getTerminalConfig(parser.required<std::string>("terminal_config"))){

                  }

        ~Voltage() = default;

        int32 createNIChannel() override {
            LOG(INFO) << "Creating Voltage Channel";

            if (this->scale_config.type == "none") {
                return ni::NiDAQmxInterface::CreateAIVoltageChan(
                        this->task_handle, 
                        this->name.c_str(),      
                        "",                         // name to assign channel
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
                        "",                         // name to assign channel
                        this->terminal_config,
                        this->min_val,
                        this->max_val,
                        DAQmx_Val_FromCustomScale,
                        this->scale_config.name.c_str()
                );

            }
        }
    };

/*
    /// @brief RMS voltage Channel
    class VoltageRMS : public Voltage {
        public: 
            explicit Voltage(config::Parser &parser, TaskHandle task_handle, std::string name)
                : Voltage(parser, task_handle, name){}

            ~VoltageRMS() = default;

            int32 createNIChannel() override {
                LOG(INFO) << "Creating Voltage RMS Channel";
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
            int32_t excitationSource = 0;
            double excitationVal = 0;
            bool32 useExcitForScaling = 0;

            explicit VoltageWithExcit(config::Parser &parser, TaskHandle task_handle, std::string name)
                : Voltage(parser, task_handle, name),
                  bridgeConfig(parser.required<int32_t>("bridge_config")),
                  excitationSource(parser.required<int32_t>("excitation_source")),
                  excitationVal(parser.required<double>("excitation_val")),
                  useExcitForScaling(parser.required<bool32>("use_excit_for_scaling")) {}

            ~VoltageWithExcit() = default;

            int32 createNIChannel() override {
                LOG(INFO) << "Creating Voltage Channel with Excitation Reference";
                if(this->scale_config.type == "none"){
                    return ni::NiDAQmxInterface::CreateAIVoltageChanWithExcit(
                            this->task_handle,
                            this->name.c_str(),
                            "",
                            this->terminal_config,
                            this->min_val,
                            this->max_val,
                            DAQmx_Val_Volts,
                            this->bridgeConfig,
                            this->excitationSource,
                            this->excitationVal,
                            this->useExcitForScaling,
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
            int32_t resitanceConfig;
            ExcitationConfig excitationConfig;
            double r0;

            explicit RTD(config::Parser &parser, TaskHandle task_handle, std::string name)
                    : Analog(parser, task_handle, name),
                      rtdType(parser.required<int32_t>("rtd_type")),
                      resistanceConfig(parser.required<int32_t>("resistance_config")),
                      excitationConfig(parser),
                      r0(parser.required<double>("r0")) {}
    }
            int32 createNIChannel() override {
                if(this->scale_config.type == "none"){
                    return ni::NiDAQmxInterface::CreateAIRTDChan(
                            this->task_handle,
                            this->name.c_str(),
                            "",
                            this->min_val,
                            this->max_val,
                            this->units,
                            this->rtdType,
                            this->resistanceConfig,
                            this->excitationConfig.voltageExcitSource,
                            this->excitationConfig.voltageExcitVal,
                            this->r0,
                            NULL
                    );
                }
            }

   

*/
    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Temperature                              //
    ///////////////////////////////////////////////////////////////////////////////////
    class Thermocouple : public Analog{
        public:
            int32_t thermocoupleType;
            int32_t cjcSource;
            double cjcVal;
            std::string cjcChannel;
            
            explicit Thermocouple(config::Parser &parser, TaskHandle task_handle, std::string name)
                    : Analog(parser, task_handle, name),
                    //   thermocoupleType(parser.required<std::string>("thermocouple_type")),
                    //   cjcSource(parser.required<std::string>("cjc_source")),
                      cjcVal(parser.required<double>("cjc_val")){
                        std::string u = parser.optional<std::string>("units", "DegC");
                        this->units = ni::UNITS_MAP.at(u); // TODO: make this optional and default to C?
                      }
                      //cjcChannel(parser.required<std::string>("cjc_channel")) {} FIXME: this property should be take form console


            ///	DAQmxErrChk (DAQmxCreateAIThrmcplChan(taskHandle,"","",0.0,100.0,DAQmx_Val_DegC,DAQmx_Val_J_Type_TC,DAQmx_Val_BuiltIn,25.0,""));

            int32 createNIChannel() override {
                LOG(INFO) << "Creating Thermocouple Channel";

                if(this->scale_config.type == "none"){
                    return ni::NiDAQmxInterface::CreateAIThrmcplChan(
                        this->task_handle,
                        this->name.c_str(),
                        "",
                        this->min_val,
                        this->max_val,
                        this->units,
                        DAQmx_Val_J_Type_TC,//this->thermocoupleType,
                        DAQmx_Val_BuiltIn,// this->cjcSource,
                        this->cjcVal,
                        ""
                    );
                }
            }


    };
    
    /*
    class TemperatureBuiltInSensor : public Analog{
        public:
            explicit TemperatureBuiltInSensor(config::Parser &parser, TaskHandle task_handle, std::string name){
                this->task_handle = task_handle;
                this->physical_channel = parser.required<std::string>("physical_channel");
                this->name = name;
                this->units = parser.required<int32_t>("units");
            }

            int32 createNIChannel() override {
                LOG(INFO) << "Creating Temperature Built In Sensor Channel";
                return ni::NiDAQmxInterface::CreateAITempBuiltInSensorChan(
                        this->task_handle,
                        this->name.c_str(),
                        "",
                        this->units,
                );
            }
    };

    class Thermistor : public Analog{
        public:
            int32_t resistanceConfig;
            ExcitationConfig excitationConfig;
            double a;
            double b;
            double c;

            explicit Thermistor(config::Parser &parser, TaskHandle task_handle, std::string name)
                    : Analog(parser, task, name),
                      resistanceConfig(parser.required<int32_t>("resistanceConfig")),
                      excitationConfig(parser),
                      a(parser.required<double>("a")),
                      b(parser.required<double>("b")),
                      c(parser.required<double>("c")) {}

            int32 createNIChannel() override {
                if(this->scale_config.type == "none"){
                    return ni::NiDAQmxInterface::CreateAIThrmsstrChanIex(
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
                      resistanceConfig(parser.required<int32_t>("resistanceConfig")),
                      excitationConfig(parser),
                      a(parser.required<double>("a")),
                      b(parser.required<double>("b")),
                      c(parser.required<double>("c")),
                      r1(parser.required<double>("r1")) {}

            int32 createNIChannel() override {
                if(this->scale_config.type == "none"){
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

            explicit Acceleration(config::Parser &parser, TaskHandle task_handle, std::string name)
                    : Analog(parser, task_handle, name),
                      sensitivity(parser.required<double>("sensitivity")),
                      sensitivityUnits(parser.required<int32_t>("sensitivity_units")),
                      excitationConfig(parser) {}
            
            int32 createNIChannel() override {
                if(this->scale_config.type == "none"){
                    return ni::NiDAQmxInterface::CreateAIAccelChan(
                            this->task_handle,
                            this->name.c_str(),
                            "",
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
    class Acceleration4WireDCVoltage : public Analog {
        public:
            double sensitivity;
            int32_t sensitivityUnits;

        explicit Acceleration4WireDCVoltage(config::Parser &parser, TaskHandle task_handle, std::string name)
                : Analog(parser, task_handle, name),
                  sensitivity(parser.required<double>("sensitivity")),
                  sensitivityUnits(parser.required<int32_t>("sensitivity_units")) {}
    };
    /// @brief acceleration channel with charge
    class AccelerationCharge : public Analog {
        public:
            double sensitivity;
            int32_t sensitivityUnits;

            explicit AccelerationCharge(config::Parser &parser, TaskHandle task_handle, std::string name)
                    : Analog(parser, task_handle, name),
                      sensitivity(parser.required<double>("sensitivity")),
                      sensitivityUnits(parser.required<int32_t>("sensitivity_units")) {}
            
            int32 createNIChannel() override {
                if(this->scale_config.type == "none"){
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
                : Analog(parser, task_handle, name){}
            
            int32 createNIChannel() override{
                if(this->scale_config.type == "none"){
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
    //                                      Charge                                   //
    ///////////////////////////////////////////////////////////////////////////////////
    class Charge : public Analog {
        explicit Charge(config::Parser &parser, TaskHandle task_handle, std::string name)
                : Analog(parser, task_handle, name) {}

        int32 createNIChannel() override {
            if(this->scale_config.type == "none"){
                return ni::NiDAQmxInterface::CreateAIChargeChan(
                        this->task_handle,
                        this->name.c_str(),
                        "",
                        this->terminal_config,
                        this->min_val,
                        this->max_val,
                        this->units,
                        this->excitationConfig.voltageExcitSource,
                        this->excitationConfig.voltageExcitVal,
                        NULL
                );
            }
        }

    }

    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Current                                  //
    ///////////////////////////////////////////////////////////////////////////////////
    class Current : public Analog{
        public:
            int32_t shuntResistorLoc;
            double extShuntResistorval;

            explicit Current(config::Parser &parser, TaskHandle task_handle, std::string name)
                    : Analog(parser, task_handle, name),
                      shuntResistorLoc(parser.required<int32_t>("shunt_resistor_loc")),
                      extShuntResistorval(parser.required<double>("ext_shunt_resistor_val")) {}
            
            int32 createNIChannel() override {
                if(this->scale_config.type == "none"){
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
                            this->excitationConfig.voltageExcitSource,
                            this->excitationConfig.voltageExcitVal,
                            NULL
                    );
                }
            }
    }

    class CurrentRMS : public Current{
        explicit CurrentRMS(config::Parser &parser, TaskHandle task_handle, std::string name)
                : Current(parser, task_handle, name) {}
        
        int32 createNIChannel() override {
            if(this->scale_config.type == "none"){
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
                        this->excitationConfig.voltageExcitSource,
                        this->excitationConfig.voltageExcitVal,
                        NULL
                );
            }
        }
    }

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
                      polynomialConfig(parser) {}
            
            int32 createNIChannel() override {
                if(this->scale_config.type == "none"){
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

    }
    class ForceBridgeTable : public Analog{
        public:
            BridgeConfig bridgeConfig;
            TableConfig tableConfig;

            explicit ForceBridgeTable(config::Parser &parser, TaskHandle task_handle, std::string name)
                    : Analog(parser, task_handle, name),
                      bridgeConfig(parser),
                      tableConfig(parser) {}

            int32 createNIChannel() override {
                if(this->scale_config.type == "none"){
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
    }

    class ForceBridgeTwoPointLin : public Analog{
        public:
            BridgeConfig bridgeConfig;
            TwoPointLinConfig twoPointLinConfig;

            explicit ForceBridgeTwoPointLin(config::Parser &parser, TaskHandle task_handle, std::string name)
                    : Analog(parser, task_handle, name),
                      bridgeConfig(parser),
                      twoPointLinConfig(parser) {}

            int32 createNIChannel() override {
                if(this->scale_config.type == "none"){
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
    }
    class ForceIEPE : public Analog{
        public:
            int32_t sensitivityUnits;
            double sensitivity;
            ExcitationConfig excitationConfig;

            explicit ForceIEPE(config::Parser &parser, TaskHandle task_handle, std::string name)
                    : Analog(parser, task_handle, name),
                      sensitivityUnits(parser.required<int32_t>("sensitivity_units")),
                      sensitivity(parser.required<double>("sensitivity")),
                      excitationConfig(parser) {}

            int32 createNIChannel() override {
                if(this->scale_config.type == "none"){
                    return ni::NiDAQmxInterface::CreateAIForceIEPEChan(
                            this->task_handle,
                            this->name.c_str(),
                            "",
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
    }

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
                      hysteresis(parser.required<double>("hysteresis")) {}
            int32 createNIChannel() override {
                if(this->scale_config.type == "none"){
                    return ni::NiDAQmxInterface::CreateAIFreqVoltageChan(
                            this->task_handle,
                            this->name.c_str(),
                            "",
                            this->terminal_config,
                            this->min_val,
                            this->max_val,
                            this->units,
                            this->thresholdLevel,
                            this->hysteresis,
                            NULL
                    );
                }
            }
    }
    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Microphone                               //
    ///////////////////////////////////////////////////////////////////////////////////
    class Microphone : public Analog{
        public:
            double micSensitivity;
            double maxSndPressLevel;
            excitConfig excitationConfig;

            explicit Microphone(config::Parser &parser, TaskHandle task_handle, std::string name)
                    : Analog(parser, task_handle, name),
                      micSensitivity(parser.required<double>("mic_sensitivity")),
                      maxSndPressLevel(parser.required<double>("max_snd_press_level")),
                      excitationConfig(parser) {} 

            int32 createNIChannel() override {
                if(this->scale_config.type == "none"){
                    return ni::NiDAQmxInterface::CreateAIMicrophoneChan(
                            this->task_handle,
                            this->name.c_str(),
                            "",
                            this->terminal_config,
                            this->min_val,
                            this->max_val,
                            this->units,
                            this->micSensitivity,
                            this->maxSndPressLevel,
                            this->excitationConfig.voltageExcitSource,
                            this->excitationConfig.voltageExcitVal,
                            NULL
                    );
                }
            }
    }

    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Pressure                                 //
    ///////////////////////////////////////////////////////////////////////////////////
    class PressureBridgePolynomial : public Analog{
        public:
            BridgeConfig bridgeConfig;
            PolynomialConfig polynomialConfig;

            explicit PressureBridgePolynomial(config::Parser &parser, TaskHandle task_handle, std::string name)
                    : Analog(parser, task_handle, name),
                      bridgeConfig(parser),
                      polynomialConfig(parser) {}

            int32 createNIChannel() override {
                if(this->scale_config.type == "none"){
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
    }
    class PressureBridgeTable : public Analog{
        public:
            BridgeConfig bridgeConfig;
            TableConfig tableConfig;

            explicit PressureBridgeTable(config::Parser &parser, TaskHandle task_handle, std::string name)
                    : Analog(parser, task_handle, name),
                      bridgeConfig(parser),
                      tableConfig(parser) {}
            
            int32 createNIChannel() override {
                if(this->scale_config.type == "none"){
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
    }
    class PressureBridgeTwoPointLin : public Analog{
        public:
            BridgeConfig bridgeConfig;
            TwoPointLinConfig twoPointLinConfig;

            explicit PressureBridgeTwoPointLin(config::Parser &parser, TaskHandle task_handle, std::string name)
                    : Analog(parser, task_handle, name),
                      bridgeConfig(parser),
                      twoPointLinConfig(parser) {}
            
            int32 createNIChannel() override {
                if(this->scale_config.type == "none"){
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
    }

    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Resistance                               //
    ///////////////////////////////////////////////////////////////////////////////////
    class Resistance : public Analog{
        public:
            int32_t resistanceConfig;
            ExcitationConfig excitationConfig;

            explicit Resistance(config::Parser &parser, TaskHandle task_handle, std::string name)
                    : Analog(parser, task_handle, name),
                      resistanceConfig(parser),
                      excitationConfig(parser) {}

            int32 createNIChannel() override {
                if(this->scale_config.type == "none"){
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

    } 

    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Rosette Strain Gage                      //
    ///////////////////////////////////////////////////////////////////////////////////
    class RosetteStrainGage : public Analog{
        public:
            int32_t rosetteType;
            double gageOrientation;
            int32_t rosseteMeasType;
            int32 strainConfig;
            ExcitationConfig excitationConfig;
            double gageFactor;
            double nominalGageResistance;
            double poissonRatio;
            double leadWireResistance;

            explicit RosetteStrainGage(config::Parser &parser, TaskHandle task_handle, std::string name)
                    : Analog(parser, task_handle, name),
                      rosetteType(parser.required<int32_t>("rosette_type")),
                      gageOrientation(parser.required<double>("gage_orientation")),
                      rosseteMeasType(parser.required<int32_t>("rosette_meas_type")),
                      strainConfig(parser.required<int32_t>("strain_config")),
                      excitationConfig(parser),
                      gageFactor(parser.required<double>("gage_factor")),
                      nominalGageResistance(parser.required<double>("nominal_gage_resistance")),
                      poissonRatio(parser.required<double>("poisson_ratio")),
                      leadWireResistance(parser.required<double>("lead_wire_resistance")) {}
                
            int32 createNIChannel() override {
                if(this->scale_config.type == "none"){
                    return ni::NiDAQmxInterface::CreateAIRosetteStrainGageChan(
                            this->task_handle,
                            this->name.c_str(),
                            "",
                            this->min_val,
                            this->max_val,
                            this->units,
                            this->rosetteType,
                            this->gageOrientation,
                            this->rosseteMeasType,
                            this->strainConfig,
                            this->excitationConfig.voltageExcitSource,
                            this->excitationConfig.voltageExcitVal,
                            this->gageFactor,
                            this->nominalGageResistance,
                            this->poissonRatio,
                            this->leadWireResistance,
                            NULL
                    );
                }
            }
    }

    ///////////////////////////////////////////////////////////////////////////////////
    //                              Strain Gage                                      //
    ///////////////////////////////////////////////////////////////////////////////////
    class Strain Gage : public Analog{
        public:
            int32_t strainConfig;
            ExcitationConfig excitationConfig;
            double gageFactor;
            double initialBridgeVoltage;
            double nominalGageResistance;
            double poissonRatio;
            double leadWireResistance;

            explicit StrainGage(config::Parser &parser, TaskHandle task_handle, std::string name)
                    : Analog(parser, task_handle, name),
                      strainConfig(parser.required<int32_t>("strain_config")),
                      excitationConfig(parser),
                      gageFactor(parser.required<double>("gage_factor")),
                      initialBridgeVoltage(parser.required<double>("initial_bridge_voltage")),
                      nominalGageResistance(parser.required<double>("nominal_gage_resistance")),
                      poissonRatio(parser.required<double>("poisson_ratio")),
                      leadWireResistance(parser.required<double>("lead_wire_resistance")) {}

            int32 createNIChannel() override {
                if(this->scale_config.type == "none"){
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
    }

    
    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Torque                                   //
    ///////////////////////////////////////////////////////////////////////////////////
    class TorqueBridgePolynomial : public Analog{
        public:
            BridgeConfig bridgeConfig;
            PolynomialConfig polynomialConfig;

            explicit TorqueBridgePolynomial(config::Parser &parser, TaskHandle task_handle, std::string name)
                    : Analog(parser, task_handle, name),
                      bridgeConfig(parser),
                      polynomialConfig(parser) {}

            int32 createNIChannel() override {
                if(this->scale_config.type == "none"){
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
    }


    class TorqueBridgeTable : public Analog{
        public:
            BridgeConfig bridgeConfig;
            TableConfig tableConfig;
    }
    class TorqueBridgeTwoPointLin : public Analog{
        public:
            BridgeConfig bridgeConfig;
            TwoPointLinConfig twoPointLinConfig;
            explicit TorqueBridgeTwoPointLin(config::Parser &parser, TaskHandle task_handle, std::string name)
                    : Analog(parser, task_handle, name),
                      bridgeConfig(parser),
                      twoPointLinConfig(parser) {}

            int32 createNIChannel() override {
                if(this->scale_config.type == "none"){
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
    }

    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Velocity                                 //
    ///////////////////////////////////////////////////////////////////////////////////
    class VelocityIEPE : public Analog{
        public:
            int32_t sensitivityUnits;
            double sensitivity;
            ExcitationConfig excitationConfig;

            explicit VelocityIEPE(config::Parser &parser, TaskHandle task_handle, std::string name)
                    : Analog(parser, task_handle, name),
                      sensitivityUnits(parser.required<int32_t>("sensitivity_units")),
                      sensitivity(parser.required<double>("sensitivity")),
                      excitationConfig(parser) {}

            int32 createNIChannel() override {
                if(this->scale_config.type == "none"){
                    return ni::NiDAQmxInterface::CreateAIVelocityIEPEChan(
                            this->task_handle,
                            this->name.c_str(),
                            "",
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
    }

*/
} // namespace ni

