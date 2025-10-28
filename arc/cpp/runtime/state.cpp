// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include "arc/cpp/runtime/state.h"

namespace arc {

// ChannelBuffer implementation

ChannelBuffer::ChannelBuffer(telem::DataType dt)
    : expected_type_(dt) {}

void ChannelBuffer::update(std::shared_ptr<telem::Series> data,
                           std::shared_ptr<telem::Series> time) {
    data_ = std::move(data);
    time_ = std::move(time);
}

telem::SampleValue ChannelBuffer::latest_value() const {
    if (!data_ || data_->empty()) {
        return telem::SampleValue{};
    }

    // Return last element based on data type
    const size_t last_idx = data_->size() - 1;
    return data_->at(last_idx);
}

// State implementation
// (Constructor is now in header as template)

void State::register_channel(ChannelKey key, telem::DataType dt) {
    channels_.emplace(key, ChannelBuffer(dt));
}

void State::register_node(const NodeMetadata& metadata) {
    nodes_[metadata.key] = metadata;

    // Pre-allocate output storage for this node's output parameters
    for (const auto& param : metadata.output_params) {
        Handle handle{metadata.key, param};
        outputs_[handle] = ValuePair{};
    }
}

void State::add_edge(const Edge& edge) {
    edges_.push_back(edge);
}

ValuePair& State::get_output(const Handle& handle) {
    auto it = outputs_.find(handle);
    if (it == outputs_.end()) {
        // Lazy initialization if not pre-allocated
        outputs_[handle] = ValuePair{};
        return outputs_[handle];
    }
    return it->second;
}

const ValuePair& State::get_output(const Handle& handle) const {
    auto it = outputs_.find(handle);
    if (it == outputs_.end()) {
        static const ValuePair empty{};
        return empty;
    }
    return it->second;
}

void State::process_input_queue() {
    ChannelUpdate update;
    while (input_queue_.pop(update)) {
        auto it = channels_.find(update.channel_id);
        if (it != channels_.end()) {
            // Update shared_ptr - atomic refcounting, RT-safe
            it->second.update(std::move(update.data), std::move(update.time));
        }
    }
}

std::pair<telem::SampleValue, xerrors::Error>
State::read_channel(ChannelKey key) const {
    auto it = channels_.find(key);
    if (it == channels_.end()) {
        return {{}, xerrors::Error("arc.state.channel_not_found")};
    }

    if (!it->second.has_data()) {
        return {{}, xerrors::Error("arc.state.no_data")};
    }

    return {it->second.latest_value(), xerrors::NIL};
}

std::vector<Edge> State::incoming_edges(const std::string& node_id) const {
    std::vector<Edge> result;
    for (const auto& edge : edges_) {
        if (edge.target.node == node_id) {
            result.push_back(edge);
        }
    }
    return result;
}

std::vector<Edge> State::outgoing_edges(const std::string& node_id) const {
    std::vector<Edge> result;
    for (const auto& edge : edges_) {
        if (edge.source.node == node_id) {
            result.push_back(edge);
        }
    }
    return result;
}

const NodeMetadata* State::get_node_metadata(const std::string& node_id) const {
    auto it = nodes_.find(node_id);
    if (it == nodes_.end()) {
        return nullptr;
    }
    return &it->second;
}

}  // namespace arc
