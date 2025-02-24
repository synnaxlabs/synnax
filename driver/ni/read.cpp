// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.


/// std
#include <utility>
#include <chrono>

/// external
#include "nlohmann/json.hpp"

/// module
#include "x/cpp/telem/telem.h"

/// internal
#include "driver/errors/errors.h"
#include "driver/ni/reader.h"

using json = nlohmann::json;

void ni::DigitalReadSource::acquire_data() {
    this->dmx->StartTask(this->task_handle);
    while (this->sample_thread_breaker.running()) {
        int32 bytes_per_sample;
        DataPacket data_packet;
        data_packet.digital_data.resize(this->cfg.buffer_size);
        data_packet.t0 = telem::TimeStamp::now();
        this->sample_timer.wait(sample_thread_breaker);
        this->dmx->ReadDigitalLines(
            this->task_handle,
            this->cfg.samples_per_channel,
            -1,
            DAQmx_Val_GroupByChannel,
            data_packet.digital_data.data(),
            data_packet.digital_data.size(),
            &data_packet.samples_read_per_channel,
            &bytes_per_sample,
            nullptr
        );
        data_packet.tf = telem::TimeStamp::now();
        queue.enqueue(data_packet);
    }
    this->dmx->StartTask(this->task_handle);
}

std::pair<synnax::Frame, xerrors::Error> ni::DigitalReadSource::read(
    breaker::Breaker &breaker
) {
    if (!sample_thread_breaker.running())
        sample_thread = std::thread([this] { this->acquire_data(); });
    auto f = synnax::Frame(this->cfg.channels.size());
    this->timer.wait(breaker);
    auto [d, err] = queue.dequeue();
    if (!err)
        return std::make_pair(std::move(f), xerrors::Error(
                                  driver::TEMPORARY_HARDWARE_ERROR,
                                  "Failed to read data from queue"));
    const auto buf = d.digital_data.data();
    uint64_t data_index = 0;
    const size_t count = this->cfg.samples_per_channel;
    for (const auto &ch: this->cfg.channels) {
        const auto start = data_index * count;
        f.emplace(ch.synnax_key, telem::Series(buf + start, count, telem::UINT8_T));
        data_index++;
    }
    if (!this->cfg.indexes.empty()) {
        const auto index_data = telem::Series::linspace(d.t0, d.tf, count);
        for (const auto &idx: this->cfg.indexes)
            f.emplace(idx, std::move(index_data.deep_copy()));
    }
    return std::make_pair(std::move(f), xerrors::NIL);
}

void ni::AnalogReadSource::acquire_data() {
    this->dmx->StartTask(this->task_handle);
    while (this->sample_thread_breaker.running()) {
        DataPacket data_packet;
        data_packet.analog_data.resize(this->cfg.buffer_size);
        data_packet.t0 = telem::TimeStamp::now();
        this->dmx->ReadAnalogF64(
            this->task_handle,
            this->cfg.samples_per_channel,
            -1,
            DAQmx_Val_GroupByChannel,
            data_packet.analog_data.data(),
            data_packet.analog_data.size(),
            &data_packet.samples_read_per_channel,
            nullptr
        );
        data_packet.tf = telem::TimeStamp::now();
        this->queue.enqueue(std::move(data_packet));
    }
    this->dmx->StopTask(this->task_handle);
}

std::pair<synnax::Frame, xerrors::Error> ni::AnalogReadSource::read(
    breaker::Breaker &breaker
) {
    if (!sample_thread_breaker.running())
        sample_thread = std::thread([this] { this->acquire_data(); });
    auto [d, ok] = this->queue.dequeue();
    if (!ok) return std::make_pair(synnax::Frame(), xerrors::NIL);
    auto f = synnax::Frame(this->cfg.channels.size());
    const size_t count = this->cfg.samples_per_channel;
    size_t data_index = 0;
    const auto buf = d.analog_data.data();
    for (const auto &ch: this->cfg.channels) {
        auto series = telem::Series(ch->ch.data_type, count);
        const int start = data_index * count;
        if (series.data_type == telem::FLOAT64_T) series.write(buf + start, count);
        else
            for (int i = 0; i < count; ++i)
                series.write(static_cast<float>(buf[start + i]));
        f.emplace(ch->synnax_key, std::move(series));
        data_index++;
    }
    if (!this->cfg.indexes.empty()) {
        const auto index_data = telem::Series::linspace(d.t0, d.tf, count);
        for (const auto &idx: this->cfg.indexes)
            f.emplace(idx, std::move(index_data.deep_copy()));
    }
    return std::make_pair(std::move(f), xerrors::NIL);
}
