// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <memory>
#include <numeric>

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/loop/loop.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/stl/stl.h"
#include "arc/cpp/types/types.h"

namespace arc::stl::time {

inline constexpr const char *MODULE_NAME = "time";

/// @brief Sentinel value indicating base_interval hasn't been set yet.
inline const x::telem::TimeSpan UNSET_BASE_INTERVAL = x::telem::TimeSpan::max();

/// @brief Calculates the tolerance for timing comparisons based on execution mode.
inline x::telem::TimeSpan calculate_tolerance(
    const runtime::loop::ExecutionMode mode,
    const x::telem::TimeSpan base_interval
) {
    if (base_interval == UNSET_BASE_INTERVAL) return 5 * x::telem::MILLISECOND;
    const auto half = base_interval / 2;
    switch (mode) {
        case runtime::loop::ExecutionMode::RT_EVENT:
        case runtime::loop::ExecutionMode::BUSY_WAIT:
            return std::min(half, 100 * x::telem::MICROSECOND);
        case runtime::loop::ExecutionMode::HIGH_RATE:
            return std::min(half, x::telem::MILLISECOND);
        default:
            return std::min(half, 5 * x::telem::MILLISECOND);
    }
}

struct IntervalInputs {
    x::telem::TimeSpan interval;

    static std::pair<IntervalInputs, x::errors::Error>
    create(const types::Params &params) {
        const auto &param = params["period"];
        auto sv = types::to_sample_value(param.value, param.type);
        if (!sv.has_value())
            return {
                {},
                x::errors::Error(
                    x::errors::VALIDATION,
                    "interval node missing required period parameter"
                )
            };
        return {
            {.interval = x::telem::TimeSpan(x::telem::cast<std::int64_t>(*sv))},
            x::errors::NIL
        };
    }
};

class Interval : public runtime::node::Node {
    runtime::state::Node state;
    IntervalInputs inputs;
    x::telem::TimeSpan last_fired;

public:
    explicit Interval(const IntervalInputs &inputs, runtime::state::Node &&state):
        state(std::move(state)),
        inputs(inputs),
        last_fired(-1 * this->inputs.interval) {}

    x::errors::Error next(runtime::node::Context &ctx) override {
        if (ctx.reason != runtime::node::RunReason::TimerTick) {
            ctx.mark_self_changed();
            ctx.set_deadline(this->last_fired + this->inputs.interval);
            return x::errors::NIL;
        }
        if (ctx.elapsed - this->last_fired < this->inputs.interval - ctx.tolerance) {
            ctx.mark_self_changed();
            ctx.set_deadline(this->last_fired + this->inputs.interval);
            return x::errors::NIL;
        }
        this->last_fired = ctx.elapsed;
        ctx.mark_self_changed();
        ctx.set_deadline(this->last_fired + this->inputs.interval);
        const auto &o = this->state.output(0);
        const auto &o_time = this->state.output_time(0);
        o->resize(1);
        o_time->resize(1);
        o->set(0, static_cast<std::uint8_t>(1));
        o_time->set(0, ctx.elapsed.nanoseconds());
        ctx.mark_changed(0);
        return x::errors::NIL;
    }

    void reset() override { last_fired = -1 * inputs.interval; }

    [[nodiscard]] bool is_output_truthy(size_t output_idx) const override {
        return state.is_output_truthy(output_idx);
    }
};

struct WaitInputs {
    x::telem::TimeSpan duration;

    static std::pair<WaitInputs, x::errors::Error> create(const types::Params &params) {
        const auto &param = params["duration"];
        auto sv = types::to_sample_value(param.value, param.type);
        if (!sv.has_value())
            return {
                {},
                x::errors::Error(
                    x::errors::VALIDATION,
                    "wait node missing required duration parameter"
                )
            };
        return {
            {.duration = x::telem::TimeSpan(x::telem::cast<std::int64_t>(*sv))},
            x::errors::NIL
        };
    }
};

/// @brief One-shot timer that fires once after a specified duration.
class Wait : public runtime::node::Node {
    runtime::state::Node state;
    WaitInputs inputs;
    x::telem::TimeSpan start_time = x::telem::TimeSpan(-1);
    bool fired = false;

public:
    explicit Wait(const WaitInputs &inputs, runtime::state::Node &&state):
        state(std::move(state)), inputs(inputs) {}

    x::errors::Error next(runtime::node::Context &ctx) override {
        if (this->fired) return x::errors::NIL;
        if (this->start_time.nanoseconds() < 0) this->start_time = ctx.elapsed;
        ctx.set_deadline(this->start_time + this->inputs.duration);
        if (ctx.reason != runtime::node::RunReason::TimerTick) {
            ctx.mark_self_changed();
            return x::errors::NIL;
        }
        if (ctx.elapsed - this->start_time < this->inputs.duration - ctx.tolerance) {
            ctx.mark_self_changed();
            return x::errors::NIL;
        }
        this->fired = true;
        const auto &o = this->state.output(0);
        const auto &o_time = this->state.output_time(0);
        o->resize(1);
        o_time->resize(1);
        o->set(0, static_cast<std::uint8_t>(1));
        o_time->set(0, ctx.elapsed.nanoseconds());
        ctx.mark_changed(0);
        return x::errors::NIL;
    }

    void reset() override {
        start_time = x::telem::TimeSpan(-1);
        fired = false;
    }

    [[nodiscard]] bool is_output_truthy(size_t output_idx) const override {
        return state.is_output_truthy(output_idx);
    }
};

struct NowInputs {
    static std::pair<NowInputs, x::errors::Error> create(const types::Params &) {
        return {NowInputs{}, x::errors::NIL};
    }
};

/// @brief Outputs the current wall-clock timestamp when triggered.
class Now : public runtime::node::Node {
    runtime::state::Node state;
    x::telem::MonoClock *clock;

public:
    explicit Now(
        const NowInputs &,
        runtime::state::Node &&state,
        x::telem::MonoClock *clock
    ):
        state(std::move(state)), clock(clock) {}

    x::errors::Error next(runtime::node::Context &ctx) override {
        const auto ts = this->clock->now();
        const auto &o = this->state.output(0);
        const auto &o_time = this->state.output_time(0);
        o->resize(1);
        o_time->resize(1);
        o->set(0, ts);
        o_time->set(0, ts);
        ctx.mark_changed(0);
        return x::errors::NIL;
    }

    void reset() override {}

    [[nodiscard]] bool is_output_truthy(size_t output_idx) const override {
        return state.is_output_truthy(output_idx);
    }
};

class Module : public stl::Module {
    x::telem::TimeSpan base = UNSET_BASE_INTERVAL;
    x::telem::MonoClock clock;

public:
    /// @brief Returns the GCD of all interval/wait durations seen during node
    /// creation. Returns UNSET_BASE_INTERVAL if no time nodes were created.
    [[nodiscard]] x::telem::TimeSpan base_interval() const { return this->base; }

    bool handles(const std::string &node_type) const override {
        return node_type == "interval" || node_type == "wait" || node_type == "now";
    }

    std::pair<std::unique_ptr<runtime::node::Node>, x::errors::Error>
    create(runtime::node::Config &&cfg) override {
        if (cfg.node.type == "interval") {
            auto [inputs, err] = IntervalInputs::create(cfg.node.inputs);
            if (err) return {nullptr, err};
            this->update_base_interval(inputs.interval);
            return {
                std::make_unique<Interval>(inputs, std::move(cfg.state)),
                x::errors::NIL
            };
        }
        if (cfg.node.type == "wait") {
            auto [inputs, err] = WaitInputs::create(cfg.node.inputs);
            if (err) return {nullptr, err};
            this->update_base_interval(inputs.duration);
            return {
                std::make_unique<Wait>(inputs, std::move(cfg.state)),
                x::errors::NIL
            };
        }
        if (cfg.node.type == "now") {
            auto [inputs, err] = NowInputs::create(cfg.node.inputs);
            if (err) return {nullptr, err};
            return {
                std::make_unique<Now>(inputs, std::move(cfg.state), &this->clock),
                x::errors::NIL
            };
        }
        return {nullptr, x::errors::NOT_FOUND};
    }

    void bind_to(wasmtime::Linker &linker, wasmtime::Store::Context cx) override {
        linker
            .func_wrap(
                MODULE_NAME,
                "now",
                [this]() -> int64_t { return this->clock.now().nanoseconds(); }
            )
            .unwrap();
    }

private:
    void update_base_interval(const x::telem::TimeSpan span) {
        if (this->base == UNSET_BASE_INTERVAL)
            this->base = span;
        else
            this->base = x::telem::TimeSpan(
                std::gcd(this->base.nanoseconds(), span.nanoseconds())
            );
    }
};

}
