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

#include "wasmtime.hh"

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/stl/stl.h"
#include "arc/cpp/types/types.h"

namespace arc::stl::time {

inline constexpr const char *MODULE_NAME = "time";

/// @brief Sentinel value indicating shortest_span hasn't been set yet.
inline const x::telem::TimeSpan UNSET_SHORTEST_SPAN = x::telem::TimeSpan::max();

/// @brief the most a timer may fire before its deadline. The loop wakes on the earliest
/// deadline, so a wider tolerance only fires a timer early on the wake of another.
inline const x::telem::TimeSpan MAX_TOLERANCE = 100 * x::telem::MICROSECOND;

/// @brief returns the timing tolerance for the given shortest span: half the span, and
/// MAX_TOLERANCE at most.
inline x::telem::TimeSpan calculate_tolerance(const x::telem::TimeSpan shortest_span) {
    return std::min(shortest_span / 2, MAX_TOLERANCE);
}

/// @brief returns the named input's current span: the referenced variable's
/// latest value when var-bound, else the value stamped at compile time.
inline x::telem::TimeSpan
live_span(const runtime::state::Node &s, const std::string &name) {
    return x::telem::TimeSpan(s.numeric_input<int64_t>(name));
}

/// @brief rejects a non-positive span stamped at compile time. Var-bound params are
/// exempt: the runtime guard covers their live values.
inline x::errors::Error
validate_static_span(const x::telem::TimeSpan span, const types::Param &p) {
    if (p.type.kind == types::Kind::VarRef || span.nanoseconds() > 0)
        return x::errors::NIL;
    return x::errors::Error(
        x::errors::VALIDATION,
        p.name + " must be positive, got " + span.to_string()
    );
}

/// @brief guards a live timer span against non-positive values. Reports the
/// first offense only, so a parked node does not re-report on every pass.
class SpanGuard {
    bool reported = false;

public:
    /// @brief returns true when span can drive a deadline. A non-positive span
    /// reports a warning naming label and returns false.
    bool usable(
        runtime::node::Context &ctx,
        const x::telem::TimeSpan span,
        const std::string &label
    ) {
        if (span.nanoseconds() > 0) {
            this->reported = false;
            return true;
        }
        if (!this->reported) {
            ctx.report_error(
                x::errors::Error(
                    runtime::errors::WARNING,
                    label + " must be positive, got " + span.to_string()
                )
            );
            this->reported = true;
        }
        return false;
    }

    void reset() { this->reported = false; }
};

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
        const auto period = x::telem::TimeSpan(x::telem::cast<std::int64_t>(*sv));
        if (auto err = validate_static_span(period, param)) return {{}, err};
        return {{.interval = period}, x::errors::NIL};
    }
};

class Interval : public runtime::node::Node {
    runtime::state::Node state;
    /// @brief the elapsed time of the last scheduled fire. Valid only when started.
    x::telem::TimeSpan last_fired;
    /// @brief true once the interval has anchored its schedule at a usable period.
    bool started = false;
    /// @brief reports a non-positive period once.
    SpanGuard guard;

public:
    explicit Interval(runtime::state::Node &&state): state(std::move(state)) {}

    x::errors::Error next(runtime::node::Context &ctx) override {
        const auto period = live_span(this->state, "period");
        // A non-positive period is a configuration error. Park without a deadline until
        // the node runs again, such as when its stage is entered again.
        if (!this->guard.usable(ctx, period, "interval period")) return x::errors::NIL;
        // The first usable run puts the first fire at now, so the interval fires on its
        // first timer tick.
        if (!this->started) {
            this->last_fired = ctx.cycle.elapsed - period;
            this->started = true;
        }
        if (ctx.cycle.reason != runtime::node::RunReason::TimerTick) {
            ctx.mark_self_changed();
            ctx.set_deadline(this->last_fired + period, period);
            return x::errors::NIL;
        }
        if (ctx.cycle.elapsed - this->last_fired < period - ctx.tolerance) {
            ctx.mark_self_changed();
            ctx.set_deadline(this->last_fired + period, period);
            return x::errors::NIL;
        }
        // Fires count from the schedule, so a late fire does not delay the fires after
        // it. A fire a full period behind restarts the schedule from now.
        this->last_fired += period;
        if (ctx.cycle.elapsed - this->last_fired >= period)
            this->last_fired = ctx.cycle.elapsed;
        ctx.mark_self_changed();
        ctx.set_deadline(this->last_fired + period, period);
        const auto &o = this->state.output(0);
        const auto &o_time = this->state.output_time(0);
        o->resize(1);
        o_time->resize(1);
        o->set(0, static_cast<std::uint8_t>(1));
        o_time->set(0, ctx.cycle.now);
        this->state.emit(ctx.mark_changed, 0);
        return x::errors::NIL;
    }

    /// @brief resets the interval so it fires immediately on the next timer tick.
    void reset(runtime::node::Context &) override {
        this->state.reset();
        this->started = false;
        this->guard.reset();
    }

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
        const auto duration = x::telem::TimeSpan(x::telem::cast<std::int64_t>(*sv));
        if (auto err = validate_static_span(duration, param)) return {{}, err};
        return {{.duration = duration}, x::errors::NIL};
    }
};

/// @brief One-shot timer that fires once after a specified duration.
class Wait : public runtime::node::Node {
    runtime::state::Node state;
    /// @brief the elapsed time the wait started, or -1 before its first usable run.
    x::telem::TimeSpan start_time = x::telem::TimeSpan(-1);
    /// @brief true once the wait has fired in the current activation.
    bool fired = false;
    /// @brief reports a non-positive duration once.
    SpanGuard guard;

public:
    explicit Wait(runtime::state::Node &&state): state(std::move(state)) {}

    x::errors::Error next(runtime::node::Context &ctx) override {
        if (this->fired) return x::errors::NIL;
        const auto duration = live_span(this->state, "duration");
        // A non-positive duration is a configuration error, not an instant fire. Park
        // without a deadline until the node runs again, such as when its stage is
        // entered again.
        if (!this->guard.usable(ctx, duration, "wait duration")) return x::errors::NIL;
        if (this->start_time.nanoseconds() < 0) this->start_time = ctx.cycle.elapsed;
        ctx.set_deadline(this->start_time + duration, duration);
        if (ctx.cycle.reason != runtime::node::RunReason::TimerTick) {
            ctx.mark_self_changed();
            return x::errors::NIL;
        }
        if (ctx.cycle.elapsed - this->start_time < duration - ctx.tolerance) {
            ctx.mark_self_changed();
            return x::errors::NIL;
        }
        this->fired = true;
        const auto &o = this->state.output(0);
        const auto &o_time = this->state.output_time(0);
        o->resize(1);
        o_time->resize(1);
        o->set(0, static_cast<std::uint8_t>(1));
        o_time->set(0, ctx.cycle.now);
        this->state.emit(ctx.mark_changed, 0);
        return x::errors::NIL;
    }

    void reset(runtime::node::Context &) override {
        this->state.reset();
        this->start_time = x::telem::TimeSpan(-1);
        this->fired = false;
        this->guard.reset();
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

public:
    explicit Now(const NowInputs &, runtime::state::Node &&state):
        state(std::move(state)) {}

    x::errors::Error next(runtime::node::Context &ctx) override {
        const auto ts = ctx.cycle.now;
        const auto &o = this->state.output(0);
        const auto &o_time = this->state.output_time(0);
        o->resize(1);
        o_time->resize(1);
        o->set(0, ts);
        o_time->set(0, ts);
        this->state.emit(ctx.mark_changed, 0);
        return x::errors::NIL;
    }

    void reset(runtime::node::Context &) override { this->state.reset(); }

    [[nodiscard]] bool is_output_truthy(size_t output_idx) const override {
        return state.is_output_truthy(output_idx);
    }
};

class Module : public stl::Module {
    /// @brief the shortest timer span of the program, or UNSET_SHORTEST_SPAN.
    x::telem::TimeSpan shortest = UNSET_SHORTEST_SPAN;
    /// @brief the current cycle's stamp, set by the runtime loop before each
    /// pass. The `now` WASM binding is called from guest code, which has no node
    /// Context to read, so the value is pushed here instead.
    x::telem::TimeStamp now;

public:
    /// @brief binds the cycle stamp the `now` host function returns for the
    /// coming pass. The runtime loop calls it before every Scheduler::next.
    void set_now(const x::telem::TimeStamp now) { this->now = now; }

    /// @brief returns the shortest literal interval or wait span seen during node
    /// creation, or UNSET_SHORTEST_SPAN if there is none.
    [[nodiscard]] x::telem::TimeSpan shortest_span() const { return this->shortest; }

    bool handles(const std::string &node_type) const override {
        return node_type == "interval" || node_type == "wait" || node_type == "now";
    }

    std::pair<std::unique_ptr<runtime::node::Node>, x::errors::Error>
    create(runtime::node::Config &&cfg) override {
        if (cfg.node.type == "interval") {
            auto [inputs, err] = IntervalInputs::create(cfg.node.inputs);
            if (err) return {nullptr, err};
            this->update_shortest_span(inputs.interval);
            this->fold_reassigned_spans(cfg, cfg.node.inputs["period"]);
            return {std::make_unique<Interval>(std::move(cfg.state)), x::errors::NIL};
        }
        if (cfg.node.type == "wait") {
            auto [inputs, err] = WaitInputs::create(cfg.node.inputs);
            if (err) return {nullptr, err};
            this->update_shortest_span(inputs.duration);
            this->fold_reassigned_spans(cfg, cfg.node.inputs["duration"]);
            return {std::make_unique<Wait>(std::move(cfg.state)), x::errors::NIL};
        }
        if (cfg.node.type == "now") {
            auto [inputs, err] = NowInputs::create(cfg.node.inputs);
            if (err) return {nullptr, err};
            return {
                std::make_unique<Now>(inputs, std::move(cfg.state)),
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
                [this]() -> int64_t { return this->now.nanoseconds(); }
            )
            .unwrap();
    }

private:
    /// @brief folds the literal reassignment values of a var-bound timer param
    /// into shortest_span, so tolerance tracks the fastest known period.
    void
    fold_reassigned_spans(const runtime::node::Config &cfg, const types::Param &p) {
        if (p.type.kind != types::Kind::VarRef) return;
        for (const auto &e: cfg.prog.edges) {
            if (e.target.node != p.type.name) continue;
            const auto *src = ir::find_node(cfg.prog, e.source.node);
            if (src == nullptr || src->type != "constant") continue;
            for (const auto &v: src->inputs) {
                if (v.name != "value" || v.value.is_null()) continue;
                if (const auto sv = types::to_sample_value(v.value, v.type))
                    this->update_shortest_span(
                        x::telem::TimeSpan(x::telem::cast<int64_t>(*sv))
                    );
            }
        }
    }

    void update_shortest_span(const x::telem::TimeSpan raw) {
        // A non-positive span parks its timer, so it is not a period.
        if (raw.nanoseconds() <= 0) return;
        this->shortest = std::min(this->shortest, raw);
    }
};

}
