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
#include <utility>

#include "x/cpp/queue/spsc.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/xthread/xthread.h"

#include "arc/cpp/runtime/constant/constant.h"
#include "arc/cpp/runtime/loop/loop.h"
#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/scheduler/scheduler.h"
#include "arc/cpp/runtime/stage/stage.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/runtime/telem/telem.h"
#include "arc/cpp/runtime/time/time.h"
#include "arc/cpp/runtime/wasm/bindings.h"
#include "arc/cpp/runtime/wasm/factory.h"
#include "arc/cpp/runtime/wasm/module.h"

namespace arc::runtime {
struct Config {
    module::Module mod;
    breaker::Config breaker;
    std::function<std::pair<std::vector<state::ChannelDigest>, xerrors::Error>(
        const std::vector<types::ChannelKey> &
    )>
        retrieve_channels;
    size_t input_queue_capacity = 256;
    size_t output_queue_capacity = 1024;
};

class Runtime {
    breaker::Breaker breaker;
    std::thread run_thread;
    std::shared_ptr<wasm::Module> mod;
    std::shared_ptr<wasm::Bindings> bindings;
    std::shared_ptr<state::State> state;
    scheduler::Scheduler scheduler;
    std::unique_ptr<loop::Loop> loop;
    queue::SPSC<telem::Frame> inputs;
    queue::SPSC<telem::Frame> outputs;
    telem::TimeStamp start_time = telem::TimeStamp(0);

public:
    std::vector<types::ChannelKey> read_channels;
    std::vector<types::ChannelKey> write_channels;
    Runtime(
        const Config &cfg,
        std::shared_ptr<wasm::Module> mod,
        std::shared_ptr<wasm::Bindings> bindings_runtime,
        std::shared_ptr<state::State> state,
        scheduler::Scheduler scheduler,
        std::unique_ptr<loop::Loop> loop,
        const std::vector<types::ChannelKey> &read_channels,
        std::vector<types::ChannelKey> write_channels
    ):
        breaker(cfg.breaker),
        mod(std::move(mod)),
        bindings(std::move(bindings_runtime)),
        state(std::move(state)),
        scheduler(std::move(scheduler)),
        loop(std::move(loop)),
        inputs(queue::SPSC<telem::Frame>(cfg.input_queue_capacity)),
        outputs(queue::SPSC<telem::Frame>(cfg.output_queue_capacity)),
        read_channels(read_channels),
        write_channels(std::move(write_channels)) {
        this->loop->watch(this->inputs.notifier());
    }

    void run() {
        this->start_time = telem::TimeStamp::now();
        xthread::set_name("runtime");
        this->loop->start();
        while (this->breaker.running()) {
            this->loop->wait(this->breaker);
            telem::Frame frame;
            bool first = true;
            while (this->inputs.try_pop(frame) || first) {
                first = false;
                this->state->ingest(frame);
                const auto elapsed = telem::TimeStamp::now() - this->start_time;
                this->scheduler.next(elapsed);
                if (auto writes = this->state->flush(); !writes.empty()) {
                    telem::Frame out_frame(writes.size());
                    for (auto &[key, series]: writes)
                        out_frame.emplace(key, series->deep_copy());
                    this->outputs.push(std::move(out_frame));
                }
            }
        }
    }

    bool start() {
        if (this->breaker.running()) return false;
        this->breaker.start();
        this->run_thread = std::thread([this]() { this->run(); });
        return true;
    }

    /// @brief closes the output queue, unblocking any pending read() calls.
    /// Call this before stopping consumers of the output queue.
    void close_outputs() { this->outputs.close(); }

    bool stop() {
        if (!this->breaker.stop()) return false;
        this->loop->stop();
        this->run_thread.join();
        this->inputs.close();
        this->outputs.close();
        return true;
    }

    xerrors::Error write(telem::Frame frame) {
        if (!this->inputs.push(std::move(frame)))
            return xerrors::Error("runtime closed");
        return xerrors::NIL;
    }

    bool read(telem::Frame &frame) { return this->outputs.pop(frame); }
};

inline std::pair<std::shared_ptr<Runtime>, xerrors::Error> load(const Config &cfg) {
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

    state::Config state_cfg{.ir = (static_cast<ir::IR>(cfg.mod)), .channels = digests};
    auto state = std::make_shared<state::State>(state_cfg);
    auto bindings_runtime = std::make_shared<wasm::Bindings>(state, nullptr);

    wasm::ModuleConfig module_cfg{
        .module = cfg.mod,
        .bindings = bindings_runtime,
    };
    auto [mod, mod_err] = wasm::Module::open(module_cfg);
    if (mod_err) return {nullptr, mod_err};

    auto wasm_factory = std::make_shared<wasm::Factory>(mod);
    auto time_factory = std::make_shared<time::Factory>();
    auto stage_factory = std::make_shared<stage::Factory>();
    auto io_factory = std::make_shared<io::Factory>();
    auto constant_factory = std::make_shared<constant::Factory>();
    node::MultiFactory fact(
        std::vector<std::shared_ptr<node::Factory>>{
            wasm_factory,
            time_factory,
            stage_factory,
            io_factory,
            constant_factory,
        }
    );

    std::unordered_map<std::string, std::unique_ptr<node::Node>> nodes;
    for (const auto &n: cfg.mod.nodes) {
        auto [node_state, node_state_err] = state->node(n.key);
        if (node_state_err) return {nullptr, node_state_err};
        auto [node, err] = fact.create(node::Config(n, std::move(node_state)));
        if (err) return {nullptr, err};
        nodes[n.key] = std::move(node);
    }

    auto sched = scheduler::Scheduler(cfg.mod, nodes);

    auto timing_interval = time_factory->timing_base;
    const bool has_intervals = timing_interval.nanoseconds() !=
                               std::numeric_limits<int64_t>::max();

    auto mode = loop::ExecutionMode::EVENT_DRIVEN;
    if (has_intervals && timing_interval < loop::timing::SOFTWARE_TIMER_THRESHOLD)
        mode = loop::ExecutionMode::HIGH_RATE;

    auto [loop, err] = loop::create(
        loop::Config{
            .mode = mode,
            .interval = has_intervals ? timing_interval : telem::TimeSpan(0),
            .rt_priority = 47,
            .cpu_affinity = -1,
        }
    );
    if (err) return {nullptr, err};

    return {
        std::make_shared<Runtime>(
            cfg,
            std::move(mod),
            bindings_runtime,
            state,
            std::move(sched),
            std::move(loop),
            std::vector(reads.begin(), reads.end()),
            std::vector(writes.begin(), writes.end())
        ),
        xerrors::NIL
    };
}

}
