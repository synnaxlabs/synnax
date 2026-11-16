// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "freighter/cpp/freighter.h"
#include "x/cpp/sync/list.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/bypass/pipeline/writer.h"
#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/testutil/pipeline.h"

namespace driver::bypass::pipeline {
/// @brief a source that numbers its frames on channel 1 from 0.
class CountingSource final : public ::driver::pipeline::Source {
    uint64_t reads = 0;

public:
    x::errors::Error read(
        x::breaker::Breaker &,
        x::telem::Frame &fr,
        ::driver::pipeline::Authorities &
    ) override {
        std::this_thread::sleep_for(std::chrono::microseconds(100));
        fr.emplace(1, x::telem::Series(this->reads++));
        return x::errors::NIL;
    }
};

/// @brief it should send a frame that the acquisition pipeline writes again to a
/// reopened writer to bus subscribers once.
TEST(WriterTest, RewrittenFrameReachesBusOnce) {
    auto bus = std::make_shared<Bus>();
    auto sub = bus->subscribe({1});
    const auto server = std::make_shared<::driver::pipeline::testutil::WriterFactory>(
        std::make_shared<x::sync::List<x::telem::Frame>>(),
        std::vector<x::errors::Error>{},
        std::vector{freighter::UNREACHABLE},
        std::vector{1}
    );
    auto pipeline = ::driver::pipeline::Acquisition(
        std::make_shared<WriterFactory>(
            server,
            bus,
            std::make_shared<control::States>(),
            0,
            ForwarderConfig{.breaker = {.base_interval = x::telem::MICROSECOND * 10}}
        ),
        synnax::framer::WriterConfig{.channels = {1}},
        std::make_shared<CountingSource>(),
        x::breaker::Config{.base_interval = x::telem::MICROSECOND * 10}
    );
    ASSERT_TRUE(pipeline.start());
    ASSERT_EVENTUALLY_GE(server->writes->size(), 3);
    ASSERT_TRUE(pipeline.stop());
    std::vector<uint64_t> received;
    x::telem::Frame fr;
    while (sub->try_pop(fr))
        received.push_back(fr.at<uint64_t>(1, 0));
    ASSERT_GE(received.size(), 3);
    for (size_t i = 0; i < received.size(); i++)
        EXPECT_EQ(received[i], i);
}

TEST(WriterTest, PublishesToBusAndForwardsToCore) {
    auto bus = std::make_shared<Bus>();
    auto states = std::make_shared<control::States>();
    auto sub = bus->subscribe({1});
    auto mock_factory = std::make_shared<::driver::pipeline::testutil::WriterFactory>();
    WriterFactory factory(mock_factory, bus, states, 0);
    auto writer = ASSERT_NIL_P(factory.open_writer({.channels = {1}}));
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(42.0)));
    ASSERT_NIL(writer->write(frame));
    ASSERT_EVENTUALLY_EQ(mock_factory->writes->size(), 1);
    x::telem::Frame received;
    ASSERT_TRUE(sub->try_pop(received));
    ASSERT_EQ(received.size(), 1);
}

TEST(WriterTest, WritesWithNoSubscribers) {
    auto bus = std::make_shared<Bus>();
    auto states = std::make_shared<control::States>();
    auto mock_factory = std::make_shared<::driver::pipeline::testutil::WriterFactory>();
    WriterFactory factory(mock_factory, bus, states, 0);
    auto writer = ASSERT_NIL_P(factory.open_writer({.channels = {1}}));
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(42.0)));
    ASSERT_NIL(writer->write(frame));
    ASSERT_EVENTUALLY_EQ(mock_factory->writes->size(), 1);
}

TEST(WriterTest, LateSubscriberReceivesFrames) {
    auto bus = std::make_shared<Bus>();
    auto states = std::make_shared<control::States>();
    auto mock_factory = std::make_shared<::driver::pipeline::testutil::WriterFactory>();
    WriterFactory factory(mock_factory, bus, states, 0);
    auto writer = ASSERT_NIL_P(factory.open_writer({.channels = {1}}));
    auto sub = bus->subscribe({1});
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(42.0)));
    ASSERT_NIL(writer->write(frame));
    x::telem::Frame received;
    ASSERT_TRUE(sub->try_pop(received));
    ASSERT_EQ(received.size(), 1);
}

TEST(WriterTest, DelegatesSetAuthority) {
    auto bus = std::make_shared<Bus>();
    auto states = std::make_shared<control::States>();
    auto mock_factory = std::make_shared<::driver::pipeline::testutil::WriterFactory>();
    WriterFactory factory(mock_factory, bus, states, 0);
    auto writer = ASSERT_NIL_P(factory.open_writer({.channels = {1}}));
    ::driver::pipeline::Authorities auth{.keys = {1}, .authorities = {200}};
    ASSERT_NIL(writer->set_authority(auth));
    ASSERT_EVENTUALLY_EQ(mock_factory->authority_changes->size(), 1);
}

TEST(WriterTest, DelegatesClose) {
    auto bus = std::make_shared<Bus>();
    auto states = std::make_shared<control::States>();
    auto mock_factory = std::make_shared<::driver::pipeline::testutil::WriterFactory>();
    WriterFactory factory(mock_factory, bus, states, 0);
    auto writer = ASSERT_NIL_P(factory.open_writer({.channels = {1}}));
    ASSERT_NIL(writer->close());
}

/// @brief it should fail to open with a permanent error from the first Core open.
TEST(WriterTest, PropagatesOpenError) {
    auto bus = std::make_shared<Bus>();
    auto states = std::make_shared<control::States>();
    auto mock_factory = std::make_shared<::driver::pipeline::testutil::WriterFactory>(
        std::make_shared<x::sync::List<x::telem::Frame>>(),
        std::vector<x::errors::Error>{x::errors::VALIDATION}
    );
    WriterFactory factory(mock_factory, bus, states, 0);
    ASSERT_OCCURRED_AS_P(factory.open_writer({.channels = {1}}), x::errors::VALIDATION);
}

/// @brief it should open while the Core is unreachable.
TEST(WriterTest, OpensWhileTheCoreIsUnreachable) {
    auto bus = std::make_shared<Bus>();
    auto states = std::make_shared<control::States>();
    auto mock_factory = std::make_shared<::driver::pipeline::testutil::WriterFactory>(
        std::make_shared<x::sync::List<x::telem::Frame>>(),
        std::vector<x::errors::Error>(1000, freighter::UNREACHABLE)
    );
    WriterFactory factory(
        mock_factory,
        bus,
        states,
        0,
        {.breaker = {.base_interval = x::telem::MILLISECOND}}
    );
    auto writer = ASSERT_NIL_P(factory.open_writer({.channels = {1}}));
    ASSERT_NIL(writer->close());
}

TEST(WriterTest, InjectsGroupIntoWriterConfig) {
    auto bus = std::make_shared<Bus>();
    auto states = std::make_shared<control::States>();
    auto mock_factory = std::make_shared<::driver::pipeline::testutil::WriterFactory>();
    WriterFactory factory(mock_factory, bus, states, 77);
    auto writer = ASSERT_NIL_P(factory.open_writer({.channels = {1}}));
    ASSERT_EVENTUALLY_EQ(mock_factory->writer_opens.load(), 1);
    ASSERT_EQ(mock_factory->config.subject.group, 77);
}

TEST(WriterTest, DoesNotOverrideExistingGroup) {
    auto bus = std::make_shared<Bus>();
    auto states = std::make_shared<control::States>();
    auto mock_factory = std::make_shared<::driver::pipeline::testutil::WriterFactory>();
    WriterFactory factory(mock_factory, bus, states, 77);
    auto writer = ASSERT_NIL_P(factory.open_writer({
        .channels = {1},
        .subject = x::control::Subject{"w", "w", 99},
    }));
    ASSERT_EVENTUALLY_EQ(mock_factory->writer_opens.load(), 1);
    ASSERT_EQ(mock_factory->config.subject.group, 99);
}

TEST(WriterTest, DoesNotInjectGroupWhenZero) {
    auto bus = std::make_shared<Bus>();
    auto states = std::make_shared<control::States>();
    auto mock_factory = std::make_shared<::driver::pipeline::testutil::WriterFactory>();
    WriterFactory factory(mock_factory, bus, states, 0);
    auto writer = ASSERT_NIL_P(factory.open_writer({.channels = {1}}));
    ASSERT_EVENTUALLY_EQ(mock_factory->writer_opens.load(), 1);
    ASSERT_EQ(mock_factory->config.subject.group, 0);
}

TEST(WriterTest, SetAuthorityIncreaseUpdatesMirror) {
    auto bus = std::make_shared<Bus>();
    auto states = std::make_shared<control::States>();
    auto mock_factory = std::make_shared<::driver::pipeline::testutil::WriterFactory>();
    WriterFactory factory(mock_factory, bus, states, 0);
    auto writer = ASSERT_NIL_P(factory.open_writer({
        .channels = {1, 2},
        .subject = x::control::Subject{"arc", "arc-1"},
    }));
    ASSERT_NIL(writer->set_authority({.keys = {1, 2}, .authorities = {200}}));
    ASSERT_TRUE(states->is_authorized(1, {"arc", "arc-1"}));
    ASSERT_TRUE(states->is_authorized(2, {"arc", "arc-1"}));
}

TEST(WriterTest, SetAuthorityDecreaseDoesNotUpdateMirror) {
    auto bus = std::make_shared<Bus>();
    auto states = std::make_shared<control::States>();
    const x::control::Subject arc{"arc", "arc-1"};
    const x::control::Subject op{"operator", "op-1"};
    states->apply_increase(op, 1, 200);
    auto mock_factory = std::make_shared<::driver::pipeline::testutil::WriterFactory>();
    WriterFactory factory(mock_factory, bus, states, 0);
    auto writer = ASSERT_NIL_P(factory.open_writer({
        .channels = {1},
        .subject = arc,
    }));
    ASSERT_NIL(writer->set_authority({.keys = {1}, .authorities = {100}}));
    ASSERT_TRUE(states->is_authorized(1, op));
    ASSERT_FALSE(states->is_authorized(1, arc));
}

TEST(WriterTest, SetAuthorityGlobalKeysExpandsToWriterChannels) {
    auto bus = std::make_shared<Bus>();
    auto states = std::make_shared<control::States>();
    auto mock_factory = std::make_shared<::driver::pipeline::testutil::WriterFactory>();
    WriterFactory factory(mock_factory, bus, states, 0);
    auto writer = ASSERT_NIL_P(factory.open_writer({
        .channels = {1, 2, 3},
        .subject = x::control::Subject{"arc", "arc-1"},
    }));
    ASSERT_NIL(writer->set_authority({.keys = {}, .authorities = {255}}));
    ASSERT_TRUE(states->is_authorized(1, {"arc", "arc-1"}));
    ASSERT_TRUE(states->is_authorized(2, {"arc", "arc-1"}));
    ASSERT_TRUE(states->is_authorized(3, {"arc", "arc-1"}));
}

TEST(WriterTest, SetAuthorityIncreaseEndToEnd) {
    auto bus = std::make_shared<Bus>();
    auto states = std::make_shared<control::States>();
    const x::control::Subject hotfire{"hotfire", "hf-1"};
    const x::control::Subject abort_sub{"abort", "abort-1"};

    states->apply_increase(hotfire, 1, 200);

    auto mock_writer_factory = std::make_shared<
        ::driver::pipeline::testutil::WriterFactory>();
    WriterFactory writer_factory(mock_writer_factory, bus, states, 0);
    auto writer = ASSERT_NIL_P(writer_factory.open_writer({
        .channels = {1},
        .subject = abort_sub,
    }));

    x::telem::Frame hotfire_frame;
    hotfire_frame.emplace(1, x::telem::Series(static_cast<float>(1.0)));
    auto filtered_before = states->filter(hotfire_frame, hotfire);
    ASSERT_EQ(filtered_before.size(), 1);

    ASSERT_NIL(writer->set_authority({.keys = {1}, .authorities = {255}}));

    auto filtered_after = states->filter(hotfire_frame, hotfire);
    ASSERT_TRUE(filtered_after.empty());

    x::telem::Frame abort_frame;
    abort_frame.emplace(1, x::telem::Series(static_cast<float>(0.0)));
    auto abort_filtered = states->filter(abort_frame, abort_sub);
    ASSERT_EQ(abort_filtered.size(), 1);
}

TEST(WriterTest, WriteFiltersUnauthorizedChannelsFromBus) {
    auto bus = std::make_shared<Bus>();
    auto states = std::make_shared<control::States>();
    const x::control::Subject arc{"arc", "arc-1"};
    const x::control::Subject other{"other", "other-1"};
    states->apply_increase(other, 1, 200);
    auto sub = bus->subscribe({1, 2});
    auto mock_factory = std::make_shared<::driver::pipeline::testutil::WriterFactory>();
    WriterFactory factory(mock_factory, bus, states, 0);
    auto writer = ASSERT_NIL_P(factory.open_writer({
        .channels = {1, 2},
        .subject = arc,
    }));
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(1.0)));
    frame.emplace(2, x::telem::Series(static_cast<float>(2.0)));
    ASSERT_NIL(writer->write(frame));
    ASSERT_EVENTUALLY_EQ(mock_factory->writes->size(), 1);
    ASSERT_EQ(mock_factory->writes->at(0).size(), 2);
    x::telem::Frame received;
    ASSERT_TRUE(sub->try_pop(received));
    ASSERT_EQ(received.size(), 1);
}

TEST(WriterTest, WritePublishesNothingToBusWhenFullyUnauthorized) {
    auto bus = std::make_shared<Bus>();
    auto states = std::make_shared<control::States>();
    const x::control::Subject arc{"arc", "arc-1"};
    const x::control::Subject other{"other", "other-1"};
    states->apply_increase(other, 1, 200);
    auto sub = bus->subscribe({1});
    auto mock_factory = std::make_shared<::driver::pipeline::testutil::WriterFactory>();
    WriterFactory factory(mock_factory, bus, states, 0);
    auto writer = ASSERT_NIL_P(factory.open_writer({
        .channels = {1},
        .subject = arc,
    }));
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(1.0)));
    ASSERT_NIL(writer->write(frame));
    ASSERT_EVENTUALLY_EQ(mock_factory->writes->size(), 1);
    x::telem::Frame received;
    ASSERT_FALSE(sub->try_pop(received));
}

TEST(WriterTest, WriteStillPublishesWhenNoAuthorityState) {
    auto bus = std::make_shared<Bus>();
    auto states = std::make_shared<control::States>();
    auto sub = bus->subscribe({1});
    auto mock_factory = std::make_shared<::driver::pipeline::testutil::WriterFactory>();
    WriterFactory factory(mock_factory, bus, states, 0);
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
    auto bus = std::make_shared<Bus>();
    auto states = std::make_shared<control::States>();
    const x::control::Subject hotfire{"hotfire", "hf-1"};
    const x::control::Subject abort_sub{"abort", "abort-1"};

    auto sub = bus->subscribe({1});
    auto mock_factory = std::make_shared<::driver::pipeline::testutil::WriterFactory>();
    WriterFactory factory(mock_factory, bus, states, 0);

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

/// @brief it should fail a later write with a permanent Core write error, after the
/// bus already has the frame.
TEST(WriterTest, WriteErrorPropagatesFromCore) {
    auto bus = std::make_shared<Bus>();
    auto states = std::make_shared<control::States>();
    auto sub = bus->subscribe({1});
    auto mock_factory = std::make_shared<::driver::pipeline::testutil::WriterFactory>(
        std::make_shared<x::sync::List<x::telem::Frame>>(),
        std::vector<x::errors::Error>{},
        std::vector<x::errors::Error>{},
        std::vector<int>{0}
    );
    WriterFactory factory(mock_factory, bus, states, 0);
    auto writer = ASSERT_NIL_P(factory.open_writer({.channels = {1}}));
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(42.0)));
    ASSERT_NIL(writer->write(frame));
    x::telem::Frame received;
    ASSERT_TRUE(sub->try_pop(received));
    ASSERT_EQ(received.size(), 1);
    ASSERT_EVENTUALLY_TRUE(writer->write(frame).matches(x::errors::VALIDATION));
    ASSERT_OCCURRED_AS(writer->close(), x::errors::VALIDATION);
}

/// @brief it should not report an unreachable Core when it closes.
TEST(WriterTest, CloseIgnoresAnUnreachableCore) {
    auto bus = std::make_shared<Bus>();
    auto states = std::make_shared<control::States>();
    auto mock_factory = std::make_shared<::driver::pipeline::testutil::WriterFactory>(
        std::make_shared<x::sync::List<x::telem::Frame>>(),
        std::vector<x::errors::Error>{},
        std::vector<x::errors::Error>{freighter::UNREACHABLE}
    );
    WriterFactory factory(mock_factory, bus, states, 0);
    auto writer = ASSERT_NIL_P(factory.open_writer({.channels = {1}}));
    ASSERT_EVENTUALLY_EQ(mock_factory->writer_opens.load(), 1);
    ASSERT_NIL(writer->close());
}

/// @brief it should reopen the Core writer after an unreachable Core fails an
/// authority change, and send the change again.
TEST(WriterTest, SetAuthorityRetriesAfterAnUnreachableCore) {
    auto bus = std::make_shared<Bus>();
    auto states = std::make_shared<control::States>();
    auto mock_factory = std::make_shared<::driver::pipeline::testutil::WriterFactory>(
        std::make_shared<x::sync::List<x::telem::Frame>>(),
        std::vector<x::errors::Error>{},
        std::vector<x::errors::Error>{},
        std::vector<int>{},
        std::vector<x::errors::Error>{freighter::UNREACHABLE}
    );
    WriterFactory factory(
        mock_factory,
        bus,
        states,
        0,
        {.breaker = {.base_interval = x::telem::MICROSECOND}}
    );
    auto writer = ASSERT_NIL_P(factory.open_writer({
        .channels = {1},
        .subject = x::control::Subject{"arc", "arc-1"},
    }));
    ASSERT_NIL(writer->set_authority({.keys = {1}, .authorities = {200}}));
    ASSERT_TRUE(states->is_authorized(1, {"arc", "arc-1"}));
    ASSERT_EVENTUALLY_EQ(mock_factory->authority_changes->size(), 2);
    ASSERT_EQ(mock_factory->writer_opens.load(), 2);
    ASSERT_NIL(writer->close());
}

/// @brief it should reopen the Core writer with the authorities it last set.
TEST(WriterTest, ReopensWithTheAuthoritiesInEffect) {
    auto bus = std::make_shared<Bus>();
    auto states = std::make_shared<control::States>();
    auto mock_factory = std::make_shared<::driver::pipeline::testutil::WriterFactory>(
        std::make_shared<x::sync::List<x::telem::Frame>>(),
        std::vector<x::errors::Error>{},
        std::vector<x::errors::Error>{freighter::UNREACHABLE},
        std::vector<int>{0}
    );
    WriterFactory factory(
        mock_factory,
        bus,
        states,
        0,
        {.breaker = {.base_interval = x::telem::MICROSECOND}}
    );
    auto writer = ASSERT_NIL_P(factory.open_writer({.channels = {1}}));
    ASSERT_NIL(writer->set_authority({.keys = {1}, .authorities = {200}}));
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(42.0)));
    ASSERT_NIL(writer->write(frame));
    ASSERT_EVENTUALLY_EQ(mock_factory->writes->size(), 1);
    ASSERT_EQ(mock_factory->writer_opens.load(), 2);
    ASSERT_EQ(
        mock_factory->config.authorities,
        std::vector<x::control::Authority>{200}
    );
    ASSERT_NIL(writer->close());
}

/// @brief it should keep accepting writes while the Core is unreachable, and deliver
/// them once it returns.
TEST(WriterTest, WritesWhileTheCoreIsUnreachable) {
    auto bus = std::make_shared<Bus>();
    auto states = std::make_shared<control::States>();
    auto sub = bus->subscribe({1});
    auto mock_factory = std::make_shared<::driver::pipeline::testutil::WriterFactory>(
        std::make_shared<x::sync::List<x::telem::Frame>>(),
        std::vector<x::errors::Error>{
            freighter::UNREACHABLE,
            freighter::UNREACHABLE,
            freighter::UNREACHABLE,
        }
    );
    WriterFactory factory(
        mock_factory,
        bus,
        states,
        0,
        {.breaker = {.base_interval = x::telem::MILLISECOND * 20}}
    );
    auto writer = ASSERT_NIL_P(factory.open_writer({.channels = {1}}));
    for (int i = 0; i < 3; i++) {
        x::telem::Frame frame;
        frame.emplace(1, x::telem::Series(static_cast<float>(i)));
        ASSERT_NIL(writer->write(frame));
        x::telem::Frame received;
        ASSERT_TRUE(sub->try_pop(received));
    }
    ASSERT_EVENTUALLY_EQ(mock_factory->writes->size(), 3);
    ASSERT_EQ(mock_factory->writer_opens.load(), 4);
    ASSERT_NIL(writer->close());
}

/// @brief set_authority should return a validation error when the authorities size does
/// not match the keys size and is not 1.
TEST(WriterTest, SetAuthorityRejectsMismatchedSizes) {
    auto bus = std::make_shared<Bus>();
    auto states = std::make_shared<control::States>();
    auto mock_factory = std::make_shared<::driver::pipeline::testutil::WriterFactory>();
    WriterFactory factory(mock_factory, bus, states, 0);
    auto writer = ASSERT_NIL_P(factory.open_writer({.channels = {1, 2, 3}}));
    ASSERT_OCCURRED_AS(
        writer->set_authority({.keys = {1, 2, 3}, .authorities = {100, 200}}),
        x::errors::VALIDATION
    );
}

/// @brief set_authority should accept a single authority broadcast to all keys.
TEST(WriterTest, SetAuthoritySingleAuthorityBroadcast) {
    auto bus = std::make_shared<Bus>();
    auto states = std::make_shared<control::States>();
    auto mock_factory = std::make_shared<::driver::pipeline::testutil::WriterFactory>();
    WriterFactory factory(mock_factory, bus, states, 0);
    auto writer = ASSERT_NIL_P(factory.open_writer({.channels = {1, 2, 3}}));
    ASSERT_NIL(writer->set_authority({.keys = {1, 2, 3}, .authorities = {200}}));
}

/// @brief set_authority should accept per-key authorities with matching sizes.
TEST(WriterTest, SetAuthorityMatchingSizes) {
    auto bus = std::make_shared<Bus>();
    auto states = std::make_shared<control::States>();
    auto mock_factory = std::make_shared<::driver::pipeline::testutil::WriterFactory>();
    WriterFactory factory(mock_factory, bus, states, 0);
    auto writer = ASSERT_NIL_P(factory.open_writer({.channels = {1, 2}}));
    ASSERT_NIL(writer->set_authority({.keys = {1, 2}, .authorities = {100, 200}}));
}
}
