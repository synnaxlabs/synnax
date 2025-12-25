// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <utility>

#include "x/cpp/xerrors/errors.h"

#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/types/types.h"

namespace arc::runtime::state {
Series parse_default_value(
    const std::optional<telem::SampleValue> &value,
    const types::Type &type
) {
    auto data_type = type.telem();
    if (value.has_value()) {
        auto casted = data_type.cast(*value);
        return xmemory::make_local_shared<telem::Series>(casted);
    }
    auto series = xmemory::make_local_shared<telem::Series>(data_type, 1);
    switch (type.kind) {
        case types::Kind::I8:
            series->write(static_cast<int8_t>(0));
            break;
        case types::Kind::I16:
            series->write(static_cast<int16_t>(0));
            break;
        case types::Kind::I32:
            series->write(static_cast<int32_t>(0));
            break;
        case types::Kind::I64:
            series->write(static_cast<int64_t>(0));
            break;
        case types::Kind::U8:
            series->write(static_cast<uint8_t>(0));
            break;
        case types::Kind::U16:
            series->write(static_cast<uint16_t>(0));
            break;
        case types::Kind::U32:
            series->write(static_cast<uint32_t>(0));
            break;
        case types::Kind::U64:
            series->write(static_cast<uint64_t>(0));
            break;
        case types::Kind::F32:
            series->write(0.0f);
            break;
        case types::Kind::F64:
            series->write(0.0);
            break;
        default:
            break;
    }
    return series;
}

State::State(const Config &cfg): cfg(cfg) {
    for (const auto &digest: cfg.channels)
        indexes[digest.key] = digest.index;
    for (const auto &node: cfg.ir.nodes) {
        for (const auto &output: node.outputs) {
            ir::Handle handle(node.key, output.name);
            outputs[handle] = Value{
                xmemory::local_shared<telem::Series>(output.type.telem(), 0),
                xmemory::local_shared<telem::Series>(telem::TIMESTAMP_T, 0)
            };
        }
    }
}

std::pair<Node, xerrors::Error> State::node(const std::string &key) {
    const auto &ir_node = this->cfg.ir.node(key);
    const size_t num_inputs = ir_node.inputs.size();
    std::vector<ir::Edge> inputs(num_inputs);
    std::vector<Series> aligned_data(num_inputs);
    std::vector<Series> aligned_time(num_inputs);
    std::vector<Node::InputEntry> accumulated(num_inputs);
    std::vector<Value *> input_sources(num_inputs);

    for (size_t i = 0; i < num_inputs; i++)
        aligned_time[i] = xmemory::make_local_shared<telem::Series>(
            telem::TIMESTAMP_T,
            0
        );

    for (size_t i = 0; i < num_inputs; i++) {
        const auto &param = ir_node.inputs[i];
        ir::Handle target_handle(key, param.name);
        auto edge = cfg.ir.edge_to(target_handle);

        if (edge) {
            inputs[i] = *edge;
            const auto &source_handle = edge->source;

            auto source_output_iter = outputs.find(source_handle);
            if (source_output_iter != outputs.end()) {
                aligned_data[i] = xmemory::make_local_shared<telem::Series>(
                    source_output_iter->second.data->data_type(),
                    0
                );
                input_sources[i] = &source_output_iter->second;
                accumulated[i].last_timestamp = telem::TimeStamp(0);
                accumulated[i].consumed = true;
            }
        } else {
            ir::Handle synthetic_handle("__default_" + key + "_" + param.name, "out");
            inputs[i] = ir::Edge(synthetic_handle, target_handle);

            auto data_series = parse_default_value(param.value, param.type);
            auto time_series = xmemory::make_local_shared<telem::Series>(
                telem::TIMESTAMP_T,
                1
            );
            time_series->write(telem::TimeStamp(0));

            aligned_data[i] = data_series;
            aligned_time[i] = time_series;

            accumulated[i].data = data_series;
            accumulated[i].time = time_series;
            accumulated[i].last_timestamp = telem::TimeStamp(0);
            accumulated[i].consumed = true;

            if (!this->outputs.contains(synthetic_handle))
                this->outputs[synthetic_handle] = Value{data_series, time_series};

            input_sources[i] = &this->outputs[synthetic_handle];
        }
    }

    // Pre-cache output value pointers
    std::vector<ir::Handle> output_handles;
    std::vector<Value *> output_cache;
    for (const auto &output_param: ir_node.outputs) {
        ir::Handle handle(key, output_param.name);
        output_handles.push_back(handle);
        output_cache.push_back(&this->outputs[handle]);
    }

    return {
        {this,
         std::move(inputs),
         std::move(output_handles),
         std::move(accumulated),
         std::move(aligned_data),
         std::move(aligned_time),
         std::move(input_sources),
         std::move(output_cache)},
        xerrors::NIL
    };
}

void State::ingest(const telem::Frame &frame) {
    for (auto i = 0; i < frame.size(); i++)
        reads[frame.channels->at(i)].push_back(
            xmemory::local_shared<telem::Series>(std::move(frame.series->at(i)))
        );
}

std::vector<std::pair<types::ChannelKey, Series>> State::flush_writes() {
    std::vector<std::pair<types::ChannelKey, Series>> result;
    result.reserve(writes.size());
    for (const auto &[key, data]: writes)
        result.push_back({key, data});
    writes.clear();
    return result;
}

void State::clear_reads() {
    reads.clear();
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
    if (inputs.empty()) return true;

    bool has_unconsumed = false;
    for (size_t i = 0; i < inputs.size(); i++) {
        const auto *src = this->input_sources[i];
        // Defensive null checks: verify both smart pointer and underlying raw pointer
        if (src == nullptr) continue;
        const auto *time_ptr = src->time.get();
        const auto *data_ptr = src->data.get();
        if (time_ptr != nullptr && data_ptr != nullptr && time_ptr->size() > 0 &&
            data_ptr->size() > 0) {
            if (auto ts = time_ptr->at<telem::TimeStamp>(-1);
                ts > this->accumulated[i].last_timestamp) {
                this->accumulated[i].data = src->data;
                this->accumulated[i].time = src->time;
                this->accumulated[i].last_timestamp = ts;
                this->accumulated[i].consumed = false;
            }
        }

        if (accumulated[i].data == nullptr || accumulated[i].data->empty())
            return false;

        if (!accumulated[i].consumed) has_unconsumed = true;
    }

    if (!has_unconsumed) return false;

    for (size_t i = 0; i < this->inputs.size(); i++) {
        this->aligned_data[i] = this->accumulated[i].data;
        this->aligned_time[i] = this->accumulated[i].time;
        this->accumulated[i].consumed = true;
    }

    return true;
}

std::pair<telem::MultiSeries, bool> State::read_channel(const types::ChannelKey key) {
    auto it = reads.find(key);
    if (it == reads.end() || it->second.empty()) return {telem::MultiSeries{}, false};
    telem::MultiSeries ms;
    for (const auto &s: it->second)
        ms.series.push_back(s->deep_copy());
    return {std::move(ms), true};
}

std::tuple<telem::MultiSeries, telem::MultiSeries, bool>
Node::read_chan(const types::ChannelKey key) {
    auto [data, ok] = state_ptr->read_channel(key);
    if (!ok) return {telem::MultiSeries{}, telem::MultiSeries{}, false};
    auto index_it = state_ptr->indexes.find(key);
    if (index_it == state_ptr->indexes.end() || index_it->second == 0)
        return {std::move(data), telem::MultiSeries{}, !data.series.empty()};
    auto [time, time_ok] = state_ptr->read_channel(index_it->second);
    if (!time_ok) return {telem::MultiSeries{}, telem::MultiSeries{}, false};
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
) {
    state_ptr->write_channel(key, data, time);
}
}
