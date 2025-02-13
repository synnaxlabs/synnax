// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/ni/ni.h"

#include "driver/ni/writer.h"

namespace ni {
template<typename T>
StateSource<T>::StateSource(
    float state_rate,
    std::vector<synnax::ChannelKey> &state_index_keys,
    std::vector<synnax::ChannelKey> &state_channel_keys
) {
    this->state_rate.value = state_rate;
    this->state_index_keys = state_index_keys;

    for (auto &key: state_channel_keys)
        this->state_map[key] = 0;
    this->timer = loop::Timer(this->state_rate);
}

template<typename T>
std::pair<synnax::Frame, xerrors::Error> StateSource<T>::read(breaker::Breaker &breaker) {
    std::unique_lock<std::mutex> lock(this->state_mutex);
    this->timer.wait(breaker);
    waiting_reader.wait_for(lock, this->state_rate.period().chrono());
    return std::make_pair(this->get_state(), xerrors::NIL);
}

template<typename T>
synnax::Frame StateSource<T>::get_state() {
    auto frame_size = this->state_map.size() + this->state_index_keys.size();
    auto state_frame = synnax::Frame(frame_size);
    
    // Create timestamp series for each index key
    for (auto &index_key : this->state_index_keys) {
        auto timestamp_series = telem::Series(
            telem::TimeStamp::now().value,
            telem::TIMESTAMP
        );
        state_frame.add(index_key, timestamp_series);
    }

    // Add each state value
    for (auto &[key, value]: this->state_map) {
        auto value_series = telem::Series(value);
        state_frame.emplace(key, std::move(value_series));
    }

    return state_frame;
}

template<typename T>
void StateSource<T>::update_state(
    std::queue<synnax::ChannelKey> &modified_state_keys,
    std::queue<T> &modified_state_values
) {
    std::unique_lock<std::mutex> lock(this->state_mutex);
    while (!modified_state_keys.empty()) {
        this->state_map[modified_state_keys.front()] = static_cast<T>(modified_state_values.front());
        modified_state_keys.pop();
        modified_state_values.pop();
    }
    waiting_reader.notify_one();
}

///@brief  Template instantiations to tell the compiler which versions to generate
template class StateSource<uint8_t>; // For DigitalStateSource
template class StateSource<double>; // For AnalogStateSource
} // namespace ni
