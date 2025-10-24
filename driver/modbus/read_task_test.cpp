// Copyright 2025 Synnax Labs, Inc.
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
#include "x/cpp/xtest/xtest.h"

#include "driver/modbus/device/device.h"
#include "driver/modbus/mock/slave.h"
#include "driver/modbus/read_task.h"
#include "driver/pipeline/mock/pipeline.h"

class ModbusReadTest : public ::testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> sy;
    synnax::Task task;
    std::shared_ptr<task::MockContext> ctx;
    std::shared_ptr<pipeline::mock::WriterFactory> mock_factory;
    synnax::Channel index_channel;
    synnax::Device device;
    synnax::Rack rack;

    void SetUp() override {
        sy = std::make_shared<synnax::Synnax>(new_test_client());

        // Create index channel
        index_channel = synnax::Channel("time_channel", telem::TIMESTAMP_T, 0, true);
        ASSERT_NIL(sy->channels.create(index_channel));

        // Create rack and device
        rack = ASSERT_NIL_P(sy->hardware.create_rack("test_rack"));

        auto conn_cfg = modbus::device::ConnectionConfig{"127.0.0.1", 1502};
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
        ASSERT_NIL(sy->hardware.create_device(device));

        ctx = std::make_shared<task::MockContext>(sy);
        mock_factory = std::make_shared<pipeline::mock::WriterFactory>();
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
        const synnax::Channel &channel,
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

TEST_F(ModbusReadTest, testInvalidDeviceConfig) {
    auto cfg = create_base_config();
    cfg["device"] = "non_existent_device";

    auto ch = ASSERT_NIL_P(sy->channels.create("test", telem::UINT8_T, true));
    cfg["channels"].push_back(create_channel_config("coil_input", ch, 0));

    auto p = xjson::Parser(cfg);
    auto task_cfg = std::make_unique<modbus::ReadTaskConfig>(sy, p);
    ASSERT_OCCURRED_AS(p.error(), xerrors::VALIDATION);
}

TEST_F(ModbusReadTest, testInvalidChannelConfig) {
    auto cfg = create_base_config();
    synnax::Channel ch;
    ch.key = 12345;
    cfg["channels"].push_back(create_channel_config("coil_input", ch, 0));
    auto p = xjson::Parser(cfg);
    auto task_cfg = std::make_unique<modbus::ReadTaskConfig>(sy, p);
    ASSERT_OCCURRED_AS(p.error(), xerrors::VALIDATION);
}

TEST_F(ModbusReadTest, testInvalidChannelType) {
    auto cfg = create_base_config();

    auto ch = ASSERT_NIL_P(sy->channels.create("test", telem::UINT8_T, true));
    cfg["channels"].push_back(
        {{"type", "invalid_type"},
         {"enabled", true},
         {"channel", ch.key},
         {"address", 0}}
    );

    auto p = xjson::Parser(cfg);
    auto task_cfg = std::make_unique<modbus::ReadTaskConfig>(sy, p);
    ASSERT_OCCURRED_AS(p.error(), xerrors::VALIDATION);
}

TEST_F(ModbusReadTest, testMultiChannelConfig) {
    auto cfg = create_base_config();

    // Create channels for different types
    auto coil_ch = ASSERT_NIL_P(sy->channels.create("coil", telem::UINT8_T, true));
    auto discrete_ch = ASSERT_NIL_P(
        sy->channels.create("discrete", telem::UINT8_T, true)
    );
    auto holding_ch = ASSERT_NIL_P(
        sy->channels.create("holding", telem::UINT16_T, true)
    );
    auto input_ch = ASSERT_NIL_P(sy->channels.create("input", telem::UINT16_T, true));

    // Add different channel types
    cfg["channels"].push_back(create_channel_config("coil_input", coil_ch, 0));
    cfg["channels"].push_back(create_channel_config("discrete_input", discrete_ch, 1));
    cfg["channels"].push_back(
        create_channel_config("holding_register_input", holding_ch, 2)
    );
    cfg["channels"].push_back(create_channel_config("register_input", input_ch, 3));

    auto p = xjson::Parser(cfg);
    auto task_cfg = std::make_unique<modbus::ReadTaskConfig>(sy, p);
    ASSERT_NIL(p.error());
}

TEST(ReadTask, testBasicReadTask) {
    modbus::mock::SlaveConfig slave_cfg;
    // Set up coil at address 0 with value 1 and ensure we have at least 1 coil mapped
    slave_cfg.coils[0] = 1;
    slave_cfg.coils[1] = 0; // Add an extra coil to ensure proper mapping size
    slave_cfg.host = "127.0.0.1";
    slave_cfg.port = 1502;

    auto slave = modbus::mock::Slave(slave_cfg);
    ASSERT_NIL(slave.start());
    x::defer stop_slave([&slave] { slave.stop(); });

    auto index_channel = synnax::Channel("time_channel", telem::TIMESTAMP_T, 0, true);

    auto data_channel = synnax::Channel(
        "data_channel",
        telem::UINT8_T,
        index_channel.key,
        false
    );

    auto sy = std::make_shared<synnax::Synnax>(new_test_client());

    ASSERT_NIL(sy->channels.create(index_channel));
    data_channel.index = index_channel.key;
    ASSERT_NIL(sy->channels.create(data_channel));

    auto rack = ASSERT_NIL_P(sy->hardware.create_rack("cat"));

    auto conn_cfg = modbus::device::ConnectionConfig{"127.0.0.1", 1502};
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

    ASSERT_NIL(sy->hardware.create_device(dev));

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
    auto p = xjson::Parser(j);
    auto cfg = std::make_unique<modbus::ReadTaskConfig>(sy, p);
    ASSERT_NIL(p.error());

    auto ctx = std::make_shared<task::MockContext>(sy);
    auto factory = std::make_shared<pipeline::mock::WriterFactory>();

    auto devs = std::make_shared<modbus::device::Manager>();

    auto modbus_dev = ASSERT_NIL_P(
        devs->acquire(modbus::device::ConnectionConfig{"127.0.0.1", 1502})
    );

    auto task = common::ReadTask(
        tsk,
        ctx,
        breaker::default_config(tsk.name),
        std::make_unique<modbus::ReadTaskSource>(modbus_dev, std::move(*cfg)),
        factory
    );

    task.start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->states.size(), 1);
    const auto first_state = ctx->states[0];
    EXPECT_EQ(first_state.key, "start_cmd");
    EXPECT_EQ(first_state.variant, "success");
    EXPECT_EQ(first_state.details.task, tsk.key);
    EXPECT_EQ(first_state.message, "Task started successfully");
    ASSERT_EVENTUALLY_GE(factory->writer_opens, 1);
    task.stop("stop_cmd", true);
    ASSERT_EQ(ctx->states.size(), 2);
    const auto second_state = ctx->states[1];
    EXPECT_EQ(second_state.key, "stop_cmd");
    EXPECT_EQ(second_state.variant, "success");
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

TEST_F(ModbusReadTest, testDiscreteInputRead) {
    // Set up mock slave with discrete input values
    modbus::mock::SlaveConfig slave_cfg;
    slave_cfg.discrete_inputs[1] = 1; // Set discrete input 1 to HIGH
    slave_cfg.discrete_inputs[2] = 0; // Set discrete input 2 to LOW
    slave_cfg.host = "127.0.0.1";
    slave_cfg.port = 1502;

    auto slave = modbus::mock::Slave(slave_cfg);
    ASSERT_NIL(slave.start());
    x::defer stop_slave([&slave] { slave.stop(); });

    // Create data channel
    auto data_channel = ASSERT_NIL_P(
        sy->channels.create("discrete_input", telem::UINT8_T, index_channel.key, false)
    );

    // Create task configuration
    auto cfg = create_base_config();
    cfg["channels"].push_back(create_channel_config("discrete_input", data_channel, 1));

    auto p = xjson::Parser(cfg);
    auto task_cfg = std::make_unique<modbus::ReadTaskConfig>(sy, p);
    ASSERT_NIL(p.error());

    auto devs = std::make_shared<modbus::device::Manager>();
    auto modbus_dev = ASSERT_NIL_P(
        devs->acquire(modbus::device::ConnectionConfig{"127.0.0.1", 1502})
    );

    auto task = common::ReadTask(
        synnax::Task(rack.key, "discrete_test", "modbus_read", ""),
        ctx,
        breaker::default_config("discrete_test"),
        std::make_unique<modbus::ReadTaskSource>(modbus_dev, std::move(*task_cfg)),
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

TEST_F(ModbusReadTest, testHoldingRegisterRead) {
    // Set up mock slave with holding register values
    modbus::mock::SlaveConfig slave_cfg;
    slave_cfg.holding_registers[0] = 12345; // Set holding register 0
    slave_cfg.host = "127.0.0.1";
    slave_cfg.port = 1502;

    auto slave = modbus::mock::Slave(slave_cfg);
    ASSERT_NIL(slave.start());
    std::this_thread::sleep_for(std::chrono::milliseconds(100));
    x::defer stop_slave([&slave] { slave.stop(); });

    // Create data channel
    auto data_channel = ASSERT_NIL_P(sy->channels.create(
        "holding_register",
        telem::UINT16_T, // Holding registers are 16-bit
        index_channel.key,
        false
    ));

    // Create task configuration
    auto cfg = create_base_config();
    cfg["channels"].push_back(
        create_channel_config("holding_register_input", data_channel, 0)
    );

    auto p = xjson::Parser(cfg);
    auto task_cfg = std::make_unique<modbus::ReadTaskConfig>(sy, p);
    ASSERT_NIL(p.error());

    auto devs = std::make_shared<modbus::device::Manager>();
    auto modbus_dev = ASSERT_NIL_P(
        devs->acquire(modbus::device::ConnectionConfig{"127.0.0.1", 1502})
    );

    auto task = common::ReadTask(
        synnax::Task(rack.key, "holding_test", "modbus_read", ""),
        ctx,
        breaker::default_config("holding_test"),
        std::make_unique<modbus::ReadTaskSource>(modbus_dev, std::move(*task_cfg)),
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

TEST_F(ModbusReadTest, testMultiChannelRead) {
    // Set up mock slave with various register values
    modbus::mock::SlaveConfig slave_cfg;
    slave_cfg.coils[0] = 1;
    slave_cfg.discrete_inputs[1] = 1;
    slave_cfg.holding_registers[2] = 12345;
    slave_cfg.input_registers[3] = 54321;
    slave_cfg.host = "127.0.0.1";
    slave_cfg.port = 1502;

    auto slave = modbus::mock::Slave(slave_cfg);
    ASSERT_NIL(slave.start());
    x::defer stop_slave([&slave] { slave.stop(); });

    // Create channels for different types
    auto coil_ch = ASSERT_NIL_P(sy->channels.create("coil", telem::UINT8_T, true));
    auto discrete_ch = ASSERT_NIL_P(
        sy->channels.create("discrete", telem::UINT8_T, true)
    );
    auto holding_ch = ASSERT_NIL_P(
        sy->channels.create("holding", telem::UINT16_T, true)
    );
    auto input_ch = ASSERT_NIL_P(sy->channels.create("input", telem::UINT16_T, true));

    // Create task configuration with all channel types
    auto cfg = create_base_config();
    cfg["channels"].push_back(create_channel_config("coil_input", coil_ch, 0));
    cfg["channels"].push_back(create_channel_config("discrete_input", discrete_ch, 1));
    cfg["channels"].push_back(
        create_channel_config("holding_register_input", holding_ch, 2)
    );
    cfg["channels"].push_back(create_channel_config("register_input", input_ch, 3));

    auto p = xjson::Parser(cfg);
    auto task_cfg = std::make_unique<modbus::ReadTaskConfig>(sy, p);
    ASSERT_NIL(p.error());

    auto devs = std::make_shared<modbus::device::Manager>();
    auto modbus_dev = ASSERT_NIL_P(
        devs->acquire(modbus::device::ConnectionConfig{"127.0.0.1", 1502})
    );

    auto task = common::ReadTask(
        synnax::Task(rack.key, "multi_test", "modbus_read", ""),
        ctx,
        breaker::default_config("multi_test"),
        std::make_unique<modbus::ReadTaskSource>(modbus_dev, std::move(*task_cfg)),
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

    auto coil_ch = ASSERT_NIL_P(
        sy->channels.create("coil", telem::UINT8_T, index_channel.key)
    );
    cfg["channels"].push_back(create_channel_config("coil_input", coil_ch, 0));

    auto p = xjson::Parser(cfg);
    auto task_cfg = std::make_unique<modbus::ReadTaskConfig>(sy, p);
    ASSERT_NIL(p.error());

    // Verify that writer_config has enable_auto_commit set to true
    auto writer_cfg = task_cfg->writer_config();
    ASSERT_TRUE(writer_cfg.enable_auto_commit);
}
