// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include "arc/cpp/runtime/state/node_state.h"

namespace arc {

NodeState::NodeState(State *state,
                     std::string node_id,
                     std::vector<Edge> inputs,
                     std::vector<Handle> outputs)
    : state_(*state),
      node_id_(std::move(node_id)),
      inputs_(std::move(inputs)),
      outputs_(std::move(outputs)) {
    // Pre-allocate alignment buffers
    accumulated_.resize(inputs_.size());
    aligned_data_.resize(inputs_.size());
    aligned_time_.resize(inputs_.size());
}

bool NodeState::refresh_inputs() {
    // 1. Accumulate new data from source outputs beyond watermark
    for (size_t i = 0; i < inputs_.size(); i++) {
        const Edge& edge = inputs_[i];
        const ValuePair& source = state_.get_output(edge.source);

        // Skip if source has no data
        if (!source.data || source.data->empty()) {
            continue;
        }
        if (!source.time || source.time->empty()) {
            continue;
        }

        // Get last timestamp from source
        auto last_ts = source.time->at<telem::TimeStamp>(-1);

        // Skip if already processed (at or before watermark)
        if (last_ts <= accumulated_[i].watermark) {
            continue;
        }

        // Accumulate new data beyond watermark
        accumulated_[i].data.push_back(source.data);
        accumulated_[i].time.push_back(source.time);
    }

    // 2. Check all inputs have data (not ready if any empty)
    for (const auto& entry : accumulated_) {
        if (entry.empty()) {
            return false;  // Not ready
        }
    }

    // 3. Find trigger input (earliest new timestamp beyond watermark)
    int trigger_idx = -1;
    int trigger_series_idx = -1;
    telem::TimeStamp trigger_ts{0};

    for (size_t i = 0; i < inputs_.size(); i++) {
        for (size_t j = 0; j < accumulated_[i].time.size(); j++) {
            const auto& time_series = accumulated_[i].time[j];
            if (time_series->empty()) continue;

            auto ts = time_series->at<telem::TimeStamp>(-1);

            // Check if this timestamp is beyond watermark
            if (ts > accumulated_[i].watermark) {
                if (trigger_idx == -1 || ts < trigger_ts) {
                    trigger_idx = i;
                    trigger_series_idx = j;
                    trigger_ts = ts;
                }
                break;  // Only check first new series per input
            }
        }
    }

    if (trigger_idx == -1) {
        return false;  // No new data beyond watermarks
    }

    // 4. Align all inputs to trigger timestamp
    for (size_t i = 0; i < inputs_.size(); i++) {
        if (static_cast<int>(i) == trigger_idx) {
            // Use trigger series directly
            aligned_data_[i] = accumulated_[i].data[trigger_series_idx];
            aligned_time_[i] = accumulated_[i].time[trigger_series_idx];
            accumulated_[i].watermark = trigger_ts;
        } else {
            // Reuse latest data for catch-up inputs
            size_t latest = accumulated_[i].data.size() - 1;
            aligned_data_[i] = accumulated_[i].data[latest];
            aligned_time_[i] = accumulated_[i].time[latest];
            accumulated_[i].watermark = trigger_ts;
        }
    }

    // 5. Prune consumed data (remove series with timestamp <= watermark)
    for (size_t i = 0; i < inputs_.size(); i++) {
        auto& entry = accumulated_[i];

        // Find first series with timestamp > watermark
        size_t keep_idx = 0;
        for (size_t j = 0; j < entry.time.size(); j++) {
            if (entry.time[j]->empty()) continue;
            auto ts = entry.time[j]->at<telem::TimeStamp>(-1);
            if (ts > entry.watermark) {
                keep_idx = j;
                break;
            }
        }

        // Prune earlier series
        if (keep_idx > 0) {
            entry.data.erase(entry.data.begin(), entry.data.begin() + keep_idx);
            entry.time.erase(entry.time.begin(), entry.time.begin() + keep_idx);
        }
    }

    return true;
}

const telem::Series& NodeState::input(size_t param_index) const {
    if (param_index >= aligned_data_.size() || !aligned_data_[param_index]) {
        static const telem::Series empty{std::vector<uint8_t>{}};
        return empty;
    }
    return *aligned_data_[param_index];
}

const telem::Series& NodeState::input_time(size_t param_index) const {
    if (param_index >= aligned_time_.size() || !aligned_time_[param_index]) {
        static const telem::Series empty{std::vector<int64_t>{}};
        return empty;
    }
    return *aligned_time_[param_index];
}

telem::Series* NodeState::output(size_t param_index) {
    if (param_index >= outputs_.size()) {
        return nullptr;
    }

    const Handle& handle = outputs_[param_index];
    ValuePair& vp = state_.get_output(handle);

    // Lazy initialize with empty vector
    if (!vp.data) {
        vp.data = std::make_shared<telem::Series>(std::vector<uint8_t>{});
    }

    return vp.data.get();
}

telem::Series* NodeState::output_time(size_t param_index) {
    if (param_index >= outputs_.size()) {
        return nullptr;
    }

    const Handle& handle = outputs_[param_index];
    ValuePair& vp = state_.get_output(handle);

    // Lazy initialize with empty timestamp vector
    if (!vp.time) {
        vp.time = std::make_shared<telem::Series>(std::vector<telem::TimeStamp>{});
    }

    return vp.time.get();
}

std::pair<telem::SampleValue, xerrors::Error>
NodeState::read_channel(ChannelKey key) const {
    return state_.read_channel(key);
}

}  // namespace arc
