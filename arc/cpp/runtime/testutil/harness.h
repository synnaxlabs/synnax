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
#include <ranges>
#include <set>
#include <stdexcept>
#include <string>
#include <unordered_map>
#include <utility>
#include <vector>

#include "client/cpp/synnax.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/program/program.h"
#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/loop/loop.h"
#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/scheduler/scheduler.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/runtime/testutil/compile.h"
#include "arc/cpp/runtime/wasm/factory.h"
#include "arc/cpp/runtime/wasm/module.h"
#include "arc/cpp/stl/channels/channels.h"
#include "arc/cpp/stl/channels/state.h"
#include "arc/cpp/stl/constant/constant.h"
#include "arc/cpp/stl/control/control.h"
#include "arc/cpp/stl/error/error.h"
#include "arc/cpp/stl/math/math.h"
#include "arc/cpp/stl/selector/selector.h"
#include "arc/cpp/stl/series/series.h"
#include "arc/cpp/stl/stable/stable.h"
#include "arc/cpp/stl/stateful/stateful.h"
#include "arc/cpp/stl/strings/strings.h"
#include "arc/cpp/stl/time/time.h"
#include "arc/cpp/stl/variable/variable.h"

namespace arc::runtime::testutil {
/// @brief Harness provides a full end-to-end test harness that compiles Arc source
/// code and executes it through the scheduler with real wasm nodes.
class Harness {
    arc::program::Program program;
    std::shared_ptr<stl::channels::State> channel_state;
    std::shared_ptr<state::State> node_state;
    std::shared_ptr<wasm::Module> wasm_module;
    std::unique_ptr<scheduler::Scheduler> sched;
    x::telem::Alignment alignment;

public:
    Harness(const synnax::Synnax &client, const std::string &source):
        Harness(client, compile_text(client, source)) {}

    Harness(const synnax::Synnax &client, arc::program::Program prog):
        program(std::move(prog)) {
        std::set<types::ChannelKey> reads;
        std::set<types::ChannelKey> writes;
        for (const auto &n: this->program.nodes) {
            const auto read_keys = std::views::keys(n.channels.read);
            reads.insert(read_keys.begin(), read_keys.end());
            const auto write_keys = std::views::keys(n.channels.write);
            writes.insert(write_keys.begin(), write_keys.end());
        }
        for (const auto &[key, val]: this->program.authorities.channels)
            writes.insert(key);

        std::vector<types::ChannelKey> keys;
        keys.insert(keys.end(), reads.begin(), reads.end());
        keys.insert(keys.end(), writes.begin(), writes.end());
        auto [channels, ch_err] = client.channels.retrieve(keys);
        if (ch_err)
            throw std::runtime_error(
                "Failed to retrieve channels: " + ch_err.message()
            );
        std::vector<state::ChannelDigest> digests;
        for (const auto &ch: channels)
            digests.push_back({ch.key, ch.data_type, ch.index});

        this->channel_state = std::make_shared<stl::channels::State>(digests);
        auto str_st = std::make_shared<stl::strings::State>();
        auto series_st = std::make_shared<stl::series::State>();
        auto var_st = std::make_shared<stl::stateful::Variables>();
        state::Config state_cfg{
            .ir = static_cast<ir::IR>(this->program),
            .channels = digests
        };
        this->node_state = std::make_shared<state::State>(
            state_cfg,
            this->channel_state,
            str_st,
            series_st,
            var_st,
            errors::noop_handler
        );

        auto time_mod = std::make_shared<stl::time::Module>();
        const std::vector<std::shared_ptr<stl::Module>> stl_modules{
            std::make_shared<stl::channels::Module>(this->channel_state, str_st),
            std::make_shared<stl::stateful::Module>(var_st, series_st, str_st),
            std::make_shared<stl::series::Module>(series_st),
            std::make_shared<stl::strings::Module>(str_st),
            std::make_shared<stl::math::Module>(),
            time_mod,
            std::make_shared<stl::error::Module>(errors::noop_handler),
            std::make_shared<stl::constant::Module>(),
            std::make_shared<stl::control::Module>(this->node_state),
            std::make_shared<stl::stable::Module>(),
            std::make_shared<stl::selector::Module>(),
            std::make_shared<stl::variable::Module>(),
        };

        std::vector<std::shared_ptr<node::Factory>> factories;
        if (!this->program.wasm.empty()) {
            auto [mod, mod_err] = wasm::Module::open({
                .program = this->program,
                .modules = stl_modules,
                .strings = str_st,
            });
            if (mod_err)
                throw std::runtime_error(
                    "Failed to open wasm module: " + mod_err.message()
                );
            this->wasm_module = std::move(mod);
            factories.push_back(std::make_shared<wasm::Factory>(this->wasm_module));
        }
        for (auto &m: stl_modules)
            factories.push_back(m);
        node::MultiFactory fact(factories);

        std::unordered_map<std::string, std::unique_ptr<node::Node>> nodes;
        const ir::IR prog_ir = static_cast<ir::IR>(this->program);
        for (const auto &ir_node: this->program.nodes) {
            auto [ns, ns_err] = this->node_state->node(ir_node.key);
            if (ns_err) throw std::runtime_error(ns_err.message());
            auto [n, err] = fact.create(node::Config(prog_ir, ir_node, std::move(ns)));
            if (err)
                throw std::runtime_error(
                    "Failed to create node " + ir_node.key + ": " + err.message()
                );
            nodes[ir_node.key] = std::move(n);
        }

        const auto base_interval = time_mod->base_interval();
        const auto loop_cfg = loop::Config{}.apply_defaults(base_interval);
        const auto tolerance = stl::time::calculate_tolerance(
            loop_cfg.mode,
            base_interval
        );
        this->sched = std::make_unique<scheduler::Scheduler>(
            prog_ir,
            nodes,
            tolerance,
            errors::noop_handler
        );
    }

    void tick(const x::telem::TimeSpan elapsed) {
        this->sched->next(elapsed, node::RunReason::TimerTick);
    }

    void ingest(const types::ChannelKey channel_key, x::telem::Series &&data) {
        data.alignment = this->alignment;
        this->alignment += x::telem::Alignment(data.size());
        x::telem::Frame fr(1);
        fr.emplace(channel_key, std::move(data));
        this->node_state->ingest(fr);
    }

    void ingest_indexed(
        const types::ChannelKey index_key,
        x::telem::Series &&timestamps,
        const types::ChannelKey data_key,
        x::telem::Series &&data
    ) {
        timestamps.alignment = this->alignment;
        data.alignment = this->alignment;
        this->alignment += x::telem::Alignment(data.size());
        x::telem::Frame fr(2);
        fr.emplace(index_key, std::move(timestamps));
        fr.emplace(data_key, std::move(data));
        this->node_state->ingest(fr);
    }

    [[nodiscard]] x::telem::Frame flush() const {
        x::telem::Frame fr;
        this->node_state->flush_into(fr);
        return fr;
    }

    [[nodiscard]] x::telem::Series
    output(const std::string &node_key, const size_t param_idx) const {
        auto [n, err] = this->node_state->node(node_key);
        if (err) throw std::runtime_error(err.message());
        return n.output(param_idx)->deep_copy();
    }

    [[nodiscard]] x::telem::Series
    output_time(const std::string &node_key, const size_t param_idx) const {
        auto [n, err] = this->node_state->node(node_key);
        if (err) throw std::runtime_error(err.message());
        return n.output_time(param_idx)->deep_copy();
    }

    /// @brief drains and returns all authority changes buffered by
    /// set_authority nodes this cycle. Tests assert on the returned vector to
    /// verify authority semantics that aren't observable via channel writes.
    [[nodiscard]] std::vector<state::AuthorityChange> flush_authority() const {
        return this->node_state->flush_authority_changes();
    }
};
}
