// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <chrono>
#include <memory>
#include <ranges>
#include <set>
#include <utility>

#include "glog/logging.h"

#include "x/cpp/queue/spsc.h"
#include "x/cpp/telem/control.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/xthread/xthread.h"

#include "arc/cpp/runtime/authority/authority.h"
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
    breaker::Config breaker;
    std::function<std::pair<std::vector<state::ChannelDigest>, xerrors::Error>(
        const std::vector<types::ChannelKey> &
    )>
        retrieve_channels;
    size_t input_queue_capacity = 256;
    size_t output_queue_capacity = 1024;
    /// @brief Loop configuration. Fields with default values are auto-selected.
    loop::Config loop;
};

/// @brief callback invoked when a fatal error occurs in the runtime.
using ErrorHandler = std::function<void(const xerrors::Error &)>;

class Runtime {
    breaker::Breaker breaker;
    std::thread run_thread;
    std::shared_ptr<wasm::Module> mod;
    std::shared_ptr<wasm::Bindings> bindings;
    std::shared_ptr<state::State> state;
    std::unique_ptr<scheduler::Scheduler> scheduler;
    std::unique_ptr<loop::Loop> loop;
    queue::SPSC<telem::Frame> inputs;
    queue::SPSC<telem::Frame> outputs;
    queue::SPSC<std::vector<state::AuthorityChange>> authority_outputs;
    std::chrono::steady_clock::time_point start_time_steady_;
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
        inputs(queue::SPSC<telem::Frame>(cfg.input_queue_capacity)),
        outputs(queue::SPSC<telem::Frame>(cfg.output_queue_capacity)),
        authority_outputs(queue::SPSC<std::vector<state::AuthorityChange>>(64)),
        error_handler(std::move(error_handler)),
        read_channels(read_channels),
        write_channels(std::move(write_channels)) {}

    void run() {
        this->start_time_steady_ = std::chrono::steady_clock::now();
        xthread::set_name("runtime");
        this->loop->start();
        if (!this->loop->watch(this->inputs.notifier())) {
            LOG(ERROR) << "[runtime] failed to watch input notifier";
            this->error_handler(xerrors::Error("failed to watch input notifier"));
            return;
        }
        while (this->breaker.running()) {
            const auto wake_reason = this->loop->wait(this->breaker);
            const bool is_timer = (wake_reason == loop::WakeReason::Timer);
            telem::Frame frame;
            bool first = true;
            while (this->inputs.try_pop(frame) || first) {
                const auto reason = (first && is_timer) ? node::RunReason::TimerTick
                                                        : node::RunReason::ChannelInput;
                first = false;
                this->state->ingest(frame);
                const auto now_steady = std::chrono::steady_clock::now();
                const auto elapsed = telem::TimeSpan(
                    std::chrono::duration_cast<std::chrono::nanoseconds>(
                        now_steady - this->start_time_steady_
                    )
                        .count()
                );
                this->scheduler->next(elapsed, reason);
                if (auto writes = this->state->flush(); !writes.empty()) {
                    telem::Frame out_frame(writes.size());
                    for (auto &[key, series]: writes)
                        out_frame.emplace(key, series->deep_copy());
                    if (!this->outputs.push(std::move(out_frame)))
                        this->error_handler(errors::QUEUE_FULL_OUTPUT);
                }
                if (auto changes = this->state->flush_authority_changes();
                    !changes.empty())
                    this->authority_outputs.push(std::move(changes));
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

    xerrors::Error write(telem::Frame frame) {
        if (this->inputs.closed()) return xerrors::Error("runtime closed");
        if (!this->inputs.push(std::move(frame))) {
            this->error_handler(errors::QUEUE_FULL_INPUT);
            return errors::QUEUE_FULL_INPUT;
        }
        return xerrors::NIL;
    }

    bool read(telem::Frame &frame) { return this->outputs.pop(frame); }

    bool read_authority_changes(std::vector<state::AuthorityChange> &changes) {
        return this->authority_outputs.try_pop(changes);
    }
};

/// @brief Builds a per-channel authority vector from the static AuthorityConfig
/// in the IR. Maps channel names to keys using node channel maps and returns
/// authorities aligned with write_keys.
inline std::vector<telem::Authority> build_authorities(
    const ir::AuthorityConfig &auth,
    const std::vector<types::ChannelKey> &write_keys,
    const std::vector<ir::Node> &nodes
) {
    if (!auth.default_authority.has_value() && auth.channels.empty()) return {};
    std::map<std::string, types::ChannelKey> name_to_key;
    for (const auto &n: nodes) {
        for (const auto &[key, name]: n.channels.read) name_to_key[name] = key;
        for (const auto &[key, name]: n.channels.write) name_to_key[name] = key;
    }
    std::vector<telem::Authority> authorities(write_keys.size());
    for (size_t i = 0; i < write_keys.size(); i++)
        authorities[i] = auth.default_authority.has_value()
                            ? *auth.default_authority
                            : telem::AUTH_ABSOLUTE;
    for (const auto &[name, value]: auth.channels) {
        auto it = name_to_key.find(name);
        if (it == name_to_key.end()) continue;
        for (size_t i = 0; i < write_keys.size(); i++) {
            if (write_keys[i] == it->second) {
                authorities[i] = value;
                break;
            }
        }
    }
    return authorities;
}

inline std::pair<std::shared_ptr<Runtime>, xerrors::Error>
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
    auto authority_factory = std::make_shared<authority::Factory>(state);
    node::MultiFactory fact(
        std::vector<std::shared_ptr<node::Factory>>{
            wasm_factory,
            time_factory,
            stage_factory,
            io_factory,
            constant_factory,
            authority_factory,
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
    const auto loop_cfg = cfg.loop.apply_defaults(time_factory->base_interval);
    const auto tolerance = time::calculate_tolerance(
        loop_cfg.mode,
        time_factory->base_interval
    );
    auto sched = std::make_unique<scheduler::Scheduler>(
        cfg.mod,
        nodes,
        tolerance,
        error_handler
    );
    auto [loop, err] = loop::create(loop_cfg);
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
        xerrors::NIL
    };
}

}
