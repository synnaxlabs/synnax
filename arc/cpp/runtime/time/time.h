// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <limits>
#include <memory>
#include <numeric>

#include "x/cpp/errors/errors.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/types/types.h"

namespace arc::runtime::time {

struct IntervalConfig {
    x::telem::TimeSpan interval;

    explicit IntervalConfig(const types::Params &params) {
        const auto param = types::find_param(params, "period");
        const auto interval_ns = param ? param->get().value.get<std::int64_t>() : 0;
        this->interval = x::telem::TimeSpan(interval_ns);
    }
};

class Interval : public node::Node {
    state::Node state;
    IntervalConfig cfg;
    x::telem::TimeSpan last_fired;

public:
    explicit Interval(const IntervalConfig &cfg, state::Node &&state):
        state(std::move(state)), cfg(cfg), last_fired(-1 * this->cfg.interval) {}

    x::errors::Error next(node::Context &ctx) override {
        if (ctx.elapsed - this->last_fired < this->cfg.interval) return x::errors::NIL;
        this->last_fired = ctx.elapsed;
        ctx.mark_changed(ir::default_output_param);
        const auto &o = this->state.output(0);
        const auto &o_time = this->state.output_time(0);
        o->resize(1);
        o_time->resize(1);
        o->set(0, static_cast<std::uint8_t>(1));
        o_time->set(0, ctx.elapsed.nanoseconds());
        return x::errors::NIL;
    }

    void reset() override { last_fired = -1 * cfg.interval; }

    [[nodiscard]] bool is_output_truthy(const std::string &param_name) const override {
        return state.is_output_truthy(param_name);
    }
};

struct WaitConfig {
    x::telem::TimeSpan duration;

    explicit WaitConfig(const types::Params &params) {
        const auto param = types::find_param(params, "duration");
        const auto duration_ns = param ? param->get().value.get<std::int64_t>() : 0;
        this->duration = x::telem::TimeSpan(duration_ns);
    }
};

/// Wait is a one-shot timer that fires once after a specified duration.
/// Unlike Interval, Wait only fires once and can be reset when a stage is entered.
class Wait : public node::Node {
    state::Node state;
    WaitConfig cfg;
    x::telem::TimeSpan start_time = x::telem::TimeSpan(-1);
    bool fired = false;

public:
    explicit Wait(const WaitConfig &cfg, state::Node &&state):
        state(std::move(state)), cfg(cfg) {}

    x::errors::Error next(node::Context &ctx) override {
        if (this->fired) return x::errors::NIL;
        if (this->start_time.nanoseconds() < 0) this->start_time = ctx.elapsed;
        if (ctx.elapsed - start_time < cfg.duration) return x::errors::NIL;
        this->fired = true;
        ctx.mark_changed(ir::default_output_param);
        const auto &o = state.output(0);
        const auto &o_time = state.output_time(0);
        o->resize(1);
        o_time->resize(1);
        o->set(0, static_cast<std::uint8_t>(1));
        o_time->set(0, ctx.elapsed.nanoseconds());
        return x::errors::NIL;
    }

    /// Reset the timer. Called when a stage containing this node is entered.
    void reset() override {
        start_time = x::telem::TimeSpan(-1);
        fired = false;
    }

    [[nodiscard]] bool is_output_truthy(const std::string &param_name) const override {
        return state.is_output_truthy(param_name);
    }
};

class Factory : public node::Factory {
public:
    x::telem::TimeSpan timing_base = x::telem::TimeSpan(
        std::numeric_limits<int64_t>::max()
    );

    bool handles(const std::string &node_type) const override {
        return node_type == "interval" || node_type == "wait";
    }

    std::pair<std::unique_ptr<node::Node>, x::errors::Error>
    create(node::Config &&cfg) override {
        if (cfg.node.type == "interval") {
            IntervalConfig node_cfg(cfg.node.config);
            this->update_timing_base(node_cfg.interval);
            return {
                std::make_unique<Interval>(node_cfg, std::move(cfg.state)),
                x::errors::NIL
            };
        }
        if (cfg.node.type == "wait") {
            WaitConfig node_cfg(cfg.node.config);
            this->update_timing_base(node_cfg.duration);
            return {
                std::make_unique<Wait>(node_cfg, std::move(cfg.state)),
                x::errors::NIL
            };
        }
        return {nullptr, x::errors::NOT_FOUND};
    }

private:
    void update_timing_base(const x::telem::TimeSpan span) {
        if (this->timing_base.nanoseconds() == std::numeric_limits<int64_t>::max())
            this->timing_base = span;
        else
            this->timing_base = x::telem::TimeSpan(
                std::gcd(this->timing_base.nanoseconds(), span.nanoseconds())
            );
    }
};
}
