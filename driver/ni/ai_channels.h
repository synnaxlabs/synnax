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
        float64* forwardCoeffs[];
        uint32_t numForwardCoeffs;
        float64* reverseCoeffs[];
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

                    //get forward coeffs (prescale -> scale)
                    if(j.contains("forward_coeffs")){
                        forwardCoeffs = new double[numForwardCoeffs];
                        for(uint32_t i = 0; i < numForwardCoeffs; i++){
                            forwardCoeffs[i] = j["forward_coeffs"][i];
                        }
                    }

                    reverse_coeffs = new double[numReverseCoeffs];
                    ni::NiDAQmxInterface::CalculateReversePolyCoeff(
                            forward_coeffs,
                            num_coeffs,
                            min_x,
                            max_x,
                            num_coeffs,
                            -1,
                            reverse_coeffs
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
            : num_electrical_vals(parser.required<uint32_t>("num_electrical_vals")),
              num_physical_vals(parser.required<uint32_t>("num_physical_vals")),
              electrical_units(parser.required<int32_t>("electrical_units")),
              physical_units(parser.required<int32_t>("physical_units")) {
                  if(!parser.ok()) return; // TODO: handle error

                  json j = parser.get_json();

                  //get electrical vals
                  if(j.contains("electrical_vals")){
                      electrical_vals = new double[num_electrical_vals];
                      for(uint32_t i = 0; i < num_electrical_vals; i++){
                          electrical_vals[i] = j["electrical_vals"][i];
                      }
                  }

                  //get physical vals
                  if(j.contains("physical_vals")){
                      physical_vals = new double[num_physical_vals];
                      for(uint32_t i = 0; i < num_physical_vals; i++){
                          physical_vals[i] = j["physical_vals"][i];
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

    typedef BridgeConfig{
        int32_t bridgeConfig;
        int32_t voltageExcitSource;
        double voltageExcitVal;
        double nominalBridgeResistance;

        BridgeConfig() = default;

        BridgeConfig(config::Parser &parser)
            :   bridgeConfig(parser.required<int32_t>("bridge_config")),
                voltageExcitSource(parser.required<int32_t>("voltage_excit_source")),
                voltageExcitVal(parser.required<double>("voltage_excit_val")),
                nominalBridgeResistance(parser.required<double>("nominal_bridge_resistance")) {
                }
        
    } BridgeConfig;

    /// @brief an object that represents and is responsible for the configuration of 
    /// a single analog channel on National Instruments hardware.
    class Analog {
    public:
        Analog() = default;

        ~Analog() = default;

        virtual int32 createNIChannel() {
            LOG(INFO) << "Creating Analog Channel";
            return 0;
        }

        static int32_t getTerminalConfig(std::string terminal_config) {
            if (terminal_config == "PseudoDiff") return DAQmx_Val_PseudoDiff;
            if (terminal_config == "Diff") return DAQmx_Val_Diff;
            if (terminal_config == "NRSE") return DAQmx_Val_NRSE;
            if (terminal_config == "RSE") return DAQmx_Val_RSE;
            return DAQmx_Val_Cfg_Default;
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
                  terminal_config(getTerminalConfig(parser.required<std::string>("terminal_config"))),
                  units(DAQmx_Val_Volts),
                  sy_key(parser.required<uint32_t>("channel")),
                  name(name),
                  type(parser.required<std::string>("type")),
                  scale_config(getScaleConfig(parser)) {
            assert(parser.ok());
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
        std::string scale_name = NULL;
        double min_val = 0;
        double max_val = 0;
        int32_t terminal_config = 0;
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

        explicit Voltage(config::Parser &parser, TaskHandle task_handle, std::string name)
                : Analog(parser, task_handle, name) {}

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
    
    /// @brief RMS voltage Channel
    class VoltageRMS : public Analog {
        public:
            explicit VoltageRMS(config::Parser &parser, TaskHandle task_handle, std::string name)
                    : Analog(parser, task_handle, name) {}

            ~VoltageRMS() = default;

            int32 createNIChannel() override {
                LOG(INFO) << "Creating Voltage RMS Channel";
                return ni::NiDAQmxInterface::CreateAIVoltageRMSChan(
                        this->task_handle, 
                        this->name.c_str(),
                        "",
                        this->min_val,
                        this->max_val,
                        DAQmx_Val_Volts,
                        NULL
                );
            }
            
    };

    /// @brief voltage Channel with excitation reference
    class VoltageWithExcit : public Analog {
        public:
            int32_t bridgeConfig = 0;
            int32_t excitationSource = 0;
            double excitationVal = 0;
            bool32 useExcitForScaling = 0;
    };


    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Temperature                              //
    ///////////////////////////////////////////////////////////////////////////////////
    class TemperatureBuiltInSensor : public Analog{
        public:
    }
    class Thermocouple : public Analog{
        public:
            int32_t thermocoupleType;
            int32_t cjcSource;
            double cjcVal;
            char cjcChannel[];
    }
    class Thermistor : public Analog{
        public:
            int32_t resistanceConfig;
            ExcitationConfig excitationConfig;
            double a;
            double b;
            double c;
    }
    class ThermistorVex : public Analog{
        public:
            int32_t resistanceConfig;
            ExcitationConfig excitationConfig;
            double a;
            double b;
            double c;
    }


    ///////////////////////////////////////////////////////////////////////////////////
    //                                    Acceleration                               //
    ///////////////////////////////////////////////////////////////////////////////////
    /// @brief acceleration channel
    class Acceleration : public Analog {
        public:
            double sensitivity;
            int32_t sensitivityUnits;
            ExcitationConfig excitationConfig;

    };
    /// @brief acceleration channel with 4 wire DC voltage
    class Acceleration4WireDCVoltage : public Analog {
        public:
            double sensitivity;
            int32_t sensitivityUnits;
    };
    /// @brief acceleration channel with charge
    class AccelerationCharge : public Analog {
        public:
            double sensitivity;
            int32_t sensitivityUnits;

    };

    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Bridge                                   //
    ///////////////////////////////////////////////////////////////////////////////////
    class Bridge : public Analog {
        public:
            BridgeConfig bridgeConfig;
    };
    class ForceBridgePolynomial : public Analog {
        public:
            double minValForScaling;
            double maxValForScaling;
            double units;
            double minValForExcitation;
            double maxValForExcitation;
            double excitationVal;
            int32_t excitationSource;
            bool32 useExcitForScaling;
    };
    class ForceBridgeTable : public Analog {
        public:
            double minValForScaling;
            double maxValForScaling;
            double units;
            double minValForExcitation;
            double maxValForExcitation;
            double excitationVal;
            int32_t excitationSource;
            bool32 useExcitForScaling;
    };
    class ForceBridgeTwoPointLin : public Analog {
        public:
            BridgeConfig bridgeConfig;
            TwoPointLinConfig twoPointLinConfig;
    };

    class ForceBridgePolynomial: public Analog{
        public:
            BridgeConfig bridgeConfig;
            PolynomialConfig polynomialConfig;
    };

    class PressureBridgeTable: public Analog{
        public: 
            BridgeConfig bridgeConfig;
            TableConfig tableConfig;
    }



    class PressureBridgeTwoPointLin: public Analog{
        public:
            BridgeConfig bridgeConfig;
            TwoPointLinConfig twoPointLinConfig;
    } 

    class TorqueBridge Polynomial: public Analog{
        public:
            BridgeConfig bridgeConfig;
            PolynomialConfig polynomialConfig;
    }

    class TorqueBridgeTable: public Analog{
        public:
            BridgeConfig bridgeConfig;
            TableConfig tableConfig;
    }

    class TorqueBridgeTwoPointLin: public Analog{
        public:
            BridgeConfig bridgeConfig;
            TwoPointLinConfig twoPointLinConfig;
    }
    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Charge                                   //
    ///////////////////////////////////////////////////////////////////////////////////
    class Charge : public Analog {

    }

    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Current                                  //
    ///////////////////////////////////////////////////////////////////////////////////
    class Current : public Analog{
        public:
            int32_t shuntResistorLoc;
            double extShuntResistorval;

    }

    class CurrentRMS : public Current{

    }

    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Force                                    //
    ///////////////////////////////////////////////////////////////////////////////////
    class ForceBridgePolynomial : public Analog{
        public:
            BridgeConfig bridgeConfig;
            PolynomialConfig polynomialConfig;


    }
    class ForceBridgeTable : public Analog{
        public:
            BridgeConfig bridgeConfig;
            TableConfig tableConfig;
    }

    class ForceBridgeTwoPointLin : public Analog{
        public:
            BridgeConfig bridgeConfig;
            TwoPointLinConfig twoPointLinConfig;
    }
    class ForceIEPE : public Analog{
        public:
            int32_t sensitivityUnits;
            double sensitivity;
            ExcitationConfig excitationConfig;
    }

    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Frequency                                //
    ///////////////////////////////////////////////////////////////////////////////////
    class FrequencyVoltage : public Analog{
        public:
            double thresholdLevel;
            double hysteresis;
    }
    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Microphone                               //
    ///////////////////////////////////////////////////////////////////////////////////
    class Microphone : public Analog{
        public:
            double micSensitivity;
            double maxSndPressLevel;
            excitConfig excitationConfig;
    }

    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Pressure                                 //
    ///////////////////////////////////////////////////////////////////////////////////
    class PressureBridgePolynomial : public Analog{
        public:
            BridgeConfig bridgeConfig;
            PolynomialConfig polynomialConfig;
    }
    class PressureBridgeTable : public Analog{
        public:
            BridgeConfig bridgeConfig;
            TableConfig tableConfig;
    }
    class PressureBridgeTwoPointLin : public Analog{
        public:
            BridgeConfig bridgeConfig;
            TwoPointLinConfig twoPointLinConfig;
    }

    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Resistance                               //
    ///////////////////////////////////////////////////////////////////////////////////
    class Resistance : public Analog{
        public:
            int32_t resistanceConfig;
            ExcitationConfig excitationConfig;

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
    }

    ///////////////////////////////////////////////////////////////////////////////////
    //                                      RTD                                      //
    ///////////////////////////////////////////////////////////////////////////////////
    class RTD : public Analog{
        public:
    }

    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Strain Gage                              //
    ///////////////////////////////////////////////////////////////////////////////////
    class StrainGage : public Analog{
        public:
            int32_t rtdType;
            int32_t resitanceConfig;
            ExcitationConfig excitationConfig;
            double r0;
    }

   

    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Torque                                   //
    ///////////////////////////////////////////////////////////////////////////////////
    class TorqueBridgePolynomial : public Analog{
        public:
            BridgeConfig bridgeConfig;
            PolynomialConfig polynomialConfig;
    }


    class TorqueBridgeTable : public Analog{
        public:
            BridgeConfig bridgeConfig;
            TableConfig tableConfig;
    }
    class TorqueBridgeTwoPointLin : public Analog{
        public:

    }

    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Velocity                                 //
    ///////////////////////////////////////////////////////////////////////////////////
    class VelocityIEPE : public Analog{
        public:
            int32_t sensitivityUnits;
            double sensitivity;
            ExcitationConfig excitationConfig;
    }

} // namespace ni