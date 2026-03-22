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

TEST(BusTest, DeliveredFrameContainsOnlySubscribedChannels) {
    Bus bus;
    auto sub = bus.subscribe({1, 3});
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(10.0)));
    frame.emplace(2, x::telem::Series(static_cast<float>(20.0)));
    frame.emplace(3, x::telem::Series(static_cast<float>(30.0)));
    bus.publish(frame);
    x::telem::Frame received;
    ASSERT_TRUE(sub->try_pop(received));
    ASSERT_EQ(received.size(), 2);
    ASSERT_EQ(received.at<float>(1, 0), 10.0f);
    ASSERT_EQ(received.at<float>(3, 0), 30.0f);
}

TEST(BusTest, DeliveredFramePassthroughWhenAllChannelsMatch) {
    Bus bus;
    auto sub = bus.subscribe({1, 2});
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(10.0)));
    frame.emplace(2, x::telem::Series(static_cast<float>(20.0)));
    bus.publish(frame);
    x::telem::Frame received;
    ASSERT_TRUE(sub->try_pop(received));
    ASSERT_EQ(received.size(), 2);
}

TEST(BusTest, DeliveredFrameFiltersExtraChannelsMultipleSubscribers) {
    Bus bus;
    auto sub1 = bus.subscribe({1});
    auto sub2 = bus.subscribe({2});
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(10.0)));
    frame.emplace(2, x::telem::Series(static_cast<float>(20.0)));
    frame.emplace(3, x::telem::Series(static_cast<float>(30.0)));
    bus.publish(frame);
    x::telem::Frame r1, r2;
    ASSERT_TRUE(sub1->try_pop(r1));
    ASSERT_EQ(r1.size(), 1);
    ASSERT_EQ(r1.at<float>(1, 0), 10.0f);
    ASSERT_TRUE(sub2->try_pop(r2));
    ASSERT_EQ(r2.size(), 1);
    ASSERT_EQ(r2.at<float>(2, 0), 20.0f);
}

TEST(BusTest, PublishAssignsIncreasingAlignmentPerChannel) {
    Bus bus;
    bus.register_channels({1, 2});
    auto sub = bus.subscribe({1, 2});
    for (int i = 0; i < 3; i++) {
        x::telem::Frame frame;
        frame.emplace(1, x::telem::Series(static_cast<float>(i)));
        frame.emplace(2, x::telem::Series(static_cast<float>(i)));
        bus.publish(frame);
    }
    uint64_t prev_1 = 0, prev_2 = 0;
    bool first = true;
    for (int i = 0; i < 3; i++) {
        x::telem::Frame r;
        ASSERT_TRUE(sub->try_pop(r));
        ASSERT_EQ(r.size(), 2);
        auto a1 = r.series->at(0).alignment.uint64();
        auto a2 = r.series->at(1).alignment.uint64();
        if (!first) {
            EXPECT_GT(a1, prev_1) << "channel 1 alignment did not increase at i=" << i;
            EXPECT_GT(a2, prev_2) << "channel 2 alignment did not increase at i=" << i;
        }
        prev_1 = a1;
        prev_2 = a2;
        first = false;
    }
}

TEST(BusTest, PublishAlignmentAdvancesBySeriesSize) {
    Bus bus;
    bus.register_channels({1});
    auto sub = bus.subscribe({1});
    auto multi = x::telem::Series(x::telem::FLOAT32_T, 3);
    multi.write(1.0f);
    multi.write(2.0f);
    multi.write(3.0f);
    x::telem::Frame frame;
    frame.emplace(1, std::move(multi));
    bus.publish(frame);

    x::telem::Frame r1;
    ASSERT_TRUE(sub->try_pop(r1));
    auto first_alignment = r1.series->at(0).alignment.uint64();

    x::telem::Frame frame2;
    frame2.emplace(1, x::telem::Series(static_cast<float>(4.0)));
    bus.publish(frame2);

    x::telem::Frame r2;
    ASSERT_TRUE(sub->try_pop(r2));
    EXPECT_GE(r2.series->at(0).alignment.uint64(), first_alignment + 3);
}

TEST(BusTest, PublishAlignmentUnregisteredChannelLeftAtZero) {
    Bus bus;
    auto sub = bus.subscribe({1});
    x::telem::Frame frame;
    frame.emplace(1, x::telem::Series(static_cast<float>(1.0)));
    bus.publish(frame);
    x::telem::Frame r;
    ASSERT_TRUE(sub->try_pop(r));
    EXPECT_EQ(r.series->at(0).alignment.uint64(), 0);
}

TEST(BusTest, ConcurrentPublishAndSubscribeLifecycle) {
    Bus bus;
    bus.register_channels({1, 2, 3});
    auto persistent_sub = bus.subscribe({1, 2, 3});
    constexpr int num_publishers = 4;
    constexpr int num_subscribers = 4;
    constexpr int frames_per_publisher = 50;
    std::vector<std::thread> threads;
    for (int p = 0; p < num_publishers; p++) {
        threads.emplace_back([&bus] {
            for (int i = 0; i < frames_per_publisher; i++) {
                x::telem::Frame frame;
                frame.emplace(1, x::telem::Series(static_cast<float>(i)));
                frame.emplace(2, x::telem::Series(static_cast<float>(i)));
                bus.publish(frame);
            }
        });
    }
    for (int s = 0; s < num_subscribers; s++) {
        threads.emplace_back([&bus] {
            auto sub = bus.subscribe({1});
            for (int i = 0; i < 10; i++) {
                x::telem::Frame r;
                sub->try_pop(r);
                std::this_thread::sleep_for(std::chrono::microseconds(100));
            }
            bus.unsubscribe(*sub);
        });
    }
    for (auto &t: threads)
        t.join();
    int received = 0;
    x::telem::Frame r;
    while (persistent_sub->try_pop(r))
        received++;
    ASSERT_EQ(received, num_publishers * frames_per_publisher);
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

TEST(SubscriptionTest, DropsOldestWhenFull) {
    Subscription sub({1}, 3);
    for (int i = 0; i < 5; i++) {
        x::telem::Frame frame;
        frame.emplace(1, x::telem::Series(static_cast<float>(i)));
        sub.push(std::move(frame));
    }
    x::telem::Frame r;
    ASSERT_TRUE(sub.try_pop(r));
    ASSERT_EQ(r.at<float>(1, 0), 2.0f);
    ASSERT_TRUE(sub.try_pop(r));
    ASSERT_EQ(r.at<float>(1, 0), 3.0f);
    ASSERT_TRUE(sub.try_pop(r));
    ASSERT_EQ(r.at<float>(1, 0), 4.0f);
    ASSERT_FALSE(sub.try_pop(r));
}

TEST(SubscriptionTest, QueueStaysAtCapacity) {
    Subscription sub({1}, 3);
    for (int i = 0; i < 100; i++) {
        x::telem::Frame frame;
        frame.emplace(1, x::telem::Series(static_cast<float>(i)));
        sub.push(std::move(frame));
    }
    int count = 0;
    x::telem::Frame r;
    while (sub.try_pop(r))
        count++;
    ASSERT_EQ(count, 3);
}
}
