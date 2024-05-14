// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//
// Created by Elham Islam on 5/14/24.
//

#include "driver/ni/ni.h"
#include "nlohmann/json.hpp"
#include "client/cpp/telem/telem.h"
#include <utility>
#include <chrono>
#include <stdio.h>
#include <cassert>
#include "glog/logging.h"


///////////////////////////////////////////////////////////////////////////////////
//                             Helper Functions                                  //
///////////////////////////////////////////////////////////////////////////////////
void ni::DaqAnalogReader::getIndexKeys(){
    std::set<std::uint32_t> index_keys;
    //iterate through channels in reader config
    for (auto &channel : this->writer_config.channels){
        auto [channel_info, err] = this->ctx->client->channels.retrieve(channel.channel_key);
        // TODO handle error with breaker
        if (err != freighter::NIL){
            // Log error
            LOG(ERROR) << "[NI Writer] failed to retrieve channel " << channel.channel_key;
            this->ok_state = false;
            return;
        } else{
            // add key to set
            index_keys.insert(channel_info.index);
        }
    }


    // now iterate through the set and add all the index channels as configs
    for (auto it = index_keys.begin(); it != index_keys.end(); ++it){
        auto index_key = *it;
        LOG(INFO) << "constructing index channel configs";
        auto [channel_info, err] = this->ctx->client->channels.retrieve(index_key);
        if (err != freighter::NIL){
            LOG(ERROR) << "[NI Writer] failed to retrieve channel " << index_key;
            this->ok_state = false;
            return;
        } else{
            ni::ChannelConfig index_channel;
            index_channel.channel_key = channel_info.key;
            index_channel.channel_type = "index";
            index_channel.name = channel_info.name;
            this->reader_config.channels.push_back(index_channel);
            LOG(INFO) << "[NI Writer] index channel " << index_channel.channel_key << " and name: " << index_channel.name <<" added to task " << this->reader_config.task_name;
        }
    }

}

///////////////////////////////////////////////////////////////////////////////////
//                                    daqWriter                                //
///////////////////////////////////////////////////////////////////////////////////



ni::DaqDigitalWriter::DaqDigitalWriter(
    TaskHandle taskHandle,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task task)
    : taskHandle(taskHandle),
      ctx(ctx){
    
    // Create parser
    auto config_parser = config::Parser(task.config);
    this->writer_config.task_name = task.name;

    // Parse configuration and make sure it is valid
    this->parseDigitalWriterConfig(config_parser);
    if (!config_parser.ok())
    {
        // Log error
        LOG(ERROR) << "[NI Writer] failed to parse configuration for " << this->writer_config.task_name;
        this->ctx->setState({.task = task.key,
                             .variant = "error",
                             .details = config_parser.error_json()});
        this->ok_state = false;
        return;
    }
    LOG(INFO) << "[NI Writer] successfully parsed configuration for " << this->writer_config.task_name;

    // TODO: get device proprties for things like authentication
    this->writer_state_source = std::make_unique<ni::daqStateWriter>(this->writer_config.state_rate,
                                                                     this->writer_config.drive_state_index_key,
                                                                     this->writer_config.drive_state_channel_keys);

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
    if (this->init())
    {
        LOG(ERROR) << "[NI Writer] Failed while configuring NI hardware for task " << this->writer_config.task_name;
        this->ok_state = false;
    }
    this->start();
}




void ni::DaqDigitalWriter::parseConfig(config::Parser &parser){
    this->writer_config.state_rate = parser.required<uint64_t>("stream_rate"); // for state writing
    this->writer_config.device_key = parser.required<uint32_t>("device"); // device key

    auto [dev, err ] = this->ctx->client->hardware.retrieveDevice(this->writer_config.device_key);

    if(err != freighter::NIL){
        LOG(ERROR) << "[NI Writer] failed to retrieve device with key " << this->writer_config.device_key;
        this->ok_state = false;
        return;
    }
    this->reader_config.device_name = dev.location;

    // task key 
    // device name
    this->writer_config.device_name = parser.required<std::string>("device_name");
    // now parse the channels
    parser.iter("channels",
                [&](config::Parser &channel_builder)
                {
                    ni::ChannelConfig config;

                    // digital channel names are formatted: <device_name>/port<port_number>/line<line_number>
                    config.name = (this->writer_config.device_name + "/port" + std::to_string(channel_builder.required<std::uint64_t>("port")) + "/line" + std::to_string(channel_builder.required<std::uint64_t>("line")));

                    config.channel_key = channel_builder.required<uint32_t>("cmd_channel");
              
                    uint32_t drive_state_key = channel_builder.required<uint32_t>("state_channel");
                    this->writer_config.drive_state_channel_keys.push_back(drive_state_key);
                    this->writer_config.drive_cmd_channel_keys.push_back(config.channel_key);
                
                    // TODO: there could be more than 2 state
                    config.min_val = 0;
                    config.max_val = 1;

                    this->writer_config.channels.push_back(config);
                });

    assert(this->writer_config.drive_state_index_key != 0);
    assert(this->writer_config.drive_state_channel_keys.size() > 0);
    assert(this->writer_config.drive_cmd_channel_keys.size() > 0);
    assert(this->writer_config.drive_cmd_channel_keys.size() == this->writer_config.drive_state_channel_keys.size());
}


int ni::DaqDigitalWriter::init(){
    int err = 0;
    auto channels = this->writer_config.channels;

    // iterate through channels
    for (auto &channel : channels){
        if (channel.channel_type != "index"){
            err = this->checkNIError(ni::NiDAQmxInterface::CreateDOChan(taskHandle, channel.name.c_str(), "", DAQmx_Val_ChanPerLine));
        }
        this->numChannels++; // includes index channels TODO: how is this different form jsut channels.size()?
        if (err < 0){
            LOG(ERROR) << "[NI Writer] failed while configuring channel " << channel.name;
            return -1;
        }
    }

    // Configure buffer size and read resources
    this->bufferSize = this->numChannels;
    this->writeBuffer = new uint8_t[this->bufferSize];

    for (int i = 0; i < this->bufferSize; i++){
        writeBuffer[i] = 0;
    }

    LOG(INFO) << "[NI Writer] successfully configured NI hardware for task " << this->writer_config.task_name;
    return 0;
}

freighter::Error ni::DaqDigitalWriter::start(){
    if(this->running){
        LOG(INFO) << "[NI Reader] attempt to start an already running NI task for task " << this->reader_config.task_name;
        return freighter::NIL; // TODO: change return value?
    }
    freighter::Error err = freighter::NIL;
    if (this->checkNIError(ni::NiDAQmxInterface::StartTask(this->taskHandle))){
        LOG(ERROR) << "[NI Writer] failed while starting writer for task " << this->writer_config.task_name;
        err = freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR);
    }
    else{
        LOG(INFO) << "[NI Writer] successfully started writer for task " << this->writer_config.task_name;
    }
    return err;
}


freighter::Error ni::daqWriter::stop(){
   if(!this->running){
        LOG(INFO) << "[NI Reader] attempt to stop an already stopped NI task for task " << this->reader_config.task_name;
        return freighter::NIL; // TODO: change return value?
    }

    freighter::Error err = freighter::NIL;

    if (this->checkNIError(ni::NiDAQmxInterface::StopTask(taskHandle))){
        LOG(ERROR) << "[NI Writer] failed while stopping writer for task " << this->writer_config.task_name;
        err = freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR);
    }
    else{
        if (this->checkNIError(ni::NiDAQmxInterface::ClearTask(taskHandle))){
            LOG(ERROR) << "[NI Writer] failed while clearing writer for task " << this->writer_config.task_name;
            err = freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR);
        }
    }

    delete[] writeBuffer;

    if (err == freighter::NIL){
        LOG(INFO) << "[NI Writer] successfully stopped and cleared writer for task " << this->writer_config.task_name;
    }

    return err;
}

freighter::Error ni::DaqDigitalWriter::write(synnax::Frame frame){
    char errBuff[2048] = {'\0'};
    int32 samplesWritten = 0;
    formatData(std::move(frame));

    // Write digital data
    if (this->checkNIError(ni::NiDAQmxInterface::WriteDigitalLines(this->taskHandle,
                                                                   1,                        // number of samples per channel
                                                                   1,                        // auto start
                                                                   10.0,                     // timeout
                                                                   DAQmx_Val_GroupByChannel, // data layout
                                                                   writeBuffer,              // data
                                                                   &samplesWritten,          // samples written
                                                                   NULL)))
    {
        LOG(ERROR) << "[NI Writer] failed while writing digital data for task " << this->writer_config.task_name;
        return freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR, "Error reading digital data");
    }

    // Construct drive state frame (can only do this after a successful write to keep consistent over failed writes)

    // return acknowledgements frame to write to the ack channel
    this->writer_state_source->updateState(this->writer_config.modified_state_keys, this->writer_config.modified_state_values);

    return freighter::NIL;
}


freighter::Error ni::DaqDigitalWriter::formatData(synnax::Frame frame){
    uint32_t frame_index = 0;
    uint32_t cmd_channel_index = 0;

    for (auto key : *(frame.channels)){ // the order the keys were pushed into the vector is the order the data is written
        // first see if the key is in the drive_cmd_channel_keys
        auto it = std::find(this->writer_config.drive_cmd_channel_keys.begin(), this->writer_config.drive_cmd_channel_keys.end(), key);
        if (it != this->writer_config.drive_cmd_channel_keys.end()){
            // if so, now find which index it is in the vector (i.e. which channel it is in the writeBuffer)
            cmd_channel_index = std::distance(this->writer_config.drive_cmd_channel_keys.begin(), it); // this corressponds to where in the order its NI channel was created
            // now we grab the level we'd like to write and put it into that location in the write_buffer
            auto series = frame.series->at(frame_index).uint8();
            writeBuffer[cmd_channel_index] = series[0];
            this->writer_config.modified_state_keys.push(this->writer_config.drive_state_channel_keys[cmd_channel_index]);
            this->writer_config.modified_state_values.push(series[0]);
        }
        frame_index++;
    }
    return freighter::NIL;
}

////////////////////////////////////////////////////////////////////////////////


int ni::daqWriter::checkNIError(int32 error)
{
    if (error < 0)
    {
        char errBuff[2048] = {'\0'};
        ni::NiDAQmxInterface::GetExtendedErrorInfo(errBuff, 2048);
        this->err_info["error type"] = "Vendor Error";
        this->err_info["error details"] = errBuff;
        this->ok_state = false;
        this->ctx->setState({.task = this->writer_config.task_key,
                             .variant = "error",
                             .details = err_info});
        LOG(ERROR) << "[NI Reader] Vendor Error: " << this->err_info["error details"];
        return -1;
    }
    return 0;
}

bool ni::daqWriter::ok()
{
    return this->ok_state;
}

ni::daqWriter::~daqWriter()
{
    LOG(INFO) << "Destroying daqWriter";
    this->stop();
}

std::vector<synnax::ChannelKey> ni::daqWriter::getCmdChannelKeys()
{
    std::vector<synnax::ChannelKey> keys;
    for (auto &channel : this->writer_config.channels)
    {
        if (channel.channel_type != "index" && channel.channel_type != "driveStateIndex")
        {
            keys.push_back(channel.channel_key); // could either be the key to a cmd channel or a key to an cmd index channel
        }
    }
    return keys;
}

std::vector<synnax::ChannelKey> ni::daqWriter::getStateChannelKeys()
{
    std::vector<synnax::ChannelKey> keys = this->writer_config.drive_state_channel_keys;
    keys.push_back(this->writer_config.drive_state_index_key);
    return keys;
}

///////////////////////////////////////////////////////////////////////////////////
//                                    daqStateWriter                           //
///////////////////////////////////////////////////////////////////////////////////

ni::daqStateWriter::daqStateWriter(std::uint64_t state_rate, synnax::ChannelKey &drive_state_index_key, std::vector<synnax::ChannelKey> &drive_state_channel_keys)
    : state_rate(state_rate)
{
    // start the periodic thread
    this->state_period = std::chrono::duration<double>(1.0 / this->state_rate);
    this->drive_state_index_key = drive_state_index_key;

    // initialize all states to 0 (logic low)
    for (auto &key : drive_state_channel_keys)
    {
        this->state_map[key] = 0;
    }
}

std::pair<synnax::Frame, freighter::Error> ni::daqStateWriter::read()
{
    std::unique_lock<std::mutex> lock(this->state_mutex);
    waitingReader.wait_for(lock, state_period); // TODO: double check this time is relative and not absolute
    return std::make_pair(std::move(this->getDriveState()), freighter::NIL);
}

freighter::Error ni::daqStateWriter::start()
{
    return freighter::NIL;
}

freighter::Error ni::daqStateWriter::stop()
{
    return freighter::NIL;
}

synnax::Frame ni::daqStateWriter::getDriveState()
{
    auto drive_state_frame = synnax::Frame(this->state_map.size() + 1);
    drive_state_frame.add(this->drive_state_index_key, synnax::Series(std::vector<uint64_t>{synnax::TimeStamp::now().value}, synnax::TIMESTAMP));

    // Iterate through map and add each state to frame
    for (auto &state : this->state_map)
    {
        drive_state_frame.add(state.first, synnax::Series(std::vector<uint8_t>{state.second}));
    }

    return std::move(drive_state_frame);
}

void ni::daqStateWriter::updateState(std::queue<synnax::ChannelKey> &modified_state_keys, std::queue<std::uint8_t> &modified_state_values)
{
    // LOG(INFO) << "Updating state";
    std::unique_lock<std::mutex> lock(this->state_mutex);
    // update state map
    while (!modified_state_keys.empty())
    {
        this->state_map[modified_state_keys.front()] = modified_state_values.front();
        modified_state_keys.pop();
        modified_state_values.pop();
    }

    waitingReader.notify_one();
}

// TODO create a helper function that takes in a frame and formats into the data to pass into writedigital
// TODO: create a helper function to parse digital data configuration of wehtehr its a port to r line
