// Copyright 2024 Synnax Labs, Inc.
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
    synnax::ChannelKey &state_index_key,
    std::vector<synnax::ChannelKey> &state_channel_keys
) {
    this->state_rate.value = state_rate;
    // start the periodic thread
    this->state_index_key = state_index_key;

    // initialize all states to 0
    for (auto &key: state_channel_keys)
        this->state_map[key] = 0;
    this->timer = loop::Timer(this->state_rate);
}

template<typename T>
std::pair<synnax::Frame, freighter::Error> StateSource<T>::read(breaker::Breaker &breaker) {
    std::unique_lock<std::mutex> lock(this->state_mutex);
    // sleep for state period
    this->timer.wait(breaker);
    waiting_reader.wait_for(lock, this->state_rate.period().chrono());
    return std::make_pair(this->get_state(), freighter::NIL);
}

template<typename T>
synnax::Frame StateSource<T>::get_state() {
    // frame size = # monitored states + 1 state index channel
    auto frame_size =this->state_map.size() + 1;
    auto state_frame = synnax::Frame(frame_size);
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

template<typename T>
void StateSource<T>::update_state(
    std::queue<synnax::ChannelKey> &modified_state_keys,
 std::queue<T> &modified_state_values
) {
    LOG(INFO) << "StateSource::update_state: updating state"; // REMOVE DEBUG
    std::unique_lock<std::mutex> lock(this->state_mutex);
    while (!modified_state_keys.empty()) {
        LOG(INFO) << "updating key: " << modified_state_keys.front(); // REMOVE DEBUG
        LOG(INFO) << "updating value: " << modified_state_values.front(); // REMOVE DEBUG
        this->state_map[modified_state_keys.front()] = static_cast<T>(modified_state_values.front());
        modified_state_keys.pop();
        modified_state_values.pop();
    }
    waiting_reader.notify_one();
}

template class StateSource<uint8_t>;  // For DigitalStateSource
template class StateSource<double>;   // For AnalogStateSource

} // namespace ni 