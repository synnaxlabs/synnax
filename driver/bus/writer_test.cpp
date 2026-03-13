// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "driver/bus/writer.h"
#include "driver/pipeline/mock/pipeline.h"

namespace driver::bus {
TEST(WriterTest, PublishesToBusAndForwardsToServer) {
    Bus bus;
    auto sub = bus.subscribe({1});
    auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>();
    WriterFactory factory(mock_factory, bus);
    auto [writer, err] = factory.open_writer({.channels = {1}});
    ASSERT_FALSE(err) << err.message();
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(42.0)));
    ASSERT_FALSE(writer->write(frame));
    ASSERT_EQ(mock_factory->writes->size(), 1);
    x::telem::Frame received;
    ASSERT_TRUE(sub->try_pop(received));
    ASSERT_EQ(received.size(), 1);
}

TEST(WriterTest, SkipsBusPublishWhenNoRoutes) {
    Bus bus;
    auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>();
    WriterFactory factory(mock_factory, bus);
    auto [writer, err] = factory.open_writer({.channels = {1}});
    ASSERT_FALSE(err) << err.message();
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(42.0)));
    ASSERT_FALSE(writer->write(frame));
    ASSERT_EQ(mock_factory->writes->size(), 1);
}

TEST(WriterTest, DelegatesSetAuthority) {
    Bus bus;
    auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>();
    WriterFactory factory(mock_factory, bus);
    auto [writer, err] = factory.open_writer({.channels = {1}});
    ASSERT_FALSE(err) << err.message();
    pipeline::Authorities auth{.keys = {1}, .authorities = {200}};
    ASSERT_FALSE(writer->set_authority(auth));
    ASSERT_EQ(mock_factory->authority_changes->size(), 1);
}

TEST(WriterTest, DelegatesClose) {
    Bus bus;
    auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>();
    WriterFactory factory(mock_factory, bus);
    auto [writer, err] = factory.open_writer({.channels = {1}});
    ASSERT_FALSE(err) << err.message();
    ASSERT_FALSE(writer->close());
}

TEST(WriterTest, PropagatesOpenError) {
    Bus bus;
    auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(
        std::make_shared<std::vector<x::telem::Frame>>(),
        std::vector<x::errors::Error>{x::errors::VALIDATION}
    );
    WriterFactory factory(mock_factory, bus);
    auto [writer, err] = factory.open_writer({.channels = {1}});
    ASSERT_TRUE(err);
    ASSERT_EQ(writer, nullptr);
}
}
