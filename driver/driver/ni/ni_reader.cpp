// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//
// Created by Emiliano Bonilla on 1/3/24.
//

#include "ni_reader.h"
#include "nlohmann/json.hpp"
#include "synnax/telem/telem.h"
#include <utility>
#include <chrono>
#include <stdio.h>
#include <cassert>

using json = nlohmann::json;
using namespace ni;

///////////////////////////////////////////////////////////////////////////////////
// niDaqReader
///////////////////////////////////////////////////////////////////////////////////

niDaqReader::niDaqReader(TaskHandle taskHandle) : taskHandle(taskHandle) {}

void ni::niDaqReader::init(json config, uint64_t acquisition_rate, uint64_t stream_rate) {
    std::vector<channel_config> channel_configs;
    auto channels = config["channels"];
    auto deviceName = config["device"].get<std::string>();
    for (auto &channel : channels) {
        auto type = channel["type"].get<std::string>();
        channel_config config;
        std::string portName =  (type == "analogVoltageInput") ? "ai" : "";
        config.name = deviceName + "/" + portName + channel["port"].dump().c_str();
        printf("Channel Name: %s\n", config.name.c_str());
        config.channel_key = channel["channel"].get<uint32_t>();
        config.min_val = -10.0;//channel["min_val"].get<float>(); // TODO: come back to when added to json
        config.max_val = 10.0;//channel["max_val"].get<float>(); // TODO: come backt o when added to json
        config.channelType = (type == "analogVoltageInput") ? ANALOG_VOLTAGE_IN
                            : (type == "thermocoupleInput") ? THERMOCOUPLE_IN
                            : (type == "analogCurrentInput") ? ANALOG_CURRENT_IN
                            : (type == "digitalInput") ? DIGITAL_IN
                            : (type == "index") ? INDEX_CHANNEL
                            : INVALID_CHANNEL;
        channel_configs.push_back(config);
    }
    // todo need to add an index channel
    init(channel_configs, acquisition_rate, stream_rate);
}

// TODO: I want to make a producer consumer model here instead? This works for now but come back to

void ni::niDaqReader::init(std::vector<channel_config> channels, uint64_t acquisition_rate, uint64_t stream_rate) {
    char errBuff[2048] = {'\0'};
    this->stream_rate = stream_rate;
    this->channels = channels;
    this->acq_rate = acquisition_rate;

    for(auto &channel : channels){ // iterate through channels, check name and determine what tasks need to be created
        switch(channel.channelType){
            case ANALOG_VOLTAGE_IN:
                DAQmxCreateAIVoltageChan(taskHandle, channel.name.c_str(), "", DAQmx_Val_Cfg_Default, channel.min_val, channel.max_val, DAQmx_Val_Volts, NULL);
                taskType = ANALOG_READER;
                break;
            case THERMOCOUPLE_IN:               //TODO: Implement
                // DAQmxCreateAIThrmcplChan(taskHandle, channel.name.c_str(), "", channel.min_val, channel.max_val, DAQmx_Val_DegC, DAQmx_Val_BuiltIn, 10.0, DAQmx_Val_Poly, 0.0, 0.0, 0.0, NULL);
                taskType = ANALOG_READER;
                break;
            case ANALOG_CURRENT_IN:             //TODO: Implement
                // DAQmxCreateAICurrentChan(taskHandle, channel.name.c_str(), "", DAQmx_Val_Cfg_Default, channel.min_val, channel.max_val, DAQmx_Val_Amps, NULL);
                taskType = ANALOG_READER;
                break;
            case DIGITAL_IN:
                DAQmxCreateDIChan(taskHandle, channel.name.c_str(), "", DAQmx_Val_ChanPerLine);
                taskType = DIGITAL_READER;
                break;
            case DIGITAL_OUT:                   //TODO: Implement
                // DAQmxCreateDOChan(taskHandle, channel.name.c_str(), "", DAQmx_Val_ChanPerLine);
                taskType = DIGITAL_WRITER;
                break;
        }
        this->numChannels++; // change to handle index channels
    }
    std::cout << "NumChannels: " << numChannels << std::endl;
    if( taskType == ANALOG_READER){
        int err = DAQmxCfgSampClkTiming(taskHandle, "", acquisition_rate, DAQmx_Val_Rising, DAQmx_Val_ContSamps, acquisition_rate);
        if(err < 0){
            printf("DAQmx Error: %d\n",err);
            std::cout << "ERROR" << std::endl;
            DAQmxGetExtendedErrorInfo(errBuff,2048);
            printf("DAQmx Error: %s\n",errBuff);
        }
    }

    this->numSamplesPerChannel =  std::floor(acquisition_rate/stream_rate);
    this->bufferSize = this->numChannels*this->numSamplesPerChannel;
    this->data = new double[bufferSize];
    this->digitalData = new uInt32[bufferSize];
    printf("configured\n");
}

freighter::Error ni::niDaqReader::configure(synnax::Module config){
    return freighter::NIL;
}

freighter::Error ni::niDaqReader::start(){
   DAQmxStartTask(taskHandle);
   return freighter::NIL;
}

freighter::Error ni::niDaqReader::stop(){
    int daqmx_err = DAQmxStopTask(taskHandle);
    daqmx_err = DAQmxClearTask(taskHandle);
    delete[] data; // free the data buffer
    delete[] digitalData; // free the digital data buffer
    return freighter::NIL;
}

std::pair<synnax::Frame, freighter::Error> ni::niDaqReader::readAnalog(){
    signed long samplesRead = 0;
    char errBuff[2048] = {'\0'};
    float64 flush[1000];                     // to flush buffer before performing a read
    signed long flushRead;
    synnax::Frame f = synnax::Frame(numChannels);

    auto err1 = DAQmxReadAnalogF64(this->taskHandle,
                                   -1,
                                   10.0,
                                   DAQmx_Val_GroupByChannel,
                                   flush,
                                   1000,
                                   &samplesRead,
                                   NULL);

    std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
    auto err = DAQmxReadAnalogF64(this->taskHandle,
                                  this->numSamplesPerChannel,
                                  -1,
                                  DAQmx_Val_GroupByChannel,
                                  this->data,
                                  this->bufferSize,
                                  &samplesRead,
                                  NULL);
    std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;

    if (err < 0) {
        std::cout << "ERROR" << std::endl;
        DAQmxGetExtendedErrorInfo(errBuff,2048);
        printf("DAQmx Error: %s\n",errBuff);
    }

    // we interpolate the timestamps between the initial and final timestamp to ensure non-overlapping timestamps between read iterations
    uint64_t diff = final_timestamp - initial_timestamp;
    uint64_t incr = diff/this->numSamplesPerChannel;

    // Construct and populate index channel
    std::vector<std::uint64_t> time_index(this->numSamplesPerChannel);
    for (uint64_t i = 0; i < samplesRead; ++i) {
        time_index[i] = initial_timestamp + (std::uint64_t)(incr*i);
    }

    // Construct and populate synnax frame
    std::vector<float> data_vec(samplesRead);
    uint64_t data_index = 0;
    for(int i = 0; i <  numChannels; i++){
        if(channels[i].channelType == INDEX_CHANNEL ){
            f.add(channels[i].channel_key, synnax::Series(time_index, synnax::TIMESTAMP));
        }
        else{
            for(int j = 0; j < samplesRead; j++){
                data_vec[j] = data[data_index*samplesRead + j];
            }
            f.add(channels[i].channel_key, synnax::Series(data_vec));
            data_index++;
        }
    }

    freighter::Error error = freighter::NIL; // TODO: implement error handling
    return {std::move(f), error};
}

std::pair<synnax::Frame, freighter::Error> ni::niDaqReader::readDigital(){
    signed long samplesRead;
    char errBuff[2048]={'\0'};
    float64 flush[1000];                     // to flush buffer before performing a read
    signed long flushRead;
    synnax::Frame f = synnax::Frame(numChannels);


    std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
    auto err = DAQmxReadDigitalU32(this->taskHandle,this->numSamplesPerChannel,-1,DAQmx_Val_GroupByChannel,this->digitalData,this->bufferSize,&samplesRead,NULL);
    std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;
    if (err < 0) {
        std::cout << "ERROR" << std::endl;
        DAQmxGetExtendedErrorInfo(errBuff,2048);
        printf("DAQmx Error: %s\n",errBuff);
    }

    // we interpolate the timestamps between the initial and final timestamp to ensure non-overlapping timestamps between read iterations
    uint64_t diff = final_timestamp - initial_timestamp;
    uint64_t incr = diff/this->numSamplesPerChannel;

    // Construct and populate index channel
    std::vector<std::uint64_t> time_index(numSamplesPerChannel);
    for (int i = 0; i < samplesRead; ++i) { // populate time index channeL
        time_index[i] = initial_timestamp + (std::uint64_t )(incr*i);
    }

    // Construct and populate synnax frame
    std::vector<float> data_vec(samplesRead);
    uint64_t data_index = 0;
    for(int i = 0; i <  numChannels; i++){
        if(channels[i].channelType == INDEX_CHANNEL ){
            f.add(channels[i].channel_key, synnax::Series(time_index, synnax::TIMESTAMP));
        }
        else{
            std::cout << "SamplesRead: " << samplesRead << std::endl;
            for(int j = 0; j < samplesRead; j++){
                data_vec[j] = digitalData[data_index*samplesRead + j];
            }
            f.add(channels[i].channel_key, synnax::Series(data_vec));
            data_index++;
        }
    }
    freighter::Error error = freighter::NIL;
    return {std::move(f), error};
}

std::pair<synnax::Frame, freighter::Error> ni::niDaqReader::read(){
    if(taskType == ANALOG_READER){
        return readAnalog();
    }
    else if(taskType == DIGITAL_READER){
        return readDigital();
    }
    else{
        return {synnax::Frame(0), freighter::NIL};
    }
}
///////////////////////////////////////////////////////////////////////////////////
// niDaqWriter
///////////////////////////////////////////////////////////////////////////////////


niDaqWriter::niDaqWriter(TaskHandle taskHandle) : taskHandle(taskHandle) {}

void niDaqWriter::init(json config, synnax::ChannelKey ack_index_key) {
    std::vector<channel_config> channel_configs;
    auto channels = config["channels"];
    auto deviceName = config["device"].get<std::string>();
    for (auto &channel : channels) {
        channel_config config;

        std::string name = channel["name"].get<std::string>();
        channelNames.emplace_back(name);

        auto type = channel["type"].get<std::string>();
        std::string portName =  (type == "digitalOutput") ? "port" : "";

        config.name = deviceName + "/" + portName + channel["port"].dump().c_str();

        config.channel_key = channel["cmd_key"].get<uint32_t>();

        config.channelType = (type == "digitalOutput") ? DIGITAL_OUT
                            : (type == "index") ? INDEX_CHANNEL
                            : INVALID_CHANNEL;

        channel_configs.push_back(config);

        cmd_channel_keys.push_back(channel["cmd_key"].get<uint32_t>());
        ack_channel_keys.push_back(channel["ack_key"].get<uint32_t>());
    }
    init(channel_configs);
    // set ack index channel keys
    this->ack_index_key = ack_index_key;
    assert(ack_index_key != 0);
    assert(cmd_channel_keys.size() == ack_channel_keys.size());
    assert(cmd_channel_keys.size() != 0);
    assert(ack_channel_keys.size() != 0);
}

void ni::niDaqWriter::init(std::vector <channel_config> channels){
    this->channels = channels;
    for(auto &channel : channels){ // iterate through channels, check name and determine what tasks need to be created
        switch(channel.channelType){
            case DIGITAL_OUT:
                DAQmxCreateDOChan(taskHandle, channel.name.c_str(), "", DAQmx_Val_ChanPerLine);
                taskType = DIGITAL_WRITER;
                break;
        }
        this->numChannels++; // change to handle index channels
    }
    this->bufferSize = this->numChannels;
    this->writeBuffer = new uint8_t[this->bufferSize];
}

freighter::Error ni::niDaqWriter::configure(synnax::Module config){
    return freighter::NIL;
}

freighter::Error ni::niDaqWriter::start(){
    DAQmxStartTask(taskHandle);
    return freighter::NIL;
}

freighter::Error ni::niDaqWriter::stop(){
    int daqmx_err = DAQmxStopTask(taskHandle);
    daqmx_err = DAQmxClearTask(taskHandle);
    delete[] writeBuffer;
    return freighter::NIL;
}

freighter::Error ni::niDaqWriter::write(synnax::Frame frame){ // TODO: should this function get a Frame or a bit vector of setpoints instead?
    if(taskType == DIGITAL_WRITER){
        return writeDigital(frame);
    }
    return freighter::NIL;
}

std::pair <synnax::Frame, freighter::Error> ni::niDaqWriter::writeDigital(synnax::Frame frame){
    char errBuff[2048] = {'\0'};
    signed long samplesWritten = 0;
    formatData(frame);
    auto err = DAQmxWriteDigitalLines(taskHandle,
                                        1, // number of samples per channel
                                        1, // auto start
                                        10.0, // timeout
                                        DAQmx_Val_GroupByChannel, //data layout
                                        writeBuffer, // data
                                        &samplesWritten, // samples written
                                        NULL); // reserved
    if(err < 0){
        std::cout << "ERROR" << std::endl;
        DAQmxGetExtendedErrorInfo(errBuff,2048);
        printf("DAQmx Error: %s\n",errBuff);
    }

    // deal with acknowledgementsb
    auto ack_frame = synnax::Frame(ack_queue.size() + 1);
    auto timeSeries = synnax::Series(std::vector<uint64_t>{synnax::TimeStamp::now().value});
    ack_frame.add(ack_index_key, timeSeries);

    while(!ack_queue.empty()){
        auto ack_key = ack_queue.front();
        ack_frame.add(ack_key, synnax::Series(1));
        ack_queue.pop();
    }

    return {std::move(ack_frame), freighter::NIL};
}

freighter::Error ni::niDaqWriter::formatData(synnax::Frame frame){
    uint32_t i = 0;
    for (auto key : cmd_channel_keys){ // the order the keys are in is the order the data is written
        auto it = keys.find(key);
        if (it != keys.end()){
            uint32_t index = std::distance(keys.begin(), it);
            auto series = (*frame.series)[index];
            memcpy(writeBuffer + i, series.data.uint8(), sizeof(uint8_t)); //TODO make sure this works
            i++;
        }
        ack_queue.push(ack_channel_keys[i]);
    }
    return freighter::NIL;
}

// TODO create a helper function that takes in a frame and formats into the data to pass into writedigital
// TODO: create a helper function to parse digital data configuration of wehtehr its a port to r line


//typedef freighter::Error (*DAQmxCreateChannel) (TaskHandle taskHandle, ChannelConfig config);
//
//freighter::Error create_ai_voltage_channel(TaskHandle taskHandle, json config) {
//    auto physical_channel = config["physical_channel"].get<std::string>();
//    auto max_val = config["max_val"].get<float>();
//    auto min_val = config["min_val"].get<float>();
//    DAQmxCreateAIVoltageChan(
//            taskHandle,
//            physical_channel.c_str(),
//            NULL,
//            DAQmx_Val_Cfg_Default,
//            min_val,
//            max_val,
//            DAQmx_Val_Volts,
//            NULL
//    );
//}
//
//static std::map<std::string, DAQmxCreateChannel> create_channel_map = {
//        {"ai_voltage", create_ai_voltage_channel}
//};
//
//freighter::Error Reader::configure(synnax::Module config) {
//    // Step 1 is parsing the configuration
//    json j = json::parse(config.config);
//
//    // Grab the sample rate key
//    auto sample_rate_val = j.find("sample_rate");
//    if (sample_rate_val == j.end()) return freighter::NIL;
//    if (!sample_rate_val->is_number_float()) return freighter::NIL;
//    auto sample_rate = synnax::Rate(sample_rate_val->get<std::float_t>());
//
//    // Grab the transfer rate key
//    auto transfer_rate_val = j.find("transfer_rate");
//    if (transfer_rate_val == j.end()) return freighter::NIL;
//    if (!transfer_rate_val->is_number_float()) return freighter::NIL;
//    auto transfer_rate = synnax::Rate(transfer_rate_val->get<std::float_t>());
//
//    // Grab the channels key
//    auto channels_val = j.find("channels");
//    if (channels_val == j.end()) return freighter::NIL;
//    if (!channels_val->is_array()) return freighter::NIL;
//    auto channels = channels_val->get<std::vector<json>>();
//    for (auto &channel: channels) {
//        if (!channel.is_object()) return freighter::NIL;
//        auto type_val = channel.find("type");
//        if (type_val == channel.end()) return freighter::NIL;
//        if (!type_val->is_string()) return freighter::NIL;
//        auto type = type_val->get<std::string>();
//    }
//}

//    int32 daqmx_err = DAQmxCreateTask(config.name.c_str(), &task);
//    uInt64 samples_per_chan = uInt64(config.sample_rate.value / config.transfer_rate.value);
//
//    for (auto &channel: config.channels) {
//        auto create_channel = create_channel_map[channel.type];
//        auto err = create_channel(task, channel);
//    }
//
//    daqmx_err = DAQmxCfgSampClkTiming(
//            task,
//            NULL,
//            config.sample_rate.value,
//            DAQmx_Val_Rising,
//            DAQmx_Val_ContSamps,
//            samples_per_chan
//    );
//
//    daqmx_err = DAQmxStartTask(task);
//}
//

//}