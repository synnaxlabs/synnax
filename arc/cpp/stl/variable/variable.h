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

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/stl/stl.h"

namespace arc::stl::variable {
inline const std::string symbol_name = "variable";
inline const std::string stateful_symbol_name = "stateful_variable";

/// @brief overwrites ts with a single sample of now, reusing its buffer.
inline void stamp_now(const runtime::state::Series &ts, const x::telem::TimeStamp now) {
    ts->resize(1);
    ts->set(0, now);
}

/// @brief Register holds what its variable is mapped to: a value, a channel key,
/// or a derivation index. Writes are last-wins; the unedged f0 holds the initial
/// value.
class Register : public runtime::node::Node {
    runtime::state::Node state;
    x::telem::MonoClock clock;
    bool stateful;

public:
    Register(runtime::state::Node &&state, const bool stateful):
        state(std::move(state)), stateful(stateful) {}

    /// @brief Reset restores a `:=` variable's initial value on scope entry. A `$=`
    /// persists. The value is emitted immediately, superseding any pending feeder.
    void reset() override {
        if (this->stateful) return;
        this->state.absorb_inputs();
        this->state.output(0)->copy_from(*this->state.input(0));
        stamp_now(this->state.output_time(0), this->clock.now());
    }

    x::errors::Error next(runtime::node::Context &ctx) override {
        auto [data, ok] = this->state.last_changed();
        if (!ok) return x::errors::NIL;
        // Feeders reuse their output buffers in place; the register value must not
        // alias them.
        this->state.output(0)->copy_from(*data);
        stamp_now(this->state.output_time(0), this->clock.now());
        ctx.mark_changed(0);
        return x::errors::NIL;
    }

    [[nodiscard]] bool is_output_truthy(size_t output_idx) const override {
        return this->state.is_output_truthy(output_idx);
    }
};

/// @brief ExprRead derefs its variable's dispatcher: values pending at a re-point
/// predate it and are absorbed, so only later inputs fire.
class ExprRead : public runtime::node::Node {
    runtime::state::Node state;
    x::telem::MonoClock clock;
    size_t sel_idx;

public:
    /// @brief marks an ExprRead constructed without a sel input.
    static constexpr size_t NO_SEL = ~size_t{0};

    ExprRead(runtime::state::Node &&state, const size_t sel_idx):
        state(std::move(state)), sel_idx(sel_idx) {}

    /// @brief Reset absorbs pending inputs, initial sel included, so only
    /// post-entry values fire.
    void reset() override { this->state.absorb_inputs(); }

    /// @brief Next re-points on sel first: the dispatcher never emits on a
    /// sel-only change, so a value paired with a fresh sel predates the re-point.
    x::errors::Error next(runtime::node::Context &ctx) override {
        bool repointed = false;
        if (this->sel_idx != NO_SEL && this->state.consume_input(this->sel_idx).second)
            repointed = true;
        auto [data, ok] = this->state.consume_input(0);
        if (!ok || repointed) return x::errors::NIL;
        this->state.output(0)->copy_from(*data);
        stamp_now(this->state.output_time(0), this->clock.now());
        ctx.mark_changed(0);
        return x::errors::NIL;
    }

    [[nodiscard]] bool is_output_truthy(size_t output_idx) const override {
        return this->state.is_output_truthy(output_idx);
    }
};

/// @brief Module is the runtime factory for the variable builtin.
class Module : public stl::Module {
public:
    bool handles(const std::string &node_type) const override {
        return node_type == symbol_name || node_type == stateful_symbol_name;
    }

    /// @brief Create dispatches on shape: a value-carrying first input makes a
    /// Register; an edge-fed one an ExprRead deref.
    std::pair<std::unique_ptr<runtime::node::Node>, x::errors::Error>
    create(runtime::node::Config &&cfg) override {
        if (!this->handles(cfg.node.type)) return {nullptr, x::errors::NOT_FOUND};
        if (!cfg.node.inputs.empty() && cfg.node.inputs[0].value.is_null()) {
            size_t sel_idx = ExprRead::NO_SEL;
            if (const auto [idx, err] = cfg.state.resolve_input("sel"); !err)
                sel_idx = idx;
            return {
                std::make_unique<ExprRead>(std::move(cfg.state), sel_idx),
                x::errors::NIL
            };
        }
        return {
            std::make_unique<Register>(
                std::move(cfg.state),
                cfg.node.type == stateful_symbol_name
            ),
            x::errors::NIL
        };
    }
};

}
