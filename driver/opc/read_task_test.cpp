// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
    synnax::Channel bool_channel;
    synnax::Channel uint16_channel;
    synnax::Channel uint32_channel;
    synnax::Channel uint64_channel;
    synnax::Channel int8_channel;
    synnax::Channel int16_channel;
    synnax::Channel int32_channel;
    synnax::Channel int64_channel;
    synnax::Channel float_channel;
    synnax::Channel double_channel;

    void SetUp() override {
        auto client = std::make_shared<synnax::Synnax>(new_test_client());

        this->index_channel = ASSERT_NIL_P(
            client->channels.create("index", telem::TIMESTAMP_T, 0, true)
        );
        this->bool_channel = ASSERT_NIL_P(
            client->channels
                .create("bool_test", telem::UINT8_T, this->index_channel.key, false)
        );
        this->uint16_channel = ASSERT_NIL_P(
            client->channels
                .create("uint16_test", telem::UINT16_T, this->index_channel.key, false)
        );
        this->uint32_channel = ASSERT_NIL_P(
            client->channels
                .create("uint32_test", telem::UINT32_T, this->index_channel.key, false)
        );
        this->uint64_channel = ASSERT_NIL_P(
            client->channels
                .create("uint64_test", telem::UINT64_T, this->index_channel.key, false)
        );
        this->int8_channel = ASSERT_NIL_P(
            client->channels
                .create("int8_test", telem::INT8_T, this->index_channel.key, false)
        );
        this->int16_channel = ASSERT_NIL_P(
            client->channels
                .create("int16_test", telem::INT16_T, this->index_channel.key, false)
        );
        this->int32_channel = ASSERT_NIL_P(
            client->channels
                .create("int32_test", telem::INT32_T, this->index_channel.key, false)
        );
        this->int64_channel = ASSERT_NIL_P(
            client->channels
                .create("int64_test", telem::INT64_T, this->index_channel.key, false)
        );
        this->float_channel = ASSERT_NIL_P(
            client->channels
                .create("float_test", telem::FLOAT32_T, this->index_channel.key, false)
        );
        this->double_channel = ASSERT_NIL_P(
            client->channels
                .create("double_test", telem::FLOAT64_T, this->index_channel.key, false)
        );
        auto rack = ASSERT_NIL_P(
            client->hardware.create_rack("opc_read_task_test_rack")
        );

        util::ConnectionConfig conn_cfg;
        conn_cfg.endpoint = "opc.tcp://localhost:4840";
        conn_cfg.security_mode = "None";
        conn_cfg.security_policy = "None";

        synnax::Device dev(
            "opc_read_task_test_server_key",
            "OPC UA Read Task Test Server",
            rack.key,
            "opc.tcp://localhost:4840",
            "opc",
            "OPC UA Server",
            nlohmann::to_string(json::object({{"connection", conn_cfg.to_json()}}))
        );
        ASSERT_NIL(client->hardware.create_device(dev));

        // Use the comprehensive default server configuration
        auto server_cfg = mock::ServerConfig::create_default();

        json task_cfg{
            {"data_saving", true},
            {"device", dev.key},
            {"channels",
             json::array(
                 {{{"key", "NS=2;I=1"},
                   {"name", "bool_test"},
                   {"node_name", "TestBoolean"},
                   {"node_id", "NS=1;S=TestBoolean"},
                   {"channel", this->bool_channel.key},
                   {"enabled", true},
                   {"use_as_index", false},
                   {"data_type", "uint8"}},
                  {{"key", "NS=2;I=2"},
                   {"name", "uint16_test"},
                   {"node_name", "TestUInt16"},
                   {"node_id", "NS=1;S=TestUInt16"},
                   {"channel", this->uint16_channel.key},
                   {"enabled", true},
                   {"use_as_index", false},
                   {"data_type", "uint16"}},
                  {{"key", "NS=2;I=3"},
                   {"name", "uint32_test"},
                   {"node_name", "TestUInt32"},
                   {"node_id", "NS=1;S=TestUInt32"},
                   {"channel", this->uint32_channel.key},
                   {"enabled", true},
                   {"use_as_index", false},
                   {"data_type", "uint32"}},
                  {{"key", "NS=2;I=4"},
                   {"name", "uint64_test"},
                   {"node_name", "TestUInt64"},
                   {"node_id", "NS=1;S=TestUInt64"},
                   {"channel", this->uint64_channel.key},
                   {"enabled", true},
                   {"use_as_index", false},
                   {"data_type", "uint64"}},
                  {{"key", "NS=2;I=5"},
                   {"name", "int8_test"},
                   {"node_name", "TestInt8"},
                   {"node_id", "NS=1;S=TestInt8"},
                   {"channel", this->int8_channel.key},
                   {"enabled", true},
                   {"use_as_index", false},
                   {"data_type", "int8"}},
                  {{"key", "NS=2;I=6"},
                   {"name", "int16_test"},
                   {"node_name", "TestInt16"},
                   {"node_id", "NS=1;S=TestInt16"},
                   {"channel", this->int16_channel.key},
                   {"enabled", true},
                   {"use_as_index", false},
                   {"data_type", "int16"}},
                  {{"key", "NS=2;I=7"},
                   {"name", "int32_test"},
                   {"node_name", "TestInt32"},
                   {"node_id", "NS=1;S=TestInt32"},
                   {"channel", this->int32_channel.key},
                   {"enabled", true},
                   {"use_as_index", false},
                   {"data_type", "int32"}},
                  {{"key", "NS=2;I=8"},
                   {"name", "int64_test"},
                   {"node_name", "TestInt64"},
                   {"node_id", "NS=1;S=TestInt64"},
                   {"channel", this->int64_channel.key},
                   {"enabled", true},
                   {"use_as_index", false},
                   {"data_type", "int64"}},
                  {{"key", "NS=2;I=9"},
                   {"name", "float_test"},
                   {"node_name", "TestFloat"},
                   {"node_id", "NS=1;S=TestFloat"},
                   {"channel", this->float_channel.key},
                   {"enabled", true},
                   {"use_as_index", false},
                   {"data_type", "float32"}},
                  {{"key", "NS=2;I=10"},
                   {"name", "double_test"},
                   {"node_name", "TestDouble"},
                   {"node_id", "NS=1;S=TestDouble"},
                   {"channel", this->double_channel.key},
                   {"enabled", true},
                   {"use_as_index", false},
                   {"data_type", "float64"}}}
             )},
            {"sample_rate", 50},
            {"array_mode", false},
            {"stream_rate", 25}
        };

        task = synnax::Task(rack.key, "OPC UA Read Task Test", "opc_read", "");

        auto p = xjson::Parser(task_cfg);
        this->cfg = std::make_unique<opc::ReadTaskConfig>(client, p);


        ctx = std::make_shared<task::MockContext>(client);
        mock_factory = std::make_shared<pipeline::mock::WriterFactory>();

        server = std::make_unique<mock::Server>(server_cfg);
        server->start();
    }

    std::unique_ptr<common::ReadTask> create_task() {
        auto client = ASSERT_EVENTUALLY_NIL_P_WITH_TIMEOUT(
            util::connect(cfg->conn, "opc_read_task_test"),
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
    EXPECT_EQ(first_state.details.task, task.key);
    EXPECT_EQ(first_state.variant, status::variant::SUCCESS);
    EXPECT_EQ(first_state.message, "Task started successfully");
    ASSERT_EVENTUALLY_GE(mock_factory->writer_opens, 1);
    ASSERT_EVENTUALLY_GE(mock_factory->writes->size(), 1);
    rt->stop("stop_cmd", true);
    const auto second_state = ctx->states[1];
    EXPECT_EQ(second_state.key, "stop_cmd");
    EXPECT_EQ(second_state.details.task, task.key);
    EXPECT_EQ(second_state.variant, status::variant::SUCCESS);
    EXPECT_EQ(second_state.message, "Task stopped successfully");
    auto &fr = mock_factory->writes->at(0);
    ASSERT_EQ(fr.size(), 11);
    ASSERT_EQ(fr.length(), 2); // 50 sample rate, 25 stream rate = 50 / 25 = 2

    // Check that all channels are present
    ASSERT_EQ(fr.contains(this->index_channel.key), true);
    ASSERT_EQ(fr.contains(this->bool_channel.key), true);
    ASSERT_EQ(fr.contains(this->uint16_channel.key), true);
    ASSERT_EQ(fr.contains(this->uint32_channel.key), true);
    ASSERT_EQ(fr.contains(this->uint64_channel.key), true);
    ASSERT_EQ(fr.contains(this->int8_channel.key), true);
    ASSERT_EQ(fr.contains(this->int16_channel.key), true);
    ASSERT_EQ(fr.contains(this->int32_channel.key), true);
    ASSERT_EQ(fr.contains(this->int64_channel.key), true);
    ASSERT_EQ(fr.contains(this->float_channel.key), true);
    ASSERT_EQ(fr.contains(this->double_channel.key), true);

    // Check values match our mock server initial values
    ASSERT_EQ(fr.at<std::uint8_t>(this->bool_channel.key, 0), 1); // true = 1
    ASSERT_EQ(fr.at<std::uint16_t>(this->uint16_channel.key, 0), 42);
    ASSERT_EQ(fr.at<std::uint32_t>(this->uint32_channel.key, 0), 12345U);
    ASSERT_EQ(fr.at<std::uint64_t>(this->uint64_channel.key, 0), 12345U);
    ASSERT_EQ(fr.at<std::int8_t>(this->int8_channel.key, 0), 42);
    ASSERT_EQ(fr.at<std::int16_t>(this->int16_channel.key, 0), 42);
    ASSERT_EQ(fr.at<std::int32_t>(this->int32_channel.key, 0), 12345);
    ASSERT_EQ(fr.at<std::int64_t>(this->int64_channel.key, 0), 12345);
    ASSERT_NEAR(fr.at<float>(this->float_channel.key, 0), 3.14159f, 0.0001f);
    ASSERT_NEAR(fr.at<double>(this->double_channel.key, 0), 2.71828, 0.0001);
    ASSERT_GE(fr.at<telem::TimeStamp>(this->index_channel.key, 0), start);
}
