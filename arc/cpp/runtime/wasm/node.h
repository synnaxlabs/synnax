// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <utility>
#include <vector>

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/runtime/wasm/module.h"
#include "arc/cpp/stl/strings/state.h"

namespace arc::runtime::wasm {
class Node : public node::Node {
    ir::Node ir;
    state::Node state;
    Module::Function func;
    std::vector<x::telem::SampleValue> inputs;
    std::vector<Module::Function::Result> results;
    std::vector<int> offsets;
    std::vector<bool> string_inputs;
    std::vector<bool> chan_inputs;
    std::vector<bool> var_inputs;
    std::vector<bool> string_outputs;
    std::shared_ptr<stl::strings::State> str_state;
    bool initialized = false;
    bool is_entry_node = false;
    /// @brief marks a node with no $sel input.
    static constexpr size_t NO_SEL = ~size_t{0};
    size_t sel_idx = NO_SEL;
    x::telem::MonoClock clock;

    /// @brief reports whether any input other than $sel has unconsumed data.
    [[nodiscard]] bool data_fresh() const {
        for (size_t i = 0; i < this->ir.inputs.size(); i++) {
            if (i == this->sel_idx || !this->ir.inputs[i].value.is_null() ||
                this->chan_inputs[i] || this->var_inputs[i])
                continue;
            if (this->state.input_fresh(i)) return true;
        }
        return false;
    }

public:
    Node(
        const ir::IR &prog,
        const ir::Node &node,
        state::Node &&state,
        const Module::Function &func,
        std::shared_ptr<stl::strings::State> str_state
    ):
        ir(node),
        state(std::move(state)),
        func(func),
        str_state(std::move(str_state)),
        is_entry_node(arc::ir::is_entry_node(prog, node)) {
        const auto &func_ir = prog.function(node.type);
        this->inputs.resize(node.inputs.size());
        this->offsets.resize(node.outputs.size());
        this->var_inputs.resize(node.inputs.size());
        for (size_t i = 0; i < node.inputs.size(); i++)
            this->var_inputs[i] = node.inputs[i].type.kind == types::Kind::VarRef;
        this->string_inputs.resize(func_ir.inputs.size());
        this->chan_inputs.resize(func_ir.inputs.size());
        for (size_t i = 0; i < func_ir.inputs.size(); i++) {
            this->string_inputs[i] = func_ir.inputs[i].type.kind == types::Kind::String;
            this->chan_inputs[i] = func_ir.inputs[i].type.kind == types::Kind::Chan;
        }
        this->string_outputs.resize(func_ir.outputs.size());
        for (size_t i = 0; i < func_ir.outputs.size(); i++)
            this->string_outputs[i] = func_ir.outputs[i].type.kind ==
                                      types::Kind::String;
        if (const auto [idx, err] = this->state.resolve_input("$sel"); !err)
            this->sel_idx = idx;
    }

    x::errors::Error next(node::Context &ctx) override {
        if (this->is_entry_node) {
            if (this->initialized) return x::errors::NIL;
            this->initialized = true;
        }

        // A $sel-only change re-points without emitting; the value fires on the
        // next input.
        if (this->sel_idx != NO_SEL && !this->data_fresh()) {
            this->state.refresh_inputs();
            return x::errors::NIL;
        }

        if (!state.refresh_inputs()) return x::errors::NIL;

        // A KindChan param holds the key of the channel the body targets. The key
        // is edge-fed and can rebind at runtime, so re-read the latest each pass.
        for (size_t i = 0; i < this->ir.inputs.size(); i++) {
            if (!this->chan_inputs[i] || !this->ir.inputs[i].value.is_null()) continue;
            const auto t = this->state.ref_input(i);
            if (t == nullptr || t->size() == 0) return x::errors::NIL;
            this->inputs[i] = t->at<uint32_t>(-1);
        }

        // A var input references a variable's node; re-read the latest each pass.
        for (size_t i = 0; i < this->ir.inputs.size(); i++) {
            if (!this->var_inputs[i]) continue;
            const auto t = this->state.ref_input(i);
            if (t == nullptr || t->size() == 0) return x::errors::NIL;
            if (this->string_inputs[i])
                this->inputs[i] = static_cast<int32_t>(
                    this->str_state->create(t->at<std::string>(-1))
                );
            else
                this->inputs[i] = t->at(-1);
        }

        int64_t max_length = 0;
        int64_t longest_input_idx = -1;
        for (size_t i = 0; i < this->ir.inputs.size(); i++) {
            if (!this->ir.inputs[i].value.is_null() || this->chan_inputs[i] ||
                this->var_inputs[i])
                continue;
            const auto inp = this->state.input(i);
            const auto data_len = static_cast<int64_t>(inp->size());
            if (data_len > max_length) {
                max_length = data_len;
                longest_input_idx = static_cast<int64_t>(i);
            }
        }

        // With no edge-fed inputs, the node executes once over its literal inputs.
        if (longest_input_idx < 0) max_length = 1;
        if (max_length <= 0) return x::errors::NIL;
        for (auto &offset: this->offsets)
            offset = 0;

        // String outputs are variable-density and cannot be resize'd. Their
        // data buffer is built once at the end of the loop from accumulated
        // strings. Numeric outputs are pre-sized here so set() can do
        // fixed-stride writes per sample.
        std::vector<std::vector<std::string>> string_results;
        for (size_t i = 0; i < this->ir.outputs.size(); i++) {
            if (this->string_outputs[i]) {
                if (string_results.empty())
                    string_results.resize(this->ir.outputs.size());
                string_results[i].reserve(max_length);
            } else {
                this->state.output(i)->resize(max_length);
            }
            this->state.output_time(i)->resize(max_length);
        }

        // Copy alignment and time range from inputs to outputs.
        // Alignments are summed to guarantee uniqueness across different input
        // sources.
        x::telem::Alignment alignment_sum;
        x::telem::TimeRange time_range{x::telem::TimeStamp(0), x::telem::TimeStamp(0)};
        for (size_t i = 0; i < this->ir.inputs.size(); i++) {
            if (!this->ir.inputs[i].value.is_null() || this->chan_inputs[i] ||
                this->var_inputs[i])
                continue;
            const auto &input = this->state.input(i);
            alignment_sum += input->alignment;
            if (time_range.start == x::telem::TimeStamp(0) ||
                input->time_range.start < time_range.start)
                time_range.start = input->time_range.start;
            if (input->time_range.end > time_range.end)
                time_range.end = input->time_range.end;
        }
        for (size_t i = 0; i < this->ir.outputs.size(); i++) {
            this->state.output(i)->alignment = alignment_sum;
            this->state.output(i)->time_range = time_range;
            this->state.output_time(i)->alignment = alignment_sum;
            this->state.output_time(i)->time_range = time_range;
        }

        state::Series longest_input_time;
        if (longest_input_idx >= 0)
            longest_input_time = this->state.input_time(longest_input_idx);

        // Dispatcher drivers alternate; no input's time is honest, so stamp the
        // clock.
        const bool clock_stamp = longest_input_idx < 0 || this->sel_idx != NO_SEL;

        this->state.set_current_node_key(this->ir.key);

        for (int i = 0; i < max_length; i++) {
            for (size_t j = 0; j < this->ir.inputs.size(); j++) {
                if (!this->ir.inputs[j].value.is_null() || this->chan_inputs[j] ||
                    this->var_inputs[j])
                    continue;
                const auto input_series = this->state.input(j);
                const auto input_len = static_cast<int>(input_series->size());
                const auto idx = i % input_len;
                if (!this->string_inputs[j]) {
                    this->inputs[j] = input_series->at(idx);
                } else {
                    // String channels are variable-length but WASM expects
                    // i32 handles. Convert inline.
                    const auto s = input_series->at<std::string>(idx);
                    this->inputs[j] = static_cast<int32_t>(this->str_state->create(s));
                }
            }

            const auto err = this->func.call(this->inputs, this->results);
            if (err) {
                ctx.report_error(
                    x::errors::Error(
                        "WASM execution failed in node " + this->ir.key +
                        " at sample " + std::to_string(i) + "/" +
                        std::to_string(max_length) + ": " + err.message()
                    )
                );
                continue;
            }

            x::telem::TimeStamp ts;
            if (clock_stamp)
                ts = this->clock.now();
            else
                ts = longest_input_time->at<x::telem::TimeStamp>(i);

            for (size_t j = 0; j < results.size(); j++) {
                auto [value, changed] = results[j];
                if (!changed) continue;
                if (this->string_outputs[j]) {
                    // WASM returned an i32 string handle; materialize it
                    // to its actual string value, mirroring the input-side
                    // conversion above.
                    const auto handle = static_cast<uint32_t>(std::get<int32_t>(value));
                    string_results[j].push_back(this->str_state->get(handle));
                } else {
                    this->state.output(j)->set(this->offsets[j], value);
                }
                this->state.output_time(j)->set(this->offsets[j], ts);
                this->offsets[j]++;
            }
        }

        for (size_t j = 0; j < this->ir.outputs.size(); j++) {
            const auto off = this->offsets[j];
            auto &out = this->state.output(j);
            if (this->string_outputs[j])
                *out = x::telem::Series(string_results[j], x::telem::STRING_T);
            else
                out->resize(off);
            this->state.output_time(j)->resize(off);
            if (off > 0) ctx.mark_changed(j);
        }

        return x::errors::NIL;
    }

    void reset() override {
        this->initialized = false;
        this->state.reset();
        this->state.clear_node(this->ir.key);
    }

    [[nodiscard]] bool is_output_truthy(size_t output_idx) const override {
        return state.is_output_truthy(output_idx);
    }
};
}
