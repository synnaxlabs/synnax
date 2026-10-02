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

#include "wasmtime.hh"

#include "x/cpp/errors/errors.h"
#include "x/cpp/os/os.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/errors/errors.h"
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
    // Variable durations leave the base unset, and their timers wake on a deadline.
    if (base_interval == UNSET_BASE_INTERVAL) return 100 * x::telem::MICROSECOND;
    const auto half = base_interval / 2;
    if (mode == runtime::loop::ExecutionMode::HIGH_RATE)
        return std::min(half, x::telem::MILLISECOND);
    return std::min(half, 100 * x::telem::MICROSECOND);
}

/// @brief returns the named input's current span: the referenced variable's
/// latest value when var-bound, else the value stamped at compile time.
inline x::telem::TimeSpan
live_span(const runtime::state::Node &s, const std::string &name) {
    return x::telem::TimeSpan(s.numeric_input<int64_t>(name));
}

/// @brief returns the minimum span and where it applies, for messages to the user.
inline std::string describe_min(const x::telem::TimeSpan min) {
    return min.to_string() + " (" + x::os::get() + ")";
}

/// @brief rejects a span stamped at compile time that is non-positive or under
/// min. Var-bound params are exempt: the runtime guard covers their live values.
inline x::errors::Error validate_static_span(
    const x::telem::TimeSpan span,
    const types::Param &p,
    const x::telem::TimeSpan min
) {
    if (p.type.kind == types::Kind::VarRef) return x::errors::NIL;
    if (span.nanoseconds() <= 0)
        return x::errors::Error(
            x::errors::VALIDATION,
            p.name + " must be positive, got " + span.to_string()
        );
    if (span < min)
        return x::errors::Error(
            x::errors::VALIDATION,
            p.name + " must be at least " + describe_min(min) + ", got " +
                span.to_string()
        );
    return x::errors::NIL;
}

/// @brief guards a live timer span against non-positive values and values under
/// min. Reports the first offense only, so a node does not re-report on every pass.
class SpanGuard {
    x::telem::TimeSpan min;
    bool reported = false;
    bool raised = false;

public:
    explicit SpanGuard(const x::telem::TimeSpan min): min(min) {}

    /// @brief returns true when span can drive a deadline. A non-positive span
    /// reports a validation error naming label and returns false.
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
                    x::errors::VALIDATION,
                    label + " must be positive, got " + span.to_string()
                )
            );
            this->reported = true;
        }
        return false;
    }

    /// @brief returns span, or min when span is shorter. A short span reports a
    /// warning naming label.
    x::telem::TimeSpan raise(
        runtime::node::Context &ctx,
        const x::telem::TimeSpan span,
        const std::string &label
    ) {
        if (span >= this->min) {
            this->raised = false;
            return span;
        }
        if (!this->raised) {
            ctx.report_error(
                x::errors::Error(
                    runtime::errors::WARNING,
                    label + " must be at least " + describe_min(this->min) + ", got " +
                        span.to_string() + ", using " + this->min.to_string()
                )
            );
            this->raised = true;
        }
        return this->min;
    }

    /// @brief returns span, or min when span is shorter, with no report.
    [[nodiscard]] x::telem::TimeSpan floor(const x::telem::TimeSpan span) const {
        return std::max(span, this->min);
    }

    /// @brief arms the validation report again. The warning stays reported, so a
    /// stage that loops does not repeat it.
    void reset() { this->reported = false; }
};

struct IntervalInputs {
    x::telem::TimeSpan interval;

    static std::pair<IntervalInputs, x::errors::Error> create(
        const types::Params &params,
        const x::telem::TimeSpan min = x::telem::TimeSpan(0)
    ) {
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
        if (auto err = validate_static_span(period, param, min)) return {{}, err};
        return {{.interval = period}, x::errors::NIL};
    }
};

class Interval : public runtime::node::Node {
    runtime::state::Node state;
    x::telem::TimeSpan last_fired;
    SpanGuard guard;

public:
    explicit Interval(
        runtime::state::Node &&state,
        const x::telem::TimeSpan period,
        const x::telem::TimeSpan min = x::telem::TimeSpan(0)
    ):
        state(std::move(state)), last_fired(-1 * std::max(period, min)), guard(min) {}

    x::errors::Error next(runtime::node::Context &ctx) override {
        const auto live = live_span(this->state, "period");
        // A non-positive period is a configuration error. Park without a deadline; a
        // later reassignment to a positive value resumes the timer.
        if (!this->guard.usable(ctx, live, "interval period")) return x::errors::NIL;
        const auto period = this->guard.raise(ctx, live, "period");
        if (ctx.cycle.reason != runtime::node::RunReason::TimerTick) {
            ctx.mark_self_changed();
            ctx.set_deadline(this->last_fired + period);
            return x::errors::NIL;
        }
        if (ctx.cycle.elapsed - this->last_fired < period - ctx.tolerance) {
            ctx.mark_self_changed();
            ctx.set_deadline(this->last_fired + period);
            return x::errors::NIL;
        }
        this->last_fired = ctx.cycle.elapsed;
        ctx.mark_self_changed();
        ctx.set_deadline(this->last_fired + period);
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
        this->last_fired = -1 * this->guard.floor(live_span(this->state, "period"));
        this->guard.reset();
    }

    [[nodiscard]] bool is_output_truthy(size_t output_idx) const override {
        return state.is_output_truthy(output_idx);
    }
};

struct WaitInputs {
    x::telem::TimeSpan duration;

    static std::pair<WaitInputs, x::errors::Error> create(
        const types::Params &params,
        const x::telem::TimeSpan min = x::telem::TimeSpan(0)
    ) {
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
        if (auto err = validate_static_span(duration, param, min)) return {{}, err};
        return {{.duration = duration}, x::errors::NIL};
    }
};

/// @brief One-shot timer that fires once after a specified duration.
class Wait : public runtime::node::Node {
    runtime::state::Node state;
    x::telem::TimeSpan start_time = x::telem::TimeSpan(-1);
    bool fired = false;
    SpanGuard guard;

public:
    explicit Wait(
        runtime::state::Node &&state,
        const x::telem::TimeSpan min = x::telem::TimeSpan(0)
    ):
        state(std::move(state)), guard(min) {}

    x::errors::Error next(runtime::node::Context &ctx) override {
        if (this->fired) return x::errors::NIL;
        const auto live = live_span(this->state, "duration");
        // A non-positive duration is a configuration error, not an instant fire: park.
        // Timing stays anchored to start_time, so recovery re-checks the live duration
        // against the original activation.
        if (!this->guard.usable(ctx, live, "wait duration")) return x::errors::NIL;
        const auto duration = this->guard.raise(ctx, live, "duration");
        if (this->start_time.nanoseconds() < 0) this->start_time = ctx.cycle.elapsed;
        ctx.set_deadline(this->start_time + duration);
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
    x::telem::TimeSpan base = UNSET_BASE_INTERVAL;
    /// @brief the current cycle's stamp, set by the runtime loop before each
    /// pass. The `now` WASM binding is called from guest code, which has no node
    /// Context to read, so the value is pushed here instead.
    x::telem::TimeStamp now;
    /// @brief the shortest literal timer span the module accepts.
    const x::telem::TimeSpan min_span;

public:
    explicit Module(
        const x::telem::TimeSpan min_span = runtime::loop::min_timer_span()
    ):
        min_span(min_span) {}

    /// @brief binds the cycle stamp the `now` host function returns for the
    /// coming pass. The runtime loop calls it before every Scheduler::next.
    void set_now(const x::telem::TimeStamp now) { this->now = now; }

    /// @brief Returns the GCD of all interval/wait durations seen during node
    /// creation. Returns UNSET_BASE_INTERVAL if no time nodes were created.
    [[nodiscard]] x::telem::TimeSpan base_interval() const { return this->base; }

    bool handles(const std::string &node_type) const override {
        return node_type == "interval" || node_type == "wait" || node_type == "now";
    }

    std::pair<std::unique_ptr<runtime::node::Node>, x::errors::Error>
    create(runtime::node::Config &&cfg) override {
        if (cfg.node.type == "interval") {
            auto [inputs, err] = IntervalInputs::create(
                cfg.node.inputs,
                this->min_span
            );
            if (err) return {nullptr, err};
            this->update_base_interval(inputs.interval);
            this->fold_reassigned_spans(cfg, cfg.node.inputs["period"]);
            return {
                std::make_unique<Interval>(
                    std::move(cfg.state),
                    inputs.interval,
                    this->min_span
                ),
                x::errors::NIL
            };
        }
        if (cfg.node.type == "wait") {
            auto [inputs, err] = WaitInputs::create(cfg.node.inputs, this->min_span);
            if (err) return {nullptr, err};
            this->update_base_interval(inputs.duration);
            this->fold_reassigned_spans(cfg, cfg.node.inputs["duration"]);
            return {
                std::make_unique<Wait>(std::move(cfg.state), this->min_span),
                x::errors::NIL
            };
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
    /// into base_interval, so tolerance tracks the fastest known period.
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
                    this->update_base_interval(
                        x::telem::TimeSpan(x::telem::cast<int64_t>(*sv))
                    );
            }
        }
    }

    void update_base_interval(const x::telem::TimeSpan raw) {
        // A non-positive span is not a real timer period. Folding it in would
        // poison the GCD and drive the loop cadence off a parked timer.
        if (raw.nanoseconds() <= 0) return;
        // A timer holds a span under the minimum at the minimum.
        const auto span = std::max(raw, this->min_span);
        if (this->base == UNSET_BASE_INTERVAL)
            this->base = span;
        else
            this->base = x::telem::TimeSpan(
                std::gcd(this->base.nanoseconds(), span.nanoseconds())
            );
    }
};

}
