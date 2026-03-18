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

#include "driver/bypass/writer.h"
#include "driver/pipeline/mock/pipeline.h"

namespace driver::bypass {
TEST(WriterTest, PublishesToBusAndForwardsToServer) {
    Bus bus;
    AuthorityMirror mirror;
    auto sub = bus.subscribe({1});
    auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>();
    WriterFactory factory(mock_factory, bus, 0, mirror);
    auto writer = ASSERT_NIL_P(factory.open_writer({.channels = {1}}));
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(42.0)));
    ASSERT_NIL(writer->write(frame));
    ASSERT_EQ(mock_factory->writes->size(), 1);
    x::telem::Frame received;
    ASSERT_TRUE(sub->try_pop(received));
    ASSERT_EQ(received.size(), 1);
}

TEST(WriterTest, WritesWithNoSubscribers) {
    Bus bus;
    AuthorityMirror mirror;
    auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>();
    WriterFactory factory(mock_factory, bus, 0, mirror);
    auto writer = ASSERT_NIL_P(factory.open_writer({.channels = {1}}));
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(42.0)));
    ASSERT_NIL(writer->write(frame));
    ASSERT_EQ(mock_factory->writes->size(), 1);
}

TEST(WriterTest, LateSubscriberReceivesFrames) {
    Bus bus;
    AuthorityMirror mirror;
    auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>();
    WriterFactory factory(mock_factory, bus, 0, mirror);
    auto writer = ASSERT_NIL_P(factory.open_writer({.channels = {1}}));
    auto sub = bus.subscribe({1});
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(42.0)));
    ASSERT_NIL(writer->write(frame));
    x::telem::Frame received;
    ASSERT_TRUE(sub->try_pop(received));
    ASSERT_EQ(received.size(), 1);
}

TEST(WriterTest, DelegatesSetAuthority) {
    Bus bus;
    AuthorityMirror mirror;
    auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>();
    WriterFactory factory(mock_factory, bus, 0, mirror);
    auto writer = ASSERT_NIL_P(factory.open_writer({.channels = {1}}));
    pipeline::Authorities auth{.keys = {1}, .authorities = {200}};
    ASSERT_NIL(writer->set_authority(auth));
    ASSERT_EQ(mock_factory->authority_changes->size(), 1);
}

TEST(WriterTest, DelegatesClose) {
    Bus bus;
    AuthorityMirror mirror;
    auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>();
    WriterFactory factory(mock_factory, bus, 0, mirror);
    auto writer = ASSERT_NIL_P(factory.open_writer({.channels = {1}}));
    ASSERT_NIL(writer->close());
}

TEST(WriterTest, PropagatesOpenError) {
    Bus bus;
    AuthorityMirror mirror;
    auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(
        std::make_shared<std::vector<x::telem::Frame>>(),
        std::vector<x::errors::Error>{x::errors::VALIDATION}
    );
    WriterFactory factory(mock_factory, bus, 0, mirror);
    ASSERT_OCCURRED_AS_P(factory.open_writer({.channels = {1}}), x::errors::VALIDATION);
}

TEST(WriterTest, InjectsGroupIntoWriterConfig) {
    Bus bus;
    AuthorityMirror mirror;
    auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>();
    WriterFactory factory(mock_factory, bus, 77, mirror);
    auto writer = ASSERT_NIL_P(factory.open_writer({.channels = {1}}));
    ASSERT_EQ(mock_factory->config.subject.group, 77);
}

TEST(WriterTest, DoesNotOverrideExistingGroup) {
    Bus bus;
    AuthorityMirror mirror;
    auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>();
    WriterFactory factory(mock_factory, bus, 77, mirror);
    auto writer = ASSERT_NIL_P(factory.open_writer({
        .channels = {1},
        .subject = x::control::Subject{"w", "w", 99},
    }));
    ASSERT_EQ(mock_factory->config.subject.group, 99);
}

TEST(WriterTest, DoesNotInjectGroupWhenZero) {
    Bus bus;
    AuthorityMirror mirror;
    auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>();
    WriterFactory factory(mock_factory, bus, 0, mirror);
    auto writer = ASSERT_NIL_P(factory.open_writer({.channels = {1}}));
    ASSERT_EQ(mock_factory->config.subject.group, 0);
}

TEST(WriterTest, SetAuthorityIncreaseUpdatesMirror) {
    Bus bus;
    AuthorityMirror mirror;
    auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>();
    WriterFactory factory(mock_factory, bus, 0, mirror);
    auto writer = ASSERT_NIL_P(factory.open_writer({
        .channels = {1, 2},
        .subject = x::control::Subject{"arc", "arc-1"},
    }));
    ASSERT_NIL(writer->set_authority({.keys = {1, 2}, .authorities = {200}}));
    ASSERT_TRUE(mirror.is_authorized(1, {"arc", "arc-1"}));
    ASSERT_TRUE(mirror.is_authorized(2, {"arc", "arc-1"}));
}

TEST(WriterTest, SetAuthorityDecreaseDoesNotUpdateMirror) {
    Bus bus;
    AuthorityMirror mirror;
    const x::control::Subject arc{"arc", "arc-1"};
    const x::control::Subject op{"operator", "op-1"};
    mirror.apply_increase(op, 1, 200);
    auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>();
    WriterFactory factory(mock_factory, bus, 0, mirror);
    auto writer = ASSERT_NIL_P(factory.open_writer({
        .channels = {1},
        .subject = arc,
    }));
    ASSERT_NIL(writer->set_authority({.keys = {1}, .authorities = {100}}));
    ASSERT_TRUE(mirror.is_authorized(1, op));
    ASSERT_FALSE(mirror.is_authorized(1, arc));
}

TEST(WriterTest, SetAuthorityGlobalKeysExpandsToWriterChannels) {
    Bus bus;
    AuthorityMirror mirror;
    auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>();
    WriterFactory factory(mock_factory, bus, 0, mirror);
    auto writer = ASSERT_NIL_P(factory.open_writer({
        .channels = {1, 2, 3},
        .subject = x::control::Subject{"arc", "arc-1"},
    }));
    ASSERT_NIL(writer->set_authority({.keys = {}, .authorities = {255}}));
    ASSERT_TRUE(mirror.is_authorized(1, {"arc", "arc-1"}));
    ASSERT_TRUE(mirror.is_authorized(2, {"arc", "arc-1"}));
    ASSERT_TRUE(mirror.is_authorized(3, {"arc", "arc-1"}));
}

TEST(WriterTest, SetAuthorityIncreaseEndToEnd) {
    Bus bus;
    AuthorityMirror mirror;
    const x::control::Subject hotfire{"hotfire", "hf-1"};
    const x::control::Subject abort_sub{"abort", "abort-1"};

    mirror.apply_increase(hotfire, 1, 200);

    auto mock_writer_factory = std::make_shared<pipeline::mock::WriterFactory>();
    WriterFactory writer_factory(mock_writer_factory, bus, 0, mirror);
    auto writer = ASSERT_NIL_P(writer_factory.open_writer({
        .channels = {1},
        .subject = abort_sub,
    }));

    x::telem::Frame hotfire_frame;
    hotfire_frame.emplace(1, x::telem::Series(static_cast<float>(1.0)));
    auto filtered_before = mirror.filter(hotfire_frame, hotfire);
    ASSERT_EQ(filtered_before.size(), 1);

    ASSERT_NIL(writer->set_authority({.keys = {1}, .authorities = {255}}));

    auto filtered_after = mirror.filter(hotfire_frame, hotfire);
    ASSERT_TRUE(filtered_after.empty());

    x::telem::Frame abort_frame;
    abort_frame.emplace(1, x::telem::Series(static_cast<float>(0.0)));
    auto abort_filtered = mirror.filter(abort_frame, abort_sub);
    ASSERT_EQ(abort_filtered.size(), 1);
}

TEST(WriterTest, WriteFiltersUnauthorizedChannelsFromBus) {
    Bus bus;
    AuthorityMirror mirror;
    const x::control::Subject arc{"arc", "arc-1"};
    const x::control::Subject other{"other", "other-1"};
    mirror.apply_increase(other, 1, 200);
    auto sub = bus.subscribe({1, 2});
    auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>();
    WriterFactory factory(mock_factory, bus, 0, mirror);
    auto writer = ASSERT_NIL_P(factory.open_writer({
        .channels = {1, 2},
        .subject = arc,
    }));
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(1.0)));
    frame.emplace(2, x::telem::Series(static_cast<float>(2.0)));
    ASSERT_NIL(writer->write(frame));
    ASSERT_EQ(mock_factory->writes->size(), 1);
    ASSERT_EQ(mock_factory->writes->at(0).size(), 2);
    x::telem::Frame received;
    ASSERT_TRUE(sub->try_pop(received));
    ASSERT_EQ(received.size(), 1);
}

TEST(WriterTest, WritePublishesNothingToBusWhenFullyUnauthorized) {
    Bus bus;
    AuthorityMirror mirror;
    const x::control::Subject arc{"arc", "arc-1"};
    const x::control::Subject other{"other", "other-1"};
    mirror.apply_increase(other, 1, 200);
    auto sub = bus.subscribe({1});
    auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>();
    WriterFactory factory(mock_factory, bus, 0, mirror);
    auto writer = ASSERT_NIL_P(factory.open_writer({
        .channels = {1},
        .subject = arc,
    }));
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(1.0)));
    ASSERT_NIL(writer->write(frame));
    ASSERT_EQ(mock_factory->writes->size(), 1);
    x::telem::Frame received;
    ASSERT_FALSE(sub->try_pop(received));
}

TEST(WriterTest, WriteStillPublishesWhenNoAuthorityState) {
    Bus bus;
    AuthorityMirror mirror;
    auto sub = bus.subscribe({1});
    auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>();
    WriterFactory factory(mock_factory, bus, 0, mirror);
    auto writer = ASSERT_NIL_P(factory.open_writer({
        .channels = {1},
        .subject = x::control::Subject{"arc", "arc-1"},
    }));
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(42.0)));
    ASSERT_NIL(writer->write(frame));
    x::telem::Frame received;
    ASSERT_TRUE(sub->try_pop(received));
    ASSERT_EQ(received.size(), 1);
}

TEST(WriterTest, WriteFilterEndToEnd) {
    Bus bus;
    AuthorityMirror mirror;
    const x::control::Subject hotfire{"hotfire", "hf-1"};
    const x::control::Subject abort_sub{"abort", "abort-1"};

    auto sub = bus.subscribe({1});
    auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>();
    WriterFactory factory(mock_factory, bus, 0, mirror);

    auto hf_writer = ASSERT_NIL_P(factory.open_writer({
        .channels = {1},
        .subject = hotfire,
    }));
    auto ab_writer = ASSERT_NIL_P(factory.open_writer({
        .channels = {1},
        .subject = abort_sub,
    }));

    ASSERT_NIL(hf_writer->set_authority({.keys = {1}, .authorities = {200}}));

    x::telem::Frame hf_frame;
    hf_frame.emplace(1, x::telem::Series(static_cast<float>(1.0)));
    ASSERT_NIL(hf_writer->write(hf_frame));
    x::telem::Frame received;
    ASSERT_TRUE(sub->try_pop(received));
    ASSERT_EQ(received.size(), 1);

    ASSERT_NIL(ab_writer->set_authority({.keys = {1}, .authorities = {255}}));

    x::telem::Frame hf_frame2;
    hf_frame2.emplace(1, x::telem::Series(static_cast<float>(2.0)));
    ASSERT_NIL(hf_writer->write(hf_frame2));
    ASSERT_FALSE(sub->try_pop(received));

    x::telem::Frame ab_frame;
    ab_frame.emplace(1, x::telem::Series(static_cast<float>(0.0)));
    ASSERT_NIL(ab_writer->write(ab_frame));
    ASSERT_TRUE(sub->try_pop(received));
    ASSERT_EQ(received.size(), 1);
}

/// @brief set_authority should return a validation error when the authorities
/// size does not match the keys size and is not 1.
TEST(WriterTest, SetAuthorityRejectsMismatchedSizes) {
    Bus bus;
    AuthorityMirror mirror;
    auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>();
    WriterFactory factory(mock_factory, bus, 0, mirror);
    auto writer = ASSERT_NIL_P(factory.open_writer({.channels = {1, 2, 3}}));
    ASSERT_OCCURRED_AS(
        writer->set_authority({.keys = {1, 2, 3}, .authorities = {100, 200}}),
        x::errors::VALIDATION
    );
}

/// @brief set_authority should accept a single authority broadcast to all keys.
TEST(WriterTest, SetAuthoritySingleAuthorityBroadcast) {
    Bus bus;
    AuthorityMirror mirror;
    auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>();
    WriterFactory factory(mock_factory, bus, 0, mirror);
    auto writer = ASSERT_NIL_P(factory.open_writer({.channels = {1, 2, 3}}));
    ASSERT_NIL(writer->set_authority({.keys = {1, 2, 3}, .authorities = {200}}));
}

/// @brief set_authority should accept per-key authorities with matching sizes.
TEST(WriterTest, SetAuthorityMatchingSizes) {
    Bus bus;
    AuthorityMirror mirror;
    auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>();
    WriterFactory factory(mock_factory, bus, 0, mirror);
    auto writer = ASSERT_NIL_P(factory.open_writer({.channels = {1, 2}}));
    ASSERT_NIL(writer->set_authority({.keys = {1, 2}, .authorities = {100, 200}}));
}
}
