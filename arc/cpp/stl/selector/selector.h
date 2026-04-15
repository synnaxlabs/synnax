// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <string>

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/stl/stl.h"

namespace arc::stl::selector {

inline const std::string true_param = "true";
inline const std::string false_param = "false";

/// @brief Select routes a u8 input to its "true" or "false" output based on
/// value: input samples equal to 1 produce samples on the "true" output,
/// all others produce samples on the "false" output. Both outputs emit a
/// value of 1 for each matched sample — the output value signals "this
/// route fired" rather than echoing the input — so the output can drive
/// a truthy-gated transition whether the matched route is true or false.
class Select : public runtime::node::Node {
    runtime::state::Node state;

public:
    explicit Select(runtime::state::Node &&state): state(std::move(state)) {}

    x::errors::Error next(runtime::node::Context &ctx) override {
        if (!this->state.refresh_inputs()) return x::errors::NIL;
        const auto &data = this->state.input(0);
        const auto &time = this->state.input_time(0);
        const auto n = data->size();
        if (n == 0) return x::errors::NIL;

        // Count true values first to pre-size output buffers.
        size_t true_count = 0;
        for (size_t i = 0; i < n; i++)
            if (data->at<uint8_t>(i) == 1) true_count++;
        const size_t false_count = n - true_count;

        auto &true_data = this->state.output(0);
        auto &true_time = this->state.output_time(0);
        auto &false_data = this->state.output(1);
        auto &false_time = this->state.output_time(1);
        true_data->resize(true_count);
        true_time->resize(true_count);
        false_data->resize(false_count);
        false_time->resize(false_count);
        true_data->alignment = data->alignment;
        true_data->time_range = data->time_range;
        true_time->alignment = data->alignment;
        true_time->time_range = data->time_range;
        false_data->alignment = data->alignment;
        false_data->time_range = data->time_range;
        false_time->alignment = data->alignment;
        false_time->time_range = data->time_range;

        size_t ti = 0, fi = 0;
        for (size_t i = 0; i < n; i++) {
            if (data->at<uint8_t>(i) == 1) {
                true_data->set(static_cast<int>(ti), static_cast<uint8_t>(1));
                true_time->set(static_cast<int>(ti), time->at<int64_t>(i));
                ti++;
            } else {
                false_data->set(static_cast<int>(fi), static_cast<uint8_t>(1));
                false_time->set(static_cast<int>(fi), time->at<int64_t>(i));
                fi++;
            }
        }

        if (true_count > 0) ctx.mark_changed(true_param);
        if (false_count > 0) ctx.mark_changed(false_param);
        return x::errors::NIL;
    }

    [[nodiscard]] bool is_output_truthy(const std::string &param_name) const override {
        return state.is_output_truthy(param_name);
    }
};

class Module : public stl::Module {
public:
    bool handles(const std::string &node_type) const override {
        return node_type == "select";
    }

    std::pair<std::unique_ptr<runtime::node::Node>, x::errors::Error>
    create(runtime::node::Config &&cfg) override {
        if (!this->handles(cfg.node.type)) return {nullptr, x::errors::NOT_FOUND};
        return {std::make_unique<Select>(std::move(cfg.state)), x::errors::NIL};
    }
};

}
