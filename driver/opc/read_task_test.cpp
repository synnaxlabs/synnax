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
#include "driver/opc/read_task.h"
#include "driver/pipeline/mock/pipeline.h"

class TestReadTask : public ::testing::Test {
protected:
    synnax::Task task;
    std::unique_ptr<opc::ReadTaskConfig> cfg;
    std::shared_ptr<task::MockContext> ctx;
    std::shared_ptr<pipeline::mock::WriterFactory> mock_factory;
    std::unique_ptr<mock::Server> server;
    synnax::Channel index_channel;
    synnax::Channel data_channel;

    void SetUp() override {
        auto client = std::make_shared<synnax::Synnax>(new_test_client());

        this->index_channel = ASSERT_NIL_P(
            client->channels.create("index", telem::TIMESTAMP_T, 0, true)
        );

        this->data_channel = ASSERT_NIL_P(
            client->channels
                .create("test", telem::FLOAT32_T, this->index_channel.key, false)
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

        mock::ServerChannel server_ch{
            .ns = 1,
            .node = "test",
            .ch = this->data_channel
        };


        mock::ServerConfig cfg{.channels = {server_ch}};


        json task_cfg{
            {"data_saving", true},
            {"device", dev.key},
            {"channels",
             json::array(
                 {{{"key", "NS=2;I=8"},
                   {"name", "test"},
                   {"node_name", "test"},
                   {"node_id", "NS=1;S=test"},
                   {"channel", this->data_channel.key},
                   {"enabled", true},
                   {"use_as_index", false},
                   {"data_type", "int64"}}}
             )},
            {"sample_rate", 50},
            {"array_mode", false},
            {"stream_rate", 25}
        };

        task = synnax::Task(rack.key, "my_task", "opc_read", "");

        auto p = xjson::Parser(task_cfg);
        this->cfg = std::make_unique<opc::ReadTaskConfig>(client, p);


        ctx = std::make_shared<task::MockContext>(client);
        mock_factory = std::make_shared<pipeline::mock::WriterFactory>();

        server = std::make_unique<mock::Server>(mock::ServerConfig(cfg));
        server->start();
    }

    std::unique_ptr<common::ReadTask> create_task() {
        auto client = ASSERT_EVENTUALLY_NIL_P_WITH_TIMEOUT(
            util::connect(cfg->conn, "write_task_test"),
            (5 * telem::SECOND).chrono(),
            (250 * telem::MILLISECOND).chrono()
        );
        return std::make_unique<common::ReadTask>(
            task,
            ctx,
            breaker::default_config(task.name),
            std::make_unique<opc::UnaryReadTaskSource>(client, std::move(*cfg)),
            mock_factory
        );
    }
};

TEST_F(TestReadTask, testBasicReadTask) {
    auto start = telem::TimeStamp::now();
    const auto rt = create_task();
    rt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->states.size(), 1);
    const auto first_state = ctx->states[0];
    EXPECT_EQ(first_state.key, "start_cmd");
    EXPECT_EQ(first_state.task, task.key);
    EXPECT_EQ(first_state.variant, status::variant::SUCCESS);
    EXPECT_EQ(first_state.details["message"], "Task started successfully");
    ASSERT_EVENTUALLY_GE(mock_factory->writer_opens, 1);
    ASSERT_EVENTUALLY_GE(mock_factory->writes->size(), 1);
    rt->stop("stop_cmd", true);
    const auto second_state = ctx->states[1];
    EXPECT_EQ(second_state.key, "stop_cmd");
    EXPECT_EQ(second_state.task, task.key);
    EXPECT_EQ(second_state.variant, status::variant::SUCCESS);
    EXPECT_EQ(second_state.details["message"], "Task stopped successfully");
    auto &fr = mock_factory->writes->at(0);
    ASSERT_EQ(fr.size(), 2);
    ASSERT_EQ(fr.length(), 2);
    ASSERT_EQ(fr.contains(this->data_channel.key), true);
    ASSERT_EQ(fr.contains(this->index_channel.key), true);
    ASSERT_EQ(fr.at<float>(this->data_channel.key, 0), 5);
    ASSERT_GE(fr.at<telem::TimeStamp>(this->index_channel.key, 0), start);
}
