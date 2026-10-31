// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>

#include "gtest/gtest.h"

#include "driver/opcua/testutil/server.h"

namespace driver::opcua::testutil {
namespace {
ServerConfig config() {
    auto cfg = ServerConfig::create_default();
    cfg.port = 4880;
    return cfg;
}
}

/// @brief it should stop without waiting for an event loop pass to time out.
TEST(Server, StopsPromptly) {
    Server server(config());
    server.start();
    ASSERT_TRUE(server.wait_until_ready());
    const auto start = std::chrono::steady_clock::now();
    server.stop();
    EXPECT_LT(std::chrono::steady_clock::now() - start, std::chrono::milliseconds(400));
}

/// @brief it should start again after a stop.
TEST(Server, RestartsAfterStop) {
    Server server(config());
    server.start();
    ASSERT_TRUE(server.wait_until_ready());
    server.stop();
    server.start();
    ASSERT_TRUE(server.wait_until_ready());
    server.stop();
}
}
