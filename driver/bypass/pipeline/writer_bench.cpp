// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <chrono>
#include <memory>
#include <mutex>
#include <string>
#include <thread>
#include <vector>

#include "benchmark/benchmark.h"

#include "driver/bypass/bypass.h"
#include "driver/bypass/pipeline/writer.h"
#include "driver/control/state.h"

namespace {
class NullWriter final : public driver::pipeline::Writer {
public:
    x::errors::Error write(const x::telem::Frame &) override { return x::errors::NIL; }
    x::errors::Error close() override { return x::errors::NIL; }
};

class NullFactory final : public driver::pipeline::WriterFactory {
public:
    std::pair<std::unique_ptr<driver::pipeline::Writer>, x::errors::Error>
    open_writer(const synnax::framer::WriterConfig &) override {
        return {std::make_unique<NullWriter>(), x::errors::NIL};
    }
};

x::telem::Frame make_frame(const synnax::channel::Key base, const uint32_t channels) {
    x::telem::Frame frame(channels);
    for (uint32_t ch = 0; ch < channels; ch++)
        frame.emplace(base + ch, x::telem::Series(std::vector<float>{1.0f}));
    return frame;
}

x::control::Update<synnax::channel::Key>
transfer(const synnax::channel::Key key, const x::control::Subject &to) {
    return {
        .transfers = {{
            .from = std::nullopt,
            .to = x::control::State<synnax::channel::Key>{
                .subject = to,
                .resource = key,
                .authority = 1,
            },
        }},
    };
}

x::control::Subject writer_subject(const int thread) {
    return {.key = "w" + std::to_string(thread), .name = "w" + std::to_string(thread)};
}

synnax::channel::Key base_key(const int thread) {
    return static_cast<synnax::channel::Key>(1 + thread * 1000);
}

/// @brief the bus and control mirror that every writer thread of one run shares.
struct Rig {
    std::shared_ptr<driver::bypass::Bus> bus = std::make_shared<driver::bypass::Bus>();
    std::shared_ptr<driver::control::States>
        states = std::make_shared<driver::control::States>();
    std::shared_ptr<driver::bypass::Subscription> sub;
    std::atomic<bool> stopped{false};
    std::thread updater;
    std::thread drainer;

    Rig(const uint32_t channels,
        const int64_t update_hz,
        const bool subscribed,
        int threads) {
        std::vector<synnax::channel::Key> all;
        for (int t = 0; t < threads; t++)
            for (uint32_t c = 0; c < channels; c++) {
                all.push_back(base_key(t) + c);
                this->states->apply(transfer(base_key(t) + c, writer_subject(t)));
            }
        this->bus->register_channels(all);
        if (subscribed) {
            this->sub = this->bus->subscribe(all);
            this->drainer = std::thread([this] {
                x::telem::Frame f;
                while (!this->stopped.load(std::memory_order_relaxed))
                    if (!this->sub->try_pop(f)) std::this_thread::yield();
            });
        }
        if (update_hz > 0)
            this->updater = std::thread([this, update_hz] {
                const x::control::Subject a{.key = "a", .name = "a"};
                const x::control::Subject b{.key = "b", .name = "b"};
                const auto period = std::chrono::microseconds(1000000 / update_hz);
                bool flip = false;
                while (!this->stopped.load(std::memory_order_relaxed)) {
                    this->states->apply(transfer(999999, flip ? a : b));
                    flip = !flip;
                    std::this_thread::sleep_for(period);
                }
            });
    }

    Rig(const Rig &) = delete;
    Rig &operator=(const Rig &) = delete;

    ~Rig() {
        this->stopped = true;
        if (this->updater.joinable()) this->updater.join();
        if (this->drainer.joinable()) this->drainer.join();
    }
};

/// @brief builds one Rig per benchmark run, for whichever thread arrives first.
class WriterBench : public benchmark::Fixture {
    std::mutex mu;
    std::shared_ptr<Rig> rig;

public:
    void SetUp(benchmark::State &state) override {
        std::lock_guard lock(this->mu);
        if (this->rig != nullptr) return;
        this->rig = std::make_shared<Rig>(
            static_cast<uint32_t>(state.range(0)),
            state.range(1),
            state.range(2) != 0,
            state.threads()
        );
    }

    void TearDown(benchmark::State &) override {
        std::lock_guard lock(this->mu);
        this->rig = nullptr;
    }

    std::shared_ptr<Rig> current() {
        std::lock_guard lock(this->mu);
        return this->rig;
    }
};
}

/// @brief the cost of a local bus write when the writer holds every channel. Args:
/// channels per frame, control updates per second, and whether a subscriber reads.
BENCHMARK_DEFINE_F(WriterBench, Write)(benchmark::State &state) {
    const auto rig = this->current();
    const auto t = state.thread_index();
    const synnax::framer::WriterConfig cfg{.subject = writer_subject(t)};
    driver::bypass::pipeline::Writer writer(
        std::make_unique<driver::bypass::pipeline::Forwarder>(
            std::make_shared<NullFactory>(),
            cfg
        ),
        rig->bus,
        rig->states,
        cfg
    );
    const auto frame = make_frame(base_key(t), static_cast<uint32_t>(state.range(0)));
    for (auto _: state) {
        auto err = writer.write(frame);
        benchmark::DoNotOptimize(err);
    }
    state.SetItemsProcessed(state.iterations());
}

BENCHMARK_REGISTER_F(WriterBench, Write)
    ->ArgsProduct({{10, 100}, {0, 100}, {0, 1}})
    ->Threads(1)
    ->Threads(4)
    ->Threads(16)
    ->UseRealTime();
