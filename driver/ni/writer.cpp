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
#include <regex>
#include <set>

#include "client/cpp/telem/telem.h"
#include "driver/ni/writer.h"
#include "nlohmann/json.hpp"
#include "glog/logging.h"

///////////////////////////////////////////////////////////////////////////////////
//                                    WriteSink                                  //
///////////////////////////////////////////////////////////////////////////////////
void ni::WriteSink::get_index_keys() {
    if (this->writer_config.state_channel_keys.empty()) return;
    
    std::set<synnax::ChannelKey> unique_keys;
    for (const auto& state_channel : this->writer_config.state_channel_keys) {
        auto [state_channel_info, err] = this->ctx->client->channels.retrieve(state_channel);
        if (err) {
            this->log_error("failed to retrieve channel " + std::to_string(state_channel));
            continue;
        }
        unique_keys.insert(state_channel_info.index);
    }

    this->writer_config.state_index_keys = {unique_keys.begin(), unique_keys.end()};
}

freighter::Error ni::WriteSink::cycle() {
    auto err = this->start_ni();
    if (err) return err;
    err = this->stop_ni();
    if (err) return err;
    return freighter::NIL;
}

freighter::Error ni::WriteSink::start(const std::string &cmd_key) {
    if (this->breaker.running() || !this->ok()) return freighter::NIL;
    this->breaker.start();
    freighter::Error err = this->start_ni();
    if (err) return err;
    ctx->set_state({
        .task = this->task.key,
        .key = cmd_key,
        .variant = "success",
        .details = {
            {"running", true},
            {"message", "Task started successfully"}
        }
    });
    return freighter::NIL;
}

freighter::Error ni::WriteSink::stop(const std::string &cmd_key) {
    if (!this->breaker.running()) return freighter::NIL;
    this->breaker.stop();
    freighter::Error err = this->stop_ni();
    if (err) return err;
    ctx->set_state({
        .task = this->task.key,
        .key = cmd_key,
        .variant = "success",
        .details = {
            {"running", false},
            {"message", "Task stopped successfully"}
        }
    });
    return freighter::NIL;
}

std::vector<synnax::ChannelKey> ni::WriteSink::get_cmd_channel_keys() {
    std::vector<synnax::ChannelKey> keys;
    for (auto &channel: this->writer_config.channels)
        if (channel.channel_type != "index" && channel.enabled)
            keys.push_back(channel.channel_key);
    return keys;
}

std::vector<synnax::ChannelKey> ni::WriteSink::get_state_channel_keys() {
    std::vector<synnax::ChannelKey> keys;
    for (auto &channel: this->writer_config.channels)
        if (channel.channel_type != "index" && channel.enabled)
            keys.push_back(channel.state_channel_key);
    for (auto &index_key : this->writer_config.state_index_keys) {
        keys.push_back(index_key);
    }
    return keys;
}

int ni::WriteSink::check_err(int32 error, std::string caller) {
    if (error == 0) return 0;
    char errBuff[2048] = {'\0'};
    this->dmx->GetExtendedErrorInfo(errBuff, 2048);

    std::string s(errBuff);
    jsonify_error(s);

    this->ctx->set_state({
        .task = this->task.key,
        .variant = "error",
        .details = err_info
    });
    this->log_error("NI Vendor Error (" + caller + "): " + std::string(errBuff));
    this->ok_state = false;
    return -1;
}

bool ni::WriteSink::ok() {
    return this->ok_state;
}

void ni::WriteSink::log_error(std::string err_msg) {
    LOG(ERROR) << "[ni.writer] " << err_msg;
    this->ok_state = false;
}

void ni::WriteSink::stopped_with_err(const freighter::Error &err) {
    this->stop("");
    this->log_error("stopped with error: " + err.message());
    json j = json(err.message());
    this->ctx->set_state({
        .task = this->task.key,
        .variant = "error",
        .details = {
            {"running", false},
            {"message", j}
        }
    });
}

void ni::WriteSink::clear_task() {
    if (this->check_err(this->dmx->ClearTask(task_handle), "clear_task.ClearTask"))
        this->log_error("failed to clear writer for task " + this->writer_config.task_name);
}

void ni::WriteSink::jsonify_error(std::string s) {
    this->err_info["running"] = false;

    std::regex status_code_regex(R"(Status Code:\s*(-?\d+))");
    std::regex channel_regex(R"(Channel Name:\s*(\S+))");
    std::regex physical_channel_regex(R"(Physical Channel Name:\s*(\S+))");
    std::regex device_regex(R"(Device:\s*(\S+))");

    std::string message = s;
    std::vector<std::string> fields = {
        "Status Code:", "Channel Name:", "Physical Channel Name:",
        "Device:", "Task Name:"
    };

    size_t first_field_pos = std::string::npos;
    for (const auto &field: fields) {
        size_t pos = s.find("\n" + field);
        if (pos != std::string::npos && (
                first_field_pos == std::string::npos || pos < first_field_pos))
            first_field_pos = pos;
    }

    if (first_field_pos != std::string::npos)
        message = s.substr(0, first_field_pos);

    message = std::regex_replace(message, std::regex("\\s+$"), "");

    std::smatch status_code_match;
    std::regex_search(s, status_code_match, status_code_regex);
    std::string sc = (!status_code_match.empty()) ? status_code_match[1].str() : "";

    bool is_port_error = (sc == "-200170");

    std::string device = "";
    std::smatch device_match;
    if (std::regex_search(s, device_match, device_regex))
        device = device_match[1].str();

    std::string cn = "";
    std::smatch physical_channel_match;
    std::smatch channel_match;
    if (std::regex_search(s, physical_channel_match, physical_channel_regex)) {
        cn = physical_channel_match[1].str();
        if (!device.empty()) cn = device + "/" + cn;
    } else if (std::regex_search(s, channel_match, channel_regex))
        cn = channel_match[1].str();

    this->err_info["path"] = channel_map.count(cn) != 0
                                 ? channel_map[cn]
                                 : !cn.empty()
                                       ? cn
                                       : "";
    if (is_port_error)
        this->err_info["path"] = this->err_info["path"].get<std::string>() + ".port";

    std::string error_message = "NI Error " + sc + ": " + message + " Path: " +
                                this->err_info["path"].get<std::string>();

    if (!cn.empty()) error_message += " Channel: " + cn;

    this->err_info["message"] = error_message;

    json j = json::array();
    j.push_back(this->err_info);
    this->err_info["errors"] = j;
}

///////////////////////////////////////////////////////////////////////////////////
//                                DigitalWriteSink                               //
///////////////////////////////////////////////////////////////////////////////////
ni::DigitalWriteSink::DigitalWriteSink(
    const std::shared_ptr<DAQmx> &dmx,
    TaskHandle task_handle,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) : WriteSink(dmx, task_handle, ctx, task) {
    auto config_parser = config::Parser(task.config);
    writer_config = WriterConfig(config_parser, ctx, true, task_handle, task.key);

    if (!config_parser.ok()) {
        this->log_error("Failed to parse config: " + config_parser.error_json().dump(4));
        return;
    }

    this->breaker = breaker::Breaker(breaker::default_config(task.name));

    if (this->init()) {
        this->log_error("Failed to configure NI hardware for task " + writer_config.task_name);
    }

    this->get_index_keys();
    this->writer_state_source = std::make_shared<ni::DigitalStateSource>(
        writer_config.state_rate,
        writer_config.state_index_keys,
        writer_config.state_channel_keys
    );
}

ni::DigitalWriteSink::~DigitalWriteSink() {
    delete[] write_buffer;
}

int ni::DigitalWriteSink::init() {
    int err = 0;
    auto channels = this->writer_config.channels;

    for (auto &channel: channels) {
        if (channel.channel_type != "index" && channel.enabled) {
            err = this->check_err(
                this->dmx->CreateDOChan(
                    this->task_handle,
                    channel.name.c_str(),
                    "",
                    DAQmx_Val_ChanPerLine
                ), "init.CreateDOChan"
            );
        }
        this->num_channels++;
        if (err < 0) {
            this->log_error("failed to create channel " + channel.name);
            return -1;
        }
    }

    this->buffer_size = this->num_channels;
    this->write_buffer = new uint8_t[this->buffer_size];
    for (int i = 0; i < this->buffer_size; i++) write_buffer[i] = 0;
    return 0;
}

freighter::Error ni::DigitalWriteSink::start_ni() {
    if (this->check_err(this->dmx->StartTask(this->task_handle), "start_ni.StartTask")) {
        this->log_error("failed to start writer for task " + this->writer_config.task_name);
        return freighter::Error(driver::CRITICAL_HARDWARE_ERROR);
        this->clear_task();
    }
    LOG(INFO) << "[ni.writer] successfully started writer for task " << this->writer_config.task_name;
    return freighter::NIL;
}

freighter::Error ni::DigitalWriteSink::stop_ni() {
    if (this->check_err(this->dmx->StopTask(task_handle), "stop_ni.StopTask")) {
        this->log_error("failed to stop writer for task " + this->writer_config.task_name);
        return freighter::Error(driver::CRITICAL_HARDWARE_ERROR);
    }
    LOG(INFO) << "[ni.writer] successfully stopped writer for task " << this->writer_config.task_name;
    return freighter::NIL;
}

freighter::Error ni::DigitalWriteSink::write(const synnax::Frame &frame) {
    auto err = format_data(frame);
    if (err != freighter::NIL) {
        this->log_error("failed to format data");
        return err;
    }

    int32 samplesWritten = 0;
    if (this->check_err(
        this->dmx->WriteDigitalLines(
            this->task_handle,
            1,
            1,
            10.0,
            DAQmx_Val_GroupByChannel,
            write_buffer,
            &samplesWritten,
            NULL
        ), "write.WriteDigitalLines")) {
        this->log_error("failed while writing digital data");
        return freighter::Error(driver::CRITICAL_HARDWARE_ERROR,
                              "Error writing digital data");
    }

    this->writer_state_source->update_state(
        this->writer_config.modified_state_keys,
        this->writer_config.digital_modified_state_values
    );

    return freighter::NIL;
}

freighter::Error ni::DigitalWriteSink::format_data(const synnax::Frame &frame) {
    uint32_t frame_index = 0;
    uint32_t cmd_channel_index = 0;

    for (auto key: *(frame.channels)) {
        auto it = std::find(this->writer_config.drive_cmd_channel_keys.begin(),
                            this->writer_config.drive_cmd_channel_keys.end(), key);
        if (it != this->writer_config.drive_cmd_channel_keys.end()) {
            cmd_channel_index = std::distance(
                this->writer_config.drive_cmd_channel_keys.begin(),
                it);
            auto series = frame.series->at(frame_index).values<uint8_t>();
            write_buffer[cmd_channel_index] = series[0];
            this->writer_config.modified_state_keys.push(
                this->writer_config.state_channel_keys[cmd_channel_index]);
            this->writer_config.digital_modified_state_values.emplace(series[0]);
            
        }
        frame_index++;
    }
    return freighter::NIL;
}

///////////////////////////////////////////////////////////////////////////////////
//                                AnalogWriteSink                                //
///////////////////////////////////////////////////////////////////////////////////
ni::AnalogWriteSink::AnalogWriteSink(
    const std::shared_ptr<DAQmx> &dmx,
    TaskHandle task_handle,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) : WriteSink(dmx, task_handle, ctx, task) {
    auto config_parser = config::Parser(task.config);
    writer_config = WriterConfig(config_parser, ctx, false, task_handle, task.key);

    if (!config_parser.ok()) {
        this->log_error("Failed to parse config: " + config_parser.error_json().dump(4));
        return;
    }

    this->breaker = breaker::Breaker(breaker::default_config(task.name));

    if (this->init()) {
        this->log_error("Failed to configure NI hardware for task " + writer_config.task_name);
    }

    this->get_index_keys();
    this->writer_state_source = std::make_shared<ni::AnalogStateSource>(
        writer_config.state_rate,
        writer_config.state_index_keys,
        writer_config.state_channel_keys
    );
}

ni::AnalogWriteSink::~AnalogWriteSink() {
    delete[] write_buffer;
}

int ni::AnalogWriteSink::init() {
    int err = 0;
    auto channels = this->writer_config.channels;

    for (auto &channel: channels) {
        this->check_err(channel.ni_channel->create_ni_scale(this->dmx), "init.create_ni_scale");
        this->check_err(channel.ni_channel->bind(this->dmx, this->task_handle), "init.bind");
        if (!this->ok()) {
            this->log_error("failed while creating channel " + channel.name);
            return -1;
        }
        this->num_channels++;
    }
    this->buffer_size = this->num_channels;
    this->write_buffer = new double[this->buffer_size];
    for (int i = 0; i < this->buffer_size; i++) write_buffer[i] = 0;
    return 0;
}

freighter::Error ni::AnalogWriteSink::start_ni() {
    if (this->check_err(this->dmx->StartTask(this->task_handle), "start_ni.StartTask")) {
        this->log_error("failed to start writer for task " + this->writer_config.task_name);
        return freighter::Error(driver::CRITICAL_HARDWARE_ERROR);
        this->clear_task();
    }
    LOG(INFO) << "[ni.writer] successfully started writer for task " << this->writer_config.task_name;
    return freighter::NIL;
}

freighter::Error ni::AnalogWriteSink::stop_ni() {
    if (this->check_err(this->dmx->StopTask(task_handle), "stop_ni.StopTask")) {
        this->log_error("failed to stop writer for task " + this->writer_config.task_name);
        return freighter::Error(driver::CRITICAL_HARDWARE_ERROR);
    }
    LOG(INFO) << "[ni.writer] successfully stopped writer for task " << this->writer_config.task_name;
    return freighter::NIL;
}

freighter::Error ni::AnalogWriteSink::write(const synnax::Frame &frame) {
    auto err = format_data(frame);
    if (err != freighter::NIL) {
        this->log_error("failed to format data");
        return err;
    }

    int32 samplesWritten = 0;
    if (this->check_err(
        this->dmx->WriteAnalogF64(
            this->task_handle,
            1,
            1,
            10.0,
            DAQmx_Val_GroupByChannel,
            write_buffer,
            &samplesWritten,
            NULL
        ), "write.WriteAnalogF64")) {
        this->log_error("failed while writing analog data");
        return freighter::Error(driver::CRITICAL_HARDWARE_ERROR,
                              "Error writing analog data");
    }

    this->writer_state_source->update_state(
        this->writer_config.modified_state_keys,
        this->writer_config.analog_modified_state_values
    );

    return freighter::NIL;
}

freighter::Error ni::AnalogWriteSink::format_data(const synnax::Frame &frame) {
    uint32_t frame_index = 0;
    uint32_t cmd_channel_index = 0;

    for (auto key: *(frame.channels)) {
        auto it = std::find(this->writer_config.drive_cmd_channel_keys.begin(),
                            this->writer_config.drive_cmd_channel_keys.end(), key);
        if (it != this->writer_config.drive_cmd_channel_keys.end()) {
            cmd_channel_index = std::distance(
                this->writer_config.drive_cmd_channel_keys.begin(),
                it);
            const auto &series = frame.series->at(frame_index);
            double value = 0.0;

            if (series.data_type == synnax::FLOAT32) {
                value = series.at<float>(0);
            } else if (series.data_type == synnax::FLOAT64) {
                value = series.at<double>(0);
            } else if (series.data_type == synnax::INT32) {
                value = static_cast<double>(series.at<int32_t>(0));
            } else if (series.data_type == synnax::SY_UINT8) {
                value = static_cast<double>(series.at<uint8_t>(0));
            } else {
                return freighter::NIL;
            }

            write_buffer[cmd_channel_index] = value;
            this->writer_config.modified_state_keys.push(
                this->writer_config.state_channel_keys[cmd_channel_index]
            );
            this->writer_config.analog_modified_state_values.push(value);
        }
        frame_index++;
    }
    return freighter::NIL;
}
