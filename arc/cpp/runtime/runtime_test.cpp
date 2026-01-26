// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/xtest/xtest.h"

#include "arc/cpp/runtime/runtime.h"

using namespace arc::runtime;

namespace {
/// @brief Creates a minimal Runtime for testing queue behavior.
std::shared_ptr<Runtime> create_test_runtime(
    size_t input_capacity,
    arc::runtime::errors::Handler error_handler
) {
    Config cfg{
        .mod = {},
        .breaker = breaker::Config{},
        .retrieve_channels = nullptr,
        .input_queue_capacity = input_capacity,
        .output_queue_capacity = 1,
        .loop = {},
    };

    return std::make_shared<Runtime>(
        cfg,
        nullptr,
        nullptr,
        nullptr,
        nullptr,
        nullptr,
        std::vector<arc::types::ChannelKey>{},
        std::vector<arc::types::ChannelKey>{},
        std::move(error_handler)
    );
}
}

/// @brief Test that write() calls error handler with QUEUE_FULL_INPUT when queue is
/// full.
TEST(RuntimeTest, WriteCallsErrorHandlerOnQueueFull) {
    std::vector<xerrors::Error> reported_errors;
    auto runtime = create_test_runtime(1, [&](const xerrors::Error &err) {
        reported_errors.push_back(err);
    });

    auto series1 = telem::Series(telem::FLOAT32_T, 1);
    series1.write(1.0f);
    auto series2 = telem::Series(telem::FLOAT32_T, 1);
    series2.write(2.0f);

    ASSERT_NIL(runtime->write(telem::Frame(1, std::move(series1))));
    ASSERT_OCCURRED_AS(
        runtime->write(telem::Frame(1, std::move(series2))),
        arc::runtime::errors::QUEUE_FULL_INPUT
    );

    ASSERT_EQ(reported_errors.size(), 1);
    ASSERT_MATCHES(reported_errors[0], arc::runtime::errors::QUEUE_FULL_INPUT);
}

/// @brief Test that write() returns error and calls handler for each failed write.
TEST(RuntimeTest, WriteReportsMultipleQueueFullErrors) {
    std::vector<xerrors::Error> reported_errors;
    const auto runtime = create_test_runtime(1, [&](const xerrors::Error &err) {
        reported_errors.push_back(err);
    });

    for (int i = 0; i < 5; i++) {
        auto series = telem::Series(telem::FLOAT32_T, 1);
        series.write(static_cast<float>(i));
        runtime->write(telem::Frame(1, std::move(series)));
    }
    ASSERT_EQ(reported_errors.size(), 4);
    for (const auto &err: reported_errors)
        ASSERT_MATCHES(err, arc::runtime::errors::QUEUE_FULL_INPUT);
}

/// @brief Test that write() succeeds when queue has capacity.
TEST(RuntimeTest, WriteSucceedsWithCapacity) {
    std::vector<xerrors::Error> reported_errors;
    const auto runtime = create_test_runtime(10, [&](const xerrors::Error &err) {
        reported_errors.push_back(err);
    });
    for (int i = 0; i < 5; i++) {
        auto series = telem::Series(telem::FLOAT32_T, 1);
        series.write(static_cast<float>(i));
        ASSERT_NIL(runtime->write(telem::Frame(1, std::move(series))));
    }
    ASSERT_TRUE(reported_errors.empty());
}
