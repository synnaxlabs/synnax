// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <utility>
#include <vector>

#include "x/cpp/telem/telem.h"
#include "x/cpp/xerrors/errors.h"

#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/runtime/wasm/module.h"
#include "arc/cpp/runtime/wasm/utils.h"

namespace arc::runtime::wasm {
class Node : public node::Node {
    ir::Node ir_node;
    state::Node state;
    Module::Function func;
    std::vector<uint64_t> inputs;
    std::vector<int> offsets;

public:
    Node(
        const ir::Node &node,
        runtime::state::Node state,
        const Module::Function &func
    ):
        ir_node(node), state(std::move(state)), func(func) {
        inputs.resize(node.inputs.size());
    }

    xerrors::Error next(node::Context &ctx) override {
        if (!state.refresh_inputs()) return xerrors::NIL;

        std::cout << "execute" << ctx.elapsed << std::endl ;
        int64_t max_length = 0;
        int64_t longest_input_idx = 0;
        for (size_t i = 0; i < ir_node.inputs.size(); i++) {
            if (const auto data_len = static_cast<int64_t>(state.input(i)->size());
                data_len > max_length) {
                max_length = data_len;
                longest_input_idx = static_cast<int64_t>(i);
            }
        }

        if (ir_node.inputs.empty()) max_length = 1;

        if (max_length <= 0) return xerrors::NIL;

        for (auto &offset: offsets)
            offset = 0;

        for (size_t i = 0; i < ir_node.outputs.size(); i++) {
            const auto out = state.output(i);
            if (const auto out_time = state.output_time(i); out && out_time) {
                out->resize(max_length);
                out_time->resize(max_length);
            }
        }

        state::Series longest_input_time;
        if (!ir_node.inputs.empty())
            longest_input_time = state.input_time(longest_input_idx);

        for (int i = 0; i < max_length; i++) {
            for (size_t j = 0; j < ir_node.inputs.size(); j++) {
                if (const auto input_series = state.input(j);
                    input_series && !input_series->empty()) {
                    const auto input_len = static_cast<int>(input_series->size());
                    inputs[j] = value_at(input_series, i % input_len);
                } else
                    inputs[j] = 0;
            }

            auto [results, err] = func.call(inputs);
            if (err) {
                ctx.report_error(
                    xerrors::Error(
                        "WASM execution failed in node " + ir_node.key + " at sample " +
                        std::to_string(i) + "/" + std::to_string(max_length) + ": " +
                        err.message()
                    )
                );
                continue;
            }

            uint64_t ts = 0;
            if (!ir_node.inputs.empty() && longest_input_time)
                ts = value_at(longest_input_time, i);
            else
                ts = static_cast<uint64_t>(telem::TimeStamp::now().nanoseconds());

            for (size_t j = 0; j < results.size(); j++) {
                if (!results[j].changed) continue;
                auto out = state.output(j);
                if (auto out_time = state.output_time(j); out && out_time) {
                    set_value_at(out, offsets[j], results[j].value);
                    set_value_at(out_time, offsets[j], ts);
                    offsets[j]++;
                }
            }
        }

        for (size_t j = 0; j < ir_node.outputs.size(); j++) {
            const auto out = state.output(j);
            if (const auto out_time = state.output_time(j); out && out_time) {
                out->resize(offsets[j]);
                out_time->resize(offsets[j]);
                if (offsets[j] > 0) ctx.mark_changed(ir_node.outputs[j].name);
            }
        }


        return xerrors::NIL;
    }
};
}
