// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

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
};

class Factory : public node::Factory {
public:
    telem::TimeSpan timing_base = telem::TimeSpan(std::numeric_limits<int64_t>::max());

    std::pair<std::unique_ptr<node::Node>, xerrors::Error>
    create(const node::Config &cfg) override {
        if (cfg.node.type != "interval") return {nullptr, xerrors::NOT_FOUND};
        IntervalConfig node_cfg(cfg.node.config);
        if (timing_base.nanoseconds() == std::numeric_limits<int64_t>::max())
            timing_base = node_cfg.interval;
        else
            timing_base = telem::TimeSpan(
                std::gcd(timing_base.nanoseconds(), node_cfg.interval.nanoseconds())
            );

        auto node = std::make_unique<Interval>(node_cfg, cfg.state);
        return {std::move(node), xerrors::NIL};
    }
};
}
