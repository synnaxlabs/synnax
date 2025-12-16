// Copyright 2025 Synnax Labs, Inc.
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

#include "x/cpp/xerrors/errors.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/node/node.h"

namespace arc::runtime::time {

struct IntervalConfig {
    telem::TimeSpan interval;

    explicit IntervalConfig(const ir::Params &params) {
        auto interval_ns = params.get("period")->value.get<std::int64_t>();
        this->interval = telem::TimeSpan(interval_ns);
    }
};

class Interval : public node::Node {
    state::Node state;
    IntervalConfig cfg;
    telem::TimeSpan last_fired = telem::TimeSpan(-1);

public:
    explicit Interval(const IntervalConfig cfg, state::Node state):
        state(state), cfg(cfg), last_fired(-1 * cfg.interval) {}

    xerrors::Error next(node::Context &ctx) override {
        if (ctx.elapsed - this->last_fired < this->cfg.interval) return xerrors::NIL;
        this->last_fired = ctx.elapsed;
        ctx.mark_changed(ir::default_output_param);
        auto &o = this->state.output(0);
        auto &o_time = this->state.output_time(0);
        o->resize(1);
        o_time->resize(1);
        o->set(0, static_cast<std::uint8_t>(1));
        o_time->set(0, ctx.elapsed.nanoseconds());
        return xerrors::NIL;
    }

    [[nodiscard]] bool is_output_truthy(const std::string &param_name) const override {
        return state.is_output_truthy(param_name);
    }
};

struct WaitConfig {
    telem::TimeSpan duration;

    explicit WaitConfig(const ir::Params &params) {
        auto duration_ns = params.get("duration")->value.get<std::int64_t>();
        this->duration = telem::TimeSpan(duration_ns);
    }
};

/// Wait is a one-shot timer that fires once after a specified duration.
/// Unlike Interval, Wait only fires once and can be reset when a stage is entered.
class Wait : public node::Node {
    state::Node state;
    WaitConfig cfg;
    telem::TimeSpan start_time = telem::TimeSpan(-1);
    bool fired = false;

public:
    explicit Wait(WaitConfig cfg, state::Node state):
        state(std::move(state)), cfg(cfg) {}

    xerrors::Error next(node::Context &ctx) override {
        if (this->fired) return xerrors::NIL;
        if (this->start_time.nanoseconds() < 0) { start_time = ctx.elapsed; }

        // Check if duration has elapsed
        if (ctx.elapsed - start_time < cfg.duration) return xerrors::NIL;

        // Fire!
        fired = true;
        ctx.mark_changed(ir::default_output_param);
        auto &o = state.output(0);
        auto &o_time = state.output_time(0);
        o->resize(1);
        o_time->resize(1);
        o->set(0, static_cast<std::uint8_t>(1));
        o_time->set(0, ctx.elapsed.nanoseconds());
        return xerrors::NIL;
    }

    /// Reset the timer. Called when a stage containing this node is entered.
    void reset() override {
        start_time = telem::TimeSpan(-1);
        fired = false;
    }

    [[nodiscard]] bool is_output_truthy(const std::string &param_name) const override {
        return state.is_output_truthy(param_name);
    }
};

class Factory : public node::Factory {
public:
    telem::TimeSpan timing_base = telem::TimeSpan(std::numeric_limits<int64_t>::max());

    std::pair<std::unique_ptr<node::Node>, xerrors::Error>
    create(const node::Config &cfg) override {
        if (cfg.node.type == "interval") {
            IntervalConfig node_cfg(cfg.node.config);
            update_timing_base(node_cfg.interval);
            auto node = std::make_unique<Interval>(node_cfg, cfg.state);
            return {std::move(node), xerrors::NIL};
        }

        if (cfg.node.type == "wait") {
            WaitConfig node_cfg(cfg.node.config);
            update_timing_base(node_cfg.duration);
            auto node = std::make_unique<Wait>(node_cfg, cfg.state);
            return {std::move(node), xerrors::NIL};
        }

        return {nullptr, xerrors::NOT_FOUND};
    }

private:
    void update_timing_base(telem::TimeSpan span) {
        if (timing_base.nanoseconds() == std::numeric_limits<int64_t>::max())
            timing_base = span;
        else
            timing_base = telem::TimeSpan(
                std::gcd(timing_base.nanoseconds(), span.nanoseconds())
            );
    }
};
}
