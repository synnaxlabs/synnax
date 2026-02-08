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
#include <memory>
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
#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/stl/series/state.h"
#include "arc/cpp/stl/stateful/state.h"
#include "arc/cpp/stl/str/state.h"
#include "arc/cpp/types/types.h"

namespace arc::runtime::state {
using Series = xmemory::local_shared<telem::Series>;

struct AuthorityChange {
    std::optional<types::ChannelKey> channel_key;
    uint8_t authority;
};

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

    /// @brief Sets the current node key on the parent state for stateful variable
    /// isolation.
    void set_current_node_key(const std::string &key);
};

class State {
    friend class Node;
    Config cfg;
    std::vector<Value> values;
    std::unordered_map<ir::Handle, size_t> value_index;
    std::unordered_map<types::ChannelKey, types::ChannelKey> indexes;
    std::unordered_map<types::ChannelKey, std::vector<Series>> reads;
    std::unordered_map<types::ChannelKey, Series> writes;

    /// @brief Per-module state slices.
    std::shared_ptr<stl::str::State> str_state;
    std::shared_ptr<stl::series::State> series_state;
    std::shared_ptr<stl::stateful::Variables> variables;

    /// @brief Callback for reporting warnings (e.g., data drops).
    errors::Handler error_handler;

    /// @brief Buffered authority changes from set_authority nodes.
    std::vector<AuthorityChange> authority_changes;

public:
    void write_channel(types::ChannelKey key, const Series &data, const Series &time);
    std::pair<telem::MultiSeries, bool> read_channel(types::ChannelKey key);
    explicit State(
        const Config &cfg,
        errors::Handler error_handler = errors::noop_handler
    );
    std::pair<Node, xerrors::Error> node(const std::string &key);
    void ingest(const telem::Frame &frame);
    std::vector<std::pair<types::ChannelKey, Series>> flush();

    /// @brief Buffers an authority change request for later flushing.
    /// If channel_key is nullopt, the change applies to all write channels.
    void set_authority(std::optional<types::ChannelKey> channel_key, uint8_t authority);

    /// @brief Returns and clears all buffered authority changes.
    std::vector<AuthorityChange> flush_authority_changes();

    /// @brief Clears all persistent state, resetting the runtime to initial conditions.
    void reset();

    /// @brief Sets the current node key for stateful variable isolation.
    /// Must be called before each WASM function invocation.
    void set_current_node_key(const std::string &key) {
        this->variables->set_current_node_key(key);
    }

    /// @brief Accessors for per-module state slices.
    std::shared_ptr<stl::str::State> get_str_state() const { return this->str_state; }

    std::shared_ptr<stl::series::State> get_series_state() const {
        return this->series_state;
    }

    std::shared_ptr<stl::stateful::Variables> get_variables() const {
        return this->variables;
    }
};
}
