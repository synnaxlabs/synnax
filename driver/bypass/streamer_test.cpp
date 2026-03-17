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

#include "driver/bypass/streamer.h"
#include "driver/pipeline/mock/pipeline.h"

namespace driver::bypass {
const x::control::Subject TEST_SUBJECT{"test_writer", "tw-1"};

TEST(StreamerTest, DeliversLocalBusFrames) {
    Bus bus;
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    auto mock_factory = pipeline::mock::simple_streamer_factory({1}, reads);
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));
    x::telem::Frame local_frame;
    local_frame.emplace(1, x::telem::Series(static_cast<float>(42.0)));
    bus.publish(local_frame);
    auto frame = ASSERT_NIL_P(streamer->read());
    ASSERT_EQ(frame.size(), 1);
    streamer->close_send();
    ASSERT_NIL(streamer->close());
}

TEST(StreamerTest, DeliversServerFrames) {
    Bus bus;
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    x::telem::Frame server_frame;
    server_frame.emplace(1, x::telem::Series(static_cast<float>(99.0)));
    reads->push_back(std::move(server_frame));
    auto mock_factory = pipeline::mock::simple_streamer_factory({1}, reads);
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));
    auto frame = ASSERT_NIL_P(streamer->read());
    ASSERT_EQ(frame.size(), 1);
    streamer->close_send();
    ASSERT_NIL(streamer->close());
}

TEST(StreamerTest, DeliversLocalFramesRegardlessOfAuthority) {
    Bus bus;
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    auto mock_factory = pipeline::mock::simple_streamer_factory({1}, reads);
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));
    x::telem::Frame local_frame;
    local_frame.emplace(1, x::telem::Series(static_cast<float>(42.0)));
    bus.publish(local_frame);
    auto frame = ASSERT_NIL_P(streamer->read());
    ASSERT_EQ(frame.size(), 1);
    streamer->close_send();
    ASSERT_NIL(streamer->close());
}

TEST(StreamerTest, CloseSendDelegatesToServer) {
    Bus bus;
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    auto mock_factory = pipeline::mock::simple_streamer_factory({1}, reads);
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));
    streamer->close_send();
    ASSERT_NIL(streamer->close());
}

TEST(StreamerTest, PropagatesOpenError) {
    Bus bus;
    auto mock_factory = std::make_shared<pipeline::mock::StreamerFactory>(
        std::vector<x::errors::Error>{x::errors::VALIDATION},
        std::make_shared<std::vector<pipeline::mock::StreamerConfig>>()
    );
    StreamerFactory factory(mock_factory, bus, TEST_SUBJECT);
    ASSERT_OCCURRED_AS_P(
        factory.open_streamer({.channels = {1}}),
        x::errors::VALIDATION
    );
}

TEST(StreamerTest, InjectsExcludeGroupsFromSubjectGroup) {
    Bus bus;
    const x::control::Subject grouped_subject{"grouped_writer", "gw-1", 42};
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    x::telem::Frame server_frame;
    server_frame.emplace(1, x::telem::Series(static_cast<float>(99.0)));
    reads->push_back(std::move(server_frame));
    auto mock_factory = std::make_shared<pipeline::mock::StreamerFactory>(
        std::vector<x::errors::Error>{},
        std::make_shared<std::vector<pipeline::mock::StreamerConfig>>(
            std::vector{pipeline::mock::StreamerConfig{reads, {}, x::errors::NIL}}
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
    Bus bus;
    const x::control::Subject no_group_subject{"no_group", "ng-1", 0};
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    x::telem::Frame server_frame;
    server_frame.emplace(1, x::telem::Series(static_cast<float>(99.0)));
    reads->push_back(std::move(server_frame));
    auto mock_factory = std::make_shared<pipeline::mock::StreamerFactory>(
        std::vector<x::errors::Error>{},
        std::make_shared<std::vector<pipeline::mock::StreamerConfig>>(
            std::vector{pipeline::mock::StreamerConfig{reads, {}, x::errors::NIL}}
        )
    );
    StreamerFactory factory(mock_factory, bus, no_group_subject);
    auto streamer = ASSERT_NIL_P(factory.open_streamer({.channels = {1}}));
    ASSERT_TRUE(mock_factory->config.exclude_groups.empty());
    streamer->close_send();
    ASSERT_NIL(streamer->close());
}
}
