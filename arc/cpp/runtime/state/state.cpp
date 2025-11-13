// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include <utility>

#include "arc/cpp/runtime/state/state.h"

namespace arc::runtime::state {
Series parse_default_value(const nlohmann::json &value, const ir::Type &type) {
    auto data_type = type.telem();
    auto series = xmemory::make_local_shared<telem::Series>(data_type, 1);
    if (value.is_null()) {
        switch (type.kind) {
            case ir::TypeKind::I8:
                series->write(static_cast<int8_t>(0));
                break;
            case ir::TypeKind::I16:
                series->write(static_cast<int16_t>(0));
                break;
            case ir::TypeKind::I32:
                series->write(static_cast<int32_t>(0));
                break;
            case ir::TypeKind::I64:
                series->write(static_cast<int64_t>(0));
                break;
            case ir::TypeKind::U8:
                series->write(static_cast<uint8_t>(0));
                break;
            case ir::TypeKind::U16:
                series->write(static_cast<uint16_t>(0));
                break;
            case ir::TypeKind::U32:
                series->write(static_cast<uint32_t>(0));
                break;
            case ir::TypeKind::U64:
                series->write(static_cast<uint64_t>(0));
                break;
            case ir::TypeKind::F32:
                series->write(0.0f);
                break;
            case ir::TypeKind::F64:
                series->write(0.0);
                break;
            default:
                break;
        }
        return series;
    }

    switch (type.kind) {
        case ir::TypeKind::I8:
            series->write(
                static_cast<int8_t>(value.is_number() ? value.get<int>() : 0)
            );
            break;
        case ir::TypeKind::I16:
            series->write(
                static_cast<int16_t>(value.is_number() ? value.get<int>() : 0)
            );
            break;
        case ir::TypeKind::I32:
            series->write(value.is_number() ? value.get<int32_t>() : 0);
            break;
        case ir::TypeKind::I64:
            series->write(value.is_number() ? value.get<int64_t>() : 0);
            break;
        case ir::TypeKind::U8:
            series->write(
                static_cast<uint8_t>(value.is_number() ? value.get<unsigned>() : 0)
            );
            break;
        case ir::TypeKind::U16:
            series->write(
                static_cast<uint16_t>(value.is_number() ? value.get<unsigned>() : 0)
            );
            break;
        case ir::TypeKind::U32:
            series->write(value.is_number() ? value.get<uint32_t>() : 0);
            break;
        case ir::TypeKind::U64:
            series->write(value.is_number() ? value.get<uint64_t>() : 0);
            break;
        case ir::TypeKind::F32:
            series->write(value.is_number() ? value.get<float>() : 0.0f);
            break;
        case ir::TypeKind::F64:
            series->write(value.is_number() ? value.get<double>() : 0.0);
            break;
        case ir::TypeKind::String:
            if (value.is_string()) {
                // String handling would require special series support
                // For now, leave empty
            }
            break;
        default:
            break;
    }

    return series;
}

telem::DataType to_telem_type(const ir::Type &type) {
    switch (type.kind) {
        case ir::TypeKind::U8:
            return telem::UINT8_T;
        case ir::TypeKind::U16:
            return telem::UINT16_T;
        case ir::TypeKind::U32:
            return telem::UINT32_T;
        case ir::TypeKind::U64:
            return telem::UINT64_T;
        case ir::TypeKind::I8:
            return telem::INT8_T;
        case ir::TypeKind::I16:
            return telem::INT16_T;
        case ir::TypeKind::I32:
            return telem::INT32_T;
        case ir::TypeKind::I64:
            return telem::INT64_T;
        case ir::TypeKind::F32:
            return telem::FLOAT32_T;
        case ir::TypeKind::F64:
            return telem::FLOAT64_T;
        case ir::TypeKind::String:
            return telem::STRING_T;
        case ir::TypeKind::TimeStamp:
            return telem::TIMESTAMP_T;
        case ir::TypeKind::Series:
            // For series, get the element type
            if (type.elem) { return to_telem_type(*type.elem); }
            return telem::UNKNOWN_T;
        default:
            return telem::UNKNOWN_T;
    }
}

State::State(const Config &cfg): cfg(cfg) {
    for (const auto &digest: cfg.channels)
        indexes[digest.key] = digest.index;
    for (const auto &node: cfg.ir.nodes) {
        for (const auto &output: node.outputs) {
            ir::Handle handle(node.key, output.name);
            outputs[handle] = Value{
                xmemory::local_shared<telem::Series>(to_telem_type(output.type), 0),
                xmemory::local_shared<telem::Series>(telem::TIMESTAMP_T, 0)
            };
        }
    }
}

Node State::node(const std::string &key) {
    auto ir_node_iter = this->cfg.ir.find_node(key);
    if (ir_node_iter == this->cfg.ir.nodes.end()) {
        throw std::runtime_error("Node not found: " + key);
    }
    const auto &ir_node = *ir_node_iter;

    const size_t num_inputs = ir_node.inputs.size();
    std::vector<ir::Edge> inputs(num_inputs);
    std::vector<Series> aligned_data(num_inputs);
    std::vector<Series> aligned_time(num_inputs);
    std::vector<Node::InputEntry> accumulated(num_inputs);
    std::vector<Value *> input_sources(num_inputs);

    // Initialize aligned time series
    for (size_t i = 0; i < num_inputs; i++)
        aligned_time[i] = xmemory::make_local_shared<telem::Series>(
            telem::TIMESTAMP_T,
            0
        );

    // Build inputs and handle default values
    for (size_t i = 0; i < num_inputs; i++) {
        const auto &param = ir_node.inputs[i];
        ir::Handle target_handle(key, param.name);
        auto edge_iter = cfg.ir.find_edge_by_target(target_handle);

        if (edge_iter != cfg.ir.edges.end()) {
            // Connected input - use edge source
            inputs[i] = *edge_iter;
            const auto &source_handle = edge_iter->source;

            // Get the data type from the source output
            auto source_output_iter = outputs.find(source_handle);
            if (source_output_iter != outputs.end()) {
                aligned_data[i] = xmemory::make_local_shared<telem::Series>(
                    source_output_iter->second.data->data_type(),
                    0
                );
                input_sources[i] = &source_output_iter->second;

                // Initialize accumulated entry for connected input
                // Data starts empty, will be populated when source produces output
                accumulated[i].last_timestamp = telem::TimeStamp(0);
                accumulated[i].consumed = true; // Start as consumed (no data yet)
            }
        } else {
            // Unconnected input - create synthetic source with default value
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
            accumulated[i].consumed = false;

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
        std::move(inputs),
        std::move(output_handles),
        *this,
        std::move(accumulated),
        std::move(aligned_data),
        std::move(aligned_time),
        std::move(input_sources),
        std::move(output_cache)
    };
}

void State::ingest(const telem::Frame &frame) {
    for (auto i = 0; i < frame.size(); i++)
        reads[frame.channels->at(i)].push_back(
            xmemory::local_shared<telem::Series>(std::move(frame.series->at(i)))
        );
}

std::vector<std::pair<arc::ChannelKey, Series>> State::flush_writes() {
    std::vector<std::pair<arc::ChannelKey, Series>> result;
    result.reserve(writes.size());

    for (const auto &[key, data]: writes) {
        result.emplace_back(key, data);
    }

    writes.clear();
    return result;
}

void State::clear_reads() {
    reads.clear();
}

void State::write_channel(
    const arc::ChannelKey key,
    const Series &data,
    const Series &time
) {
    writes[key] = data;
    if (const auto idx_iter = indexes.find(key);
        idx_iter != indexes.end() && idx_iter->second != 0)
        writes[idx_iter->second] = time;
}

// Node implementation

bool Node::refresh_inputs() {
    if (inputs.empty()) return true;

    bool has_unconsumed = false;
    for (size_t i = 0; i < inputs.size(); i++) {

        if (const auto *src = this->input_sources[i];
            src != nullptr && src->time != nullptr && !src->time->empty()) {
            if (auto ts = src->time->at<telem::TimeStamp>(-1);
                ts > this->accumulated[i].last_timestamp) {
                this->accumulated[i].data = src->data;
                this->accumulated[i].time = src->time;
                this->accumulated[i].last_timestamp = ts;
                this->accumulated[i].consumed = false;
            }
        }

        // Early exit if any input has no data
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
}
