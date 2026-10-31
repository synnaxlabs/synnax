// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <filesystem>
#include <fstream>
#include <string>
#include <unordered_map>
#include <vector>

#include "gtest/gtest.h"
#include "nlohmann/json.hpp"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/common/scan_task.h"
#include "driver/opcua/certificate/store.h"
#include "driver/opcua/opcua.h"
#include "driver/opcua/scan_task.h"
#include "driver/opcua/testutil/server.h"
#include "driver/opcua/testutil/testutil.h"

namespace driver::opcua {
class TestScanTask : public ::testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> client;
    std::shared_ptr<task::MockContext> ctx;
    std::filesystem::path dir;
    std::shared_ptr<certificate::Store> store;
    std::shared_ptr<connection::Pool> conn_pool;
    std::unique_ptr<testutil::Server> server;
    std::unique_ptr<testutil::Server> secured_server;
    synnax::task::Task task;
    synnax::rack::Rack rack;

    void SetUp() override {
        client = std::make_shared<synnax::Synnax>(new_test_client());
        ctx = std::make_shared<task::MockContext>(client);
        dir = std::filesystem::temp_directory_path() /
              ("opcua-scan-" + x::uuid::create().to_string());
        store = std::make_shared<certificate::Store>(
            certificate::StoreConfig{.dir = dir}
        );
        conn_pool = std::make_shared<connection::Pool>(store);

        rack = ASSERT_NIL_P(client->racks.create("opc_scan_task_test_rack"));

        task = synnax::task::Task{
            .rack = rack.key,
            .name = "OPC UA Scan Task Test",
            .type = "opc_scan",
        };

        auto server_cfg = testutil::ServerConfig::create_default();
        server = std::make_unique<testutil::Server>(server_cfg);
        server->start();

        // Wait for server to be ready by attempting to connect
        connection::Config test_conn_cfg;
        test_conn_cfg.endpoint = "opc.tcp://localhost:4840";
        test_conn_cfg.security_mode = "None";
        test_conn_cfg.security_policy = "None";
        auto test_client = ASSERT_EVENTUALLY_NIL_P_WITH_TIMEOUT(
            connection::connect(test_conn_cfg, {}, "test"),
            (5 * x::telem::SECOND).chrono(),
            (250 * x::telem::MILLISECOND).chrono()
        );
        UA_Client_disconnect(test_client.get());
    }

    void TearDown() override {
        if (secured_server) secured_server->stop();
        std::filesystem::remove_all(dir);
    }

    void start_secured_server(
        const certificate::Identity &id,
        std::vector<certificate::Certificate> trusted_clients = {}
    ) {
        auto cfg = testutil::ServerConfig::create_default();
        cfg.port = 4871;
        cfg.identity = id;
        cfg.trusted_clients = std::move(trusted_clients);
        secured_server = std::make_unique<testutil::Server>(cfg);
        secured_server->start();
        ASSERT_TRUE(secured_server->wait_until_ready());
    }

    /// @brief replaces the Store with one that issues certificates inside the renewal
    /// window, so that renewal starts at once.
    void use_short_lived_store() {
        store = std::make_shared<certificate::Store>(certificate::StoreConfig{
            .dir = dir,
            .validity = 30 * x::telem::DAY,
        });
        conn_pool = std::make_shared<connection::Pool>(store);
    }

    [[nodiscard]] static connection::Config secured_config() {
        connection::Config cfg;
        cfg.endpoint = "opc.tcp://localhost:4871";
        cfg.security_mode = "SignAndEncrypt";
        cfg.security_policy = "Basic256Sha256";
        return cfg;
    }

    [[nodiscard]] synnax::device::Device
    secured_device(const connection::Config &cfg) const {
        synnax::device::Device dev;
        dev.key = "secured-device";
        dev.name = "Secured Device";
        dev.make = "opc";
        dev.rack = rack.key;
        dev.properties = x::json::json{
            {"connection", testutil::connection_json(cfg)},
            {"channels", x::json::json::array()}
        };
        return dev;
    }

    [[nodiscard]] std::vector<synnax::device::Device>
    scan(Scanner &scanner, const synnax::device::Device &dev) const {
        std::unordered_map<std::string, synnax::device::Device> devices{{dev.key, dev}};
        common::ScannerContext scan_ctx;
        scan_ctx.devices = &devices;
        auto [scanned, err] = scanner.scan(scan_ctx);
        EXPECT_FALSE(err) << err;
        return scanned;
    }
};

/// @brief it should browse and return OPC UA server nodes.
TEST_F(TestScanTask, testBasicScan) {
    const auto cfg = ScanTaskConfig{};
    auto scan_task = std::make_unique<common::ScanTask>(
        std::make_unique<Scanner>(ctx, task, conn_pool, store),
        ctx,
        task,
        x::breaker::default_config(task.name),
        cfg.rate
    );

    connection::Config conn_cfg;
    conn_cfg.endpoint = "opc.tcp://localhost:4840";
    conn_cfg.security_mode = "None";
    conn_cfg.security_policy = "None";

    x::json::json scan_cmd{
        {"connection", testutil::connection_json(conn_cfg)},
    };

    synnax::task::Command cmd{
        .task = task.key,
        .type = BROWSE_CMD_TYPE,
        .key = "scan_cmd",
        .args = scan_cmd
    };

    scan_task->exec(cmd);

    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    const auto &state = ctx->statuses[0];
    EXPECT_EQ(state.key, synnax::task::status_key(task));
    EXPECT_EQ(state.details.cmd, "scan_cmd");
    EXPECT_EQ(state.variant, synnax::status::VARIANT_SUCCESS);

    ASSERT_TRUE(state.details.data.has_value());
    const auto &data = state.details.data.value();
    ASSERT_TRUE(data.contains("channels"));
    auto channels = data.at("channels");
    ASSERT_TRUE(channels.is_array());
    EXPECT_GE(channels.size(), 11);

    bool found_boolean = false;
    bool found_uint16 = false;
    bool found_float = false;
    bool found_double = false;
    bool found_double_array = false;

    for (const auto &ch: channels) {
        ASSERT_TRUE(ch.contains("name"));
        ASSERT_TRUE(ch.contains("node_id"));
        ASSERT_TRUE(ch.contains("data_type"));
        ASSERT_TRUE(ch.contains("node_class"));

        const auto name = ch["name"].get<std::string>();
        if (name == "TestBoolean") {
            found_boolean = true;
            EXPECT_EQ(ch["data_type"], "boolean");
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
            EXPECT_FALSE(ch["is_array"].get<bool>());
        } else if (name == "TestDoubleArray") {
            found_double_array = true;
            EXPECT_EQ(ch["data_type"], "float64");
            EXPECT_TRUE(ch["is_array"].get<bool>());
        }
    }

    EXPECT_TRUE(found_boolean);
    EXPECT_TRUE(found_uint16);
    EXPECT_TRUE(found_float);
    EXPECT_TRUE(found_double);
    EXPECT_TRUE(found_double_array);
}

/// @brief it should reuse pooled connections for multiple scans.
TEST_F(TestScanTask, testConnectionPooling) {
    const auto cfg = ScanTaskConfig{};
    auto scan_task = std::make_unique<common::ScanTask>(
        std::make_unique<Scanner>(ctx, task, conn_pool, store),
        ctx,
        task,
        x::breaker::default_config(task.name),
        cfg.rate
    );

    connection::Config conn_cfg;
    conn_cfg.endpoint = "opc.tcp://localhost:4840";
    conn_cfg.security_mode = "None";
    conn_cfg.security_policy = "None";

    x::json::json scan_cmd{
        {"connection", testutil::connection_json(conn_cfg)},
    };

    synnax::task::Command cmd1{
        .task = task.key,
        .type = BROWSE_CMD_TYPE,
        .key = "scan_cmd_1",
        .args = scan_cmd
    };

    scan_task->exec(cmd1);
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    EXPECT_EQ(ctx->statuses[0].variant, synnax::status::VARIANT_SUCCESS);

    synnax::task::Command cmd2{
        .task = task.key,
        .type = BROWSE_CMD_TYPE,
        .key = "scan_cmd_2",
        .args = scan_cmd
    };

    scan_task->exec(cmd2);
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 2);
    EXPECT_EQ(ctx->statuses[1].variant, synnax::status::VARIANT_SUCCESS);
}

/// @brief it should successfully test connection to OPC UA server.
TEST_F(TestScanTask, testTestConnection) {
    const auto cfg = ScanTaskConfig{};
    const auto scan_task = std::make_unique<common::ScanTask>(
        std::make_unique<Scanner>(ctx, task, conn_pool, store),
        ctx,
        task,
        x::breaker::default_config(task.name),
        cfg.rate
    );

    connection::Config conn_cfg;
    conn_cfg.endpoint = "opc.tcp://localhost:4840";
    conn_cfg.security_mode = "None";
    conn_cfg.security_policy = "None";

    x::json::json test_conn_cmd{
        {"connection", testutil::connection_json(conn_cfg)},
    };

    synnax::task::Command cmd{
        .task = task.key,
        .type = TEST_CONNECTION_CMD_TYPE,
        .key = "test_conn_cmd",
        .args = test_conn_cmd
    };

    scan_task->exec(cmd);

    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    const auto &state = ctx->statuses[0];
    EXPECT_EQ(state.key, synnax::task::status_key(task));
    EXPECT_EQ(state.details.cmd, "test_conn_cmd");
    EXPECT_EQ(state.variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(state.message, "Connected to server");
}

/// @brief it should return error for invalid connection endpoint.
TEST_F(TestScanTask, testInvalidConnection) {
    auto cfg = ScanTaskConfig{};
    const auto scan_task = std::make_unique<common::ScanTask>(
        std::make_unique<Scanner>(ctx, task, conn_pool, store),
        ctx,
        task,
        x::breaker::default_config(task.name),
        cfg.rate
    );

    connection::Config conn_cfg;
    conn_cfg.endpoint = "opc.tcp://localhost:9999";
    conn_cfg.security_mode = "None";
    conn_cfg.security_policy = "None";

    x::json::json scan_cmd{
        {"connection", testutil::connection_json(conn_cfg)},
    };

    synnax::task::Command cmd{
        .task = task.key,
        .type = BROWSE_CMD_TYPE,
        .key = "invalid_scan_cmd",
        .args = scan_cmd
    };

    scan_task->exec(cmd);

    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    const auto &state = ctx->statuses[0];
    EXPECT_EQ(state.key, synnax::task::status_key(task));
    EXPECT_EQ(state.details.cmd, "invalid_scan_cmd");
    EXPECT_EQ(state.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(state.message, "Failed to connect to server");
}

/// @returns a command of type for the server at endpoint.
synnax::task::Command
command(const std::string &type, const std::string &endpoint, x::json::json args = {}) {
    connection::Config conn_cfg;
    conn_cfg.endpoint = endpoint;
    args["connection"] = testutil::connection_json(conn_cfg);
    return {.type = type, .key = type + "_cmd", .args = args};
}

/// @brief it should browse from a node ID with no namespace, and print node IDs with
/// the URI of their namespace.
TEST_F(TestScanTask, testBrowsePrintsNamespaceURIs) {
    Scanner scanner(ctx, task, conn_pool, store);
    auto cmd = command(
        BROWSE_CMD_TYPE,
        "opc.tcp://localhost:4840",
        {{"node_id", "i=85"}}
    );
    ASSERT_TRUE(scanner.exec(cmd, task, ctx));
    ASSERT_EQ(ctx->statuses.size(), 1);
    const auto &state = ctx->statuses[0];
    ASSERT_EQ(state.variant, synnax::status::VARIANT_SUCCESS) << state.description;
    const x::json::json data = state.details.data.value();
    EXPECT_FALSE(data.at("truncated").get<bool>());
    const auto &channels = data.at("channels");
    const auto it = std::ranges::find_if(channels, [](const auto &ch) {
        return ch["name"] == "TestBoolean";
    });
    ASSERT_NE(it, channels.end());
    EXPECT_EQ((*it)["node_id"], "nsu=urn:open62541.server.application;s=TestBoolean");
}

/// @brief it should print the URI of a namespace that the server added after the
/// connection opened.
TEST_F(TestScanTask, testBrowsePrintsNamespaceAddedAfterConnect) {
    Scanner scanner(ctx, task, conn_pool, store);
    auto cmd = command(
        BROWSE_CMD_TYPE,
        "opc.tcp://localhost:4840",
        {{"node_id", "i=85"}}
    );
    ASSERT_TRUE(scanner.exec(cmd, task, ctx));
    UA_Float value = 5.0f;
    UA_Variant variant;
    UA_Variant_setScalar(&variant, &value, &UA_TYPES[UA_TYPES_FLOAT]);
    server->add_namespace(
        "urn:late",
        {{0, "Late", &UA_TYPES[UA_TYPES_FLOAT], variant, "Late"}}
    );
    ASSERT_TRUE(scanner.exec(cmd, task, ctx));
    ASSERT_EQ(ctx->statuses.size(), 2);
    const auto &state = ctx->statuses[1];
    ASSERT_EQ(state.variant, synnax::status::VARIANT_SUCCESS) << state.description;
    const auto channels = x::json::json(state.details.data.value()).at("channels");
    const auto it = std::ranges::find_if(channels, [](const auto &ch) {
        return ch["name"] == "Late";
    });
    ASSERT_NE(it, channels.end());
    EXPECT_EQ((*it)["node_id"], "nsu=urn:late;s=Late");
    EXPECT_EQ(conn_pool->size(), 1);
}

/// @brief it should fail to browse from a node in a namespace that the server lacks.
TEST_F(TestScanTask, testBrowseUnknownNamespace) {
    Scanner scanner(ctx, task, conn_pool, store);
    auto cmd = command(
        BROWSE_CMD_TYPE,
        "opc.tcp://localhost:4840",
        {{"node_id", "nsu=urn:missing;i=85"}}
    );
    ASSERT_TRUE(scanner.exec(cmd, task, ctx));
    ASSERT_EQ(ctx->statuses.size(), 1);
    const auto &state = ctx->statuses[0];
    EXPECT_EQ(state.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(state.message, "Failed to browse nodes");
    EXPECT_EQ(state.description, "The server has no namespace urn:missing.");
}

/// @brief it should list the endpoints of a server without opening a Session, and close
/// its channel.
TEST_F(TestScanTask, testGetEndpoints) {
    const auto server_id = ASSERT_NIL_P(testutil::create_server_identity());
    start_secured_server(server_id);
    Scanner scanner(ctx, task, conn_pool, store);
    auto cmd = command(GET_ENDPOINTS_CMD_TYPE, "opc.tcp://localhost:4871");
    ASSERT_TRUE(scanner.exec(cmd, task, ctx));
    ASSERT_EQ(ctx->statuses.size(), 1);
    const auto &state = ctx->statuses[0];
    ASSERT_EQ(state.variant, synnax::status::VARIANT_SUCCESS) << state.description;
    EXPECT_EQ(state.message, "Received endpoints");
    const auto endpoints = x::json::json(state.details.data.value()).at("endpoints");
    const auto find = [&](const std::string &mode, const std::string &policy) {
        return std::ranges::find_if(endpoints, [&](const auto &e) {
            return e["security_mode"] == mode && e["security_policy"] == policy;
        });
    };
    const auto none = find("None", "None");
    ASSERT_NE(none, endpoints.end());
    EXPECT_FALSE((*none)["url"].get<std::string>().empty());
    const auto tokens = (*none)["user_token_types"].get<std::vector<std::string>>();
    EXPECT_NE(std::ranges::find(tokens, "anonymous"), tokens.end());
    const auto secured = find("SignAndEncrypt", "Basic256Sha256");
    ASSERT_NE(secured, endpoints.end());
    EXPECT_GT(
        (*secured)["security_level"].get<int>(),
        (*none)["security_level"].get<int>()
    );
    EXPECT_NE(find("Sign", "Basic256Sha256"), endpoints.end());
    EXPECT_EQ(conn_pool->size(), 0);
    EXPECT_EQ(secured_server->sessions_created.load(), 0);
    ASSERT_EVENTUALLY_EQ(secured_server->channels.load(), 0);
}

/// @brief it should fail to list the endpoints of a server that does not answer.
TEST_F(TestScanTask, testGetEndpointsUnreachable) {
    Scanner scanner(ctx, task, conn_pool, store);
    auto cmd = command(GET_ENDPOINTS_CMD_TYPE, "opc.tcp://localhost:9999");
    ASSERT_TRUE(scanner.exec(cmd, task, ctx));
    ASSERT_EQ(ctx->statuses.size(), 1);
    EXPECT_EQ(ctx->statuses[0].variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(ctx->statuses[0].message, "Failed to get endpoints");
}

/// @brief Tests that Scanner::config() returns correct values.
TEST_F(TestScanTask, testConfigReturnsCorrectValues) {
    const Scanner scanner(ctx, task, conn_pool, store);
    auto cfg = scanner.config();
    EXPECT_EQ(cfg.make, "opc");
}

/// @brief Tests that exec() returns false for unknown commands.
TEST_F(TestScanTask, testExecReturnsFalseForUnknownCommand) {
    Scanner scanner(ctx, task, conn_pool, store);
    synnax::task::Command cmd{.task = task.key, .type = "unknown_command"};
    bool handled = scanner.exec(cmd, task, ctx);
    EXPECT_FALSE(handled);
}

/// @brief Tests that scan() checks device health and updates status.
TEST_F(TestScanTask, testScanChecksDeviceHealth) {
    Scanner scanner(ctx, task, conn_pool, store);

    // Create device with valid OPC connection properties
    synnax::device::Device dev;
    dev.key = "health-test-device";
    dev.name = "Health Test Device";
    dev.make = "opc";
    dev.rack = rack.key;
    dev.properties = x::json::json{
        {"connection",
         {{"endpoint", "opc.tcp://localhost:4840"},
          {"security_mode", "None"},
          {"security_policy", "None"}}},
        {"channels", x::json::json::array()}
    };

    // Pass devices via ScannerContext
    std::unordered_map<std::string, synnax::device::Device> devices_map;
    devices_map[dev.key] = dev;
    common::ScannerContext scan_ctx;
    scan_ctx.devices = &devices_map;

    auto devices = ASSERT_NIL_P(scanner.scan(scan_ctx));
    ASSERT_EQ(devices.size(), 1);
    EXPECT_EQ(devices[0].status.value().variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(devices[0].status.value().message, "Server connected");
}

/// @brief it should not open a session of its own when a task received a reply from the
/// server since the last check, and open one when no task did.
TEST_F(TestScanTask, testHealthCheckRidesTaskSession) {
    Scanner scanner(ctx, task, conn_pool, store);
    connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4840";
    synnax::device::Device dev;
    dev.key = "ride-device";
    dev.name = "Ride Device";
    dev.make = "opc";
    dev.rack = rack.key;
    dev.properties = x::json::json{
        {"connection", testutil::connection_json(cfg)},
        {"channels", x::json::json::array()}
    };
    const auto task_conn = ASSERT_NIL_P(conn_pool->acquire(cfg, "[test] "));
    const auto first = this->scan(scanner, dev);
    EXPECT_EQ(first[0].status.value().message, "Server connected");
    EXPECT_EQ(conn_pool->size(), 1);
    const auto second = this->scan(scanner, dev);
    EXPECT_EQ(second[0].status.value().message, "Server connected");
    EXPECT_EQ(conn_pool->size(), 2);
}

/// @brief it should probe the server on each scan when no task gets replies, so the
/// reply to its own probe does not hide an outage on the next scan.
TEST_F(TestScanTask, testHealthCheckProbesEachScan) {
    Scanner scanner(ctx, task, conn_pool, store);
    connection::Config cfg;
    cfg.endpoint = "opc.tcp://localhost:4840";
    synnax::device::Device dev;
    dev.key = "probe-device";
    dev.name = "Probe Device";
    dev.make = "opc";
    dev.rack = rack.key;
    dev.properties = x::json::json{
        {"connection", testutil::connection_json(cfg)},
        {"channels", x::json::json::array()}
    };
    const auto first = this->scan(scanner, dev);
    EXPECT_EQ(first[0].status.value().message, "Server connected");
    server->stop();
    const auto second = this->scan(scanner, dev);
    EXPECT_EQ(second[0].status.value().variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(second[0].status.value().message, "Failed to reach server");
}

/// @brief Tests that health check detects connection state changes (server up/down/up).
TEST_F(TestScanTask, testHealthCheckDetectsConnectionStateChanges) {
    // Create device with connection properties
    synnax::device::Device dev;
    dev.key = "connection-state-device";
    dev.name = "Connection State Test Device";
    dev.make = "opc";
    dev.rack = rack.key;
    dev.properties = x::json::json{
        {"connection",
         {{"endpoint", "opc.tcp://localhost:4840"},
          {"security_mode", "None"},
          {"security_policy", "None"}}},
        {"channels", x::json::json::array()}
    };

    std::unordered_map<std::string, synnax::device::Device> devices_map;
    devices_map[dev.key] = dev;
    common::ScannerContext scan_ctx;
    scan_ctx.devices = &devices_map;

    // Use a fresh connection pool to avoid cached connections
    auto fresh_conn_pool = std::make_shared<connection::Pool>(store);
    Scanner scanner(ctx, task, fresh_conn_pool, store);

    // Step 1: Server is running (started in SetUp) - health should be good
    {
        auto devices = ASSERT_NIL_P(scanner.scan(scan_ctx));
        ASSERT_EQ(devices.size(), 1);
        EXPECT_EQ(devices[0].status.value().variant, synnax::status::VARIANT_SUCCESS);
        EXPECT_EQ(devices[0].status.value().message, "Server connected");
    }

    // Step 2: Stop the server - health should be bad
    server->stop();
    // Clear the connection pool to force new connection attempts
    fresh_conn_pool = std::make_shared<connection::Pool>(store);
    // Recreate scanner with fresh pool
    Scanner scanner2(ctx, task, fresh_conn_pool, store);

    {
        auto devices = ASSERT_NIL_P(scanner2.scan(scan_ctx));
        ASSERT_EQ(devices.size(), 1);
        // When server is down, health check should return WARNING with connection error
        EXPECT_EQ(devices[0].status.value().variant, synnax::status::VARIANT_WARNING);
        // The message should indicate a connection failure (not empty)
        EXPECT_FALSE(devices[0].status.value().message.empty());
        EXPECT_NE(devices[0].status.value().message, "Server connected");
        LOG(INFO) << "[test] Server down - status: "
                  << devices[0].status.value().variant
                  << ", message: " << devices[0].status.value().message;
    }

    // Step 3: Restart the server - health should be good again
    auto server_cfg = testutil::ServerConfig::create_default();
    server = std::make_unique<testutil::Server>(server_cfg);
    server->start();

    // Wait for server to be ready
    connection::Config test_conn_cfg;
    test_conn_cfg.endpoint = "opc.tcp://localhost:4840";
    test_conn_cfg.security_mode = "None";
    test_conn_cfg.security_policy = "None";
    auto test_client = ASSERT_EVENTUALLY_NIL_P_WITH_TIMEOUT(
        connection::connect(test_conn_cfg, {}, "test"),
        (5 * x::telem::SECOND).chrono(),
        (250 * x::telem::MILLISECOND).chrono()
    );
    UA_Client_disconnect(test_client.get());

    // Use fresh connection pool again
    fresh_conn_pool = std::make_shared<connection::Pool>(store);
    Scanner scanner3(ctx, task, fresh_conn_pool, store);

    {
        auto devices = ASSERT_NIL_P(scanner3.scan(scan_ctx));
        ASSERT_EQ(devices.size(), 1);
        EXPECT_EQ(devices[0].status.value().variant, synnax::status::VARIANT_SUCCESS);
        EXPECT_EQ(devices[0].status.value().message, "Server connected");
    }
}

/// @brief it should reply with a server certificate that the command does not trust,
/// and connect once the command trusts that certificate.
TEST_F(TestScanTask, testTestConnectionUntrustedCertificate) {
    const auto server_id = ASSERT_NIL_P(testutil::create_server_identity());
    start_secured_server(server_id);
    Scanner scanner(ctx, task, conn_pool, store);
    auto conn_cfg = secured_config();
    synnax::task::Command cmd{
        .task = task.key,
        .type = TEST_CONNECTION_CMD_TYPE,
        .key = "untrusted_cmd",
        .args = {{"connection", testutil::connection_json(conn_cfg)}},
    };
    ASSERT_TRUE(scanner.exec(cmd, task, ctx));
    ASSERT_EQ(ctx->statuses.size(), 1);
    const auto rejected = ctx->statuses[0];
    EXPECT_EQ(rejected.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(rejected.message, "Server certificate not trusted");
    EXPECT_EQ(
        rejected.description,
        "The Driver does not trust the server certificate " +
            server_id.certificate.thumbprint() +
            " yet. Compare the thumbprint with the certificate on the server, then "
            "trust it."
    );
    ASSERT_TRUE(rejected.details.data.has_value());
    EXPECT_EQ(
        x::json::json(rejected.details.data.value()),
        x::json::json({{"server_certificate", server_id.certificate.info().to_json()}})
    );

    conn_cfg.server_certificate = server_id.certificate;
    cmd.key = "trusted_cmd";
    cmd.args = {{"connection", testutil::connection_json(conn_cfg)}};
    ASSERT_TRUE(scanner.exec(cmd, task, ctx));
    ASSERT_EQ(ctx->statuses.size(), 2);
    EXPECT_EQ(ctx->statuses[1].variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(ctx->statuses[1].message, "Connected to server");
}

/// @brief it should trust the first certificate of a secured device that trusts
/// none, and keep using its connection once the device trusts that certificate.
TEST_F(TestScanTask, testHealthCheckTrustsFirstCertificate) {
    const auto server_id = ASSERT_NIL_P(testutil::create_server_identity());
    start_secured_server(server_id);
    Scanner scanner(ctx, task, conn_pool, store);
    auto scanned = scan(scanner, secured_device(secured_config()));
    ASSERT_EQ(scanned.size(), 1);
    EXPECT_EQ(scanned[0].status.value().variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(scanned[0].status.value().message, "Server connected");
    EXPECT_EQ(
        scanned[0].properties.at("connection").at("server_certificate"),
        server_id.certificate.pem()
    );

    scanned = scan(scanner, scanned[0]);
    ASSERT_EQ(scanned.size(), 1);
    EXPECT_EQ(scanned[0].status.value().message, "Server connected");
    EXPECT_EQ(conn_pool->size(), 1);
}

/// @brief it should report a secured device whose server presents another
/// certificate than the trusted one.
TEST_F(TestScanTask, testHealthCheckCertificateMismatch) {
    const auto server_id = ASSERT_NIL_P(testutil::create_server_identity());
    const auto other_id = ASSERT_NIL_P(testutil::create_server_identity());
    start_secured_server(server_id);
    Scanner scanner(ctx, task, conn_pool, store);
    auto conn_cfg = secured_config();
    conn_cfg.server_certificate = other_id.certificate;
    const auto scanned = scan(scanner, secured_device(conn_cfg));
    ASSERT_EQ(scanned.size(), 1);
    const auto &status = scanned[0].status.value();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(status.message, "Server certificate not trusted");
    EXPECT_EQ(
        status.description,
        "The server presented the certificate " + server_id.certificate.thumbprint() +
            " instead of the trusted certificate " + other_id.certificate.thumbprint() +
            ". If the server certificate changed on purpose, trust the new one in the "
            "device's connection settings."
    );
    EXPECT_EQ(
        scanned[0].properties.at("connection").at("server_certificate"),
        other_id.certificate.pem()
    );
}

/// @brief it should warn about a secured device that accepts any server certificate.
TEST_F(TestScanTask, testHealthCheckValidationDisabled) {
    const auto server_id = ASSERT_NIL_P(testutil::create_server_identity());
    start_secured_server(server_id);
    Scanner scanner(ctx, task, conn_pool, store);
    auto conn_cfg = secured_config();
    conn_cfg.certificate_validation_disabled = true;
    const auto scanned = scan(scanner, secured_device(conn_cfg));
    ASSERT_EQ(scanned.size(), 1);
    EXPECT_EQ(scanned[0].status.value().variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(
        scanned[0].status.value().message,
        "Connected without certificate validation"
    );
    EXPECT_FALSE(scanned[0].properties.at("connection").contains("server_certificate"));
}

/// @brief it should name the Driver certificate when the server rejects it.
TEST_F(TestScanTask, testHealthCheckDriverCertificateRejected) {
    const auto server_id = ASSERT_NIL_P(testutil::create_server_identity());
    const auto other_id = ASSERT_NIL_P(testutil::create_server_identity());
    start_secured_server(server_id, {other_id.certificate});
    Scanner scanner(ctx, task, conn_pool, store);
    auto conn_cfg = secured_config();
    conn_cfg.server_certificate = server_id.certificate;
    const auto scanned = scan(scanner, secured_device(conn_cfg));
    ASSERT_EQ(scanned.size(), 1);
    const auto current = ASSERT_NIL_P(store->current());
    const auto &status = scanned[0].status.value();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(status.message, "Server rejected the Driver certificate");
    EXPECT_EQ(
        status.description,
        "The server rejected the connection's security checks "
        "(BadSecurityChecksFailed). Most often, the server does not trust the client "
        "certificate " +
            current.certificate.thumbprint() +
            ". Approve it in the server's trusted certificates, then connect again."
    );
}

/// @brief it should reply with the current Driver certificate.
TEST_F(TestScanTask, testGetCertificate) {
    Scanner scanner(ctx, task, conn_pool, store);
    synnax::task::Command cmd{
        .task = task.key,
        .type = GET_CERTIFICATE_CMD_TYPE,
        .key = "get_cmd",
    };
    ASSERT_TRUE(scanner.exec(cmd, task, ctx));
    ASSERT_EQ(ctx->statuses.size(), 1);
    const auto current = ASSERT_NIL_P(store->current());
    const auto &status = ctx->statuses[0];
    EXPECT_EQ(status.variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(status.message, "Retrieved the Driver certificate");
    EXPECT_EQ(status.details.cmd, "get_cmd");
    ASSERT_TRUE(status.details.data.has_value());
    EXPECT_EQ(
        x::json::json(status.details.data.value()),
        x::json::json({{"current", current.certificate.info().to_json()}})
    );
}

/// @brief it should replace the Driver certificate and reply with the new one.
TEST_F(TestScanTask, testRegenerateCertificate) {
    const auto old = ASSERT_NIL_P(store->current());
    Scanner scanner(ctx, task, conn_pool, store);
    synnax::task::Command cmd{
        .task = task.key,
        .type = REGENERATE_CERTIFICATE_CMD_TYPE,
        .key = "regenerate_cmd",
    };
    ASSERT_TRUE(scanner.exec(cmd, task, ctx));
    ASSERT_EQ(ctx->statuses.size(), 1);
    const auto current = ASSERT_NIL_P(store->current());
    ASSERT_NE(current.certificate, old.certificate);
    const auto &status = ctx->statuses[0];
    EXPECT_EQ(status.variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(status.message, "Regenerated the Driver certificate");
    ASSERT_TRUE(status.details.data.has_value());
    EXPECT_EQ(
        x::json::json(status.details.data.value()),
        x::json::json({{"current", current.certificate.info().to_json()}})
    );
}

/// @brief it should switch to the next Driver certificate once every secured server
/// accepts it.
TEST_F(TestScanTask, testRenewalSwitchesWhenServersAccept) {
    use_short_lived_store();
    const auto old = ASSERT_NIL_P(store->current());
    const auto server_id = ASSERT_NIL_P(testutil::create_server_identity());
    start_secured_server(server_id);
    Scanner scanner(ctx, task, conn_pool, store);
    auto conn_cfg = secured_config();
    conn_cfg.server_certificate = server_id.certificate;
    const auto scanned = scan(scanner, secured_device(conn_cfg));
    ASSERT_EQ(scanned.size(), 1);
    EXPECT_EQ(scanned[0].status.value().message, "Server connected");
    const auto current = ASSERT_NIL_P(store->current());
    EXPECT_NE(current.certificate, old.certificate);
    const auto next = ASSERT_NIL_P(store->next());
    EXPECT_FALSE(next.has_value());
}

/// @brief it should keep the current Driver certificate and warn while a server does
/// not trust the next one.
TEST_F(TestScanTask, testRenewalWaitsForServers) {
    use_short_lived_store();
    const auto old = ASSERT_NIL_P(store->current());
    const auto server_id = ASSERT_NIL_P(testutil::create_server_identity());
    start_secured_server(server_id, {old.certificate});
    Scanner scanner(ctx, task, conn_pool, store);
    auto conn_cfg = secured_config();
    conn_cfg.server_certificate = server_id.certificate;
    const auto scanned = scan(scanner, secured_device(conn_cfg));
    ASSERT_EQ(scanned.size(), 1);
    const auto current = ASSERT_NIL_P(store->current());
    EXPECT_EQ(current.certificate, old.certificate);
    const auto next = ASSERT_NIL_P(store->next());
    ASSERT_TRUE(next.has_value());
    const auto &status = scanned[0].status.value();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(status.message, "Server does not trust the next certificate");
    EXPECT_EQ(
        status.description,
        "Approve the next Driver certificate " + next.value().certificate.thumbprint() +
            " on the server before " + old.certificate.not_after().iso8601() +
            ", when the current certificate expires."
    );
}

/// @brief it should not report that a server distrusts the next Driver certificate
/// when the renewal attempt could not reach the server.
TEST_F(TestScanTask, testRenewalIgnoresUnreachableServer) {
    use_short_lived_store();
    const auto old = ASSERT_NIL_P(store->current());
    const auto server_id = ASSERT_NIL_P(testutil::create_server_identity());
    Scanner scanner(ctx, task, conn_pool, store);
    auto conn_cfg = secured_config();
    conn_cfg.server_certificate = server_id.certificate;
    const auto dev = secured_device(conn_cfg);
    const auto unreachable = scan(scanner, dev);
    ASSERT_EQ(unreachable.size(), 1);
    EXPECT_EQ(unreachable[0].status.value().message, "Failed to reach server");
    start_secured_server(server_id);
    const auto scanned = scan(scanner, dev);
    ASSERT_EQ(scanned.size(), 1);
    EXPECT_EQ(scanned[0].status.value().variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(scanned[0].status.value().message, "Server connected");
    const auto current = ASSERT_NIL_P(store->current());
    EXPECT_EQ(current.certificate, old.certificate);
}

/// @brief it should keep the scanned devices and report a failed renewal as a partial
/// scan failure.
TEST_F(TestScanTask, testRenewalFailureIsPartialScan) {
    const auto certs = dir / "own" / "certs";
    std::filesystem::create_directories(certs);
    const auto corrupt = certs / "corrupt.der";
    std::ofstream(corrupt) << "not a certificate";
    Scanner scanner(ctx, task, conn_pool, store);
    const auto dev = secured_device(secured_config());
    std::unordered_map<std::string, synnax::device::Device> devices{{dev.key, dev}};
    common::ScannerContext scan_ctx;
    scan_ctx.devices = &devices;
    const auto [scanned, err] = scanner.scan(scan_ctx);
    ASSERT_OCCURRED_AS(err, common::PARTIAL_SCAN_ERROR);
    ASSERT_EQ(
        err.data,
        "Failed to renew the Driver certificate: Failed to parse " + corrupt.string()
    );
    ASSERT_EQ(scanned.size(), 1);
}
/// @brief browses the node at path in the server on port 4881, and returns the status.
synnax::task::Status browse_node(
    const std::shared_ptr<task::MockContext> &ctx,
    const synnax::task::Task &task,
    const std::shared_ptr<connection::Pool> &pool,
    const std::shared_ptr<certificate::Store> &store,
    const std::string &node_id,
    const std::string &field
) {
    connection::Config conn_cfg;
    conn_cfg.endpoint = "opc.tcp://localhost:4881";
    conn_cfg.security_mode = "None";
    conn_cfg.security_policy = "None";
    Scanner scanner(ctx, task, pool, store);
    const auto before = ctx->statuses.size();
    synnax::task::Command cmd{
        .task = task.key,
        .type = BROWSE_CMD_TYPE,
        .key = "browse_cmd",
        .args = {
            {"connection", testutil::connection_json(conn_cfg)},
            {"node_id", node_id},
            {"field", field},
        },
    };
    EXPECT_TRUE(scanner.exec(cmd, task, ctx));
    EXPECT_EQ(ctx->statuses.size(), before + 1);
    return ctx->statuses.back();
}

/// @brief starts a server that holds a TestMotor node, of the Motor structure type, and
/// the Pump object.
std::unique_ptr<testutil::Server> start_motor_server() {
    auto cfg = testutil::ServerConfig::create_default();
    cfg.port = 4881;
    cfg.pump = testutil::Pump{};
    cfg.motor = testutil::Motor{
        .speed = 1500.0,
        .running = true,
        .position = {.x = 1.5F, .y = 2.5F},
    };
    auto server = std::make_unique<testutil::Server>(cfg);
    server->start();
    EXPECT_TRUE(server->wait_until_ready());
    return server;
}

/// @brief it should list the members of a structure variable, with their fields, and
/// not the type definition of the variable.
TEST_F(TestScanTask, testBrowsesStructureMembers) {
    const auto motor_server = start_motor_server();
    const auto
        status = browse_node(ctx, task, conn_pool, store, "NS=1;S=TestMotor", "");
    ASSERT_EQ(status.variant, synnax::status::VARIANT_SUCCESS) << status.description;
    const auto &channels = status.details.data.value().at("channels");
    ASSERT_EQ(channels.size(), 3);
    EXPECT_EQ(channels[0]["name"], "Speed");
    EXPECT_EQ(channels[0]["field"], "Speed");
    EXPECT_EQ(channels[0]["data_type"], "float64");
    EXPECT_EQ(channels[0]["node_id"], "NS=1;S=TestMotor");
    EXPECT_EQ(channels[0]["node_class"], "Variable");
    EXPECT_EQ(channels[1]["field"], "Running");
    EXPECT_EQ(channels[1]["data_type"], "boolean");
    EXPECT_EQ(channels[2]["field"], "Position");
    motor_server->stop();
}

/// @brief it should list the members of a nested structure member.
TEST_F(TestScanTask, testBrowsesNestedMembers) {
    const auto motor_server = start_motor_server();
    const auto status = browse_node(
        ctx,
        task,
        conn_pool,
        store,
        "NS=1;S=TestMotor",
        "Position"
    );
    ASSERT_EQ(status.variant, synnax::status::VARIANT_SUCCESS) << status.description;
    const auto &channels = status.details.data.value().at("channels");
    ASSERT_EQ(channels.size(), 2);
    EXPECT_EQ(channels[0]["field"], "Position.X");
    EXPECT_EQ(channels[0]["data_type"], "float32");
    EXPECT_EQ(channels[1]["field"], "Position.Y");
    motor_server->stop();
}

/// @brief it should list the children of a structure variable whose type does not
/// build, and warn that its members do not list.
TEST_F(TestScanTask, testBrowsesChildrenOfUnbuiltStructure) {
    auto cfg = testutil::ServerConfig::create_default();
    cfg.port = 4881;
    cfg.broken = testutil::Broken{};
    testutil::Server server(cfg);
    server.start();
    ASSERT_TRUE(server.wait_until_ready());
    const auto
        status = browse_node(ctx, task, conn_pool, store, "NS=1;S=TestBroken", "");
    EXPECT_EQ(status.variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(status.message, "Failed to list structure members");
    EXPECT_EQ(
        status.description,
        "cannot decode Broken: the server did not define the type of a member"
    );
    const auto &channels = status.details.data.value().at("channels");
    ASSERT_EQ(channels.size(), 1);
    EXPECT_EQ(
        channels[0]["node_id"],
        "nsu=urn:open62541.server.application;s=TestBrokenChild"
    );
    server.stop();
}

/// @brief it should fail to list the members of a node that holds no structure.
TEST_F(TestScanTask, testRejectsFieldOfScalar) {
    const auto motor_server = start_motor_server();
    const auto
        status = browse_node(ctx, task, conn_pool, store, "NS=1;S=TestDouble", "Speed");
    EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(status.message, "Failed to browse nodes");
    EXPECT_EQ(status.description, "Double has no member Speed");
    motor_server->stop();
}

/// @returns the status of a command of type with args to the server on port.
synnax::task::Status run_command(
    const std::shared_ptr<task::MockContext> &ctx,
    const synnax::task::Task &task,
    const std::shared_ptr<connection::Pool> &pool,
    const std::shared_ptr<certificate::Store> &store,
    const std::string &type,
    const int port,
    x::json::json args
) {
    Scanner scanner(ctx, task, pool, store);
    auto cmd = command(
        type,
        "opc.tcp://localhost:" + std::to_string(port),
        std::move(args)
    );
    const auto before = ctx->statuses.size();
    EXPECT_TRUE(scanner.exec(cmd, task, ctx));
    EXPECT_EQ(ctx->statuses.size(), before + 1);
    return ctx->statuses.back();
}

/// @brief starts a server whose Generated folder holds the variables and empty folders.
std::unique_ptr<testutil::Server>
start_generated_server(const std::uint32_t variables, const std::uint32_t folders = 0) {
    auto cfg = testutil::ServerConfig::create_default();
    cfg.port = 4882;
    cfg.generated_variables = variables;
    cfg.generated_folders = folders;
    cfg.max_nodes_per_browse = 1000;
    auto server = std::make_unique<testutil::Server>(cfg);
    server->start();
    EXPECT_TRUE(server->wait_until_ready());
    return server;
}

/// @brief it should find the variables under a folder at any depth, with their paths
/// from below the folder, and visit a folder that a cycle reaches only once.
TEST_F(TestScanTask, testBrowseVariables) {
    const auto generated = start_generated_server(3);
    const auto status = run_command(
        ctx,
        task,
        conn_pool,
        store,
        BROWSE_VARIABLES_CMD_TYPE,
        4882,
        {{"node_id", "NS=1;S=Generated"}}
    );
    ASSERT_EQ(status.variant, synnax::status::VARIANT_SUCCESS) << status.description;
    EXPECT_EQ(status.message, "Found variables");
    const x::json::json data = status.details.data.value();
    EXPECT_FALSE(data.at("truncated").get<bool>());
    EXPECT_TRUE(data.at("skipped").empty());
    const auto &channels = data.at("channels");
    ASSERT_EQ(channels.size(), 4);
    EXPECT_EQ(channels[0]["name"], "V0");
    EXPECT_EQ(channels[0]["path"], x::json::json::array({"V0"}));
    EXPECT_EQ(channels[0]["data_type"], "float64");
    EXPECT_EQ(channels[0]["node_class"], "Variable");
    EXPECT_EQ(
        channels[0]["node_id"],
        "nsu=urn:open62541.server.application;s=Generated.V0"
    );
    EXPECT_EQ(channels[2]["path"], x::json::json::array({"V2"}));
    EXPECT_EQ(channels[3]["name"], "Deep");
    EXPECT_EQ(channels[3]["path"], x::json::json::array({"Inner", "Deep"}));
    generated->stop();
}

/// @brief it should return why it skipped a node that failed to browse.
TEST_F(TestScanTask, testBrowseVariablesReturnsSkippedErrors) {
    const auto generated = start_generated_server(1);
    const auto status = run_command(
        ctx,
        task,
        conn_pool,
        store,
        BROWSE_VARIABLES_CMD_TYPE,
        4882,
        {{"node_id", "NS=1;S=Missing"}}
    );
    ASSERT_EQ(status.variant, synnax::status::VARIANT_SUCCESS) << status.description;
    const x::json::json data = status.details.data.value();
    EXPECT_TRUE(data.at("channels").empty());
    const auto &skipped = data.at("skipped");
    ASSERT_EQ(skipped.size(), 1);
    const auto text = skipped[0].get<std::string>();
    EXPECT_TRUE(text.starts_with("NS=1;S=Missing: ")) << text;
    generated->stop();
}

/// @brief it should stop at the most variables that it returns, and say so.
TEST_F(TestScanTask, testBrowseVariablesStopsAtLimit) {
    const auto generated = start_generated_server(MAX_VARIABLES + 1);
    const auto status = run_command(
        ctx,
        task,
        conn_pool,
        store,
        BROWSE_VARIABLES_CMD_TYPE,
        4882,
        {{"node_id", "NS=1;S=Generated"}}
    );
    ASSERT_EQ(status.variant, synnax::status::VARIANT_SUCCESS) << status.description;
    const x::json::json data = status.details.data.value();
    EXPECT_TRUE(data.at("truncated").get<bool>());
    EXPECT_EQ(data.at("channels").size(), MAX_VARIABLES);
    generated->stop();
}

/// @brief it should return the first children of a node that has more than the most
/// that a browse lists, and say so.
TEST_F(TestScanTask, testBrowseStopsAtChildLimit) {
    const auto generated = start_generated_server(connection::MAX_CHILDREN + 1);
    const auto status = run_command(
        ctx,
        task,
        conn_pool,
        store,
        BROWSE_CMD_TYPE,
        4882,
        {{"node_id", "NS=1;S=Generated"}}
    );
    ASSERT_EQ(status.variant, synnax::status::VARIANT_SUCCESS) << status.description;
    const x::json::json data = status.details.data.value();
    EXPECT_TRUE(data.at("truncated").get<bool>());
    EXPECT_EQ(data.at("channels").size(), connection::MAX_CHILDREN);
    generated->stop();
}

/// @brief starts a server whose Generated folder holds the variables, which waits 100
/// ms before it handles each request and takes the type of one variable in each Read.
std::unique_ptr<testutil::Server> start_slow_server(const std::uint32_t variables) {
    auto cfg = testutil::ServerConfig::create_default();
    cfg.port = 4882;
    cfg.generated_variables = variables;
    cfg.max_nodes_per_read = 2;
    auto server = std::make_unique<testutil::Server>(cfg);
    server->start();
    EXPECT_TRUE(server->wait_until_ready());
    server->probe_delay = 100 * x::telem::MILLISECOND;
    return server;
}

/// @brief it should return the nodes that it typed before the 10 s that the Console
/// waits, and say that it stopped early.
TEST_F(TestScanTask, testBrowseStopsAtDeadline) {
    const auto slow = start_slow_server(300);
    const auto start = std::chrono::steady_clock::now();
    const auto status = run_command(
        ctx,
        task,
        conn_pool,
        store,
        BROWSE_CMD_TYPE,
        4882,
        {{"node_id", "NS=1;S=Generated"}}
    );
    EXPECT_LT(std::chrono::steady_clock::now() - start, std::chrono::seconds(10));
    ASSERT_EQ(status.variant, synnax::status::VARIANT_SUCCESS) << status.description;
    const x::json::json data = status.details.data.value();
    EXPECT_TRUE(data.at("truncated").get<bool>());
    EXPECT_GT(data.at("channels").size(), 0);
    EXPECT_LT(data.at("channels").size(), 301);
    slow->stop();
}

/// @brief it should return the variables that it typed before the 30 s that the Console
/// waits, and say that it stopped early.
TEST_F(TestScanTask, testBrowseVariablesStopsAtDeadline) {
    const auto slow = start_slow_server(300);
    const auto start = std::chrono::steady_clock::now();
    const auto status = run_command(
        ctx,
        task,
        conn_pool,
        store,
        BROWSE_VARIABLES_CMD_TYPE,
        4882,
        {{"node_id", "NS=1;S=Generated"}}
    );
    EXPECT_LT(std::chrono::steady_clock::now() - start, std::chrono::seconds(30));
    ASSERT_EQ(status.variant, synnax::status::VARIANT_SUCCESS) << status.description;
    const x::json::json data = status.details.data.value();
    EXPECT_TRUE(data.at("truncated").get<bool>());
    EXPECT_GT(data.at("channels").size(), 0);
    EXPECT_LT(data.at("channels").size(), 301);
    slow->stop();
}

/// @brief it should stop at the most objects that it browses, and say so.
TEST_F(TestScanTask, testBrowseVariablesStopsAtObjectLimit) {
    const auto generated = start_generated_server(1, MAX_OBJECTS);
    const auto status = run_command(
        ctx,
        task,
        conn_pool,
        store,
        BROWSE_VARIABLES_CMD_TYPE,
        4882,
        {{"node_id", "NS=1;S=Generated"}}
    );
    ASSERT_EQ(status.variant, synnax::status::VARIANT_SUCCESS) << status.description;
    const x::json::json data = status.details.data.value();
    EXPECT_TRUE(data.at("truncated").get<bool>());
    EXPECT_EQ(data.at("channels")[0]["name"], "V0");
    generated->stop();
}

/// @brief it should read the attributes and the value of a variable.
TEST_F(TestScanTask, testReadNode) {
    const auto status = run_command(
        ctx,
        task,
        conn_pool,
        store,
        READ_NODE_CMD_TYPE,
        4840,
        {{"node_id", "NS=1;S=TestDouble"}}
    );
    ASSERT_EQ(status.variant, synnax::status::VARIANT_SUCCESS) << status.description;
    EXPECT_EQ(status.message, "Read node");
    const x::json::json data = status.details.data.value();
    EXPECT_EQ(data["node_class"], "Variable");
    EXPECT_EQ(data["browse_name"], "TestDouble");
    EXPECT_EQ(data["display_name"], "TestDouble");
    EXPECT_EQ(data["type_name"], "Double");
    EXPECT_EQ(data["value_rank"], UA_VALUERANK_SCALAR);
    EXPECT_TRUE(data["readable"].get<bool>());
    EXPECT_TRUE(data["writable"].get<bool>());
    EXPECT_FALSE(data["history_readable"].get<bool>());
    EXPECT_EQ(data["status"], "Good");
    EXPECT_DOUBLE_EQ(std::stod(data["value"].get<std::string>()), 2.71828);
    EXPECT_TRUE(data.contains("server_timestamp"));
    const auto &node = data["node"];
    EXPECT_EQ(node["name"], "TestDouble");
    EXPECT_EQ(node["data_type"], "float64");
    EXPECT_EQ(node["node_id"], "nsu=urn:open62541.server.application;s=TestDouble");
    EXPECT_FALSE(node["is_array"].get<bool>());
    EXPECT_EQ(node["field"], "");
}

/// @brief it should read the dimensions and every element of an array variable.
TEST_F(TestScanTask, testReadNodeArray) {
    const auto status = run_command(
        ctx,
        task,
        conn_pool,
        store,
        READ_NODE_CMD_TYPE,
        4840,
        {{"node_id", "NS=1;S=TestDoubleArray"}}
    );
    ASSERT_EQ(status.variant, synnax::status::VARIANT_SUCCESS) << status.description;
    const x::json::json data = status.details.data.value();
    EXPECT_EQ(data["value_rank"], UA_VALUERANK_ONE_DIMENSION);
    ASSERT_EQ(data["array_dimensions"].size(), 1);
    const auto value = x::json::json::parse(data["value"].get<std::string>());
    ASSERT_TRUE(value.is_array());
    EXPECT_EQ(value.size(), data["array_dimensions"][0].get<size_t>());
    EXPECT_TRUE(data["node"]["is_array"].get<bool>());
}

/// @brief it should read the declared input arguments of a method.
TEST_F(TestScanTask, testReadNodeMethod) {
    const auto pump_server = start_motor_server();
    const auto status = run_command(
        ctx,
        task,
        conn_pool,
        store,
        READ_NODE_CMD_TYPE,
        4881,
        {{"node_id", "NS=1;S=Pump.Configure"}}
    );
    ASSERT_EQ(status.variant, synnax::status::VARIANT_SUCCESS) << status.description;
    const x::json::json data = status.details.data.value();
    EXPECT_EQ(data["node_class"], "Method");
    EXPECT_EQ(
        data["input_arguments"],
        x::json::json::array({
            {{"name", "mode"}, {"type_name", "String"}, {"data_type", "string"}},
            {{"name", "speed"}, {"type_name", "Double"}, {"data_type", "float64"}},
        })
    );
}

/// @brief it should read a node that is not a variable without a value.
TEST_F(TestScanTask, testReadNodeObject) {
    const auto status = run_command(
        ctx,
        task,
        conn_pool,
        store,
        READ_NODE_CMD_TYPE,
        4840,
        {{"node_id", "i=85"}}
    );
    ASSERT_EQ(status.variant, synnax::status::VARIANT_SUCCESS) << status.description;
    const x::json::json data = status.details.data.value();
    EXPECT_EQ(data["node_class"], "Object");
    EXPECT_EQ(data["browse_name"], "Objects");
    EXPECT_FALSE(data.contains("value"));
    EXPECT_FALSE(data.contains("type_name"));
    EXPECT_EQ(data["node"]["node_class"], "Object");
}

/// @brief it should decode a structure value that the client does not know, and read
/// one member of it.
TEST_F(TestScanTask, testReadNodeStructure) {
    const auto motor_server = start_motor_server();
    const auto whole = run_command(
        ctx,
        task,
        conn_pool,
        store,
        READ_NODE_CMD_TYPE,
        4881,
        {{"node_id", "NS=1;S=TestMotor"}}
    );
    ASSERT_EQ(whole.variant, synnax::status::VARIANT_SUCCESS) << whole.description;
    const x::json::json motor = whole.details.data.value();
    EXPECT_EQ(motor["type_name"], "Motor");
    const auto value = x::json::json::parse(motor["value"].get<std::string>());
    EXPECT_DOUBLE_EQ(value["Speed"].get<double>(), 1500.0);
    EXPECT_TRUE(value["Running"].get<bool>());
    EXPECT_DOUBLE_EQ(value["Position"]["Y"].get<double>(), 2.5);

    const auto member = run_command(
        ctx,
        task,
        conn_pool,
        store,
        READ_NODE_CMD_TYPE,
        4881,
        {{"node_id", "NS=1;S=TestMotor"}, {"field", "Position.Y"}}
    );
    ASSERT_EQ(member.variant, synnax::status::VARIANT_SUCCESS) << member.description;
    const x::json::json y = member.details.data.value();
    EXPECT_EQ(y["type_name"], "Float");
    EXPECT_DOUBLE_EQ(std::stod(y["value"].get<std::string>()), 2.5);
    EXPECT_EQ(y["node"]["name"], "Y");
    EXPECT_EQ(y["node"]["field"], "Position.Y");
    EXPECT_EQ(y["node"]["data_type"], "float32");
    motor_server->stop();
}

/// @brief it should fail to read a node that the server does not have.
TEST_F(TestScanTask, testReadNodeUnknown) {
    const auto status = run_command(
        ctx,
        task,
        conn_pool,
        store,
        READ_NODE_CMD_TYPE,
        4840,
        {{"node_id", "NS=1;S=Missing"}}
    );
    EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(status.message, "Failed to read node");
    EXPECT_EQ(
        status.description,
        "The node id refers to a node that does not exist in the server address space."
    );
}
}
