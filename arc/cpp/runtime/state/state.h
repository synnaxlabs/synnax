// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

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
using Series = x::mem::local_shared<telem::Series>;

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
};

class State {
    friend class Node;
    Config cfg;
    std::vector<Value> values;
    std::unordered_map<ir::Handle, size_t> value_index;
    std::unordered_map<types::ChannelKey, types::ChannelKey> indexes;
    std::unordered_map<types::ChannelKey, std::vector<Series>> reads;
    std::unordered_map<types::ChannelKey, Series> writes;

    void write_channel(types::ChannelKey key, const Series &data, const Series &time);
    std::pair<telem::MultiSeries, bool> read_channel(types::ChannelKey key);

public:
    explicit State(const Config &cfg);
    std::pair<Node, xerrors::Error> node(const std::string &key);
    void ingest(const telem::Frame &frame);
    std::vector<std::pair<types::ChannelKey, Series>> flush_writes();
    void clear_reads();
};
}
