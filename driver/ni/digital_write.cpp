// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <utility>
#include <chrono>
#include <stdio.h>
#include <cassert>

#include "client/cpp/telem/telem.h"
#include "driver/ni/ni.h"

#include "nlohmann/json.hpp"
#include "glog/logging.h"

///////////////////////////////////////////////////////////////////////////////////
//                             Helper Functions                                  //
///////////////////////////////////////////////////////////////////////////////////
void ni::DigitalWriteSink::getIndexKeys() {
    assert(this->writer_config.drive_state_channel_keys.size() > 0);
    auto state_channel = this->writer_config.drive_state_channel_keys[0];
    auto [state_channel_info, err] = this->ctx->client->channels.
            retrieve(state_channel);
    if (err != freighter::NIL) {
        LOG(ERROR) << "[ni.writer] failed to retrieve channel " << state_channel;
        this->ok_state = false;
        return;
    } else {
        this->writer_config.drive_state_index_key = state_channel_info.index;
    }
}

///////////////////////////////////////////////////////////////////////////////////
//                                    daqWriter                                  //
///////////////////////////////////////////////////////////////////////////////////
ni::DigitalWriteSink::DigitalWriteSink(
    TaskHandle task_handle,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task task)
    : task_handle(task_handle),
      ctx(ctx) {
    // print task handle
    LOG(INFO) << "Task handle: " << task_handle;

    // Create parser
    auto config_parser = config::Parser(task.config);
    this->writer_config.task_name = task.name;

    // Parse configuration and make sure it is valid
    this->parseConfig(config_parser);
    if (!config_parser.ok()) {
        // Log error
        LOG(ERROR) << "[ni.writer] failed to parse configuration for " << this->
                writer_config.task_name;
        this->ctx->setState({
            .task = this->writer_config.task_key,
            .variant = "error",
            .details = config_parser.error_json()
        });
        this->ok_state = false;
        return;
    }
    LOG(INFO) << "[ni.writer] successfully parsed configuration for " << this->
            writer_config.task_name;

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
    if (this->init()) {
        LOG(ERROR) << "[ni.writer] Failed while configuring NI hardware for task " <<
                this->writer_config.task_name;
        this->ok_state = false;
    }

    // Get index keys
    this->getIndexKeys();

    // TODO: get device proprties for things like authentication
    this->writer_state_source = std::make_shared<ni::StateSource>(
        this->writer_config.state_rate,
        this->writer_config.drive_state_index_key,
        this->writer_config.drive_state_channel_keys);

    this->start();
}


void ni::DigitalWriteSink::parseConfig(config::Parser &parser) {
    this->writer_config.state_rate = parser.required<uint64_t>("state_rate");
    // for state writing
    this->writer_config.device_key = parser.required<std::string>("device");
    // device key

    assert(parser.ok());
    auto [dev, err] = this->ctx->client->hardware.retrieveDevice(
        this->writer_config.device_key);

    if (err != freighter::NIL) {
        LOG(ERROR) << "[ni.writer] failed to retrieve device with key " << this->
                writer_config.device_key;
        this->ok_state = false;
        return;
    }
    this->writer_config.device_name = dev.location;

    // task key
    // device name
    parser.iter("channels",
        [&](config::Parser &channel_builder) {

            ni::ChannelConfig config;
            // digital channel names are formatted: <device_name>/port<port_number>/line<line_number>
            std::string port = "port" + std::to_string(
                channel_builder.required<std::uint64_t>("port"));
            std::string line = "line" + std::to_string(
                channel_builder.required<std::uint64_t>("line"));
                
            config.name = (this->writer_config.device_name + "/" + port + "/" + line);

            config.channel_key = channel_builder.required<uint32_t>(
                "cmd_channel");
            this->writer_config.drive_cmd_channel_keys.push_back(
                config.channel_key);

            uint32_t drive_state_key = channel_builder.required<uint32_t>(
                "state_channel");
            this->writer_config.drive_state_channel_keys.push_back(
                drive_state_key);

            // TODO: there could be more than 2 state

            this->writer_config.channels.push_back(config);
        });

    assert(this->writer_config.drive_state_channel_keys.size() > 0);
    assert(this->writer_config.drive_cmd_channel_keys.size() > 0);
    assert(
        this->writer_config.drive_cmd_channel_keys.size() == this->writer_config.
        drive_state_channel_keys.size());
}


int ni::DigitalWriteSink::init() {
    int err = 0;
    auto channels = this->writer_config.channels;

    // iterate through channels
    for (auto &channel: channels) {
        if (channel.channel_type != "index") {
            err = this->checkNIError(ni::NiDAQmxInterface::CreateDOChan(
                this->task_handle, channel.name.c_str(), "",
                DAQmx_Val_ChanPerLine));
        }
        this->numChannels++;
        // includes index channels TODO: how is this different form jsut channels.size()?
        if (err < 0) {
            LOG(ERROR) << "[ni.writer] failed while configuring channel " << channel.
                    name;
            return -1;
        }
    }

    // Configure buffer size and read resources
    this->bufferSize = this->numChannels;
    this->writeBuffer = new uint8_t[this->bufferSize];

    for (int i = 0; i < this->bufferSize; i++) {
        writeBuffer[i] = 0;
    }

    LOG(INFO) << "[ni.writer] successfully configured NI hardware for task " << this->
            writer_config.task_name;
    return 0;
}

freighter::Error ni::DigitalWriteSink::start() {
    if (this->running.exchange(true)) {
        return freighter::NIL;
    }
    if (this->checkNIError(ni::NiDAQmxInterface::StartTask(this->task_handle))) {
        LOG(ERROR) << "[ni.writer] failed while starting writer for task " << this->
                writer_config.task_name;
        return freighter::Error(driver::CRITICAL_HARDWARE_ERROR);
    }
    LOG(INFO) << "[ni.writer] successfully started writer for task " << this->
            writer_config.task_name;
    ctx->setState({
        .task = this->writer_config.task_key,
        .variant = "success",
        .details = {
            {"running", true}
        }
    });
    return freighter::NIL;
}


freighter::Error ni::DigitalWriteSink::stop() {
    if (!this->running.exchange(false)) {
        return freighter::NIL;
    }
    if (this->checkNIError(ni::NiDAQmxInterface::StopTask(task_handle))) {
        LOG(ERROR) << "[ni.writer] failed while stopping writer for task " << this->
                writer_config.task_name;
        return freighter::Error(driver::CRITICAL_HARDWARE_ERROR);
    }
    LOG(INFO) << "[ni.writer] successfully stopped writer for task " << this->
            writer_config.task_name;
    ctx->setState({
        .task = this->writer_config.task_key,
        .variant = "success",
        .details = {
            {"running", false}
        }
    });
    return freighter::NIL;
}

freighter::Error ni::DigitalWriteSink::write(synnax::Frame frame) {
    int32 samplesWritten = 0;
    formatData(std::move(frame));

    // Write digital data
    if (this->checkNIError(ni::NiDAQmxInterface::WriteDigitalLines(this->task_handle,
        1, // number of samples per channel
        1, // auto start
        10.0, // timeout
        DAQmx_Val_GroupByChannel, // data layout
        writeBuffer, // data
        &samplesWritten, // samples written
        NULL))) {
        LOG(ERROR) << "[ni.writer] failed while writing digital data for task " << this
                ->writer_config.task_name;
        return freighter::Error(driver::CRITICAL_HARDWARE_ERROR,
                                "Error reading digital data");
    }
    this->writer_state_source->updateState(this->writer_config.modified_state_keys,
                                           this->writer_config.modified_state_values);

    return freighter::NIL;
}


freighter::Error ni::DigitalWriteSink::formatData(synnax::Frame frame) {
    uint32_t frame_index = 0;
    uint32_t cmd_channel_index = 0;

    for (auto key: *(frame.channels)) {
        // the order the keys were pushed into the vector is the order the data is written
        // first see if the key is in the drive_cmd_channel_keys
        auto it = std::find(this->writer_config.drive_cmd_channel_keys.begin(),
                            this->writer_config.drive_cmd_channel_keys.end(), key);
        if (it != this->writer_config.drive_cmd_channel_keys.end()) {
            // if so, now find which index it is in the vector (i.e. which channel it is in the writeBuffer)
            cmd_channel_index = std::distance(
                this->writer_config.drive_cmd_channel_keys.begin(),
                it);
            // this corressponds to where in the order its NI channel was created
            // now we grab the level we'd like to write and put it into that location in the write_buffer
            auto series = frame.series->at(frame_index).uint8();
            writeBuffer[cmd_channel_index] = series[0];
            this->writer_config.modified_state_keys.push(
                this->writer_config.drive_state_channel_keys[cmd_channel_index]);
            this->writer_config.modified_state_values.push(series[0]);
        }
        frame_index++;
    }
    return freighter::NIL;
}


int ni::DigitalWriteSink::checkNIError(int32 error) {
    if (error < 0) {
        char errBuff[2048] = {'\0'};
        ni::NiDAQmxInterface::GetExtendedErrorInfo(errBuff, 2048);
        this->err_info["error type"] = "Vendor Error";
        this->err_info["error details"] = errBuff;
        this->ok_state = false;
        this->ctx->setState({
            .task = this->writer_config.task_key,
            .variant = "error",
            .details = err_info
        });
        LOG(ERROR) << "[NI Reader] Vendor Error: " << this->err_info["error details"];
        return -1;
    }
    return 0;
}


bool ni::DigitalWriteSink::ok() {
    return this->ok_state;
}

ni::DigitalWriteSink::~DigitalWriteSink() {
    freighter::Error err = freighter::NIL;
    if (this->checkNIError(ni::NiDAQmxInterface::ClearTask(task_handle))) {
        LOG(ERROR) << "[ni.writer] failed while clearing writer for task " << this->
                writer_config.task_name;
    }
    delete[] this->writeBuffer;
    LOG(INFO) << "[ni.writer] successfully cleared writer for task " << this->
            writer_config.task_name;
}


std::vector<synnax::ChannelKey> ni::DigitalWriteSink::getCmdChannelKeys() {
    std::vector<synnax::ChannelKey> keys;
    for (auto &channel: this->writer_config.channels) {
        if (channel.channel_type != "index") {
            keys.push_back(
                channel.channel_key);
            // could either be the key to a cmd channel or a key to an cmd index channel
        }
    }
    return keys;
}

std::vector<synnax::ChannelKey> ni::DigitalWriteSink::getStateChannelKeys() {
    std::vector<synnax::ChannelKey> keys = this->writer_config.drive_state_channel_keys;
    keys.push_back(this->writer_config.drive_state_index_key);
    return keys;
}

///////////////////////////////////////////////////////////////////////////////////
//                                    StateSource                                //
///////////////////////////////////////////////////////////////////////////////////
ni::StateSource::StateSource(std::uint64_t state_rate,
                             synnax::ChannelKey &drive_state_index_key,
                             std::vector<synnax::ChannelKey> &drive_state_channel_keys)
    : state_rate(state_rate) {
    // start the periodic thread
    this->state_period = std::chrono::duration<double>(1.0 / this->state_rate);
    this->drive_state_index_key = drive_state_index_key;

    // initialize all states to 0 (logic low)
    for (auto &key: drive_state_channel_keys) {
        this->state_map[key] = 0;
    }
}

std::pair<synnax::Frame, freighter::Error> ni::StateSource::read(
    breaker::Breaker &breaker) {
    std::unique_lock<std::mutex> lock(this->state_mutex);
    waiting_reader.wait_for(lock, state_period);
    return std::make_pair(this->getDriveState(), freighter::NIL);
}


//TODO: do i need this?
freighter::Error ni::StateSource::start() {
    return freighter::NIL;
}

// TODO: do i need this?
freighter::Error ni::StateSource::stop() {
    return freighter::NIL;
}

synnax::Frame ni::StateSource::getDriveState() {
    auto drive_state_frame = synnax::Frame(this->state_map.size() + 1);
    drive_state_frame.add(this->drive_state_index_key,
                          synnax::Series(
                              std::vector<uint64_t>{synnax::TimeStamp::now().value},
                              synnax::TIMESTAMP));

    // Iterate through map and add each state to frame
    for (auto &state: this->state_map) {
        drive_state_frame.add(state.first,
                              synnax::Series(std::vector<uint8_t>{state.second}));
    }

    return drive_state_frame;
}

void ni::StateSource::updateState(std::queue<synnax::ChannelKey> &modified_state_keys,
                                  std::queue<std::uint8_t> &modified_state_values) {
    std::unique_lock<std::mutex> lock(this->state_mutex);
    // update state map
    while (!modified_state_keys.empty()) {
        this->state_map[modified_state_keys.front()] = modified_state_values.front();
        modified_state_keys.pop();
        modified_state_values.pop();
    }

    waiting_reader.notify_one();
}
