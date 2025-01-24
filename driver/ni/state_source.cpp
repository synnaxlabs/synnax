#include "driver/ni/ni.h"

namespace ni {

template<typename T>
StateSource<T>::StateSource(
    float state_rate,
    synnax::ChannelKey &state_index_key,
    std::vector<synnax::ChannelKey> &state_channel_keys
) : state_rate(synnax::Rate(state_rate)), 
    state_index_key(state_index_key) {
    for (const auto &key : state_channel_keys) {
        state_map[key] = T();
    }
}

template<typename T>
std::pair<synnax::Frame, freighter::Error> StateSource<T>::read(breaker::Breaker &breaker) {
    std::unique_lock<std::mutex> lock(state_mutex);
    waiting_reader.wait_for(lock, std::chrono::milliseconds(timer.period_ms()));
    return {get_state(), freighter::Error()};
}

template<typename T>
synnax::Frame StateSource<T>::get_state() {
    auto frame = synnax::Frame();
    frame.set_index({state_index_key}, {timer.now()});
    for (const auto &[key, value] : state_map) {
        frame.set_value(key, value);
    }
    return frame;
}

template<typename T>
void StateSource<T>::update_state(
    std::queue<synnax::ChannelKey> &modified_state_keys,
    std::queue<T> &modified_state_values
) {
    std::lock_guard<std::mutex> lock(state_mutex);
    while (!modified_state_keys.empty()) {
        auto key = modified_state_keys.front();
        auto value = modified_state_values.front();
        state_map[key] = value;
        modified_state_keys.pop();
        modified_state_values.pop();
    }
    waiting_reader.notify_one();
}

// Explicit template instantiations
template class StateSource<uint8_t>;
template class StateSource<double>;

} // namespace ni 