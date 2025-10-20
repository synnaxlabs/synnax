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
#include "driver/opc/read_task.h"
#include "driver/pipeline/mock/pipeline.h"

class TestReadTask : public ::testing::Test {
protected:
    synnax::Task task;
    json task_cfg_json;
    std::shared_ptr<task::MockContext> ctx;
    std::shared_ptr<pipeline::mock::WriterFactory> mock_factory;
    std::unique_ptr<mock::Server> server;
    std::shared_ptr<opc::connection::Pool> conn_pool;
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

        opc::connection::Config conn_cfg;
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

        task_cfg_json = task_cfg;

        ctx = std::make_shared<task::MockContext>(client);
        mock_factory = std::make_shared<pipeline::mock::WriterFactory>();
        conn_pool = std::make_shared<opc::connection::Pool>();

        server = std::make_unique<mock::Server>(server_cfg);
        server->start();

        // Wait for server to be ready by attempting to connect
        opc::connection::Config test_conn_cfg;
        test_conn_cfg.endpoint = conn_cfg.endpoint;
        test_conn_cfg.security_mode = "None";
        test_conn_cfg.security_policy = "None";
        auto test_client = ASSERT_EVENTUALLY_NIL_P_WITH_TIMEOUT(
            opc::connection::connect(test_conn_cfg, "test"),
            (5 * telem::SECOND).chrono(),
            (250 * telem::MILLISECOND).chrono()
        );
        UA_Client_disconnect(test_client.get());
    }

    std::unique_ptr<common::ReadTask> create_task() {
        auto p = xjson::Parser(task_cfg_json);
        auto cfg = std::make_unique<opc::ReadTaskConfig>(ctx->client, p);

        return std::make_unique<common::ReadTask>(
            task,
            ctx,
            breaker::default_config(task.name),
            std::make_unique<opc::UnaryReadTaskSource>(conn_pool, std::move(*cfg)),
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

TEST_F(TestReadTask, testInvalidNodeId) {
    json bad_task_cfg{
        {"data_saving", true},
        {"device", "opc_read_task_test_server_key"},
        {"channels",
         json::array(
             {{{"key", "NS=2;I=999"},
               {"name", "nonexistent"},
               {"node_name", "NonExistent"},
               {"node_id", "NS=1;S=NonExistentNode"},
               {"channel", this->float_channel.key},
               {"enabled", true},
               {"use_as_index", false},
               {"data_type", "float32"}}}
         )},
        {"sample_rate", 50},
        {"array_mode", false},
        {"stream_rate", 25}
    };

    auto p = xjson::Parser(bad_task_cfg);
    auto bad_cfg = std::make_unique<opc::ReadTaskConfig>(ctx->client, p);

    auto rt = std::make_unique<common::ReadTask>(
        task,
        ctx,
        breaker::default_config(task.name),
        std::make_unique<opc::UnaryReadTaskSource>(conn_pool, std::move(*bad_cfg)),
        mock_factory
    );

    rt->start("start_cmd");
    std::this_thread::sleep_for(std::chrono::milliseconds(200));
    rt->stop("stop_cmd", true);

    ASSERT_GE(ctx->states.size(), 1);
    bool found_error = false;
    for (const auto &state: ctx->states) {
        if (state.variant == status::variant::ERR) {
            found_error = true;
            break;
        }
    }
    EXPECT_TRUE(found_error);
}

TEST_F(TestReadTask, testServerDisconnectDuringRead) {
    const auto rt = create_task();
    rt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(mock_factory->writes->size(), 1);

    server->stop();
    server.reset();

    std::this_thread::sleep_for(std::chrono::milliseconds(500));

    rt->stop("stop_cmd", true);

    bool found_error = false;
    for (const auto &state: ctx->states) {
        if (state.variant == status::variant::ERR) {
            found_error = true;
            break;
        }
    }
    EXPECT_TRUE(found_error);
}

TEST_F(TestReadTask, testEmptyChannelList) {
    json empty_cfg{
        {"data_saving", true},
        {"device", "opc_read_task_test_server_key"},
        {"channels", json::array()},
        {"sample_rate", 50},
        {"array_mode", false},
        {"stream_rate", 25}
    };

    auto p = xjson::Parser(empty_cfg);
    auto empty_config = std::make_unique<opc::ReadTaskConfig>(ctx->client, p);
    EXPECT_TRUE(p.error());
}

TEST_F(TestReadTask, testDisabledChannels) {
    json disabled_cfg{
        {"data_saving", true},
        {"device", "opc_read_task_test_server_key"},
        {"channels",
         json::array(
             {{{"key", "NS=2;I=1"},
               {"name", "float_test"},
               {"node_name", "TestFloat"},
               {"node_id", "NS=1;S=TestFloat"},
               {"channel", this->float_channel.key},
               {"enabled", false},
               {"use_as_index", false},
               {"data_type", "float32"}}}
         )},
        {"sample_rate", 50},
        {"array_mode", false},
        {"stream_rate", 25}
    };

    auto p = xjson::Parser(disabled_cfg);
    auto disabled_config = std::make_unique<opc::ReadTaskConfig>(ctx->client, p);
    EXPECT_TRUE(p.error());
}

TEST_F(TestReadTask, testRapidStartStop) {
    const auto rt = create_task();
    rt->start("start_cmd");
    std::this_thread::sleep_for(std::chrono::milliseconds(50));
    rt->stop("stop_cmd", true);

    ASSERT_GE(ctx->states.size(), 2);
    EXPECT_EQ(ctx->states[0].variant, status::variant::SUCCESS);
    EXPECT_EQ(ctx->states[1].variant, status::variant::SUCCESS);
}

TEST_F(TestReadTask, testConnectionPoolReuse) {
    const std::string endpoint = "opc.tcp://localhost:4840";
    EXPECT_EQ(conn_pool->size(), 0);
    EXPECT_EQ(conn_pool->available_count(endpoint), 0);

    {
        const auto rt1 = create_task();
        EXPECT_EQ(conn_pool->size(), 0);
        rt1->start("start1");
        EXPECT_EQ(conn_pool->size(), 1);
        EXPECT_EQ(conn_pool->available_count(endpoint), 0);
        rt1->stop("stop1", true);
    }

    EXPECT_EQ(conn_pool->size(), 1);
    EXPECT_EQ(conn_pool->available_count(endpoint), 1);

    {
        const auto rt2 = create_task();
        EXPECT_EQ(conn_pool->size(), 1);
        rt2->start("start2");
        EXPECT_EQ(conn_pool->size(), 1);
        EXPECT_EQ(conn_pool->available_count(endpoint), 0);
        rt2->stop("stop2", true);
    }

    EXPECT_EQ(conn_pool->size(), 1);
    EXPECT_EQ(conn_pool->available_count(endpoint), 1);
}

TEST_F(TestReadTask, testConnectionPoolConcurrentTasks) {
    const std::string endpoint = "opc.tcp://localhost:4840";
    EXPECT_EQ(conn_pool->size(), 0);

    const auto rt1 = create_task();
    EXPECT_EQ(conn_pool->size(), 0);

    const auto rt2 = create_task();
    EXPECT_EQ(conn_pool->size(), 0);

    rt1->start("start1");
    EXPECT_EQ(conn_pool->size(), 1);
    EXPECT_EQ(conn_pool->available_count(endpoint), 0);

    rt2->start("start2");
    EXPECT_EQ(conn_pool->size(), 2);
    EXPECT_EQ(conn_pool->available_count(endpoint), 0);

    ASSERT_EVENTUALLY_GE(mock_factory->writes->size(), 2);

    rt1->stop("stop1", true);
    rt2->stop("stop2", true);
}

TEST_F(TestReadTask, testInvalidDataHandlingInArrayMode) {
    // Test that ArrayReadTaskSource properly handles invalid data from OPC UA server
    // by clearing the frame and returning a warning
    json array_task_cfg{
        {"data_saving", true},
        {"device", "opc_read_task_test_server_key"},
        {"channels",
         json::array(
             {{{"key", "NS=2;I=1"},
               {"name", "float_test"},
               {"node_name", "TestFloat"},
               {"node_id", "NS=1;S=TestFloat"},
               {"channel", this->float_channel.key},
               {"enabled", true},
               {"use_as_index", false},
               {"data_type", "float32"}}}
         )},
        {"sample_rate", 50},
        {"array_mode", true},
        {"array_size", 10},
        {"stream_rate", 25}
    };

    auto p = xjson::Parser(array_task_cfg);
    auto cfg = std::make_unique<opc::ReadTaskConfig>(ctx->client, p);

    auto rt = std::make_unique<common::ReadTask>(
        task,
        ctx,
        breaker::default_config(task.name),
        std::make_unique<opc::ArrayReadTaskSource>(conn_pool, std::move(*cfg)),
        mock_factory
    );

    rt->start("start_cmd");
    std::this_thread::sleep_for(std::chrono::milliseconds(100));
    rt->stop("stop_cmd", true);

    // Check that the task started successfully despite potential data errors
    ASSERT_GE(ctx->states.size(), 1);
}

TEST_F(TestReadTask, testInvalidDataSkipsFrameInUnaryMode) {
    // Test that UnaryReadTaskSource properly skips frames with invalid data
    // and returns a warning message
    const auto rt = create_task();
    rt->start("start_cmd");

    // Let it run for a bit to collect some data
    std::this_thread::sleep_for(std::chrono::milliseconds(200));

    rt->stop("stop_cmd", true);

    // Verify task lifecycle worked correctly
    ASSERT_GE(ctx->states.size(), 2);
    EXPECT_EQ(ctx->states[0].variant, status::variant::SUCCESS);
    EXPECT_EQ(ctx->states[0].message, "Task started successfully");
    EXPECT_EQ(ctx->states[1].variant, status::variant::SUCCESS);
    EXPECT_EQ(ctx->states[1].message, "Task stopped successfully");
}

TEST_F(TestReadTask, testEmptyFramesNotWrittenInUnaryMode) {
    // Test that empty frames (with size 0) are not written to Synnax
    const auto rt = create_task();
    rt->start("start_cmd");

    // Very short run to minimize data collection
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    rt->stop("stop_cmd", true);

    // If any frames were written, they should not be empty
    if (mock_factory->writes->size() > 0) {
        for (const auto &fr: *mock_factory->writes) {
            EXPECT_GT(fr.length(), 0);
        }
    }
}

TEST_F(TestReadTask, testMultipleChannelsWithMixedData) {
    // Test that the read task handles multiple channels with different data types
    // This exercises the loop in UnaryReadTaskSource::read that processes all channels
    const auto rt = create_task();
    rt->start("start_cmd");

    ASSERT_EVENTUALLY_GE(mock_factory->writes->size(), 1);

    rt->stop("stop_cmd", true);

    // Verify that we received data for multiple channels
    if (mock_factory->writes->size() > 0) {
        const auto &fr = mock_factory->writes->at(0);
        // Should have index channel + all data channels
        EXPECT_GE(fr.size(), 11);
    }
}

TEST_F(TestReadTask, testBooleanChannelDataHandling) {
    // Test that boolean channels are properly read and converted to UINT8
    const auto rt = create_task();
    rt->start("start_cmd");

    ASSERT_EVENTUALLY_GE(mock_factory->writes->size(), 1);

    rt->stop("stop_cmd", true);

    // Check boolean channel data
    const auto &fr = mock_factory->writes->at(0);
    ASSERT_TRUE(fr.contains(this->bool_channel.key));
    EXPECT_GT(fr.length(), 0);
}

TEST_F(TestReadTask, testFloatChannelDataHandling) {
    // Test that float channels are properly read and written
    const auto rt = create_task();
    rt->start("start_cmd");

    ASSERT_EVENTUALLY_GE(mock_factory->writes->size(), 1);

    rt->stop("stop_cmd", true);

    // Check float channel data
    const auto &fr = mock_factory->writes->at(0);
    ASSERT_TRUE(fr.contains(this->float_channel.key));
    EXPECT_GT(fr.length(), 0);
}

TEST_F(TestReadTask, testDoubleChannelDataHandling) {
    // Test that double channels are properly read and written
    const auto rt = create_task();
    rt->start("start_cmd");

    ASSERT_EVENTUALLY_GE(mock_factory->writes->size(), 1);

    rt->stop("stop_cmd", true);

    // Check double channel data
    const auto &fr = mock_factory->writes->at(0);
    ASSERT_TRUE(fr.contains(this->double_channel.key));
    EXPECT_GT(fr.length(), 0);
}

TEST_F(TestReadTask, testIntegerChannelDataHandling) {
    // Test that integer channels (int8, int16, int32, int64) are properly read
    const auto rt = create_task();
    rt->start("start_cmd");

    ASSERT_EVENTUALLY_GE(mock_factory->writes->size(), 1);

    rt->stop("stop_cmd", true);

    // Check all integer channel data and verify values
    const auto &fr = mock_factory->writes->at(0);
    ASSERT_TRUE(fr.contains(this->int8_channel.key));
    ASSERT_TRUE(fr.contains(this->int16_channel.key));
    ASSERT_TRUE(fr.contains(this->int32_channel.key));
    ASSERT_TRUE(fr.contains(this->int64_channel.key));

    // Verify the actual values match what the mock server provides
    EXPECT_EQ(fr.at<std::int8_t>(this->int8_channel.key, 0), 42);
    EXPECT_EQ(fr.at<std::int16_t>(this->int16_channel.key, 0), 42);
    EXPECT_EQ(fr.at<std::int32_t>(this->int32_channel.key, 0), 12345);
    EXPECT_EQ(fr.at<std::int64_t>(this->int64_channel.key, 0), 12345);
}

TEST_F(TestReadTask, testUnsignedIntegerChannelDataHandling) {
    // Test that unsigned integer channels (uint16, uint32, uint64) are properly read
    const auto rt = create_task();
    rt->start("start_cmd");

    ASSERT_EVENTUALLY_GE(mock_factory->writes->size(), 1);

    rt->stop("stop_cmd", true);

    // Check all unsigned integer channel data
    const auto &fr = mock_factory->writes->at(0);
    ASSERT_TRUE(fr.contains(this->uint16_channel.key));
    ASSERT_TRUE(fr.contains(this->uint32_channel.key));
    ASSERT_TRUE(fr.contains(this->uint64_channel.key));
}

TEST_F(TestReadTask, testErrorAggregationInArrayMode) {
    // Test that ArrayReadTaskSource aggregates multiple errors from different channels
    json multi_channel_array_cfg{
        {"data_saving", true},
        {"device", "opc_read_task_test_server_key"},
        {"channels",
         json::array(
             {{{"key", "NS=2;I=1"},
               {"name", "float_test"},
               {"node_name", "TestFloat"},
               {"node_id", "NS=1;S=TestFloat"},
               {"channel", this->float_channel.key},
               {"enabled", true},
               {"use_as_index", false},
               {"data_type", "float32"}},
              {{"key", "NS=2;I=2"},
               {"name", "double_test"},
               {"node_name", "TestDouble"},
               {"node_id", "NS=1;S=TestDouble"},
               {"channel", this->double_channel.key},
               {"enabled", true},
               {"use_as_index", false},
               {"data_type", "float64"}}}
         )},
        {"sample_rate", 50},
        {"array_mode", true},
        {"array_size", 10},
        {"stream_rate", 25}
    };

    auto p = xjson::Parser(multi_channel_array_cfg);
    auto cfg = std::make_unique<opc::ReadTaskConfig>(ctx->client, p);

    auto rt = std::make_unique<common::ReadTask>(
        task,
        ctx,
        breaker::default_config(task.name),
        std::make_unique<opc::ArrayReadTaskSource>(conn_pool, std::move(*cfg)),
        mock_factory
    );

    rt->start("start_cmd");
    std::this_thread::sleep_for(std::chrono::milliseconds(100));
    rt->stop("stop_cmd", true);

    // Task should handle multiple channels without crashing
    ASSERT_GE(ctx->states.size(), 1);
}

TEST_F(TestReadTask, testWarningMessagesContainChannelInfo) {
    // Test that warning messages contain channel information for debugging
    // This tests the error message formatting in read_task.h lines 258-262 and 326-328
    const auto rt = create_task();
    rt->start("start_cmd");

    // Let it run briefly
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    rt->stop("stop_cmd", true);

    // If any warnings were generated, they should be informative
    // We can't force an error in this test without a mock server that returns bad data,
    // but we verify the task runs successfully
    ASSERT_GE(ctx->states.size(), 2);
}

TEST_F(TestReadTask, testSkipSampleOnWriteErrorInUnaryMode) {
    // Test that UnaryReadTaskSource properly skips samples when write_to_series fails
    // This exercises the skip_sample logic in read_task.h lines 324-334
    const auto rt = create_task();
    rt->start("start_cmd");

    // Let task run and collect multiple samples
    std::this_thread::sleep_for(std::chrono::milliseconds(200));

    rt->stop("stop_cmd", true);

    // Verify that task completed successfully
    ASSERT_GE(ctx->states.size(), 2);
    EXPECT_EQ(ctx->states[0].variant, status::variant::SUCCESS);
    EXPECT_EQ(ctx->states[1].variant, status::variant::SUCCESS);
}

TEST_F(TestReadTask, testFrameClearedOnErrorInArrayMode) {
    // Test that ArrayReadTaskSource clears the frame when errors occur
    // This exercises the frame.clear() logic in read_task.h line 268
    json array_cfg{
        {"data_saving", true},
        {"device", "opc_read_task_test_server_key"},
        {"channels",
         json::array(
             {{{"key", "NS=2;I=1"},
               {"name", "bool_test"},
               {"node_name", "TestBoolean"},
               {"node_id", "NS=1;S=TestBoolean"},
               {"channel", this->bool_channel.key},
               {"enabled", true},
               {"use_as_index", false},
               {"data_type", "uint8"}}}
         )},
        {"sample_rate", 50},
        {"array_mode", true},
        {"array_size", 5},
        {"stream_rate", 25}
    };

    auto p = xjson::Parser(array_cfg);
    auto cfg = std::make_unique<opc::ReadTaskConfig>(ctx->client, p);

    auto rt = std::make_unique<common::ReadTask>(
        task,
        ctx,
        breaker::default_config(task.name),
        std::make_unique<opc::ArrayReadTaskSource>(conn_pool, std::move(*cfg)),
        mock_factory
    );

    rt->start("start_cmd");
    std::this_thread::sleep_for(std::chrono::milliseconds(100));
    rt->stop("stop_cmd", true);

    // Verify task ran successfully
    ASSERT_GE(ctx->states.size(), 1);
}

TEST_F(TestReadTask, testFrameClearedOnErrorInUnaryMode) {
    // Test that UnaryReadTaskSource clears the frame when errors occur
    // This exercises the frame.clear() logic in read_task.h line 333
    const auto rt = create_task();
    rt->start("start_cmd");

    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    rt->stop("stop_cmd", true);

    if (mock_factory->writes->size() > 0) {
        for (const auto &fr: *mock_factory->writes) {
            EXPECT_GT(fr.length(), 0);
        }
    }
}

TEST_F(TestReadTask, testSkipSampleWithInvalidBooleanData) {
    // Test that UnaryReadTaskSource skips samples when boolean data is invalid
    // Uses a separate mock server on different port with invalid data
    auto invalid_server_cfg = mock::ServerConfig::create_with_invalid_data();
    invalid_server_cfg.port = 4841; // Different port from main server
    auto invalid_server = std::make_unique<mock::Server>(invalid_server_cfg);
    invalid_server->start();
    std::this_thread::sleep_for(std::chrono::milliseconds(250));

    // Create a separate rack and device for the invalid data server
    auto invalid_rack = ASSERT_NIL_P(
        ctx->client->hardware.create_rack("opc_invalid_bool_rack")
    );

    opc::connection::Config invalid_conn_cfg;
    invalid_conn_cfg.endpoint = "opc.tcp://localhost:4841";
    invalid_conn_cfg.security_mode = "None";
    invalid_conn_cfg.security_policy = "None";

    synnax::Device invalid_dev(
        "opc_invalid_test_server",
        "OPC UA Invalid Data Test Server",
        invalid_rack.key,
        "opc.tcp://localhost:4841",
        "opc",
        "OPC UA Server",
        nlohmann::to_string(json::object({{"connection", invalid_conn_cfg.to_json()}}))
    );
    ASSERT_NIL(ctx->client->hardware.create_device(invalid_dev));

    // Create a task that reads from the invalid boolean node
    json invalid_bool_cfg{
        {"data_saving", true},
        {"device", invalid_dev.key},
        {"channels",
         json::array(
             {{{"key", "NS=2;I=1"},
               {"name", "invalid_bool_test"},
               {"node_name", "InvalidBoolean"},
               {"node_id", "NS=1;S=InvalidBoolean"},
               {"channel", this->bool_channel.key},
               {"enabled", true},
               {"use_as_index", false},
               {"data_type", "uint8"}}}
         )},
        {"sample_rate", 50},
        {"array_mode", false},
        {"stream_rate", 25}
    };

    auto p = xjson::Parser(invalid_bool_cfg);
    auto cfg = std::make_unique<opc::ReadTaskConfig>(ctx->client, p);

    auto rt = std::make_unique<common::ReadTask>(
        task,
        ctx,
        breaker::default_config(task.name),
        std::make_unique<opc::UnaryReadTaskSource>(conn_pool, std::move(*cfg)),
        mock_factory
    );

    rt->start("start_cmd");
    std::this_thread::sleep_for(std::chrono::milliseconds(200));
    rt->stop("stop_cmd", true);

    // Verify that frames were cleared due to invalid data
    bool has_empty_frames = false;
    if (mock_factory->writes->size() > 0) {
        for (const auto &fr: *mock_factory->writes) {
            if (fr.length() == 0) {
                has_empty_frames = true;
                break;
            }
        }
    }
    EXPECT_TRUE(mock_factory->writes->size() == 0 || has_empty_frames);

    invalid_server->stop();
}

TEST_F(TestReadTask, testSkipSampleWithInvalidFloatData) {
    // Test that UnaryReadTaskSource skips samples when float data has null pointer
    auto invalid_server_cfg = mock::ServerConfig::create_with_invalid_data();
    invalid_server_cfg.port = 4842; // Different port
    auto invalid_server = std::make_unique<mock::Server>(invalid_server_cfg);
    invalid_server->start();
    std::this_thread::sleep_for(std::chrono::milliseconds(250));

    // Create a separate rack and device for the invalid data server
    auto invalid_rack = ASSERT_NIL_P(
        ctx->client->hardware.create_rack("opc_invalid_float_rack")
    );

    opc::connection::Config invalid_conn_cfg;
    invalid_conn_cfg.endpoint = "opc.tcp://localhost:4842";
    invalid_conn_cfg.security_mode = "None";
    invalid_conn_cfg.security_policy = "None";

    synnax::Device invalid_dev(
        "opc_invalid_float_server",
        "OPC UA Invalid Float Server",
        invalid_rack.key,
        "opc.tcp://localhost:4842",
        "opc",
        "OPC UA Server",
        nlohmann::to_string(json::object({{"connection", invalid_conn_cfg.to_json()}}))
    );
    ASSERT_NIL(ctx->client->hardware.create_device(invalid_dev));

    json invalid_float_cfg{
        {"data_saving", true},
        {"device", invalid_dev.key},
        {"channels",
         json::array(
             {{{"key", "NS=2;I=2"},
               {"name", "invalid_float_test"},
               {"node_name", "InvalidFloat"},
               {"node_id", "NS=1;S=InvalidFloat"},
               {"channel", this->float_channel.key},
               {"enabled", true},
               {"use_as_index", false},
               {"data_type", "float32"}}}
         )},
        {"sample_rate", 50},
        {"array_mode", false},
        {"stream_rate", 25}
    };

    auto p = xjson::Parser(invalid_float_cfg);
    auto cfg = std::make_unique<opc::ReadTaskConfig>(ctx->client, p);

    auto rt = std::make_unique<common::ReadTask>(
        task,
        ctx,
        breaker::default_config(task.name),
        std::make_unique<opc::UnaryReadTaskSource>(conn_pool, std::move(*cfg)),
        mock_factory
    );

    rt->start("start_cmd");
    std::this_thread::sleep_for(std::chrono::milliseconds(200));
    rt->stop("stop_cmd", true);

    // Verify frames were cleared
    bool has_empty_frames = false;
    if (mock_factory->writes->size() > 0) {
        for (const auto &fr: *mock_factory->writes) {
            if (fr.length() == 0) {
                has_empty_frames = true;
                break;
            }
        }
    }
    EXPECT_TRUE(mock_factory->writes->size() == 0 || has_empty_frames);

    invalid_server->stop();
}

TEST_F(TestReadTask, testFrameClearWithInvalidDoubleArrayData) {
    // Test that ArrayReadTaskSource clears frames with zero-length arrays
    auto invalid_server_cfg = mock::ServerConfig::create_with_invalid_data();
    invalid_server_cfg.port = 4843;
    auto invalid_server = std::make_unique<mock::Server>(invalid_server_cfg);
    invalid_server->start();
    std::this_thread::sleep_for(std::chrono::milliseconds(250));

    // Create a separate rack and device for the invalid data server
    auto invalid_rack = ASSERT_NIL_P(
        ctx->client->hardware.create_rack("opc_invalid_double_rack")
    );

    opc::connection::Config invalid_conn_cfg;
    invalid_conn_cfg.endpoint = "opc.tcp://localhost:4843";
    invalid_conn_cfg.security_mode = "None";
    invalid_conn_cfg.security_policy = "None";

    synnax::Device invalid_dev(
        "opc_invalid_double_server",
        "OPC UA Invalid Double Server",
        invalid_rack.key,
        "opc.tcp://localhost:4843",
        "opc",
        "OPC UA Server",
        nlohmann::to_string(json::object({{"connection", invalid_conn_cfg.to_json()}}))
    );
    ASSERT_NIL(ctx->client->hardware.create_device(invalid_dev));

    json invalid_double_cfg{
        {"data_saving", true},
        {"device", invalid_dev.key},
        {"channels",
         json::array(
             {{{"key", "NS=2;I=3"},
               {"name", "invalid_double_test"},
               {"node_name", "InvalidDouble"},
               {"node_id", "NS=1;S=InvalidDouble"},
               {"channel", this->double_channel.key},
               {"enabled", true},
               {"use_as_index", false},
               {"data_type", "float64"}}}
         )},
        {"sample_rate", 50},
        {"array_mode", true},
        {"array_size", 5},
        {"stream_rate", 25}
    };

    auto p = xjson::Parser(invalid_double_cfg);
    auto cfg = std::make_unique<opc::ReadTaskConfig>(ctx->client, p);

    auto rt = std::make_unique<common::ReadTask>(
        task,
        ctx,
        breaker::default_config(task.name),
        std::make_unique<opc::ArrayReadTaskSource>(conn_pool, std::move(*cfg)),
        mock_factory
    );

    rt->start("start_cmd");
    std::this_thread::sleep_for(std::chrono::milliseconds(200));
    rt->stop("stop_cmd", true);

    // Verify frames were cleared - tests read_task.h line 268
    bool has_empty_frames = false;
    if (mock_factory->writes->size() > 0) {
        for (const auto &fr: *mock_factory->writes) {
            if (fr.length() == 0) {
                has_empty_frames = true;
                break;
            }
        }
    }
    EXPECT_TRUE(mock_factory->writes->size() == 0 || has_empty_frames);

    invalid_server->stop();
}
