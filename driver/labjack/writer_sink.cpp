// Copyright 2025 Synnax Labs, Inc.
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
#include <algorithm>

///////////////////////////////////////////////////////////////////////////////////
//                                    Helpers                                    //
///////////////////////////////////////////////////////////////////////////////////
telem::Series val_to_series(double val, telem::DataType data_type) {
    if (data_type == telem::FLOAT64_T)
        return telem::Series(static_cast<double>(val), telem::FLOAT64_T);
    if (data_type == telem::FLOAT32_T)
        return telem::Series(static_cast<float>(val), telem::FLOAT32_T);
    if (data_type == telem::INT32_T)
        return telem::Series(static_cast<int32_t>(val), telem::INT32_T);
    if (data_type == telem::INT16_T)
        return telem::Series(static_cast<int16_t>(val), telem::INT16_T);
    if (data_type == telem::INT8_T)
        return telem::Series(static_cast<int8_t>(val), telem::INT8_T);
    if (data_type == telem::UINT32_T)
        return telem::Series(static_cast<uint32_t>(val), telem::UINT32_T);
    if (data_type == telem::UINT16_T)
        return telem::Series(static_cast<uint16_t>(val), telem::UINT16_T);
    if (data_type == telem::UINT8_T)
        return telem::Series(static_cast<uint8_t>(val), telem::UINT8_T);
    LOG(ERROR) << "[labjack.writer] Invalid data type";
}

double series_to_val(const telem::Series &series) {
    telem::DataType data_type = series.data_type;
    if (data_type == telem::FLOAT64_T)
        return static_cast<double>(series.values<double>()[0]);
    if (data_type == telem::FLOAT32_T)
        return static_cast<double>(series.values<float>()[0]);
    if (data_type == telem::INT32_T)
        return static_cast<double>(series.values<int32_t>()[0]);
    if (data_type == telem::INT16_T)
        return static_cast<double>(series.values<int16_t>()[0]);
    if (data_type == telem::INT8_T)
        return static_cast<double>(series.values<int8_t>()[0]);
    if (data_type == telem::UINT32_T)
        return static_cast<double>(series.values<uint32_t>()[0]);
    if (data_type == telem::UINT16_T)
        return static_cast<double>(series.values<uint16_t>()[0]);
    if (data_type == telem::UINT8_T)
        return static_cast<double>(series.values<uint8_t>()[0]);
    LOG(ERROR) << "[labjack.writer] Invalid data type";
}

///////////////////////////////////////////////////////////////////////////////////
//                                    StateSource                                //
///////////////////////////////////////////////////////////////////////////////////
labjack::StateSource::StateSource(
    const telem::Rate state_rate,
    std::vector<synnax::ChannelKey> state_index_keys,
    const std::map<synnax::ChannelKey, out_state> state_map
) : state_rate(state_rate),
    state_map(state_map),
    state_index_keys(state_index_keys) {
    this->timer = loop::Timer(this->state_rate);
}

std::pair<synnax::Frame, xerrors::Error> labjack::StateSource::read(
    breaker::Breaker &breaker) {
    std::unique_lock<std::mutex> lock(this->state_mutex);
    this->timer.wait(breaker); // sleep for state period
    waiting_reader.wait_for(lock, this->state_rate.period().chrono());
    return std::make_pair(this->get_state(), xerrors::NIL);
}

synnax::Frame labjack::StateSource::get_state() {
    // frame size = # monitored states + # index channels for the states
    auto state_frame = synnax::Frame(
        this->state_map.size() + this->state_index_keys.size());

    for (auto key: this->state_index_keys) {
        auto t = telem::Series(telem::TimeStamp::now().value, telem::TIMESTAMP_T);
        state_frame.emplace(key, std::move(t));
    }
    for (auto &[key, value]: this->state_map) {
        auto s = val_to_series(value.state, value.data_type);
        state_frame.emplace(value.state_key, std::move(s));
    }

    return state_frame;
}

void labjack::StateSource::update_state(const synnax::Frame &frame) {
    std::unique_lock<std::mutex> lock(this->state_mutex);
    auto frame_index = 0;
    for (auto key: *(frame.channels)) {
        if (std::find(
                state_index_keys.begin(),
                state_index_keys.end(),
                key
            ) != state_index_keys.end())
            continue;

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
    const labjack::WriterConfig &writer_config,
    std::shared_ptr<labjack::DeviceManager> device_manager
) : ctx(ctx),
    task(task),
    writer_config(writer_config),
    device_manager(device_manager),
    breaker(breaker::default_config(task.name)) {

    auto state_index_keys = this->get_index_keys();
    // retrieve state index from first state channel

    this->state_source = std::make_shared<labjack::StateSource>(
        this->writer_config.state_rate,
        state_index_keys,
        this->writer_config.initial_state_map
    );

    this->handle = this->device_manager->get_device_handle(
        this->writer_config.serial_number);

    if (this->writer_config.channels.empty())
        this->log_err("No channels enabled/set");
}

labjack::WriteSink::~WriteSink() {
    this->stop("");
}


void labjack::WriteSink::init() {
    if (this->writer_config.device_type == "") {
        auto [dev, err] = this->ctx->client->hardware.retrieve_device(
            this->writer_config.device_key
        );
        if (err != xerrors::NIL) {
            this->log_err("Error retrieving device.");
            return;
        }
        this->writer_config.device_type = dev.model;
    }
    // Set all DO channels to low because LabJack devices factory default is for DIO to be high
    for (auto &channel: this->writer_config.channels) {
        if (channel.enabled && channel.channel_type == "DO") {
            check_err(
                LJM_eWriteName(
                    this->handle,
                    channel.location.c_str(),
                    0
                ), "init.LJM_EWRITENAME"
            );
        }
    }
}

xerrors::Error labjack::WriteSink::write(const synnax::Frame &frame) {
    auto frame_index = 0;
    for (auto key: *(frame.channels)) {
        double value = series_to_val(frame.series->at(frame_index));
        std::string loc = this->writer_config.initial_state_map[key].location;
        check_err(
            LJM_eWriteName(
                this->handle,
                loc.c_str(),
                value
            ), "write.LJM_EWRITENAME"
        );
        frame_index++;
    }
    this->state_source->update_state(std::move(frame));
    return xerrors::NIL;
}


xerrors::Error labjack::WriteSink::stop(const std::string &cmd_key) {
    if (!this->ok())
        return xerrors::Error(
            "Device disconnected or is in error. Please reconfigure task and try again");
    ctx->set_state({
        .task = task.key,
        .key = cmd_key,
        .variant = "success",
        .details = {
            {"running", false},
            {"message", "Task stopped successfully"}
        }
    });
    return xerrors::NIL;
}

xerrors::Error labjack::WriteSink::start(const std::string &cmd_key) {
    this->init();
    ctx->set_state({
        .task = task.key,
        .key = cmd_key,
        .variant = "success",
        .details = {
            {"running", true},
            {"message", "Task started successfully"}
        }
    });
    return xerrors::NIL;
}

std::vector<synnax::ChannelKey> labjack::WriteSink::get_cmd_channel_keys() {
    std::vector<synnax::ChannelKey> keys;
    for (auto &channel: this->writer_config.channels) {
        if (channel.enabled) keys.push_back(channel.cmd_key);
    }
    // Don't need index key as we're only using this for streaming cmds
    return keys;
}

std::vector<synnax::ChannelKey> labjack::WriteSink::get_state_channel_keys() {
    std::vector<synnax::ChannelKey> keys;
    for (auto &channel: this->writer_config.channels) {
        if (channel.enabled) keys.push_back(channel.state_key);
    }
    for (auto &channel: this->writer_config.state_index_keys) {
        keys.push_back(channel);
    }
    return keys;
}

std::vector<synnax::ChannelKey> labjack::WriteSink::get_index_keys() {
    if (this->writer_config.channels.empty()) {
        return {};
    }

    std::set<synnax::ChannelKey> unique_keys;
    for (auto &channel: this->writer_config.channels) {
        auto [channel_info, err] = this->ctx->client->channels.retrieve(
            channel.state_key);
        if (err) {
            this->log_err("Failed to retrieve channel for port: " + channel.location);
            return {};
        }
        unique_keys.insert(channel_info.index);
    }

    this->writer_config.state_index_keys = {unique_keys.begin(), unique_keys.end()};
    return this->writer_config.state_index_keys;
}

int labjack::WriteSink::check_err(int err, std::string caller) {
    labjack::check_err_internal(
        err,
        caller,
        "writer",
        this->ctx,
        this->ok_state,
        this->task.key
    );
    if (err == LJME_RECONNECT_FAILED) {
        this->device_manager->close_device(this->writer_config.serial_number);
    }
    return err;
}

bool labjack::WriteSink::ok() {
    return this->ok_state;
}

void labjack::WriteSink::log_err(std::string msg) {
    LOG(ERROR) << "[labjack.writer] " << msg;
    this->ok_state = false;
    ctx->set_state({
        .task = this->task.key,
        .variant = "error",
        .details = {
            {"running", false},
            {"message", msg}
        }
    });
}
