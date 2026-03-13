// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <vector>

#include "benchmark/benchmark.h"

#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

namespace {
x::telem::Series make_float_series(const size_t num_samples) {
    std::vector<float> data(num_samples, 1.0f);
    return x::telem::Series(data);
}

x::telem::Series make_uint8_series(const size_t num_samples) {
    std::vector<uint8_t> data(num_samples, 1);
    return x::telem::Series(data);
}

x::telem::Series make_float64_series(const size_t num_samples) {
    std::vector<double> data(num_samples, 1.0);
    return x::telem::Series(data);
}

x::telem::Frame make_frame(
    const uint32_t num_channels,
    const size_t samples_per_channel,
    const x::telem::DataType &dt
) {
    x::telem::Frame frame(num_channels);
    for (uint32_t ch = 1; ch <= num_channels; ch++) {
        if (dt == x::telem::FLOAT32_T)
            frame.emplace(ch, make_float_series(samples_per_channel));
        else if (dt == x::telem::UINT8_T)
            frame.emplace(ch, make_uint8_series(samples_per_channel));
        else if (dt == x::telem::FLOAT64_T)
            frame.emplace(ch, make_float64_series(samples_per_channel));
    }
    return frame;
}

struct FrameWorkload {
    const char *name;
    uint32_t channels;
    size_t samples;
    x::telem::DataType dt;

    [[nodiscard]] size_t total_bytes() const {
        return channels * samples * dt.density();
    }
};

const FrameWorkload WORKLOADS[] = {
    {"single_f64", 1, 1, x::telem::FLOAT64_T},
    {"small_cmd", 10, 1, x::telem::UINT8_T},
    {"medium", 10, 1000, x::telem::FLOAT32_T},
    {"large_acq", 30, 4000, x::telem::FLOAT32_T},
};

}

static void BM_SeriesDeepCopy(benchmark::State &state) {
    const auto num_bytes = state.range(0);
    const auto num_samples = static_cast<size_t>(num_bytes) / sizeof(float);
    auto series = make_float_series(num_samples);
    for (auto _: state) {
        auto copy = series.deep_copy();
        benchmark::DoNotOptimize(copy);
    }
    state.SetBytesProcessed(state.iterations() * num_bytes);
}

BENCHMARK(BM_SeriesDeepCopy)->ArgNames({"bytes"})->Arg(32)->Arg(16384)->Arg(65536)->Arg(
    491520
);

static void BM_SeriesMove(benchmark::State &state) {
    const auto num_bytes = state.range(0);
    const auto num_samples = static_cast<size_t>(num_bytes) / sizeof(float);
    for (auto _: state) {
        state.PauseTiming();
        auto series = make_float_series(num_samples);
        state.ResumeTiming();
        auto moved = x::telem::Series(std::move(series));
        benchmark::DoNotOptimize(moved);
    }
    state.SetBytesProcessed(state.iterations() * num_bytes);
}

BENCHMARK(BM_SeriesMove)->ArgNames({"bytes"})->Arg(32)->Arg(16384)->Arg(65536)->Arg(
    491520
);

static void BM_FrameDeepCopy(benchmark::State &state) {
    const auto &w = WORKLOADS[state.range(0)];
    auto frame = make_frame(w.channels, w.samples, w.dt);
    for (auto _: state) {
        auto copy = frame.deep_copy();
        benchmark::DoNotOptimize(copy);
    }
    state.SetBytesProcessed(
        static_cast<int64_t>(state.iterations()) *
        static_cast<int64_t>(w.total_bytes())
    );
    state.SetLabel(w.name);
}

BENCHMARK(BM_FrameDeepCopy)->DenseRange(0, 3);

static void BM_FrameMove(benchmark::State &state) {
    const auto &w = WORKLOADS[state.range(0)];
    for (auto _: state) {
        state.PauseTiming();
        auto frame = make_frame(w.channels, w.samples, w.dt);
        state.ResumeTiming();
        auto moved = x::telem::Frame(std::move(frame));
        benchmark::DoNotOptimize(moved);
    }
    state.SetLabel(w.name);
}

BENCHMARK(BM_FrameMove)->DenseRange(0, 3);

static void BM_FrameConstruct(benchmark::State &state) {
    const auto &w = WORKLOADS[state.range(0)];
    std::vector<std::vector<float>> data(w.channels);
    for (auto &d: data) d.assign(w.samples, 1.0f);
    for (auto _: state) {
        x::telem::Frame frame(w.channels);
        for (uint32_t ch = 0; ch < w.channels; ch++)
            frame.emplace(ch + 1, x::telem::Series(data[ch]));
        benchmark::DoNotOptimize(frame);
    }
    state.SetBytesProcessed(
        static_cast<int64_t>(state.iterations()) *
        static_cast<int64_t>(w.total_bytes())
    );
    state.SetLabel(w.name);
}

BENCHMARK(BM_FrameConstruct)->DenseRange(0, 3);

static void BM_FrameIterate(benchmark::State &state) {
    const auto &w = WORKLOADS[state.range(0)];
    auto frame = make_frame(w.channels, w.samples, w.dt);
    for (auto _: state) {
        uint32_t key_sum = 0;
        for (auto [key, series]: frame) key_sum += key;
        benchmark::DoNotOptimize(key_sum);
    }
    state.SetLabel(w.name);
}

BENCHMARK(BM_FrameIterate)->DenseRange(0, 3);
