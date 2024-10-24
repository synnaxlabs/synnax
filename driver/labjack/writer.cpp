// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/labjack/writer.h"
#include "driver/labjack/util.h"
#include <thread>

///////////////////////////////////////////////////////////////////////////////////
//                                    Helper                                     //
///////////////////////////////////////////////////////////////////////////////////
synnax::Series val_to_series(double val, synnax::DataType data_type){ // no discard

    if(data_type == synnax::FLOAT64)
        return synnax::Series(static_cast<double>(val), synnax::FLOAT64);
    if(data_type == synnax::FLOAT32)
        return synnax::Series(static_cast<float>(val), synnax::FLOAT32);
    if(data_type == synnax::INT32)
        return synnax::Series(static_cast<int32_t>(val), synnax::INT32);
    if(data_type == synnax::INT16)
        return synnax::Series(static_cast<int16_t>(val), synnax::INT16);
    if(data_type == synnax::INT8)
        return synnax::Series(static_cast<int8_t>(val), synnax::INT8);
    if(data_type == synnax::UINT32)
        return synnax::Series(static_cast<uint32_t>(val), synnax::UINT32);
    if(data_type == synnax::SY_UINT16)
        return synnax::Series(static_cast<uint16_t>(val), synnax::SY_UINT16);
    if(data_type == synnax::SY_UINT8)
        return synnax::Series(static_cast<uint8_t>(val), synnax::SY_UINT8);
    LOG(ERROR) << "[labjack.writer] Invalid data type";
}

double series_to_val(const synnax::Series &series){
    synnax::DataType data_type = series.data_type;
    if(data_type == synnax::FLOAT64)
        return static_cast<double>(series.values<double>()[0]);
    if(data_type == synnax::FLOAT32)
        return static_cast<double>(series.values<float>()[0]);
    if(data_type == synnax::INT32)
        return static_cast<double>(series.values<int32_t>()[0]);
    if(data_type == synnax::INT16)
        return static_cast<double>(series.values<int16_t>()[0]);
    if(data_type == synnax::INT8)
        return static_cast<double>(series.values<int8_t>()[0]);
    if(data_type == synnax::UINT32)
        return static_cast<double>(series.values<uint32_t>()[0]);
    if(data_type == synnax::SY_UINT16)
        return static_cast<double>(series.values<uint16_t>()[0]);
    if(data_type == synnax::SY_UINT8)
        return static_cast<double>(series.values<uint8_t>()[0]);
    LOG(ERROR) << "[labjack.writer] Invalid data type";
}

///////////////////////////////////////////////////////////////////////////////////
//                                    StateSource                                //
///////////////////////////////////////////////////////////////////////////////////
labjack::StateSource::StateSource(
    const synnax::Rate state_rate, // TODO: make this synnax::Rate?
    const synnax::ChannelKey &state_index_key,
    const std::map<synnax::ChannelKey, out_state> state_map
) : state_rate(state_rate),
    state_index_key(state_index_key),
    state_map(state_map){
    this->timer = loop::Timer(this->state_rate); // check if ic an move this to member initializer list
}


std::pair<synnax::Frame, freighter::Error> labjack::StateSource::read(
    breaker::Breaker &breaker){
    std::unique_lock<std::mutex> lock(this->state_mutex);
    // sleep for state period
    this->timer.wait(breaker);
    waiting_reader.wait_for(lock, this->state_rate.period().chrono());
    return std::make_pair(this->get_state(), freighter::NIL);
}

synnax::Frame labjack::StateSource::get_state(){
    // frame size = # monitored states + 1 index channel for all those states
    auto state_frame = synnax::Frame(this->state_map.size() + 1);
    state_frame.add(
        this->state_index_key,
        synnax::Series(
            synnax::TimeStamp::now().value,
            synnax::TIMESTAMP
        )
    );
    for(auto &[key, value]: this->state_map) {
        auto s = val_to_series(value.state, value.data_type);
        state_frame.add(
            value.state_key,
            std::move(s)
        );
    }

    return state_frame;
}

void labjack::StateSource::update_state(synnax::Frame frame){ // maybe just pass the key and value?
    std::unique_lock<std::mutex> lock(this->state_mutex);
    auto frame_index = 0;
    for (auto key: *(frame.channels)){
        if (key == this->state_index_key) continue;
        double value = series_to_val(frame.series->at(frame_index));
        this->state_map[key].state = value;
        frame_index++;
    }

    waiting_reader.notify_one();
}

///////////////////////////////////////////////////////////////////////////////////
//                                   WriteSink                                   //
///////////////////////////////////////////////////////////////////////////////////
labjack::WriteSink::WriteSink(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task,
        const labjack::WriterConfig &writer_config
    ) : ctx(ctx),
        task(task),
        writer_config(writer_config){

    auto breaker_config = breaker::Config{
        .name = task.name,
        .base_interval = 1 * SECOND,
        .max_retries = 20,
        .scale = 1.2,
    };

    this->breaker = breaker::Breaker(breaker_config);

    this->get_index_keys(); // retrieve state index from first state channel

    this->state_source = std::make_shared<labjack::StateSource>(
        this->writer_config.state_rate,
        this->writer_config.state_index_key,
        this->writer_config.initial_state_map
    );
}

labjack::WriteSink::~WriteSink(){
    this->stop("");
    CloseOrDie(this->handle);
}

void labjack::WriteSink::init(){
    if(this->writer_config.device_type == ""){
        auto [dev, err] = this->ctx->client->hardware.retrieveDevice(
                this->writer_config.device_key
        );
        if(err != freighter::NIL){
            LOG(ERROR) << "[labjack.writer] Error retrieving device: " << err.message();
            return;
        }
        this->writer_config.device_type = dev.model;
    }
    int err;
    {
        std::lock_guard<std::mutex> lock(labjack::device_mutex);
         err = LJM_Open(LJM_dtANY, LJM_ctANY, this->writer_config.serial_number.c_str(), &this->handle);
    }
    ErrorCheck(err, "[labjack.writer] LJM_Open error on serial num: %s ", this->writer_config.serial_number.c_str());
}

freighter::Error labjack::WriteSink::write(synnax::Frame frame){
    auto frame_index = 0;
    for(auto key: *(frame.channels)){
        double value = series_to_val(frame.series->at(frame_index));
        std::string loc = this->writer_config.initial_state_map[key].location;
        auto err = LJM_eWriteName(this->handle, loc.c_str(), value);
        frame_index++;
    }
    this->state_source->update_state(std::move(frame));
    return freighter::NIL;
}

freighter::Error labjack::WriteSink::stop(const std::string &cmd_key){
    CloseOrDie(this->handle);
    ctx->setState({
                          .task = task.key,
                          .key = cmd_key,
                          .variant = "success",
                          .details = {
                                  {"running", false},
                                  {"message", "Task stopped successfully"}
                          }
                  });
    return freighter::NIL;
}

freighter::Error labjack::WriteSink::start(const std::string &cmd_key){
    this->init();
    ctx->setState({
                          .task = task.key,
                          .key = cmd_key,
                          .variant = "success",
                          .details = {
                                  {"running", true},
                                  {"message", "Task started successfully"}
                          }
                  });
    return freighter::NIL;
}

std::vector<synnax::ChannelKey> labjack::WriteSink::get_cmd_channel_keys(){ // TODO: rename to get_cmd_keys
    std::vector<synnax::ChannelKey> keys;
    for (auto &channel: this->writer_config.channels){
        if(channel.enabled) keys.push_back(channel.cmd_key);
    }
    // Don't need index key as we're only using this for streaming cmds
    return keys;
}

std::vector<synnax::ChannelKey> labjack::WriteSink::get_state_channel_keys(){ // TODO: rename to get_state_keys
    std::vector<synnax::ChannelKey> keys;
    for(auto &channel: this->writer_config.channels){
        if(channel.enabled) keys.push_back(channel.state_key);
    }
    keys.push_back(this->writer_config.state_index_key);
    return keys;
}

void labjack::WriteSink::get_index_keys(){
    if(this->writer_config.channels.empty()){
        LOG(ERROR) << "[labjack.writer] No channels configured";
        return;
    }

    auto state_channel = this->writer_config.channels[0].state_key;
    auto [state_channel_info, err] = this->ctx->client->channels.retrieve(state_channel);
    if(err){
        LOG(ERROR) << "[labjack.writer] Failed to retrieve state channel: " << state_channel;
        return;
    }
    this->writer_config.state_index_key = state_channel_info.index;
}