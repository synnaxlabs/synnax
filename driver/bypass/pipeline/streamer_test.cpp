// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <functional>
#include <random>
#include <sstream>
#include <thread>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/bypass/pipeline/streamer.h"
#include "driver/pipeline/testutil/pipeline.h"

namespace driver::bypass::pipeline {
const x::control::Subject TEST_SUBJECT{"test_writer", "tw-1"};

TEST(StreamerTest, DeliversLocalBusFrames) {
    auto bus = std::make_shared<Bus>();
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    auto mock_factory = ::driver::pipeline::testutil::simple_streamer_factory(
        {1},
        reads
    );
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));
    x::telem::Frame local_frame;
    local_frame.emplace(1, x::telem::Series(static_cast<float>(42.0)));
    bus->publish(local_frame);
    auto frame = ASSERT_NIL_P(streamer->read());
    ASSERT_EQ(frame.size(), 1);
    streamer->close_send();
    ASSERT_NIL(streamer->close());
}

TEST(StreamerTest, DeliversCoreFrames) {
    auto bus = std::make_shared<Bus>();
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    x::telem::Frame server_frame;
    server_frame.emplace(1, x::telem::Series(static_cast<float>(99.0)));
    reads->push_back(std::move(server_frame));
    auto mock_factory = ::driver::pipeline::testutil::simple_streamer_factory(
        {1},
        reads
    );
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));
    auto frame = ASSERT_NIL_P(streamer->read());
    ASSERT_EQ(frame.size(), 1);
    streamer->close_send();
    ASSERT_NIL(streamer->close());
}

TEST(StreamerTest, DeliversLocalFramesRegardlessOfAuthority) {
    auto bus = std::make_shared<Bus>();
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    auto mock_factory = ::driver::pipeline::testutil::simple_streamer_factory(
        {1},
        reads
    );
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));
    x::telem::Frame local_frame;
    local_frame.emplace(1, x::telem::Series(static_cast<float>(42.0)));
    bus->publish(local_frame);
    auto frame = ASSERT_NIL_P(streamer->read());
    ASSERT_EQ(frame.size(), 1);
    streamer->close_send();
    ASSERT_NIL(streamer->close());
}

TEST(StreamerTest, CloseSendDelegatesToCore) {
    auto bus = std::make_shared<Bus>();
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    auto mock_factory = ::driver::pipeline::testutil::simple_streamer_factory(
        {1},
        reads
    );
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));
    streamer->close_send();
    ASSERT_NIL(streamer->close());
}

TEST(StreamerTest, PropagatesOpenError) {
    auto bus = std::make_shared<Bus>();
    auto mock_factory = std::make_shared<::driver::pipeline::testutil::StreamerFactory>(
        std::vector<x::errors::Error>{x::errors::VALIDATION},
        std::make_shared<std::vector<::driver::pipeline::testutil::StreamerConfig>>()
    );
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT);
    ASSERT_OCCURRED_AS_P(
        factory.open_streamer({.channels = {1}}),
        x::errors::VALIDATION
    );
}

TEST(StreamerTest, InjectsExcludeGroupsFromSubjectGroup) {
    auto bus = std::make_shared<Bus>();
    const x::control::Subject grouped_subject{"grouped_writer", "gw-1", 42};
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    x::telem::Frame server_frame;
    server_frame.emplace(1, x::telem::Series(static_cast<float>(99.0)));
    reads->push_back(std::move(server_frame));
    auto mock_factory = std::make_shared<::driver::pipeline::testutil::StreamerFactory>(
        std::vector<x::errors::Error>{},
        std::make_shared<std::vector<::driver::pipeline::testutil::StreamerConfig>>(
            std::vector{
                ::driver::pipeline::testutil::StreamerConfig{reads, {}, x::errors::NIL}
            }
        )
    );
    StreamerFactory factory(mock_factory, bus, grouped_subject);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));
    ASSERT_EQ(mock_factory->config.exclude_groups.size(), 1);
    ASSERT_EQ(mock_factory->config.exclude_groups[0], 42);
    streamer->close_send();
    ASSERT_NIL(streamer->close());
}

TEST(StreamerTest, DoesNotInjectExcludeGroupsWhenGroupIsZero) {
    auto bus = std::make_shared<Bus>();
    const x::control::Subject no_group_subject{"no_group", "ng-1", 0};
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    x::telem::Frame server_frame;
    server_frame.emplace(1, x::telem::Series(static_cast<float>(99.0)));
    reads->push_back(std::move(server_frame));
    auto mock_factory = std::make_shared<::driver::pipeline::testutil::StreamerFactory>(
        std::vector<x::errors::Error>{},
        std::make_shared<std::vector<::driver::pipeline::testutil::StreamerConfig>>(
            std::vector{
                ::driver::pipeline::testutil::StreamerConfig{reads, {}, x::errors::NIL}
            }
        )
    );
    StreamerFactory factory(mock_factory, bus, no_group_subject);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));
    ASSERT_TRUE(mock_factory->config.exclude_groups.empty());
    streamer->close_send();
    ASSERT_NIL(streamer->close());
}

TEST(StreamerTest, CoreErrorDuringActiveReading) {
    auto bus = std::make_shared<Bus>();
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    x::telem::Frame valid_frame;
    valid_frame.emplace(1, x::telem::Series(static_cast<float>(99.0)));
    reads->push_back(std::move(valid_frame));
    reads->emplace_back();
    auto read_errors = std::make_shared<std::vector<x::errors::Error>>();
    read_errors->push_back(x::errors::NIL);
    read_errors->push_back(freighter::UNREACHABLE);
    auto mock_factory = std::make_shared<::driver::pipeline::testutil::StreamerFactory>(
        std::vector<x::errors::Error>{},
        std::make_shared<std::vector<::driver::pipeline::testutil::StreamerConfig>>(
            std::vector{::driver::pipeline::testutil::StreamerConfig{
                reads,
                read_errors,
                x::errors::NIL
            }}
        )
    );
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));
    auto first = ASSERT_NIL_P(streamer->read());
    ASSERT_EQ(first.size(), 1);
    ASSERT_OCCURRED_AS_P(streamer->read(), freighter::UNREACHABLE);
    ASSERT_OCCURRED_AS(streamer->close(), freighter::UNREACHABLE);
}

TEST(StreamerTest, ReadWakesOnLocalPushNotification) {
    auto bus = std::make_shared<Bus>();
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    auto mock_factory = ::driver::pipeline::testutil::simple_streamer_factory(
        {1},
        reads
    );
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));
    std::thread publisher([&bus] {
        std::this_thread::sleep_for(std::chrono::milliseconds(1));
        x::telem::Frame frame;
        frame.emplace(1, x::telem::Series(static_cast<float>(77.0)));
        bus->publish(frame);
    });
    auto frame = ASSERT_NIL_P(streamer->read());
    ASSERT_EQ(frame.size(), 1);
    publisher.join();
    streamer->close_send();
    ASSERT_NIL(streamer->close());
}

TEST(StreamerTest, LocalFramesTakePriorityOverCoreFrames) {
    auto bus = std::make_shared<Bus>();
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    x::telem::Frame server_frame;
    server_frame.emplace(1, x::telem::Series(static_cast<float>(1.0)));
    reads->push_back(std::move(server_frame));
    auto mock_factory = ::driver::pipeline::testutil::simple_streamer_factory(
        {1},
        reads
    );
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));
    std::this_thread::sleep_for(std::chrono::milliseconds(10));
    x::telem::Frame local_frame;
    local_frame.emplace(1, x::telem::Series(static_cast<float>(2.0)));
    bus->publish(local_frame);
    auto first = ASSERT_NIL_P(streamer->read());
    ASSERT_EQ(first.at<float>(1, 0), 2.0f);
    auto second = ASSERT_NIL_P(streamer->read());
    ASSERT_EQ(second.at<float>(1, 0), 1.0f);
    streamer->close_send();
    ASSERT_NIL(streamer->close());
}

TEST(StreamerTest, CloseWithoutPriorCloseSendShutsDownCleanly) {
    auto bus = std::make_shared<Bus>();
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    auto mock_factory = ::driver::pipeline::testutil::simple_streamer_factory(
        {1},
        reads
    );
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));
    ASSERT_NIL(streamer->close());
}

TEST(StreamerTest, DoubleCloseIsIdempotent) {
    auto bus = std::make_shared<Bus>();
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    auto mock_factory = ::driver::pipeline::testutil::simple_streamer_factory(
        {1},
        reads
    );
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));
    streamer->close_send();
    ASSERT_NIL(streamer->close());
    ASSERT_NIL(streamer->close());
}

TEST(StreamerTest, ServerFramesAdvanceBusAlignment) {
    auto bus = std::make_shared<Bus>();
    bus->register_channels({1});

    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    x::telem::Frame server_frame;
    auto server_series = x::telem::Series(static_cast<float>(99.0));
    server_series.alignment = x::telem::Alignment(5000);
    server_frame.emplace(1, std::move(server_series));
    reads->push_back(std::move(server_frame));
    auto mock_factory = ::driver::pipeline::testutil::simple_streamer_factory(
        {1},
        reads
    );
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));

    // The background thread reads the server frame and advances counters
    // immediately. Give it a moment to process before publishing a local
    // frame, verifying that the counter is advanced eagerly (not lazily
    // when the server frame is consumed via read()).
    std::this_thread::sleep_for(std::chrono::milliseconds(50));

    x::telem::Frame bus_frame;
    bus_frame.emplace(1, x::telem::Series(static_cast<float>(1.0)));
    bus->publish(bus_frame);
    auto local = ASSERT_NIL_P(streamer->read());
    auto local_alignment = local.series->at(0).alignment.uint64();
    EXPECT_GT(local_alignment, 5000)
        << "bus alignment should exceed server alignment even "
        << "before server frame is consumed, got " << local_alignment;

    streamer->close_send();
    ASSERT_NIL(streamer->close());
}

TEST(StreamerTest, PropagatesCoreErrorThroughClose) {
    auto bus = std::make_shared<Bus>();
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    auto read_errors = std::make_shared<std::vector<x::errors::Error>>();
    read_errors->push_back(freighter::UNREACHABLE);
    auto mock_factory = std::make_shared<::driver::pipeline::testutil::StreamerFactory>(
        std::vector<x::errors::Error>{},
        std::make_shared<std::vector<::driver::pipeline::testutil::StreamerConfig>>(
            std::vector{::driver::pipeline::testutil::StreamerConfig{
                reads,
                read_errors,
                x::errors::NIL
            }}
        )
    );
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));
    ASSERT_OCCURRED_AS_P(streamer->read(), freighter::UNREACHABLE);
    ASSERT_OCCURRED_AS(streamer->close(), freighter::UNREACHABLE);
}

/// @brief pushes 20,000 frames at random gaps of 0 to 30 µs, each after the reader
/// takes the one before. Fails if any frame takes over 2 ms to leave streamer.read().
void expect_prompt_wakes(
    Streamer &streamer,
    const std::function<void(x::telem::Frame)> &push
) {
    constexpr int frame_count = 20000;
    constexpr auto slow_threshold = std::chrono::milliseconds(2);
    std::vector<std::chrono::steady_clock::time_point> pushed(frame_count);
    std::atomic<int> received{0};
    std::thread producer([&] {
        // NOLINTNEXTLINE(bugprone-random-generator-seed): repeatable push gaps.
        std::mt19937 rng(42);
        std::uniform_int_distribution delay_us(0, 30);
        for (int i = 0; i < frame_count; i++) {
            while (received.load() < i) {}
            const auto until = std::chrono::steady_clock::now() +
                               std::chrono::microseconds(delay_us(rng));
            while (std::chrono::steady_clock::now() < until) {}
            x::telem::Frame frame(1);
            frame.emplace(1, x::telem::Series(static_cast<int64_t>(i)));
            pushed[i] = std::chrono::steady_clock::now();
            push(std::move(frame));
        }
    });
    std::vector<double> slow_ms;
    for (int i = 0; i < frame_count; i++) {
        auto [frame, err] = streamer.read();
        if (err) {
            ADD_FAILURE() << err;
            received.store(frame_count);
            break;
        }
        const auto latency = std::chrono::steady_clock::now() - pushed[i];
        if (latency > slow_threshold)
            slow_ms.push_back(
                std::chrono::duration<double, std::milli>(latency).count()
            );
        received.store(i + 1);
    }
    producer.join();
    streamer.close_send();
    ASSERT_NIL(streamer.close());
    std::ostringstream slow_list;
    for (const auto ms: slow_ms)
        slow_list << ms << " ";
    EXPECT_TRUE(slow_ms.empty())
        << slow_ms.size() << " of " << frame_count << " frames took more than "
        << slow_threshold.count() << " ms: " << slow_list.str();
}

/// @brief a Core frame should reach a reader waiting in read() within 2 ms.
TEST(StreamerTest, CoreFrameWakesWaitingReader) {
    auto bus = std::make_shared<Bus>();
    auto core = std::make_unique<::driver::pipeline::testutil::HandoffStreamer>();
    auto *core_ptr = core.get();
    Streamer streamer(std::move(core), bus->subscribe({1}), bus);
    expect_prompt_wakes(streamer, [core_ptr](x::telem::Frame frame) {
        core_ptr->push(std::move(frame));
    });
}

/// @brief a bus frame should reach a reader waiting in read() within 2 ms.
TEST(StreamerTest, LocalFrameWakesWaitingReader) {
    auto bus = std::make_shared<Bus>();
    Streamer streamer(
        std::make_unique<::driver::pipeline::testutil::HandoffStreamer>(),
        bus->subscribe({1}),
        bus
    );
    expect_prompt_wakes(streamer, [&bus](const x::telem::Frame &frame) {
        bus->publish(frame);
    });
}
}
