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
using Series = xmemory::local_shared<telem::Series>;

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
    /// Non-owning reference to parent State for channel I/O. Always valid after
    /// construction via State::node(). Null only in error-return case.
    State *state_ptr = nullptr;
    std::vector<arc::ir::Edge> inputs;
    std::vector<arc::ir::Handle> outputs;

    struct InputEntry {
        Series data;
        Series time;
        telem::TimeStamp last_timestamp;
        bool consumed;

        InputEntry(): data(), time(), last_timestamp(0), consumed(false) {}
    };

    std::vector<InputEntry> accumulated;
    std::vector<Series> aligned_data;
    std::vector<Series> aligned_time;
    std::vector<Value *> input_sources;
    std::vector<Value *> output_cache;
    Node() = default;

    Node(
        State *state_ptr,
        std::vector<arc::ir::Edge> inputs,
        std::vector<arc::ir::Handle> outputs,
        std::vector<InputEntry> accumulated,
        std::vector<Series> aligned_data,
        std::vector<Series> aligned_time,
        std::vector<Value *> input_sources,
        std::vector<Value *> output_cache
    ):
        state_ptr(state_ptr),
        inputs(std::move(inputs)),
        outputs(std::move(outputs)),
        accumulated(std::move(accumulated)),
        aligned_data(std::move(aligned_data)),
        aligned_time(std::move(aligned_time)),
        input_sources(std::move(input_sources)),
        output_cache(std::move(output_cache)) {}

public:
    bool refresh_inputs();

    [[nodiscard]] const Series &input(const size_t param_index) const {
        return this->aligned_data[param_index];
    }

    [[nodiscard]] const Series &input_time(const size_t param_index) const {
        return this->aligned_time[param_index];
    }

    [[nodiscard]] Series &output(const size_t param_index) {
        return this->output_cache[param_index]->data;
    }

    [[nodiscard]] Series &output_time(const size_t param_index) {
        return this->output_cache[param_index]->time;
    }

    /// Reads buffered data and time series from a channel. Returns (data, index_data,
    /// ok). If the channel has an associated index, both data and time are returned.
    std::tuple<telem::MultiSeries, telem::MultiSeries, bool>
    read_chan(types::ChannelKey key);

    /// Writes data and time series to a channel buffer.
    void write_chan(types::ChannelKey key, const Series &data, const Series &time);

    /// @brief Checks if the output at the given param name is truthy.
    /// Returns false if the param doesn't exist, if the output is empty,
    /// or if the last element is zero. Returns true otherwise.
    [[nodiscard]] bool is_output_truthy(const std::string &param_name) const {
        for (size_t i = 0; i < outputs.size(); ++i) {
            if (outputs[i].param == param_name) {
                const auto *series = output_cache[i]->data.get();
                if (series == nullptr) return false;
                return is_series_truthy(*series);
            }
        }
        return false;
    }

    /// @brief Checks if a series is truthy by examining its last element.
    /// Empty series are falsy. A series with a last element of zero is falsy.
    [[nodiscard]] static bool is_series_truthy(const telem::Series &series) {
        if (series.size() == 0) return false;
        const auto last_value = series.at(-1);
        return std::visit(
            [](const auto &v) -> bool {
                using T = std::decay_t<decltype(v)>;
                if constexpr (std::is_same_v<T, std::string>)
                    return !v.empty();
                else if constexpr (std::is_same_v<T, telem::TimeStamp>)
                    return v.nanoseconds() != 0;
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
    std::unordered_map<ir::Handle, Value> outputs;
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
