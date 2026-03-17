// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <thread>
#include <vector>

#include "benchmark/benchmark.h"
#include "driver/bypass/authority.h"
#include "driver/bypass/bypass.h"
#include "driver/bypass/streamer.h"
#include "driver/bypass/writer.h"
#include "driver/pipeline/mock/pipeline.h"

namespace {
x::telem::Frame make_frame(const uint32_t num_channels, const size_t samples) {
    std::vector<float> data(samples, 1.0f);
    x::telem::Frame frame(num_channels);
    for (uint32_t ch = 1; ch <= num_channels; ch++)
        frame.emplace(ch, x::telem::Series(data));
    return frame;
}

struct FrameWorkload {
    const char *name;
    uint32_t channels;
    size_t samples;

    [[nodiscard]] size_t total_bytes() const {
        return channels * samples * sizeof(float);
    }
};

const FrameWorkload WORKLOADS[] = {
    {"small_cmd", 10, 1},
    {"medium", 10, 1000},
    {"large_acq", 30, 4000},
};
constexpr int NUM_WORKLOADS = 3;
}

// --- Publish benchmarks ---

static void BM_BusPublish_NoSubs(benchmark::State &state) {
    const auto &w = WORKLOADS[state.range(0)];
    driver::bypass::Bus bus;
    auto frame = make_frame(w.channels, w.samples);
    for (auto _: state)
        bus.publish(frame);
    state.SetLabel(w.name);
}

BENCHMARK(BM_BusPublish_NoSubs)->DenseRange(0, NUM_WORKLOADS - 1);

static void BM_BusPublish_1Sub(benchmark::State &state) {
    const auto &w = WORKLOADS[state.range(0)];
    driver::bypass::Bus bus;
    std::vector<synnax::channel::Key> keys(w.channels);
    for (uint32_t i = 0; i < w.channels; i++)
        keys[i] = i + 1;
    auto sub = bus.subscribe(keys);
    auto frame = make_frame(w.channels, w.samples);
    for (auto _: state) {
        bus.publish(frame);
        x::telem::Frame drain;
        while (sub->try_pop(drain)) {}
    }
    state.SetBytesProcessed(
        static_cast<int64_t>(state.iterations()) * static_cast<int64_t>(w.total_bytes())
    );
    state.SetLabel(w.name);
}

BENCHMARK(BM_BusPublish_1Sub)->DenseRange(0, NUM_WORKLOADS - 1);

static void BM_BusPublish_NSubs(benchmark::State &state) {
    const auto num_subs = state.range(0);
    const auto &w = WORKLOADS[2];
    driver::bypass::Bus bus;
    std::vector<synnax::channel::Key> keys(w.channels);
    for (uint32_t i = 0; i < w.channels; i++)
        keys[i] = i + 1;
    std::vector<std::shared_ptr<driver::bypass::Subscription>> subs;
    for (int i = 0; i < num_subs; i++)
        subs.push_back(bus.subscribe(keys));
    auto frame = make_frame(w.channels, w.samples);
    for (auto _: state) {
        bus.publish(frame);
        for (auto &sub: subs) {
            x::telem::Frame drain;
            while (sub->try_pop(drain)) {}
        }
    }
    state.SetLabel(w.name);
}

BENCHMARK(BM_BusPublish_NSubs)->Arg(1)->Arg(2)->Arg(5);

// --- Subscription benchmarks ---

static void BM_SubscriptionPushPop(benchmark::State &state) {
    const auto &w = WORKLOADS[state.range(0)];
    std::vector<synnax::channel::Key> keys(w.channels);
    for (uint32_t i = 0; i < w.channels; i++)
        keys[i] = i + 1;
    driver::bypass::Subscription sub(keys);
    for (auto _: state) {
        sub.push(make_frame(w.channels, w.samples));
        x::telem::Frame out;
        sub.try_pop(out);
        benchmark::DoNotOptimize(out);
    }
    state.SetBytesProcessed(
        static_cast<int64_t>(state.iterations()) * static_cast<int64_t>(w.total_bytes())
    );
    state.SetLabel(w.name);
}

BENCHMARK(BM_SubscriptionPushPop)->DenseRange(0, NUM_WORKLOADS - 1);

static void BM_SubscriptionCrossThread(benchmark::State &state) {
    const auto &w = WORKLOADS[2];
    std::vector<synnax::channel::Key> keys(w.channels);
    for (uint32_t i = 0; i < w.channels; i++)
        keys[i] = i + 1;
    driver::bypass::Subscription sub(keys);
    std::atomic<bool> done{false};
    std::thread producer([&] {
        while (!done.load(std::memory_order_relaxed))
            sub.push(make_frame(w.channels, w.samples));
    });
    for (auto _: state) {
        x::telem::Frame out;
        sub.pop(out);
        benchmark::DoNotOptimize(out);
    }
    done.store(true);
    sub.close();
    producer.join();
    state.SetLabel(w.name);
}

BENCHMARK(BM_SubscriptionCrossThread);

// --- Authority filter benchmarks ---

static void BM_AuthorityFilter_AllPass(benchmark::State &state) {
    const auto &w = WORKLOADS[state.range(0)];
    driver::bypass::AuthorityMirror mirror;
    x::control::Subject subject{.name = "bench", .key = "bench-key"};
    auto frame = make_frame(w.channels, w.samples);
    for (auto _: state) {
        auto filtered = mirror.filter(frame, subject);
        benchmark::DoNotOptimize(filtered);
    }
    state.SetBytesProcessed(
        static_cast<int64_t>(state.iterations()) * static_cast<int64_t>(w.total_bytes())
    );
    state.SetLabel(w.name);
}

BENCHMARK(BM_AuthorityFilter_AllPass)->DenseRange(0, NUM_WORKLOADS - 1);

static void BM_AuthorityFilter_HalfPass(benchmark::State &state) {
    const auto &w = WORKLOADS[state.range(0)];
    driver::bypass::AuthorityMirror mirror;
    x::control::Subject subject{.name = "bench", .key = "bench-key"};
    x::control::Subject other{.name = "other", .key = "other-key"};
    for (uint32_t ch = 1; ch <= w.channels; ch++) {
        if (ch % 2 == 0) {
            x::control::Update update;
            update.transfers.push_back(
                x::control::Transfer{
                    .from = std::nullopt,
                    .to = x::control::State{.resource = ch, .subject = other},
                }
            );
            mirror.apply(update);
        }
    }
    auto frame = make_frame(w.channels, w.samples);
    for (auto _: state) {
        auto filtered = mirror.filter(frame, subject);
        benchmark::DoNotOptimize(filtered);
    }
    state.SetLabel(w.name);
}

BENCHMARK(BM_AuthorityFilter_HalfPass)->DenseRange(0, NUM_WORKLOADS - 1);

static void BM_AuthorityFilter_NonePass(benchmark::State &state) {
    const auto &w = WORKLOADS[state.range(0)];
    driver::bypass::AuthorityMirror mirror;
    x::control::Subject subject{.name = "bench", .key = "bench-key"};
    x::control::Subject other{.name = "other", .key = "other-key"};
    for (uint32_t ch = 1; ch <= w.channels; ch++) {
        x::control::Update update;
        update.transfers.push_back(
            x::control::Transfer{
                .from = std::nullopt,
                .to = x::control::State{.resource = ch, .subject = other},
            }
        );
        mirror.apply(update);
    }
    auto frame = make_frame(w.channels, w.samples);
    for (auto _: state) {
        auto filtered = mirror.filter(frame, subject);
        benchmark::DoNotOptimize(filtered);
    }
    state.SetLabel(w.name);
}

BENCHMARK(BM_AuthorityFilter_NonePass)->DenseRange(0, NUM_WORKLOADS - 1);

// --- Move-based authority filter benchmarks ---

static void BM_AuthorityFilterMove_AllPass(benchmark::State &state) {
    const auto &w = WORKLOADS[state.range(0)];
    driver::bypass::AuthorityMirror mirror;
    x::control::Subject subject{.name = "bench", .key = "bench-key"};
    for (auto _: state) {
        state.PauseTiming();
        auto frame = make_frame(w.channels, w.samples);
        state.ResumeTiming();
        auto filtered = mirror.filter(std::move(frame), subject);
        benchmark::DoNotOptimize(filtered);
    }
    state.SetBytesProcessed(
        static_cast<int64_t>(state.iterations()) * static_cast<int64_t>(w.total_bytes())
    );
    state.SetLabel(w.name);
}

BENCHMARK(BM_AuthorityFilterMove_AllPass)->DenseRange(0, NUM_WORKLOADS - 1);

static void BM_AuthorityFilterMove_HalfPass(benchmark::State &state) {
    const auto &w = WORKLOADS[state.range(0)];
    driver::bypass::AuthorityMirror mirror;
    x::control::Subject subject{.name = "bench", .key = "bench-key"};
    x::control::Subject other{.name = "other", .key = "other-key"};
    for (uint32_t ch = 1; ch <= w.channels; ch++) {
        if (ch % 2 == 0) {
            x::control::Update update;
            update.transfers.push_back(
                x::control::Transfer{
                    .from = std::nullopt,
                    .to = x::control::State{.resource = ch, .subject = other},
                }
            );
            mirror.apply(update);
        }
    }
    for (auto _: state) {
        state.PauseTiming();
        auto frame = make_frame(w.channels, w.samples);
        state.ResumeTiming();
        auto filtered = mirror.filter(std::move(frame), subject);
        benchmark::DoNotOptimize(filtered);
    }
    state.SetLabel(w.name);
}

BENCHMARK(BM_AuthorityFilterMove_HalfPass)->DenseRange(0, NUM_WORKLOADS - 1);

static void BM_AuthorityFilterMove_NonePass(benchmark::State &state) {
    const auto &w = WORKLOADS[state.range(0)];
    driver::bypass::AuthorityMirror mirror;
    x::control::Subject subject{.name = "bench", .key = "bench-key"};
    x::control::Subject other{.name = "other", .key = "other-key"};
    for (uint32_t ch = 1; ch <= w.channels; ch++) {
        x::control::Update update;
        update.transfers.push_back(
            x::control::Transfer{
                .from = std::nullopt,
                .to = x::control::State{.resource = ch, .subject = other},
            }
        );
        mirror.apply(update);
    }
    for (auto _: state) {
        state.PauseTiming();
        auto frame = make_frame(w.channels, w.samples);
        state.ResumeTiming();
        auto filtered = mirror.filter(std::move(frame), subject);
        benchmark::DoNotOptimize(filtered);
    }
    state.SetLabel(w.name);
}

BENCHMARK(BM_AuthorityFilterMove_NonePass)->DenseRange(0, NUM_WORKLOADS - 1);

// --- End-to-end benchmarks ---

static void BM_EndToEnd(benchmark::State &state) {
    const auto &w = WORKLOADS[state.range(0)];
    driver::bypass::Bus bus;
    driver::bypass::AuthorityMirror mirror;
    x::control::Subject subject{.name = "bench", .key = "bench-key"};

    std::vector<synnax::channel::Key> keys(w.channels);
    for (uint32_t i = 0; i < w.channels; i++)
        keys[i] = i + 1;

    auto mock_writes = std::make_shared<std::vector<x::telem::Frame>>();
    auto mock_writer_factory = std::make_shared<driver::pipeline::mock::WriterFactory>(
        mock_writes
    );
    auto [server_writer, open_err] = mock_writer_factory->open_writer(
        synnax::framer::WriterConfig{.channels = keys}
    );

    driver::bypass::Writer writer(std::move(server_writer), bus, mirror, subject, keys);

    auto sub = bus.subscribe(keys);

    auto frame = make_frame(w.channels, w.samples);
    for (auto _: state) {
        static_cast<void>(writer.write(frame));
        x::telem::Frame local;
        sub->try_pop(local);
        auto filtered = mirror.filter(std::move(local), subject);
        benchmark::DoNotOptimize(filtered);
    }

    mock_writes->clear();
    state.SetBytesProcessed(
        static_cast<int64_t>(state.iterations()) * static_cast<int64_t>(w.total_bytes())
    );
    state.SetLabel(w.name);
}

BENCHMARK(BM_EndToEnd)->DenseRange(0, NUM_WORKLOADS - 1);
