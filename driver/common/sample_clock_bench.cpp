// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <set>

#include "benchmark/benchmark.h"

#include "x/cpp/bench/bench.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/series.h"

#include "driver/common/sample_clock.h"

/// @brief stamps the index series of one acquisition frame. Arg 0 is the number of
/// index channels. Arg 1 is the number of samples in the frame.
static void BM_GenerateIndexData(benchmark::State &state) {
    const auto indexes = static_cast<synnax::channel::Key>(state.range(0));
    const auto samples = static_cast<size_t>(state.range(1));
    x::telem::Frame frame(indexes);
    std::set<synnax::channel::Key> index_keys;
    for (synnax::channel::Key key = 1; key <= indexes; key++) {
        frame.emplace(key, x::telem::Series(x::telem::TIMESTAMP_T, samples));
        index_keys.insert(key);
    }
    const x::telem::TimeStamp start(0);
    const auto end = start + x::telem::MILLISECOND * 5;
    x::bench::run_with_alloc_tracking(state, [&] {
        driver::common::generate_index_data(frame, index_keys, start, end, samples, 0);
        benchmark::DoNotOptimize(frame);
    });
}

BENCHMARK(BM_GenerateIndexData)->Args({1, 125})->Args({2, 125});
