// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// ReSharper disable CppUseStructuredBinding

/// external
#include "gtest/gtest.h"
#include "nlohmann/json.hpp"

/// module
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/xtest/xtest.h"

/// internal
#include "driver/opc/mock/server.h"
#include "driver/opc/opc.h"
#include "driver/opc/write_task.h"
#include "driver/pipeline/mock/pipeline.h"

class TestWriteTask : public ::testing::Test {
protected:
    synnax::Task task;
    std::unique_ptr<opc::WriteTaskConfig> cfg;
    std::shared_ptr<task::MockContext> ctx;
    std::shared_ptr<pipeline::mock::StreamerFactory> mock_factory;
    std::unique_ptr<mock::Server> server;
    std::shared_ptr<UA_Client> ua_client;

    synnax::Channel cmd_channel;

    void SetUp() override {
        auto client = std::make_shared<synnax::Synnax>(new_test_client());

        this->cmd_channel = ASSERT_NIL_P(
            client->channels.create("cmd", telem::FLOAT32_T, true)
        );

        auto rack = ASSERT_NIL_P(client->hardware.create_rack("cat"));

        util::ConnectionConfig conn_cfg;
        conn_cfg.endpoint = "opc.tcp://0.0.0.0:4840";
        conn_cfg.security_mode = "None";
        conn_cfg.security_policy = "None";

        synnax::Device dev(
            "abc123",
            "my_device",
            rack.key,
            "dev1",
            "ni",
            "PXI-6255",
            nlohmann::to_string(json::object({{"connection", conn_cfg.to_json()}}))
        );
        ASSERT_NIL(client->hardware.create_device(dev));

        json task_cfg = {
            {"data_saving", true},
            {"device", dev.key},
            {"channels",
             json::array(
                 {{{"key", "NS=2;I=12"},
                   {"name", "test"},
                   {"node_name", "test"},
                   {"node_id", "NS=1;S=test"},
                   {"cmd_channel", this->cmd_channel.key},
                   {"enabled", true},
                   {"data_type", "int64"}}}
             )}
        };

        task = synnax::Task(rack.key, "my_task", "opc_write", "");

        auto p = xjson::Parser(task_cfg);
        this->cfg = std::make_unique<opc::WriteTaskConfig>(client, p);

        mock::ServerChannel server_ch{.ns = 1, .node = "test", .ch = this->cmd_channel};


        mock::ServerConfig cfg{.channels = {server_ch}};

        ctx = std::make_shared<task::MockContext>(client);
        auto reads = std::make_shared<std::vector<synnax::Frame>>();
        auto fr = synnax::Frame(1);
        fr.emplace(this->cmd_channel.key, telem::Series(1, telem::FLOAT32_T));
        reads->push_back(std::move(fr));
        mock_factory = pipeline::mock::simple_streamer_factory(
            {this->cmd_channel.key},
            reads
        );

        server = std::make_unique<mock::Server>(mock::ServerConfig(cfg));
        server->start();
    }

    std::unique_ptr<common::WriteTask> create_task() {
        this->ua_client = ASSERT_EVENTUALLY_NIL_P_WITH_TIMEOUT(
            util::connect(cfg->conn, "write_task_test"),
            (5 * telem::SECOND).chrono(),
            (250 * telem::MILLISECOND).chrono()
        );
        return std::make_unique<common::WriteTask>(
            task,
            ctx,
            breaker::default_config(task.name),
            std::make_unique<opc::WriteTaskSink>(ua_client, std::move(*cfg)),
            nullptr,
            mock_factory
        );
    }
};

TEST_F(TestWriteTask, testBasicWriteTask) {
    auto wt = create_task();
    wt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->states.size(), 1);
    const auto first_state = ctx->states[0];
    EXPECT_EQ(first_state.key, "start_cmd");
    EXPECT_EQ(first_state.task, task.key);
    EXPECT_EQ(first_state.variant, "success");
    EXPECT_EQ(first_state.details["message"], "Task started successfully");
    ASSERT_EVENTUALLY_GE(mock_factory->streamer_opens, 1);
    wt->stop("stop_cmd", true);
    const auto second_state = ctx->states[1];
    EXPECT_EQ(second_state.key, "stop_cmd");
    EXPECT_EQ(second_state.task, task.key);
    EXPECT_EQ(second_state.variant, "success");
    EXPECT_EQ(second_state.details["message"], "Task stopped successfully");
}
