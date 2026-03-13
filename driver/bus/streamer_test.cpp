// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "driver/bus/streamer.h"
#include "driver/pipeline/mock/pipeline.h"

namespace driver::bus {
const x::control::Subject TEST_SUBJECT{"test_writer", "tw-1"};

TEST(StreamerTest, ReturnsLocalFramesFirst) {
    Bus bus;
    AuthorityMirror mirror;
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    x::telem::Frame server_frame;
    server_frame.emplace(1, x::telem::Series(static_cast<float>(99.0)));
    reads->push_back(std::move(server_frame));
    auto mock_factory = pipeline::mock::simple_streamer_factory({1}, reads);
    StreamerFactory factory(mock_factory, bus, mirror, TEST_SUBJECT);
    auto [streamer, err] = factory.open_streamer({.channels = {1}});
    ASSERT_FALSE(err) << err.message();
    x::telem::Frame local_frame;
    local_frame.emplace(1, x::telem::Series(static_cast<float>(42.0)));
    bus.publish(local_frame);
    auto [frame, read_err] = streamer->read();
    ASSERT_FALSE(read_err) << read_err.message();
    ASSERT_EQ(frame.size(), 1);
}

TEST(StreamerTest, FallsThroughToServerWhenNoLocalFrames) {
    Bus bus;
    AuthorityMirror mirror;
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    x::telem::Frame server_frame;
    server_frame.emplace(1, x::telem::Series(static_cast<float>(99.0)));
    reads->push_back(std::move(server_frame));
    auto mock_factory = pipeline::mock::simple_streamer_factory({1}, reads);
    StreamerFactory factory(mock_factory, bus, mirror, TEST_SUBJECT);
    auto [streamer, err] = factory.open_streamer({.channels = {1}});
    ASSERT_FALSE(err) << err.message();
    auto [frame, read_err] = streamer->read();
    ASSERT_FALSE(read_err) << read_err.message();
    ASSERT_EQ(frame.size(), 1);
}

TEST(StreamerTest, FiltersLocalFramesByAuthority) {
    Bus bus;
    AuthorityMirror mirror;
    const x::control::Subject OTHER{"other_writer", "ow-1"};
    mirror.apply({.transfers = {{
        .from = std::nullopt,
        .to = x::control::State{.resource = 1, .subject = OTHER, .authority = 200},
    }}});
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    x::telem::Frame server_frame;
    server_frame.emplace(1, x::telem::Series(static_cast<float>(99.0)));
    reads->push_back(std::move(server_frame));
    auto mock_factory = pipeline::mock::simple_streamer_factory({1}, reads);
    StreamerFactory factory(mock_factory, bus, mirror, TEST_SUBJECT);
    auto [streamer, err] = factory.open_streamer({.channels = {1}});
    ASSERT_FALSE(err) << err.message();
    x::telem::Frame local_frame;
    local_frame.emplace(1, x::telem::Series(static_cast<float>(42.0)));
    bus.publish(local_frame);
    auto [frame, read_err] = streamer->read();
    ASSERT_FALSE(read_err) << read_err.message();
    ASSERT_EQ(frame.size(), 1);
}

TEST(StreamerTest, CloseSendDelegatesToServer) {
    Bus bus;
    AuthorityMirror mirror;
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    auto mock_factory = pipeline::mock::simple_streamer_factory({1}, reads);
    StreamerFactory factory(mock_factory, bus, mirror, TEST_SUBJECT);
    auto [streamer, err] = factory.open_streamer({.channels = {1}});
    ASSERT_FALSE(err) << err.message();
    streamer->close_send();
    ASSERT_FALSE(streamer->close());
}

TEST(StreamerTest, PropagatesOpenError) {
    Bus bus;
    AuthorityMirror mirror;
    auto mock_factory = std::make_shared<pipeline::mock::StreamerFactory>(
        std::vector<x::errors::Error>{x::errors::VALIDATION},
        std::make_shared<std::vector<pipeline::mock::StreamerConfig>>()
    );
    StreamerFactory factory(mock_factory, bus, mirror, TEST_SUBJECT);
    auto [streamer, err] = factory.open_streamer({.channels = {1}});
    ASSERT_TRUE(err);
    ASSERT_EQ(streamer, nullptr);
}
}
