// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cstring>
#include <utility>

#include "x/cpp/errors/errors.h"

#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/types/types.h"

namespace arc::runtime::state {
void mark_write_active(
    std::unordered_map<types::ChannelKey, Series> &writes,
    std::vector<types::ChannelKey> &active_write_keys,
    const types::ChannelKey key
) {
    auto &buf = writes[key];
    if (buf == nullptr || buf->empty()) active_write_keys.push_back(key);
}

template<typename T>
void append_fixed_sample(
    std::unordered_map<types::ChannelKey, Series> &writes,
    std::vector<types::ChannelKey> &active_write_keys,
    const types::ChannelKey key,
    const x::telem::DataType dt,
    const T value
) {
    mark_write_active(writes, active_write_keys, key);
    auto &buf = writes[key];
    if (buf == nullptr) {
        buf = x::mem::make_local_shared<x::telem::Series>(dt, 1);
    } else if (buf->data_type() != dt) {
        // Defensive fallback: replace mismatched buffers with latest value.
        buf = x::mem::make_local_shared<x::telem::Series>(dt, 1);
        buf->write(value);
        return;
    } else if (buf->cap() < buf->size() + 1) {
        const auto grown_cap = std::max(buf->size() + 1, buf->cap() * 2 + 1);
        auto grown = x::mem::make_local_shared<x::telem::Series>(dt, grown_cap);
        grown->write(*buf);
        grown->time_range = buf->time_range;
        grown->alignment = buf->alignment;
        buf = std::move(grown);
    }
    buf->write(value);
}

template<typename T>
void write_channel_fixed(
    std::unordered_map<types::ChannelKey, Series> &writes,
    std::vector<types::ChannelKey> &active_write_keys,
    const std::unordered_map<types::ChannelKey, types::ChannelKey> &indexes,
    const types::ChannelKey key,
    const x::telem::DataType dt,
    const T value
) {
    append_fixed_sample(writes, active_write_keys, key, dt, value);
    if (const auto idx_iter = indexes.find(key);
        idx_iter != indexes.end() && idx_iter->second != 0)
        append_fixed_sample(
            writes,
            active_write_keys,
            idx_iter->second,
            x::telem::TIMESTAMP_T,
            x::telem::TimeStamp::now()
        );
}

void append_to_write_buffer(Series &dest, const Series &src) {
    if (src == nullptr || src->empty()) return;
    if (dest == nullptr) {
        dest = x::mem::make_local_shared<x::telem::Series>(src->deep_copy());
        return;
    }
    if (dest->data_type() != src->data_type())
        throw std::runtime_error("cannot append series with mismatched data types");

    if (dest->data_type().is_variable()) {
        const auto old_time_range = dest->time_range;
        auto merged = dest->strings();
        auto incoming = src->strings();
        merged.insert(merged.end(), incoming.begin(), incoming.end());
        dest = x::mem::make_local_shared<x::telem::Series>(merged, dest->data_type());
        dest->time_range = old_time_range;
    } else {
        const auto required_cap = dest->size() + src->size();
        if (dest->cap() < required_cap) {
            const auto grown_cap = std::max(required_cap, dest->cap() * 2 + 1);
            auto grown = x::mem::make_local_shared<x::telem::Series>(
                dest->data_type(),
                grown_cap
            );
            grown->write(*dest);
            dest = std::move(grown);
        }
        dest->write(*src);
    }
    if (dest->time_range == x::telem::TimeRange()) {
        dest->time_range = src->time_range;
    } else {
        if (src->time_range.start < dest->time_range.start)
            dest->time_range.start = src->time_range.start;
        if (src->time_range.end > dest->time_range.end)
            dest->time_range.end = src->time_range.end;
    }
}

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
    cfg(cfg), error_handler(std::move(error_handler)) {
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
        // Keep only the last series to preserve the latest value for each channel.
        // This allows channel_read_* calls to return the most recent value even if
        // new frames for that channel haven't arrived yet.
        auto last = std::move(series_vec.back());
        series_vec.clear();
        series_vec.push_back(std::move(last));
    }
    this->series_handles.clear();
    this->series_handle_counter = 1;
    this->strings.clear();
    this->string_handle_counter = 1;

    std::vector<std::pair<types::ChannelKey, Series>> result;
    result.reserve(active_write_keys.size());
    for (const auto key: active_write_keys) {
        const auto it = writes.find(key);
        if (it == writes.end() || it->second == nullptr || it->second->empty())
            continue;
        result.push_back(
            {key, x::mem::make_local_shared<x::telem::Series>(it->second->deep_copy())}
        );
        it->second->clear();
        it->second->time_range = x::telem::TimeRange();
        it->second->alignment = x::telem::Alignment();
    }
    active_write_keys.clear();
    return result;
}

void State::reset() {
    this->reads.clear();
    this->writes.clear();
    this->active_write_keys.clear();
    this->strings.clear();
    this->series_handles.clear();
    this->string_handle_counter = 1;
    this->series_handle_counter = 1;
    this->config_strings.clear();
    this->config_string_handle_counter = k_config_string_handle_base;
    this->var_u8.clear();
    this->var_u16.clear();
    this->var_u32.clear();
    this->var_u64.clear();
    this->var_i8.clear();
    this->var_i16.clear();
    this->var_i32.clear();
    this->var_i64.clear();
    this->var_f32.clear();
    this->var_f64.clear();
    this->var_string.clear();
    this->var_series.clear();
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
    auto &data_buf = writes[key];
    if (data_buf == nullptr || data_buf->empty()) active_write_keys.push_back(key);
    append_to_write_buffer(data_buf, data);
    if (const auto idx_iter = indexes.find(key);
        idx_iter != indexes.end() && idx_iter->second != 0) {
        auto &time_buf = writes[idx_iter->second];
        if (time_buf == nullptr || time_buf->empty())
            active_write_keys.push_back(idx_iter->second);
        append_to_write_buffer(time_buf, time);
    }
}

#define IMPL_CHANNEL_WRITE_OPS(suffix, cpptype, data_type_const)                       \
    void State::write_channel_##suffix(                                                \
        const types::ChannelKey key,                                                   \
        const cpptype value                                                            \
    ) {                                                                                \
        write_channel_fixed(                                                           \
            this->writes,                                                              \
            this->active_write_keys,                                                   \
            this->indexes,                                                             \
            key,                                                                       \
            data_type_const,                                                           \
            value                                                                      \
        );                                                                             \
    }

IMPL_CHANNEL_WRITE_OPS(u8, uint8_t, x::telem::UINT8_T)
IMPL_CHANNEL_WRITE_OPS(u16, uint16_t, x::telem::UINT16_T)
IMPL_CHANNEL_WRITE_OPS(u32, uint32_t, x::telem::UINT32_T)
IMPL_CHANNEL_WRITE_OPS(u64, uint64_t, x::telem::UINT64_T)
IMPL_CHANNEL_WRITE_OPS(i8, int8_t, x::telem::INT8_T)
IMPL_CHANNEL_WRITE_OPS(i16, int16_t, x::telem::INT16_T)
IMPL_CHANNEL_WRITE_OPS(i32, int32_t, x::telem::INT32_T)
IMPL_CHANNEL_WRITE_OPS(i64, int64_t, x::telem::INT64_T)
IMPL_CHANNEL_WRITE_OPS(f32, float, x::telem::FLOAT32_T)
IMPL_CHANNEL_WRITE_OPS(f64, double, x::telem::FLOAT64_T)

#undef IMPL_CHANNEL_WRITE_OPS

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

uint32_t State::string_from_memory(const uint8_t *data, const uint32_t len) {
    const std::string str(reinterpret_cast<const char *>(data), len);
    const uint32_t handle = this->string_handle_counter++;
    this->strings[handle] = str;
    return handle;
}

uint32_t State::string_create(const std::string &str) {
    const uint32_t handle = this->string_handle_counter++;
    this->strings[handle] = str;
    return handle;
}

uint32_t State::string_create_config(const std::string &str) {
    const uint32_t handle = this->config_string_handle_counter++;
    this->config_strings[handle] = str;
    return handle;
}

std::string State::string_get(const uint32_t handle) const {
    const auto it = this->strings.find(handle);
    if (it != this->strings.end()) return it->second;
    const auto cfg_it = this->config_strings.find(handle);
    if (cfg_it == this->config_strings.end()) return "";
    return cfg_it->second;
}

bool State::string_exists(const uint32_t handle) const {
    return this->strings.contains(handle) || this->config_strings.contains(handle);
}

x::telem::Series *State::series_get(const uint32_t handle) {
    const auto it = this->series_handles.find(handle);
    if (it == this->series_handles.end()) return nullptr;
    return &it->second;
}

const x::telem::Series *State::series_get(const uint32_t handle) const {
    const auto it = this->series_handles.find(handle);
    if (it == this->series_handles.end()) return nullptr;
    return &it->second;
}

uint32_t State::series_store(x::telem::Series series) {
    const uint32_t handle = this->series_handle_counter++;
    this->series_handles.emplace(handle, std::move(series));
    return handle;
}

#define IMPL_VAR_OPS(suffix, cpptype)                                                  \
    cpptype State::var_load_##suffix(                                                  \
        const uint32_t var_id,                                                         \
        const cpptype init_value                                                       \
    ) {                                                                                \
        auto &inner = this->var_##suffix[this->current_node_key];                      \
        const auto it = inner.find(var_id);                                            \
        if (it != inner.end()) return it->second;                                      \
        inner[var_id] = init_value;                                                    \
        return init_value;                                                             \
    }                                                                                  \
    void State::var_store_##suffix(const uint32_t var_id, const cpptype value) {       \
        this->var_##suffix[this->current_node_key][var_id] = value;                    \
    }

IMPL_VAR_OPS(u8, uint8_t)
IMPL_VAR_OPS(u16, uint16_t)
IMPL_VAR_OPS(u32, uint32_t)
IMPL_VAR_OPS(u64, uint64_t)
IMPL_VAR_OPS(i8, int8_t)
IMPL_VAR_OPS(i16, int16_t)
IMPL_VAR_OPS(i32, int32_t)
IMPL_VAR_OPS(i64, int64_t)
IMPL_VAR_OPS(f32, float)
IMPL_VAR_OPS(f64, double)

#undef IMPL_VAR_OPS

uint32_t State::var_load_str(const uint32_t var_id, const uint32_t init_handle) {
    auto &inner = this->var_string[this->current_node_key];
    if (const auto it = inner.find(var_id); it != inner.end()) {
        this->strings[this->string_handle_counter] = it->second;
        return this->string_handle_counter++;
    }
    // Use string_get to resolve both transient and config string handles.
    inner[var_id] = this->string_get(init_handle);
    this->strings[this->string_handle_counter] = inner[var_id];
    return this->string_handle_counter++;
}

void State::var_store_str(const uint32_t var_id, const uint32_t str_handle) {
    if (!this->string_exists(str_handle)) return;
    // Use string_get to resolve both transient and config string handles.
    this->var_string[this->current_node_key][var_id] = this->string_get(str_handle);
}

uint32_t State::var_load_series(const uint32_t var_id, const uint32_t init_handle) {
    auto &inner = this->var_series[this->current_node_key];
    if (const auto state_it = inner.find(var_id); state_it != inner.end()) {
        auto copy = state_it->second.deep_copy();
        const uint32_t handle = this->series_handle_counter++;
        this->series_handles.emplace(handle, std::move(copy));
        return handle;
    }
    if (const auto init_it = this->series_handles.find(init_handle);
        init_it != this->series_handles.end()) {
        inner.emplace(var_id, init_it->second.deep_copy());
    }
    return init_handle;
}

void State::var_store_series(const uint32_t var_id, const uint32_t handle) {
    if (const auto it = this->series_handles.find(handle);
        it != this->series_handles.end()) {
        this->var_series[this->current_node_key].insert_or_assign(
            var_id,
            it->second.deep_copy()
        );
    }
}
}
