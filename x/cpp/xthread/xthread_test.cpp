// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"
#include "x/cpp/xthread/xthread.h"

#include <thread>

TEST(XThreadTest, SetName) {
    std::thread t([]() {
        xthread::set_name("test-thread");
        // Thread name is set - verification requires platform-specific APIs
        // or manual inspection in a debugger
    });
    t.join();
}

TEST(XThreadTest, SetNameCurrentThread) {
    xthread::set_name("main-test");
    // Verify no crash/error when setting name on main thread
    SUCCEED();
}