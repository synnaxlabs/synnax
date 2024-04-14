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

#include "driver/driver/ni/ni_reader.h"
#include "nlohmann/json.hpp"
#include "client/cpp/synnax/telem/telem.h"
#include <utility>
#include <chrono>
#include <stdio.h>
#include <cassert>

using json = nlohmann::json;
using namespace ni; // TODO; remove

///////////////////////////////////////////////////////////////////////////////////
// NI Error Handling
///////////////////////////////////////////////////////////////////////////////////
int ni::checkNIError(int32 error, json &errInfo){
    if(error < 0){
        char errBuff[2048] = {'\0'};
        DAQmxGetExtendedErrorInfo(errBuff,2048);
        errInfo["error"] = errBuff;
        return -1;
    }
    return 0;
}

///////////////////////////////////////////////////////////////////////////////////
// niDaqReader
///////////////////////////////////////////////////////////////////////////////////

niDaqReader::niDaqReader(TaskHandle taskHandle) : taskHandle(taskHandle) {}

std::pair<json,int> ni::niDaqReader::init(json config, uint64_t acquisition_rate, uint64_t stream_rate) {
    std::cout << "Init Reader" << std::endl;
    std::vector<channel_config> channel_configs;
    auto channels = config["channels"];
    auto deviceName = config["hardware"].get<std::string>();
    for (auto &channel : channels) {
        auto type = channel["type"].get<std::string>();
        channel_config config;
        std::string portName =  (type == "analogVoltageInput") ? "ai"
                             :  (type == "digitalInput") ? "port"
                             : "";
        config.name = (type == "analogVoltageInput") ? deviceName + "/" + portName + channel["port"].dump().c_str()
                    : (type == "digitalInput") ? deviceName + "/" + portName + channel["port"].dump().c_str() + "/line" + channel["line"].dump().c_str()
                    : (type == "index") ? channel["name"].get<std::string>()
                    : "INVALID CHANNEL";
//        printf("Channel Name: %s\n", config.name.c_str());
        std::cout << "Channel Name: " << config.name << std::endl;
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
    auto [err_info, err] =  init(channel_configs, acquisition_rate, stream_rate);
    if(err < 0){
        return {err_info, -1};
    }
    return {err_info, 0};
}

// TODO: I want to make a producer consumer model here instead? This works for now but come back to

std::pair<json,int> ni::niDaqReader::init(std::vector<channel_config> channels, uint64_t acquisition_rate, uint64_t stream_rate) {
    printf("Init Reader\n");
    char errBuff[2048] = {'\0'};
    this->stream_rate = stream_rate;
    this->channels = channels;
    this->acq_rate = acquisition_rate;
    int err = 0;
    for(auto &channel : channels){ // iterate through channels, check name and determine what tasks need to be created
        switch(channel.channelType){
            case ANALOG_VOLTAGE_IN:
                printf("Creating AI Voltage Channel\n");
                err = ni::checkNIError(DAQmxCreateAIVoltageChan(taskHandle,
                                         channel.name.c_str(),
                                         "",
                                         DAQmx_Val_Cfg_Default,
                                         channel.min_val,
                                         channel.max_val,
                                         DAQmx_Val_Volts, NULL),
                             errInfo);
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
                err = ni::checkNIError(DAQmxCreateDIChan(taskHandle,
                                  channel.name.c_str(),
                                  "",
                                  DAQmx_Val_ChanPerLine),
                            this->errInfo);
                taskType = DIGITAL_READER;
                break;
        }
        this->numChannels++; // change to handle index channels
        if(err < 0){
            return {this->errInfo, -1};
        }
    }

    if( taskType == ANALOG_READER){ // only configure timing if we are reading analog data
         err = ni::checkNIError(DAQmxCfgSampClkTiming(taskHandle,
                                        "",
                                        acquisition_rate,
                                        DAQmx_Val_Rising,
                                        DAQmx_Val_ContSamps,
                                        acquisition_rate),
                     this->errInfo);
    }else if (taskType == DIGITAL_READER){
//        DAQmxSetSampTimingType(taskHandle, DAQmx_Val_SampClk);
        err = ni::checkNIError(DAQmxCfgSampClkTiming(taskHandle,
                                        "",
                                        acquisition_rate,
                                        DAQmx_Val_Rising,
                                        DAQmx_Val_ContSamps,
                                        acquisition_rate),
                         this->errInfo);
    }

    if(err < 0){
        return {this->errInfo, -1};
    }

    this->numSamplesPerChannel =  std::floor(acquisition_rate/stream_rate);
    this->bufferSize = this->numChannels*this->numSamplesPerChannel;
    this->data = new double[bufferSize];
    this->digitalData = new uInt8[bufferSize]; // TODO don't let multiple news happen
    return {this->errInfo, 0};
}




freighter::Error ni::niDaqReader::start(){
   auto err = ni::checkNIError(DAQmxStartTask(taskHandle),
                               this->errInfo);
   if(err < 0){
       printf("Error starting task\n"); // print error info
       printf("Error: %s\n", errInfo.dump().c_str());
       return freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR);
   }
   return freighter::NIL;
}

freighter::Error ni::niDaqReader::stop(){ //TODO: don't let multiple closes happen
    auto err = freighter::NIL;
    auto err1 = ni::checkNIError(DAQmxStopTask(taskHandle),
                                this->errInfo);
    if(err1 < 0){
        err = freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR);
    }
    auto err2 = ni::checkNIError(DAQmxClearTask(taskHandle),
                                 this->errInfo);
    if(err2 < 0){
        err = freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR);
    }
    delete[] data; // free the data buffer TODO: change this to a smart pointer
    delete[] digitalData; // free the digital data buffer TODO: change this to a smart pointer
    return err;
}

std::pair<synnax::Frame, freighter::Error> ni::niDaqReader::readAnalog(){
    signed long samplesRead = 0;
    char errBuff[2048] = {'\0'};
    float64 flush[1000];                     // to flush buffer before performing a read
    signed long flushRead;
    synnax::Frame f = synnax::Frame(numChannels);
    int err = 0;
    // initial read to flush buffer
    err = ni::checkNIError(DAQmxReadAnalogF64(this->taskHandle,
                                   -1,                          // reads all available samples in buffer
                                   10.0,
                                   DAQmx_Val_GroupByChannel,
                                   flush,
                                   1000,
                                   &samplesRead,
                                   NULL),
                           this->errInfo);
    if (err < 0) {
        return std::make_pair(std::move(f), freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR, "error reading analog data"));
    }

    std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
    // actual read of analog lines
    err = ni::checkNIError(DAQmxReadAnalogF64(this->taskHandle,
                                  this->numSamplesPerChannel,
                                  -1,
                                  DAQmx_Val_GroupByChannel,
                                  this->data,
                                  this->bufferSize,
                                  &samplesRead,
                                  NULL),
                           errInfo);
    std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;
   if (err < 0) {
        return std::make_pair(std::move(f), freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR, "Error reading analog data"));
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
    // return synnax frame
    freighter::Error error = freighter::NIL; // TODO: implement error handling
    return std::make_pair(std::move(f), error);
}

std::pair<synnax::Frame, freighter::Error> ni::niDaqReader::readDigital(){
    signed long samplesRead;
    char errBuff[2048]={'\0'};
    uInt8 flushBuffer[10000];                     // to flush buffer before performing a read
    uInt8 dataBuffer[10000];
    signed long flushRead;
    synnax::Frame f = synnax::Frame(numChannels);
    int32 * numBytesPerSamp;
    int err = 0;
    //initial read to flush buffer
    std::cout << "Flushing buffer" << std::endl;
    err = ni::checkNIError(DAQmxReadDigitalLines(this->taskHandle, //TODO: come back to and make sure this call to flush will be fine at any scale (elham)
                                     -1, // reads all available samples in the buffer
                                     -1,
                                     DAQmx_Val_GroupByChannel,
                                     flushBuffer,
                                     1000,
                                     &samplesRead,
                                     numBytesPerSamp,
                                     NULL),
                           this->errInfo);
    if (err < 0) {
        return std::make_pair(std::move(f), freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR, "Error reading digital data"));
    }
    std::cout << "performing actual read" << std::endl;
    std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
    // actual read to of digital lines
    err = ni::checkNIError(DAQmxReadDigitalLines(this->taskHandle,                                      //task handle
                                     this->numSamplesPerChannel,                            //numSampsPerChan
                                     -1,                                                    //timeout
                                     DAQmx_Val_GroupByChannel,                              //dataLayout
                                     dataBuffer,                                            //readArray
                                     10000,          //arraySizeInSamps
                                     &samplesRead,                                          //sampsPerChanRead
                                     NULL,                                       //numBytesPerSamp
                                     NULL),
                           this->errInfo);                                                 //reserved
    std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;
    if (err < 0) {
        return std::make_pair(std::move(f), freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR, "error reading analog data"));
    }
    std::cout << "Read complete " << std::endl;
    std::cout << "Samples Read: " << samplesRead << std::endl;
    // we interpolate the timestamps between the initial and final timestamp to ensure non-overlapping timestamps between read iterations
    uint64_t diff = final_timestamp - initial_timestamp;
    std::cout << "Diff: " << diff << std::endl;
    uint64_t incr = diff/this->numSamplesPerChannel;
    // Construct and populate index channel
    std::vector<std::uint64_t> time_index(numSamplesPerChannel);
    for (int i = 0; i < samplesRead; ++i) { // populate time index channeL
        time_index[i] = initial_timestamp + (std::uint64_t )(incr*i);
    }
    // Construct and populate synnax frame
    std::vector<uint8_t> data_vec(samplesRead);
    uint64_t data_index = 0;
    for(int i = 0; i <  numChannels; i++){
        if(channels[i].channelType == INDEX_CHANNEL ){
            f.add(channels[i].channel_key, synnax::Series(time_index, synnax::TIMESTAMP));
        }
        else{
            for(int j = 0; j < samplesRead; j++){
                data_vec[j] = dataBuffer[data_index*samplesRead + j];
            }
            f.add(channels[i].channel_key, synnax::Series(data_vec));
            data_index++;
        }
    }
    // return synnax frame
    freighter::Error error = freighter::NIL;
    return std::make_pair(std::move(f), error);
}

std::pair<synnax::Frame, freighter::Error> ni::niDaqReader::read(){
    if(taskType == ANALOG_READER){
        return readAnalog();
    }
    else if(taskType == DIGITAL_READER){
        return readDigital();
    }
    else{
        return std::make_pair(synnax::Frame(0), freighter::NIL);
    }
}


json ni::niDaqReader::getErrorInfo() {
    if(errInfo.empty()){
        return NULL;
    } else {
        this->stop();
        return errInfo;
    }
}

///////////////////////////////////////////////////////////////////////////////////
// niDaqWriter
///////////////////////////////////////////////////////////////////////////////////


niDaqWriter::niDaqWriter(TaskHandle taskHandle) : taskHandle(taskHandle) {}

std::pair<json,int> niDaqWriter::init(json config, synnax::ChannelKey ack_index_key) {
    std::vector<channel_config> channel_configs;
    auto channels = config["channels"];
    auto deviceName = config["hardware"].get<std::string>();
    for (auto &channel : channels) {
        channel_config config;

        std::string name = channel["name"].get<std::string>();

        auto type = channel["type"].get<std::string>();
        std::string portName =  (type == "digitalOutput") ? "port" : "";

        config.name = deviceName + "/" + portName + channel["port"].dump().c_str() + "/line" + channel["line"].dump().c_str();
        config.channel_key = channel["cmd_key"].get<uint32_t>();

        config.channelType = (type == "digitalOutput") ? DIGITAL_OUT
                            : (type == "index") ? INDEX_CHANNEL
                            : (type == "ackIndex") ? INDEX_CHANNEL
                            : INVALID_CHANNEL;

        channel_configs.push_back(config);

        cmd_channel_keys.push_back(channel["cmd_key"].get<uint32_t>());
        ack_channel_keys.push_back(channel["ack_key"].get<uint32_t>());
    }
    // set ack index channel keys
    this->ack_index_key = ack_index_key;
    assert(ack_index_key != 0);
    assert(cmd_channel_keys.size() == ack_channel_keys.size());
    assert(cmd_channel_keys.size() != 0);
    assert(ack_channel_keys.size() != 0);
    auto [error_info, err] = init(channel_configs);
    if(err < 0){
        return {error_info, -1};
    }
    return {error_info, 0};
}

std::pair<json,int> ni::niDaqWriter::init(std::vector <channel_config> channels){
    int err = 0;
    this->channels = channels;
    for(auto &channel : channels){ // iterate through channels, check name and determine what tasks need to be created
        switch(channel.channelType){
            case DIGITAL_OUT:
                err = ni::checkNIError(DAQmxCreateDOChan(taskHandle,
                                                   channel.name.c_str(),
                                                   "",
                                                   DAQmx_Val_ChanPerLine),
                                 errInfo);
                taskType = DIGITAL_WRITER;
                break;
        }
        this->numChannels++; // change to handle index channels
        if(err < 0){
            return {this->errInfo, -1};
        }
    }
    this->bufferSize = this->numChannels;
    this->writeBuffer = new uint8_t[this->bufferSize];
    for(int i = 0; i < this->bufferSize; i++){
        writeBuffer[i] = 0;
    }
    return {this->errInfo, 0};
}

freighter::Error ni::niDaqWriter::start(){
    auto err = ni::checkNIError(DAQmxStartTask(taskHandle),
                                this->errInfo);
    if(err < 0){
        return freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR);
    }
    return freighter::NIL;
}

freighter::Error ni::niDaqWriter::stop(){
    auto err = freighter::NIL;

    auto err1 = ni::checkNIError(DAQmxStopTask(taskHandle),
                                 this->errInfo);
    if(err1 < 0){
        err = freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR);
    }
    auto err2 = ni::checkNIError(DAQmxClearTask(taskHandle),
                                 this->errInfo);
    if(err2 < 0){
        err = freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR);
    }
    DAQmxClearTask(taskHandle);
    delete[] writeBuffer;
    return err;
}

std::pair <synnax::Frame, freighter::Error> ni::niDaqWriter::write(synnax::Frame frame){ // TODO: should this function get a Frame or a bit vector of setpoints instead?
    if(taskType == DIGITAL_WRITER){
        return writeDigital(std::move(frame));
    }else{
        return {synnax::Frame(0), freighter::NIL};
    }
}

std::pair <synnax::Frame, freighter::Error> ni::niDaqWriter::writeDigital(synnax::Frame frame){
    std::cout << "Writing to daq" << std::endl;
    char errBuff[2048] = {'\0'};
    signed long samplesWritten = 0;
    formatData(std::move(frame));
    int err = 0;
    err = ni::checkNIError(DAQmxWriteDigitalLines(this->taskHandle,
                                        1, // number of samples per channel
                                        1, // auto start
                                        10.0, // timeout
                                        DAQmx_Val_GroupByChannel, //data layout
                                        writeBuffer, // data
                                        &samplesWritten, // samples written
                                        NULL),
                           this->errInfo); // reserved
    if(err < 0){
        return std::make_pair(synnax::Frame(0), freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR, "Error reading digital data"));
    }
    // Construct acknowledgement frame
    auto ack_frame = synnax::Frame(ack_queue.size() + 1);
    ack_frame.add(ack_index_key, synnax::Series(std::vector<uint64_t>{synnax::TimeStamp::now().value}, synnax::TIMESTAMP));
    std::cout << "Ack Index Key: " << ack_index_key << std::endl;
    while(!ack_queue.empty()){
        std::cout << "Ack Queue Size: " << ack_queue.size() << std::endl;
        auto ack_key = ack_queue.front();
        std::cout << "Ack Key: " << ack_key << std::endl;
        ack_frame.add(ack_key, synnax::Series(std::vector<uint8_t>{1}));
        ack_queue.pop();
    }
    return {std::move(ack_frame), freighter::NIL};
}

freighter::Error ni::niDaqWriter::formatData(synnax::Frame frame){
    uint32_t frame_index = 0;
    uint32_t cmd_channel_index = 0;
    for (auto key : *(frame.channels)){ // the order the keys are in is the order the data is written
        auto it = std::find(cmd_channel_keys.begin(), cmd_channel_keys.end(), key);
        if (it != cmd_channel_keys.end()){
            std::cout << "channel key is " << key << std::endl;
            std::cout << "frame index is" << frame_index << std::endl;
            cmd_channel_index = std::distance(cmd_channel_keys.begin(), it) ;
            auto series = frame.series->at(frame_index).uint8(); // used to be auto &series
            writeBuffer[cmd_channel_index] = series[0];
//                    std::cout << "series is " << (uint32_t)series[0] << std::endl;
            ack_queue.push(ack_channel_keys[cmd_channel_index]);
        }
        frame_index++;
    }
    return freighter::NIL;
}

json ni::niDaqWriter::getErrorInfo() {
    if(errInfo.empty()){
        return NULL;
    } else {
        this->stop();
        return errInfo;
    }
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

//