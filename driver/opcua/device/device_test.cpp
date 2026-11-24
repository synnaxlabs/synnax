// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <string>

#include "gtest/gtest.h"

#include "client/cpp/testutil/pipe.h"
#include "client/cpp/testutil/testutil.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/errors/errors.h"
#include "driver/opcua/device/device.h"
#include "driver/opcua/testutil/testutil.h"

namespace driver::opcua::device {
namespace {
Connector new_connector(const synnax::Config &cfg, const std::string &key) {
    auto client = std::make_shared<synnax::Synnax>(cfg);
    return Connector(
        client,
        std::make_shared<connection::Pool>(testutil::create_store()),
        std::make_shared<secret::Reader>(client, secret::Config{}),
        key,
        "[opcua.test] "
    );
}

/// @returns the key of a new device whose server never answers.
std::string create_device() {
    auto client = new_test_client();
    const auto rack = ASSERT_NIL_P(client.racks.create("opcua_device_test_rack"));
    connection::Config conn;
    conn.endpoint = "opc.tcp://localhost:1";
    conn.security_mode = "None";
    conn.security_policy = "None";
    synnax::device::Device dev{
        .key = make_unique_channel_name("opcua_device_test"),
        .rack = rack.key,
        .location = conn.endpoint,
        .make = "opc",
        .name = "OPC UA Device Test",
        .properties = x::json::json::object(
            {{"connection", testutil::connection_json(conn)}}
        ),
    };
    EXPECT_NIL(client.devices.create(dev));
    return dev.key;
}
}

/// @brief it should return the Core error when no retrieve has read settings yet.
TEST(ConnectorTest, ConnectReturnsCoreErrorBeforeSettingsLoad) {
    auto cfg = test_client_config();
    cfg.port = 1;
    auto connector = new_connector(cfg, "missing-device");
    ASSERT_OCCURRED_AS_P(connector.connect(), freighter::UNREACHABLE);
}

/// @brief it should keep the last settings and connect with them when the Core is
/// unreachable, so the error comes from the device instead of the Core.
TEST(ConnectorTest, ConnectKeepsSettingsWhenCoreUnreachable) {
    const auto key = create_device();
    CorePipe pipe(test_client_config().port);
    auto cfg = test_client_config();
    cfg.host = "127.0.0.1";
    cfg.port = pipe.port;
    auto connector = new_connector(cfg, key);
    ASSERT_OCCURRED_AS_P(connector.start(), driver::errors::BASE_ERROR);
    pipe.pause();
    ASSERT_OCCURRED_AS_P(connector.connect(), driver::errors::BASE_ERROR);
}

/// @brief it should return the Core error when the Core rejects the read.
TEST(ConnectorTest, ConnectReturnsRejectedCoreError) {
    auto connector = new_connector(test_client_config(), "missing-device");
    ASSERT_OCCURRED_AS_P(connector.connect(), x::errors::NOT_FOUND);
}
}
