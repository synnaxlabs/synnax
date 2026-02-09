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

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/loop/loop.h"
#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/stl/stl.h"

namespace arc::stl::time {

/// @brief Sentinel value indicating base_interval hasn't been set yet.
inline const ::x::telem::TimeSpan UNSET_BASE_INTERVAL = ::x::telem::TimeSpan::max();

/// @brief Calculates the tolerance for timing comparisons based on execution mode.
inline ::x::telem::TimeSpan calculate_tolerance(
    const runtime::loop::ExecutionMode mode,
    const ::x::telem::TimeSpan base_interval
) {
    if (base_interval == UNSET_BASE_INTERVAL) return 5 * ::x::telem::MILLISECOND;
    const auto half = base_interval / 2;
    switch (mode) {
        case runtime::loop::ExecutionMode::RT_EVENT:
        case runtime::loop::ExecutionMode::BUSY_WAIT:
            return std::min(half, 100 * ::x::telem::MICROSECOND);
        case runtime::loop::ExecutionMode::HIGH_RATE:
            return std::min(half, ::x::telem::MILLISECOND);
        default:
            return std::min(half, 5 * ::x::telem::MILLISECOND);
    }
}

struct IntervalConfig {
    ::x::telem::TimeSpan interval;

    explicit IntervalConfig(const ir::Params &params) {
        const auto interval_ns = params["period"].get<std::int64_t>();
        this->interval = ::x::telem::TimeSpan(interval_ns);
    }
};

class Interval : public runtime::node::Node {
    runtime::state::Node state;
    IntervalConfig cfg;
    ::x::telem::TimeSpan last_fired;

public:
    explicit Interval(const IntervalConfig &cfg, runtime::state::Node &&state):
        state(std::move(state)), cfg(cfg), last_fired(-1 * this->cfg.interval) {}

    x::errors::Error next(runtime::node::Context &ctx) override {
        if (ctx.reason != runtime::node::RunReason::TimerTick) return x::errors::NIL;
        if (ctx.elapsed - this->last_fired < this->cfg.interval - ctx.tolerance)
            return x::errors::NIL;
        this->last_fired = ctx.elapsed;
        const auto &o = this->state.output(0);
        const auto &o_time = this->state.output_time(0);
        o->resize(1);
        o_time->resize(1);
        o->set(0, static_cast<std::uint8_t>(1));
        o_time->set(0, ctx.elapsed.nanoseconds());
        ctx.mark_changed(ir::default_output_param);
        return x::errors::NIL;
    }

    void reset() override { last_fired = -1 * cfg.interval; }

    [[nodiscard]] bool is_output_truthy(const std::string &param_name) const override {
        return state.is_output_truthy(param_name);
    }
};

struct WaitConfig {
    ::x::telem::TimeSpan duration;

    explicit WaitConfig(const ir::Params &params) {
        const auto duration_ns = params["duration"].get<std::int64_t>();
        this->duration = ::x::telem::TimeSpan(duration_ns);
    }
};

/// @brief One-shot timer that fires once after a specified duration.
class Wait : public runtime::node::Node {
    runtime::state::Node state;
    WaitConfig cfg;
    ::x::telem::TimeSpan start_time = ::x::telem::TimeSpan(-1);
    bool fired = false;

public:
    explicit Wait(const WaitConfig &cfg, runtime::state::Node &&state):
        state(std::move(state)), cfg(cfg) {}

    x::errors::Error next(runtime::node::Context &ctx) override {
        if (ctx.reason != runtime::node::RunReason::TimerTick) return x::errors::NIL;
        if (this->fired) return x::errors::NIL;
        if (this->start_time.nanoseconds() < 0) this->start_time = ctx.elapsed;
        if (ctx.elapsed - this->start_time < this->cfg.duration - ctx.tolerance)
            return x::errors::NIL;
        this->fired = true;
        const auto &o = this->state.output(0);
        const auto &o_time = this->state.output_time(0);
        o->resize(1);
        o_time->resize(1);
        o->set(0, static_cast<std::uint8_t>(1));
        o_time->set(0, ctx.elapsed.nanoseconds());
        ctx.mark_changed(ir::default_output_param);
        return x::errors::NIL;
    }

    void reset() override {
        start_time = ::x::telem::TimeSpan(-1);
        fired = false;
    }

    [[nodiscard]] bool is_output_truthy(const std::string &param_name) const override {
        return state.is_output_truthy(param_name);
    }
};

class Factory : public runtime::node::Factory {
    ::x::telem::TimeSpan base = UNSET_BASE_INTERVAL;

public:
    /// @brief Returns the GCD of all interval/wait durations seen during node
    /// creation. Returns UNSET_BASE_INTERVAL if no time nodes were created.
    [[nodiscard]] ::x::telem::TimeSpan base_interval() const { return this->base; }

    bool handles(const std::string &node_type) const override {
        return node_type == "interval" || node_type == "wait";
    }

    std::pair<std::unique_ptr<runtime::node::Node>, x::errors::Error>
    create(runtime::node::Config &&cfg) override {
        if (cfg.node.type == "interval") {
            IntervalConfig node_cfg(cfg.node.config);
            this->update_base_interval(node_cfg.interval);
            return {
                std::make_unique<Interval>(node_cfg, std::move(cfg.state)),
                x::errors::NIL
            };
        }
        if (cfg.node.type == "wait") {
            WaitConfig node_cfg(cfg.node.config);
            this->update_base_interval(node_cfg.duration);
            return {
                std::make_unique<Wait>(node_cfg, std::move(cfg.state)),
                x::errors::NIL
            };
        }
        return {nullptr, x::errors::NOT_FOUND};
    }

private:
    void update_base_interval(const ::x::telem::TimeSpan span) {
        if (this->base == UNSET_BASE_INTERVAL)
            this->base = span;
        else
            this->base = ::x::telem::TimeSpan(
                std::gcd(this->base.nanoseconds(), span.nanoseconds())
            );
    }
};

class Module : public stl::Module {
public:
    void bind_to(wasmtime::Linker &linker, wasmtime::Store::Context cx) override {
        linker
            .func_wrap(
                "time",
                "now",
                []() -> int64_t { return ::x::telem::TimeStamp::now().nanoseconds(); }
            )
            .unwrap();
    }
};

}
