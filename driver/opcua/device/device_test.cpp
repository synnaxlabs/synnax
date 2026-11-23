// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/opcua/device/device.h"
#include "driver/opcua/testutil/testutil.h"

namespace driver::opcua::device {
namespace {
Connector new_connector(const synnax::Config &cfg) {
    auto client = std::make_shared<synnax::Synnax>(cfg);
    return Connector(
        client,
        std::make_shared<connection::Pool>(testutil::create_store()),
        std::make_shared<secret::Reader>(client, secret::Config{}),
        "missing-device",
        "[opcua.test] "
    );
}
}

/// @brief it should keep the last settings and connect with them when the Core is
/// unreachable.
TEST(ConnectorTest, ConnectKeepsSettingsWhenCoreUnreachable) {
    auto cfg = test_client_config();
    cfg.port = 1;
    auto connector = new_connector(cfg);
    const auto [conn, err] = connector.connect();
    ASSERT_TRUE(err);
    EXPECT_FALSE(err.matches(freighter::UNREACHABLE)) << err;
}

/// @brief it should return the Core error when the Core rejects the read.
TEST(ConnectorTest, ConnectReturnsRejectedCoreError) {
    auto connector = new_connector(test_client_config());
    ASSERT_OCCURRED_AS_P(connector.connect(), x::errors::NOT_FOUND);
}
}
