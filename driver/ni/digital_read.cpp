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

#include "driver/ni/ni.h"
#include "nlohmann/json.hpp"
#include "client/cpp/telem/telem.h"
#include <utility>
#include <chrono>
#include <stdio.h>
#include <cassert>
#include "glog/logging.h"

using json = nlohmann::json;

void ni::DigitalReadSource::getIndexKeys(){
    std::set<std::uint32_t> index_keys;
    //iterate through channels in reader config
    for (auto &channel : this->reader_config.channels){
        auto [channel_info, err] = this->ctx->client->channels.retrieve(channel.channel_key);
        // TODO handle error with breaker
        if (err != freighter::NIL){
            // Log error
            LOG(ERROR) << "[NI Reader] failed to retrieve channel " << channel.channel_key;
            this->ok_state = false;
            return;
        } else{
            index_keys.insert(channel_info.index);
        }
    }
    // now iterate through the set and add all the index channels as configs
    for (auto it = index_keys.begin(); it != index_keys.end(); ++it){
        auto index_key = *it;
        LOG(INFO) << "constructing index channel configs";
        auto [channel_info, err] = this->ctx->client->channels.retrieve(index_key);
        if (err != freighter::NIL){
            LOG(ERROR) << "[NI Reader] failed to retrieve channel " << index_key;
            this->ok_state = false;
            return;
        } else{
            ni::ChannelConfig index_channel;
            index_channel.channel_key = channel_info.key;
            index_channel.channel_type = "index";
            index_channel.name = channel_info.name;
            this->reader_config.channels.push_back(index_channel);
            LOG(INFO) << "[NI Reader] index channel " << index_channel.channel_key << " and name: " << index_channel.name <<" added to task " << this->reader_config.task_name;
        }
    }
}




// TODO: Code dedup
ni::DigitalReadSource::DigitalReadSource(
    TaskHandle task_handle,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task task) : task_handle(task_handle), ctx(ctx){
    // Create parser
    auto config_parser = config::Parser(task.config);
    this->reader_config.task_name = task.name;
    this->reader_config.task_key = task.key;


    // Parse configuration and make sure it is valid
    this->parseConfig(config_parser);

    if (!config_parser.ok()){
        // Log error
        LOG(ERROR) << "[NI Reader] failed to parse configuration for " << this->reader_config.task_name;
        this->ctx->setState({.task = task.key,
                             .variant = "error",
                             .details = config_parser.error_json()});
        this->ok_state = false;
        return;
    }
    LOG(INFO) << "[NI Reader] successfully parsed configuration for " << this->reader_config.task_name;

    //TODO: 
    this->getIndexKeys(); // get index keys for the task     

    LOG(INFO) << "[NI Reader] index keys retrieved " << this->reader_config.task_name;

    // Create breaker
    auto breaker_config = breaker::Config{
        .name = task.name,
        .base_interval = 1 * SECOND,
        .max_retries = 20,
        .scale = 1.2,
    };
    this->breaker = breaker::Breaker(breaker_config);

    // TODO: make sure you have all the channel info you could possible need
    // Now configure the actual NI hardware
    if (this->init()){
        LOG(ERROR) << "[NI Reader] Failed while configuring NI hardware for task " << this->reader_config.task_name;
        this->ok_state = false;
    }

    this->start(); // errors are handled in start
}


void ni::DigitalReadSource::parseConfig(config::Parser &parser){
    // Get Acquisition Rate and Stream Rates
    this->reader_config.sample_rate = parser.required<uint64_t>("sample_rate");
    this->reader_config.stream_rate = parser.required<uint64_t>("stream_rate");
    this->reader_config.device_key = parser.required<std::string>("device");
    this->reader_config.timing_source = "none"; // parser.required<std::string>("timing_source"); TODO: uncomment this when ui provides timing source

    // TODO: add a parser ok check here


    auto [dev, err] = this->ctx->client->hardware.retrieveDevice(this->reader_config.device_key);

    if (err != freighter::NIL) {
        LOG(ERROR) << "[NI Reader] failed to retrieve device " << this->reader_config.device_name;
        this->ok_state = false;
        return;
    }
    this->reader_config.device_name = dev.location;
    assert(parser.ok());

    // now parse the channels
    parser.iter("channels",
                [&](config::Parser &channel_builder){
                    ni::ChannelConfig config;

                    // digital channel names are formatted: <device_name>/port<port_number>/line<line_number>
                    config.name = (this->reader_config.device_name + "/port" + std::to_string(channel_builder.required<std::uint64_t>("port")) + "/line" + std::to_string(channel_builder.required<std::uint64_t>("line")));

                    config.channel_key = channel_builder.required<uint32_t>("channel");

                    // TODO: there could be more than 2 state logic
                    config.min_val = 0;
                    config.max_val = 1;

                    this->reader_config.channels.push_back(config);
                });
    assert(parser.ok());
}

int ni::DigitalReadSource::init(){
    int err = 0;
    auto channels = this->reader_config.channels;

    for (auto &channel : channels){
        if (channel.channel_type != "index" ){
            err = this->checkNIError(ni::NiDAQmxInterface::CreateDIChan(task_handle, channel.name.c_str(), "", DAQmx_Val_ChanPerLine));
            LOG(INFO) << "Channel name: " << channel.name;
        } 
        LOG(INFO) << "Index channel added to task: " << channel.name;
        this->numChannels++; 
        if (err < 0){
            LOG(ERROR) << "[NI Reader] failed while configuring channel " << channel.name;
            this->ok_state = false;
            return -1;
        }
    }

    // Configure buffer size and read resources
     if(this->reader_config.sample_rate < this->reader_config.stream_rate){
        this->err_info["error type"] = "Configuration Error";
        this->err_info["error details"] = "Stream rate is greater than sample rate";
        
        this->ctx->setState({.task = this->reader_config.task_key,
                             .variant = "error",
                             .details = err_info});
        LOG(ERROR) << "[NI Reader] stream rate is greater than sample rate " << this->reader_config.task_name;
        this->ok_state = false;
        return -1;
    }

    if (this->configureTiming()){
        LOG(ERROR) << "[NI Reader] Failed while configuring timing for NI hardware for task " << this->reader_config.task_name;
        this->ok_state = false;
    }
    
    LOG(INFO) << "[NI Reader] successfully configured NI hardware for task " << this->reader_config.task_name;
    return 0;
}

int ni::DigitalReadSource::configureTiming(){

    if(this->reader_config.timing_source == "none"){ // if timing is not enabled, implement timing in software
        this->reader_config.period = (uint32_t)((1.0 / this->reader_config.sample_rate) * 1000000); // convert to microseconds

        this->numSamplesPerChannel = 1;
    } else{
        if (this->checkNIError(ni::NiDAQmxInterface::CfgSampClkTiming(this->task_handle,
                                                                    this->reader_config.timing_source.c_str(),
                                                                    this->reader_config.sample_rate,
                                                                    DAQmx_Val_Rising,
                                                                    DAQmx_Val_ContSamps,
                                                                    this->reader_config.sample_rate))){
            LOG(ERROR) << "[NI Reader] failed while configuring timing for task " << this->reader_config.task_name;
            this->ok_state = false;
            return -1;
        }

        this->numSamplesPerChannel = std::floor(this->reader_config.sample_rate / this->reader_config.stream_rate);
    }
    this->bufferSize = this->numChannels * this->numSamplesPerChannel;
    return 0;
}

freighter::Error ni::DigitalReadSource::start(){
    freighter::Error err = freighter::NIL;
    if (this->checkNIError(ni::NiDAQmxInterface::StartTask(this->task_handle))){
        LOG(ERROR) << "[NI Reader] failed while starting reader for task " << this->reader_config.task_name;
        err = freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR);
    }
    else{
        LOG(INFO) << "[NI Reader] successfully started reader for task " << this->reader_config.task_name;
    }
    return err;
}


freighter::Error ni::DigitalReadSource::stop(){ 
    freighter::Error err = freighter::NIL;
    if (this->checkNIError(ni::NiDAQmxInterface::StopTask(this->task_handle))){
        LOG(ERROR) << "[NI Reader] failed while stopping reader for task " << this->reader_config.task_name;
        err = freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR);
    }
    else{
        if (this->checkNIError(ni::NiDAQmxInterface::ClearTask(this->task_handle))){
            LOG(ERROR) << "[NI Reader] failed while clearing reader for task " << this->reader_config.task_name;
            err = freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR);
        }
    }
    if (err == freighter::NIL){
        LOG(INFO) << "[NI Reader] successfully stopped and cleared reader for task " << this->reader_config.task_name;
    }
    return err;
}


std::pair<synnax::Frame, freighter::Error> ni::DigitalReadSource::read(){
    int32 samplesRead;
    char errBuff[2048] = {'\0'};
    uInt8 flushBuffer[10000]; // to flush buffer before performing a read
    uInt8 dataBuffer[10000]; // TODO fix this
    int32 flushRead;
    synnax::Frame f = synnax::Frame(numChannels);
    int32 numBytesPerSamp; // TODO do i need this?
    int err = 0;

    // initial read to flush buffer
    if (this->checkNIError(ni::NiDAQmxInterface::ReadDigitalLines(this->task_handle, // TODO: come back to and make sure this call to flush will be fine at any scale (elham)
                                                                  -1,               // reads all available samples in the buffer
                                                                  -1,
                                                                  DAQmx_Val_GroupByChannel,
                                                                  flushBuffer,
                                                                  1000,
                                                                  &samplesRead,
                                                                  &numBytesPerSamp,
                                                                  NULL))){
        LOG(ERROR) << "[NI Reader] failed while flushing buffer for task " << this->reader_config.task_name;
        return std::make_pair(std::move(f), freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR, "error reading digital data"));
    }

    // sleep if period is not 0
    if(this->reader_config.period != 0){
        std::this_thread::sleep_for(std::chrono::microseconds(this->reader_config.period));
    }

    // actual read to of digital lines
    std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
    if (this->checkNIError(ni::NiDAQmxInterface::ReadDigitalLines(this->task_handle,           // task handle
                                                                  this->numSamplesPerChannel, // numSampsPerChan
                                                                  -1,                         // timeout
                                                                  DAQmx_Val_GroupByChannel,   // dataLayout
                                                                  dataBuffer,                 // readArray
                                                                  10000,                      // arraySizeInSamps
                                                                  &samplesRead,               // sampsPerChanRead
                                                                  NULL,                       // numBytesPerSamp
                                                                  NULL))){
        LOG(ERROR) << "[NI Reader] failed while reading digital data for task " << this->reader_config.task_name;
        return std::make_pair(std::move(f), freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR, "error reading digital data"));
    }
    std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;

    // we interpolate the timestamps between the initial and final timestamp to ensure non-overlapping timestamps between read iterations
    uint64_t diff = final_timestamp - initial_timestamp;
    uint64_t incr = diff / this->numSamplesPerChannel;

    // Construct and populate index channel
    std::vector<std::uint64_t> time_index(numSamplesPerChannel);
    for (int i = 0; i < samplesRead; ++i){
        time_index[i] = initial_timestamp + (std::uint64_t)(incr * i);
    }
    
    // Construct and populate synnax frame
    std::vector<uint8_t> data_vec(samplesRead);
    uint64_t data_index = 0; // TODO: put a comment explaining the function of data_index
    for (int i = 0; i < numChannels; i++){
        if (this->reader_config.channels[i].channel_type == "index"){
            f.add(this->reader_config.channels[i].channel_key, synnax::Series(time_index, synnax::TIMESTAMP));
        }
        else{
            for (int j = 0; j < samplesRead; j++){
                data_vec[j] = dataBuffer[data_index * samplesRead + j];
            }
            f.add(this->reader_config.channels[i].channel_key, synnax::Series(data_vec));
            data_index++;
        }
    }

    // return synnax frame
    return std::make_pair(std::move(f), freighter::NIL);
}

bool ni::DigitalReadSource::ok(){ 
    return this->ok_state;
}

ni::DigitalReadSource::~DigitalReadSource(){
    this->stop();
    delete[] this->data;
}


int ni::DigitalReadSource::checkNIError(int32 error){
    if (error < 0){
        char errBuff[2048] = {'\0'};

        ni::NiDAQmxInterface::GetExtendedErrorInfo(errBuff, 2048);

        this->err_info["error type"] = "Vendor Error";
        this->err_info["error details"] = errBuff;
        
        this->ctx->setState({.task = this->reader_config.task_key,
                             .variant = "error",
                             .details = err_info});

        LOG(ERROR) << "[NI Reader] Vendor Error: " << this->err_info["error details"];
        this->ok_state = false;

        return -1;
    }
    return 0;
}



std::vector<synnax::ChannelKey> ni::DigitalReadSource::getChannelKeys(){
    std::vector<synnax::ChannelKey> keys;
    for (auto &channel : this->reader_config.channels){
        keys.push_back(channel.channel_key);
    }
    return keys;
}