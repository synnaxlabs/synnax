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
                // strcpy(this->scale_name, this->scale_config.name.c_str()); // FIXME
                this->units = DAQmx_Val_FromCustomScale;
            }
            LOG(INFO) << "Analog Channel constructor end";
        }

        TaskHandle task_handle = 0;
        char *scale_name = NULL;
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
    /// @brief voltage channel. Can be configured to measure RMS voltage instead or scale 
    /// with internal excitaiton
    class Voltage : public Analog {
    public:

        explicit Voltage(config::Parser &parser, TaskHandle task_handle, std::string name)
                : Analog(parser, task_handle, name) {
        }

        ~Voltage() = default;

        int32 createNIChannel() override {
            LOG(INFO) << "Creating Voltage Channel";

            if (this->scale_config.type == "none") {
                return ni::NiDAQmxInterface::CreateAIVoltageChan(
                        this->task_handle, this->name.c_str(),
                        "",
                        this->terminal_config,
                        this->min_val,
                        this->max_val,
                        DAQmx_Val_Volts,
                        NULL
                );
            } else {
                return ni::NiDAQmxInterface::CreateAIVoltageChan(
                        this->task_handle, this->name.c_str(),
                        "",
                        this->terminal_config,
                        this->min_val,
                        this->max_val,
                        DAQmx_Val_FromCustomScale,
                        this->scale_config.name.c_str()
                );

            }
        }
    };
    // DAQmxCreateAIVoltageChan
    // DAQmxCreateAIVoltageRMSChan
    // DAQmxCreateAIVoltageChanWithExcit


    ///////////////////////////////////////////////////////////////////////////////////
    //                                    Acceleration                               //
    ///////////////////////////////////////////////////////////////////////////////////
    // DAQmxCreateAIAccelChan
    // DAQmxCreateAIAccel4WireDCVoltageChan
    // DAQmxCreateAIAccelChargeChan

    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Bridge                                   //
    ///////////////////////////////////////////////////////////////////////////////////
    // DAQmxCreateAIBridgeChan
    // DAQmxCreateAIForceBridgePolynomialChan
    // DAQmxCreateAIForceBridgeTableChan
    // DAQmxCreateAIForceBridgeTwoPointLinChan
    // DAQmxCreateAIPressureBridgePolynomialChan
    // DAQmxCreateAIPressureBridgeTableChan
    // DAQmxCreateAIPressureBridgeTwoPointLinChan
    // DAQmxCreateAITorqueBridgePolynomialChan
    // DAQmxCreateAITorqueBridgeTableChan
    // DAQmxCreateAITorqueBridgeTwoPointLinChan

    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Charge                                   //
    ///////////////////////////////////////////////////////////////////////////////////
    // DAQmxCreateAIChargeChan

    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Current                                  //
    ///////////////////////////////////////////////////////////////////////////////////
    // DAQmxCreateAICurrentChan
    // DAQmxCreateAICurrentRMSChan

    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Force                                    //
    ///////////////////////////////////////////////////////////////////////////////////
    // DAQmxCreateAIForceBridgePolynomialChan
    // DAQmxCreateAIForceBridgeTableChan
    // DAQmxCreateAIForceBridgeTwoPointLinChan
    // DAQmxCreateAIForceIEPEChan

    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Frequency                                //
    ///////////////////////////////////////////////////////////////////////////////////
    // DAQmxCreateAIFreqVoltageChan

    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Microphone                               //
    ///////////////////////////////////////////////////////////////////////////////////
    // DAQmxCreateAIMicrophoneChan

    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Pressure                                 //
    ///////////////////////////////////////////////////////////////////////////////////
    // DAQmxCreateAIPressureBridgePolynomialChan
    // DAQmxCreateAIPressureBridgeTableChan
    // DAQmxCreateAIPressureBridgeTwoPointLinChan

    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Resistance                               //
    ///////////////////////////////////////////////////////////////////////////////////
    // DAQmxCreateAIResistanceChan

    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Rosette Strain Gage                      //
    ///////////////////////////////////////////////////////////////////////////////////
    // DAQmxCreateAIRosetteStrainGageChan

    ///////////////////////////////////////////////////////////////////////////////////
    //                                      RTD                                      //
    ///////////////////////////////////////////////////////////////////////////////////
    // DAQmxCreateAIRTDChan

    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Strain Gage                              //
    ///////////////////////////////////////////////////////////////////////////////////
    // DAQmxCreateAIStrainGageChan

    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Temperature                              //
    ///////////////////////////////////////////////////////////////////////////////////
    // DAQmxCreateAITempBuiltInSensorChan
    // DAQmxCreateAIThrmcplChan
    // DAQmxCreateAIThrmstrChanIex
    // DAQmxCreateAIThrmstrChanVex

    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Torque                                   //
    ///////////////////////////////////////////////////////////////////////////////////
    // DAQmxCreateAITorqueBridgePolynomialChan
    // DAQmxCreateAITorqueBridgeTableChan
    // DAQmxCreateAITorqueBridgeTwoPointLinChan

    ///////////////////////////////////////////////////////////////////////////////////
    //                                      Velocity                                 //
    ///////////////////////////////////////////////////////////////////////////////////
    // DAQmxCreateAIVelocityIEPEChan



}