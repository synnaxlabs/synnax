// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <vector>

#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/runtime/core/types.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/xerrors/errors.h"

namespace arc {

/// @brief Input accumulation buffer for temporal alignment.
///
/// Stores accumulated input data from a source node's output.
/// Tracks watermark to prevent re-processing consumed data.
struct InputEntry {
    std::vector<std::shared_ptr<telem::Series>> data;  ///< Accumulated data series
    std::vector<std::shared_ptr<telem::Series>> time;  ///< Accumulated timestamp series
    telem::TimeStamp watermark{0};                     ///< Last processed timestamp

    InputEntry() = default;

    bool empty() const { return data.empty(); }

    void clear() {
        data.clear();
        time.clear();
    }
};

/// @brief Per-node facade for state operations with temporal alignment.
///
/// Provides a scoped interface for a single node to interact with the global
/// State. Implements the watermark-based temporal alignment algorithm from
/// the Go runtime.
///
/// Key architectural pattern:
/// - Inputs are EDGES (from other node outputs), not channel keys
/// - Outputs are HANDLES (this node + param), not channel keys
/// - refresh_inputs() performs temporal alignment across multi-rate inputs
/// - Parameter-indexed I/O: input(i), output(i)
/// - Channel I/O is separate (for external Synnax channels)
class NodeState {
    State& state_;                       ///< Reference to global state (non-owning)
    std::string node_id_;                ///< Node identifier

    // Input sources (edges from other nodes)
    std::vector<Edge> inputs_;

    // Output handles (this node's outputs)
    std::vector<Handle> outputs_;

    // Temporal alignment state (one per input)
    std::vector<InputEntry> accumulated_;           ///< Buffered input data
    std::vector<std::shared_ptr<telem::Series>> aligned_data_;   ///< Aligned data (after refresh)
    std::vector<std::shared_ptr<telem::Series>> aligned_time_;   ///< Aligned timestamps

public:
    /// @brief Construct node state for a specific node.
    /// @param state Pointer to global state (not owned).
    /// @param node_id Node identifier.
    /// @param inputs Incoming edges (source node outputs).
    /// @param outputs Output handles (this node's output parameters).
    NodeState(State *state,
              std::string node_id,
              std::vector<Edge> inputs,
              std::vector<Handle> outputs);

    // === Temporal Alignment ===

    /// @brief Refresh inputs with temporal alignment.
    ///
    /// Implements watermark-based multi-rate temporal alignment:
    /// 1. Accumulate new data from source outputs beyond watermark
    /// 2. Check all inputs have data (not ready if any empty)
    /// 3. Find trigger input (earliest new timestamp)
    /// 4. Align all inputs to trigger timestamp
    /// 5. Prune consumed data
    ///
    /// @return true if new aligned data is available, false if not ready.
    /// @note RT-safe: Uses pre-allocated buffers.
    bool refresh_inputs();

    // === Parameter-Indexed I/O ===

    /// @brief Get aligned input data by parameter index.
    /// @param param_index Input parameter index (0-based).
    /// @return Aligned input series (valid after refresh_inputs).
    const telem::Series& input(size_t param_index) const;

    /// @brief Get aligned input timestamps by parameter index.
    /// @param param_index Input parameter index (0-based).
    /// @return Aligned timestamp series.
    const telem::Series& input_time(size_t param_index) const;

    /// @brief Get mutable output data by parameter index.
    /// @param param_index Output parameter index (0-based).
    /// @return Pointer to output series for writing.
    telem::Series* output(size_t param_index);

    /// @brief Get mutable output timestamps by parameter index.
    /// @param param_index Output parameter index (0-based).
    /// @return Pointer to output timestamp series.
    telem::Series* output_time(size_t param_index);

    // === Channel I/O (External Synnax) ===

    /// @brief Read from external channel (for WASM channel_read_* functions).
    /// @param key Channel identifier.
    /// @return Value and error status.
    std::pair<telem::SampleValue, xerrors::Error>
    read_channel(ChannelKey key) const;

    /// @brief Write to external channel (for WASM channel_write_* functions).
    /// @tparam T Value type.
    /// @param key Channel identifier.
    /// @param value Value to write.
    /// @return Error status.
    template <typename T>
    xerrors::Error write_channel(ChannelKey key, T value);

    // === State Variables ===

    /// @brief Load a state variable (for WASM state_load_* functions).
    /// @tparam T Value type.
    /// @param var_id Variable identifier (within this node's function scope).
    /// @param init_value Initialization value if not found.
    /// @return Current or initialized value.
    template <typename T>
    T load_state_var(uint32_t var_id, T init_value);

    /// @brief Store a state variable (for WASM state_store_* functions).
    /// @tparam T Value type.
    /// @param var_id Variable identifier (within this node's function scope).
    /// @param value Value to store.
    template <typename T>
    void store_state_var(uint32_t var_id, T value);

    // === Accessors ===

    /// @brief Get node identifier.
    /// @return Node ID string.
    const std::string& node_id() const { return node_id_; }

    /// @brief Get number of inputs.
    /// @return Number of input parameters.
    size_t num_inputs() const { return inputs_.size(); }

    /// @brief Get number of outputs.
    /// @return Number of output parameters.
    size_t num_outputs() const { return outputs_.size(); }
};

// Template implementations

template <typename T>
xerrors::Error NodeState::write_channel(ChannelKey key, T value) {
    return state_.write_channel(key, value);
}

template <typename T>
T NodeState::load_state_var(uint32_t var_id, T init_value) {
    // Create state key using node's function ID (hash of node_id for now)
    // In full implementation, this would come from compiled WASM metadata
    const uint32_t func_id = std::hash<std::string>{}(node_id_) & 0xFFFFFFFF;
    const StateKey key = make_state_key(func_id, var_id);
    return state_.load_state(key, init_value);
}

template <typename T>
void NodeState::store_state_var(uint32_t var_id, T value) {
    const uint32_t func_id = std::hash<std::string>{}(node_id_) & 0xFFFFFFFF;
    const StateKey key = make_state_key(func_id, var_id);
    state_.store_state(key, value);
}

}  // namespace arc
