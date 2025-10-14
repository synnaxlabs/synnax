// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <thread>

#include "glog/logging.h"
#include "gtest/gtest.h"
#include "nlohmann/json.hpp"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/xtest/xtest.h"

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
    std::shared_ptr<opc::conn::Pool> conn_pool;

    // Command channels for different data types
    synnax::Channel bool_cmd_channel;
    synnax::Channel uint16_cmd_channel;
    synnax::Channel uint32_cmd_channel;
    synnax::Channel uint64_cmd_channel;
    synnax::Channel int8_cmd_channel;
    synnax::Channel int16_cmd_channel;
    synnax::Channel int32_cmd_channel;
    synnax::Channel int64_cmd_channel;
    synnax::Channel float_cmd_channel;
    synnax::Channel double_cmd_channel;

    void SetUp() override {
        auto client = std::make_shared<synnax::Synnax>(new_test_client());

        // Create command channels for different OPC UA data types
        this->bool_cmd_channel = ASSERT_NIL_P(
            client->channels.create("bool_cmd", telem::UINT8_T, true)
        );
        this->uint16_cmd_channel = ASSERT_NIL_P(
            client->channels.create("uint16_cmd", telem::UINT16_T, true)
        );
        this->uint32_cmd_channel = ASSERT_NIL_P(
            client->channels.create("uint32_cmd", telem::UINT32_T, true)
        );
        this->uint64_cmd_channel = ASSERT_NIL_P(
            client->channels.create("uint64_cmd", telem::UINT64_T, true)
        );
        this->int8_cmd_channel = ASSERT_NIL_P(
            client->channels.create("int8_cmd", telem::INT8_T, true)
        );
        this->int16_cmd_channel = ASSERT_NIL_P(
            client->channels.create("int16_cmd", telem::INT16_T, true)
        );
        this->int32_cmd_channel = ASSERT_NIL_P(
            client->channels.create("int32_cmd", telem::INT32_T, true)
        );
        this->int64_cmd_channel = ASSERT_NIL_P(
            client->channels.create("int64_cmd", telem::INT64_T, true)
        );
        this->float_cmd_channel = ASSERT_NIL_P(
            client->channels.create("float_cmd", telem::FLOAT32_T, true)
        );
        this->double_cmd_channel = ASSERT_NIL_P(
            client->channels.create("double_cmd", telem::FLOAT64_T, true)
        );

        auto rack = ASSERT_NIL_P(client->hardware.create_rack("cat"));

        opc::conn::Config conn_cfg;
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
                 {{{"key", "NS=2;I=1"},
                   {"name", "bool_write_test"},
                   {"node_name", "TestBoolean"},
                   {"node_id", "NS=1;S=TestBoolean"},
                   {"cmd_channel", this->bool_cmd_channel.key},
                   {"enabled", true},
                   {"data_type", "uint8"}},
                  {{"key", "NS=2;I=2"},
                   {"name", "uint16_write_test"},
                   {"node_name", "TestUInt16"},
                   {"node_id", "NS=1;S=TestUInt16"},
                   {"cmd_channel", this->uint16_cmd_channel.key},
                   {"enabled", true},
                   {"data_type", "uint16"}},
                  {{"key", "NS=2;I=3"},
                   {"name", "uint32_write_test"},
                   {"node_name", "TestUInt32"},
                   {"node_id", "NS=1;S=TestUInt32"},
                   {"cmd_channel", this->uint32_cmd_channel.key},
                   {"enabled", true},
                   {"data_type", "uint32"}},
                  {{"key", "NS=2;I=4"},
                   {"name", "uint64_write_test"},
                   {"node_name", "TestUInt64"},
                   {"node_id", "NS=1;S=TestUInt64"},
                   {"cmd_channel", this->uint64_cmd_channel.key},
                   {"enabled", true},
                   {"data_type", "uint64"}},
                  {{"key", "NS=2;I=5"},
                   {"name", "int8_write_test"},
                   {"node_name", "TestInt8"},
                   {"node_id", "NS=1;S=TestInt8"},
                   {"cmd_channel", this->int8_cmd_channel.key},
                   {"enabled", true},
                   {"data_type", "int8"}},
                  {{"key", "NS=2;I=6"},
                   {"name", "int16_write_test"},
                   {"node_name", "TestInt16"},
                   {"node_id", "NS=1;S=TestInt16"},
                   {"cmd_channel", this->int16_cmd_channel.key},
                   {"enabled", true},
                   {"data_type", "int16"}},
                  {{"key", "NS=2;I=7"},
                   {"name", "int32_write_test"},
                   {"node_name", "TestInt32"},
                   {"node_id", "NS=1;S=TestInt32"},
                   {"cmd_channel", this->int32_cmd_channel.key},
                   {"enabled", true},
                   {"data_type", "int32"}},
                  {{"key", "NS=2;I=8"},
                   {"name", "int64_write_test"},
                   {"node_name", "TestInt64"},
                   {"node_id", "NS=1;S=TestInt64"},
                   {"cmd_channel", this->int64_cmd_channel.key},
                   {"enabled", true},
                   {"data_type", "int64"}},
                  {{"key", "NS=2;I=9"},
                   {"name", "float_write_test"},
                   {"node_name", "TestFloat"},
                   {"node_id", "NS=1;S=TestFloat"},
                   {"cmd_channel", this->float_cmd_channel.key},
                   {"enabled", true},
                   {"data_type", "float32"}},
                  {{"key", "NS=2;I=10"},
                   {"name", "double_write_test"},
                   {"node_name", "TestDouble"},
                   {"node_id", "NS=1;S=TestDouble"},
                   {"cmd_channel", this->double_cmd_channel.key},
                   {"enabled", true},
                   {"data_type", "float64"}}}
             )}
        };

        task = synnax::Task(rack.key, "opc_ua_write_task_test", "opc_write", "");

        auto p = xjson::Parser(task_cfg);
        this->cfg = std::make_unique<opc::WriteTaskConfig>(client, p);

        // Use the comprehensive default server configuration
        auto server_cfg = mock::ServerConfig::create_default();

        ctx = std::make_shared<task::MockContext>(client);
        auto reads = std::make_shared<std::vector<synnax::Frame>>();

        // Create test frames with different data types
        auto fr = synnax::Frame(10);

        // Create Series with single values using the value constructor
        fr.emplace(
            this->bool_cmd_channel.key,
            telem::Series(static_cast<uint8_t>(1), telem::UINT8_T)
        ); // bool = true = 1
        fr.emplace(
            this->uint16_cmd_channel.key,
            telem::Series(static_cast<uint16_t>(100), telem::UINT16_T)
        );
        fr.emplace(
            this->uint32_cmd_channel.key,
            telem::Series(static_cast<uint32_t>(12345), telem::UINT32_T)
        );
        fr.emplace(
            this->uint64_cmd_channel.key,
            telem::Series(static_cast<uint64_t>(12345), telem::UINT64_T)
        );
        fr.emplace(
            this->int8_cmd_channel.key,
            telem::Series(static_cast<int8_t>(100), telem::INT8_T)
        );
        fr.emplace(
            this->int32_cmd_channel.key,
            telem::Series(static_cast<int32_t>(54321), telem::INT32_T)
        );
        fr.emplace(
            this->int16_cmd_channel.key,
            telem::Series(static_cast<int16_t>(100), telem::INT16_T)
        );
        fr.emplace(
            this->int64_cmd_channel.key,
            telem::Series(static_cast<int64_t>(12345), telem::INT64_T)
        );
        fr.emplace(
            this->float_cmd_channel.key,
            telem::Series(2.718f, telem::FLOAT32_T)
        );
        fr.emplace(
            this->double_cmd_channel.key,
            telem::Series(3.14159, telem::FLOAT64_T)
        );
        reads->push_back(std::move(fr));

        mock_factory = pipeline::mock::simple_streamer_factory(
            {this->bool_cmd_channel.key,
             this->uint16_cmd_channel.key,
             this->uint32_cmd_channel.key,
             this->uint64_cmd_channel.key,
             this->int8_cmd_channel.key,
             this->int32_cmd_channel.key,
             this->int16_cmd_channel.key,
             this->int64_cmd_channel.key,
             this->float_cmd_channel.key,
             this->double_cmd_channel.key},
            reads
        );

        conn_pool = std::make_shared<opc::conn::Pool>();

        server = std::make_unique<mock::Server>(server_cfg);
        server->start();

        // Wait for server to be ready by attempting to connect
        auto test_client = ASSERT_EVENTUALLY_NIL_P_WITH_TIMEOUT(
            opc::conn::connect(conn_cfg, "test"),
            (5 * telem::SECOND).chrono(),
            (250 * telem::MILLISECOND).chrono()
        );
        UA_Client_disconnect(test_client.get());
    }

    std::unique_ptr<common::WriteTask> create_task() {
        return std::make_unique<common::WriteTask>(
            task,
            ctx,
            breaker::default_config(task.name),
            std::make_unique<opc::WriteTaskSink>(conn_pool, std::move(*cfg)),
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
    EXPECT_EQ(first_state.details.task, task.key);
    EXPECT_EQ(first_state.variant, status::variant::SUCCESS);
    EXPECT_EQ(first_state.message, "Task started successfully");
    ASSERT_EVENTUALLY_GE(mock_factory->streamer_opens, 1);

    wt->stop("stop_cmd", true);
    ASSERT_EVENTUALLY_GE(ctx->states.size(), 2);
    const auto second_state = ctx->states[1];
    EXPECT_EQ(second_state.key, "stop_cmd");
    EXPECT_EQ(second_state.details.task, task.key);
    EXPECT_EQ(second_state.variant, status::variant::SUCCESS);
    EXPECT_EQ(second_state.message, "Task stopped successfully");
}

TEST_F(TestWriteTask, testWriteValuesArePersisted) {
    // Save connection config before moving cfg
    auto conn_cfg = cfg->conn;

    auto wt = create_task();
    wt->start("start_cmd");
    x::defer stop_task([&wt]() { wt->stop("defer_stop", true); });
    ASSERT_EVENTUALLY_GE(mock_factory->streamer_opens, 1);

    // Give the write task time to process the frame
    std::this_thread::sleep_for(std::chrono::milliseconds(500));

    // Connect and read back the values to verify they were written
    auto [client, conn_err] = opc::conn::connect(
        conn_cfg,
        "[test.write_verification] "
    );
    ASSERT_FALSE(conn_err) << conn_err;

    // Verify boolean value (should be 1)
    auto [bool_result, bool_err] = util::simple_read(client, "NS=1;S=TestBoolean");
    ASSERT_FALSE(bool_err) << bool_err;
    EXPECT_EQ(bool_result.at<uint8_t>(0), 1);

    // Verify uint32 value (should be 12345)
    auto [uint32_result, uint32_err] = util::simple_read(client, "NS=1;S=TestUInt32");
    ASSERT_FALSE(uint32_err) << uint32_err;
    EXPECT_EQ(uint32_result.at<uint32_t>(0), 12345);

    // Verify float value (should be 2.718f)
    auto [float_result, float_err] = util::simple_read(client, "NS=1;S=TestFloat");
    ASSERT_FALSE(float_err) << float_err;
    EXPECT_FLOAT_EQ(float_result.at<float>(0), 2.718f);
}

TEST_F(TestWriteTask, testReconnectAfterServerRestart) {
    // Save connection config before moving cfg
    auto conn_cfg = cfg->conn;

    auto sink = std::make_unique<opc::WriteTaskSink>(conn_pool, std::move(*cfg));
    ASSERT_FALSE(sink->start());

    // First write should succeed
    auto fr1 = synnax::Frame(1);
    fr1.emplace(
        this->uint32_cmd_channel.key,
        telem::Series(static_cast<uint32_t>(11111), telem::UINT32_T)
    );
    auto write_err1 = sink->write(fr1);
    EXPECT_FALSE(write_err1) << write_err1;

    // Stop the server to simulate connection loss
    server->stop();
    std::this_thread::sleep_for(std::chrono::milliseconds(500));

    // Write while server is down - should fail
    auto fr2 = synnax::Frame(1);
    fr2.emplace(
        this->uint32_cmd_channel.key,
        telem::Series(static_cast<uint32_t>(22222), telem::UINT32_T)
    );
    auto write_err2 = sink->write(fr2);
    EXPECT_TRUE(write_err2) << "Write should fail when server is down";
    EXPECT_TRUE(write_err2.matches(opc::errors::UNREACHABLE))
        << "Error should be UNREACHABLE_ERROR, got: " << write_err2;

    // Restart the server and wait for it to be ready
    server->start();
    auto test_client = ASSERT_EVENTUALLY_NIL_P_WITH_TIMEOUT(
        opc::conn::connect(conn_cfg, "test"),
        (5 * telem::SECOND).chrono(),
        (250 * telem::MILLISECOND).chrono()
    );
    UA_Client_disconnect(test_client.get());

    // Write after server restart - should trigger reconnect and succeed
    auto fr3 = synnax::Frame(1);
    fr3.emplace(
        this->uint32_cmd_channel.key,
        telem::Series(static_cast<uint32_t>(33333), telem::UINT32_T)
    );
    auto write_err3 = sink->write(fr3);
    EXPECT_FALSE(write_err3) << "Write after reconnect should succeed: " << write_err3;

    // Verify the third value was written
    auto [client, conn_err] = opc::conn::connect(conn_cfg, "[test.reconnect] ");
    ASSERT_FALSE(conn_err) << conn_err;

    auto [result, read_err] = util::simple_read(client, "NS=1;S=TestUInt32");
    ASSERT_FALSE(read_err) << read_err;
    EXPECT_EQ(result.at<uint32_t>(0), 33333);

    ASSERT_FALSE(sink->stop());
}

TEST_F(TestWriteTask, testMultipleSequentialWrites) {
    // Save connection config before moving cfg
    auto conn_cfg = cfg->conn;

    auto sink = std::make_unique<opc::WriteTaskSink>(conn_pool, std::move(*cfg));
    ASSERT_FALSE(sink->start());

    // Perform multiple writes with different values
    for (int i = 0; i < 5; i++) {
        auto fr = synnax::Frame(1);
        fr.emplace(
            this->uint32_cmd_channel.key,
            telem::Series(static_cast<uint32_t>(i * 1000), telem::UINT32_T)
        );

        auto write_err = sink->write(fr);
        EXPECT_FALSE(write_err) << "Write " << i << " failed: " << write_err;

        std::this_thread::sleep_for(std::chrono::milliseconds(100));
    }

    // Verify the final value
    auto [client, conn_err] = opc::conn::connect(conn_cfg, "[test.multi_write] ");
    ASSERT_FALSE(conn_err) << conn_err;

    auto [result, read_err] = util::simple_read(client, "NS=1;S=TestUInt32");
    ASSERT_FALSE(read_err) << read_err;
    EXPECT_EQ(result.at<uint32_t>(0), 4000);

    ASSERT_FALSE(sink->stop());
}
