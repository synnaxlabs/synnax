// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include <array>
#include <memory>
#include <string>
#include <unordered_map>
#include <vector>

#include "arc/cpp/runtime/core/types.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/xerrors/errors.h"

namespace arc {

/// @brief Channel buffer storing shared ownership of Series data.
///
/// No pre-allocation needed - I/O thread allocates Series, RT thread stores
/// shared_ptr and reads directly. Atomic refcounting ensures thread safety.
class ChannelBuffer {
    std::shared_ptr<telem::Series> data_;  /// Channel data
    std::shared_ptr<telem::Series> time_;  /// Timestamps
    telem::DataType expected_type_;        /// Expected data type

public:
    /// @brief Construct channel buffer with expected data type.
    /// @param dt Expected data type for validation.
    explicit ChannelBuffer(telem::DataType dt);

    /// @brief Update channel data from I/O thread (via queue).
    /// @param data New data series (shared ownership transferred).
    /// @param time New time series (shared ownership transferred).
    /// @note RT-safe: shared_ptr assignment uses atomic refcounting.
    void update(std::shared_ptr<telem::Series> data,
                std::shared_ptr<telem::Series> time);

    /// @brief Read latest value from channel data.
    /// @return Latest sample value, or empty if no data.
    /// @note RT-safe: Series::at() performs no allocations.
    telem::SampleValue latest_value() const;

    /// @brief Get shared pointer to data series.
    /// @return Shared pointer to data (may be null).
    std::shared_ptr<telem::Series> data() const { return data_; }

    /// @brief Get shared pointer to time series.
    /// @return Shared pointer to time (may be null).
    std::shared_ptr<telem::Series> time() const { return time_; }

    /// @brief Check if channel has data available.
    /// @return true if data exists and is non-empty.
    bool has_data() const { return data_ && !data_->empty(); }

    /// @brief Get expected data type for this channel.
    /// @return Expected data type.
    telem::DataType expected_type() const { return expected_type_; }
};

/// @brief Arc runtime state management.
///
/// Manages node outputs, channel I/O, state variables, and dataflow graph.
/// Designed for multi-threaded operation with RT-safe guarantees:
///   - I/O thread pushes ChannelUpdate via input queue
///   - RT thread processes queue and stores shared_ptr<Series>
///   - RT thread reads from Series with zero allocations
///   - RT thread writes via output queue to I/O thread
///
/// Key architectural distinction:
///   - Node outputs: Stored globally by Handle (node + param)
///   - Channels: External Synnax I/O, separate from node graph
///   - Edges: Connect node outputs to other node inputs
class State {
    /// Node output storage (Handle → ValuePair)
    std::unordered_map<Handle, ValuePair, HandleHash> outputs_;

    /// Edge graph (dataflow connections)
    std::vector<Edge> edges_;

    /// Node metadata
    std::unordered_map<std::string, NodeMetadata> nodes_;

    /// Channel storage (external Synnax channels)
    std::unordered_map<ChannelKey, ChannelBuffer> channels_;

    /// State variable storage (fixed-size for RT-safety)
    static constexpr size_t MAX_STATE_VARS = 4096;
    std::array<StateKey, MAX_STATE_VARS> state_keys_;
    std::array<telem::SampleValue, MAX_STATE_VARS> state_values_;
    std::array<bool, MAX_STATE_VARS> state_used_;

    /// Queue references (non-owning)
    queue::SPSC<ChannelUpdate>& input_queue_;
    queue::SPSC<ChannelOutput>& output_queue_;

public:
    /// @brief Construct state with queue references.
    /// @param input_queue Queue for incoming channel updates from I/O thread.
    /// @param output_queue Queue for outgoing channel outputs to I/O thread.
    State(queue::SPSC<ChannelUpdate>* input_queue,
          queue::SPSC<ChannelOutput>* output_queue)
        : input_queue_(*input_queue), output_queue_(*output_queue) {
        state_used_.fill(false);
    }

    // === Initialization (non-RT) ===

    /// @brief Register a channel with expected data type.
    /// @param key Channel identifier.
    /// @param dt Expected data type.
    void register_channel(ChannelKey key, telem::DataType dt);

    /// @brief Register a node with metadata.
    /// @param metadata Node metadata (key, type, params, channels).
    /// @note Pre-allocates output storage for this node's output parameters.
    void register_node(const NodeMetadata& metadata);

    /// @brief Add an edge to the dataflow graph.
    /// @param edge Edge connecting source handle to target handle.
    void add_edge(const Edge& edge);

    // === RT Thread Operations ===

    /// @brief Get mutable reference to node output storage.
    /// @param handle Output handle (node + param).
    /// @return Reference to ValuePair for this output.
    /// @note RT-safe: No allocation, returns existing or creates empty slot.
    ValuePair& get_output(const Handle& handle);

    /// @brief Get const reference to node output storage.
    /// @param handle Output handle (node + param).
    /// @return Const reference to ValuePair for this output.
    const ValuePair& get_output(const Handle& handle) const;

    /// @brief Process incoming channel updates from input queue.
    /// @note Called at start of each RT cycle. Updates shared_ptr references.
    void process_input_queue();

    /// @brief Read channel value for WASM host functions.
    /// @param key Channel identifier.
    /// @return Latest value and error status.
    /// @note RT-safe: Series::at() performs no allocations.
    std::pair<telem::SampleValue, xerrors::Error>
    read_channel(ChannelKey key) const;

    /// @brief Write channel value from WASM host functions.
    /// @tparam T Value type.
    /// @param key Channel identifier.
    /// @param value Value to write.
    /// @return Error status.
    /// @note RT-safe: Pushes to lock-free output queue.
    template <typename T>
    xerrors::Error write_channel(ChannelKey key, T value);

    /// @brief Load state variable with initialization value.
    /// @tparam T Value type.
    /// @param key State key (funcID << 32 | varID).
    /// @param init_value Initialization value if not found.
    /// @return Current or initialized value.
    /// @note RT-safe: Fixed-size hash table with bounded lookup.
    template <typename T>
    T load_state(StateKey key, T init_value);

    /// @brief Store state variable.
    /// @tparam T Value type.
    /// @param key State key (funcID << 32 | varID).
    /// @param value Value to store.
    /// @note RT-safe: Fixed-size hash table with bounded insertion.
    template <typename T>
    void store_state(StateKey key, T value);

    // === Graph Queries ===

    /// @brief Get incoming edges for a node.
    /// @param node_id Node identifier.
    /// @return Vector of edges feeding into this node.
    std::vector<Edge> incoming_edges(const std::string& node_id) const;

    /// @brief Get outgoing edges from a node.
    /// @param node_id Node identifier.
    /// @return Vector of edges from this node to others.
    std::vector<Edge> outgoing_edges(const std::string& node_id) const;

    /// @brief Get node metadata.
    /// @param node_id Node identifier.
    /// @return Pointer to metadata (null if not found).
    const NodeMetadata* get_node_metadata(const std::string& node_id) const;
};

// Template implementations

template <typename T>
xerrors::Error State::write_channel(ChannelKey key, T value) {
    auto it = channels_.find(key);
    if (it == channels_.end())
        return xerrors::Error("arc.state.channel_not_found");

    ChannelOutput output{key, telem::SampleValue(value),
                        telem::TimeStamp::now()};
    if (!output_queue_.push(std::move(output))) {
        return xerrors::Error("arc.state.output_queue_full");
    }

    return xerrors::NIL;
}

template <typename T>
T State::load_state(StateKey key, T init_value) {
    const size_t idx = key % MAX_STATE_VARS;
    for (size_t i = 0; i < MAX_STATE_VARS; i++) {
        const size_t probe = (idx + i) % MAX_STATE_VARS;
        if (state_used_[probe]) {
            if (state_keys_[probe] == key) {
                return std::get<T>(state_values_[probe]);
            }
        } else {
            // Not found, initialize
            state_keys_[probe] = key;
            state_values_[probe] = init_value;
            state_used_[probe] = true;
            return init_value;
        }
    }
    // Table full (should not happen with proper sizing)
    return init_value;
}

template <typename T>
void State::store_state(StateKey key, T value) {
    const size_t idx = key % MAX_STATE_VARS;
    for (size_t i = 0; i < MAX_STATE_VARS; i++) {
        const size_t probe = (idx + i) % MAX_STATE_VARS;
        if (state_used_[probe] && state_keys_[probe] == key) {
            state_values_[probe] = value;
            return;
        }
        if (!state_used_[probe]) {
            state_keys_[probe] = key;
            state_values_[probe] = value;
            state_used_[probe] = true;
            return;
        }
    }
}

}  // namespace arc
