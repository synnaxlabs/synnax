// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstring>
#include <utility>

#include "x/cpp/errors/errors.h"

#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/types/types.h"

namespace arc::runtime::state {
Series parse_default_value(
    const std::optional<x::telem::SampleValue> &value,
    const types::Type &type
) {
    auto data_type = type.telem();
    if (value.has_value()) {
        auto casted = data_type.cast(*value);
        return x::mem::make_local_shared<x::telem::Series>(casted);
    }
    switch (type.kind) {
        case types::Kind::I8:
            return x::mem::make_local_shared<x::telem::Series>(static_cast<int8_t>(0));
        case types::Kind::I16:
            return x::mem::make_local_shared<x::telem::Series>(static_cast<int16_t>(0));
        case types::Kind::I32:
            return x::mem::make_local_shared<x::telem::Series>(static_cast<int32_t>(0));
        case types::Kind::I64:
            return x::mem::make_local_shared<x::telem::Series>(static_cast<int64_t>(0));
        case types::Kind::U8:
            return x::mem::make_local_shared<x::telem::Series>(static_cast<uint8_t>(0));
        case types::Kind::U16:
            return x::mem::make_local_shared<x::telem::Series>(
                static_cast<uint16_t>(0)
            );
        case types::Kind::U32:
            return x::mem::make_local_shared<x::telem::Series>(
                static_cast<uint32_t>(0)
            );
        case types::Kind::U64:
            return x::mem::make_local_shared<x::telem::Series>(
                static_cast<uint64_t>(0)
            );
        case types::Kind::F32:
            return x::mem::make_local_shared<x::telem::Series>(0.0f);
        case types::Kind::F64:
            return x::mem::make_local_shared<x::telem::Series>(0.0);
        case types::Kind::Invalid:
        case types::Kind::String:
        case types::Kind::Chan:
        case types::Kind::Series:
            return x::mem::make_local_shared<x::telem::Series>(data_type, 0);
    }
    return x::mem::make_local_shared<x::telem::Series>(data_type, 0);
}

State::State(const Config &cfg, errors::Handler error_handler):
    cfg(cfg),
    strings(std::make_shared<stl::str::State>()),
    series(std::make_shared<stl::series::State>()),
    vars(std::make_shared<stl::stateful::Variables>()),
    error_handler(std::move(error_handler)) {
    size_t total = 0;
    for (const auto &node: cfg.ir.nodes)
        total += node.outputs.size();
    this->values.reserve(total);

    for (const auto &digest: cfg.channels)
        this->indexes[digest.key] = digest.index;

    for (const auto &node: cfg.ir.nodes) {
        for (const auto &output: node.outputs) {
            ir::Handle handle(node.key, output.name);
            this->value_index[handle] = this->values.size();
            this->values.emplace_back(
                Value{
                    x::mem::local_shared<x::telem::Series>(output.type.telem(), 0),
                    x::mem::local_shared<x::telem::Series>(x::telem::TIMESTAMP_T, 0)
                }
            );
        }
    }
}

std::pair<Node, x::errors::Error> State::node(const std::string &key) {
    const auto &ir_node = this->cfg.ir.node(key);
    const size_t num_inputs = ir_node.inputs.size();
    std::vector<ir::Edge> inputs(num_inputs);
    std::vector<Series> aligned_data(num_inputs);
    std::vector<Series> aligned_time(num_inputs);
    std::vector<Node::InputEntry> accumulated(num_inputs);
    std::vector<size_t> input_source_idx(num_inputs);

    for (size_t i = 0; i < num_inputs; i++)
        aligned_time[i] = x::mem::make_local_shared<x::telem::Series>(
            x::telem::TIMESTAMP_T,
            0
        );

    for (size_t i = 0; i < num_inputs; i++) {
        const auto &param = ir_node.inputs[i];
        ir::Handle target_handle(key, param.name);
        if (auto edge = this->cfg.ir.edge_to(target_handle)) {
            inputs[i] = *edge;
            const auto &source_handle = edge->source;
            auto idx_iter = this->value_index.find(source_handle);
            if (idx_iter != this->value_index.end()) {
                size_t idx = idx_iter->second;
                aligned_data[i] = x::mem::make_local_shared<x::telem::Series>(
                    this->values[idx].data->data_type(),
                    0
                );
                input_source_idx[i] = idx;
                accumulated[i].source = idx;
                accumulated[i].last_timestamp = x::telem::TimeStamp(0);
                accumulated[i].consumed = true;
            }
        } else {
            ir::Handle synthetic_handle("__default_" + key + "_" + param.name, "out");
            inputs[i] = ir::Edge(synthetic_handle, target_handle);

            auto data_series = parse_default_value(param.value, param.type);
            auto time_series = x::mem::make_local_shared<x::telem::Series>(
                x::telem::TimeStamp(0)
            );

            aligned_data[i] = data_series;
            aligned_time[i] = time_series;

            accumulated[i].data = data_series;
            accumulated[i].time = time_series;
            accumulated[i].last_timestamp = x::telem::TimeStamp(0);
            accumulated[i].consumed = false;

            if (!this->value_index.contains(synthetic_handle)) {
                this->value_index[synthetic_handle] = this->values.size();
                this->values.emplace_back(Value{data_series, time_series});
            }
            input_source_idx[i] = this->value_index[synthetic_handle];
            accumulated[i].source = input_source_idx[i];
        }
    }

    std::vector<ir::Handle> output_handles;
    std::vector<size_t> output_idx;
    std::unordered_map<std::string, size_t> output_name_idx;
    for (size_t i = 0; i < ir_node.outputs.size(); i++) {
        const auto &output_param = ir_node.outputs[i];
        ir::Handle handle(key, output_param.name);
        output_handles.push_back(handle);
        output_idx.push_back(this->value_index[handle]);
        output_name_idx[output_param.name] = i;
    }

    return {
        Node(
            *this,
            std::move(inputs),
            std::move(output_handles),
            std::move(input_source_idx),
            std::move(output_idx),
            std::move(output_name_idx),
            std::move(accumulated),
            std::move(aligned_data),
            std::move(aligned_time)
        ),
        x::errors::NIL
    };
}

void State::ingest(const x::telem::Frame &frame) {
    for (size_t i = 0; i < frame.size(); i++)
        reads[frame.channels->at(i)].push_back(
            x::mem::local_shared(std::move(frame.series->at(i)))
        );
}

std::vector<std::pair<types::ChannelKey, Series>> State::flush() {
    for (auto &series_vec: reads | std::views::values) {
        if (series_vec.size() <= 1) continue;
        auto last = std::move(series_vec.back());
        series_vec.clear();
        series_vec.push_back(std::move(last));
    }
    this->series->clear();
    this->strings->clear();

    std::vector<std::pair<types::ChannelKey, Series>> result;
    result.reserve(writes.size());
    for (const auto &[key, data]: writes)
        result.push_back({key, data});
    writes.clear();
    return result;
}

void State::reset() {
    this->reads.clear();
    this->writes.clear();
    this->strings->clear();
    this->series->clear();
    this->vars->reset();
}

void State::set_authority(
    std::optional<types::ChannelKey> channel_key,
    const uint8_t authority
) {
    authority_changes.push_back({std::move(channel_key), authority});
}

std::vector<AuthorityChange> State::flush_authority_changes() {
    std::vector<AuthorityChange> result;
    result.swap(authority_changes);
    return result;
}

void State::write_channel(
    const types::ChannelKey key,
    const Series &data,
    const Series &time
) {
    writes[key] = data;
    if (const auto idx_iter = indexes.find(key);
        idx_iter != indexes.end() && idx_iter->second != 0)
        writes[idx_iter->second] = time;
}

bool Node::refresh_inputs() {
    if (this->inputs.empty()) return true;
    bool has_unconsumed = false;
    for (size_t i = 0; i < this->inputs.size(); i++) {
        const Value &src = this->state.values[this->accumulated[i].source];
        const auto *time_ptr = src.time.get();
        const auto *data_ptr = src.data.get();
        if (time_ptr != nullptr && data_ptr != nullptr && time_ptr->size() > 0 &&
            data_ptr->size() > 0) {
            if (auto ts = time_ptr->at<x::telem::TimeStamp>(-1);
                ts > this->accumulated[i].last_timestamp) {
                this->accumulated[i].data = src.data;
                this->accumulated[i].time = src.time;
                this->accumulated[i].last_timestamp = ts;
                this->accumulated[i].consumed = false;
            }
        }
        if (this->accumulated[i].data == nullptr || this->accumulated[i].data->empty())
            return false;
        if (!this->accumulated[i].consumed) has_unconsumed = true;
    }
    if (!has_unconsumed) return false;
    for (size_t i = 0; i < this->inputs.size(); i++) {
        this->aligned_data[i] = this->accumulated[i].data;
        this->aligned_time[i] = this->accumulated[i].time;
        this->accumulated[i].consumed = true;
    }
    return true;
}

std::pair<x::telem::MultiSeries, bool>
State::read_channel(const types::ChannelKey key) {
    const auto it = reads.find(key);
    if (it == reads.end() || it->second.empty())
        return {x::telem::MultiSeries{}, false};
    x::telem::MultiSeries ms;
    for (const auto &s: it->second)
        ms.series.push_back(s->deep_copy());
    return {std::move(ms), true};
}

std::tuple<x::telem::MultiSeries, x::telem::MultiSeries, bool>
Node::read_chan(const types::ChannelKey key) const {
    auto [data, ok] = this->state.read_channel(key);
    if (!ok) return {x::telem::MultiSeries{}, x::telem::MultiSeries{}, false};
    const auto index_it = this->state.indexes.find(key);
    if (index_it == this->state.indexes.end() || index_it->second == 0)
        return {std::move(data), x::telem::MultiSeries{}, !data.series.empty()};
    auto [time, time_ok] = this->state.read_channel(index_it->second);
    if (!time_ok) return {x::telem::MultiSeries{}, x::telem::MultiSeries{}, false};
    return {
        std::move(data),
        std::move(time),
        !data.series.empty() && !time.series.empty()
    };
}

void Node::write_chan(
    const types::ChannelKey key,
    const Series &data,
    const Series &time
) const {
    this->state.write_channel(key, data, time);
}

const Series &Node::input_time(const size_t param_index) const {
    return this->aligned_time[param_index];
}
Series &Node::output(const size_t param_index) const {
    return this->state.values[this->output_idx[param_index]].data;
}

Series &Node::output_time(const size_t param_index) const {
    return this->state.values[this->output_idx[param_index]].time;
}

bool Node::is_output_truthy(const std::string &param_name) const {
    const auto it = this->output_name_idx.find(param_name);
    if (it == this->output_name_idx.end()) return false;
    const auto *s = this->state.values[this->output_idx[it->second]].data.get();
    return s != nullptr && this->is_series_truthy(*s);
}

void Node::set_current_node_key(const std::string &key) {
    this->state.set_current_node_key(key);
}

}
