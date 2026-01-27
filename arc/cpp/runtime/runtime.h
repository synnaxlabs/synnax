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

#include "glog/logging.h"

#include "x/cpp/queue/spsc.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/thread/thread.h"

#include "arc/cpp/runtime/constant/constant.h"
#include "arc/cpp/runtime/errors/errors.h"
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
    x::breaker::Config breaker;
    std::function<std::pair<std::vector<state::ChannelDigest>, x::errors::Error>(
        const std::vector<types::ChannelKey> &
    )>
        retrieve_channels;
    size_t input_queue_capacity = 256;
    size_t output_queue_capacity = 1024;
    /// @brief Loop configuration. Fields with default values are auto-selected.
    loop::Config loop;
};

/// @brief callback invoked when a fatal error occurs in the runtime.
using ErrorHandler = std::function<void(const x::errors::Error &)>;

class Runtime {
    x::breaker::Breaker breaker;
    std::thread run_thread;
    std::shared_ptr<wasm::Module> mod;
    std::shared_ptr<wasm::Bindings> bindings;
    std::shared_ptr<state::State> state;
    std::unique_ptr<scheduler::Scheduler> scheduler;
    std::unique_ptr<loop::Loop> loop;
    x::queue::SPSC<x::telem::Frame> inputs;
    x::queue::SPSC<x::telem::Frame> outputs;
    x::telem::TimeStamp start_time = telem::TimeStamp(0);
    errors::Handler error_handler;

public:
    std::vector<types::ChannelKey> read_channels;
    std::vector<types::ChannelKey> write_channels;
    Runtime(
        const Config &cfg,
        std::shared_ptr<wasm::Module> mod,
        std::shared_ptr<wasm::Bindings> bindings_runtime,
        std::shared_ptr<state::State> state,
        std::unique_ptr<scheduler::Scheduler> scheduler,
        std::unique_ptr<loop::Loop> loop,
        const std::vector<types::ChannelKey> &read_channels,
        std::vector<types::ChannelKey> write_channels,
        errors::Handler error_handler = errors::noop_handler
    ):
        breaker(cfg.breaker),
        mod(std::move(mod)),
        bindings(std::move(bindings_runtime)),
        state(std::move(state)),
        scheduler(std::move(scheduler)),
        loop(std::move(loop)),
        inputs(x::queue::SPSC<x::telem::Frame>(cfg.input_queue_capacity)),
        outputs(x::queue::SPSC<x::telem::Frame>(cfg.output_queue_capacity)),
        error_handler(std::move(error_handler)),
        read_channels(read_channels),
        write_channels(std::move(write_channels)) {}

    void run() {
        this->start_time = x::telem::TimeStamp::now();
        x::thread::set_name("runtime");
        this->loop->start();
        if (!this->loop->watch(this->inputs.notifier())) {
            LOG(ERROR) << "[runtime] failed to watch input notifier";
            this->error_handler(x::errors::Error("failed to watch input notifier"));
            return;
        }
        while (this->breaker.running()) {
            this->loop->wait(this->breaker);
            x::telem::Frame frame;
            bool first = true;
            while (this->inputs.try_pop(frame) || first) {
                first = false;
                this->state->ingest(frame);
                const auto elapsed = x::telem::TimeStamp::now() - this->start_time;
                this->scheduler->next(elapsed);
                if (auto writes = this->state->flush(); !writes.empty()) {
                    x::telem::Frame out_frame(writes.size());
                    for (auto &[key, series]: writes)
                        out_frame.emplace(key, series->deep_copy());
                    if (!this->outputs.push(std::move(out_frame)))
                        this->error_handler(errors::QUEUE_FULL_OUTPUT);
                }
            }
        }
    }

    bool start() {
        if (this->breaker.running()) return false;
        this->inputs.reset();
        this->outputs.reset();
        this->breaker.start();
        this->run_thread = std::thread([this]() { this->run(); });
        return true;
    }

    /// @brief closes the output queue, unblocking any pending read() calls.
    /// Call this before stopping consumers of the output queue.
    void close_outputs() { this->outputs.close(); }

    bool stop() {
        if (!this->breaker.stop()) return false;
        this->loop->wake();
        this->run_thread.join();
        this->inputs.close();
        this->outputs.close();
        this->state->reset();
        this->scheduler->reset();
        return true;
    }

    x::errors::Error write(x::telem::Frame frame) {
        if (this->inputs.closed()) return x::errors::Error("runtime closed");
        if (!this->inputs.push(std::move(frame))) {
            this->error_handler(errors::QUEUE_FULL_INPUT);
            return errors::QUEUE_FULL_INPUT;
        }
        return x::errors::NIL;
    }

    bool read(x::telem::Frame &frame) { return this->outputs.pop(frame); }
};

inline std::pair<std::shared_ptr<Runtime>, x::errors::Error>
load(const Config &cfg, errors::Handler error_handler = errors::noop_handler) {
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
    keys.insert(keys.end(), reads.begin(), reads.end());
    keys.insert(keys.end(), writes.begin(), writes.end());
    auto [digests, state_err] = cfg.retrieve_channels(keys);
    if (state_err) return {nullptr, state_err};
    for (const auto &d: digests) {
        if (reads.contains(d.key) && d.index != 0) reads.insert(d.index);
        if (writes.contains(d.key) && d.index != 0) writes.insert(d.index);
    }

    state::Config state_cfg{.ir = (static_cast<ir::IR>(cfg.mod)), .channels = digests};
    auto state = std::make_shared<state::State>(state_cfg, error_handler);
    auto bindings_runtime = std::make_shared<wasm::Bindings>(
        state,
        nullptr,
        error_handler
    );

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
    const ir::IR prog = static_cast<ir::IR>(cfg.mod);
    for (const auto &mod_node: cfg.mod.nodes) {
        auto [node_state, node_state_err] = state->node(mod_node.key);
        if (node_state_err) return {nullptr, node_state_err};
        auto [node, err] = fact.create(
            node::Config(prog, mod_node, std::move(node_state))
        );
        if (err) return {nullptr, err};
        nodes[mod_node.key] = std::move(node);
    }
    auto sched = std::make_unique<scheduler::Scheduler>(cfg.mod, nodes, error_handler);
    auto [loop, err] = loop::create(cfg.loop.apply_defaults(time_factory->timing_base));
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
            std::vector(writes.begin(), writes.end()),
            std::move(error_handler)
        ),
        x::errors::NIL
    };
}

}
