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
    if (value.has_value())
        return x::mem::make_local_shared<x::telem::Series>(data_type.cast(*value));
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
        default:
            return x::mem::make_local_shared<x::telem::Series>(data_type, 0);
    }
    return x::mem::make_local_shared<x::telem::Series>(data_type, 0);
}

State::State(const Config &cfg, errors::Handler error_handler):
    State(
        cfg,
        std::make_shared<stl::channels::State>(cfg.channels),
        std::make_shared<stl::strings::State>(),
        std::make_shared<stl::series::State>(),
        std::make_shared<stl::stateful::Variables>(),
        std::move(error_handler)
    ) {}

State::State(
    const Config &cfg,
    std::shared_ptr<stl::channels::State> channel,
    std::shared_ptr<stl::strings::State> strings,
    std::shared_ptr<stl::series::State> series,
    std::shared_ptr<stl::stateful::Variables> vars,
    errors::Handler error_handler
):
    cfg(cfg),
    channel(std::move(channel)),
    strings(std::move(strings)),
    series(std::move(series)),
    vars(std::move(vars)),
    error_handler(std::move(error_handler)) {
    size_t total = 0;
    for (const auto &node: cfg.ir.nodes)
        total += node.outputs.size();
    this->values.reserve(total);

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
    std::vector<size_t> input_source_idx(num_inputs, Node::NO_SOURCE);
    std::vector<bool> is_reference(num_inputs);

    for (size_t i = 0; i < num_inputs; i++)
        aligned_time[i] = x::mem::make_local_shared<x::telem::Series>(
            x::telem::TIMESTAMP_T,
            0
        );

    bool has_edge_fed = false;
    for (size_t i = 0; i < num_inputs; i++) {
        const auto &param = ir_node.inputs[i];
        // A channel input is a reference resolved by key in the host functions, not
        // a value stream. It carries no data series and never gates execution.
        if (param.type.kind == types::Kind::Chan) {
            is_reference[i] = true;
            if (auto edge = this->cfg.ir.edge_to(ir::Handle(key, param.name))) {
                inputs[i] = *edge;
                if (auto it = this->value_index.find(edge->source);
                    it != this->value_index.end())
                    input_source_idx[i] = it->second;
            }
            continue;
        }
        // A var input names its variable's node in type.name. It binds that
        // node's output slot directly: no edge, never gates or wakes.
        if (param.type.kind == types::Kind::VarRef) {
            is_reference[i] = true;
            ir::Handle src(param.type.name, ir::default_output_param);
            inputs[i] = ir::Edge(src, ir::Handle(key, param.name));
            if (auto it = this->value_index.find(src); it != this->value_index.end())
                input_source_idx[i] = it->second;
            continue;
        }
        ir::Handle target_handle(key, param.name);
        if (auto edge = this->cfg.ir.edge_to(target_handle)) {
            has_edge_fed = true;
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
                // Starting armed lets an OnReset input, which preserves this flag,
                // fire on its first arrival.
                accumulated[i].consumed = false;
            }
        } else {
            ir::Handle synthetic_handle("__default_" + key + "_" + param.name, "out");
            inputs[i] = ir::Edge(synthetic_handle, target_handle);

            auto data_series = parse_default_value(
                types::to_sample_value(param.value, param.type),
                param.type
            );
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

    // A node with no edge-fed data input never re-arms on its own; its trigger
    // edges register as gating-only entries so each fire re-runs it exactly once.
    // SY-4495: registering unconditionally would make multi-trigger nodes await
    // fresh values on every trigger before running.
    if (!has_edge_fed) {
        for (const auto &e: this->cfg.ir.edges) {
            if (e.target.node != key) continue;
            if (!ir_node.resolve_input(e.target.param).second) continue;
            inputs.push_back(e);
            aligned_data.emplace_back();
            aligned_time.push_back(
                x::mem::make_local_shared<x::telem::Series>(x::telem::TIMESTAMP_T, 0)
            );
            Node::InputEntry entry;
            if (const auto it = this->value_index.find(e.source);
                it != this->value_index.end())
                entry.source = it->second;
            entry.last_timestamp = x::telem::TimeStamp(0);
            entry.consumed = false;
            input_source_idx.push_back(entry.source);
            accumulated.push_back(std::move(entry));
            is_reference.push_back(false);
        }
    }

    // Register reads re-arm on fresh values; deref reads on post-entry values;
    // self-write feeders on Reset only.
    std::vector<Node::Rearm> rearm(inputs.size(), Node::Rearm::Always);
    for (size_t i = 0; i < inputs.size(); i++) {
        const auto *src_node = ir::find_node(this->cfg.ir, inputs[i].source.node);
        if (src_node == nullptr ||
            (src_node->type != "variable" && src_node->type != "stateful_variable"))
            continue;
        rearm[i] = Node::Rearm::OnFresh;
        if (!src_node->inputs.empty() && src_node->inputs[0].value.is_null())
            rearm[i] = Node::Rearm::OnArrival;
        for (const auto &e: this->cfg.ir.edges)
            if (e.target.node == src_node->key && e.source.node == key) {
                rearm[i] = Node::Rearm::OnReset;
                break;
            }
    }

    // A node that feeds a register it reads by var ref is an entry one-shot:
    // its trigger entries latch on Reset, matching the edge-fed read latch.
    bool self_write = false;
    for (const auto &param: ir_node.inputs) {
        if (param.type.kind != types::Kind::VarRef) continue;
        for (const auto &e: this->cfg.ir.edges)
            if (e.source.node == key && e.target.node == param.type.name) {
                self_write = true;
                break;
            }
    }
    if (self_write)
        for (size_t i = num_inputs; i < inputs.size(); i++)
            rearm[i] = Node::Rearm::OnReset;

    std::vector<ir::Handle> output_handles;
    std::vector<size_t> output_idx;
    for (const auto &output_param: ir_node.outputs) {
        ir::Handle handle(key, output_param.name);
        output_handles.push_back(handle);
        output_idx.push_back(this->value_index[handle]);
    }

    return {
        Node(
            *this,
            std::move(inputs),
            std::move(output_handles),
            std::move(input_source_idx),
            std::move(output_idx),
            std::move(accumulated),
            std::move(aligned_data),
            std::move(aligned_time),
            std::move(is_reference),
            std::move(rearm),
            ir_node.inputs
        ),
        x::errors::NIL
    };
}

void State::ingest(const x::telem::Frame &frame) {
    this->channel->ingest(frame);
}

void State::flush_into(x::telem::Frame &out) {
    this->channel->flush_into(out);
    this->series->clear();
    this->strings->clear();
}

void State::reset() {
    this->channel->reset();
    this->strings->reset();
    this->series->clear();
    this->vars->reset();
    this->authority_changes.clear();
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

void Node::init_input(size_t param_index, const Series &data, const Series &time) {
    if (this->accumulated[param_index].source == NO_SOURCE) return;
    auto &src = this->state.values[this->accumulated[param_index].source];
    src.data = data;
    src.time = time;
    this->accumulated[param_index].data = data;
    this->accumulated[param_index].time = time;
    this->accumulated[param_index].last_timestamp = x::telem::TimeStamp(0);
    this->accumulated[param_index].consumed = false;
}

bool Node::refresh_inputs() {
    bool has_data_input = false;
    bool has_unconsumed = false;
    for (size_t i = 0; i < this->inputs.size(); i++) {
        if (this->is_reference[i]) continue;
        has_data_input = true;
        if (this->accumulated[i].source != NO_SOURCE) {
            const Value &src = this->state.values[this->accumulated[i].source];
            const auto *time_ptr = src.time.get();
            const auto *data_ptr = src.data.get();
            if (time_ptr != nullptr && data_ptr != nullptr && time_ptr->size() > 0 &&
                data_ptr->size() > 0) {
                if (auto ts = time_ptr->at<x::telem::TimeStamp>(-1);
                    ts > this->accumulated[i].last_timestamp) {
                    bool consumed = false;
                    if (this->rearm[i] == Rearm::OnReset)
                        consumed = this->accumulated[i].consumed;
                    this->accumulated[i].data = src.data;
                    this->accumulated[i].time = src.time;
                    this->accumulated[i].last_timestamp = ts;
                    this->accumulated[i].consumed = consumed;
                }
            }
        }
        if (this->accumulated[i].data == nullptr || this->accumulated[i].data->empty())
            return false;
        if (!this->accumulated[i].consumed) has_unconsumed = true;
    }
    if (!has_data_input) return true;
    if (!has_unconsumed) return false;
    for (size_t i = 0; i < this->inputs.size(); i++) {
        if (this->is_reference[i]) continue;
        this->aligned_data[i] = this->accumulated[i].data;
        this->aligned_time[i] = this->accumulated[i].time;
        this->accumulated[i].consumed = true;
    }
    return true;
}

bool Node::ref_sourced(const size_t param_index) const {
    return param_index < this->input_source_idx.size() &&
           this->is_reference[param_index] &&
           this->input_source_idx[param_index] != NO_SOURCE;
}

Series Node::ref_input(const size_t param_index) const {
    if (param_index < this->input_source_idx.size() &&
        this->is_reference[param_index]) {
        if (const auto idx = this->input_source_idx[param_index]; idx != NO_SOURCE)
            return this->state.values[idx].data;
    }
    return {};
}

std::string Node::string_input(const std::string &name) const {
    const auto [i, err] = this->resolve_input(name);
    if (err) return "";
    if (const auto s = this->ref_input(i); s != nullptr && s->size() > 0)
        return x::telem::cast<std::string>(s->at(-1));
    const auto &p = this->params[i];
    if (p.value.is_string()) return p.value.get<std::string>();
    return "";
}

void Node::absorb_inputs() {
    for (size_t i = 0; i < this->inputs.size(); i++)
        this->absorb_input(i);
}

void Node::absorb_input(const size_t i) {
    if (this->is_reference[i]) return;
    const auto src_idx = this->input_source_idx[i];
    if (src_idx == NO_SOURCE) return;
    const Value &src = this->state.values[src_idx];
    x::telem::TimeStamp ts(0);
    if (src.time != nullptr && src.time->size() > 0)
        ts = src.time->at<x::telem::TimeStamp>(-1);
    this->accumulated[i].source = src_idx;
    this->accumulated[i].data = src.data;
    this->accumulated[i].time = src.time;
    this->accumulated[i].last_timestamp = ts;
    this->accumulated[i].consumed = true;
}

std::pair<Series, bool> Node::consume_input(const size_t i) {
    if (i >= this->inputs.size() || this->is_reference[i]) return {Series(), false};
    const auto src_idx = this->input_source_idx[i];
    if (src_idx == NO_SOURCE) return {Series(), false};
    const Value &src = this->state.values[src_idx];
    if (src.data == nullptr || src.data->empty()) return {Series(), false};
    x::telem::TimeStamp ts(0);
    if (src.time != nullptr && src.time->size() > 0)
        ts = src.time->at<x::telem::TimeStamp>(-1);
    if (ts <= this->accumulated[i].last_timestamp && this->accumulated[i].consumed)
        return {Series(), false};
    this->accumulated[i].source = src_idx;
    this->accumulated[i].data = src.data;
    this->accumulated[i].time = src.time;
    this->accumulated[i].last_timestamp = ts;
    this->accumulated[i].consumed = true;
    return {src.data, true};
}

bool Node::input_fresh(const size_t i) const {
    if (i >= this->inputs.size() || this->is_reference[i]) return false;
    const auto src_idx = this->input_source_idx[i];
    if (src_idx == NO_SOURCE) return false;
    const Value &src = this->state.values[src_idx];
    if (src.data == nullptr || src.data->empty()) return false;
    x::telem::TimeStamp ts(0);
    if (src.time != nullptr && src.time->size() > 0)
        ts = src.time->at<x::telem::TimeStamp>(-1);
    return ts > this->accumulated[i].last_timestamp || !this->accumulated[i].consumed;
}

std::pair<Series, bool> Node::last_changed() {
    size_t best = 0;
    x::telem::TimeStamp best_ts(0);
    bool found = false;
    for (size_t i = 0; i < this->inputs.size(); i++) {
        if (this->is_reference[i]) continue;
        const auto src_idx = this->input_source_idx[i];
        if (src_idx == NO_SOURCE) continue;
        const Value &src = this->state.values[src_idx];
        if (src.data == nullptr || src.data->empty()) continue;
        x::telem::TimeStamp ts(0);
        if (src.time != nullptr && src.time->size() > 0)
            ts = src.time->at<x::telem::TimeStamp>(-1);
        if (ts <= this->accumulated[i].last_timestamp && this->accumulated[i].consumed)
            continue;
        if (!found || ts > best_ts) {
            best = i;
            best_ts = ts;
            found = true;
        }
    }
    if (!found) return {Series(), false};
    const Value &src = this->state.values[this->input_source_idx[best]];
    this->accumulated[best].source = this->input_source_idx[best];
    this->accumulated[best].data = src.data;
    this->accumulated[best].time = src.time;
    this->accumulated[best].last_timestamp = best_ts;
    this->accumulated[best].consumed = true;
    return {src.data, true};
}

std::pair<size_t, x::errors::Error> Node::resolve_input(const std::string &name) const {
    for (size_t i = 0; i < this->params.size(); ++i)
        if (this->params[i].name == name) return {i, x::errors::NIL};
    return {
        0,
        x::errors::Error(x::errors::NOT_FOUND, "node has no input named " + name)
    };
}

std::tuple<x::telem::MultiSeries, x::telem::MultiSeries, bool>
Node::read_series(const types::ChannelKey key) const {
    return this->state.channel->read_series(key);
}

void Node::write_series(
    const types::ChannelKey key,
    const Series &data,
    const Series &time
) const {
    this->state.channel->write_series(key, data, time);
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

bool Node::is_output_truthy(size_t output_idx) const {
    if (output_idx >= this->output_idx.size()) return false;
    const auto *s = this->state.values[this->output_idx[output_idx]].data.get();
    return s != nullptr && Node::is_series_truthy(*s);
}

void Node::set_current_node_key(const std::string &key) {
    this->state.set_current_node_key(key);
}

}
