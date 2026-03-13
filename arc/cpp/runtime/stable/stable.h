// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <functional>
#include <memory>
#include <optional>

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/state/state.h"

namespace arc::runtime::stable {

struct StableForConfig {
    x::telem::TimeSpan duration;

    explicit StableForConfig(const ir::Params &params) {
        const auto duration_ns = params["duration"].get<std::int64_t>();
        this->duration = x::telem::TimeSpan(duration_ns);
    }
};

using NowFn = std::function<x::telem::TimeStamp()>;

inline NowFn default_now = [] { return x::telem::TimeStamp::now(); };

/// @brief StableFor outputs a value only after the input has remained unchanged
/// for a configured duration. Used to debounce noisy signals.
///
/// Stability is measured from the input sample's timestamp (not scheduler
/// elapsed time), and the current time is obtained via an injectable now()
/// function, matching the Go runtime behavior.
class StableFor : public node::Node {
    state::Node state;
    StableForConfig cfg;
    NowFn now;
    std::optional<uint8_t> value;
    std::optional<uint8_t> last_sent;
    x::telem::TimeStamp last_changed{0};

public:
    explicit StableFor(
        const StableForConfig &cfg,
        state::Node &&state,
        NowFn now = default_now
    ):
        state(std::move(state)), cfg(cfg), now(std::move(now)) {}

    x::errors::Error next(node::Context &ctx) override {
        if (this->state.refresh_inputs()) {
            const auto &input_data = this->state.input(0);
            const auto &input_time = this->state.input_time(0);
            if (input_data->size() > 0) {
                for (size_t i = 0; i < input_data->size(); i++) {
                    const auto current_value = input_data->at<uint8_t>(i);
                    if (!this->value.has_value() || *this->value != current_value) {
                        this->value = current_value;
                        this->last_changed = x::telem::TimeStamp(
                            input_time->at<int64_t>(i)
                        );
                    }
                }
            }
        }

        if (!this->value.has_value()) return x::errors::NIL;
        const auto current_value = *this->value;
        const auto current_time = this->now();
        if (x::telem::TimeSpan(current_time - this->last_changed) >=
            this->cfg.duration) {
            if (!this->last_sent.has_value() || *this->last_sent != current_value) {
                const auto &o = this->state.output(0);
                const auto &o_time = this->state.output_time(0);
                *o = x::telem::Series(current_value);
                *o_time = x::telem::Series(current_time.nanoseconds());
                this->last_sent = current_value;
                ctx.mark_changed(ir::default_output_param);
            }
        }
        return x::errors::NIL;
    }

    void reset() override {
        this->value = std::nullopt;
        this->last_sent = std::nullopt;
        this->last_changed = x::telem::TimeStamp(0);
    }

    [[nodiscard]] bool is_output_truthy(const std::string &param_name) const override {
        return state.is_output_truthy(param_name);
    }
};

class Factory : public node::Factory {
public:
    bool handles(const std::string &node_type) const override {
        return node_type == "stable_for";
    }

    std::pair<std::unique_ptr<node::Node>, x::errors::Error>
    create(node::Config &&cfg) override {
        if (!this->handles(cfg.node.type)) return {nullptr, x::errors::NOT_FOUND};
        StableForConfig node_cfg(cfg.node.config);
        return {
            std::make_unique<StableFor>(node_cfg, std::move(cfg.state)),
            x::errors::NIL
        };
    }
};

}
