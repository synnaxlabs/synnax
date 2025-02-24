// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/ni/writer.h"

template<typename T>
std::pair<Frame, xerrors::Error> ni::WriteSinkStateSource<T>::read(breaker::Breaker &breaker) {
    auto fr = synnax::Frame(this->state, this->state.size() + this->state_indexes.size());
    this->state_timer.wait(breaker);
    if (!this->state_indexes.empty()) {
        const auto idx_ser = telem::Series(telem::TimeStamp::now());
        for (const auto idx: this->state_indexes) fr.emplace(idx, idx_ser.deep_copy());
    }
    return {std::move(fr), xerrors::NIL};
}

template<typename T>
xerrors::Error ni::WriteSinkStateSource<T>::write(const synnax::Frame &frame) {
    if (const auto err = this->write_ni(this->format_data(frame))) return err;
    for (const auto &[key, series]: frame) {
        const auto state_key = this->cfg.cmd_to_state[key];
        this->state[state_key] = series.at(0);
    }
    return xerrors::NIL;
}

template<typename T>
T * ni::WriteSinkStateSource<T>::format_data(const synnax::Frame &frame) {
    for (const auto &[key, series]: frame) {
        auto it = this->cfg.channels.find(key);
        if (it == this->cfg.channels.end()) continue;
        auto buf = this->write_buffer.get();
        buf[it->second->index] = telem::cast_numeric_sample_value<T>(series.at_numeric(0));
    }
    return this->write_buffer.get();
}

xerrors::Error ni::AnalogWriteSink::write_ni(double *data) const {
    this->dmx->WriteAnalogF64(
        this->task_handle,
        1,
        1,
        10.0,
        DAQmx_Val_GroupByChannel,
        data,
        nullptr,
        nullptr
    );
    return xerrors::NIL;
}

xerrors::Error ni::DigitalWriteSink::write_ni(uint8_t * data) const {
    this->dmx->WriteDigitalLines(
        this->task_handle,
        1,
        1,
        10.0,
        DAQmx_Val_GroupByChannel,
        data,
        nullptr,
        nullptr
    );
    return xerrors::NIL;
}

