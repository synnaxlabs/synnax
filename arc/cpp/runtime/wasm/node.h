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

#include "x/cpp/telem/telem.h"
#include "x/cpp/xerrors/errors.h"

#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/runtime/wasm/module.h"

namespace arc::runtime::wasm {
class Node : public node::Node {
    ir::Node ir;
    state::Node state;
    Module::Function func;
    std::vector<telem::SampleValue> inputs;
    std::vector<int> offsets;
    bool initialized = false;
    bool is_expression = false;

public:
    Node(const ir::Node &node, state::Node &&state, const Module::Function &func):
        ir(node),
        state(std::move(state)),
        func(func),
        is_expression(node.key.rfind("expression_", 0) == 0) {
        this->inputs.resize(node.inputs.size());
        this->offsets.resize(node.outputs.size());
    }

    xerrors::Error next(node::Context &ctx) override {
        if (!this->is_expression) {
            if (this->initialized) return xerrors::NIL;
            this->initialized = true;
        }

        if (!state.refresh_inputs()) return xerrors::NIL;

        int64_t max_length = 0;
        int64_t longest_input_idx = 0;
        for (size_t i = 0; i < this->ir.inputs.size(); i++) {
            const auto data_len = static_cast<int64_t>(this->state.input(i)->size());
            if (data_len > max_length) {
                max_length = data_len;
                longest_input_idx = static_cast<int64_t>(i);
            }
        }

        if (this->ir.inputs.empty()) max_length = 1;
        if (max_length <= 0) return xerrors::NIL;
        for (auto &offset: this->offsets)
            offset = 0;

        for (size_t i = 0; i < this->ir.outputs.size(); i++) {
            this->state.output(i)->resize(max_length);
            this->state.output_time(i)->resize(max_length);
        }

        state::Series longest_input_time;
        if (!this->ir.inputs.empty())
            longest_input_time = this->state.input_time(longest_input_idx);

        for (int i = 0; i < max_length; i++) {
            for (size_t j = 0; j < this->ir.inputs.size(); j++) {
                const auto input_series = this->state.input(j);
                const auto input_len = static_cast<int>(input_series->size());
                this->inputs[j] = input_series->at(i % input_len);
            }

            auto [results, err] = this->func.call(this->inputs);
            if (err) {
                ctx.report_error(
                    xerrors::Error(
                        "WASM execution failed in node " + this->ir.key +
                        " at sample " + std::to_string(i) + "/" +
                        std::to_string(max_length) + ": " + err.message()
                    )
                );
                continue;
            }

            telem::TimeStamp ts;
            if (!this->ir.inputs.empty() && longest_input_time)
                ts = longest_input_time->at<telem::TimeStamp>(i);
            else
                ts = telem::TimeStamp::now();

            for (size_t j = 0; j < results.size(); j++) {
                auto [value, changed] = results[j];
                if (!changed) continue;
                this->state.output(j)->set(this->offsets[j], value);
                this->state.output_time(j)->set(this->offsets[j], ts);
                this->offsets[j]++;
            }
        }

        for (size_t j = 0; j < this->ir.outputs.size(); j++) {
            const auto off = this->offsets[j];
            this->state.output(j)->resize(off);
            this->state.output_time(j)->resize(off);
            if (off > 0) ctx.mark_changed(this->ir.outputs[j].name);
        }

        return xerrors::NIL;
    }

    void reset() override { this->initialized = false; }

    [[nodiscard]] bool is_output_truthy(const std::string &param_name) const override {
        return state.is_output_truthy(param_name);
    }
};
}
