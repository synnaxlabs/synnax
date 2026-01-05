// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstring>
#include <thread>

#include "gtest/gtest.h"

#include "x/cpp/xthread/xthread.h"

TEST(XThreadTest, SetAndGetName) {
    std::thread t([]() {
        xthread::set_name("test-thread");
        char buf[xthread::MAX_NAME_LEN];
        ASSERT_TRUE(xthread::get_name(buf, sizeof(buf)));
        EXPECT_STREQ(buf, "test-thread");
    });
    t.join();
}

TEST(XThreadTest, SetAndGetNameCurrentThread) {
    xthread::set_name("main-test");
    char buf[xthread::MAX_NAME_LEN];
    ASSERT_TRUE(xthread::get_name(buf, sizeof(buf)));
    EXPECT_STREQ(buf, "main-test");
}

TEST(XThreadTest, NameTruncation) {
    // Thread names are limited to 15-16 characters on most platforms.
    // On Linux/macOS, names longer than 15 characters are truncated.
    std::thread t([]() {
        xthread::set_name("this-is-a-very-long-thread-name");
        char buf[xthread::MAX_NAME_LEN];
        ASSERT_TRUE(xthread::get_name(buf, sizeof(buf)));
        // Should be truncated to 15 chars on POSIX systems
        EXPECT_LE(strlen(buf), 15u);
    });
    t.join();
}

TEST(XThreadTest, EmptyName) {
    std::thread t([]() {
        xthread::set_name("");
        char buf[xthread::MAX_NAME_LEN];
        ASSERT_TRUE(xthread::get_name(buf, sizeof(buf)));
        EXPECT_STREQ(buf, "");
    });
    t.join();
}

TEST(XThreadTest, MultipleThreadsWithDifferentNames) {
    std::thread t1([]() {
        xthread::set_name("thread-one");
        char buf[xthread::MAX_NAME_LEN];
        ASSERT_TRUE(xthread::get_name(buf, sizeof(buf)));
        EXPECT_STREQ(buf, "thread-one");
    });

    std::thread t2([]() {
        xthread::set_name("thread-two");
        char buf[xthread::MAX_NAME_LEN];
        ASSERT_TRUE(xthread::get_name(buf, sizeof(buf)));
        EXPECT_STREQ(buf, "thread-two");
    });

    t1.join();
    t2.join();
}
