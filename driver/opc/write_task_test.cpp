// Copyright 2026 Synnax Labs, Inc.
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
#include "x/cpp/test/test.h"

#include "driver/opc/mock/server.h"
#include "driver/opc/opc.h"
#include "driver/opc/testutil/testutil.h"
#include "driver/opc/write_task.h"
#include "driver/pipeline/mock/pipeline.h"

class TestWriteTask : public ::testing::Test {
protected:
    synnax::Task task;
    std::unique_ptr<driver::opc::WriteTaskConfig> cfg;
    std::shared_ptr<driver::task::MockContext> ctx;
    std::shared_ptr<driver::pipeline::mock::StreamerFactory> mock_factory;
    std::unique_ptr<mock::Server> server;
    std::shared_ptr<driver::opc::connection::Pool> conn_pool;

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
        this->bool_cmd_channel = ASSERT_NIL_P(client->channels.create(
            make_unique_channel_name("bool_cmd"),
            x::telem::UINT8_T,
            true
        ));
        this->uint16_cmd_channel = ASSERT_NIL_P(client->channels.create(
            make_unique_channel_name("uint16_cmd"),
            x::telem::UINT16_T,
            true
        ));
        this->uint32_cmd_channel = ASSERT_NIL_P(client->channels.create(
            make_unique_channel_name("uint32_cmd"),
            x::telem::UINT32_T,
            true
        ));
        this->uint64_cmd_channel = ASSERT_NIL_P(client->channels.create(
            make_unique_channel_name("uint64_cmd"),
            x::telem::UINT64_T,
            true
        ));
        this->int8_cmd_channel = ASSERT_NIL_P(client->channels.create(
            make_unique_channel_name("int8_cmd"),
            x::telem::INT8_T,
            true
        ));
        this->int16_cmd_channel = ASSERT_NIL_P(client->channels.create(
            make_unique_channel_name("int16_cmd"),
            x::telem::INT16_T,
            true
        ));
        this->int32_cmd_channel = ASSERT_NIL_P(client->channels.create(
            make_unique_channel_name("int32_cmd"),
            x::telem::INT32_T,
            true
        ));
        this->int64_cmd_channel = ASSERT_NIL_P(client->channels.create(
            make_unique_channel_name("int64_cmd"),
            x::telem::INT64_T,
            true
        ));
        this->float_cmd_channel = ASSERT_NIL_P(client->channels.create(
            make_unique_channel_name("float_cmd"),
            x::telem::FLOAT32_T,
            true
        ));
        this->double_cmd_channel = ASSERT_NIL_P(client->channels.create(
            make_unique_channel_name("double_cmd"),
            x::telem::FLOAT64_T,
            true
        ));

        auto rack = ASSERT_NIL_P(client->racks.create("cat"));

        driver::opc::connection::Config conn_cfg;
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
        ASSERT_NIL(client->devices.create(dev));

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

        auto p = x::json::Parser(task_cfg);
        this->cfg = std::make_unique<driver::opc::WriteTaskConfig>(client, p);

        // Use the comprehensive default server configuration
        auto server_cfg = mock::ServerConfig::create_default();

        ctx = std::make_shared<driver::task::MockContext>(client);
        auto reads = std::make_shared<std::vector<x::telem::Frame>>();

        // Create test frames with different data types
        auto fr = x::telem::Frame(10);

        // Create Series with single values using the value constructor
        fr.emplace(
            this->bool_cmd_channel.key,
            x::telem::Series(static_cast<uint8_t>(1), x::telem::UINT8_T)
        ); // bool = true = 1
        fr.emplace(
            this->uint16_cmd_channel.key,
            x::telem::Series(static_cast<uint16_t>(100), x::telem::UINT16_T)
        );
        fr.emplace(
            this->uint32_cmd_channel.key,
            x::telem::Series(static_cast<uint32_t>(12345), x::telem::UINT32_T)
        );
        fr.emplace(
            this->uint64_cmd_channel.key,
            x::telem::Series(static_cast<uint64_t>(12345), x::telem::UINT64_T)
        );
        fr.emplace(
            this->int8_cmd_channel.key,
            x::telem::Series(static_cast<int8_t>(100), x::telem::INT8_T)
        );
        fr.emplace(
            this->int32_cmd_channel.key,
            x::telem::Series(static_cast<int32_t>(54321), x::telem::INT32_T)
        );
        fr.emplace(
            this->int16_cmd_channel.key,
            x::telem::Series(static_cast<int16_t>(100), x::telem::INT16_T)
        );
        fr.emplace(
            this->int64_cmd_channel.key,
            x::telem::Series(static_cast<int64_t>(12345), x::telem::INT64_T)
        );
        fr.emplace(
            this->float_cmd_channel.key,
            x::telem::Series(2.718f, x::telem::FLOAT32_T)
        );
        fr.emplace(
            this->double_cmd_channel.key,
            x::telem::Series(3.14159, x::telem::FLOAT64_T)
        );
        reads->push_back(std::move(fr));

        mock_factory = driver::pipeline::mock::simple_streamer_factory(
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

        conn_pool = std::make_shared<driver::opc::connection::Pool>();

        server = std::make_unique<mock::Server>(server_cfg);
        server->start();

        // Wait for server to be ready by attempting to connect
        auto test_client = ASSERT_EVENTUALLY_NIL_P_WITH_TIMEOUT(
            driver::opc::connection::connect(conn_cfg, "test"),
            (5 * x::telem::SECOND).chrono(),
            (250 * x::telem::MILLISECOND).chrono()
        );
        UA_Client_disconnect(test_client.get());
    }

    std::unique_ptr<driver::task::common::WriteTask> create_task() {
        return std::make_unique<driver::task::common::WriteTask>(
            task,
            ctx,
            x::breaker::default_config(task.name),
            std::make_unique<driver::opc::WriteTaskSink>(conn_pool, std::move(*cfg)),
            nullptr,
            mock_factory
        );
    }
};

TEST_F(TestWriteTask, testBasicWriteTask) {
    auto wt = create_task();
    wt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    const auto first_state = ctx->statuses[0];
    EXPECT_EQ(first_state.key, task.status_key());
    EXPECT_EQ(first_state.details.cmd, "start_cmd");
    EXPECT_EQ(first_state.details.task, task.key);
    EXPECT_EQ(first_state.variant, status::variant::SUCCESS);
    EXPECT_EQ(first_state.message, "Task started successfully");
    ASSERT_EVENTUALLY_GE(mock_factory->streamer_opens, 1);

    wt->stop("stop_cmd", true);
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 2);
    const auto second_state = ctx->statuses[1];
    EXPECT_EQ(second_state.key, task.status_key());
    EXPECT_EQ(second_state.details.cmd, "stop_cmd");
    EXPECT_EQ(second_state.details.task, task.key);
    EXPECT_EQ(second_state.variant, status::variant::SUCCESS);
    EXPECT_EQ(second_state.message, "Task stopped successfully");
}

TEST_F(TestWriteTask, testWriteValuesArePersisted) {
    // Save connection config before moving cfg
    auto conn_cfg = cfg->connection;

    auto wt = create_task();
    wt->start("start_cmd");
    x::defer::defer stop_task([&wt]() { wt->stop("defer_stop", true); });
    ASSERT_EVENTUALLY_GE(mock_factory->streamer_opens, 1);

    // Give the write task time to process the frame
    std::this_thread::sleep_for(std::chrono::milliseconds(500));

    // Connect and read back the values to verify they were written
    auto client = ASSERT_NIL_P(
        driver::opc::connection::connect(conn_cfg, "[test.write_verification] ")
    );

    // Verify boolean value (should be 1)
    const auto bool_result = ASSERT_NIL_P(
        driver::opc::testutil::simple_read(client, "NS=1;S=TestBoolean")
    );
    EXPECT_EQ(bool_result.at<uint8_t>(0), 1);

    // Verify uint32 value (should be 12345)
    const auto uint32_result = ASSERT_NIL_P(
        driver::opc::testutil::simple_read(client, "NS=1;S=TestUInt32")
    );
    EXPECT_EQ(uint32_result.at<uint32_t>(0), 12345);

    // Verify float value (should be 2.718f)
    const auto float_result = ASSERT_NIL_P(
        driver::opc::testutil::simple_read(client, "NS=1;S=TestFloat")
    );
    EXPECT_FLOAT_EQ(float_result.at<float>(0), 2.718f);
}

TEST_F(TestWriteTask, testReconnectAfterServerRestart) {
    // Save connection config before moving cfg
    auto conn_cfg = cfg->connection;

    auto sink = std::make_unique<driver::opc::WriteTaskSink>(conn_pool, std::move(*cfg));
    ASSERT_NIL(sink->start());

    // First write should succeed
    auto fr1 = x::telem::Frame(1);
    fr1.emplace(
        this->uint32_cmd_channel.key,
        x::telem::Series(static_cast<uint32_t>(11111), x::telem::UINT32_T)
    );
    ASSERT_NIL(sink->write(fr1));

    // Stop the server to simulate connection loss
    server->stop();
    std::this_thread::sleep_for(std::chrono::milliseconds(500));

    // Write while server is down - should fail
    auto fr2 = x::telem::Frame(1);
    fr2.emplace(
        this->uint32_cmd_channel.key,
        x::telem::Series(static_cast<uint32_t>(22222), x::telem::UINT32_T)
    );
    ASSERT_OCCURRED_AS(sink->write(fr2), driver::opc::errors::UNREACHABLE);

    // Restart the server and wait for it to be ready
    server->start();
    auto test_client = ASSERT_EVENTUALLY_NIL_P_WITH_TIMEOUT(
        driver::opc::connection::connect(conn_cfg, "test"),
        (5 * x::telem::SECOND).chrono(),
        (250 * x::telem::MILLISECOND).chrono()
    );
    UA_Client_disconnect(test_client.get());

    // Write after server restart - should trigger reconnect and succeed
    auto fr3 = x::telem::Frame(1);
    fr3.emplace(
        this->uint32_cmd_channel.key,
        x::telem::Series(static_cast<uint32_t>(33333), x::telem::UINT32_T)
    );
    ASSERT_NIL(sink->write(fr3));

    // Verify the third value was written
    auto client = ASSERT_NIL_P(driver::opc::connection::connect(conn_cfg, "[test.reconnect] "));
    const auto result = ASSERT_NIL_P(
        driver::opc::testutil::simple_read(client, "NS=1;S=TestUInt32")
    );
    EXPECT_EQ(result.at<uint32_t>(0), 33333);

    ASSERT_NIL(sink->stop());
}

TEST_F(TestWriteTask, testMultipleSequentialWrites) {
    // Save connection config before moving cfg
    auto conn_cfg = cfg->connection;

    auto sink = std::make_unique<driver::opc::WriteTaskSink>(conn_pool, std::move(*cfg));
    ASSERT_NIL(sink->start());

    // Perform multiple writes with different values
    for (int i = 0; i < 5; i++) {
        auto fr = x::telem::Frame(1);
        fr.emplace(
            this->uint32_cmd_channel.key,
            x::telem::Series(static_cast<uint32_t>(i * 1000), x::telem::UINT32_T)
        );
        ASSERT_NIL(sink->write(fr));
        std::this_thread::sleep_for(std::chrono::milliseconds(100));
    }

    // Verify the final value
    auto client = ASSERT_NIL_P(
        driver::opc::connection::connect(conn_cfg, "[test.multi_write] ")
    );
    const auto result = ASSERT_NIL_P(
        driver::opc::testutil::simple_read(client, "NS=1;S=TestUInt32")
    );
    EXPECT_EQ(result.at<uint32_t>(0), 4000);

    ASSERT_NIL(sink->stop());
}

TEST_F(TestWriteTask, testInvalidNodeIdErrorContainsChannelInfo) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());

    auto invalid_cmd_channel = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("invalid_node_cmd"),
        x::telem::UINT32_T,
        true
    ));

    auto rack = ASSERT_NIL_P(client->racks.create("invalid_node_test_rack"));

    driver::opc::connection::Config conn_cfg;
    conn_cfg.endpoint = "opc.tcp://0.0.0.0:4840";
    conn_cfg.security_mode = "None";
    conn_cfg.security_policy = "None";

    synnax::Device dev(
        "invalid_node_dev",
        "invalid_node_device",
        rack.key,
        "dev_invalid",
        "ni",
        "PXI-6255",
        nlohmann::to_string(json::object({{"connection", conn_cfg.to_json()}}))
    );
    ASSERT_NIL(client->devices.create(dev));

    // Create config with an invalid node ID that doesn't exist on the server
    json task_cfg = {
        {"data_saving", true},
        {"device", dev.key},
        {"channels",
         json::array(
             {{{"node_id", "NS=99;I=99999"},
               {"cmd_channel", invalid_cmd_channel.key},
               {"enabled", true}}}
         )}
    };

    auto p = x::json::Parser(task_cfg);
    auto invalid_cfg = driver::opc::WriteTaskConfig(client, p);
    ASSERT_FALSE(p.error()) << p.error().message();

    auto sink = std::make_unique<driver::opc::WriteTaskSink>(conn_pool, std::move(invalid_cfg));
    ASSERT_NIL(sink->start());

    // Attempt to write to the invalid node
    auto fr = x::telem::Frame(1);
    fr.emplace(
        invalid_cmd_channel.key,
        x::telem::Series(static_cast<uint32_t>(12345), x::telem::UINT32_T)
    );

    auto err = sink->write(fr);
    ASSERT_TRUE(err) << "Expected error for invalid node ID";

    const std::string err_msg = err.data;
    EXPECT_TRUE(err_msg.find(invalid_cmd_channel.name) != std::string::npos)
        << "Error message should contain channel name. Got: " << err_msg;
    EXPECT_TRUE(err_msg.find("NS=99;I=99999") != std::string::npos)
        << "Error message should contain node ID. Got: " << err_msg;

    ASSERT_NIL(sink->stop());
}
