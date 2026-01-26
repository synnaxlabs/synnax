// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <string>
#include <tuple>
#include <unordered_map>
#include <vector>

#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/xerrors/errors.h"
#include "x/cpp/xmemory/local_shared.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/types/types.h"

namespace arc::runtime::state {
using Series = xmemory::local_shared<telem::Series>;

/// Generates a unique key for stateful variables from function ID and variable ID.
inline uint64_t state_key(const uint32_t func_id, const uint32_t var_id) {
    return static_cast<uint64_t>(func_id) << 32 | static_cast<uint64_t>(var_id);
}

struct Value {
    Series data;
    Series time;
};

struct ChannelDigest {
    types::ChannelKey key;
    telem::DataType data_type;
    types::ChannelKey index;
};

struct Config {
    ir::IR ir;
    std::vector<ChannelDigest> channels;
};

class State;

class Node {
    friend class State;
    State &state;
    std::vector<ir::Edge> inputs;
    std::vector<ir::Handle> outputs;
    std::vector<size_t> input_source_idx;
    std::vector<size_t> output_idx;
    std::unordered_map<std::string, size_t> output_name_idx;

    struct InputEntry {
        size_t source;
        Series data;
        Series time;
        telem::TimeStamp last_timestamp{0};
        bool consumed{true};
    };

    std::vector<InputEntry> accumulated;
    std::vector<Series> aligned_data;
    std::vector<Series> aligned_time;

    Node(
        State &state,
        std::vector<ir::Edge> inputs,
        std::vector<ir::Handle> outputs,
        std::vector<size_t> input_source_idx,
        std::vector<size_t> output_idx,
        std::unordered_map<std::string, size_t> output_name_idx,
        std::vector<InputEntry> accumulated,
        std::vector<Series> aligned_data,
        std::vector<Series> aligned_time
    ):
        state(state),
        inputs(std::move(inputs)),
        outputs(std::move(outputs)),
        input_source_idx(std::move(input_source_idx)),
        output_idx(std::move(output_idx)),
        output_name_idx(std::move(output_name_idx)),
        accumulated(std::move(accumulated)),
        aligned_data(std::move(aligned_data)),
        aligned_time(std::move(aligned_time)) {}

public:
    Node(const Node &) = delete;
    Node &operator=(const Node &) = delete;
    Node &operator=(Node &&) = delete;
    Node(Node &&) noexcept = default;

    bool refresh_inputs();

    [[nodiscard]] const Series &input(const size_t param_index) const {
        return this->aligned_data[param_index];
    }

    [[nodiscard]] const Series &input_time(size_t param_index) const;

    [[nodiscard]] Series &output(size_t param_index) const;
    [[nodiscard]] Series &output_time(size_t param_index) const;

    /// Reads buffered data and time series from a channel. Returns (data, index_data,
    /// ok). If the channel has an associated index, both data and time are returned.
    std::tuple<telem::MultiSeries, telem::MultiSeries, bool>
    read_chan(types::ChannelKey key) const;

    /// Writes data and time series to a channel buffer.
    void
    write_chan(types::ChannelKey key, const Series &data, const Series &time) const;

    [[nodiscard]] bool is_output_truthy(const std::string &param_name) const;

    /// @brief Checks if a series is truthy by examining its last element.
    /// Empty series are falsy. A series with a last element of zero is falsy.
    [[nodiscard]] static bool is_series_truthy(const telem::Series &series) {
        if (series.size() == 0) return false;
        const auto last_value = series.at(-1);
        return std::visit(
            []<typename T0>(const T0 &v) -> bool {
                using T = std::decay_t<T0>;
                if constexpr (std::is_same_v<T, std::string>)
                    return !v.empty();
                else
                    return v != 0;
            },
            last_value
        );
    }

    /// @brief Resets accumulated input state for runtime restart.
    void reset() {
        for (auto &entry: this->accumulated) {
            entry.last_timestamp = telem::TimeStamp(0);
            entry.consumed = true;
        }
    }
};

class State {
    friend class Node;
    Config cfg;
    std::vector<Value> values;
    std::unordered_map<ir::Handle, size_t> value_index;
    std::unordered_map<types::ChannelKey, types::ChannelKey> indexes;
    std::unordered_map<types::ChannelKey, std::vector<Series>> reads;
    std::unordered_map<types::ChannelKey, Series> writes;

    /// @brief Transient string handles - cleared each execution cycle.
    std::unordered_map<uint32_t, std::string> strings;
    uint32_t string_handle_counter = 1;

    /// @brief Transient series handles - cleared each execution cycle.
    std::unordered_map<uint32_t, telem::Series> series_handles;
    uint32_t series_handle_counter = 1;

    /// @brief Persistent stateful variable storage - keyed by state_key(func_id,
    /// var_id).
    std::unordered_map<uint64_t, uint8_t> var_u8;
    std::unordered_map<uint64_t, uint16_t> var_u16;
    std::unordered_map<uint64_t, uint32_t> var_u32;
    std::unordered_map<uint64_t, uint64_t> var_u64;
    std::unordered_map<uint64_t, int8_t> var_i8;
    std::unordered_map<uint64_t, int16_t> var_i16;
    std::unordered_map<uint64_t, int32_t> var_i32;
    std::unordered_map<uint64_t, int64_t> var_i64;
    std::unordered_map<uint64_t, float> var_f32;
    std::unordered_map<uint64_t, double> var_f64;
    std::unordered_map<uint64_t, std::string> var_string;
    std::unordered_map<uint64_t, telem::Series> var_series;

public:
    void write_channel(types::ChannelKey key, const Series &data, const Series &time);
    std::pair<telem::MultiSeries, bool> read_channel(types::ChannelKey key);
    explicit State(const Config &cfg);
    std::pair<Node, xerrors::Error> node(const std::string &key);
    void ingest(const telem::Frame &frame);
    std::vector<std::pair<types::ChannelKey, Series>> flush();

    /// @brief Clears all persistent state, resetting the runtime to initial conditions.
    void reset();

    /// @brief Creates a string handle from raw memory pointer and length.
    uint32_t string_from_memory(const uint8_t *data, uint32_t len);

    /// @brief Creates a string handle from a C++ string.
    uint32_t string_create(const std::string &str);

    /// @brief Gets the string value for a handle. Returns empty string if not found.
    std::string string_get(uint32_t handle) const;

    /// @brief Checks if a string handle exists.
    bool string_exists(uint32_t handle) const;

    /// @brief Gets a series by handle. Returns nullptr if not found.
    telem::Series *series_get(uint32_t handle);
    const telem::Series *series_get(uint32_t handle) const;

    /// @brief Stores a series and returns its handle.
    uint32_t series_store(telem::Series series);

#define DECLARE_VAR_OPS(suffix, cpptype)                                               \
    cpptype var_load_##suffix(uint32_t func_id, uint32_t var_id, cpptype init_value);  \
    void var_store_##suffix(uint32_t func_id, uint32_t var_id, cpptype value);

    DECLARE_VAR_OPS(u8, uint8_t)
    DECLARE_VAR_OPS(u16, uint16_t)
    DECLARE_VAR_OPS(u32, uint32_t)
    DECLARE_VAR_OPS(u64, uint64_t)
    DECLARE_VAR_OPS(i8, int8_t)
    DECLARE_VAR_OPS(i16, int16_t)
    DECLARE_VAR_OPS(i32, int32_t)
    DECLARE_VAR_OPS(i64, int64_t)
    DECLARE_VAR_OPS(f32, float)
    DECLARE_VAR_OPS(f64, double)

#undef DECLARE_VAR_OPS

    uint32_t var_load_str(uint32_t func_id, uint32_t var_id, uint32_t init_handle);
    void var_store_str(uint32_t func_id, uint32_t var_id, uint32_t str_handle);

    uint32_t var_load_series(uint32_t func_id, uint32_t var_id, uint32_t init_handle);
    void var_store_series(uint32_t func_id, uint32_t var_id, uint32_t handle);
};
}
