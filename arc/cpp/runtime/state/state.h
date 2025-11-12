//
// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include <functional>
#include <memory>
#include <optional>
#include <unordered_map>
#include <vector>

#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/xmemory/local_shared.h"

#include "arc/cpp/ir/ir.h"

namespace arc::runtime::state {
using Series = xmemory::local_shared<telem::Series>;

struct Value {
    Series data;
    Series time;
};

struct ChannelDigest {
    arc::ChannelKey key;
    telem::DataType data_type;
    arc::ChannelKey index;
};

struct Config {
    ir::IR ir;
    std::vector<ChannelDigest> channels;
};

class State;

class Node {
    friend class State;

    std::vector<arc::ir::Edge> inputs;
    std::vector<arc::ir::Handle> outputs;
    State &state_;

    struct InputEntry {
        Series data;
        Series time;
        telem::TimeStamp last_timestamp;
        bool consumed;

        InputEntry():
            data(nullptr), time(nullptr), last_timestamp(0), consumed(false) {}
    };

    std::vector<InputEntry> accumulated;
    std::vector<Series> aligned_data;
    std::vector<Series> aligned_time;
    std::vector<Value *> input_sources;
    std::vector<Value *> output_cache;

    Node(
        std::vector<arc::ir::Edge> inputs,
        std::vector<arc::ir::Handle> outputs,
        State &state,
        std::vector<InputEntry> accumulated,
        std::vector<Series> aligned_data,
        std::vector<Series> aligned_time,
        std::vector<Value *> input_sources,
        std::vector<Value *> output_cache
    ):
        inputs(std::move(inputs)),
        outputs(std::move(outputs)),
        state_(state),
        accumulated(std::move(accumulated)),
        aligned_data(std::move(aligned_data)),
        aligned_time(std::move(aligned_time)),
        input_sources(std::move(input_sources)),
        output_cache(std::move(output_cache)) {}

public:
    bool refresh_inputs();

    [[nodiscard]] const Series &input(const size_t param_index) const {
        return aligned_data[param_index];
    }

    [[nodiscard]] const Series &input_time(const size_t param_index) const {
        return aligned_time[param_index];
    }

    [[nodiscard]] Series &output(const size_t param_index) const {
        return output_cache[param_index]->data;
    }

    [[nodiscard]] Series &output_time(const size_t param_index) const {
        return output_cache[param_index]->time;
    }

    void write_channel(arc::ChannelKey key, const Series &data, const Series &time);

    template<typename T>
    void write_channel_value(arc::ChannelKey key, T value, telem::TimeStamp timestamp) {
        const auto data_series = std::make_shared<telem::Series>(
            telem::Series(1, &value)
        );
        const auto time_series = std::make_shared<telem::Series>(telem::TIMESTAMP_T, 1);
        time_series->write(timestamp);
        write_channel(key, data_series, time_series);
    }
};

class State {
    friend class Node;

    Config cfg;
    std::unordered_map<ir::Handle, Value, ir::Handle::Hasher> outputs;
    std::unordered_map<ChannelKey, ChannelKey> indexes;
    std::unordered_map<ChannelKey, std::vector<Series>> reads;
    std::unordered_map<ChannelKey, Series> writes;

    void write_channel(ChannelKey key, const Series &data, const Series &time);

public:
    explicit State(const Config &cfg);

    Node node(const std::string &key);

    void ingest(const telem::Frame &frame);

    std::vector<std::pair<ChannelKey, Series>> flush_writes();

    void clear_reads();

    template<typename T>
    void write_channel_value(arc::ChannelKey key, T value, telem::TimeStamp timestamp) {
        const auto data_series = std::make_shared<telem::Series>(
            telem::Series(1, &value)
        );
        const auto time_series = std::make_shared<telem::Series>(telem::TIMESTAMP_T, 1);
        time_series->write(timestamp);
        write_channel(key, data_series, time_series);
    }
};

} // namespace arc::runtime::state
