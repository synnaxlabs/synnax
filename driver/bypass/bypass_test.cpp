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

#include "x/cpp/telem/series.h"

#include "driver/bypass/bypass.h"

namespace driver::bypass {
TEST(BusTest, PublishNoSubscribers) {
    Bus bus;
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(1.0)));
    bus.publish(frame);
}

TEST(BusTest, SingleSubscriber) {
    Bus bus;
    auto sub = bus.subscribe({1, 2});
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(42.0)));
    bus.publish(frame);
    x::telem::Frame received;
    ASSERT_TRUE(sub->try_pop(received));
    ASSERT_EQ(received.size(), 1);
}

TEST(BusTest, MultipleSubscribers) {
    Bus bus;
    auto sub1 = bus.subscribe({1});
    auto sub2 = bus.subscribe({1});
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(1.0)));
    bus.publish(frame);
    x::telem::Frame r1, r2;
    ASSERT_TRUE(sub1->try_pop(r1));
    ASSERT_TRUE(sub2->try_pop(r2));
}

TEST(BusTest, SubscriberKeyFiltering) {
    Bus bus;
    auto sub = bus.subscribe({2});
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(1.0)));
    bus.publish(frame);
    x::telem::Frame received;
    ASSERT_FALSE(sub->try_pop(received));
}

TEST(BusTest, Unsubscribe) {
    Bus bus;
    auto sub = bus.subscribe({1});
    bus.unsubscribe(*sub);
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(1.0)));
    bus.publish(frame);
    x::telem::Frame received;
    ASSERT_FALSE(sub->try_pop(received));
}

TEST(BusTest, DestroyedSubscriptionExpires) {
    Bus bus;
    { auto sub = bus.subscribe({1}); }
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(1.0)));
    bus.publish(frame);
}

TEST(BusTest, DestroyedSubscriptionDoesNotReceive) {
    Bus bus;
    auto sub1 = bus.subscribe({1});
    auto sub2 = bus.subscribe({1});
    sub1.reset();
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(1.0)));
    bus.publish(frame);
    x::telem::Frame received;
    ASSERT_TRUE(sub2->try_pop(received));
    ASSERT_EQ(received.size(), 1);
}

TEST(BusTest, DeepCopyIsolation) {
    Bus bus;
    auto sub = bus.subscribe({1});
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(42.0)));
    bus.publish(frame);
    frame.clear();
    x::telem::Frame received;
    ASSERT_TRUE(sub->try_pop(received));
    ASSERT_EQ(received.size(), 1);
}

TEST(BusTest, MultipleFramesQueue) {
    Bus bus;
    auto sub = bus.subscribe({1});
    for (int i = 0; i < 3; i++) {
        x::telem::Frame frame;
        frame.emplace(1, x::telem::Series(static_cast<float>(i)));
        bus.publish(frame);
    }
    x::telem::Frame r;
    ASSERT_TRUE(sub->try_pop(r));
    ASSERT_TRUE(sub->try_pop(r));
    ASSERT_TRUE(sub->try_pop(r));
    ASSERT_FALSE(sub->try_pop(r));
}

TEST(BusTest, BlockingPop) {
    Bus bus;
    auto sub = bus.subscribe({1});
    std::thread publisher([&bus] {
        std::this_thread::sleep_for(std::chrono::milliseconds(10));
        x::telem::Frame frame;
        frame.emplace(1, x::telem::Series(static_cast<float>(99.0)));
        bus.publish(frame);
    });
    x::telem::Frame received;
    ASSERT_TRUE(sub->pop(received));
    ASSERT_EQ(received.size(), 1);
    publisher.join();
}

TEST(BusTest, CloseUnblocksPop) {
    Bus bus;
    auto sub = bus.subscribe({1});
    std::thread closer([&sub] {
        std::this_thread::sleep_for(std::chrono::milliseconds(10));
        sub->close();
    });
    x::telem::Frame received;
    ASSERT_FALSE(sub->pop(received));
    closer.join();
}

TEST(BusTest, SingleDeliveryPerPublish) {
    Bus bus;
    auto sub = bus.subscribe({1, 2});
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(1.0)));
    frame.emplace(2, x::telem::Series(static_cast<float>(2.0)));
    bus.publish(frame);
    x::telem::Frame received;
    ASSERT_TRUE(sub->try_pop(received));
    ASSERT_FALSE(sub->try_pop(received));
}

TEST(BusTest, ExpiredEntriesSweptOnPublish) {
    Bus bus;
    auto sub1 = bus.subscribe({1});
    auto sub2 = bus.subscribe({1});
    sub1.reset();
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(1.0)));
    bus.publish(frame);
    x::telem::Frame received;
    ASSERT_TRUE(sub2->try_pop(received));
    ASSERT_EQ(received.size(), 1);
    sub2.reset();
    bus.publish(frame);
}

TEST(SubscriptionTest, Empty) {
    Subscription sub({1});
    ASSERT_TRUE(sub.empty());
    sub.push(x::telem::Frame());
    ASSERT_FALSE(sub.empty());
    x::telem::Frame out;
    sub.try_pop(out);
    ASSERT_TRUE(sub.empty());
}
}
