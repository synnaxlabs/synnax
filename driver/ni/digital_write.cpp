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

#include "client/cpp/telem/telem.h"
#include "driver/ni/ni.h"

#include "nlohmann/json.hpp"
#include "glog/logging.h"

///////////////////////////////////////////////////////////////////////////////////
//                             Helper Functions                                  //
///////////////////////////////////////////////////////////////////////////////////
void ni::DigitalWriteSink::get_index_keys() {
    if (this->writer_config.state_channel_keys.empty()) return;
    auto state_channel = this->writer_config.state_channel_keys[0];
    auto [state_channel_info, err] = this->ctx->client->channels.
            retrieve(state_channel);
    if (err) return this->log_error("failed to retrieve channel " + state_channel);
    this->writer_config.state_index_key = state_channel_info.index;
}

///////////////////////////////////////////////////////////////////////////////////
//                                    daqWriter                                  //
///////////////////////////////////////////////////////////////////////////////////
ni::DigitalWriteSink::DigitalWriteSink(
    TaskHandle task_handle,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task)
    : task_handle(task_handle),
      ctx(ctx),
      task(task),
      err_info({}) {
    auto config_parser = config::Parser(task.config);
    this->writer_config.task_name = task.name;
    this->parse_config(config_parser);
    if (!config_parser.ok()) {
        this->log_error(
            "failed to parse configuration for " + this->writer_config.task_name);
        this->ctx->setState({
            .task = this->task.key,
            .variant = "error",
            .details = config_parser.error_json()
        });
        return;
    }
    auto breaker_config = breaker::Config{
        .name = task.name,
        .base_interval = 1 * SECOND,
        .max_retries = 20,
        .scale = 1.2,
    };
    this->breaker = breaker::Breaker(breaker_config);
    if (this->init())
        this->log_error("failed to configure NI hardware for task " + this->
                        writer_config.task_name);

    this->get_index_keys();
    this->writer_state_source = std::make_shared<ni::StateSource>(
        this->writer_config.state_rate,
        this->writer_config.state_index_key,
        this->writer_config.state_channel_keys);
}


void ni::DigitalWriteSink::parse_config(config::Parser &parser) {
    this->writer_config.state_rate = parser.required<float>("state_rate");
    this->writer_config.device_key = parser.required<std::string>("device");

    auto [dev, err] = this->ctx->client->hardware.retrieveDevice(
        this->writer_config.device_key);

    if (err != freighter::NIL)
        return this->log_error(
            "failed to retrieve device with key " + this->writer_config.device_key);

    this->writer_config.device_name = dev.location;
    std::uint64_t c_count = 0;
    parser.iter("channels",
                [&](config::Parser &channel_builder) {
                    ni::ChannelConfig config;
                    // digital channel names are formatted: <device_name>/port<port_number>/line<line_number>
                    auto port = "port" + std::to_string(
                                    channel_builder.required<std::uint64_t>(
                                        "port"));
                    auto line = "line" + std::to_string(
                                    channel_builder.required<std::uint64_t>(
                                        "line"));

                    config.name = (this->writer_config.device_name + "/" + port + "/" +
                                   line);

                    config.channel_key = channel_builder.required<uint32_t>(
                        "cmd_channel");
                    this->writer_config.drive_cmd_channel_keys.push_back(
                        config.channel_key);

                    auto state_key = channel_builder.required<uint32_t>(
                        "state_channel");
                    this->writer_config.state_channel_keys.push_back(
                        state_key);

                    this->channel_map[config.name] =
                            "channels." + std::to_string(c_count);
                    this->writer_config.channels.push_back(config);
                    c_count++;
                });
}


int ni::DigitalWriteSink::init() {
    int err = 0;
    auto channels = this->writer_config.channels;

    for (auto &channel: channels) {
        if (channel.channel_type != "index") {
            err = this->check_ni_error(ni::NiDAQmxInterface::CreateDOChan(
                this->task_handle, channel.name.c_str(), "",
                DAQmx_Val_ChanPerLine));
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

freighter::Error ni::DigitalWriteSink::cycle() {
    auto err = this->start_ni();
    if (err) return err;
    err = this->stop_ni();
    if (err) return err;
    return freighter::NIL;
}

freighter::Error ni::DigitalWriteSink::start_ni() {
    if (this->check_ni_error(ni::NiDAQmxInterface::StartTask(this->task_handle))) {
        this->log_error(
            "failed to start writer for task " + this->writer_config.task_name);
        return freighter::Error(driver::CRITICAL_HARDWARE_ERROR);
        this->clear_task();
    }
    LOG(INFO) << "[ni.writer] successfully started writer for task " << this->
            writer_config.task_name;
    return freighter::NIL;
}


freighter::Error ni::DigitalWriteSink::stop_ni() {
    if (this->check_ni_error(ni::NiDAQmxInterface::StopTask(task_handle))) {
        this->log_error(
            "failed to stop writer for task " + this->writer_config.task_name);
        return freighter::Error(driver::CRITICAL_HARDWARE_ERROR);
    }
    LOG(INFO) << "[ni.writer] successfully stopped writer for task " << this->
            writer_config.task_name;
    return freighter::NIL;
}

freighter::Error ni::DigitalWriteSink::start(const std::string &cmd_key) {
    if (this->breaker.running() || !this->ok()) return freighter::NIL;
    this->breaker.start();
    freighter::Error err = this->start_ni();
    if (err) return err;
    ctx->setState({
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


freighter::Error ni::DigitalWriteSink::stop(const std::string &cmd_key) {
    if (!this->breaker.running()) return freighter::NIL;
    this->breaker.stop();
    freighter::Error err = this->stop_ni();
    if (err) return err;
    ctx->setState({
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

freighter::Error ni::DigitalWriteSink::write(synnax::Frame frame) {
    int32 samplesWritten = 0;
    format_data(std::move(frame));

    if (this->check_ni_error(ni::NiDAQmxInterface::WriteDigitalLines(this->task_handle,
                                                                     1, // number of samples per channel
                                                                     1, // auto start
                                                                     10.0, // timeout
                                                                     DAQmx_Val_GroupByChannel, // data layout
                                                                     write_buffer, // data
                                                                     &samplesWritten, // samples written
                                                                     NULL))) {
        this->log_error("failed while writing digital data");
        return freighter::Error(driver::CRITICAL_HARDWARE_ERROR,
                                "Error writing digital data");
    }
    this->writer_state_source->update_state(this->writer_config.modified_state_keys,
                                            this->writer_config.modified_state_values);

    return freighter::NIL;
}


freighter::Error ni::DigitalWriteSink::format_data(const synnax::Frame &frame) {
    uint32_t frame_index = 0;
    uint32_t cmd_channel_index = 0;

    for (auto key: *(frame.channels)) {
        // the order the keys were pushed into the vector is the order the data is written
        // first see if the key is in the drive_cmd_channel_keys
        auto it = std::find(this->writer_config.drive_cmd_channel_keys.begin(),
                            this->writer_config.drive_cmd_channel_keys.end(), key);
        if (it != this->writer_config.drive_cmd_channel_keys.end()) {
            // if so, now find which index it is in the vector (i.e. which channel it is in the write_buffer)
            cmd_channel_index = std::distance(
                this->writer_config.drive_cmd_channel_keys.begin(),
                it);
            // this corresponds to where in the order its NI channel was created
            // now we grab the level we'd like to write and put it into that location in the write_buffer
            auto series = frame.series->at(frame_index).values<uint8_t>();
            write_buffer[cmd_channel_index] = series[0];
            this->writer_config.modified_state_keys.push(
                this->writer_config.state_channel_keys[cmd_channel_index]);
            this->writer_config.modified_state_values.push(series[0]);
        }
        frame_index++;
    }
    return freighter::NIL;
}

ni::DigitalWriteSink::~DigitalWriteSink() {
    this->clear_task();
    delete[] this->write_buffer;
}

void ni::DigitalWriteSink::clear_task() {
    if (this->check_ni_error(ni::NiDAQmxInterface::ClearTask(task_handle)))
        this->log_error(
            "failed to clear writer for task " + this->writer_config.task_name);
}

std::vector<synnax::ChannelKey> ni::DigitalWriteSink::get_cmd_channel_keys() {
    std::vector<synnax::ChannelKey> keys;
    for (auto &channel: this->writer_config.channels)
        if (channel.channel_type != "index") keys.push_back(channel.channel_key);
    return keys;
}

std::vector<synnax::ChannelKey> ni::DigitalWriteSink::get_state_channel_keys() {
    std::vector<synnax::ChannelKey> keys = this->writer_config.state_channel_keys;
    keys.push_back(this->writer_config.state_index_key);
    return keys;
}

int ni::DigitalWriteSink::check_ni_error(int32 error) {
    if (error < 0) {
        char errBuff[2048] = {'\0'};
        ni::NiDAQmxInterface::GetExtendedErrorInfo(errBuff, 2048);

        std::string s(errBuff);
        jsonify_error(s);

        this->ctx->setState({
            .task = this->task.key,
            .variant = "error",
            .details = err_info
        });
        this->log_error("NI Vendor Error: " + std::string(errBuff));
        return -1;
    }
    return 0;
}


bool ni::DigitalWriteSink::ok() {
    return this->ok_state;
}

void ni::DigitalWriteSink::log_error(std::string err_msg) {
    // TODO get rid of the fields outside of the errors array
    LOG(ERROR) << "[ni.writer] " << err_msg;
    this->ok_state = false;
}

void ni::DigitalWriteSink::stopped_with_err(const freighter::Error &err) {
    // Unprompted stop so we pass in an empty command key
    this->stop("");
    this->log_error("stopped with error: " + err.message());
    json j = json(err.message());
    this->ctx->setState({
        .task = this->task.key,
        .variant = "error",
        .details = {
            {"running", false},
            {"message", j}
        }
    });
}


void ni::DigitalWriteSink::jsonify_error(std::string s) {
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

    if (first_field_pos != std::string::npos) message = s.substr(0, first_field_pos);

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
        <<
        <<
        <<
        <
        HEAD
        if (!device.empty())
            cn = device + "/" + cn; // Combine device and physical channel name
    } else if (std::regex_search(s, channel_match, channel_regex))
        cn = channel_match[1].str();
    ==
    ==
    ==
    =
    if (!device.empty()) cn = device + "/" + cn; // Combine device and physical channel name
}

else
if
(std::regex_search
(s
,
channel_match
,
channel_regex
)
)
cn= channel_match
[1]
.
str();

>
>
>
>
>
>
>
6ac783907e6ea38ca7856416cf48895c0bf1eab7

// Check if the channel name is in the channel map
this
->
err_info ["path"] = channel_map
.
count (cn)
!=
0
?
channel_map [cn]
    :

!
cn
.
empty()

?
cn:
"";
// Handle the special case for -200170 error
if
(is_port_error)
this
->
err_info ["path"] =
this
->
err_info ["path"]
.
get<std::string>()

+
".port";

std::string error_message = "NI Error " + sc + ": " + message + " Path: " + this->
                            err_info["path"].get<std::string>();

if
(
!
cn
.
empty()

)
error_message
+=
" Channel: "
+
cn;

this
->
err_info ["message"] = error_message;

json j = json::array();
j
.
push_back (
this
->
err_info
);
this
->
err_info ["errors"] = j;
}

///////////////////////////////////////////////////////////////////////////////////
//                                    StateSource                                //
///////////////////////////////////////////////////////////////////////////////////
ni::StateSource::StateSource(
    float state_rate,
    synnax::ChannelKey &state_index_key,
    std::vector<synnax::ChannelKey> &state_channel_keys
) {
    this->state_rate.value = state_rate;
    // start the periodic thread
    this->state_index_key = state_index_key;

    // initialize all states to 0 (logic low)
    for (auto &key: state_channel_keys)
        this->state_map[key] = 0;
    this->timer = loop::Timer(this->state_rate);
}

std::pair<synnax::Frame, freighter::Error> ni::StateSource::read(
    breaker::Breaker &breaker) {
    std::unique_lock<std::mutex> lock(this->state_mutex);
    // sleep for state period
    this->timer.wait(breaker);
    waiting_reader.wait_for(lock, this->state_rate.period().chrono());
    return std::make_pair(this->get_state(), freighter::NIL);
}

synnax::Frame ni::StateSource::get_state() {
    // frame size = # monitored states + 1 state index channel
    auto state_frame = synnax::Frame(this->state_map.size() + 1);
    state_frame.add(
        this->state_index_key,
        synnax::Series(
            synnax::TimeStamp::now().value,
            synnax::TIMESTAMP
        )
    );
    for (auto &[key, value]: this->state_map)
        state_frame.add(key, synnax::Series(value));
    return state_frame;
}

void ni::StateSource::update_state(
    std::queue<synnax::ChannelKey> &modified_state_keys,
    std::queue<std::uint8_t> &modified_state_values
) {
    std::unique_lock<std::mutex> lock(this->state_mutex);
    while (!modified_state_keys.empty()) {
        this->state_map[modified_state_keys.front()] = modified_state_values.front();
        modified_state_keys.pop();
        modified_state_values.pop();
    }
    waiting_reader.notify_one();
}
