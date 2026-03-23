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

#include "driver/bypass/pipeline/streamer.h"
#include "driver/pipeline/mock/pipeline.h"

namespace driver::bypass::pipeline {
const x::control::Subject TEST_SUBJECT{"test_writer", "tw-1"};

TEST(StreamerTest, DeliversLocalBusFrames) {
    auto bus = std::make_shared<Bus>();
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    auto mock_factory = ::driver::pipeline::mock::simple_streamer_factory({1}, reads);
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
    auto mock_factory = ::driver::pipeline::mock::simple_streamer_factory({1}, reads);
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
    auto mock_factory = ::driver::pipeline::mock::simple_streamer_factory({1}, reads);
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
    auto mock_factory = ::driver::pipeline::mock::simple_streamer_factory({1}, reads);
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));
    streamer->close_send();
    ASSERT_NIL(streamer->close());
}

TEST(StreamerTest, PropagatesOpenError) {
    auto bus = std::make_shared<Bus>();
    auto mock_factory = std::make_shared<::driver::pipeline::mock::StreamerFactory>(
        std::vector<x::errors::Error>{x::errors::VALIDATION},
        std::make_shared<std::vector<::driver::pipeline::mock::StreamerConfig>>()
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
    auto mock_factory = std::make_shared<::driver::pipeline::mock::StreamerFactory>(
        std::vector<x::errors::Error>{},
        std::make_shared<std::vector<::driver::pipeline::mock::StreamerConfig>>(
            std::vector{
                ::driver::pipeline::mock::StreamerConfig{reads, {}, x::errors::NIL}
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
    auto mock_factory = std::make_shared<::driver::pipeline::mock::StreamerFactory>(
        std::vector<x::errors::Error>{},
        std::make_shared<std::vector<::driver::pipeline::mock::StreamerConfig>>(
            std::vector{
                ::driver::pipeline::mock::StreamerConfig{reads, {}, x::errors::NIL}
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
    reads->push_back(x::telem::Frame());
    auto read_errors = std::make_shared<std::vector<x::errors::Error>>();
    read_errors->push_back(x::errors::NIL);
    read_errors->push_back(freighter::UNREACHABLE);
    auto mock_factory = std::make_shared<::driver::pipeline::mock::StreamerFactory>(
        std::vector<x::errors::Error>{},
        std::make_shared<
            std::vector<::driver::pipeline::mock::StreamerConfig>>(std::vector{
            ::driver::pipeline::mock::StreamerConfig{reads, read_errors, x::errors::NIL}
        })
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
    auto mock_factory = ::driver::pipeline::mock::simple_streamer_factory({1}, reads);
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
    auto mock_factory = ::driver::pipeline::mock::simple_streamer_factory({1}, reads);
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
    auto mock_factory = ::driver::pipeline::mock::simple_streamer_factory({1}, reads);
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));
    ASSERT_NIL(streamer->close());
}

TEST(StreamerTest, DoubleCloseIsIdempotent) {
    auto bus = std::make_shared<Bus>();
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    auto mock_factory = ::driver::pipeline::mock::simple_streamer_factory({1}, reads);
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));
    streamer->close_send();
    ASSERT_NIL(streamer->close());
    ASSERT_NIL(streamer->close());
}

TEST(StreamerTest, PropagatesCoreErrorThroughClose) {
    auto bus = std::make_shared<Bus>();
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    auto read_errors = std::make_shared<std::vector<x::errors::Error>>();
    read_errors->push_back(freighter::UNREACHABLE);
    auto mock_factory = std::make_shared<::driver::pipeline::mock::StreamerFactory>(
        std::vector<x::errors::Error>{},
        std::make_shared<
            std::vector<::driver::pipeline::mock::StreamerConfig>>(std::vector{
            ::driver::pipeline::mock::StreamerConfig{reads, read_errors, x::errors::NIL}
        })
    );
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));
    ASSERT_OCCURRED_AS_P(streamer->read(), freighter::UNREACHABLE);
    ASSERT_OCCURRED_AS(streamer->close(), freighter::UNREACHABLE);
}
}
