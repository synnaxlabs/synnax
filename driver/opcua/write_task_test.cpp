// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <thread>

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/defer/defer.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/opcua/testutil/server.h"
#include "driver/opcua/testutil/testutil.h"
#include "driver/opcua/write_task.h"
#include "driver/pipeline/testutil/pipeline.h"

namespace driver::opcua {
class TestWriteTask : public ::testing::Test {
protected:
    synnax::task::Task task;
    std::unique_ptr<WriteTaskConfig> cfg;
    std::shared_ptr<task::MockContext> ctx;
    std::shared_ptr<pipeline::testutil::StreamerFactory> mock_factory;
    std::unique_ptr<testutil::Server> server;
    std::shared_ptr<connection::Pool> conn_pool;
    connection::Config conn_cfg;

    // Command channels for different data types
    synnax::channel::Channel bool_cmd_channel;
    synnax::channel::Channel uint16_cmd_channel;
    synnax::channel::Channel uint32_cmd_channel;
    synnax::channel::Channel uint64_cmd_channel;
    synnax::channel::Channel int8_cmd_channel;
    synnax::channel::Channel int16_cmd_channel;
    synnax::channel::Channel int32_cmd_channel;
    synnax::channel::Channel int64_cmd_channel;
    synnax::channel::Channel float_cmd_channel;
    synnax::channel::Channel double_cmd_channel;

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

        conn_cfg.endpoint = "opc.tcp://0.0.0.0:4840";
        conn_cfg.security_mode = "None";
        conn_cfg.security_policy = "None";

        synnax::device::Device dev{
            .key = "abc123",
            .rack = rack.key,
            .location = "dev1",
            .make = "ni",
            .model = "PXI-6255",
            .name = "my_device",
            .properties = x::json::json::object(
                {{"connection", testutil::connection_json(conn_cfg)}}
            ),
        };
        ASSERT_NIL(client->devices.create(dev));

        x::json::json task_cfg = {
            {"data_saving_disabled", false},
            {"device", dev.key},
            {"channels",
             x::json::json::array(
                 {{{"key", "NS=2;I=1"},
                   {"name", "bool_write_test"},
                   {"node_name", "TestBoolean"},
                   {"type", "node"},
                   {"node_id", "NS=1;S=TestBoolean"},
                   {"cmd_channel", this->bool_cmd_channel.key},
                   {"disabled", false},
                   {"data_type", "uint8"}},
                  {{"key", "NS=2;I=2"},
                   {"name", "uint16_write_test"},
                   {"node_name", "TestUInt16"},
                   {"type", "node"},
                   {"node_id", "NS=1;S=TestUInt16"},
                   {"cmd_channel", this->uint16_cmd_channel.key},
                   {"disabled", false},
                   {"data_type", "uint16"}},
                  {{"key", "NS=2;I=3"},
                   {"name", "uint32_write_test"},
                   {"node_name", "TestUInt32"},
                   {"type", "node"},
                   {"node_id", "NS=1;S=TestUInt32"},
                   {"cmd_channel", this->uint32_cmd_channel.key},
                   {"disabled", false},
                   {"data_type", "uint32"}},
                  {{"key", "NS=2;I=4"},
                   {"name", "uint64_write_test"},
                   {"node_name", "TestUInt64"},
                   {"type", "node"},
                   {"node_id", "NS=1;S=TestUInt64"},
                   {"cmd_channel", this->uint64_cmd_channel.key},
                   {"disabled", false},
                   {"data_type", "uint64"}},
                  {{"key", "NS=2;I=5"},
                   {"name", "int8_write_test"},
                   {"node_name", "TestInt8"},
                   {"type", "node"},
                   {"node_id", "NS=1;S=TestInt8"},
                   {"cmd_channel", this->int8_cmd_channel.key},
                   {"disabled", false},
                   {"data_type", "int8"}},
                  {{"key", "NS=2;I=6"},
                   {"name", "int16_write_test"},
                   {"node_name", "TestInt16"},
                   {"type", "node"},
                   {"node_id", "NS=1;S=TestInt16"},
                   {"cmd_channel", this->int16_cmd_channel.key},
                   {"disabled", false},
                   {"data_type", "int16"}},
                  {{"key", "NS=2;I=7"},
                   {"name", "int32_write_test"},
                   {"node_name", "TestInt32"},
                   {"type", "node"},
                   {"node_id", "NS=1;S=TestInt32"},
                   {"cmd_channel", this->int32_cmd_channel.key},
                   {"disabled", false},
                   {"data_type", "int32"}},
                  {{"key", "NS=2;I=8"},
                   {"name", "int64_write_test"},
                   {"node_name", "TestInt64"},
                   {"type", "node"},
                   {"node_id", "NS=1;S=TestInt64"},
                   {"cmd_channel", this->int64_cmd_channel.key},
                   {"disabled", false},
                   {"data_type", "int64"}},
                  {{"key", "NS=2;I=9"},
                   {"name", "float_write_test"},
                   {"node_name", "TestFloat"},
                   {"type", "node"},
                   {"node_id", "NS=1;S=TestFloat"},
                   {"cmd_channel", this->float_cmd_channel.key},
                   {"disabled", false},
                   {"data_type", "float32"}},
                  {{"key", "NS=2;I=10"},
                   {"name", "double_write_test"},
                   {"node_name", "TestDouble"},
                   {"type", "node"},
                   {"node_id", "NS=1;S=TestDouble"},
                   {"cmd_channel", this->double_cmd_channel.key},
                   {"disabled", false},
                   {"data_type", "float64"}}}
             )}
        };

        task = synnax::task::Task{
            .rack = rack.key,
            .name = "opc_ua_write_task_test",
            .type = "opc_write",
        };

        auto p = x::json::Parser(task_cfg);
        this->cfg = std::make_unique<WriteTaskConfig>(client, p);

        // Use the comprehensive default server configuration
        auto server_cfg = testutil::ServerConfig::create_default();
        server_cfg.pump = testutil::Pump{};

        ctx = std::make_shared<task::MockContext>(client);
        auto reads = std::make_shared<std::vector<::x::telem::Frame>>();

        // Create test frames with different data types
        auto fr = ::x::telem::Frame(10);

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

        mock_factory = driver::pipeline::testutil::simple_streamer_factory(
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

        conn_pool = std::make_shared<connection::Pool>(testutil::create_store());

        server = std::make_unique<testutil::Server>(server_cfg);
        server->start();

        // Wait for server to be ready by attempting to connect
        auto test_client = ASSERT_EVENTUALLY_NIL_P_WITH_TIMEOUT(
            connection::connect(conn_cfg, {}, "test"),
            (5 * x::telem::SECOND).chrono(),
            (250 * x::telem::MILLISECOND).chrono()
        );
        UA_Client_disconnect(test_client.get());
    }

    std::unique_ptr<common::WriteTask> create_task() {
        return std::make_unique<common::WriteTask>(
            task,
            ctx,
            x::breaker::default_config(task.name),
            std::make_unique<WriteTaskSink>(ctx->client, conn_pool, std::move(*cfg)),
            nullptr,
            mock_factory
        );
    }

    /// @returns a sink that calls method on object with arguments, with the commands of
    /// the double command channel.
    [[nodiscard]] std::unique_ptr<WriteTaskSink> create_method_sink(
        const std::string &method,
        const x::json::json &arguments,
        const std::string &object = "NS=1;S=Pump"
    ) const {
        x::json::json task_cfg = {
            {"device", "abc123"},
            {"channels",
             x::json::json::array({
                 {{"type", "method"},
                  {"node_id", method},
                  {"object_id", object},
                  {"arguments", arguments},
                  {"cmd_channel", this->double_cmd_channel.key}},
                 {{"type", "node"},
                  {"node_id", "NS=1;S=TestFloat"},
                  {"cmd_channel", this->float_cmd_channel.key}},
             })}
        };
        auto p = x::json::Parser(task_cfg);
        auto write_cfg = WriteTaskConfig(ctx->client, p);
        EXPECT_FALSE(p.error()) << p.error().message();
        return std::make_unique<WriteTaskSink>(
            ctx->client,
            conn_pool,
            std::move(write_cfg)
        );
    }

    /// @returns a frame that commands the double channel with value.
    [[nodiscard]] x::telem::Frame method_command(const double value) const {
        auto fr = x::telem::Frame(1);
        fr.emplace(this->double_cmd_channel.key, x::telem::Series(value));
        return fr;
    }

    static constexpr std::uint16_t SECURED_PORT = 4862;
    inline static const std::string SECURED_DEVICE_KEY = "secured_write_device";
    synnax::channel::Channel secured_cmd_channel;

    /// @brief creates a command channel, and a device for the secured server that
    /// presents server_cert.
    void create_secured_device(const certificate::Certificate &server_cert) {
        this->secured_cmd_channel = ASSERT_NIL_P(ctx->client->channels.create(
            make_unique_channel_name("secured_cmd"),
            x::telem::UINT32_T,
            true
        ));
        const auto rack = ASSERT_NIL_P(ctx->client->racks.create("secured_write_rack"));
        connection::Config secured_cfg;
        secured_cfg.endpoint = "opc.tcp://localhost:" + std::to_string(SECURED_PORT);
        secured_cfg.security_mode = "SignAndEncrypt";
        secured_cfg.security_policy = "Basic256Sha256";
        secured_cfg.server_certificate = server_cert;
        synnax::device::Device dev{
            .key = SECURED_DEVICE_KEY,
            .rack = rack.key,
            .location = secured_cfg.endpoint,
            .make = "opc",
            .model = "OPC UA Server",
            .name = "Secured Write Device",
            .properties = x::json::json::object(
                {{"connection", testutil::connection_json(secured_cfg)}}
            ),
        };
        ASSERT_NIL(ctx->client->devices.create(dev));
    }

    /// @brief creates a sink that writes the secured command channel to node_id.
    [[nodiscard]] std::unique_ptr<WriteTaskSink>
    create_secured_sink(const std::string &node_id) const {
        x::json::json task_cfg = {
            {"data_saving_disabled", false},
            {"device", SECURED_DEVICE_KEY},
            {"channels",
             x::json::json::array(
                 {{{"type", "node"},
                   {"node_id", node_id},
                   {"cmd_channel", this->secured_cmd_channel.key},
                   {"disabled", false}}}
             )}
        };
        auto p = x::json::Parser(task_cfg);
        auto write_cfg = WriteTaskConfig(ctx->client, p);
        EXPECT_FALSE(p.error()) << p.error().message();
        return std::make_unique<WriteTaskSink>(
            ctx->client,
            conn_pool,
            std::move(write_cfg)
        );
    }
};

TEST_F(TestWriteTask, testBasicWriteTask) {
    auto wt = create_task();
    wt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    const auto first_state = ctx->statuses[0];
    EXPECT_EQ(first_state.key, synnax::task::status_key(task));
    EXPECT_EQ(first_state.details.cmd, "start_cmd");
    EXPECT_EQ(first_state.details.task, task.key);
    EXPECT_EQ(first_state.variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(first_state.message, "Started task");
    ASSERT_EVENTUALLY_GE(
        mock_factory->streamer_opens.load(std::memory_order_acquire),
        1
    );

    wt->stop("stop_cmd", true);
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 2);
    const auto second_state = ctx->statuses[1];
    EXPECT_EQ(second_state.key, synnax::task::status_key(task));
    EXPECT_EQ(second_state.details.cmd, "stop_cmd");
    EXPECT_EQ(second_state.details.task, task.key);
    EXPECT_EQ(second_state.variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(second_state.message, "Stopped task");
}

TEST_F(TestWriteTask, testWriteValuesArePersisted) {
    auto wt = create_task();
    wt->start("start_cmd");
    x::defer::defer stop_task([&wt]() { wt->stop("defer_stop", true); });
    ASSERT_EVENTUALLY_GE(
        mock_factory->streamer_opens.load(std::memory_order_acquire),
        1
    );

    // Give the write task time to process the frame
    std::this_thread::sleep_for(std::chrono::milliseconds(500));

    // Connect and read back the values to verify they were written
    auto client = ASSERT_NIL_P(
        connection::connect(conn_cfg, {}, "[test.write_verification] ")
    );

    // Verify boolean value (should be 1)
    const auto bool_result = ASSERT_NIL_P(
        testutil::simple_read(client, "NS=1;S=TestBoolean")
    );
    EXPECT_EQ(bool_result.at<uint8_t>(0), 1);

    // Verify uint32 value (should be 12345)
    const auto uint32_result = ASSERT_NIL_P(
        testutil::simple_read(client, "NS=1;S=TestUInt32")
    );
    EXPECT_EQ(uint32_result.at<uint32_t>(0), 12345);

    // Verify float value (should be 2.718f)
    const auto float_result = ASSERT_NIL_P(
        testutil::simple_read(client, "NS=1;S=TestFloat")
    );
    EXPECT_FLOAT_EQ(float_result.at<float>(0), 2.718f);
}

/// @brief when a frame contains multiple samples for a channel, only the last
/// sample should be written to hardware.
TEST_F(TestWriteTask, testLastWriteWins) {
    auto sink = std::make_unique<WriteTaskSink>(
        ctx->client,
        conn_pool,
        std::move(*cfg)
    );
    ASSERT_NIL(sink->start());

    auto fr = ::x::telem::Frame(1);
    fr.emplace(
        this->uint32_cmd_channel.key,
        x::telem::Series(std::vector<uint32_t>{111, 222, 333})
    );
    ASSERT_NIL(sink->write(fr));

    std::this_thread::sleep_for(std::chrono::milliseconds(500));

    auto client = ASSERT_NIL_P(
        connection::connect(conn_cfg, {}, "[test.last_write_wins] ")
    );
    const auto result = ASSERT_NIL_P(
        testutil::simple_read(client, "NS=1;S=TestUInt32")
    );
    EXPECT_EQ(result.at<uint32_t>(0), 333);

    ASSERT_NIL(sink->stop());
}

TEST_F(TestWriteTask, testReconnectAfterServerRestart) {
    auto sink = std::make_unique<WriteTaskSink>(
        ctx->client,
        conn_pool,
        std::move(*cfg)
    );
    ASSERT_NIL(sink->start());

    // First write should succeed
    auto fr1 = ::x::telem::Frame(1);
    fr1.emplace(
        this->uint32_cmd_channel.key,
        x::telem::Series(static_cast<uint32_t>(11111), x::telem::UINT32_T)
    );
    ASSERT_NIL(sink->write(fr1));

    // Stop the server to simulate connection loss
    server->stop();
    std::this_thread::sleep_for(std::chrono::milliseconds(500));

    // Write while server is down - should fail
    auto fr2 = ::x::telem::Frame(1);
    fr2.emplace(
        this->uint32_cmd_channel.key,
        x::telem::Series(static_cast<uint32_t>(22222), x::telem::UINT32_T)
    );
    ASSERT_OCCURRED_AS(sink->write(fr2), errors::UNREACHABLE);

    // Restart the server and wait for it to be ready
    server->start();
    auto test_client = ASSERT_EVENTUALLY_NIL_P_WITH_TIMEOUT(
        connection::connect(conn_cfg, {}, "test"),
        (5 * x::telem::SECOND).chrono(),
        (250 * x::telem::MILLISECOND).chrono()
    );
    UA_Client_disconnect(test_client.get());

    // Write after server restart - should trigger reconnect and succeed
    auto fr3 = ::x::telem::Frame(1);
    fr3.emplace(
        this->uint32_cmd_channel.key,
        x::telem::Series(static_cast<uint32_t>(33333), x::telem::UINT32_T)
    );
    ASSERT_NIL(sink->write(fr3));

    // Verify the third value was written
    auto client = ASSERT_NIL_P(connection::connect(conn_cfg, {}, "[test.reconnect] "));
    const auto result = ASSERT_NIL_P(
        testutil::simple_read(client, "NS=1;S=TestUInt32")
    );
    EXPECT_EQ(result.at<uint32_t>(0), 33333);

    ASSERT_NIL(sink->stop());
}

/// @brief it should retrieve the device again when its connection breaks, so a changed
/// endpoint takes effect without configuring the task again.
TEST_F(TestWriteTask, testReconnectUsesUpdatedDevice) {
    auto sink = std::make_unique<WriteTaskSink>(
        ctx->client,
        conn_pool,
        std::move(*cfg)
    );
    ASSERT_NIL(sink->start());
    auto moved_server_cfg = testutil::ServerConfig::create_default();
    moved_server_cfg.port = 4858;
    testutil::Server moved_server(moved_server_cfg);
    moved_server.start();
    ASSERT_TRUE(moved_server.wait_until_ready());
    auto moved_cfg = conn_cfg;
    moved_cfg.endpoint = "opc.tcp://localhost:4858";
    auto dev = ASSERT_NIL_P(ctx->client->devices.retrieve("abc123"));
    dev.properties["connection"] = testutil::connection_json(moved_cfg);
    ASSERT_NIL(ctx->client->devices.create(dev));
    server->stop();
    std::this_thread::sleep_for(std::chrono::milliseconds(500));

    auto fr = ::x::telem::Frame(1);
    fr.emplace(
        this->uint32_cmd_channel.key,
        x::telem::Series(static_cast<uint32_t>(22222), x::telem::UINT32_T)
    );
    ASSERT_NIL(sink->write(fr));

    auto client = ASSERT_NIL_P(connection::connect(moved_cfg, {}, "[test.moved] "));
    const auto result = ASSERT_NIL_P(
        testutil::simple_read(client, "NS=1;S=TestUInt32")
    );
    EXPECT_EQ(result.at<uint32_t>(0), 22222);
    ASSERT_NIL(sink->stop());
}

TEST_F(TestWriteTask, testMultipleSequentialWrites) {
    auto sink = std::make_unique<WriteTaskSink>(
        ctx->client,
        conn_pool,
        std::move(*cfg)
    );
    ASSERT_NIL(sink->start());

    // Perform multiple writes with different values
    for (int i = 0; i < 5; i++) {
        auto fr = ::x::telem::Frame(1);
        fr.emplace(
            this->uint32_cmd_channel.key,
            x::telem::Series(static_cast<uint32_t>(i * 1000), x::telem::UINT32_T)
        );
        ASSERT_NIL(sink->write(fr));
        std::this_thread::sleep_for(std::chrono::milliseconds(100));
    }

    // Verify the final value
    auto client = ASSERT_NIL_P(
        connection::connect(conn_cfg, {}, "[test.multi_write] ")
    );
    const auto result = ASSERT_NIL_P(
        testutil::simple_read(client, "NS=1;S=TestUInt32")
    );
    EXPECT_EQ(result.at<uint32_t>(0), 4000);

    ASSERT_NIL(sink->stop());
}

/// @brief it should warn about a node that the server rejects and still write the other
/// nodes in the command.
TEST_F(TestWriteTask, testRejectedNodeWritesOthers) {
    const auto bad_cmd = ASSERT_NIL_P(ctx->client->channels.create(
        make_unique_channel_name("bad_node_cmd"),
        x::telem::UINT32_T,
        true
    ));
    x::json::json task_cfg = {
        {"data_saving_disabled", false},
        {"device", "abc123"},
        {"channels",
         x::json::json::array({
             {{"type", "node"},
              {"node_id", "NS=99;I=99999"},
              {"cmd_channel", bad_cmd.key},
              {"disabled", false}},
             {{"type", "node"},
              {"node_id", "NS=1;S=TestUInt32"},
              {"cmd_channel", this->uint32_cmd_channel.key},
              {"disabled", false}},
         })}
    };
    auto p = x::json::Parser(task_cfg);
    auto write_cfg = WriteTaskConfig(ctx->client, p);
    ASSERT_FALSE(p.error()) << p.error().message();
    const auto sink = std::make_unique<WriteTaskSink>(
        ctx->client,
        conn_pool,
        std::move(write_cfg)
    );
    ASSERT_NIL(sink->start());
    auto fr = x::telem::Frame(2);
    fr.emplace(
        bad_cmd.key,
        x::telem::Series(static_cast<uint32_t>(1), x::telem::UINT32_T)
    );
    fr.emplace(
        this->uint32_cmd_channel.key,
        x::telem::Series(static_cast<uint32_t>(4321), x::telem::UINT32_T)
    );
    const auto err = sink->write(fr);
    ASSERT_MATCHES(err, errors::COMMAND_REJECTED);
    EXPECT_EQ(
        err.data,
        bad_cmd.name +
            " returned BadNodeIdUnknown. The node id refers to a node that does not "
            "exist in the server address space."
    );
    auto client = ASSERT_NIL_P(connection::connect(conn_cfg, {}, "[test.rejected] "));
    const auto result = ASSERT_NIL_P(
        testutil::simple_read(client, "NS=1;S=TestUInt32")
    );
    EXPECT_EQ(result.at<uint32_t>(0), 4321);
    ASSERT_NIL(sink->stop());
}

/// @returns a sink that writes cmd_channel to node_id.
std::unique_ptr<WriteTaskSink> sink_for(
    const std::shared_ptr<synnax::Synnax> &client,
    const std::shared_ptr<connection::Pool> &pool,
    const synnax::channel::Key cmd_channel,
    const std::string &node_id
) {
    x::json::json task_cfg = {
        {"data_saving_disabled", false},
        {"device", "abc123"},
        {"channels",
         x::json::json::array({
             {{"type", "node"},
              {"node_id", node_id},
              {"cmd_channel", cmd_channel},
              {"disabled", false}},
         })}
    };
    auto p = x::json::Parser(task_cfg);
    auto write_cfg = WriteTaskConfig(client, p);
    EXPECT_FALSE(p.error()) << p.error().message();
    return std::make_unique<WriteTaskSink>(client, pool, std::move(write_cfg));
}

/// @brief it should write a node by the URI of its namespace.
TEST_F(TestWriteTask, testWritesNamespaceURINode) {
    const auto sink = sink_for(
        ctx->client,
        conn_pool,
        this->uint32_cmd_channel.key,
        "nsu=urn:open62541.server.application;s=TestUInt32"
    );
    ASSERT_NIL(sink->start());
    auto fr = x::telem::Frame(1);
    fr.emplace(
        this->uint32_cmd_channel.key,
        x::telem::Series(static_cast<uint32_t>(2468), x::telem::UINT32_T)
    );
    ASSERT_NIL(sink->write(fr));
    auto client = ASSERT_NIL_P(connection::connect(conn_cfg, {}, "[test.nsu] "));
    const auto result = ASSERT_NIL_P(
        testutil::simple_read(client, "NS=1;S=TestUInt32")
    );
    EXPECT_EQ(result.at<uint32_t>(0), 2468);
    ASSERT_NIL(sink->stop());
}

/// @brief it should reject a value for a node in a namespace that the server lacks.
TEST_F(TestWriteTask, testUnknownNamespaceRejectsValue) {
    const auto sink = sink_for(
        ctx->client,
        conn_pool,
        this->uint32_cmd_channel.key,
        "nsu=urn:missing;s=TestUInt32"
    );
    ASSERT_NIL(sink->start());
    auto fr = x::telem::Frame(1);
    fr.emplace(
        this->uint32_cmd_channel.key,
        x::telem::Series(static_cast<uint32_t>(1357), x::telem::UINT32_T)
    );
    const auto err = sink->write(fr);
    ASSERT_MATCHES(err, errors::COMMAND_REJECTED);
    EXPECT_EQ(
        err.data,
        this->uint32_cmd_channel.name + ": the server has no namespace urn:missing."
    );
    ASSERT_NIL(sink->stop());
}

/// @brief it should convert a command to the data type of its node.
TEST_F(TestWriteTask, testConvertsCommandToNodeType) {
    const auto cmd = ASSERT_NIL_P(ctx->client->channels.create(
        make_unique_channel_name("float_cmd"),
        x::telem::FLOAT64_T,
        true
    ));
    const auto sink = sink_for(ctx->client, conn_pool, cmd.key, "NS=1;S=TestUInt32");
    ASSERT_NIL(sink->start());
    auto fr = x::telem::Frame(1);
    fr.emplace(cmd.key, x::telem::Series(77.0));
    ASSERT_NIL(sink->write(fr));
    auto client = ASSERT_NIL_P(connection::connect(conn_cfg, {}, "[test.convert] "));
    const auto result = ASSERT_NIL_P(
        testutil::simple_read(client, "NS=1;S=TestUInt32")
    );
    EXPECT_EQ(result.at<uint32_t>(0), 77);
    ASSERT_NIL(sink->stop());
}

/// @brief it should not send a command that the type of its node cannot hold.
TEST_F(TestWriteTask, testRejectsCommandOutOfRange) {
    const auto cmd = ASSERT_NIL_P(ctx->client->channels.create(
        make_unique_channel_name("float_cmd"),
        x::telem::FLOAT64_T,
        true
    ));
    const auto sink = sink_for(ctx->client, conn_pool, cmd.key, "NS=1;S=TestUInt32");
    ASSERT_NIL(sink->start());
    auto fr = x::telem::Frame(1);
    fr.emplace(cmd.key, x::telem::Series(-1.0));
    const auto err = sink->write(fr);
    ASSERT_MATCHES(err, errors::COMMAND_REJECTED);
    EXPECT_EQ(err.data, cmd.name + ": -1 is out of range for a UInt32 node.");
    ASSERT_NIL(sink->stop());
}

/// @brief it should write a node in a namespace that the server adds after the task
/// starts, on the same Session, converted to the data type of the node.
TEST_F(TestWriteTask, testNamespaceAddedAfterStart) {
    const auto cmd = ASSERT_NIL_P(ctx->client->channels.create(
        make_unique_channel_name("float_cmd"),
        x::telem::FLOAT64_T,
        true
    ));
    const auto sink = sink_for(ctx->client, conn_pool, cmd.key, "nsu=urn:late;s=Late");
    ASSERT_NIL(sink->start());
    auto fr = x::telem::Frame(1);
    fr.emplace(cmd.key, x::telem::Series(97.0));
    ASSERT_OCCURRED_AS(sink->write(fr), errors::COMMAND_REJECTED);
    const auto sessions = server->sessions_created.load();
    UA_UInt32 initial = 0;
    UA_Variant value;
    UA_Variant_setScalar(&value, &initial, &UA_TYPES[UA_TYPES_UINT32]);
    server->add_namespace(
        "urn:late",
        {{0, "Late", &UA_TYPES[UA_TYPES_UINT32], value, "Late"}}
    );
    ASSERT_NIL(sink->write(fr));
    EXPECT_EQ(server->sessions_created.load(), sessions);
    auto client = ASSERT_NIL_P(connection::connect(conn_cfg, {}, "[test.late] "));
    const auto result = ASSERT_NIL_P(testutil::simple_read(client, "NS=2;S=Late"));
    EXPECT_EQ(result.at<uint32_t>(0), 97);
    ASSERT_NIL(sink->stop());
}

/// @brief it should convert a command to the data type of a node that the server adds
/// after the task starts, on the same Session.
TEST_F(TestWriteTask, testNodeAddedAfterStart) {
    const auto cmd = ASSERT_NIL_P(ctx->client->channels.create(
        make_unique_channel_name("float_cmd"),
        x::telem::FLOAT64_T,
        true
    ));
    const auto sink = sink_for(ctx->client, conn_pool, cmd.key, "NS=1;S=Late");
    ASSERT_NIL(sink->start());
    auto fr = x::telem::Frame(1);
    fr.emplace(cmd.key, x::telem::Series(97.0));
    ASSERT_OCCURRED_AS(sink->write(fr), errors::COMMAND_REJECTED);
    const auto sessions = server->sessions_created.load();
    UA_UInt32 initial = 0;
    UA_Variant value;
    UA_Variant_setScalar(&value, &initial, &UA_TYPES[UA_TYPES_UINT32]);
    server->add_namespace(
        testutil::SERVER_APPLICATION_URI,
        {{0, "Late", &UA_TYPES[UA_TYPES_UINT32], value, "Late"}}
    );
    ASSERT_NIL(sink->write(fr));
    EXPECT_EQ(server->sessions_created.load(), sessions);
    auto client = ASSERT_NIL_P(connection::connect(conn_cfg, {}, "[test.late] "));
    const auto result = ASSERT_NIL_P(testutil::simple_read(client, "NS=1;S=Late"));
    EXPECT_EQ(result.at<uint32_t>(0), 97);
    ASSERT_NIL(sink->stop());
}

/// @brief it should write more nodes than the server allows in one request.
TEST_F(TestWriteTask, testSplitsWriteAtServerLimit) {
    server->stop();
    server->cfg.max_nodes_per_write = 2;
    server->start();
    ASSERT_TRUE(server->wait_until_ready());
    const auto sink = std::make_unique<WriteTaskSink>(
        ctx->client,
        conn_pool,
        std::move(*cfg)
    );
    ASSERT_NIL(sink->start());
    auto fr = x::telem::Frame(3);
    fr.emplace(
        this->bool_cmd_channel.key,
        x::telem::Series(static_cast<uint8_t>(0), x::telem::UINT8_T)
    );
    fr.emplace(
        this->uint32_cmd_channel.key,
        x::telem::Series(static_cast<uint32_t>(9753), x::telem::UINT32_T)
    );
    fr.emplace(
        this->float_cmd_channel.key,
        x::telem::Series(1.5f, x::telem::FLOAT32_T)
    );
    ASSERT_NIL(sink->write(fr));
    auto client = ASSERT_NIL_P(connection::connect(conn_cfg, {}, "[test.split] "));
    EXPECT_EQ(
        ASSERT_NIL_P(testutil::simple_read(client, "NS=1;S=TestBoolean"))
            .at<uint8_t>(0),
        0
    );
    EXPECT_EQ(
        ASSERT_NIL_P(testutil::simple_read(client, "NS=1;S=TestUInt32"))
            .at<uint32_t>(0),
        9753
    );
    EXPECT_FLOAT_EQ(
        ASSERT_NIL_P(testutil::simple_read(client, "NS=1;S=TestFloat")).at<float>(0),
        1.5f
    );
    ASSERT_NIL(sink->stop());
}

/// @brief it should start while the server is off, and write once the server returns.
TEST_F(TestWriteTask, testStartsWhileServerOff) {
    server->stop();
    const auto sink = std::make_unique<WriteTaskSink>(
        ctx->client,
        conn_pool,
        std::move(*cfg)
    );
    ASSERT_NIL(sink->start());
    auto fr = x::telem::Frame(1);
    fr.emplace(
        this->uint32_cmd_channel.key,
        x::telem::Series(static_cast<uint32_t>(777), x::telem::UINT32_T)
    );
    ASSERT_OCCURRED_AS(sink->write(fr), errors::UNREACHABLE);
    server->start();
    ASSERT_TRUE(server->wait_until_ready());
    ASSERT_NIL(sink->write(fr));
    auto client = ASSERT_NIL_P(connection::connect(conn_cfg, {}, "[test.off] "));
    const auto result = ASSERT_NIL_P(
        testutil::simple_read(client, "NS=1;S=TestUInt32")
    );
    EXPECT_EQ(result.at<uint32_t>(0), 777);
    ASSERT_NIL(sink->stop());
}

/// @brief it should keep writing after the Driver certificate changes and the server
/// trusts only the new certificate.
TEST_F(TestWriteTask, testWriteUsesNewCertificate) {
    const auto store = testutil::create_store();
    conn_pool = std::make_shared<connection::Pool>(store);
    const auto old = ASSERT_NIL_P(store->current());
    const auto server_id = ASSERT_NIL_P(testutil::create_server_identity());
    auto server_cfg = testutil::ServerConfig::create_default();
    server_cfg.port = SECURED_PORT;
    server_cfg.identity = server_id;
    server_cfg.trusted_clients = {old.certificate};
    auto secured = std::make_unique<testutil::Server>(server_cfg);
    secured->start();
    ASSERT_TRUE(secured->wait_until_ready());
    this->create_secured_device(server_id.certificate);
    const auto sink = this->create_secured_sink("NS=1;S=TestUInt32");
    ASSERT_NIL(sink->start());
    auto fr = x::telem::Frame(1);
    fr.emplace(
        this->secured_cmd_channel.key,
        x::telem::Series(static_cast<uint32_t>(1), x::telem::UINT32_T)
    );
    ASSERT_NIL(sink->write(fr));
    const auto next = ASSERT_NIL_P(store->regenerate());
    secured->stop();
    server_cfg.trusted_clients = {next.certificate};
    secured = std::make_unique<testutil::Server>(server_cfg);
    secured->start();
    ASSERT_TRUE(secured->wait_until_ready());
    ASSERT_NIL(sink->write(fr));
    ASSERT_NIL(sink->stop());
}

/// @brief it should not send a write again for a node error after the Driver
/// certificate changes, since a new connection cannot fix it.
TEST_F(TestWriteTask, testNodeErrorKeepsConnection) {
    const auto store = testutil::create_store();
    conn_pool = std::make_shared<connection::Pool>(store);
    const auto server_id = ASSERT_NIL_P(testutil::create_server_identity());
    auto server_cfg = testutil::ServerConfig::create_default();
    server_cfg.port = SECURED_PORT;
    server_cfg.identity = server_id;
    testutil::Server secured(server_cfg);
    secured.start();
    ASSERT_TRUE(secured.wait_until_ready());
    this->create_secured_device(server_id.certificate);
    const auto sink = this->create_secured_sink("NS=99;I=99999");
    ASSERT_NIL(sink->start());
    auto fr = x::telem::Frame(1);
    fr.emplace(
        this->secured_cmd_channel.key,
        x::telem::Series(static_cast<uint32_t>(1), x::telem::UINT32_T)
    );
    ASSERT_NIL(store->regenerate().second);
    ASSERT_MATCHES(sink->write(fr), errors::COMMAND_REJECTED);
    ASSERT_EQ(conn_pool->size(), 1);
    ASSERT_NIL(sink->stop());
}

/// @brief it should configure when a disabled channel has no node id or Synnax
/// channel bound to it.
TEST_F(TestWriteTask, testUnboundDisabledChannel) {
    x::json::json task_cfg{
        {"data_saving_disabled", false},
        {"device", "abc123"},
        {"channels",
         x::json::json::array(
             {{{"key", "NS=2;I=1"},
               {"name", "bool_write_test"},
               {"node_name", "TestBoolean"},
               {"type", "node"},
               {"node_id", "NS=1;S=TestBoolean"},
               {"cmd_channel", this->bool_cmd_channel.key},
               {"disabled", false},
               {"data_type", "uint8"}},
              {{"key", "blank"}, {"type", "node"}, {"disabled", true}}}
         )}
    };

    auto p = x::json::Parser(task_cfg);
    const auto parsed = std::make_unique<WriteTaskConfig>(ctx->client, p);
    ASSERT_FALSE(p.error()) << p.error().message();
    EXPECT_EQ(parsed->channels.size(), 1);
    EXPECT_TRUE(parsed->channels.contains(this->bool_cmd_channel.key));
}

/// @brief it should call a method with the command as its argument, and write the
/// values of node channels in the same frame.
TEST_F(TestWriteTask, testCallsMethodWithCommand) {
    const auto sink = this->create_method_sink(
        "NS=1;S=Pump.SetSpeed",
        {{{"source", "command"}}}
    );
    ASSERT_NIL(sink->start());
    auto fr = this->method_command(42);
    fr.emplace(this->float_cmd_channel.key, x::telem::Series(1.5f));
    ASSERT_NIL(sink->write(fr));
    const auto client = ASSERT_NIL_P(connection::connect(conn_cfg, {}, "[test] "));
    const auto speed = ASSERT_NIL_P(testutil::simple_read(client, "NS=1;S=Pump.Speed"));
    EXPECT_EQ(speed.at<double>(0), 42);
    const auto value = ASSERT_NIL_P(testutil::simple_read(client, "NS=1;S=TestFloat"));
    EXPECT_FLOAT_EQ(value.at<float>(0), 1.5f);
    ASSERT_NIL(sink->stop());
}

/// @brief it should convert fixed arguments and the command to the declared types of
/// the method's arguments.
TEST_F(TestWriteTask, testCallsMethodWithFixedArguments) {
    const auto sink = this->create_method_sink(
        "NS=1;S=Pump.Configure",
        {{{"source", "value"}, {"value", "auto"}}, {{"source", "command"}}}
    );
    ASSERT_NIL(sink->start());
    auto fr = this->method_command(7);
    ASSERT_NIL(sink->write(fr));
    const auto client = ASSERT_NIL_P(connection::connect(conn_cfg, {}, "[test] "));
    const auto speed = ASSERT_NIL_P(testutil::simple_read(client, "NS=1;S=Pump.Speed"));
    EXPECT_EQ(speed.at<double>(0), 7);
    const auto mode = ASSERT_NIL_P(testutil::simple_read(client, "NS=1;S=Pump.Mode"));
    EXPECT_EQ(mode.at<std::string>(0), "auto");
    ASSERT_NIL(sink->stop());
}

/// @brief it should reject a command when the method fails, naming the channel.
TEST_F(TestWriteTask, testRejectsFailedMethod) {
    const auto sink = this->create_method_sink(
        "NS=1;S=Pump.Fail",
        x::json::json::array()
    );
    ASSERT_NIL(sink->start());
    auto fr = this->method_command(1);
    const auto err = sink->write(fr);
    ASSERT_MATCHES(err, errors::COMMAND_REJECTED);
    EXPECT_EQ(
        err.data,
        this->double_cmd_channel.name + " returned BadInvalidState. " +
            errors::parse(UA_STATUSCODE_BADINVALIDSTATE).data
    );
    ASSERT_NIL(sink->stop());
}

/// @brief it should convert the arguments of a method that the server adds after the
/// task starts to their declared types, on the same Session.
TEST_F(TestWriteTask, testMethodAddedAfterStart) {
    server->stop();
    server->cfg.pump.reset();
    server->start();
    ASSERT_TRUE(server->wait_until_ready());
    const auto sink = this->create_method_sink(
        "NS=1;S=Pump.SetSpeed",
        {{{"source", "value"}, {"value", "5"}}}
    );
    ASSERT_NIL(sink->start());
    auto fr = this->method_command(1);
    ASSERT_OCCURRED_AS(sink->write(fr), errors::COMMAND_REJECTED);
    const auto sessions = server->sessions_created.load();
    server->add_pump(testutil::Pump{});
    ASSERT_NIL(sink->write(fr));
    EXPECT_EQ(server->sessions_created.load(), sessions);
    const auto client = ASSERT_NIL_P(connection::connect(conn_cfg, {}, "[test] "));
    const auto speed = ASSERT_NIL_P(testutil::simple_read(client, "NS=1;S=Pump.Speed"));
    EXPECT_EQ(speed.at<double>(0), 5);
    ASSERT_NIL(sink->stop());
}

/// @brief it should call a method on an object in a namespace that the server adds
/// after the task starts, on the same Session.
TEST_F(TestWriteTask, testCallsMethodOfObjectAddedLater) {
    const auto sink = this->create_method_sink(
        "NS=1;S=Pump.SetSpeed",
        {{{"source", "command"}}},
        "nsu=urn:late;s=Pump"
    );
    ASSERT_NIL(sink->start());
    auto fr = this->method_command(3);
    ASSERT_OCCURRED_AS(sink->write(fr), errors::COMMAND_REJECTED);
    const auto sessions = server->sessions_created.load();
    server->add_pump_object("urn:late");
    ASSERT_NIL(sink->write(fr));
    EXPECT_EQ(server->sessions_created.load(), sessions);
    const auto client = ASSERT_NIL_P(connection::connect(conn_cfg, {}, "[test] "));
    const auto speed = ASSERT_NIL_P(testutil::simple_read(client, "NS=1;S=Pump.Speed"));
    EXPECT_EQ(speed.at<double>(0), 3);
    ASSERT_NIL(sink->stop());
}

/// @brief it should not call a method again when the call times out, since the server
/// might have run it.
TEST_F(TestWriteTask, testDoesNotResendTimedOutCall) {
    auto timeout_cfg = conn_cfg;
    timeout_cfg.client_timeout_ms = 500;
    auto dev = ASSERT_NIL_P(ctx->client->devices.retrieve("abc123"));
    dev.properties = x::json::json::object(
        {{"connection", testutil::connection_json(timeout_cfg)}}
    );
    ASSERT_NIL(ctx->client->devices.create(dev));
    conn_pool = std::make_shared<connection::Pool>(testutil::create_store());
    const auto sink = this->create_method_sink(
        "NS=1;S=Pump.SetSpeed",
        {{{"source", "command"}}}
    );
    ASSERT_NIL(sink->start());
    const auto client = ASSERT_NIL_P(connection::connect(conn_cfg, {}, "[test] "));
    server->probe_delay = 2 * x::telem::SECOND;
    // The server waits for a request before it starts the delay, so send one first.
    ASSERT_NIL_P(testutil::simple_read(client, "NS=1;S=Pump.Speed"));
    auto fr = this->method_command(42);
    const auto err = sink->write(fr);
    server->probe_delay = x::telem::TimeSpan{};
    ASSERT_MATCHES(err, errors::COMMAND_REJECTED);
    EXPECT_EQ(
        err.data,
        this->double_cmd_channel.name +
            ": lost the connection during the call, so the method might not have run."
    );
    // The server can answer a later read before it runs the call that timed out.
    ASSERT_EVENTUALLY_EQ_WITH_TIMEOUT(
        server->method_calls.load(),
        1,
        std::chrono::seconds(5),
        std::chrono::milliseconds(10)
    );
    const auto speed = ASSERT_NIL_P(testutil::simple_read(client, "NS=1;S=Pump.Speed"));
    EXPECT_EQ(speed.at<double>(0), 42);
    EXPECT_EQ(server->method_calls, 1);
    ASSERT_NIL(sink->stop());
}

/// @brief it should reject a command when the channel gives a different number of
/// arguments than the method takes.
TEST_F(TestWriteTask, testRejectsArgumentCount) {
    const auto sink = this->create_method_sink(
        "NS=1;S=Pump.SetSpeed",
        x::json::json::array()
    );
    ASSERT_NIL(sink->start());
    auto fr = this->method_command(1);
    const auto err = sink->write(fr);
    ASSERT_MATCHES(err, errors::COMMAND_REJECTED);
    EXPECT_EQ(
        err.data,
        this->double_cmd_channel.name +
            ": the method takes 1 argument, but the channel gives 0 arguments."
    );
    ASSERT_NIL(sink->stop());
}

/// @brief it should reject a command when a fixed argument does not convert to the
/// declared type of its argument.
TEST_F(TestWriteTask, testRejectsFixedArgumentThatDoesNotConvert) {
    const auto sink = this->create_method_sink(
        "NS=1;S=Pump.Configure",
        {{{"source", "value"}, {"value", "auto"}},
         {{"source", "value"}, {"value", "fast"}}}
    );
    ASSERT_NIL(sink->start());
    auto fr = this->method_command(1);
    const auto err = sink->write(fr);
    ASSERT_MATCHES(err, errors::COMMAND_REJECTED);
    EXPECT_TRUE(err.data.starts_with(this->double_cmd_channel.name + ": argument 2: "))
        << err.data;
    ASSERT_NIL(sink->stop());
}

/// @brief it should reject a method channel that takes the command as more than one
/// argument, or names no object.
TEST_F(TestWriteTask, testRejectsInvalidMethodChannel) {
    x::json::json task_cfg = {
        {"device", "abc123"},
        {"channels",
         x::json::json::array({
             {{"type", "method"},
              {"node_id", "NS=1;S=Pump.Configure"},
              {"arguments", {{{"source", "command"}}, {{"source", "command"}}}},
              {"cmd_channel", this->double_cmd_channel.key}},
         })}
    };
    auto p = x::json::Parser(task_cfg);
    const auto parsed = WriteTaskConfig(ctx->client, p);
    ASSERT_OCCURRED_AS(p.error(), x::errors::VALIDATION);
    EXPECT_NE(
        p.error().data.find("Only one argument can take the command"),
        std::string::npos
    );
    EXPECT_NE(p.error().data.find("object_id"), std::string::npos);
}
}
