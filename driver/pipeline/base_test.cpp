// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/pipeline/base.h"

namespace driver::pipeline {
class ThrowingPipeline final : public Base {
public:
    explicit ThrowingPipeline(const x::breaker::Config &config): Base(config) {}

    void run() override { throw std::runtime_error("test exception"); }
};

class StdExceptionPipeline final : public Base {
public:
    explicit StdExceptionPipeline(const x::breaker::Config &config): Base(config) {}

    void run() override { throw std::out_of_range("test std::exception"); }
};

class SelfStoppingPipeline final : public Base {
    std::atomic<int> run_count{0};

public:
    explicit SelfStoppingPipeline(const x::breaker::Config &config): Base(config) {}

    int runs() const { return this->run_count.load(); }

    void run() override {
        this->run_count.fetch_add(1);
        this->stop();
    }
};

/// @brief it should catch and handle unknown exceptions in run().
TEST(BasePipeline, testUnknownExceptionHandling) {
    auto pipeline = ThrowingPipeline(x::breaker::Config{});
    ASSERT_TRUE(pipeline.start());
    std::this_thread::sleep_for(std::chrono::milliseconds(5));
    ASSERT_TRUE(pipeline.stop());
}

/// @brief it should catch and handle std::exception in run().
TEST(BasePipeline, testStdExceptionHandling) {
    auto pipeline = StdExceptionPipeline(x::breaker::Config{});
    ASSERT_TRUE(pipeline.start());
    std::this_thread::sleep_for(std::chrono::milliseconds(5));
    ASSERT_TRUE(pipeline.stop());
}

/// @brief calling start() after run() self-stops must not abort or deadlock.
/// Before the fix, the old std::thread was still joinable when start() assigned
/// a new one, triggering std::terminate(). The fix joins the old thread before
/// restarting the breaker to also prevent the old thread from seeing the new
/// breaker state and re-entering its loop.
TEST(BasePipeline, testStartAfterSelfStop) {
    auto pipeline = SelfStoppingPipeline(x::breaker::Config{});
    ASSERT_TRUE(pipeline.start());
    ASSERT_EVENTUALLY_EQ(pipeline.runs(), 1);
    ASSERT_TRUE(pipeline.start());
    ASSERT_EVENTUALLY_EQ(pipeline.runs(), 2);
    ASSERT_TRUE(pipeline.start());
    ASSERT_EVENTUALLY_EQ(pipeline.runs(), 3);
    pipeline.stop();
}
}
