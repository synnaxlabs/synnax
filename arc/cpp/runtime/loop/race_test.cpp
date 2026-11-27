// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <thread>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "arc/cpp/runtime/loop/loop.h"

namespace arc::runtime::loop {
/// @brief it should let one thread wake the loop while another starts it.
TEST(LoopRaceTest, WakeWhileStarting) {
    Config config;
    config.mode = ExecutionMode::EVENT_DRIVEN;
    config.interval = x::telem::TimeSpan(0);
    config.cpu_affinity = CPU_AFFINITY_NONE;
    const auto loop = create(config);
    std::atomic<bool> started{false};
    std::thread waker([&] {
        while (!started.load())
            loop->wake();
    });
    ASSERT_NIL(loop->start());
    started.store(true);
    waker.join();
    loop->wake();
}
}
