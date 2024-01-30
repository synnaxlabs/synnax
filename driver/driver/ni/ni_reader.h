// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "daqmx.h"
#include "synnax/synnax.h"
#include <string>
#include <vector>
#include "driver/pipeline/acq.h"
#include "driver/modules/module.h"

#pragma once


namespace ni {
    typedef enum {
        ANALOG_VOLTAGE_IN,
        THERMOCOUPLE_IN,
        ANALOG_CURRENT_IN,
        DIGITAL_IN,
        DIGITAL_OUT
    } ChannelType;

    typedef struct channel_config {
        std::string name;
        uint32_t channel_key;
//    uint64_t port;
//    uint64_t line;
        ChannelType channelType;
        float min_val;
        float max_val;

        //TODO: implement a calibration class later and put that in here too
    } channel_config;
    class niDaqReader : daq::AcqReader {
    public:
        niDaqReader() {
            DAQmxCreateTask("", &taskHandle);
        }
        void init(std::vector <channel_config> channels, uint64_t acquisition_rate, uint64_t stream_rate);
    private:
        std::vector <channel_config> channels;
        std::uint64_t acq_rate = 0;
        std::uint64_t stream_rate = 0;
        std::int64_t numChannels = 0;
        TaskHandle taskHandle = 0;




    public:
        std::pair <synnax::Frame, freighter::Error> read();

        freighter::Error configure(synnax::Module config);

        freighter::Error stop();

        freighter::Error start();
//        freighter::Error parseJSONConfig(json config);
    };
}

//class Factory {
//};