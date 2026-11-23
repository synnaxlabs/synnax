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

#include "client/cpp/framer/framer.h"
#include "x/cpp/bench/bench.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/series.h"

/// @brief decodes a frame that holds many series for one channel, such as one per
/// stored domain. Arg 0 is the number of series. Arg 1 is the samples in each series.
static void BM_DecodeOneChannel(benchmark::State &state) {
    const synnax::channel::Key key = 1;
    const auto series = static_cast<size_t>(state.range(0));
    const std::vector<float> samples(static_cast<size_t>(state.range(1)), 1.0f);
    synnax::framer::Codec codec({key}, {x::telem::FLOAT32_T});
    x::telem::Frame frame(series);
    for (size_t i = 0; i < series; i++)
        frame.emplace(key, x::telem::Series(samples));
    std::vector<uint8_t> wire;
    if (const auto err = codec.encode(frame, wire)) {
        state.SkipWithError(err.message());
        return;
    }
    x::bench::run_with_alloc_tracking(state, [&] {
        auto decoded = codec.decode(wire);
        benchmark::DoNotOptimize(decoded);
    });
}

BENCHMARK(BM_DecodeOneChannel)->Args({100, 10});
