// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <stdexcept>
#include <vector>

#include "benchmark/benchmark.h"

#include "client/cpp/framer/framer.h"
#include "x/cpp/bench/bench.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/series.h"

namespace {
/// @brief a codec of n float32 channels, and a frame that holds a series of the given
/// number of samples for each.
struct Shape {
    synnax::framer::Codec codec;
    x::telem::Frame frame;

    Shape(const std::size_t n, const std::size_t samples):
        codec(keys(n), std::vector(n, x::telem::FLOAT32_T)), frame(n) {
        const std::vector<float> values(samples, 1.0f);
        for (const auto key: keys(n))
            frame.emplace(key, x::telem::Series(values));
    }

    static std::vector<synnax::channel::Key> keys(const std::size_t n) {
        std::vector<synnax::channel::Key> keys(n);
        for (std::size_t i = 0; i < n; i++)
            keys[i] = static_cast<synnax::channel::Key>(i + 1);
        return keys;
    }

    /// @brief returns the frame encoded after the first frame of its state, which the
    /// codec has decoded, so a benchmark measures the frame a stream sends most.
    std::vector<uint8_t> steady() {
        std::vector<uint8_t> wire;
        if (const auto err = this->codec.encode(this->frame, wire))
            throw std::runtime_error(err.message());
        if (const auto [_, err] = this->codec.decode(wire); err)
            throw std::runtime_error(err.message());
        if (const auto err = this->codec.encode(this->frame, wire))
            throw std::runtime_error(err.message());
        return wire;
    }
};
}

/// @brief encodes a steady frame. Arg 0 is the number of channels. Arg 1 is the samples
/// in each series.
static void BM_Encode(benchmark::State &state) {
    Shape shape(state.range(0), state.range(1));
    auto wire = shape.steady();
    x::bench::run_with_alloc_tracking(state, [&] {
        auto err = shape.codec.encode(shape.frame, wire);
        benchmark::DoNotOptimize(err);
    });
}

/// @brief decodes a steady frame. Arg 0 is the number of channels. Arg 1 is the samples
/// in each series.
static void BM_Decode(benchmark::State &state) {
    Shape shape(state.range(0), state.range(1));
    const auto wire = shape.steady();
    x::bench::run_with_alloc_tracking(state, [&] {
        auto decoded = shape.codec.decode(wire);
        benchmark::DoNotOptimize(decoded);
    });
}

BENCHMARK(BM_Encode)->Args({1, 3})->Args({100, 10})->Args({1000, 1});
BENCHMARK(BM_Decode)->Args({1, 3})->Args({100, 10})->Args({1000, 1});

/// @brief decodes a steady frame that holds many series for one channel, such as one
/// per stored domain. Arg 0 is the number of series. Arg 1 is the samples in each
/// series.
static void BM_DecodeOneChannel(benchmark::State &state) {
    const synnax::channel::Key key = 1;
    const auto series = static_cast<size_t>(state.range(0));
    const std::vector<float> samples(static_cast<size_t>(state.range(1)), 1.0f);
    synnax::framer::Codec codec({key}, {x::telem::FLOAT32_T});
    x::telem::Frame frame(series);
    for (size_t i = 0; i < series; i++)
        frame.emplace(key, x::telem::Series(samples));
    std::vector<uint8_t> wire;
    for (int i = 0; i < 2; i++) {
        if (const auto err = codec.encode(frame, wire)) {
            state.SkipWithError(err.message());
            return;
        }
        if (i > 0) break;
        if (const auto [_, err] = codec.decode(wire); err) {
            state.SkipWithError(err.message());
            return;
        }
    }
    x::bench::run_with_alloc_tracking(state, [&] {
        auto decoded = codec.decode(wire);
        benchmark::DoNotOptimize(decoded);
    });
}

BENCHMARK(BM_DecodeOneChannel)->Args({100, 10});
