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

void ni::DigitalReadSource::parseChannels(config::Parser &parser) {
    VLOG(1) << "[ni.reader] Parsing Channels for task " << this->reader_config.
            task_name;
    // now parse the channels
    parser.iter("channels",
            [&](config::Parser &channel_builder) {

            ni::ChannelConfig config;
            // digital channel names are formatted: <device_name>/port<port_number>/line<line_number>
            std::string port = "port" + std::to_string(
                channel_builder.required<std::uint64_t>("port"));
            std::string line = "line" + std::to_string(
                channel_builder.required<std::uint64_t>("line"));

            config.name = (this->reader_config.device_name + "/" + port + "/" + line);

            config.channel_key = channel_builder.required<uint32_t>("channel");
            this->reader_config.channels.push_back(config);
        });
    if(!parser.ok()) LOG(ERROR) << "Failed to parse channels for task " << this->reader_config.task_name;
 }

int ni::DigitalReadSource::createChannels() {
    int err = 0;
    auto channels = this->reader_config.channels;
    for (auto &channel: channels) {
        if (channel.channel_type != "index") {
            err = this->checkNIError(
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

int ni::DigitalReadSource::configureTiming() {
    if (this->reader_config.timing_source == "none") {
        // if timing is not enabled, implement timing in software
        this->numSamplesPerChannel = 1;
    } else {
        if (this->checkNIError(ni::NiDAQmxInterface::CfgSampClkTiming(this->task_handle,
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
            this->reader_config.sample_rate.value / this->reader_config.stream_rate.value);
    }
    this->bufferSize = this->numChannels * this->numSamplesPerChannel;
    return 0;
}


void ni::DigitalReadSource::acquireData() {
    while (this->running) {
        int32 numBytesPerSamp;
        DataPacket data_packet;
        data_packet.data = new uInt8[this->bufferSize];
        data_packet.t0 = (uint64_t) ((synnax::TimeStamp::now()).value);

        // sleep per sample rate
        auto samp_period = this->reader_config.stream_rate.period().chrono();
        std::this_thread::sleep_for(samp_period);

        if (this->checkNIError(
                ni::NiDAQmxInterface::ReadDigitalLines(
                this->task_handle, // task handle
                this->numSamplesPerChannel, // numSampsPerChan
                -1, // timeout
                DAQmx_Val_GroupByChannel, // dataLayout
                static_cast<uInt8 *>(data_packet.data), // readArray
                this->bufferSize, // arraySizeInSamps
                &data_packet.samplesReadPerChannel, // sampsPerChanRead
                &numBytesPerSamp, // numBytesPerSamp
                NULL))) {
                this->logError("failed while reading digital data for task " + this->reader_config.task_name);
        }
        data_packet.tf = (uint64_t) ((synnax::TimeStamp::now()).value);
        data_queue.enqueue(data_packet);
    }
}

std::pair<synnax::Frame, freighter::Error> ni::DigitalReadSource::read(
    breaker::Breaker &breaker) {
    synnax::Frame f = synnax::Frame(numChannels);
    // sleep per stream rate
    auto ns_period = this->reader_config.stream_rate.period().chrono();
    std::this_thread::sleep_for(ns_period);

    // take data off of queue
    auto [d, valid] = data_queue.dequeue();
    if (!valid){
        return std::make_pair(std::move(f), freighter::Error(
                                  driver::TEMPORARY_HARDWARE_ERROR,
                                  "Failed to read data from queue"));
                                  LOG(ERROR) << "Failed to read data from queue";
                                  }


    uInt8 *data = static_cast<uInt8 *>(d.data);
    // interpolate  timestamps between the initial and final timestamp to ensure non-overlapping timestamps between batched reads
    uint64_t incr = ((d.tf - d.t0) / this->numSamplesPerChannel);

    // Construct and populate index channel
    std::vector<std::uint64_t> time_index(this->numSamplesPerChannel);
    for (uint64_t i = 0; i < d.samplesReadPerChannel; ++i)
        time_index[i] = d.t0 + (std::uint64_t) (incr * i);

    // Construct and populate synnax frame
    uint64_t data_index = 0;
    for (int i = 0; i < numChannels; i++) {
        if (this->reader_config.channels[i].channel_type == "index") {
            f.add(this->reader_config.channels[i].channel_key,
                  synnax::Series(time_index, synnax::TIMESTAMP));
            continue;
        }

        std::vector<uint8_t> data_vec(d.samplesReadPerChannel);
        for (int j = 0; j < d.samplesReadPerChannel; j++)
            data_vec[j] = data[data_index * d.samplesReadPerChannel + j];

        f.add(this->reader_config.channels[i].channel_key,
              synnax::Series(data_vec, synnax::UINT8));
        data_index++;
    }
    delete[] data;
    return std::make_pair(std::move(f), freighter::NIL);
}
