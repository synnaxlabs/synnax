// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/test/xtest.h"

#include "driver/pipeline/base.h"

class ThrowingPipeline final : public driver::pipeline::Base {
public:
    explicit ThrowingPipeline(const x::breaker::Config &config): Base(config) {}

    void run() override { throw std::runtime_error("test exception"); }
};

class StdExceptionPipeline final : public driver::pipeline::Base {
public:
    explicit StdExceptionPipeline(const x::breaker::Config &config): Base(config) {}

    void run() override { throw std::out_of_range("test std::exception"); }
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
