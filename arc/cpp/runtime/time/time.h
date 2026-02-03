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
#include <numeric>

#include "x/cpp/xerrors/errors.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/loop/loop.h"
#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/node/node.h"

namespace arc::runtime::time {

/// @brief Sentinel value indicating base_interval hasn't been set yet.
/// Using TimeSpan::max() ensures any real interval will be smaller and will replace it.
inline const telem::TimeSpan UNSET_BASE_INTERVAL = telem::TimeSpan::max();

/// @brief Calculates the tolerance for timing comparisons based on execution mode.
inline telem::TimeSpan calculate_tolerance(
    const loop::ExecutionMode mode,
    const telem::TimeSpan base_interval
) {
    if (base_interval == UNSET_BASE_INTERVAL) return 5 * telem::MILLISECOND;
    const auto half = base_interval / 2;
    switch (mode) {
        case loop::ExecutionMode::RT_EVENT:
        case loop::ExecutionMode::BUSY_WAIT:
            return std::min(half, 100 * telem::MICROSECOND);
        case loop::ExecutionMode::HIGH_RATE:
            return std::min(half, telem::MILLISECOND);
        default:
            return std::min(half, 5 * telem::MILLISECOND);
    }
}

struct IntervalConfig {
    telem::TimeSpan interval;

    explicit IntervalConfig(const ir::Params &params) {
        const auto interval_ns = params["period"].get<std::int64_t>();
        this->interval = telem::TimeSpan(interval_ns);
    }
};

class Interval : public node::Node {
    state::Node state;
    IntervalConfig cfg;
    telem::TimeSpan last_fired;

public:
    explicit Interval(const IntervalConfig &cfg, state::Node &&state):
        state(std::move(state)), cfg(cfg), last_fired(-1 * this->cfg.interval) {}

    xerrors::Error next(node::Context &ctx) override {
        if (ctx.elapsed - this->last_fired < this->cfg.interval - ctx.tolerance)
            return xerrors::NIL;
        this->last_fired = ctx.elapsed;
        const auto &o = this->state.output(0);
        const auto &o_time = this->state.output_time(0);
        o->resize(1);
        o_time->resize(1);
        o->set(0, static_cast<std::uint8_t>(1));
        o_time->set(0, ctx.elapsed.nanoseconds());
        ctx.mark_changed(ir::default_output_param);
        return xerrors::NIL;
    }

    void reset() override { last_fired = -1 * cfg.interval; }

    [[nodiscard]] bool is_output_truthy(const std::string &param_name) const override {
        return state.is_output_truthy(param_name);
    }
};

struct WaitConfig {
    telem::TimeSpan duration;

    explicit WaitConfig(const ir::Params &params) {
        const auto duration_ns = params["duration"].get<std::int64_t>();
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
    explicit Wait(const WaitConfig &cfg, state::Node &&state):
        state(std::move(state)), cfg(cfg) {}

    xerrors::Error next(node::Context &ctx) override {
        if (this->fired) return xerrors::NIL;
        if (this->start_time.nanoseconds() < 0) this->start_time = ctx.elapsed;
        if (ctx.elapsed - start_time < cfg.duration - ctx.tolerance)
            return xerrors::NIL;
        this->fired = true;
        const auto &o = state.output(0);
        const auto &o_time = state.output_time(0);
        o->resize(1);
        o_time->resize(1);
        o->set(0, static_cast<std::uint8_t>(1));
        o_time->set(0, ctx.elapsed.nanoseconds());
        ctx.mark_changed(ir::default_output_param);
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
    telem::TimeSpan base_interval = UNSET_BASE_INTERVAL;

    bool handles(const std::string &node_type) const override {
        return node_type == "interval" || node_type == "wait";
    }

    std::pair<std::unique_ptr<node::Node>, xerrors::Error>
    create(node::Config &&cfg) override {
        if (cfg.node.type == "interval") {
            IntervalConfig node_cfg(cfg.node.config);
            this->update_base_interval(node_cfg.interval);
            return {
                std::make_unique<Interval>(node_cfg, std::move(cfg.state)),
                xerrors::NIL
            };
        }
        if (cfg.node.type == "wait") {
            WaitConfig node_cfg(cfg.node.config);
            this->update_base_interval(node_cfg.duration);
            return {
                std::make_unique<Wait>(node_cfg, std::move(cfg.state)),
                xerrors::NIL
            };
        }
        return {nullptr, xerrors::NOT_FOUND};
    }

private:
    void update_base_interval(const telem::TimeSpan span) {
        if (this->base_interval == UNSET_BASE_INTERVAL)
            this->base_interval = span;
        else
            this->base_interval = telem::TimeSpan(
                std::gcd(this->base_interval.nanoseconds(), span.nanoseconds())
            );
    }
};
}
