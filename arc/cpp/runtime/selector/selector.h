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
#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/state/state.h"

namespace arc::runtime::selector {

inline const std::string true_param = "true";
inline const std::string false_param = "false";

/// @brief Select routes u8 input to "true" or "false" outputs based on value.
/// Values equal to 1 go to "true" output, all others go to "false" output.
class Select : public node::Node {
    state::Node state;

public:
    explicit Select(state::Node &&state): state(std::move(state)) {}

    x::errors::Error next(node::Context &ctx) override {
        if (!this->state.refresh_inputs()) return x::errors::NIL;
        const auto &data = this->state.input(0);
        const auto &time = this->state.input_time(0);
        if (data->size() == 0) return x::errors::NIL;

        std::vector<uint8_t> true_values, false_values;
        std::vector<int64_t> true_times, false_times;
        for (size_t i = 0; i < data->size(); i++) {
            if (data->at<uint8_t>(i) == 1) {
                true_values.push_back(1);
                true_times.push_back(time->at<int64_t>(i));
            } else {
                false_values.push_back(0);
                false_times.push_back(time->at<int64_t>(i));
            }
        }

        auto &true_data = this->state.output(0);
        auto &true_time = this->state.output_time(0);
        auto &false_data = this->state.output(1);
        auto &false_time = this->state.output_time(1);
        *true_data = x::telem::Series(true_values);
        *true_time = x::telem::Series(true_times);
        *false_data = x::telem::Series(false_values);
        *false_time = x::telem::Series(false_times);

        if (!true_values.empty()) ctx.mark_changed(true_param);
        if (!false_values.empty()) ctx.mark_changed(false_param);
        return x::errors::NIL;
    }

    [[nodiscard]] bool is_output_truthy(const std::string &param_name) const override {
        return state.is_output_truthy(param_name);
    }
};

class Factory : public node::Factory {
public:
    bool handles(const std::string &node_type) const override {
        return node_type == "select";
    }

    std::pair<std::unique_ptr<node::Node>, x::errors::Error>
    create(node::Config &&cfg) override {
        if (!this->handles(cfg.node.type)) return {nullptr, x::errors::NOT_FOUND};
        return {std::make_unique<Select>(std::move(cfg.state)), x::errors::NIL};
    }
};

}
