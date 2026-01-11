// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"
#include "nlohmann/json.hpp"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/test/test.h"

#include "driver/opc/mock/server.h"
#include "driver/opc/opc.h"
#include "driver/opc/scan_task.h"
#include "driver/task/common/scan_task.h"

class TestScanTask : public ::testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> client;
    std::shared_ptr<driver::task::MockContext> ctx;
    std::shared_ptr<driver::opc::connection::Pool> conn_pool;
    std::unique_ptr<mock::Server> server;
    synnax::task::Task task;
    synnax::rack::Rack rack;

    void SetUp() override {
        client = std::make_shared<synnax::Synnax>(new_test_client());
        ctx = std::make_shared<driver::task::MockContext>(client);
        conn_pool = std::make_shared<driver::opc::connection::Pool>();

        rack = ASSERT_NIL_P(client->racks.create("opc_scan_task_test_rack"));

        task = synnax::task::Task{.name = "OPC UA Scan Task Test", .type = "opc_scan"};

        auto server_cfg = mock::ServerConfig::create_default();
        server = std::make_unique<mock::Server>(server_cfg);
        server->start();

        // Wait for server to be ready by attempting to connect
        driver::opc::connection::Config test_conn_cfg;
        test_conn_cfg.endpoint = "opc.tcp://localhost:4840";
        test_conn_cfg.security_mode = "None";
        test_conn_cfg.security_policy = "None";
        auto test_client = ASSERT_EVENTUALLY_NIL_P_WITH_TIMEOUT(
            driver::opc::connection::connect(test_conn_cfg, "test"),
            (5 * x::telem::SECOND).chrono(),
            (250 * x::telem::MILLISECOND).chrono()
        );
        UA_Client_disconnect(test_client.get());
    }
};

/// @brief it should browse and return OPC UA server nodes.
TEST_F(TestScanTask, testBasicScan) {
    const auto cfg = driver::opc::ScanTaskConfig{};
    auto scan_task = std::make_unique<driver::task::common::ScanTask>(
        std::make_unique<driver::opc::Scanner>(ctx, task, conn_pool),
        ctx,
        task,
        x::breaker::default_config(task.name),
        cfg.scan_rate
    );

    driver::opc::connection::Config conn_cfg;
    conn_cfg.endpoint = "opc.tcp://localhost:4840";
    conn_cfg.security_mode = "None";
    conn_cfg.security_policy = "None";

    json scan_cmd{
        {"connection", conn_cfg.to_json()},
    };

    synnax::task::Command cmd{
        .task = task.key,
        .type = driver::opc::BROWSE_CMD_TYPE,
        .key = "scan_cmd",
        .args = scan_cmd
    };

    scan_task->exec(cmd);

    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    const auto &state = ctx->statuses[0];
    EXPECT_EQ(state.key, synnax::task::status_key(task));
    EXPECT_EQ(state.details.cmd, "scan_cmd");
    EXPECT_EQ(state.variant, x::status::VARIANT_SUCCESS);

    ASSERT_TRUE(state.details.data.has_value());
    auto &data = *state.details.data;
    ASSERT_TRUE(data.contains("channels"));
    auto channels = data["channels"];
    ASSERT_TRUE(channels.is_array());
    EXPECT_GE(channels.size(), 11);

    bool found_boolean = false;
    bool found_uint16 = false;
    bool found_float = false;
    bool found_double = false;

    for (const auto &ch: channels) {
        ASSERT_TRUE(ch.contains("name"));
        ASSERT_TRUE(ch.contains("node_id"));
        ASSERT_TRUE(ch.contains("data_type"));
        ASSERT_TRUE(ch.contains("node_class"));

        const auto name = ch["name"].get<std::string>();
        if (name == "TestBoolean") {
            found_boolean = true;
            EXPECT_EQ(ch["data_type"], "uint8");
            EXPECT_EQ(ch["node_class"], "Variable");
        } else if (name == "TestUInt16") {
            found_uint16 = true;
            EXPECT_EQ(ch["data_type"], "uint16");
            EXPECT_EQ(ch["node_class"], "Variable");
        } else if (name == "TestFloat") {
            found_float = true;
            EXPECT_EQ(ch["data_type"], "float32");
            EXPECT_EQ(ch["node_class"], "Variable");
        } else if (name == "TestDouble") {
            found_double = true;
            EXPECT_EQ(ch["data_type"], "float64");
            EXPECT_EQ(ch["node_class"], "Variable");
        }
    }

    EXPECT_TRUE(found_boolean);
    EXPECT_TRUE(found_uint16);
    EXPECT_TRUE(found_float);
    EXPECT_TRUE(found_double);
}

/// @brief it should reuse pooled connections for multiple scans.
TEST_F(TestScanTask, testConnectionPooling) {
    const auto cfg = driver::opc::ScanTaskConfig{};
    auto scan_task = std::make_unique<driver::task::common::ScanTask>(
        std::make_unique<driver::opc::Scanner>(ctx, task, conn_pool),
        ctx,
        task,
        x::breaker::default_config(task.name),
        cfg.scan_rate
    );

    driver::opc::connection::Config conn_cfg;
    conn_cfg.endpoint = "opc.tcp://localhost:4840";
    conn_cfg.security_mode = "None";
    conn_cfg.security_policy = "None";

    json scan_cmd{
        {"connection", conn_cfg.to_json()},
    };

    synnax::task::Command cmd1{
        .task = task.key,
        .type = driver::opc::BROWSE_CMD_TYPE,
        .key = "scan_cmd_1",
        .args = scan_cmd
    };

    scan_task->exec(cmd1);
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    EXPECT_EQ(ctx->statuses[0].variant, x::status::VARIANT_SUCCESS);

    synnax::task::Command cmd2{
        .task = task.key,
        .type = driver::opc::BROWSE_CMD_TYPE,
        .key = "scan_cmd_2",
        .args = scan_cmd
    };

    scan_task->exec(cmd2);
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 2);
    EXPECT_EQ(ctx->statuses[1].variant, x::status::VARIANT_SUCCESS);
}

/// @brief it should successfully test connection to OPC UA server.
TEST_F(TestScanTask, testTestConnection) {
    const auto cfg = driver::opc::ScanTaskConfig{};
    auto scan_task = std::make_unique<driver::task::common::ScanTask>(
        std::make_unique<driver::opc::Scanner>(ctx, task, conn_pool),
        ctx,
        task,
        x::breaker::default_config(task.name),
        cfg.scan_rate
    );

    driver::opc::connection::Config conn_cfg;
    conn_cfg.endpoint = "opc.tcp://localhost:4840";
    conn_cfg.security_mode = "None";
    conn_cfg.security_policy = "None";

    json test_conn_cmd{
        {"connection", conn_cfg.to_json()},
    };

    synnax::task::Command cmd{
        .task = task.key,
        .type = driver::opc::TEST_CONNECTION_CMD_TYPE,
        .key = "test_conn_cmd",
        .args = test_conn_cmd
    };

    scan_task->exec(cmd);

    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    const auto &state = ctx->statuses[0];
    EXPECT_EQ(state.key, synnax::task::status_key(task));
    EXPECT_EQ(state.details.cmd, "test_conn_cmd");
    EXPECT_EQ(state.variant, x::status::VARIANT_SUCCESS);
    EXPECT_EQ(state.message, "Connection successful");
}

/// @brief it should return error for invalid connection endpoint.
TEST_F(TestScanTask, testInvalidConnection) {
    auto cfg = driver::opc::ScanTaskConfig{};
    const auto scan_task = std::make_unique<driver::task::common::ScanTask>(
        std::make_unique<driver::opc::Scanner>(ctx, task, conn_pool),
        ctx,
        task,
        x::breaker::default_config(task.name),
        cfg.scan_rate
    );

    driver::opc::connection::Config conn_cfg;
    conn_cfg.endpoint = "opc.tcp://localhost:9999";
    conn_cfg.security_mode = "None";
    conn_cfg.security_policy = "None";

    json scan_cmd{
        {"connection", conn_cfg.to_json()},
    };

    synnax::task::Command cmd{
        .task = task.key,
        .type = driver::opc::BROWSE_CMD_TYPE,
        .key = "invalid_scan_cmd",
        .args = scan_cmd
    };

    scan_task->exec(cmd);

    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    const auto &state = ctx->statuses[0];
    EXPECT_EQ(state.key, synnax::task::status_key(task));
    EXPECT_EQ(state.details.cmd, "invalid_scan_cmd");
    EXPECT_EQ(state.variant, x::status::VARIANT_ERROR);
}

/// @brief Tests that driver::opc::Scanner::config() returns correct values.
TEST_F(TestScanTask, testConfigReturnsCorrectValues) {
    const driver::opc::Scanner scanner(ctx, task, conn_pool);
    auto cfg = scanner.config();
    EXPECT_EQ(cfg.make, "opc");
}

/// @brief Tests that exec() returns false for unknown commands.
TEST_F(TestScanTask, testExecReturnsFalseForUnknownCommand) {
    driver::opc::Scanner scanner(ctx, task, conn_pool);
    synnax::task::Command cmd{
        .task = task.key,
        .type = "unknown_command",
        .args = json{}
    };
    bool handled = scanner.exec(cmd, task, ctx);
    EXPECT_FALSE(handled);
}

/// @brief Tests that scan() checks device health and updates status.
TEST_F(TestScanTask, testScanChecksDeviceHealth) {
    driver::opc::Scanner scanner(ctx, task, conn_pool);

    // Create device with valid OPC connection properties
    synnax::device::Device dev;
    dev.key = "health-test-device";
    dev.name = "Health Test Device";
    dev.make = "opc";
    dev.rack = rack.key;
    dev.properties = json{
        {"connection",
         {{"endpoint", "opc.tcp://localhost:4840"},
          {"security_mode", "None"},
          {"security_policy", "None"}}},
        {"channels", json::array()}
    };

    // Pass devices via ScannerContext
    std::unordered_map<std::string, synnax::device::Device> devices_map;
    devices_map[dev.key] = dev;
    driver::task::common::ScannerContext scan_ctx;
    scan_ctx.devices = &devices_map;

    auto [devices, err] = scanner.scan(scan_ctx);
    ASSERT_NIL(err);
    ASSERT_EQ(devices.size(), 1);
    EXPECT_EQ(devices[0].status->variant, x::status::VARIANT_SUCCESS);
    EXPECT_EQ(devices[0].status->message, "Server connected");
}

/// @brief Tests that health check detects connection state changes (server up/down/up).
TEST_F(TestScanTask, testHealthCheckDetectsConnectionStateChanges) {
    // Create device with connection properties
    synnax::device::Device dev;
    dev.key = "connection-state-device";
    dev.name = "Connection State Test Device";
    dev.make = "opc";
    dev.rack = rack.key;
    dev.properties = json{
        {"connection",
         {{"endpoint", "opc.tcp://localhost:4840"},
          {"security_mode", "None"},
          {"security_policy", "None"}}},
        {"channels", json::array()}
    };

    std::unordered_map<std::string, synnax::device::Device> devices_map;
    devices_map[dev.key] = dev;
    driver::task::common::ScannerContext scan_ctx;
    scan_ctx.devices = &devices_map;

    // Use a fresh connection pool to avoid cached connections
    auto fresh_conn_pool = std::make_shared<driver::opc::connection::Pool>();
    driver::opc::Scanner scanner(ctx, task, fresh_conn_pool);

    // Step 1: Server is running (started in SetUp) - health should be good
    {
        auto [devices, err] = scanner.scan(scan_ctx);
        ASSERT_NIL(err);
        ASSERT_EQ(devices.size(), 1);
        EXPECT_EQ(devices[0].status->variant, x::status::VARIANT_SUCCESS);
        EXPECT_EQ(devices[0].status->message, "Server connected");
    }

    // Step 2: Stop the server - health should be bad
    server->stop();
    // Clear the connection pool to force new connection attempts
    fresh_conn_pool = std::make_shared<driver::opc::connection::Pool>();
    // Recreate scanner with fresh pool
    driver::opc::Scanner scanner2(ctx, task, fresh_conn_pool);

    {
        auto [devices, err] = scanner2.scan(scan_ctx);
        ASSERT_NIL(err);
        ASSERT_EQ(devices.size(), 1);
        // When server is down, health check should return WARNING with connection error
        EXPECT_EQ(devices[0].status->variant, x::status::VARIANT_WARNING);
        // The message should indicate a connection failure (not empty)
        EXPECT_FALSE(devices[0].status->message.empty());
        EXPECT_NE(devices[0].status->message, "Server connected");
        LOG(INFO) << "[test] Server down - status: " << devices[0].status->variant
                  << ", message: " << devices[0].status->message;
    }

    // Step 3: Restart the server - health should be good again
    auto server_cfg = mock::ServerConfig::create_default();
    server = std::make_unique<mock::Server>(server_cfg);
    server->start();

    // Wait for server to be ready
    driver::opc::connection::Config test_conn_cfg;
    test_conn_cfg.endpoint = "opc.tcp://localhost:4840";
    test_conn_cfg.security_mode = "None";
    test_conn_cfg.security_policy = "None";
    auto test_client = ASSERT_EVENTUALLY_NIL_P_WITH_TIMEOUT(
        driver::opc::connection::connect(test_conn_cfg, "test"),
        (5 * x::telem::SECOND).chrono(),
        (250 * x::telem::MILLISECOND).chrono()
    );
    UA_Client_disconnect(test_client.get());

    // Use fresh connection pool again
    fresh_conn_pool = std::make_shared<driver::opc::connection::Pool>();
    driver::opc::Scanner scanner3(ctx, task, fresh_conn_pool);

    {
        auto [devices, err] = scanner3.scan(scan_ctx);
        ASSERT_NIL(err);
        ASSERT_EQ(devices.size(), 1);
        EXPECT_EQ(devices[0].status->variant, x::status::VARIANT_SUCCESS);
        EXPECT_EQ(devices[0].status->message, "Server connected");
    }
}
