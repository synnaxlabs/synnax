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
    // now parse the channels
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
                    this->reader_config.channels.push_back(config);
                });
    if (!parser.ok()) LOG(ERROR) << "Failed to parse channels for task " << this->
                      reader_config.task_name;
}

int ni::DigitalReadSource::create_channels() {
    int err = 0;
    auto channels = this->reader_config.channels;
    for (auto &channel: channels) {
        if (channel.channel_type != "index") {
            err = this->check_ni_error(
                ni::NiDAQmxInterface::CreateDIChan(task_handle, channel.name.c_str(),
                                                   "", DAQmx_Val_ChanPerLine));
            VLOG(1) << "Channel name: " << channel.name;
        }
        this->numChannels++;
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
        this->numSamplesPerChannel = 1;
    } else {
        if (this->check_ni_error(ni::NiDAQmxInterface::CfgSampClkTiming(this->task_handle,
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
        this->numSamplesPerChannel = std::floor(
            this->reader_config.sample_rate.value / this->reader_config.stream_rate.
            value);
    }
    this->bufferSize = this->numChannels * this->numSamplesPerChannel;
    this->timer = loop::Timer(this->reader_config.stream_rate);
    return 0;
}


void ni::DigitalReadSource::acquire_data() {
    while (this->breaker.running()) {
        int32 numBytesPerSamp;
        DataPacket data_packet;
        data_packet.digital_data.resize(this->bufferSize);
        data_packet.t0 = (uint64_t) ((synnax::TimeStamp::now()).value);
        // sleep per sample rate
        auto samp_period = this->reader_config.sample_rate.period().chrono();
        std::this_thread::sleep_for(samp_period);
        if (this->check_ni_error(
            ni::NiDAQmxInterface::ReadDigitalLines(
                this->task_handle, // task handle
                this->numSamplesPerChannel, // numSampsPerChan
                -1, // timeout
                DAQmx_Val_GroupByChannel, // dataLayout
                data_packet.digital_data.data(),// readArray
                data_packet.digital_data.size(), // arraySizeInSamps
                &data_packet.samplesReadPerChannel, // sampsPerChanRead
                &numBytesPerSamp, // numBytesPerSamp
                NULL))) {
            this->log_error(
                "failed while reading digital data for task " + this->reader_config.
                task_name);
        }
        data_packet.tf = (uint64_t) ((synnax::TimeStamp::now()).value);
        data_queue.enqueue(data_packet);
    }
}

std::pair<synnax::Frame, freighter::Error> ni::DigitalReadSource::read(
    breaker::Breaker &breaker) {
    synnax::Frame f = synnax::Frame(numChannels);

    // sleep per stream rate
    timer.wait(breaker);
    auto [d, err] = data_queue.dequeue();
    if (!err)
        return std::make_pair(std::move(f), freighter::Error(
                                  driver::TEMPORARY_HARDWARE_ERROR,
                                  "Failed to read data from queue"));
    // interpolate  timestamps between the initial and final timestamp to ensure non-overlapping timestamps between batched reads
    uint64_t incr = ((d.tf - d.t0) / this->numSamplesPerChannel);
    // Construct and populate index channel
    std::vector<std::uint64_t> time_index(this->numSamplesPerChannel);
    for (uint64_t i = 0; i < d.samplesReadPerChannel; ++i)
        time_index[i] = d.t0 + (std::uint64_t) (incr * i);

    auto s = d.samplesReadPerChannel;
    // Construct and populate synnax frame
    uint64_t data_index = 0;
    for (int i = 0; i < numChannels; i++) {
        if (this->reader_config.channels[i].channel_type == "index") {
            f.add(this->reader_config.channels[i].channel_key,
                  synnax::Series(time_index, synnax::TIMESTAMP));
            continue;
        }
        auto series = synnax::Series(synnax::UINT8,s);
        // copy data into vector
        for (int j = 0; j < d.samplesReadPerChannel; j++) 
            series.write((uint8_t) d.digital_data[data_index + j]);

        f.add(this->reader_config.channels[i].channel_key, std::move(series));
        data_index++;
    }
    return std::make_pair(std::move(f), freighter::NIL);
}

int ni::DigitalReadSource::validate_channels() {
    LOG(INFO) << "[NI Reader] Validating channels for task " << this->reader_config.
            task_name;
    for (auto &channel: this->reader_config.channels) {
        if (channel.channel_type == "index") {
            if (channel.channel_key == 0) {
                LOG(ERROR) << "[NI Reader] Index channel key is 0";
                return -1;
            }
            continue;
        }
        // if not index, make sure channel type is valid
        auto [channel_info, err] = this->ctx->client->channels.retrieve(
            channel.channel_key);
        if(channel_info.data_type != synnax::FLOAT32 || channel_info.data_type != synnax::FLOAT64) {
            LOG(ERROR) << "[NI Reader] Channel " << channel.name << " is not of type FLOAT32";
            return -1;
        }
    }
    return 0;
}