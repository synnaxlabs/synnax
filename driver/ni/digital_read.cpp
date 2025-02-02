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
    digital_config = DigitalReaderConfig(parser);
    reader_config = static_cast<BaseReaderConfig>(digital_config);
    
    for (size_t i = 0; i < digital_config.channels.size(); i++) {
        if (digital_config.channels[i].enabled) {
            this->channel_map[digital_config.channels[i].name] = "channels." + std::to_string(i);
        }
    }
}

int ni::DigitalReadSource::create_channels() {
    int err = 0;
    for (auto &channel: digital_config.channels) {
        if (channel.channel_type != "index" && channel.enabled) {
            err = this->check_ni_error(
                this->dmx->CreateDIChan(task_handle,
                                      channel.name.c_str(),
                                      "", DAQmx_Val_ChanPerLine));
            VLOG(1) << "Channel name: " << channel.name;
        }
        this->num_channels++;
        if (err < 0) {
            LOG(ERROR) << "[ni.reader] failed while configuring channel " << channel.name;
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
            this->dmx->CfgSampClkTiming(this->task_handle,
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
            this->dmx->ReadDigitalLines(
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

    timer.wait(breaker);
    auto [d, err] = data_queue.dequeue();
    if (!err)
        return std::make_pair(std::move(f), freighter::Error(
                                  driver::TEMPORARY_HARDWARE_ERROR,
                                  "Failed to read data from queue"));

    uint64_t incr = ((d.tf - d.t0) / this->num_samples_per_channel);
    uint64_t data_index = 0;

    for (int i = 0; i < num_channels; i++) {
        if (!this->digital_config.channels[i].enabled) continue;
        if (this->digital_config.channels[i].channel_type == "index") {
            auto t = synnax::Series(synnax::TIMESTAMP, this->num_samples_per_channel);
            for (uint64_t j = 0; j < d.samples_read_per_channel; ++j)
                t.write(d.t0 + j * incr);

            f.add(this->digital_config.channels[i].channel_key, std::move(t));
            continue;
        }
        auto series = synnax::Series(synnax::SY_UINT8, d.samples_read_per_channel);

        for (int j = 0; j < d.samples_read_per_channel; j++)
            series.write((uint8_t) d.digital_data[data_index + j]);

        f.add(this->digital_config.channels[i].channel_key, std::move(series));
        data_index++;
    }
    return std::make_pair(std::move(f), freighter::NIL);
}

int ni::DigitalReadSource::validate_channels() {
    for (auto &channel: digital_config.channels) {
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
            this->ctx->set_state({
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
