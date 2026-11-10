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

#include "x/cpp/testutil/testutil.h"

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

TEST(Loopback, RefusesASecondOpenOfAnExclusiveChannelUntilTheFirstCloses) {
    Backend backend({}, true);
    auto first = ASSERT_NIL_P(backend.open(props("can0")));
    auto [second, err] = backend.open(props("can0"));
    ASSERT_MATCHES(err, UNREACHABLE_ERROR);
    EXPECT_EQ(err.data, "channel can0 is in use");
    ASSERT_NIL(first->close());
    auto third = ASSERT_NIL_P(backend.open(props("can0")));
    EXPECT_EQ(backend.attempts("can0"), 3);
    EXPECT_EQ(backend.opens("can0"), 2);
    EXPECT_EQ(backend.in_use("can0"), 1);
}

TEST(Loopback, NeverCountsANodeAsAnOpenOfTheChannel) {
    Backend backend({}, true);
    auto node = backend.node(props("can0"));
    auto bus = ASSERT_NIL_P(backend.open(props("can0")));
    ASSERT_NIL(node->send(Frame{.id = 3}));
    Frame got;
    ASSERT_TRUE(ASSERT_NIL_P(bus->receive(got, x::telem::SECOND)));
    EXPECT_EQ(got.id, 3);
}

TEST(Loopback, FailsTheBusesOfAnUnpluggedChannelUntilTheyReopen) {
    Backend backend;
    auto node = backend.node(props("can0"));
    auto bus = ASSERT_NIL_P(backend.open(props("can0")));
    backend.unplug("can0");
    Frame got;
    ASSERT_OCCURRED_AS_P(bus->receive(got, x::telem::SECOND), UNREACHABLE_ERROR);
    const auto err = bus->send(Frame{.id = 1});
    ASSERT_MATCHES(err, UNREACHABLE_ERROR);
    EXPECT_EQ(err.data, "channel can0 was unplugged");
    auto [none, open_err] = backend.open(props("can0"));
    ASSERT_MATCHES(open_err, UNREACHABLE_ERROR);
    EXPECT_EQ(open_err.data, "channel can0 is not attached");
    backend.plug("can0");
    ASSERT_OCCURRED_AS(bus->send(Frame{.id = 1}), UNREACHABLE_ERROR);
    auto reopened = ASSERT_NIL_P(backend.open(props("can0")));
    ASSERT_NIL(node->send(Frame{.id = 2}));
    ASSERT_TRUE(ASSERT_NIL_P(reopened->receive(got, x::telem::SECOND)));
    EXPECT_EQ(got.id, 2);
}

TEST(Loopback, WakesAReceiverWhenTheChannelIsUnplugged) {
    Backend backend;
    auto bus = ASSERT_NIL_P(backend.open(props("can0")));
    std::thread unplug([&] {
        std::this_thread::sleep_for(std::chrono::milliseconds(20));
        backend.unplug("can0");
    });
    Frame got;
    const auto start = x::telem::TimeStamp::now();
    ASSERT_OCCURRED_AS_P(bus->receive(got, x::telem::SECOND * 5), UNREACHABLE_ERROR);
    EXPECT_LT(x::telem::TimeStamp::now() - start, x::telem::SECOND * 5);
    unplug.join();
}

TEST(Loopback, FailsEachKindOfCallOnDemand) {
    Backend backend({"can0"});
    const x::errors::Error fault(CRITICAL_HARDWARE_ERROR, "fault");
    backend.fail(Call::SCAN, fault);
    ASSERT_OCCURRED_AS_P(backend.scan(), CRITICAL_HARDWARE_ERROR);
    EXPECT_EQ(ASSERT_NIL_P(backend.scan()).size(), 1);
    backend.fail(Call::OPEN, fault);
    ASSERT_OCCURRED_AS_P(backend.open(props("can0")), CRITICAL_HARDWARE_ERROR);
    auto bus = ASSERT_NIL_P(backend.open(props("can0")));
    backend.fail(Call::TRANSMIT, fault, 2);
    ASSERT_OCCURRED_AS(bus->send(Frame{.id = 1}), CRITICAL_HARDWARE_ERROR);
    ASSERT_OCCURRED_AS(bus->send(Frame{.id = 1}), CRITICAL_HARDWARE_ERROR);
    ASSERT_NIL(bus->send(Frame{.id = 1}));
    backend.fail(Call::RECEIVE, fault);
    Frame got;
    ASSERT_OCCURRED_AS_P(bus->receive(got, SHORT), CRITICAL_HARDWARE_ERROR);
    EXPECT_FALSE(ASSERT_NIL_P(bus->receive(got, SHORT)));
    backend.fail(Call::CLOSE, fault);
    ASSERT_OCCURRED_AS(bus->close(), CRITICAL_HARDWARE_ERROR);
    EXPECT_EQ(backend.in_use("can0"), 0);
}

TEST(Loopback, ScansTheChannelsItWasGiven) {
    Backend backend({"can0", "can1"});
    const auto channels = ASSERT_NIL_P(backend.scan());
    ASSERT_EQ(channels.size(), 2);
    EXPECT_EQ(channels[0].backend, BACKEND);
    EXPECT_EQ(channels[1].name, "can1");
}

TEST(Loopback, LeavesAnUnpluggedChannelOutOfAScan) {
    Backend backend({"can0", "can1"});
    backend.unplug("can0");
    const auto channels = ASSERT_NIL_P(backend.scan());
    ASSERT_EQ(channels.size(), 1);
    EXPECT_EQ(channels[0].name, "can1");
}
}
