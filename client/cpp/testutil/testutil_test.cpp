// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <stdexcept>

#include "gmock/gmock.h"
#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/env/env.h"

class TestClientConfig : public ::testing::Test {
protected:
    void SetUp() override {
        x::env::unset("SYNNAX_HOST");
        x::env::unset("SYNNAX_PORT");
    }
};

/// @brief it should default to an insecure Core on localhost:9090.
TEST_F(TestClientConfig, testDefaultsToLocalhost) {
    const auto cfg = test_client_config();
    EXPECT_EQ(cfg.host, "localhost");
    EXPECT_EQ(cfg.port, 9090);
    EXPECT_FALSE(cfg.secure);
}

/// @brief it should read the host and port from the environment.
TEST_F(TestClientConfig, testReadsHostAndPortFromEnvironment) {
    x::env::set("SYNNAX_HOST", "core.local");
    x::env::set("SYNNAX_PORT", "9190");
    const auto cfg = test_client_config();
    EXPECT_EQ(cfg.host, "core.local");
    EXPECT_EQ(cfg.port, 9190);
}

/// @brief it should throw on a port that is not a number.
TEST_F(TestClientConfig, testThrowsOnPortThatIsNotANumber) {
    x::env::set("SYNNAX_PORT", "not_a_port");
    EXPECT_THAT(
        [] { static_cast<void>(test_client_config()); },
        ::testing::ThrowsMessage<std::runtime_error>(
            ::testing::HasSubstr("failed to convert SYNNAX_PORT")
        )
    );
}

/// @brief it should throw on a port that would wrap around to another port.
TEST_F(TestClientConfig, testThrowsOnPortOutOfRange) {
    x::env::set("SYNNAX_PORT", "74626");
    EXPECT_THAT(
        [] { static_cast<void>(test_client_config()); },
        ::testing::ThrowsMessage<std::runtime_error>(
            ::testing::HasSubstr("value is not an integer in range")
        )
    );
}
