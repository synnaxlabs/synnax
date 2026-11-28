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

#include "x/cpp/shutdown/shutdown.h"

namespace x::shutdown {
/// @brief it should listen to every source of shutdown signals by default.
TEST(ListenOptions, ListensToEverySourceByDefault) {
    constexpr ListenOptions options;
    ASSERT_FALSE(options.signals_ignored);
    ASSERT_FALSE(options.stdin_ignored);
}

/// @brief it should return on signal_shutdown when it ignores signals and stdin.
TEST(Listen, ReturnsOnSignalShutdownWhenSourcesIgnored) {
    std::thread signaler(signal_shutdown);
    listen({.signals_ignored = true, .stdin_ignored = true});
    signaler.join();
    ASSERT_TRUE(details::should_shutdown());
}
}
