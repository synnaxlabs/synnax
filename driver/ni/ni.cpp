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
            channel.channel_key);
        if (err)
            return this->log_error(
                "failed to retrieve channel " + std::to_string(channel.channel_key));
        index_keys.insert(channel_info.index);
    }
    for (auto &index_key: index_keys) {
        auto [channel_info, err] = this->ctx->client->channels.retrieve(index_key);
        if (err)
            return this->log_error(
                "failed to retrieve channel " + std::to_string(index_key));
        this->reader_config.channels.emplace_back(ni::ReaderChannelConfig{
            .channel_key = channel_info.key,
            .name = channel_info.name,
            .channel_type = "index"
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

freighter::Error ni::Source::cycle() {
    auto err = this->start_ni();
    if (err) return err;
    err = this->stop_ni();
    if (err) return err;
    return freighter::NIL;
}

freighter::Error ni::Source::start_ni() {
    if (this->check_error(this->dmx->StartTask(this->task_handle), "StartTask")) {
        this->log_error(
            "failed while starting reader for task " + this->reader_config.task_name +
            " requires reconfigure");
        this->clear_task();
        return driver::CRITICAL_HARDWARE_ERROR;
    }
    return freighter::NIL;
}

freighter::Error ni::Source::stop_ni() {
    if (this->check_error(this->dmx->StopTask(this->task_handle), "StopTask")) {
        this->log_error(
            "failed while stopping reader for task " + this->reader_config.task_name);
        return driver::CRITICAL_HARDWARE_ERROR;
    }
    return freighter::NIL;
}

freighter::Error ni::Source::start(const std::string &cmd_key) {
    if (this->breaker.running() || !this->ok()) return freighter::NIL;
    this->breaker.start();
    this->start_ni();
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
    return freighter::NIL;
}

freighter::Error ni::Source::stop(const std::string &cmd_key) {
    if (!this->breaker.running() || !this->ok()) return freighter::NIL;
    this->breaker.stop();
    if (this->sample_thread.joinable()) this->sample_thread.join();
    this->stop_ni();
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
    return freighter::NIL;
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
            keys.push_back(channel.channel_key);
        }
    }
    return keys;
}

void ni::Source::log_error(std::string err_msg) {
    LOG(ERROR) << "[ni.reader] " << err_msg;
    this->ok_state = false;
}

void ni::Source::stopped_with_err(const freighter::Error &err) {
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
    this->err_info["running"] = false;

    // Define regex patterns
    std::regex status_code_regex(R"(Status Code:\s*(-?\d+))");
    std::regex channel_regex(R"(Channel Name:\s*(\S+))");
    std::regex physical_channel_regex(R"(Physical Channel Name:\s*(\S+))");
    std::regex device_regex(R"(Device:\s*(\S+))");
    std::regex possible_values_regex(R"(Possible Values:\s*([\w\s,.-]+))");
    std::regex max_value_regex(R"(Maximum Value:\s*([\d.\s,eE-]+))");
    std::regex min_value_regex(R"(Minimum Value:\s*([\d.\s,eE-]+))");
    std::regex property_regex(R"(Property:\s*(\S+))");
    std::regex task_name_regex(R"(Task Name:\s*(\S+))");

    // Remove the Task Name line if it exists
    std::regex task_name_line_regex(R"(\nTask Name:.*\n?)");
    s = std::regex_replace(s, task_name_line_regex, "");

    // Extract status code
    std::string sc = "";
    std::smatch status_code_match;
    if (std::regex_search(s, status_code_match, status_code_regex))
        sc = status_code_match[1].str();

    // Remove the redundant Status Code line at the end
    std::regex status_code_line_regex(R"(\nStatus Code:.*$)");
    s = std::regex_replace(s, status_code_line_regex, "");

    // Extract device name
    std::string device = "";
    std::smatch device_match;
    if (std::regex_search(s, device_match, device_regex))
        device = device_match[1].str();

    // Extract physical channel name or channel name
    std::string cn = "";
    std::smatch physical_channel_match;
    if (std::regex_search(s, physical_channel_match, physical_channel_regex)) {
        cn = physical_channel_match[1].str();
        if (!device.empty()) cn = device + "/" + cn;
        // Combine device and physical channel name
    } else {
        std::smatch channel_match;
        if (std::regex_search(s, channel_match, channel_regex))
            cn = channel_match[1].str();
    }

    // Extract the first property
    std::string p = "";
    std::smatch property_match;
    if (std::regex_search(s, property_match, property_regex))
        p = property_match[1].str();
    if (sc == "-200170") p = "port";

    // Extract possible values
    std::string possible_values = "";
    std::smatch possible_values_match;
    if (std::regex_search(s, possible_values_match, possible_values_regex)) {
        possible_values = possible_values_match[1].str();
        size_t pos = possible_values.find("Channel Name");
        if (pos != std::string::npos)
            possible_values.erase(
                pos, std::string("Channel Name").length());
    }

    // Extract maximum value
    std::string max_value = "";
    std::smatch max_value_match;
    if (std::regex_search(s, max_value_match, max_value_regex))
        max_value = max_value_match[1].str();

    // Extract minimum value
    std::string min_value = "";
    std::smatch min_value_match;
    if (std::regex_search(s, min_value_match, min_value_regex))
        min_value = min_value_match[1].str();

    // Check if the channel name is in the channel map
    if (channel_map.count(cn) != 0) this->err_info["path"] = channel_map[cn] + ".";
    else if (!cn.empty()) this->err_info["path"] = cn + ".";
    else this->err_info["path"] = "";

    // Check if the property is in the field map
    if (FIELD_MAP.count(p) == 0)
        this->err_info["path"] =
                this->err_info["path"].get<std::string>() + p;
    else
        this->err_info["path"] = this->err_info["path"].get<std::string>() + FIELD_MAP.
                                 at(p);

    // Construct the error message
    std::string error_message = "NI Error " + sc + ": " + s + "\nPath: " + this->
                                err_info["path"].get<std::string>();
    if (!cn.empty()) error_message += " Channel: " + cn;
    if (!possible_values.empty())
        error_message += " Possible Values: " +
                possible_values;
    if (!max_value.empty()) error_message += " Maximum Value: " + max_value;
    if (!min_value.empty()) error_message += " Minimum Value: " + min_value;
    this->err_info["message"] = error_message;

    json j = json::array();
    j.push_back(this->err_info);

    LOG(INFO) << this->err_info.dump(4);
}
