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
#include <thread>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/lib/lazy.h"
#include "x/cpp/testutil/testutil.h"

namespace x::lib {
const auto LOAD_FAILED = x::errors::Error("test.load_failed", "library is missing");

/// @brief it should load again after a failed load, and keep the first value that
/// loads.
TEST(LazyTest, RetriesUntilLoadSucceeds) {
    int loads = 0;
    Lazy<int> lazy([&loads]() -> std::pair<std::shared_ptr<int>, x::errors::Error> {
        if (loads++ == 0) return {nullptr, LOAD_FAILED};
        return {std::make_shared<int>(loads), x::errors::NIL};
    });
    ASSERT_OCCURRED_AS_P(lazy.get(), LOAD_FAILED);
    const auto value = ASSERT_NIL_P(lazy.get());
    EXPECT_EQ(*value, 2);
    EXPECT_EQ(ASSERT_NIL_P(lazy.get()), value);
    EXPECT_EQ(loads, 2);
}

/// @brief it should load once when many callers ask for the value at the same time.
TEST(LazyTest, ConcurrentCallersShareOneLoad) {
    std::atomic<int> loads = 0;
    Lazy<int> lazy([&loads]() -> std::pair<std::shared_ptr<int>, x::errors::Error> {
        loads++;
        // A slow load gives every caller time to arrive while it runs.
        std::this_thread::sleep_for(std::chrono::milliseconds(20));
        return {std::make_shared<int>(1), x::errors::NIL};
    });
    std::vector<std::shared_ptr<int>> values(8);
    std::vector<std::thread> threads;
    threads.reserve(values.size());
    for (auto &value: values)
        threads.emplace_back([&lazy, &value] { value = lazy.get().first; });
    for (auto &t: threads)
        t.join();
    EXPECT_EQ(loads.load(), 1);
    for (const auto &value: values)
        EXPECT_EQ(value, values[0]);
}
}
