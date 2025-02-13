// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/ni/ni.h"
#include "driver/ni/reader.h"
#include "driver/ni/scale.h"
#include "driver/ni/util.h"
#include <map>
#include <regex>


///////////////////////////////////////////////////////////////////////////////////
//                                    NiSource                                   //
///////////////////////////////////////////////////////////////////////////////////
void ni::Source::get_index_keys() {
    std::set<std::uint32_t> index_keys;
    for (auto &channel: this->reader_config.channels) {
        auto [channel_info, err] = this->ctx->client->channels.retrieve(
            channel.key);
        if (err)
            return this->log_error(
                "failed to retrieve channel " + std::to_string(channel.key));
        index_keys.insert(channel_info.index);
    }
    for (auto &index_key: index_keys) {
        auto [channel_info, err] = this->ctx->client->channels.retrieve(index_key);
        if (err)
            return this->log_error(
                "failed to retrieve channel " + std::to_string(index_key));
        this->reader_config.channels.emplace_back(ni::ReaderChannelConfig{
            .key = channel_info.key,
            .name = channel_info.name,
            .type = "index"
        });
    }
}

ni::Source::Source(
    const std::shared_ptr<DAQmx> &dmx,
    TaskHandle task_handle,
    const std::shared_ptr<task::Context> &ctx,
    synnax::Task task
) : dmx(dmx),
    task_handle(task_handle),
    ctx(ctx),
    task(task),
    err_info({}) {
}

void ni::Source::parse_config(config::Parser &parser) {
    this->reader_config.sample_rate.value = parser.required<uint64_t>("sample_rate");
    this->reader_config.stream_rate.value = parser.required<uint64_t>("stream_rate");
    this->reader_config.device_key = parser.optional<std::string>(
        "device", "cross-device");
    if (this->reader_config.device_key == "")
        this->reader_config.device_key = "cross-device";
    this->reader_config.timing_source = "none";
    if (this->reader_config.device_key != "cross-device") {
        auto [dev, err] = this->ctx->client->hardware.retrieve_device(
            this->reader_config.device_key);
        if (err)
            return this->log_error(
                "failed to retrieve device " + this->reader_config.device_name);
        this->reader_config.device_name = dev.location;
    }
    this->parse_channels(parser);
}

int ni::Source::init() {
    auto config_parser = config::Parser(this->task.config);
    this->reader_config.task_name = this->task.name;
    this->reader_config.task_key = this->task.key;
    this->parse_config(config_parser);
    if (!config_parser.ok()) {
        this->log_error(
            "failed to parse configuration for " + this->reader_config.task_name +
            " Parser Error: " +
            config_parser.error_json().dump());
        this->ctx->set_state({
            .task = task.key,
            .variant = "error",
            .details = {
                {"running", false},
                {"message", "Failed to parse configuration for " + this->reader_config.task_name}
            }
        });
        return -1;
    }
    this->get_index_keys();
    this->validate_channels();
    this->breaker = breaker::Breaker(breaker::default_config(task.name));
    int err = this->create_channels();
    if (err) {
        this->log_error(
            "failed to create channels for " + this->reader_config.task_name);
        return -1;
    }
    if (this->reader_config.sample_rate < this->reader_config.stream_rate || this->
        reader_config.sample_rate.value <
        1) {
        this->log_error(
            "Failed while configuring timing for NI hardware for task " + this->
            reader_config.task_name);
        this->err_info["message"] =
                "sample rate must be greater than or equal to 1 and greater than or equal to the stream rate";
        this->err_info["running"] = false;

        this->ctx->set_state({
            .task = this->task.key,
            .variant = "error",
            .details = err_info
        });
        return -1;
    }
    if (this->configure_timing())
        this->log_error(
            "[ni.reader] Failed while configuring timing for NI hardware for task " +
            this->reader_config.task_name);

    return 0;
}

xerrors::Error ni::Source::cycle() {
    auto err = this->silent_start();
    if (err) return err;
    err = this->silent_stop();
    if (err) return err;
    return xerrors::NIL;
}

xerrors::Error ni::Source::silent_start() {
    if (this->check_error(this->dmx->StartTask(this->task_handle), "StartTask")) {
        this->log_error(
            "failed while starting reader for task " + this->reader_config.task_name +
            " requires reconfigure");
        this->clear_task();
        return driver::CRITICAL_HARDWARE_ERROR;
    }
    return xerrors::NIL;
}

xerrors::Error ni::Source::silent_stop() {
    if (this->check_error(this->dmx->StopTask(this->task_handle), "StopTask")) {
        this->log_error(
            "failed while stopping reader for task " + this->reader_config.task_name);
        return driver::CRITICAL_HARDWARE_ERROR;
    }
    return xerrors::NIL;
}

xerrors::Error ni::Source::start(const std::string &cmd_key) {
    if (this->breaker.running() || !this->ok()) return xerrors::NIL;
    this->breaker.start();
    this->silent_start();
    this->sample_thread = std::thread(&ni::Source::acquire_data, this);
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

xerrors::Error ni::Source::stop(const std::string &cmd_key) {
    if (!this->breaker.running() || !this->ok()) return xerrors::NIL;
    this->breaker.stop();
    if (this->sample_thread.joinable()) this->sample_thread.join();
    this->silent_stop();
    data_queue.reset();
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

void ni::Source::clear_task() {
    this->check_error(this->dmx->ClearTask(this->task_handle), "ClearTask");
}


ni::Source::~Source() {
    this->clear_task();
    if (this->sample_thread.joinable()) this->sample_thread.join();
    VLOG(1) << "[ni.reader] joined sample thread";
}

int ni::Source::check_error(int32 error, std::string caller) {
    if(!this->ok()) return 0;
    if (error == 0) return 0;

    char errBuff[4096] = {'\0'};
    this->dmx->GetExtendedErrorInfo(errBuff, 4096);

    std::string s(errBuff);
    jsonify_error(s);

    this->ctx->set_state({
        .task = this->task.key,
        .variant = "error",
        .details = err_info
    });

    LOG(ERROR) << "[ni.reader] Vendor error (" << caller << "): " << s;
    this->ok_state = false;
    return -1;
}

bool ni::Source::ok() {
    return this->ok_state;
}

std::vector<synnax::ChannelKey> ni::Source::get_channel_keys() {
    std::vector<synnax::ChannelKey> keys;
    for (const auto &channel : this->reader_config.channels) {
        if (channel.enabled) {
            keys.push_back(channel.key);
        }
    }
    return keys;
}

void ni::Source::log_error(std::string err_msg) {
    LOG(ERROR) << "[ni.reader] " << err_msg;
    this->ok_state = false;
}

void ni::Source::stopped_with_err(const xerrors::Error &err) {
    if(this->ok()) return;
    this->log_error("stopped with error: " + err.message());
    json j = json(err.message());
    this->ctx->set_state({
        .task = this->reader_config.task_key,
        .variant = "error",
        .details = {
            {"running", false},
            {"message", j}
        }
    });
    // Unprompted stop so we pass in an empty command key.
    this->stop("");
    this->clear_task();
}


void ni::Source::jsonify_error(std::string s) {
    this->err_info = format_ni_error(parse_ni_error(s), s, this->channel_map);
}
