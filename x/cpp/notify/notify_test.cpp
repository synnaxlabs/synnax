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

#include "x/cpp/notify/notify.h"
#include "x/cpp/xtest/xtest.h"

namespace x::notify {
/// @brief it should create a notifier successfully.
TEST(NotifierTest, Create) {
    auto notifier = create();
    ASSERT_NE(notifier, nullptr);
}

/// @brief it should wake a waiting thread when signaled.
TEST(NotifierTest, SignalWait) {
    auto notifier = create();

    std::thread signaler([&]() {
        std::this_thread::sleep_for((10 * telem::MILLISECOND).chrono());
        notifier->signal();
    });

    EXPECT_TRUE(notifier->wait(telem::SECOND));

    signaler.join();
}

/// @brief it should return immediately when signaled before wait.
TEST(NotifierTest, SignalBeforeWait) {
    auto notifier = create();
    notifier->signal();
    EXPECT_TRUE(notifier->wait(telem::MILLISECOND * 100));
}

/// @brief it should return false when timeout expires without signal.
TEST(NotifierTest, TimeoutExpires) {
    auto notifier = notify::create();
    const auto sw = telem::Stopwatch();
    EXPECT_FALSE(notifier->wait(telem::MILLISECOND * 50));
    EXPECT_GE(sw.elapsed(), 40 * telem::MILLISECOND);
}

/// @brief it should return false on poll when not signaled.
TEST(NotifierTest, Poll) {
    auto notifier = create();
    EXPECT_FALSE(notifier->poll());
    notifier->signal();
    EXPECT_TRUE(notifier->poll());
    EXPECT_FALSE(notifier->poll());
}

/// @brief it should coalesce multiple signals into a single wake.
TEST(NotifierTest, MultipleSignalsCoalesce) {
    auto notifier = create();
    notifier->signal();
    notifier->signal();
    notifier->signal();
    EXPECT_TRUE(notifier->poll());
    EXPECT_FALSE(notifier->poll());
}

/// @brief it should return a valid fd on Linux/macOS or -1 on other platforms.
TEST(NotifierTest, FdAvailability) {
    auto notifier = create();
    const int fd = notifier->fd();
#if defined(__linux__) || defined(__APPLE__)
    EXPECT_GE(fd, 0);
#else
    EXPECT_EQ(fd, -1);
#endif
}

/// @brief it should handle producer-consumer signaling pattern.
TEST(NotifierTest, ProducerConsumerPattern) {
    auto notifier = create();
    constexpr int num_signals = 100;
    int received = 0;

    std::thread producer([&]() {
        for (int i = 0; i < num_signals; i++) {
            std::this_thread::sleep_for((100 * telem::MICROSECOND).chrono());
            notifier->signal();
        }
    });

    std::thread consumer([&]() {
        while (received < num_signals) {
            if (notifier->wait(telem::MILLISECOND * 100)) received++;
        }
    });

    producer.join();
    consumer.join();
    EXPECT_GE(received, 1);
}

/// @brief it should return immediately with zero timeout.
TEST(NotifierTest, ZeroTimeout) {
    auto notifier = notify::create();
    const auto sw = telem::Stopwatch();
    EXPECT_FALSE(notifier->wait(telem::TimeSpan(0)));
    EXPECT_LE(sw.elapsed(), 10 * telem::MILLISECOND);
}
}
