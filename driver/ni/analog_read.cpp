// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std
#include <cassert>
#include <chrono>
#include <cstdio>
#include <utility>

#include "x/cpp/telem/telem.h"
#include "driver/ni/reader.h"
#include "glog/logging.h"
#include "nlohmann/json.hpp"


using json = nlohmann::json;

void ni::AnalogReadSource::parse_channels(xjson::Parser &parser) {
    std::uint64_t c_count = 0;
    parser.iter("channels",
                [&](xjson::Parser &channel_builder) {
                    ni::ReaderChannelConfig config;
                    // analog channel names are formatted: <device_name>/ai<port>
                    std::string port = std::to_string(
                        channel_builder.required<std::uint64_t>("port"));

                    std::string name;
                    if (this->reader_config.device_key != "cross-device") {
                        name = this->reader_config.device_name;
                    } else {
                        auto device_key = channel_builder.required<std::string>(
                            "device");
                        auto [dev, err] = this->ctx->client->hardware.retrieve_device(
                            device_key
                        );
                        if (err) {
                            this->log_error(
                                "failed to retrieve device with key " + device_key);
                            return;
                        }
                        name = dev.location;
                    }
                    config.name = name + "/ai" + port;

                    config.channel_key = channel_builder.required<uint32_t>("channel");
                    config.channel_type = channel_builder.required<std::string>("type");

                    config.ni_channel = this->parse_channel(
                        channel_builder, config.channel_type, config.name
                    );

                    this->channel_map[config.name] =
                            "channels." + std::to_string(c_count);

                    this->port_to_channel[channel_builder.required<std::uint64_t>(
                        "port")] = config.name;

                    config.enabled = channel_builder.optional<bool>("enabled", true);

                    this->reader_config.channels.push_back(config);

                    c_count++;
                }
    );
}

std::shared_ptr<ni::Analog> ni::AnalogReadSource::parse_channel(
    xjson::Parser &parser,
    const std::string &channel_type,
    const std::string &channel_name
) {
    auto channel = AnalogInputChannelFactory::create_channel(
        channel_type,
        parser,
        channel_name,
        this->port_to_channel
    );
    if (channel == nullptr) {
        std::string msg = "Channel " + channel_name + " has an unrecognized type: " +
                          channel_type;
        this->ctx->set_state({
            .task = task.key,
            .variant = "error",
            .details = {
                {"running", false},
                {"message", msg}
            }
        });
        this->log_error(msg);
    }

    return channel;
}

int ni::AnalogReadSource::configure_timing() {
    if (this->reader_config.timing_source == "none") {
        if (this->check_error(
            this->dmx->CfgSampClkTiming(
                this->task_handle,
                "",
                this->reader_config.sample_rate.value,
                DAQmx_Val_Rising,
                DAQmx_Val_ContSamps,
                this->reader_config.sample_rate.value
            ), "configure_timing.CfgSampClkTiming"
        )) {
            this->log_error("failed while configuring timing for task "
                            + this->reader_config.task_name);
            return -1;
        }
    } else if (this->check_error(
        this->dmx->CfgSampClkTiming(
            this->task_handle,
            this->reader_config.timing_source.c_str(),
            this->reader_config.sample_rate.value,
            DAQmx_Val_Rising,
            DAQmx_Val_ContSamps,
            this->reader_config.sample_rate.value
        ), "configure_timing.CfgSampClkTiming"
    )) {
        this->log_error("failed while configuring timing for task "
                        + this->reader_config.task_name);
        return -1;
    }

    // we read data in chunks of num_samples_per_channel such that we can send frames of
    // this->log data of size num_samples_per_channel at the stream rate
    // e.g. if we have 4 channels and want to stream at 100Hz at a 1000hz sample rate
    // make a call to read 10 samples at 100hz
    this->num_samples_per_channel = std::floor(
        this->reader_config.sample_rate.value /
        this->reader_config.stream_rate.value);

    this->buffer_size = this->num_ai_channels * this->num_samples_per_channel;
    this->timer = loop::Timer(this->reader_config.stream_rate);
    return 0;
}

void ni::AnalogReadSource::acquire_data() {
    while (this->breaker.running() && this->ok()) {
        DataPacket data_packet;
        data_packet.analog_data.resize(this->buffer_size);
        data_packet.t0 = telem::TimeStamp::now().value;
        if (this->check_error(
            this->dmx->ReadAnalogF64(
                this->task_handle,
                this->num_samples_per_channel,
                -1,
                DAQmx_Val_GroupByChannel,
                data_packet.analog_data.data(),
                data_packet.analog_data.size(),
                &data_packet.samples_read_per_channel,
                NULL
            ), "acquire_data.ReadAnalogF64"
        )) {
            this->log_error(
                "failed while reading analog data for task " + this->reader_config.
                task_name);
        }
        data_packet.tf = telem::TimeStamp::now().value;
        data_queue.enqueue(data_packet);
    }
}

std::pair<synnax::Frame, xerrors::Error> ni::AnalogReadSource::read(
    breaker::Breaker &breaker) {
    auto f = synnax::Frame(num_channels);

    auto [d, err] = data_queue.dequeue();
    if (!err)
        return std::make_pair(std::move(f), xerrors::Error(
                                  driver::CRITICAL_HARDWARE_ERROR,
                                  "Failed to read data from queue"));

    // Interpolate  timestamps between the initial and final timestamp to ensure
    // non-overlapping timestamps between batched reads
    const uint64_t incr = (d.tf - d.t0) / this->num_samples_per_channel;
    const size_t count = d.samples_read_per_channel;
    size_t data_index = 0;
    for (const auto &ch: this->reader_config.channels) {
        if (!ch.enabled) continue;
        if (ch.channel_type == "index") {
            auto t = telem::Series(telem::TIMESTAMP_T, count);
            for (uint64_t i = 0; i < count; ++i) t.write(d.t0 + i * incr);
            f.emplace(ch.channel_key, std::move(t));
            continue;
        }
        auto series = telem::Series(ch.data_type, count);
        const auto buf = d.analog_data.data();
        const int start = data_index * count;
        if (series.data_type == telem::FLOAT64_T) series.write(buf + start, count);
        else
            for (int i = 0; i < count; ++i)
                series.write(static_cast<float>(buf[start + i]));
        f.emplace(ch.channel_key, std::move(series));
        data_index++;
    }
    return std::make_pair(std::move(f), xerrors::NIL);
}

int ni::AnalogReadSource::create_channels() {
    for (auto &channel: this->reader_config.channels) {
        this->num_channels++;
        if (channel.channel_type == "index" || !channel.enabled ||
            !channel.ni_channel)
            continue;
        this->num_ai_channels++;
        this->check_error(channel.ni_channel->create_ni_scale(this->dmx),
                          "create_channels.create_ni_scale");
        this->check_error(channel.ni_channel->bind(this->dmx, this->task_handle),
                          "create_channels.bind");
        if (!this->ok()) {
            this->log_error("failed while creating channel " + channel.name);
            return -1;
        }
    }
    return 0;
}

int ni::AnalogReadSource::validate_channels() {
    for (auto &channel: this->reader_config.channels) {
        if (channel.channel_type == "index") {
            if (channel.channel_key == 0) {
                LOG(ERROR) << "[ni.reader] Index channel key is 0";
                return -1;
            }
            continue;
        }
        // if not index, make sure channel type is valid
        auto [channel_info, err] = this->ctx->client->channels.retrieve(
            channel.channel_key);
        if (channel_info.data_type != telem::FLOAT32_T && channel_info.data_type !=
            telem::FLOAT64_T) {
            this->log_error(
                "Channel " + channel.name + " is not of type float32 or float64");
            this->ctx->set_state({
                .task = task.key,
                .variant = "error",
                .details = {
                    {"running", false},
                    {
                        "message",
                        "Channel " + channel.name +
                        " must be type float32 or float64. Got " + channel_info.
                        data_type.
                        value
                    },
                    {"path", channel.name}
                }
            });
            return -1;
        }
        channel.data_type = channel_info.data_type;
    }
    return 0;
}
