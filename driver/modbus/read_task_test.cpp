// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/defer/defer.h"
#include "x/cpp/test/test.h"

#include "driver/modbus/device/device.h"
#include "driver/modbus/mock/slave.h"
#include "driver/modbus/modbus.h"
#include "driver/modbus/read_task.h"
#include "driver/pipeline/mock/pipeline.h"

/// @brief Test fixture for Modbus read task tests.
class ModbusReadTest : public ::testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> client;
    synnax::Task task;
    std::shared_ptr<driver::task::MockContext> ctx;
    std::shared_ptr<driver::pipeline::mock::WriterFactory> mock_factory;
    synnax::channel::Channel index_channel;
    synnax::Device device;
    synnax::Rack rack;

    void SetUp() override {
        client = std::make_shared<synnax::Synnax>(new_test_client());

        // Create index channel
        index_channel = synnax::channel::Channel(
            make_unique_channel_name("time_channel"),
            x::telem::TIMESTAMP_T,
            0,
            true
        );
        ASSERT_NIL(client->channels.create(index_channel));

        // Create rack and device
        rack = ASSERT_NIL_P(client->racks.create("test_rack"));

        auto conn_cfg = driver::modbus::device::ConnectionConfig{"127.0.0.1", 1502};
        json properties{{"connection", conn_cfg.to_json()}};

        device = synnax::Device(
            "modbus_test_device",
            "modbus_test_device",
            rack.key,
            "dev1",
            "modbus",
            "Modbus Device",
            nlohmann::to_string(properties)
        );
        ASSERT_NIL(client->devices.create(device));

        ctx = std::make_shared<driver::task::MockContext>(client);
        mock_factory = std::make_shared<driver::pipeline::mock::WriterFactory>();
    }

    // Helper to create a basic task configuration
    json create_base_config() {
        return {
            {"data_saving", false},
            {"sample_rate", 25},
            {"stream_rate", 25},
            {"device", device.key},
            {"channels", json::array()}
        };
    }

    static json create_channel_config(
        const std::string &type,
        const synnax::channel::Channel &channel,
        uint16_t address,
        bool enabled = true
    ) {
        json cfg = {
            {"type", type},
            {"enabled", enabled},
            {"channel", channel.key},
            {"address", address}
        };

        // Add data_type for register inputs if the channel type requires it
        if (type == "holding_register_input" || type == "register_input") {
            cfg["data_type"] = channel.data_type.name();
        }
        return cfg;
    }
};

/// @brief it should return validation error for non-existent device.
TEST_F(ModbusReadTest, testInvalidDeviceConfig) {
    auto cfg = create_base_config();
    cfg["device"] = "non_existent_device";

    auto ch = ASSERT_NIL_P(
        client->channels.create(make_unique_channel_name("test"), x::telem::UINT8_T, true)
    );
    cfg["channels"].push_back(create_channel_config("coil_input", ch, 0));

    auto p = x::json::Parser(cfg);
    auto task_cfg = std::make_unique<driver::modbus::ReadTaskConfig>(client, p);
    ASSERT_OCCURRED_AS(p.error(), x::errors::VALIDATION);
}

/// @brief it should return validation error for non-existent channel.
TEST_F(ModbusReadTest, testInvalidChannelConfig) {
    auto cfg = create_base_config();
    synnax::channel::Channel ch;
    ch.key = 12345;
    cfg["channels"].push_back(create_channel_config("coil_input", ch, 0));
    auto p = x::json::Parser(cfg);
    auto task_cfg = std::make_unique<driver::modbus::ReadTaskConfig>(client, p);
    ASSERT_OCCURRED_AS(p.error(), x::errors::VALIDATION);
}

/// @brief it should return validation error for invalid channel type.
TEST_F(ModbusReadTest, testInvalidChannelType) {
    auto cfg = create_base_config();

    auto ch = ASSERT_NIL_P(
        client->channels.create(make_unique_channel_name("test"), x::telem::UINT8_T, true)
    );
    cfg["channels"].push_back(
        {{"type", "invalid_type"},
         {"enabled", true},
         {"channel", ch.key},
         {"address", 0}}
    );

    auto p = x::json::Parser(cfg);
    auto task_cfg = std::make_unique<driver::modbus::ReadTaskConfig>(client, p);
    ASSERT_OCCURRED_AS(p.error(), x::errors::VALIDATION);
}

/// @brief it should parse configuration with multiple channel types.
TEST_F(ModbusReadTest, testMultiChannelConfig) {
    auto cfg = create_base_config();

    // Create channels for different types
    auto coil_ch = ASSERT_NIL_P(
        client->channels.create(make_unique_channel_name("coil"), x::telem::UINT8_T, true)
    );
    auto discrete_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("discrete"),
        x::telem::UINT8_T,
        true
    ));
    auto holding_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("holding"),
        x::telem::UINT16_T,
        true
    ));
    auto input_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("input"),
        x::telem::UINT16_T,
        true
    ));

    // Add different channel types
    cfg["channels"].push_back(create_channel_config("coil_input", coil_ch, 0));
    cfg["channels"].push_back(create_channel_config("discrete_input", discrete_ch, 1));
    cfg["channels"].push_back(
        create_channel_config("holding_register_input", holding_ch, 2)
    );
    cfg["channels"].push_back(create_channel_config("register_input", input_ch, 3));

    auto p = x::json::Parser(cfg);
    auto task_cfg = std::make_unique<driver::modbus::ReadTaskConfig>(client, p);
    ASSERT_NIL(p.error());
}

/// @brief it should read coil values from Modbus device.
TEST(ReadTask, testBasicReadTask) {
    driver::modbus::mock::SlaveConfig slave_cfg;
    // Set up coil at address 0 with value 1 and ensure we have at least 1 coil mapped
    slave_cfg.coils[0] = 1;
    slave_cfg.coils[1] = 0; // Add an extra coil to ensure proper mapping size
    slave_cfg.host = "127.0.0.1";
    slave_cfg.port = 1502;

    auto slave = driver::modbus::mock::Slave(slave_cfg);
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });

    auto index_channel = synnax::channel::Channel(
        make_unique_channel_name("time_channel"),
        x::telem::TIMESTAMP_T,
        0,
        true
    );

    auto data_channel = synnax::channel::Channel(
        make_unique_channel_name("data_channel"),
        x::telem::UINT8_T,
        index_channel.key,
        false
    );

    auto client = std::make_shared<synnax::Synnax>(new_test_client());

    ASSERT_NIL(client->channels.create(index_channel));
    data_channel.index = index_channel.key;
    ASSERT_NIL(client->channels.create(data_channel));

    auto rack = ASSERT_NIL_P(client->racks.create("cat"));

    auto conn_cfg = driver::modbus::device::ConnectionConfig{"127.0.0.1", 1502};
    json properties{{"connection", conn_cfg.to_json()}};
    synnax::Device dev(
        "my_modbus_lover",
        "my_mobdus_lover",
        rack.key,
        "dev1",
        "modbus",
        "Modbus Device",
        nlohmann::to_string(properties)
    );

    ASSERT_NIL(client->devices.create(dev));

    auto tsk = synnax::Task(rack.key, "my_task", "modbus_read", "");

    json j{
        {"data_saving", false},
        {"sample_rate", 25},
        {"stream_rate", 25},
        {"device", dev.key},
        {"channels",
         json::array(
             {{{"type", "coil_input"},
               {"enabled", true},
               {"channel", data_channel.key},
               {"address", 0}}}
         )}
    };
    auto p = x::json::Parser(j);
    auto cfg = std::make_unique<driver::modbus::ReadTaskConfig>(client, p);
    ASSERT_NIL(p.error());

    auto ctx = std::make_shared<driver::task::MockContext>(client);
    auto factory = std::make_shared<driver::pipeline::mock::WriterFactory>();

    auto devs = std::make_shared<driver::modbus::device::Manager>();

    auto modbus_dev = ASSERT_NIL_P(
        devs->acquire(driver::modbus::device::ConnectionConfig{"127.0.0.1", 1502})
    );

    auto task = driver::task::common::ReadTask(
        tsk,
        ctx,
        x::breaker::default_config(tsk.name),
        std::make_unique<driver::modbus::ReadTaskSource>(modbus_dev, std::move(*cfg)),
        factory
    );

    task.start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    const auto first_state = ctx->statuses[0];
    EXPECT_EQ(first_state.key, tsk.status_key());
    EXPECT_EQ(first_state.details.cmd, "start_cmd");
    EXPECT_EQ(first_state.variant, status::variant::SUCCESS);
    EXPECT_EQ(first_state.details.task, tsk.key);
    EXPECT_EQ(first_state.message, "Task started successfully");
    ASSERT_EVENTUALLY_GE(factory->writer_opens, 1);
    task.stop("stop_cmd", true);
    ASSERT_EQ(ctx->statuses.size(), 2);
    const auto second_state = ctx->statuses[1];
    EXPECT_EQ(second_state.key, tsk.status_key());
    EXPECT_EQ(second_state.details.cmd, "stop_cmd");
    EXPECT_EQ(second_state.variant, status::variant::SUCCESS);
    EXPECT_EQ(second_state.details.task, tsk.key);
    EXPECT_EQ(second_state.message, "Task stopped successfully");

    ASSERT_GE(factory->writes->size(), 1);
    auto &fr = factory->writes->at(0);
    ASSERT_EQ(fr.size(), 2);
    ASSERT_EQ(fr.length(), 1);
    ASSERT_EQ(fr.contains(data_channel.key), true);
    ASSERT_EQ(fr.contains(index_channel.key), true);
    ASSERT_EQ(fr.at<uint8_t>(data_channel.key, 0), 1);
    ASSERT_GE(fr.at<uint64_t>(index_channel.key, 0), 0);
}

/// @brief it should read discrete input values from Modbus device.
TEST_F(ModbusReadTest, testDiscreteInputRead) {
    // Set up mock slave with discrete input values
    driver::modbus::mock::SlaveConfig slave_cfg;
    slave_cfg.discrete_inputs[1] = 1; // Set discrete input 1 to HIGH
    slave_cfg.discrete_inputs[2] = 0; // Set discrete input 2 to LOW
    slave_cfg.host = "127.0.0.1";
    slave_cfg.port = 1502;

    auto slave = driver::modbus::mock::Slave(slave_cfg);
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });

    // Create data channel
    auto data_channel = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("discrete_input"),
        x::telem::UINT8_T,
        index_channel.key,
        false
    ));

    // Create task configuration
    auto cfg = create_base_config();
    cfg["channels"].push_back(create_channel_config("discrete_input", data_channel, 1));

    auto p = x::json::Parser(cfg);
    auto task_cfg = std::make_unique<driver::modbus::ReadTaskConfig>(client, p);
    ASSERT_NIL(p.error());

    auto devs = std::make_shared<driver::modbus::device::Manager>();
    auto modbus_dev = ASSERT_NIL_P(
        devs->acquire(driver::modbus::device::ConnectionConfig{"127.0.0.1", 1502})
    );

    auto task = driver::task::common::ReadTask(
        synnax::Task(rack.key, "discrete_test", "modbus_read", ""),
        ctx,
        x::breaker::default_config("discrete_test"),
        std::make_unique<driver::modbus::ReadTaskSource>(modbus_dev, std::move(*task_cfg)),
        mock_factory
    );

    task.start("start_cmd");
    ASSERT_EVENTUALLY_GE(mock_factory->writes->size(), 1);
    task.stop("stop_cmd", true);

    auto &fr = mock_factory->writes->at(0);
    ASSERT_EQ(fr.size(), 2); // Data channel + index channel
    ASSERT_EQ(fr.length(), 1);
    ASSERT_EQ(fr.at<uint8_t>(data_channel.key, 0), 1);
}

/// @brief it should read holding register values from Modbus device.
TEST_F(ModbusReadTest, testHoldingRegisterRead) {
    // Set up mock slave with holding register values
    driver::modbus::mock::SlaveConfig slave_cfg;
    slave_cfg.holding_registers[0] = 12345; // Set holding register 0
    slave_cfg.host = "127.0.0.1";
    slave_cfg.port = 1502;

    auto slave = driver::modbus::mock::Slave(slave_cfg);
    ASSERT_NIL(slave.start());
    std::this_thread::sleep_for(std::chrono::milliseconds(100));
    x::defer::defer stop_slave([&slave] { slave.stop(); });

    // Create data channel
    auto data_channel = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("holding_register"),
        x::telem::UINT16_T, // Holding registers are 16-bit
        index_channel.key,
        false
    ));

    // Create task configuration
    auto cfg = create_base_config();
    cfg["channels"].push_back(
        create_channel_config("holding_register_input", data_channel, 0)
    );

    auto p = x::json::Parser(cfg);
    auto task_cfg = std::make_unique<driver::modbus::ReadTaskConfig>(client, p);
    ASSERT_NIL(p.error());

    auto devs = std::make_shared<driver::modbus::device::Manager>();
    auto modbus_dev = ASSERT_NIL_P(
        devs->acquire(driver::modbus::device::ConnectionConfig{"127.0.0.1", 1502})
    );

    auto task = driver::task::common::ReadTask(
        synnax::Task(rack.key, "holding_test", "modbus_read", ""),
        ctx,
        x::breaker::default_config("holding_test"),
        std::make_unique<driver::modbus::ReadTaskSource>(modbus_dev, std::move(*task_cfg)),
        mock_factory
    );

    task.start("start_cmd");
    ASSERT_EVENTUALLY_GE(mock_factory->writes->size(), 1);
    task.stop("stop_cmd", true);

    auto &fr = mock_factory->writes->at(0);
    ASSERT_EQ(fr.size(), 2);
    ASSERT_EQ(fr.length(), 1);
    ASSERT_EQ(fr.at<uint16_t>(data_channel.key, 0), 12345);
}

/// @brief it should read multiple channel types simultaneously.
TEST_F(ModbusReadTest, testMultiChannelRead) {
    // Set up mock slave with various register values
    driver::modbus::mock::SlaveConfig slave_cfg;
    slave_cfg.coils[0] = 1;
    slave_cfg.discrete_inputs[1] = 1;
    slave_cfg.holding_registers[2] = 12345;
    slave_cfg.input_registers[3] = 54321;
    slave_cfg.host = "127.0.0.1";
    slave_cfg.port = 1502;

    auto slave = driver::modbus::mock::Slave(slave_cfg);
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });

    // Create channels for different types
    auto coil_ch = ASSERT_NIL_P(
        client->channels.create(make_unique_channel_name("coil"), x::telem::UINT8_T, true)
    );
    auto discrete_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("discrete"),
        x::telem::UINT8_T,
        true
    ));
    auto holding_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("holding"),
        x::telem::UINT16_T,
        true
    ));
    auto input_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("input"),
        x::telem::UINT16_T,
        true
    ));

    // Create task configuration with all channel types
    auto cfg = create_base_config();
    cfg["channels"].push_back(create_channel_config("coil_input", coil_ch, 0));
    cfg["channels"].push_back(create_channel_config("discrete_input", discrete_ch, 1));
    cfg["channels"].push_back(
        create_channel_config("holding_register_input", holding_ch, 2)
    );
    cfg["channels"].push_back(create_channel_config("register_input", input_ch, 3));

    auto p = x::json::Parser(cfg);
    auto task_cfg = std::make_unique<driver::modbus::ReadTaskConfig>(client, p);
    ASSERT_NIL(p.error());

    auto devs = std::make_shared<driver::modbus::device::Manager>();
    auto modbus_dev = ASSERT_NIL_P(
        devs->acquire(driver::modbus::device::ConnectionConfig{"127.0.0.1", 1502})
    );

    auto task = driver::task::common::ReadTask(
        synnax::Task(rack.key, "multi_test", "modbus_read", ""),
        ctx,
        x::breaker::default_config("multi_test"),
        std::make_unique<driver::modbus::ReadTaskSource>(modbus_dev, std::move(*task_cfg)),
        mock_factory
    );

    task.start("start_cmd");
    ASSERT_EVENTUALLY_GE(mock_factory->writes->size(), 1);
    task.stop("stop_cmd", true);

    auto &fr = mock_factory->writes->at(0);
    ASSERT_EQ(fr.size(), 4); // 4 data channels + 1 index channel
    ASSERT_EQ(fr.length(), 1);
    ASSERT_EQ(fr.at<uint8_t>(coil_ch.key, 0), 1);
    ASSERT_EQ(fr.at<uint8_t>(discrete_ch.key, 0), 1);
    ASSERT_EQ(fr.at<uint16_t>(holding_ch.key, 0), 12345);
    ASSERT_EQ(fr.at<uint16_t>(input_ch.key, 0), 54321);
}

/// Regression test to ensure enable_auto_commit is set to true in WriterConfig.
/// This prevents data from being written but not committed, making it unavailable for
/// reads.
TEST_F(ModbusReadTest, testModbusDriverSetsAutoCommitTrue) {
    auto cfg = create_base_config();
    cfg["data_saving"] = true;

    auto coil_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("coil"),
        x::telem::UINT8_T,
        index_channel.key
    ));
    cfg["channels"].push_back(create_channel_config("coil_input", coil_ch, 0));

    auto p = x::json::Parser(cfg);
    auto task_cfg = std::make_unique<driver::modbus::ReadTaskConfig>(client, p);
    ASSERT_NIL(p.error());

    // Verify that writer_config has enable_auto_commit set to true
    auto writer_cfg = task_cfg->writer_config();
    ASSERT_TRUE(writer_cfg.enable_auto_commit);
}

/// Regression test for buffer size calculation bug with UINT8 registers.
/// This test ensures that multiple sequential UINT8 input registers are read correctly,
/// especially the last channel which was previously always zero due to an off-by-one
/// error in the buffer size calculation (density / 2 should be ceiling division).
TEST_F(ModbusReadTest, testMultipleUint8InputRegisters) {
    // Set up mock slave with multiple UINT8 input register values
    driver::modbus::mock::SlaveConfig slave_cfg;
    slave_cfg.input_registers[0] = static_cast<uint8_t>(100);
    slave_cfg.input_registers[1] = static_cast<uint8_t>(150);
    slave_cfg.input_registers[2] = static_cast<uint8_t>(200);
    slave_cfg.host = "127.0.0.1";
    slave_cfg.port = 1502;

    auto slave = driver::modbus::mock::Slave(slave_cfg);
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });

    // Create three UINT8 channels for sequential input registers
    auto input0 = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("input_reg_0"),
        x::telem::UINT8_T,
        index_channel.key
    ));
    auto input1 = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("input_reg_1"),
        x::telem::UINT8_T,
        index_channel.key
    ));
    auto input2 = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("input_reg_2"),
        x::telem::UINT8_T,
        index_channel.key
    ));

    // Create task configuration with three sequential UINT8 input registers
    auto cfg = create_base_config();
    cfg["channels"].push_back(create_channel_config("register_input", input0, 0));
    cfg["channels"].push_back(create_channel_config("register_input", input1, 1));
    cfg["channels"].push_back(create_channel_config("register_input", input2, 2));

    auto p = x::json::Parser(cfg);
    auto task_cfg = std::make_unique<driver::modbus::ReadTaskConfig>(client, p);
    ASSERT_NIL(p.error());

    auto devs = std::make_shared<driver::modbus::device::Manager>();
    auto modbus_dev = ASSERT_NIL_P(
        devs->acquire(driver::modbus::device::ConnectionConfig{"127.0.0.1", 1502})
    );

    auto task = driver::task::common::ReadTask(
        synnax::Task(rack.key, "uint8_test", "modbus_read", ""),
        ctx,
        x::breaker::default_config("uint8_test"),
        std::make_unique<driver::modbus::ReadTaskSource>(modbus_dev, std::move(*task_cfg)),
        mock_factory
    );

    task.start("start_cmd");
    ASSERT_EVENTUALLY_GE(mock_factory->writes->size(), 1);
    task.stop("stop_cmd", true);

    auto &fr = mock_factory->writes->at(0);
    ASSERT_EQ(fr.size(), 4); // 3 data channels + 1 index channel
    ASSERT_EQ(fr.length(), 1);
    // All three channels should have correct values, including the last one
    ASSERT_EQ(fr.at<uint8_t>(input0.key, 0), 100);
    ASSERT_EQ(fr.at<uint8_t>(input1.key, 0), 150);
    ASSERT_EQ(fr.at<uint8_t>(input2.key, 0), 200);
}

/// Regression test for buffer size calculation bug with UINT8 holding registers.
/// Similar to testMultipleUint8InputRegisters but tests holding registers instead.
TEST_F(ModbusReadTest, testMultipleUint8HoldingRegisters) {
    // Set up mock slave with multiple UINT8 holding register values
    driver::modbus::mock::SlaveConfig slave_cfg;
    slave_cfg.holding_registers[0] = static_cast<uint8_t>(50);
    slave_cfg.holding_registers[1] = static_cast<uint8_t>(75);
    slave_cfg.holding_registers[2] = static_cast<uint8_t>(125);
    slave_cfg.host = "127.0.0.1";
    slave_cfg.port = 1502;

    auto slave = driver::modbus::mock::Slave(slave_cfg);
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });

    // Create three UINT8 channels for sequential holding registers
    auto holding0 = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("holding_reg_0"),
        x::telem::UINT8_T,
        index_channel.key
    ));
    auto holding1 = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("holding_reg_1"),
        x::telem::UINT8_T,
        index_channel.key
    ));
    auto holding2 = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("holding_reg_2"),
        x::telem::UINT8_T,
        index_channel.key
    ));

    // Create task configuration with three sequential UINT8 holding registers
    auto cfg = create_base_config();
    cfg["channels"].push_back(
        create_channel_config("holding_register_input", holding0, 0)
    );
    cfg["channels"].push_back(
        create_channel_config("holding_register_input", holding1, 1)
    );
    cfg["channels"].push_back(
        create_channel_config("holding_register_input", holding2, 2)
    );

    auto p = x::json::Parser(cfg);
    auto task_cfg = std::make_unique<driver::modbus::ReadTaskConfig>(client, p);
    ASSERT_NIL(p.error());

    auto devs = std::make_shared<driver::modbus::device::Manager>();
    auto modbus_dev = ASSERT_NIL_P(
        devs->acquire(driver::modbus::device::ConnectionConfig{"127.0.0.1", 1502})
    );

    auto task = driver::task::common::ReadTask(
        synnax::Task(rack.key, "uint8_holding_test", "modbus_read", ""),
        ctx,
        x::breaker::default_config("uint8_holding_test"),
        std::make_unique<driver::modbus::ReadTaskSource>(modbus_dev, std::move(*task_cfg)),
        mock_factory
    );

    task.start("start_cmd");
    ASSERT_EVENTUALLY_GE(mock_factory->writes->size(), 1);
    task.stop("stop_cmd", true);

    auto &fr = mock_factory->writes->at(0);
    ASSERT_EQ(fr.size(), 4); // 3 data channels + 1 index channel
    ASSERT_EQ(fr.length(), 1);
    // All three channels should have correct values, including the last one
    ASSERT_EQ(fr.at<uint8_t>(holding0.key, 0), 50);
    ASSERT_EQ(fr.at<uint8_t>(holding1.key, 0), 75);
    ASSERT_EQ(
        fr.at<uint8_t>(holding2.key, 0),
        125
    ); // This would have been 0 before the fix
}

/// Test that auto_start=true causes the task to start automatically
TEST_F(ModbusReadTest, testAutoStartTrue) {
    // Set up mock slave
    driver::modbus::mock::SlaveConfig slave_cfg;
    slave_cfg.input_registers[0] = 42;
    slave_cfg.host = "127.0.0.1";
    slave_cfg.port = 1502;

    auto slave = driver::modbus::mock::Slave(slave_cfg);
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });

    // Create data channel
    auto data_channel = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("input_reg"),
        x::telem::UINT8_T,
        index_channel.key,
        false
    ));

    // Create task with auto_start=true
    json config{
        {"data_saving", false},
        {"sample_rate", 25},
        {"stream_rate", 25},
        {"device", device.key},
        {"auto_start", true}, // Enable auto-start
        {"channels",
         json::array(
             {{{"type", "register_input"},
               {"enabled", true},
               {"channel", data_channel.key},
               {"address", 0},
               {"data_type", "uint8"}}}
         )}
    };

    task = synnax::Task(rack.key, "test_task", "modbus_read", config.dump(), false);

    // Configure task through factory
    auto factory = driver::modbus::Factory();
    auto [configured_task, ok] = factory.configure_task(ctx, task);

    ASSERT_TRUE(ok);
    ASSERT_NE(configured_task, nullptr);

    // Task should have auto-started - check that a start status was sent
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    bool found_start = false;
    for (const auto &s: ctx->statuses) {
        if (s.details.running && s.variant == status::variant::SUCCESS) {
            found_start = true;
            break;
        }
    }
    ASSERT_TRUE(found_start);

    // Stop the task to clean up
    driver::task::Command stop_cmd(task.key, "stop", {});
    configured_task->exec(stop_cmd);
}

/// Test that auto_start=false does NOT start the task automatically
TEST_F(ModbusReadTest, testAutoStartFalse) {
    // Set up mock slave
    driver::modbus::mock::SlaveConfig slave_cfg;
    slave_cfg.input_registers[0] = 99;
    slave_cfg.host = "127.0.0.1";
    slave_cfg.port = 1502;

    auto slave = driver::modbus::mock::Slave(slave_cfg);
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });

    // Create data channel
    auto data_channel = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("input_reg_2"),
        x::telem::UINT8_T,
        index_channel.key,
        false
    ));

    // Create task with auto_start=false
    json config{
        {"data_saving", false},
        {"sample_rate", 25},
        {"stream_rate", 25},
        {"device", device.key},
        {"auto_start", false}, // Disable auto-start
        {"channels",
         json::array(
             {{{"type", "register_input"},
               {"enabled", true},
               {"channel", data_channel.key},
               {"address", 0},
               {"data_type", "uint8"}}}
         )}
    };

    task = synnax::Task(
        rack.key,
        "test_task_no_auto",
        "modbus_read",
        config.dump(),
        false
    );

    // Configure task through factory
    auto factory = driver::modbus::Factory();
    auto [configured_task, ok] = factory.configure_task(ctx, task);

    ASSERT_TRUE(ok);
    ASSERT_NE(configured_task, nullptr);

    // Task should NOT have auto-started - check that the status is "configured" not
    // "running"
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    const auto &initial_state = ctx->statuses[0];
    ASSERT_FALSE(initial_state.details.running);
    ASSERT_EQ(initial_state.variant, status::variant::SUCCESS);
    ASSERT_EQ(initial_state.message, "Task configured successfully");

    // Manually start the task
    driver::task::Command start_cmd(task.key, "start", {});
    configured_task->exec(start_cmd);

    // Now task should be running
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 2);
    bool found_start = false;
    for (const auto &s: ctx->statuses) {
        if (s.details.running && s.variant == status::variant::SUCCESS) {
            found_start = true;
            break;
        }
    }
    ASSERT_TRUE(found_start);

    // Stop the task to clean up
    driver::task::Command stop_cmd(task.key, "stop", {});
    configured_task->exec(stop_cmd);
}
