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

#include "glog/logging.h"
#include "nlohmann/json.hpp"

using json = nlohmann::json;

void ni::DigitalReadSource::parse_channels(config::Parser &parser) {
    VLOG(1) << "[ni.reader] Parsing Channels for task " << this->reader_config.
            task_name;
    parser.iter("channels",
                [&](config::Parser &channel_builder) {
                    ni::ChannelConfig config;
                    // digital channel names are formatted: <device_name>/port<port_number>/line<line_number>
                    std::string port = "port" + std::to_string(
                                           channel_builder.required<std::uint64_t>(
                                               "port"));
                    std::string line = "line" + std::to_string(
                                           channel_builder.required<std::uint64_t>(
                                               "line"));

                    config.channel_key = channel_builder.required<uint32_t>("channel");
                    config.name = (this->reader_config.device_name + "/" + port + "/" +
                                   line);
                    config.enabled = channel_builder.optional<bool>("enabled", true);
                    this->reader_config.channels.push_back(config);
                });
    if (!parser.ok())
        LOG(ERROR) << "Failed to parse channels for task " << this->
                reader_config.task_name;
}

int ni::DigitalReadSource::create_channels() {
    int err = 0;
    auto channels = this->reader_config.channels;
    for (auto &channel: channels) {
        if (channel.channel_type != "index" && channel.enabled) {
            err = this->check_ni_error(
                ni::NiDAQmxInterface::CreateDIChan(task_handle,
                                                   channel.name.c_str(),
                                                   "", DAQmx_Val_ChanPerLine));
            VLOG(1) << "Channel name: " << channel.name;
        }
        this->num_channels++;
        if (err < 0) {
            LOG(ERROR) << "[ni.reader] failed while configuring channel " << channel.
                    name;
            this->ok_state = false;
            return -1;
        }
    }
    return 0;
}

int ni::DigitalReadSource::configure_timing() {
    if (this->reader_config.timing_source == "none") {
        // if timing is not enabled, implement timing in software, reading one sample at a time
        this->num_samples_per_channel = 1;
    } else {
        if (this->check_ni_error(
            ni::NiDAQmxInterface::CfgSampClkTiming(this->task_handle,
                                                   this->reader_config.timing_source.c_str(),
                                                   this->reader_config.sample_rate.value,
                                                   DAQmx_Val_Rising,
                                                   DAQmx_Val_ContSamps,
                                                   this->reader_config.sample_rate.value))) {
            LOG(ERROR) << "[ni.reader] failed while configuring timing for task " <<
                    this->reader_config.task_name;
            this->ok_state = false;
            return -1;
        }
        this->num_samples_per_channel = std::floor(
            this->reader_config.sample_rate.value / this->reader_config.stream_rate.
            value);
    }
    this->buffer_size = this->num_channels * this->num_samples_per_channel;
    this->timer = loop::Timer(this->reader_config.stream_rate);
    this->sample_timer = loop::Timer(this->reader_config.sample_rate);
    return 0;
}


void ni::DigitalReadSource::acquire_data() {
    while (this->breaker.running()) {
        int32 numBytesPerSamp;
        DataPacket data_packet;
        data_packet.digital_data.resize(this->buffer_size);
        data_packet.t0 = synnax::TimeStamp::now().value;

        // sleep per sample rate
        this->sample_timer.wait();
        if (this->check_ni_error(
            ni::NiDAQmxInterface::ReadDigitalLines(
                this->task_handle,
                this->num_samples_per_channel,
                -1,
                DAQmx_Val_GroupByChannel,
                data_packet.digital_data.data(),
                data_packet.digital_data.size(),
                &data_packet.samples_read_per_channel,
                &numBytesPerSamp,
                NULL))) {
            this->log_error(
                "failed while reading digital data for task " + this->reader_config.
                task_name);
        }
        data_packet.tf = synnax::TimeStamp::now().value;
        data_queue.enqueue(data_packet);
    }
}

std::pair<synnax::Frame, freighter::Error> ni::DigitalReadSource::read(
    breaker::Breaker &breaker) {
    auto f = synnax::Frame(num_channels);

    // sleep per stream rate
    timer.wait(breaker);
    auto [d, err] = data_queue.dequeue();
    if (!err)
        return std::make_pair(std::move(f), freighter::Error(
                                  driver::TEMPORARY_HARDWARE_ERROR,
                                  "Failed to read data from queue"));
    // interpolate  timestamps between the initial and final timestamp to ensure
    // non-overlapping timestamps between batched reads
    uint64_t incr = ((d.tf - d.t0) / this->num_samples_per_channel);

    uint64_t data_index = 0;

    for (int i = 0; i < num_channels; i++) {
        if (!this->reader_config.channels[i].enabled) continue;
        if (this->reader_config.channels[i].channel_type == "index") {
            auto t = synnax::Series(synnax::TIMESTAMP, this->num_samples_per_channel);
            for (uint64_t j = 0; j < d.samples_read_per_channel; ++j)
                t.write(d.t0 + j * incr);

            f.add(this->reader_config.channels[i].channel_key, std::move(t));
            continue;
        }
        auto series = synnax::Series(synnax::SY_UINT8, d.samples_read_per_channel);

        for (int j = 0; j < d.samples_read_per_channel; j++)
            series.write((uint8_t) d.digital_data[data_index + j]);

        f.add(this->reader_config.channels[i].channel_key, std::move(series));
        data_index++;
    }
    return std::make_pair(std::move(f), freighter::NIL);
}

int ni::DigitalReadSource::validate_channels() {
    for (auto &channel: this->reader_config.channels) {
        if (channel.channel_type == "index") {
            if (channel.channel_key == 0) {
                LOG(ERROR) << "[ni.reader] Index channel key is 0";
                return -1;
            }
            continue;
        }
        auto [channel_info, err] = this->ctx->client->channels.retrieve(
            channel.channel_key);
        if (channel_info.data_type != synnax::SY_UINT8) {
            this->log_error("Channel " + channel.name + " is not of type SY_UINT8");
            this->ctx->setState({
                .task = task.key,
                .variant = "error",
                .details = {
                    {"running", "false"},
                    {
                        "message", "Channel " + channel.name +
                                   " is not of type SY_UINT8"
                    }
                }
            });
            return -1;
        }
    }
    return 0;
}
