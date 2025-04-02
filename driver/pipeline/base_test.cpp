// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// external
#include "gtest/gtest.h"

/// module
#include "x/cpp/xtest/xtest.h"

/// internal
#include "driver/pipeline/base.h"

class ThrowingPipeline final : public pipeline::Base {
public:
    explicit ThrowingPipeline(const breaker::Config& config) : Base(config) {}

    void run() override {
        throw std::runtime_error("test exception");
    }
};

class StdExceptionPipeline final : public pipeline::Base {
public:
    explicit StdExceptionPipeline(const breaker::Config& config) : Base(config) {}

    void run() override {
        throw std::out_of_range("test std::exception");
    }
};

TEST(BasePipeline, testUnknownExceptionHandling) {
    auto pipeline = ThrowingPipeline(breaker::Config{});
    ASSERT_TRUE(pipeline.start());
    std::this_thread::sleep_for(std::chrono::milliseconds(5));
    ASSERT_TRUE(pipeline.stop());
}

TEST(BasePipeline, testStdExceptionHandling) {
    auto pipeline = StdExceptionPipeline(breaker::Config{});
    ASSERT_TRUE(pipeline.start());
    std::this_thread::sleep_for(std::chrono::milliseconds(5));
    ASSERT_TRUE(pipeline.stop());
} 