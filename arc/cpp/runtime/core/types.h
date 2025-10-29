// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <memory>
#include <string>
#include <vector>

#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

namespace arc {

/// @brief State variable key type.
/// Encodes funcID in upper 32 bits, varID in lower 32 bits.
using StateKey = std::uint64_t;

/// @brief Create a state key from function ID and variable ID.
/// @param func_id Function/stage identifier.
/// @param var_id Variable identifier within the function.
/// @return Composite state key.
constexpr StateKey make_state_key(std::uint32_t func_id, std::uint32_t var_id) {
    return (static_cast<StateKey>(func_id) << 32) | var_id;
}

/// @brief Extract function ID from state key.
/// @param key Composite state key.
/// @return Function ID (upper 32 bits).
constexpr std::uint32_t state_key_func_id(StateKey key) {
    return static_cast<std::uint32_t>(key >> 32);
}

/// @brief Extract variable ID from state key.
/// @param key Composite state key.
/// @return Variable ID (lower 32 bits).
constexpr std::uint32_t state_key_var_id(StateKey key) {
    return static_cast<std::uint32_t>(key & 0xFFFFFFFF);
}

/// @brief Message for channel data updates from I/O thread to RT thread.
///
/// Contains shared ownership of Series objects. I/O thread allocates and
/// moves into queue, RT thread receives and stores in State.
struct ChannelUpdate {
    ChannelKey channel_id; ///< Target channel ID
    std::shared_ptr<telem::Series> data; ///< Channel data (shared ownership)
    std::shared_ptr<telem::Series> time; ///< Timestamps (shared ownership)

    ChannelUpdate() = default;
    ChannelUpdate(
        ChannelKey id,
        std::shared_ptr<telem::Series> d,
        std::shared_ptr<telem::Series> t
    ):
        channel_id(id), data(std::move(d)), time(std::move(t)) {}
};

/// @brief Message for channel data output from RT thread to I/O thread.
///
/// Contains single scalar values written by WASM. RT thread writes to queue,
/// I/O thread reads and sends to Synnax cluster.
struct ChannelOutput {
    ChannelKey channel_id; ///< Source channel ID
    telem::SampleValue value; ///< Output value
    telem::TimeStamp timestamp; ///< Output timestamp

    ChannelOutput() = default;
    ChannelOutput(ChannelKey id, telem::SampleValue v, telem::TimeStamp ts):
        channel_id(id), value(std::move(v)), timestamp(ts) {}
};

/// @brief Value pair for node outputs (data + timestamps).
///
/// Stores the output data and timestamps for a node's output parameter.
/// Both data and time use shared_ptr for zero-copy sharing between nodes.
struct ValuePair {
    std::shared_ptr<telem::Series> data; ///< Output data series
    std::shared_ptr<telem::Series> time; ///< Output timestamp series

    ValuePair() = default;
    ValuePair(std::shared_ptr<telem::Series> d, std::shared_ptr<telem::Series> t):
        data(std::move(d)), time(std::move(t)) {}
};

/// @brief Node metadata for graph traversal and initialization.
///
/// Contains structural information about a node extracted from IR.
/// Used during initialization to build the dataflow graph.
struct NodeMetadata {
    std::string key; ///< Node identifier
    std::string type; ///< Function type name
    std::vector<std::string> input_params; ///< Input parameter names (ordered)
    std::vector<std::string> output_params; ///< Output parameter names (ordered)
    std::vector<ChannelKey> read_channels; ///< External channels read
    std::vector<ChannelKey> write_channels; ///< External channels written

    NodeMetadata() = default;
    explicit NodeMetadata(std::string k): key(std::move(k)) {}
};

} // namespace arc
