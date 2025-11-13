// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <ranges>
#include <set>

#include "x/cpp/queue/spsc.h"
#include "x/cpp/telem/frame.h"

#include "arc/cpp/runtime/loop/loop.h"
#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/scheduler/scheduler.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/runtime/time/time.h"
#include "arc/cpp/runtime/wasm/factory.h"
#include "arc/cpp/runtime/wasm/module.h"

namespace arc::runtime {
struct Config {
    module::Module mod;
    breaker::Config breaker;
    std::function<std::pair<
        std::vector<state::ChannelDigest>,
        xerrors::Error>(const std::vector<types::ChannelKey> &)>
        retrieve_channels;
};

class Runtime {
    static constexpr size_t DEFAULT_QUEUE_CAPACITY = 1024;

    breaker::Breaker breaker;
    std::shared_ptr<wasm::Module> mod;
    std::unique_ptr<state::State> state;
    std::unique_ptr<scheduler::Scheduler> scheduler;
    std::unique_ptr<loop::Loop> loop;

    std::unique_ptr<queue::SPSC<telem::Frame>> inputs;
    std::unique_ptr<queue::SPSC<telem::Frame>> outputs;

public:
    Runtime(
        const breaker::Config &breaker_cfg,
        std::shared_ptr<wasm::Module> mod,
        std::unique_ptr<state::State> state,
        std::unique_ptr<scheduler::Scheduler> scheduler,
        std::unique_ptr<loop::Loop> loop
    ):
        breaker(breaker_cfg),
        mod(std::move(mod)),
        state(std::move(state)),
        scheduler(std::move(scheduler)),
        loop(std::move(loop)),
        inputs(std::make_unique<queue::SPSC<telem::Frame>>(DEFAULT_QUEUE_CAPACITY)),
        outputs(std::make_unique<queue::SPSC<telem::Frame>>(DEFAULT_QUEUE_CAPACITY)) {}

    void run() {
        while (this->breaker.running()) {
            this->loop->wait(this->breaker);
            telem::Frame frame;
            while (this->inputs->pop(frame))
                this->state->ingest(frame);

            this->scheduler->next();

            auto writes = this->state->flush_writes();

            if (!writes.empty()) {
                telem::Frame out_frame(writes.size());
                for (auto &[key, series]: writes)
                    out_frame.emplace(key, std::move(*series));
                this->outputs->push(std::move(out_frame));
            }

            this->state->clear_reads();
        }
    }

    bool write(telem::Frame frame) const {
        return this->inputs->push(std::move(frame));
    }

    bool read(telem::Frame &frame) const { return this->outputs->pop(frame); }
};

inline std::pair<std::unique_ptr<Runtime>, xerrors::Error> load(Config cfg) {

    // Step 1: Initialize state
    std::set<types::ChannelKey> reads;
    std::set<types::ChannelKey> writes;
    for (const auto &n: cfg.mod.nodes) {
        const auto read_keys = std::views::keys(n.channels.read);
        reads.insert(read_keys.begin(), read_keys.end());
        const auto write_keys = std::views::keys(n.channels.write);
        writes.insert(write_keys.begin(), write_keys.end());
    }

    std::vector<types::ChannelKey> keys;
    keys.reserve(reads.size() + writes.size());
    auto [digests, state_err] = cfg.retrieve_channels(keys);
    if (state_err) return {nullptr, state_err};
    for (const auto &d: digests) {
        if (reads.contains(d.key) && d.index != 0) reads.insert(d.index);
        if (writes.contains(d.key) && d.index != 0) writes.insert(d.index);
    }

    state::Config state_cfg{.ir = cfg.mod, .channels = digests};
    auto state = std::make_unique<state::State>(state_cfg);

    // Step 2: Initialize WASM Module
    wasm::ModuleConfig module_cfg{.module = cfg.mod};
    auto [mod, mod_err] = wasm::Module::open(module_cfg);
    if (mod_err) return {nullptr, mod_err};

    // Step 3: Put together factories.
    auto wasm_factory = std::make_unique<wasm::Factory>(mod);
    auto interval_factory = std::make_unique<time::Factory>();
    node::MultiFactory fact{wasm_factory, interval_factory};

    // Step 4: Construct nodes.
    std::unordered_map<std::string, std::unique_ptr<node::Node>> nodes;
    for (const auto &n: cfg.mod.nodes) {
        auto [node_state, node_state_err] = state->node(n.key);
        if (node_state_err) return {nullptr, node_state_err};
        auto [node, err] = fact.create(node::Config{
            .node = n,
            .state = node_state,
        });
        if (err) return {nullptr, err};
    }

    // Step 5: Construct scheduler.
    auto sched = std::make_unique<scheduler::Scheduler>(cfg.mod, nodes);

    // Step 6: Construct Loop
    auto [loop, err] = loop::create(loop::Config{});
    if (err) return {nullptr, err};

    // Step 6: Build Runtime
    return {
        std::make_unique<Runtime>(
            cfg.breaker,
            std::move(mod),
            std::move(state),
            std::move(sched),
            std::move(loop)
        ),
        xerrors::NIL
    };
}

}
