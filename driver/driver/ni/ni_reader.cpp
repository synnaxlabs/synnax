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
#include <stdio.h>

//#include "<cmath>"

using json = nlohmann::json;
using namespace ni;

niDaqReader::niDaqReader(TaskHandle taskHandle) : taskHandle(taskHandle) {}

void ni::niDaqReader::init(std::vector<channel_config> channels, uint64_t acquisition_rate, uint64_t stream_rate) {
    this->stream_rate = stream_rate;
    this->channels = channels;
    this->acq_rate = acquisition_rate;
    // iterate through channels, check name and determine what tasks nbeed to be created
    for(auto &channel : channels){
//        std::cout << "Device name: " << channel.name.c_str() << std::endl;
        switch(channel.channelType){
            case ANALOG_VOLTAGE_IN:
//                std::cout << "Creating AI Voltage Channel" << std::endl;
                DAQmxCreateAIVoltageChan(taskHandle, channel.name.c_str(), "", DAQmx_Val_Cfg_Default, channel.min_val, channel.max_val, DAQmx_Val_Volts, NULL);
                taskType = ANALOG_READER;
                break;
            case THERMOCOUPLE_IN: //TODO: double check the function calls below (elham)
//                DAQmxCreateAIThrmcplChan(taskHandle, channel.name.c_str(), "", channel.min_val, channel.max_val, DAQmx_Val_DegC, DAQmx_Val_BuiltIn, 10.0, DAQmx_Val_Poly, 0.0, 0.0, 0.0, NULL);
                taskType = ANALOG_READER;
                break;
            case ANALOG_CURRENT_IN://TODO: double check the function calls below (elham)
//                DAQmxCreateAICurrentChan(taskHandle, channel.name.c_str(), "", DAQmx_Val_Cfg_Default, channel.min_val, channel.max_val, DAQmx_Val_Amps, NULL);
                taskType = ANALOG_READER;
                break;
            case DIGITAL_IN://TODO: double check the function calls below (elham)
                std::cout << "Creating DIGITAL in Channel" << std::endl;
                DAQmxCreateDIChan(taskHandle, channel.name.c_str(), "", DAQmx_Val_ChanPerLine);
                taskType = DIGITAL_READER;
                break;
            case DIGITAL_OUT://TODO: double check the function calls below (elham)
//                DAQmxCreateDOChan(taskHandle, channel.name.c_str(), "", DAQmx_Val_ChanPerLine);
                taskType = DIGITAL_WRITER;
                break;
        }
        numChannels++;
    }
    if( taskType == ANALOG_READER) DAQmxCfgSampClkTiming(taskHandle, "", acquisition_rate, DAQmx_Val_Rising, DAQmx_Val_ContSamps, 10000);
    this->numSamplesPerChannel = std::floor(acquisition_rate/stream_rate);
    this->bufferSize = this->numChannels*this->numSamplesPerChannel;
    this->data = new double[bufferSize];
    this->digitalData = new uInt32[bufferSize];
    std::cout << " finished configuring taskHandle" << std::endl;
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
    delete[] data; // free the data buffer
    delete[] digitalData; // free the digital data buffer
    daqmx_err = DAQmxClearTask(taskHandle);
    return freighter::NIL;
}



std::pair<synnax::Frame, freighter::Error> ni::niDaqReader::readAnalog(){
    signed long samplesRead;
    char errBuff[2048]={'\0'};
    std::vector<long> time_index(numSamplesPerChannel);
    synnax::Frame f = synnax::Frame(numChannels); // make a synnax frame
    long initial_timestamp = (synnax::TimeStamp::now()).value;

    std::cout << "Acquiring samples" << std::endl;
    std::cout << "numSamplesPerChan: " << this->numSamplesPerChannel << std::endl;
    DAQmxReadAnalogF64(this->taskHandle,this->numSamplesPerChannel,-1,DAQmx_Val_GroupByChannel,this->data,this->bufferSize,&samplesRead,NULL);
    printf("Acquired %d samples\n",(int)samplesRead);

    DAQmxGetExtendedErrorInfo(errBuff,2048);
    printf("DAQmx Error: %s\n",errBuff);
    for (int i = 0; i < samplesRead; ++i) { // populate time index channeL
        time_index[i] = initial_timestamp + ((synnax::NANOSECOND/acq_rate)*i).value;
    }

    std::vector<float> data_vec(samplesRead);
    // populate data
    for(int i = 0; i <  numChannels; i++){
        std::cout << "SamplesRead: " << samplesRead << std::endl;
        for(int j = 0; j < samplesRead; j++){
            data_vec[j] = data[i*samplesRead + j];
        }
        f.add(channels[i].channel_key, synnax::Series(data_vec));
    }
    freighter::Error error = freighter::NIL;
    return {std::move(f), error};
}

std::pair<synnax::Frame, freighter::Error> ni::niDaqReader::readDigital(){
    signed long samplesRead;
    char errBuff[2048]={'\0'};
    std::vector<long> time_index(numSamplesPerChannel);
    synnax::Frame f = synnax::Frame(numChannels); // make a synnax frame
    long initial_timestamp = (synnax::TimeStamp::now()).value;
    std::cout << "Acquiring samples" << std::endl;
    std::cout << "numSamplesPerChan: " << this->numSamplesPerChannel << std::endl;
    std::cout << "Buffer Size: " << this->bufferSize << std::endl;
    DAQmxReadDigitalU32(this->taskHandle,this->numSamplesPerChannel,-1,DAQmx_Val_GroupByChannel,this->digitalData,this->bufferSize,&samplesRead,NULL);
    printf("Acquired %d samples\n",(int)samplesRead);

    DAQmxGetExtendedErrorInfo(errBuff,2048);
    printf("DAQmx Error: %s\n",errBuff);
    for (int i = 0; i < samplesRead; ++i) { // populate time index channeL
        time_index[i] = initial_timestamp + ((synnax::NANOSECOND/acq_rate)*i).value;
    }
    std::cout << "Time Index data written" << std::endl;

    std::vector<float> data_vec(samplesRead);
    // populate data
    std::cout << "Populating data" << std::endl;
    for(int i = 0; i <  numChannels; i++){
        std::cout << "SamplesRead: " << samplesRead << std::endl;
        for(int j = 0; j < samplesRead; j++){
//            std::cout << "Data: " << data[i*samplesRead + j] << std::endl;
            data_vec[j] = digitalData[i*samplesRead + j];
        }

        f.add(channels[i].channel_key, synnax::Series(data_vec));
    }
    freighter::Error error = freighter::NIL;
    return {std::move(f), error};
}

std::pair<synnax::Frame, freighter::Error> ni::niDaqReader::read(){\
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

//
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