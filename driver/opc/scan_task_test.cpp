// Copyright 2025 Synnax Labs, Inc.
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
#include "x/cpp/xtest/xtest.h"

#include "driver/opc/mock/server.h"
#include "driver/opc/opc.h"
#include "driver/opc/scan_task.h"

class TestScanTask : public ::testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> client;
    std::shared_ptr<task::MockContext> ctx;
    std::shared_ptr<opc::conn::Pool> conn_pool;
    std::unique_ptr<mock::Server> server;
    synnax::Task task;
    synnax::Rack rack;

    void SetUp() override {
        client = std::make_shared<synnax::Synnax>(new_test_client());
        ctx = std::make_shared<task::MockContext>(client);
        conn_pool = std::make_shared<opc::conn::Pool>();

        rack = ASSERT_NIL_P(client->hardware.create_rack("opc_scan_task_test_rack"));

        task = synnax::Task(rack.key, "OPC UA Scan Task Test", "opc_scan", "");

        auto server_cfg = mock::ServerConfig::create_default();
        server = std::make_unique<mock::Server>(server_cfg);
        server->start();

        // Wait for server to be ready by attempting to connect
        util::ConnectionConfig test_conn_cfg;
        test_conn_cfg.endpoint = "opc.tcp://localhost:4840";
        test_conn_cfg.security_mode = "None";
        test_conn_cfg.security_policy = "None";
        auto test_client = ASSERT_EVENTUALLY_NIL_P_WITH_TIMEOUT(
            util::connect(test_conn_cfg, "test"),
            (5 * telem::SECOND).chrono(),
            (250 * telem::MILLISECOND).chrono()
        );
        UA_Client_disconnect(test_client.get());
    }
};

TEST_F(TestScanTask, testBasicScan) {
    auto scan_task = std::make_unique<opc::ScanTask>(ctx, task, conn_pool);

    opc::conn::Config conn_cfg;
    conn_cfg.endpoint = "opc.tcp://localhost:4840";
    conn_cfg.security_mode = "None";
    conn_cfg.security_policy = "None";

    json scan_cmd{
        {"connection", conn_cfg.to_json()},
    };

    task::Command cmd(task.key, opc::SCAN_CMD_TYPE, scan_cmd);
    cmd.key = "scan_cmd";

    scan_task->exec(cmd);

    ASSERT_EVENTUALLY_GE(ctx->states.size(), 1);
    const auto &state = ctx->states[0];
    EXPECT_EQ(state.key, "scan_cmd");
    EXPECT_EQ(state.variant, status::variant::SUCCESS);

    auto data = state.details.data;
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

TEST_F(TestScanTask, testConnectionPooling) {
    auto scan_task = std::make_unique<opc::ScanTask>(ctx, task, conn_pool);

    opc::conn::Config conn_cfg;
    conn_cfg.endpoint = "opc.tcp://localhost:4840";
    conn_cfg.security_mode = "None";
    conn_cfg.security_policy = "None";

    json scan_cmd{
        {"connection", conn_cfg.to_json()},
    };

    task::Command cmd1(task.key, opc::SCAN_CMD_TYPE, scan_cmd);
    cmd1.key = "scan_cmd_1";

    scan_task->exec(cmd1);
    ASSERT_EVENTUALLY_GE(ctx->states.size(), 1);
    EXPECT_EQ(ctx->states[0].variant, status::variant::SUCCESS);

    task::Command cmd2(task.key, opc::SCAN_CMD_TYPE, scan_cmd);
    cmd2.key = "scan_cmd_2";

    scan_task->exec(cmd2);
    ASSERT_EVENTUALLY_GE(ctx->states.size(), 2);
    EXPECT_EQ(ctx->states[1].variant, status::variant::SUCCESS);
}

TEST_F(TestScanTask, testTestConnection) {
    auto scan_task = std::make_unique<opc::ScanTask>(ctx, task, conn_pool);

    opc::conn::Config conn_cfg;
    conn_cfg.endpoint = "opc.tcp://localhost:4840";
    conn_cfg.security_mode = "None";
    conn_cfg.security_policy = "None";

    json test_conn_cmd{
        {"connection", conn_cfg.to_json()},
    };

    task::Command cmd(task.key, opc::TEST_CONNECTION_CMD_TYPE, test_conn_cmd);
    cmd.key = "test_conn_cmd";

    scan_task->exec(cmd);

    ASSERT_EVENTUALLY_GE(ctx->states.size(), 1);
    const auto &state = ctx->states[0];
    EXPECT_EQ(state.key, "test_conn_cmd");
    EXPECT_EQ(state.variant, status::variant::SUCCESS);
    EXPECT_EQ(state.message, "Connection successful");
}

TEST_F(TestScanTask, testInvalidConnection) {
    auto scan_task = std::make_unique<opc::ScanTask>(ctx, task, conn_pool);

    opc::conn::Config conn_cfg;
    conn_cfg.endpoint = "opc.tcp://localhost:9999";
    conn_cfg.security_mode = "None";
    conn_cfg.security_policy = "None";

    json scan_cmd{
        {"connection", conn_cfg.to_json()},
    };

    task::Command cmd(task.key, opc::SCAN_CMD_TYPE, scan_cmd);
    cmd.key = "invalid_scan_cmd";

    scan_task->exec(cmd);

    ASSERT_EVENTUALLY_GE(ctx->states.size(), 1);
    const auto &state = ctx->states[0];
    EXPECT_EQ(state.key, "invalid_scan_cmd");
    EXPECT_EQ(state.variant, status::variant::ERR);
}
