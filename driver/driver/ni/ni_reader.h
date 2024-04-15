// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once
#include "daqmx.h"
#include "client/cpp/synnax/synnax.h"
#include <string>
#include <vector>
#include "driver/driver/pipeline/acqReader.h"
//#include "driver/driver/modules/module.h"
#include <queue>
#include "driver/driver/errors/errors.h"

namespace ni {
    typedef enum {
        ANALOG_VOLTAGE_IN,
        THERMOCOUPLE_IN,
        ANALOG_CURRENT_IN,
        DIGITAL_IN,
        DIGITAL_OUT,
        INDEX_CHANNEL,
        INVALID_CHANNEL
    } ChannelType;

    typedef enum {
        ANALOG_READER,
        DIGITAL_READER,
        DIGITAL_WRITER,
    } TaskType;

    typedef struct channel_config { //TODO: replace with json parsing
        std::string name;
        uint32_t channel_key;
        ChannelType channelType;
        float min_val;
        float max_val;

        //TODO: implement a calibration class later and put that in here too
    } channel_config;



    class niDaqReader : public daq::AcqReader { // public keyword required to store pointer to niDaqreader in a pointer to acqReader
    public:
        niDaqReader(TaskHandle taskHandle);
        std::pair<json,int> init(std::vector <channel_config> channels, uint64_t acquisition_rate, uint64_t stream_rate);
        std::pair<json, int> init(json channels, uint64_t acquisition_rate, uint64_t stream_rate);
//        int ni::niDaqReader::init(std::string config, uint64_t acquisition_rate, uint64_t stream_rate);
        std::pair <synnax::Frame, freighter::Error> read();
        freighter::Error stop();
        freighter::Error start();
//        freighter::Error parseJSONConfig(json config);
        json getErrorInfo();
    private:
        std::vector <channel_config> channels;
        std::uint64_t acq_rate = 0;
        std::uint64_t stream_rate = 0;
        std::int64_t numChannels = 0;
        TaskHandle taskHandle = 0;
        TaskType taskType;
        std::pair <synnax::Frame, freighter::Error> readAnalog();
        std::pair <synnax::Frame, freighter::Error> readDigital();
        double* data; /// @brief pointer to heap allocated dataBuffer to provide to DAQmx read functions
        uInt8* digitalData; /// @brief pointer to heap allocated dataBuffer to provide to DAQmx read functions
        int bufferSize = 0; // size of the data buffer
        int numSamplesPerChannel =0 ;
        json errInfo;
    };


    class   niDaqWriter : public daq::daqWriter {
    public:
        niDaqWriter(TaskHandle taskHandle);
        std::pair<json,int> init(std::vector <channel_config> channels);
        std::pair<json,int> init(json channels, synnax::ChannelKey ack_index_key);
        std::pair <synnax::Frame, freighter::Error> write(synnax::Frame frame);
        std::pair <synnax::Frame, freighter::Error> writeDigital(synnax::Frame frame);
        freighter::Error stop();
        freighter::Error start();
        freighter::Error formatData(synnax::Frame frame);
        json getErrorInfo();
    private:
        std::vector <channel_config> channels;
        std::int64_t numChannels = 0;
        TaskHandle taskHandle = 0;
        TaskType taskType;
//        freighter::Error writeAnalog(synnax::Frame frame);  // Implement later
        uint8_t * writeBuffer; /// @brief pointer to heap allocated dataBuffer to provide to DAQmx read functions
        int bufferSize = 0; // size of the data buffer
        int numSamplesPerChannel = 0 ;
        std::vector<synnax::ChannelKey> ack_channel_keys;
        std::vector<synnax::ChannelKey> cmd_channel_keys;
        std::queue<synnax::ChannelKey> ack_queue; // queue of ack channels to write to
        synnax::ChannelKey ack_index_key;
        json errInfo;
    };
    // Global Function for DAQmx Error Handling
    int checkNIError(int32 error, json &errInfo);
}
