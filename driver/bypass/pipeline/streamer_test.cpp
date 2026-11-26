// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/errors/errors.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/bypass/pipeline/streamer.h"
#include "driver/pipeline/control.h"
#include "driver/pipeline/testutil/pipeline.h"

namespace driver::bypass::pipeline {
const x::control::Subject TEST_SUBJECT{"test_writer", "tw-1"};
const x::breaker::Config TEST_BREAKER{
    .name = "test_streamer",
    .base_interval = x::telem::MILLISECOND,
    .max_retries = 2,
    .scale = 1,
    .max_interval = x::telem::MILLISECOND,
};

namespace {
/// @brief returns channels with no index for the given keys.
std::vector<synnax::channel::Channel>
unindexed(const std::vector<synnax::channel::Key> &keys) {
    std::vector<synnax::channel::Channel> channels(keys.size());
    for (size_t i = 0; i < keys.size(); i++)
        channels[i].key = keys[i];
    return channels;
}

std::shared_ptr<::driver::pipeline::testutil::StreamerFactory> failing_core_factory(
    std::vector<x::errors::Error> open_errors,
    std::vector<::driver::pipeline::testutil::StreamerConfig> configs = {
        {std::make_shared<std::vector<x::telem::Frame>>(), {}, x::errors::NIL}
    }
) {
    return std::make_shared<::driver::pipeline::testutil::StreamerFactory>(
        std::move(open_errors),
        std::make_shared<std::vector<::driver::pipeline::testutil::StreamerConfig>>(
            std::move(configs)
        )
    );
}

x::telem::Frame float_frame(const float value) {
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(value));
    return frame;
}
}

TEST(StreamerTest, DeliversLocalBusFrames) {
    auto bus = std::make_shared<Bus>();
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    auto mock_factory = ::driver::pipeline::testutil::simple_streamer_factory(
        {1},
        reads
    );
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT, TEST_BREAKER);
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
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT, TEST_BREAKER);
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
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT, TEST_BREAKER);
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
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT, TEST_BREAKER);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));
    streamer->close_send();
    ASSERT_NIL(streamer->close());
}

/// @brief it should open the Core streamer before open_streamer returns.
TEST(StreamerTest, OpensCoreBeforeReturning) {
    auto bus = std::make_shared<Bus>();
    auto mock_factory = failing_core_factory({});
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT, TEST_BREAKER);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));
    ASSERT_EQ(mock_factory->streamer_opens.load(), 1);
    ASSERT_NIL(streamer->close());
}

/// @brief it should return a Core open error that is not an outage, without retry.
TEST(StreamerTest, ReturnsNonOutageOpenError) {
    for (const auto &open_err:
         {x::errors::VALIDATION, synnax::errors::LICENSE_MISSING}) {
        auto bus = std::make_shared<Bus>();
        auto mock_factory = failing_core_factory({open_err});
        StreamerFactory factory(mock_factory, bus, TEST_SUBJECT, TEST_BREAKER);
        ASSERT_OCCURRED_AS_P(factory.open_streamer({.channels = {1}}), open_err);
        ASSERT_EQ(mock_factory->streamer_opens.load(), 1);
    }
}

/// @brief it should deliver local frames while the Core streamer fails to open as
/// unreachable.
TEST(StreamerTest, DeliversLocalFramesWhileCoreIsUnreachable) {
    auto bus = std::make_shared<Bus>();
    auto mock_factory = failing_core_factory(
        std::vector(10000, freighter::UNREACHABLE)
    );
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT, TEST_BREAKER);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));
    ASSERT_EVENTUALLY_GE(mock_factory->streamer_opens.load(), 5);
    for (const float value: {1.0f, 2.0f}) {
        bus->publish(float_frame(value));
        auto frame = ASSERT_NIL_P(streamer->read());
        ASSERT_EQ(frame.at<float>(1, 0), value);
    }
    ASSERT_NIL(streamer->close());
}

/// @brief it should reopen the Core streamer after an unreachable Core, and deliver
/// Core frames again.
TEST(StreamerTest, ReopensCoreAfterUnreachable) {
    auto first_reads = std::make_shared<std::vector<x::telem::Frame>>();
    first_reads->push_back(float_frame(1));
    first_reads->emplace_back();
    auto first_errors = std::make_shared<std::vector<x::errors::Error>>(
        std::vector{x::errors::NIL, freighter::UNREACHABLE}
    );
    auto second_reads = std::make_shared<std::vector<x::telem::Frame>>();
    second_reads->push_back(float_frame(2));
    auto bus = std::make_shared<Bus>();
    auto mock_factory = failing_core_factory(
        {x::errors::NIL, freighter::UNREACHABLE},
        {
            {first_reads, first_errors, x::errors::NIL},
            {second_reads, {}, x::errors::NIL},
        }
    );
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT, TEST_BREAKER);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));
    auto first = ASSERT_NIL_P(streamer->read());
    ASSERT_EQ(first.at<float>(1, 0), 1.0f);
    auto second = ASSERT_NIL_P(streamer->read());
    ASSERT_EQ(second.at<float>(1, 0), 2.0f);
    ASSERT_EQ(mock_factory->streamer_opens.load(), 3);
    ASSERT_NIL(streamer->close());
}

/// @brief it should end read at the first license error and leave the retry to the
/// pipeline.
TEST(StreamerTest, LicenseReadErrorEndsRead) {
    auto bus = std::make_shared<Bus>();
    auto mock_factory = failing_core_factory(
        {},
        {{std::make_shared<std::vector<x::telem::Frame>>(),
          std::make_shared<std::vector<x::errors::Error>>(
              std::vector{synnax::errors::LICENSE_MISSING}
          ),
          x::errors::NIL}}
    );
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT, TEST_BREAKER);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));
    ASSERT_OCCURRED_AS_P(streamer->read(), synnax::errors::LICENSE_MISSING);
    ASSERT_EQ(mock_factory->streamer_opens.load(), 1);
    ASSERT_OCCURRED_AS(streamer->close(), synnax::errors::LICENSE_MISSING);
}

/// @brief it should let a control pipeline write local frames to its sink while the
/// Core is unreachable.
TEST(StreamerTest, ControlPipelineWritesLocalFramesWhileCoreIsUnreachable) {
    auto bus = std::make_shared<Bus>();
    auto mock_factory = failing_core_factory(
        std::vector(10000, freighter::UNREACHABLE)
    );
    auto factory = std::make_shared<StreamerFactory>(
        mock_factory,
        bus,
        TEST_SUBJECT,
        TEST_BREAKER
    );
    auto sink = std::make_shared<::driver::pipeline::testutil::Sink>();
    ::driver::pipeline::Control control(factory, {.channels = {1}}, sink, TEST_BREAKER);
    control.start();
    ASSERT_EVENTUALLY_EQ(sink->opens.load(), 1);
    bus->publish(float_frame(1));
    bus->publish(float_frame(2));
    ASSERT_EVENTUALLY_EQ(sink->writes->size(), 2);
    ASSERT_GE(mock_factory->streamer_opens.load(), 1);
    control.stop();
    ASSERT_NIL(sink->stop_err);
    ASSERT_EQ(sink->opens.load(), 1);
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
    StreamerFactory factory(mock_factory, bus, grouped_subject, TEST_BREAKER);
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
    StreamerFactory factory(mock_factory, bus, no_group_subject, TEST_BREAKER);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));
    ASSERT_TRUE(mock_factory->config.exclude_groups.empty());
    streamer->close_send();
    ASSERT_NIL(streamer->close());
}

/// @brief it should end read with a Core read error that is not an outage.
TEST(StreamerTest, NonOutageReadErrorEndsRead) {
    auto bus = std::make_shared<Bus>();
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    x::telem::Frame valid_frame;
    valid_frame.emplace(1, x::telem::Series(static_cast<float>(99.0)));
    reads->push_back(std::move(valid_frame));
    reads->emplace_back();
    auto read_errors = std::make_shared<std::vector<x::errors::Error>>();
    read_errors->push_back(x::errors::NIL);
    read_errors->push_back(x::errors::VALIDATION);
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
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT, TEST_BREAKER);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));
    auto first = ASSERT_NIL_P(streamer->read());
    ASSERT_EQ(first.size(), 1);
    ASSERT_OCCURRED_AS_P(streamer->read(), x::errors::VALIDATION);
    ASSERT_OCCURRED_AS(streamer->close(), x::errors::VALIDATION);
}

TEST(StreamerTest, ReadWakesOnLocalPushNotification) {
    auto bus = std::make_shared<Bus>();
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    auto mock_factory = ::driver::pipeline::testutil::simple_streamer_factory(
        {1},
        reads
    );
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT, TEST_BREAKER);
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
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT, TEST_BREAKER);
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
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT, TEST_BREAKER);
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
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT, TEST_BREAKER);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));
    streamer->close_send();
    ASSERT_NIL(streamer->close());
    ASSERT_NIL(streamer->close());
}

TEST(StreamerTest, ServerFramesAdvanceBusAlignment) {
    auto bus = std::make_shared<Bus>();
    bus->register_channels(unindexed({1}));

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
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT, TEST_BREAKER);
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

/// @brief it should return a non-outage Core error from close.
TEST(StreamerTest, PropagatesNonOutageCoreErrorThroughClose) {
    auto bus = std::make_shared<Bus>();
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    auto read_errors = std::make_shared<std::vector<x::errors::Error>>();
    read_errors->push_back(x::errors::VALIDATION);
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
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT, TEST_BREAKER);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));
    ASSERT_OCCURRED_AS_P(streamer->read(), x::errors::VALIDATION);
    ASSERT_OCCURRED_AS(streamer->close(), x::errors::VALIDATION);
}
}
