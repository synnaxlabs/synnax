// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <thread>

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/can/loopback/loopback.h"

namespace driver::can::loopback {
namespace {
synnax::can::Properties
props(const std::string &channel, const bool listen_only = false) {
    synnax::can::Properties p;
    p.backend = BACKEND;
    p.channel = channel;
    p.fd = true;
    p.listen_only = listen_only;
    return p;
}

const auto SHORT = x::telem::MILLISECOND * 10;
}

TEST(Loopback, DeliversAFrameToTheOtherBusesOnTheChannel) {
    Backend backend;
    auto a = ASSERT_NIL_P(backend.open(props("can0")));
    auto b = ASSERT_NIL_P(backend.open(props("can0")));
    Frame sent{.id = 0x42, .fd = true, .length = 12};
    sent.data[11] = 0xAB;
    ASSERT_NIL(a->send(sent));
    Frame got;
    ASSERT_TRUE(ASSERT_NIL_P(b->receive(got, x::telem::SECOND)));
    EXPECT_EQ(got.id, 0x42);
    EXPECT_TRUE(got.fd);
    EXPECT_EQ(got.length, 12);
    EXPECT_EQ(got.data[11], 0xAB);
    EXPECT_EQ(got.clock, Clock::HOST);
}

TEST(Loopback, NeverDeliversAFrameToTheBusThatSentIt) {
    Backend backend;
    auto a = ASSERT_NIL_P(backend.open(props("can0")));
    ASSERT_NIL(a->send(Frame{.id = 1}));
    Frame got;
    EXPECT_FALSE(ASSERT_NIL_P(a->receive(got, SHORT)));
}

TEST(Loopback, KeepsChannelsApart) {
    Backend backend;
    auto a = ASSERT_NIL_P(backend.open(props("can0")));
    auto b = ASSERT_NIL_P(backend.open(props("can1")));
    ASSERT_NIL(a->send(Frame{.id = 1}));
    Frame got;
    EXPECT_FALSE(ASSERT_NIL_P(b->receive(got, SHORT)));
}

TEST(Loopback, WakesABlockedReceiver) {
    Backend backend;
    auto a = ASSERT_NIL_P(backend.open(props("can0")));
    auto b = ASSERT_NIL_P(backend.open(props("can0")));
    std::thread sender([&] {
        std::this_thread::sleep_for(std::chrono::milliseconds(20));
        ASSERT_NIL(a->send(Frame{.id = 7}));
    });
    Frame got;
    ASSERT_TRUE(ASSERT_NIL_P(b->receive(got, x::telem::SECOND * 5)));
    EXPECT_EQ(got.id, 7);
    sender.join();
}

TEST(Loopback, RefusesToSendOnAListenOnlyBus) {
    Backend backend;
    auto a = ASSERT_NIL_P(backend.open(props("can0", true)));
    const auto err = a->send(Frame{.id = 1});
    ASSERT_MATCHES(err, LISTEN_ONLY_ERROR);
    EXPECT_EQ(err.data, "channel can0 is listen only");
}

TEST(Loopback, StopsDeliveringToAClosedBus) {
    Backend backend;
    auto a = ASSERT_NIL_P(backend.open(props("can0")));
    auto b = ASSERT_NIL_P(backend.open(props("can0")));
    ASSERT_NIL(b->close());
    ASSERT_NIL(b->close());
    ASSERT_NIL(a->send(Frame{.id = 1}));
}

TEST(Loopback, ScansTheChannelsItWasGiven) {
    Backend backend({"can0", "can1"});
    const auto channels = ASSERT_NIL_P(backend.scan());
    ASSERT_EQ(channels.size(), 2);
    EXPECT_EQ(channels[0].backend, BACKEND);
    EXPECT_EQ(channels[1].name, "can1");
}
}
