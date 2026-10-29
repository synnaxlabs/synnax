// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <iomanip>
#include <iostream>
#include <optional>
#include <thread>

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/defer/defer.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/modbus/device/device.h"
#include "driver/modbus/modbus.h"
#include "driver/modbus/read_task.h"
#include "driver/modbus/testutil/slave.h"
#include "driver/pipeline/testutil/pipeline.h"

namespace driver::modbus {
/// @brief Allowed relative error of a task's achieved sample rate.
constexpr double RATE_TOLERANCE = 0.10;
/// @brief Wall time each rate runs for.
const auto RATE_SPAN = 1 * x::telem::SECOND;

/// @brief Logs the rate a task held and checks it against the tolerance.
void expect_rate(const double measured, const int rate_hz) {
    const double error = (measured - rate_hz) / rate_hz * 100;
    std::cout << std::fixed << std::setprecision(1);
    std::cout << rate_hz << " Hz: " << measured << " Hz measured (" << error << "%)\n";
    EXPECT_NEAR(measured, rate_hz, rate_hz * RATE_TOLERANCE)
        << "at " << rate_hz << " Hz";
}

/// @brief Test fixture for Modbus read task tests.
class ModbusReadTest : public ::testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> client;
    synnax::task::Task task;
    std::shared_ptr<task::MockContext> ctx;
    std::shared_ptr<pipeline::testutil::WriterFactory> mock_factory;
    synnax::channel::Channel index_channel;
    synnax::device::Device device;
    synnax::rack::Rack rack;

    void SetUp() override {
        client = std::make_shared<synnax::Synnax>(new_test_client());

        // Create index channel
        index_channel = synnax::channel::Channel{
            .name = make_unique_channel_name("time_channel"),
            .data_type = x::telem::TIMESTAMP_T,
            .is_index = true,
            .index = 0
        };
        ASSERT_NIL(client->channels.create(index_channel));

        // Create rack and device
        rack = ASSERT_NIL_P(client->racks.create("test_rack"));

        auto conn_cfg = device::ConnectionConfig{"127.0.0.1", 1502};
        x::json::json properties{{"connection", conn_cfg.to_json()}};

        device = synnax::device::Device{
            .key = "modbus_test_device",
            .rack = rack.key,
            .location = "dev1",
            .make = MAKE,
            .model = "Modbus Device",
            .name = "modbus_test_device",
            .properties = properties
        };
        ASSERT_NIL(client->devices.create(device));

        ctx = std::make_shared<driver::task::MockContext>(client);
        mock_factory = std::make_shared<driver::pipeline::testutil::WriterFactory>();
    }

    // Helper to create a basic task configuration
    x::json::json create_base_config() {
        return {
            {"data_saving_disabled", true},
            {"sample_rate", 25},
            {"stream_rate", 25},
            {"device", device.key},
            {"channels", x::json::json::array()}
        };
    }

    static x::json::json create_channel_config(
        const std::string &type,
        const synnax::channel::Channel &channel,
        uint16_t address,
        bool disabled = false
    ) {
        x::json::json cfg = {
            {"type", type},
            {"disabled", disabled},
            {"channel", channel.key},
            {"address", address}
        };

        // Add data_type for register inputs if the channel type requires it
        if (type == "holding_register" || type == "input_register") {
            cfg["data_type"] = channel.data_type.name();
        }
        return cfg;
    }
};

/// @brief it should return validation error for non-existent device.
TEST_F(ModbusReadTest, testInvalidDeviceConfig) {
    auto cfg = create_base_config();
    cfg["device"] = "non_existent_device";

    auto ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("test"),
        x::telem::UINT8_T,
        true
    ));
    cfg["channels"].push_back(create_channel_config("coil", ch, 0));

    auto p = x::json::Parser(cfg);
    auto task_cfg = std::make_unique<ReadTaskConfig>(client, p);
    ASSERT_OCCURRED_AS(p.error(), x::errors::VALIDATION);
}

/// @brief it should return validation error for non-existent channel.
TEST_F(ModbusReadTest, testInvalidChannelConfig) {
    auto cfg = create_base_config();
    synnax::channel::Channel ch;
    ch.key = 12345;
    cfg["channels"].push_back(create_channel_config("coil", ch, 0));
    auto p = x::json::Parser(cfg);
    auto task_cfg = std::make_unique<ReadTaskConfig>(client, p);
    ASSERT_OCCURRED_AS(p.error(), x::errors::VALIDATION);
}

/// @brief it should return validation error for invalid channel type.
TEST_F(ModbusReadTest, testInvalidChannelType) {
    auto cfg = create_base_config();

    auto ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("test"),
        x::telem::UINT8_T,
        true
    ));
    cfg["channels"].push_back(
        {{"type", "invalid_type"},
         {"disabled", false},
         {"channel", ch.key},
         {"address", 0}}
    );

    auto p = x::json::Parser(cfg);
    auto task_cfg = std::make_unique<ReadTaskConfig>(client, p);
    ASSERT_OCCURRED_AS(p.error(), x::errors::VALIDATION);
}

/// @brief it should read a whole register as uint16 when the config omits the data
/// type.
TEST(ModbusChannels, testMissingDataTypeDefaultsToUint16) {
    auto parser = x::json::Parser(
        x::json::json{
            {"type", "input_register"},
            {"key", "chan-1"},
            {"address", 3},
            {"channel", 42}
        }
    );
    const auto cfg = ::synnax::modbus::InputRegisterReadChannel::parse(parser);
    ASSERT_NIL(parser.error());
    const channel::InputRegister ch(cfg, device::ConnectionConfig{});
    EXPECT_EQ(ch.value_type, x::telem::UINT16_T);
}

/// @brief it should take the device's byte and word order when the channel has no
/// override.
TEST(ModbusChannels, RegisterInheritsDeviceOrder) {
    auto parser = x::json::Parser(
        x::json::json{{"type", "holding_register"}, {"data_type", "float32"}}
    );
    const auto cfg = ::synnax::modbus::HoldingRegisterReadChannel::parse(parser);
    ASSERT_NIL(parser.error());
    const channel::InputRegister ch(cfg, device::ConnectionConfig("h", 1, true, true));
    EXPECT_TRUE(ch.bytes_swapped);
    EXPECT_TRUE(ch.words_swapped);
}

/// @brief it should let a channel's override win over the device's order.
TEST(ModbusChannels, RegisterOverrideBeatsDevice) {
    auto parser = x::json::Parser(
        x::json::json{
            {"type", "input_register"},
            {"data_type", "float32"},
            {"bytes_swapped", false},
            {"words_swapped", true}
        }
    );
    const auto cfg = ::synnax::modbus::InputRegisterReadChannel::parse(parser);
    ASSERT_NIL(parser.error());
    const channel::InputRegister ch(cfg, device::ConnectionConfig("h", 1, true, false));
    EXPECT_FALSE(ch.bytes_swapped);
    EXPECT_TRUE(ch.words_swapped);
}

/// @brief it should parse configuration with multiple channel types.
TEST_F(ModbusReadTest, testMultiChannelConfig) {
    auto cfg = create_base_config();

    // Create channels for different types
    auto coil_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("coil"),
        x::telem::UINT8_T,
        true
    ));
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
    cfg["channels"].push_back(create_channel_config("coil", coil_ch, 0));
    cfg["channels"].push_back(create_channel_config("discrete_input", discrete_ch, 1));
    cfg["channels"].push_back(create_channel_config("holding_register", holding_ch, 2));
    cfg["channels"].push_back(create_channel_config("input_register", input_ch, 3));

    auto p = x::json::Parser(cfg);
    auto task_cfg = std::make_unique<ReadTaskConfig>(client, p);
    ASSERT_NIL(p.error());
}

/// @brief it should configure when a disabled channel has no Synnax channel bound
/// to it.
TEST_F(ModbusReadTest, testUnboundDisabledChannel) {
    auto cfg = create_base_config();
    const auto ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("coil"),
        x::telem::UINT8_T,
        true
    ));
    cfg["channels"].push_back(create_channel_config("coil", ch, 0));
    synnax::channel::Channel unbound;
    cfg["channels"].push_back(create_channel_config("coil", unbound, 1, true));

    auto p = x::json::Parser(cfg);
    const auto task_cfg = std::make_unique<ReadTaskConfig>(client, p);
    ASSERT_NIL(p.error());
    EXPECT_EQ(task_cfg->data_channel_count, 1);
}

/// @brief it should read coil values from Modbus device.
TEST(ReadTask, testBasicReadTask) {
    testutil::SlaveConfig slave_cfg;
    // Set up coil at address 0 with value 1 and ensure we have at least 1 coil mapped
    slave_cfg.coils[0] = 1;
    slave_cfg.coils[1] = 0; // Add an extra coil to ensure proper mapping size
    slave_cfg.host = "127.0.0.1";
    slave_cfg.port = 1502;

    auto slave = testutil::Slave(slave_cfg);
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });

    auto index_channel = synnax::channel::Channel{
        .name = make_unique_channel_name("time_channel"),
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
        .index = 0
    };

    auto data_channel = synnax::channel::Channel{
        .name = make_unique_channel_name("data_channel"),
        .data_type = x::telem::UINT8_T,
        .is_index = false,
        .index = index_channel.key
    };

    auto client = std::make_shared<synnax::Synnax>(new_test_client());

    ASSERT_NIL(client->channels.create(index_channel));
    data_channel.index = index_channel.key;
    ASSERT_NIL(client->channels.create(data_channel));

    auto rack = ASSERT_NIL_P(client->racks.create("cat"));

    auto conn_cfg = device::ConnectionConfig{"127.0.0.1", 1502};
    x::json::json properties{{"connection", conn_cfg.to_json()}};
    synnax::device::Device dev{
        .key = "my_modbus_lover",
        .rack = rack.key,
        .location = "dev1",
        .make = MAKE,
        .model = "Modbus Device",
        .name = "my_mobdus_lover",
        .properties = properties,
    };

    ASSERT_NIL(client->devices.create(dev));

    auto tsk = synnax::task::Task{
        .rack = rack.key,
        .name = "my_task",
        .type = "modbus_read",
    };

    x::json::json j{
        {"data_saving_disabled", true},
        {"sample_rate", 25},
        {"stream_rate", 25},
        {"device", dev.key},
        {"channels",
         x::json::json::array(
             {{{"type", "coil"},
               {"disabled", false},
               {"channel", data_channel.key},
               {"address", 0}}}
         )}
    };
    auto p = x::json::Parser(j);
    auto cfg = std::make_unique<ReadTaskConfig>(client, p);
    ASSERT_NIL(p.error());

    auto ctx = std::make_shared<driver::task::MockContext>(client);
    auto factory = std::make_shared<driver::pipeline::testutil::WriterFactory>();

    auto devs = std::make_shared<device::Manager>();

    auto task = common::ReadTask(
        tsk,
        ctx,
        x::breaker::default_config(tsk.name),
        std::make_unique<ReadTaskSource>(devs, std::move(*cfg)),
        factory
    );

    task.start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    const auto first_state = ctx->statuses[0];
    EXPECT_EQ(first_state.key, synnax::task::status_key(tsk));
    EXPECT_EQ(first_state.details.cmd, "start_cmd");
    EXPECT_EQ(first_state.variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(first_state.details.task, tsk.key);
    EXPECT_EQ(first_state.message, "Started task");
    ASSERT_EVENTUALLY_GE(factory->writer_opens.load(std::memory_order_acquire), 1);
    task.stop("stop_cmd", true);
    ASSERT_EQ(ctx->statuses.size(), 2);
    const auto second_state = ctx->statuses[1];
    EXPECT_EQ(second_state.key, synnax::task::status_key(tsk));
    EXPECT_EQ(second_state.details.cmd, "stop_cmd");
    EXPECT_EQ(second_state.variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(second_state.details.task, tsk.key);
    EXPECT_EQ(second_state.message, "Stopped task");

    ASSERT_GE(factory->writes->size(), 1);
    const auto &fr = factory->writes->at(0);
    ASSERT_EQ(fr.size(), 2);
    ASSERT_EQ(fr.length(), 1);
    ASSERT_EQ(fr.contains(data_channel.key), true);
    ASSERT_EQ(fr.contains(index_channel.key), true);
    ASSERT_EQ(fr.at<uint8_t>(data_channel.key, 0), 1);
    ASSERT_GE(fr.at<uint64_t>(index_channel.key, 0), 0);
}

/// @brief it should reconnect to a restarted server on the next start.
TEST(ReadTask, testReconnectAfterServerRestart) {
    testutil::SlaveConfig slave_cfg;
    slave_cfg.coils[0] = 1;
    slave_cfg.coils[1] = 0;
    slave_cfg.host = "127.0.0.1";
    slave_cfg.port = 1502;

    auto slave = std::make_unique<testutil::Slave>(slave_cfg);
    ASSERT_NIL(slave->start());

    auto index_channel = synnax::channel::Channel{
        .name = make_unique_channel_name("time_channel"),
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true,
        .index = 0
    };
    auto data_channel = synnax::channel::Channel{
        .name = make_unique_channel_name("data_channel"),
        .data_type = x::telem::UINT8_T,
        .is_index = false,
        .index = index_channel.key
    };

    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    ASSERT_NIL(client->channels.create(index_channel));
    data_channel.index = index_channel.key;
    ASSERT_NIL(client->channels.create(data_channel));

    auto rack = ASSERT_NIL_P(client->racks.create("reconnect_rack"));

    auto conn_cfg = device::ConnectionConfig{"127.0.0.1", 1502};
    x::json::json properties{{"connection", conn_cfg.to_json()}};
    synnax::device::Device dev{
        .key = "modbus_reconnect_dev",
        .rack = rack.key,
        .location = "dev1",
        .make = MAKE,
        .model = "Modbus Device",
        .name = "modbus_reconnect_dev",
        .properties = properties,
    };
    ASSERT_NIL(client->devices.create(dev));

    auto tsk = synnax::task::Task{
        .rack = rack.key,
        .name = "reconnect_task",
        .type = "modbus_read",
    };

    x::json::json j{
        {"data_saving_disabled", true},
        {"sample_rate", 25},
        {"stream_rate", 25},
        {"device", dev.key},
        {"channels",
         x::json::json::array(
             {{{"type", "coil"},
               {"disabled", false},
               {"channel", data_channel.key},
               {"address", 0}}}
         )}
    };
    auto p = x::json::Parser(j);
    auto cfg = std::make_unique<ReadTaskConfig>(client, p);
    ASSERT_NIL(p.error());

    auto ctx = std::make_shared<driver::task::MockContext>(client);
    auto factory = std::make_shared<driver::pipeline::testutil::WriterFactory>();
    auto devs = std::make_shared<device::Manager>();

    auto task = common::ReadTask(
        tsk,
        ctx,
        x::breaker::default_config(tsk.name),
        std::make_unique<ReadTaskSource>(devs, std::move(*cfg)),
        factory
    );

    task.start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    EXPECT_EQ(ctx->statuses[0].variant, synnax::status::VARIANT_SUCCESS);
    ASSERT_EVENTUALLY_GE(factory->writer_opens.load(std::memory_order_acquire), 1);
    task.stop("stop_cmd", true);
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 2);

    slave->stop();
    slave = std::make_unique<testutil::Slave>(slave_cfg);
    ASSERT_NIL(slave->start());
    x::defer::defer stop_slave([&slave] { slave->stop(); });

    task.start("start_cmd_2");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 3);
    EXPECT_EQ(ctx->statuses[2].variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(ctx->statuses[2].message, "Started task");
    ASSERT_EVENTUALLY_GE(factory->writer_opens.load(std::memory_order_acquire), 2);
    task.stop("stop_cmd_2", true);
}

/// @brief it should read discrete input values from Modbus device.
TEST_F(ModbusReadTest, testDiscreteInputRead) {
    // Set up mock slave with discrete input values
    testutil::SlaveConfig slave_cfg;
    slave_cfg.discrete_inputs[1] = 1; // Set discrete input 1 to HIGH
    slave_cfg.discrete_inputs[2] = 0; // Set discrete input 2 to LOW
    slave_cfg.host = "127.0.0.1";
    slave_cfg.port = 1502;

    auto slave = testutil::Slave(slave_cfg);
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
    auto task_cfg = std::make_unique<ReadTaskConfig>(client, p);
    ASSERT_NIL(p.error());

    auto devs = std::make_shared<device::Manager>();
    auto task = common::ReadTask(
        synnax::task::Task{
            .rack = rack.key,
            .name = "discrete_test",
            .type = "modbus_read",
        },
        ctx,
        x::breaker::default_config("discrete_test"),
        std::make_unique<ReadTaskSource>(devs, std::move(*task_cfg)),
        mock_factory
    );

    task.start("start_cmd");
    ASSERT_EVENTUALLY_GE(mock_factory->writes->size(), 1);
    task.stop("stop_cmd", true);

    const auto &fr = mock_factory->writes->at(0);
    ASSERT_EQ(fr.size(), 2); // Data channel + index channel
    ASSERT_EQ(fr.length(), 1);
    ASSERT_EQ(fr.at<uint8_t>(data_channel.key, 0), 1);
}

/// @brief it should read holding register values from Modbus device.
TEST_F(ModbusReadTest, testHoldingRegisterRead) {
    // Set up mock slave with holding register values
    testutil::SlaveConfig slave_cfg;
    slave_cfg.holding_registers[0] = 12345; // Set holding register 0
    slave_cfg.host = "127.0.0.1";
    slave_cfg.port = 1502;

    auto slave = testutil::Slave(slave_cfg);
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
        create_channel_config("holding_register", data_channel, 0)
    );

    auto p = x::json::Parser(cfg);
    auto task_cfg = std::make_unique<ReadTaskConfig>(client, p);
    ASSERT_NIL(p.error());

    auto devs = std::make_shared<device::Manager>();
    auto task = common::ReadTask(
        synnax::task::Task{
            .rack = rack.key,
            .name = "holding_test",
            .type = "modbus_read",
        },
        ctx,
        x::breaker::default_config("holding_test"),
        std::make_unique<ReadTaskSource>(devs, std::move(*task_cfg)),
        mock_factory
    );

    task.start("start_cmd");
    ASSERT_EVENTUALLY_GE(mock_factory->writes->size(), 1);
    task.stop("stop_cmd", true);

    const auto &fr = mock_factory->writes->at(0);
    ASSERT_EQ(fr.size(), 2);
    ASSERT_EQ(fr.length(), 1);
    ASSERT_EQ(fr.at<uint16_t>(data_channel.key, 0), 12345);
}

/// @brief reads one float32 holding register at address 0 from a device whose
/// connection swaps words, and checks the task wrote pi.
void expect_pi_from_word_swapped_device(
    const std::shared_ptr<synnax::Synnax> &client,
    const std::shared_ptr<task::MockContext> &ctx,
    const synnax::rack::Rack &rack,
    const synnax::channel::Channel &index,
    const uint16_t first_register,
    const uint16_t second_register,
    const std::optional<bool> words_swapped
) {
    testutil::SlaveConfig slave_cfg;
    slave_cfg.holding_registers[0] = first_register;
    slave_cfg.holding_registers[1] = second_register;
    slave_cfg.host = "127.0.0.1";
    slave_cfg.port = 1502;
    auto slave = testutil::Slave(slave_cfg);
    ASSERT_NIL(slave.start());
    std::this_thread::sleep_for(std::chrono::milliseconds(100));
    x::defer::defer stop_slave([&slave] { slave.stop(); });

    auto dev = synnax::device::Device{
        .key = "modbus_word_swapped_device",
        .rack = rack.key,
        .location = "dev1",
        .make = "modbus",
        .model = "Modbus Device",
        .name = "modbus_word_swapped_device",
        .properties = x::json::json{
            {"connection",
             device::ConnectionConfig{"127.0.0.1", 1502, false, true}.to_json()}
        }
    };
    ASSERT_NIL(client->devices.create(dev));
    auto data_channel = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("float_register"),
        x::telem::FLOAT32_T,
        index.key,
        false
    ));
    x::json::json ch = {
        {"type", "holding_register"},
        {"channel", data_channel.key},
        {"address", 0},
        {"data_type", "float32"}
    };
    if (words_swapped.has_value()) ch["words_swapped"] = *words_swapped;
    x::json::json cfg = {
        {"data_saving_disabled", true},
        {"sample_rate", 25},
        {"stream_rate", 25},
        {"device", dev.key},
        {"channels", x::json::json::array({ch})}
    };
    auto p = x::json::Parser(cfg);
    auto task_cfg = std::make_unique<ReadTaskConfig>(client, p);
    ASSERT_NIL(p.error());

    auto factory = std::make_shared<pipeline::testutil::WriterFactory>();
    auto task = common::ReadTask(
        synnax::task::Task{.rack = rack.key, .name = "swap", .type = "modbus_read"},
        ctx,
        x::breaker::default_config("swap"),
        std::make_unique<ReadTaskSource>(
            std::make_shared<device::Manager>(),
            std::move(*task_cfg)
        ),
        factory
    );
    task.start("start_cmd");
    ASSERT_EVENTUALLY_GE(factory->writes->size(), 1);
    task.stop("stop_cmd", true);
    ASSERT_FLOAT_EQ(factory->writes->at(0).at<float>(data_channel.key, 0), 3.14159274F);
}

/// @brief it should decode a channel with no override using the device's word swap.
TEST_F(ModbusReadTest, ChannelInheritsDeviceWordSwap) {
    // 0x40490FDB is pi as a float32. The device swaps words, so the high word comes
    // first.
    expect_pi_from_word_swapped_device(
        client,
        ctx,
        rack,
        index_channel,
        0x4049,
        0x0FDB,
        std::nullopt
    );
}

/// @brief it should decode a channel with an override using the channel's word order,
/// not the device's.
TEST_F(ModbusReadTest, ChannelOverridesDeviceWordSwap) {
    expect_pi_from_word_swapped_device(
        client,
        ctx,
        rack,
        index_channel,
        0x0FDB,
        0x4049,
        false
    );
}

/// @brief it should read multiple channel types simultaneously.
TEST_F(ModbusReadTest, testMultiChannelRead) {
    // Set up mock slave with various register values
    testutil::SlaveConfig slave_cfg;
    slave_cfg.coils[0] = 1;
    slave_cfg.discrete_inputs[1] = 1;
    slave_cfg.holding_registers[2] = 12345;
    slave_cfg.input_registers[3] = 54321;
    slave_cfg.host = "127.0.0.1";
    slave_cfg.port = 1502;

    auto slave = testutil::Slave(slave_cfg);
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });

    // Create channels for different types
    auto coil_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("coil"),
        x::telem::UINT8_T,
        true
    ));
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
    cfg["channels"].push_back(create_channel_config("coil", coil_ch, 0));
    cfg["channels"].push_back(create_channel_config("discrete_input", discrete_ch, 1));
    cfg["channels"].push_back(create_channel_config("holding_register", holding_ch, 2));
    cfg["channels"].push_back(create_channel_config("input_register", input_ch, 3));

    auto p = x::json::Parser(cfg);
    auto task_cfg = std::make_unique<ReadTaskConfig>(client, p);
    ASSERT_NIL(p.error());

    auto devs = std::make_shared<device::Manager>();
    auto task = common::ReadTask(
        synnax::task::Task{
            .rack = rack.key,
            .name = "multi_test",
            .type = "modbus_read",
        },
        ctx,
        x::breaker::default_config("multi_test"),
        std::make_unique<ReadTaskSource>(devs, std::move(*task_cfg)),
        mock_factory
    );

    task.start("start_cmd");
    ASSERT_EVENTUALLY_GE(mock_factory->writes->size(), 1);
    task.stop("stop_cmd", true);

    const auto &fr = mock_factory->writes->at(0);
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
    cfg["data_saving_disabled"] = false;

    auto coil_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("coil"),
        x::telem::UINT8_T,
        index_channel.key
    ));
    cfg["channels"].push_back(create_channel_config("coil", coil_ch, 0));

    auto p = x::json::Parser(cfg);
    auto task_cfg = std::make_unique<ReadTaskConfig>(client, p);
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
    testutil::SlaveConfig slave_cfg;
    slave_cfg.input_registers[0] = static_cast<uint8_t>(100);
    slave_cfg.input_registers[1] = static_cast<uint8_t>(150);
    slave_cfg.input_registers[2] = static_cast<uint8_t>(200);
    slave_cfg.host = "127.0.0.1";
    slave_cfg.port = 1502;

    auto slave = testutil::Slave(slave_cfg);
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
    cfg["channels"].push_back(create_channel_config("input_register", input0, 0));
    cfg["channels"].push_back(create_channel_config("input_register", input1, 1));
    cfg["channels"].push_back(create_channel_config("input_register", input2, 2));

    auto p = x::json::Parser(cfg);
    auto task_cfg = std::make_unique<ReadTaskConfig>(client, p);
    ASSERT_NIL(p.error());

    auto devs = std::make_shared<device::Manager>();
    auto task = common::ReadTask(
        synnax::task::Task{
            .rack = rack.key,
            .name = "uint8_test",
            .type = "modbus_read",
        },
        ctx,
        x::breaker::default_config("uint8_test"),
        std::make_unique<ReadTaskSource>(devs, std::move(*task_cfg)),
        mock_factory
    );

    task.start("start_cmd");
    ASSERT_EVENTUALLY_GE(mock_factory->writes->size(), 1);
    task.stop("stop_cmd", true);

    const auto &fr = mock_factory->writes->at(0);
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
    testutil::SlaveConfig slave_cfg;
    slave_cfg.holding_registers[0] = static_cast<uint8_t>(50);
    slave_cfg.holding_registers[1] = static_cast<uint8_t>(75);
    slave_cfg.holding_registers[2] = static_cast<uint8_t>(125);
    slave_cfg.host = "127.0.0.1";
    slave_cfg.port = 1502;

    auto slave = testutil::Slave(slave_cfg);
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
    cfg["channels"].push_back(create_channel_config("holding_register", holding0, 0));
    cfg["channels"].push_back(create_channel_config("holding_register", holding1, 1));
    cfg["channels"].push_back(create_channel_config("holding_register", holding2, 2));

    auto p = x::json::Parser(cfg);
    auto task_cfg = std::make_unique<ReadTaskConfig>(client, p);
    ASSERT_NIL(p.error());

    auto devs = std::make_shared<device::Manager>();
    auto task = common::ReadTask(
        synnax::task::Task{
            .rack = rack.key,
            .name = "uint8_holding_test",
            .type = "modbus_read",
        },
        ctx,
        x::breaker::default_config("uint8_holding_test"),
        std::make_unique<ReadTaskSource>(devs, std::move(*task_cfg)),
        mock_factory
    );

    task.start("start_cmd");
    ASSERT_EVENTUALLY_GE(mock_factory->writes->size(), 1);
    task.stop("stop_cmd", true);

    const auto &fr = mock_factory->writes->at(0);
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
    testutil::SlaveConfig slave_cfg;
    slave_cfg.input_registers[0] = 42;
    slave_cfg.host = "127.0.0.1";
    slave_cfg.port = 1502;

    auto slave = testutil::Slave(slave_cfg);
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
    x::json::json config{
        {"data_saving_disabled", true},
        {"sample_rate", 25},
        {"stream_rate", 25},
        {"device", device.key},
        {"auto_start", true}, // Enable auto-start
        {"channels",
         x::json::json::array(
             {{{"type", "input_register"},
               {"disabled", false},
               {"channel", data_channel.key},
               {"address", 0},
               {"data_type", "uint8"}}}
         )}
    };

    task = synnax::task::Task{
        .rack = rack.key,
        .name = "test_task",
        .type = "modbus_read",
        .config = config
    };

    // Configure task through factory
    auto factory = Factory();
    auto [configured_task, ok] = factory.configure_task(ctx, task, "");

    ASSERT_TRUE(ok);
    ASSERT_NE(configured_task, nullptr);

    // Task should have auto-started - check that a start status was sent
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    bool found_start = false;
    for (const auto &s: ctx->statuses) {
        if (s.details.running && s.variant == synnax::status::VARIANT_SUCCESS) {
            found_start = true;
            break;
        }
    }
    ASSERT_TRUE(found_start);

    // Stop the task to clean up
    synnax::task::Command stop_cmd{.task = task.key, .type = "stop"};
    configured_task->exec(stop_cmd);
}

/// Test that auto_start=false does NOT start the task automatically
TEST_F(ModbusReadTest, testAutoStartFalse) {
    // Set up mock slave
    testutil::SlaveConfig slave_cfg;
    slave_cfg.input_registers[0] = 99;
    slave_cfg.host = "127.0.0.1";
    slave_cfg.port = 1502;

    auto slave = testutil::Slave(slave_cfg);
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
    x::json::json config{
        {"data_saving_disabled", true},
        {"sample_rate", 25},
        {"stream_rate", 25},
        {"device", device.key},
        {"auto_start", false}, // Disable auto-start
        {"channels",
         x::json::json::array(
             {{{"type", "input_register"},
               {"disabled", false},
               {"channel", data_channel.key},
               {"address", 0},
               {"data_type", "uint8"}}}
         )}
    };

    task = synnax::task::Task{
        .rack = rack.key,
        .name = "test_task_no_auto",
        .type = "modbus_read",
        .config = config
    };

    // Configure task through factory
    auto factory = Factory();
    auto [configured_task, ok] = factory.configure_task(ctx, task, "cmd1");

    ASSERT_TRUE(ok);
    ASSERT_NE(configured_task, nullptr);

    // Task should NOT have auto-started. A successful configure writes no status, and
    // an auto-start would have written its start status before returning.
    ASSERT_TRUE(ctx->statuses.empty());

    // Manually start the task
    synnax::task::Command start_cmd{.task = task.key, .type = "start"};
    configured_task->exec(start_cmd);

    // Now task should be running
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    bool found_start = false;
    for (const auto &s: ctx->statuses) {
        if (s.details.running && s.variant == synnax::status::VARIANT_SUCCESS) {
            found_start = true;
            break;
        }
    }
    ASSERT_TRUE(found_start);

    // Stop the task to clean up
    synnax::task::Command stop_cmd{.task = task.key, .type = "stop"};
    configured_task->exec(stop_cmd);
}

/// @brief it should hold each configured sample rate over time.
TEST_F(ModbusReadTest, testHoldsSampleRate) {
    testutil::SlaveConfig slave_cfg;
    slave_cfg.input_registers[0] = 42;
    slave_cfg.host = "127.0.0.1";
    slave_cfg.port = 1502;
    auto slave = testutil::Slave(slave_cfg);
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });

    const auto data_channel = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("rate_input"),
        x::telem::UINT16_T,
        index_channel.key,
        false
    ));

    for (const int rate_hz: {25, 50, 100}) {
        auto cfg = create_base_config();
        cfg["sample_rate"] = rate_hz;
        cfg["stream_rate"] = rate_hz;
        cfg["channels"].push_back(
            create_channel_config("input_register", data_channel, 0)
        );
        auto p = x::json::Parser(cfg);
        auto task_cfg = std::make_unique<ReadTaskConfig>(client, p);
        ASSERT_NIL(p.error());

        auto factory = std::make_shared<pipeline::testutil::WriterFactory>();
        auto devs = std::make_shared<device::Manager>();
        auto task = common::ReadTask(
            synnax::task::Task{
                .rack = rack.key,
                .name = "rate_test",
                .type = "modbus_read",
            },
            ctx,
            x::breaker::default_config("rate_test"),
            std::make_unique<ReadTaskSource>(devs, std::move(*task_cfg)),
            factory
        );

        task.start("start_cmd");
        ASSERT_EVENTUALLY_GE(factory->writes->size(), 1);
        std::this_thread::sleep_for(RATE_SPAN.chrono());
        task.stop("stop_cmd", true);

        expect_rate(
            pipeline::testutil::measured_rate(*factory->writes, index_channel.key),
            rate_hz
        );
    }
}

/// @brief it should reject a Synnax channel that two channels of the task read into.
TEST_F(ModbusReadTest, testDuplicateChannelRejected) {
    auto ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("dup"),
        x::telem::UINT8_T,
        true
    ));
    auto cfg = create_base_config();
    cfg["channels"].push_back(create_channel_config("coil", ch, 0));
    cfg["channels"].push_back(create_channel_config("coil", ch, 1));
    auto p = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, p);
    ASSERT_OCCURRED_AS(p.error(), x::errors::VALIDATION);
    const auto errors = p.error_json()["errors"];
    ASSERT_EQ(errors.size(), 1);
    EXPECT_EQ(errors[0]["path"], "channels.1.channel");
    EXPECT_EQ(errors[0]["message"], "another channel in the task reads into it");
}

/// @brief it should reject a register data type the Driver cannot decode.
TEST_F(ModbusReadTest, testUnsupportedDataTypeRejected) {
    auto ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("ts"),
        x::telem::FLOAT32_T,
        true
    ));
    auto cfg = create_base_config();
    cfg["channels"].push_back(create_channel_config("holding_register", ch, 0));
    cfg["channels"][0]["data_type"] = "timestamp";
    auto p = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, p);
    ASSERT_OCCURRED_AS(p.error(), x::errors::VALIDATION);
    const auto errors = p.error_json()["errors"];
    ASSERT_EQ(errors.size(), 1);
    EXPECT_EQ(errors[0]["path"], "channels.0.data_type");
    EXPECT_EQ(errors[0]["message"], "unsupported data type: timestamp");
}

/// @brief it should reject a string with no length, a string read into a channel that
/// is not a string, and a bit past 15.
TEST_F(ModbusReadTest, testStringAndBitRejected) {
    auto str = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("str"),
        x::telem::STRING_T,
        true
    ));
    auto num = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("num"),
        x::telem::UINT16_T,
        true
    ));
    auto bit = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("bit"),
        x::telem::UINT8_T,
        true
    ));
    auto cfg = create_base_config();
    cfg["channels"].push_back(
        {{"type", "input_register"},
         {"channel", str.key},
         {"address", 0},
         {"data_type", "string"}}
    );
    cfg["channels"].push_back(
        {{"type", "input_register"},
         {"channel", num.key},
         {"address", 10},
         {"data_type", "string"},
         {"string_length", 4}}
    );
    cfg["channels"].push_back(
        {{"type", "holding_register_bit"},
         {"channel", bit.key},
         {"address", 0},
         {"bit", 16}}
    );
    auto p = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, p);
    ASSERT_OCCURRED_AS(p.error(), x::errors::VALIDATION);
    const auto errors = p.error_json()["errors"];
    ASSERT_EQ(errors.size(), 3);
    EXPECT_EQ(errors[0]["path"], "channels.0.string_length");
    EXPECT_EQ(errors[0]["message"], "must be from 1 to 250");
    EXPECT_EQ(errors[1]["path"], "channels.2.bit");
    EXPECT_EQ(errors[1]["message"], "must be from 0 to 15");
    EXPECT_EQ(errors[2]["path"], "channels.1.channel");
    EXPECT_EQ(errors[2]["message"], "a string must read into a string channel");
}

/// @brief it should read a string and the bits of a register into one frame.
TEST_F(ModbusReadTest, testStringAndBitRead) {
    testutil::SlaveConfig slave_cfg;
    slave_cfg.port = 1502;
    slave_cfg.holding_registers[0] = static_cast<uint16_t>('O' << 8 | 'K');
    slave_cfg.holding_registers[1] = static_cast<uint16_t>(0b10);
    auto slave = testutil::Slave(slave_cfg);
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });

    auto str = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("str"),
        x::telem::STRING_T,
        index_channel.key,
        false
    ));
    auto bit = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("bit"),
        x::telem::UINT8_T,
        index_channel.key,
        false
    ));
    auto cfg = create_base_config();
    cfg["channels"].push_back(
        {{"type", "holding_register"},
         {"channel", str.key},
         {"address", 0},
         {"data_type", "string"},
         {"string_length", 2}}
    );
    cfg["channels"].push_back(
        {{"type", "holding_register_bit"},
         {"channel", bit.key},
         {"address", 1},
         {"bit", 1}}
    );
    auto p = x::json::Parser(cfg);
    auto task_cfg = std::make_unique<ReadTaskConfig>(client, p);
    ASSERT_NIL(p.error());

    auto devs = std::make_shared<device::Manager>();
    auto task = common::ReadTask(
        synnax::task::Task{
            .rack = rack.key,
            .name = "string_bit_test",
            .type = "modbus_read",
        },
        ctx,
        x::breaker::default_config("string_bit_test"),
        std::make_unique<ReadTaskSource>(devs, std::move(*task_cfg)),
        mock_factory
    );
    task.start("start_cmd");
    ASSERT_EVENTUALLY_GE(mock_factory->writes->size(), 2);
    task.stop("stop_cmd", true);

    for (std::size_t i = 0; i < 2; i++) {
        const auto &fr = mock_factory->writes->at(i);
        ASSERT_EQ(fr.length(), 1);
        EXPECT_EQ(fr.at<uint8_t>(bit.key, 0), 1);
        EXPECT_EQ(std::get<std::string>(fr.at(str.key, 0)), "OK");
    }
}

/// @brief it should reject a register value wider than one request of the device.
TEST_F(ModbusReadTest, testValueWiderThanMaxRegistersRejected) {
    device.properties["requests"] = {{"max_registers", 1}};
    ASSERT_NIL(client->devices.create(device));
    auto ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("wide"),
        x::telem::FLOAT32_T,
        true
    ));
    auto cfg = create_base_config();
    cfg["channels"].push_back(create_channel_config("holding_register", ch, 0));
    cfg["channels"][0]["data_type"] = "float32";
    auto p = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, p);
    ASSERT_OCCURRED_AS(p.error(), x::errors::VALIDATION);
    const auto errors = p.error_json()["errors"];
    ASSERT_EQ(errors.size(), 1);
    EXPECT_EQ(errors[0]["path"], "channels.0.data_type");
    EXPECT_EQ(
        errors[0]["message"],
        "needs 2 registers, but the device reads at most 1 per request"
    );
}

/// @brief it should reject a string wider than one request of the device.
TEST_F(ModbusReadTest, testStringWiderThanMaxRegistersRejected) {
    device.properties["requests"] = {{"max_registers", 2}};
    ASSERT_NIL(client->devices.create(device));
    auto ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("wide_str"),
        x::telem::STRING_T,
        true
    ));
    auto cfg = create_base_config();
    cfg["channels"].push_back(
        {{"type", "holding_register"},
         {"channel", ch.key},
         {"address", 0},
         {"data_type", "string"},
         {"string_length", 5}}
    );
    auto p = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, p);
    ASSERT_OCCURRED_AS(p.error(), x::errors::VALIDATION);
    const auto errors = p.error_json()["errors"];
    ASSERT_EQ(errors.size(), 1);
    EXPECT_EQ(errors[0]["path"], "channels.0.string_length");
    EXPECT_EQ(
        errors[0]["message"],
        "needs 3 registers, but the device reads at most 2 per request"
    );
}

/// @brief it should reject a Synnax channel that cannot hold a number.
TEST_F(ModbusReadTest, testStringChannelRejected) {
    auto ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("str"),
        x::telem::STRING_T,
        true
    ));
    auto cfg = create_base_config();
    cfg["channels"].push_back(create_channel_config("coil", ch, 0));
    auto p = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, p);
    ASSERT_OCCURRED_AS(p.error(), x::errors::VALIDATION);
    const auto errors = p.error_json()["errors"];
    ASSERT_EQ(errors.size(), 1);
    EXPECT_EQ(errors[0]["path"], "channels.0.channel");
    EXPECT_EQ(errors[0]["message"], "must be a numeric channel, not a string channel");
}

/// @brief it should keep reading after the server drops the connection, without a
/// restart.
TEST_F(ModbusReadTest, testKeepsReadingAfterDroppedConnection) {
    testutil::SlaveConfig slave_cfg;
    slave_cfg.port = 1502;
    slave_cfg.coils[0] = 1;
    testutil::Slave slave(slave_cfg);
    ASSERT_NIL(slave.start());
    x::defer::defer stop_slave([&slave] { slave.stop(); });

    auto data_ch = synnax::channel::Channel{
        .name = make_unique_channel_name("coil"),
        .data_type = x::telem::UINT8_T,
        .index = index_channel.key,
    };
    ASSERT_NIL(client->channels.create(data_ch));
    auto cfg = create_base_config();
    cfg["channels"].push_back(create_channel_config("coil", data_ch, 0));
    auto p = x::json::Parser(cfg);
    auto task_cfg = std::make_unique<ReadTaskConfig>(client, p);
    ASSERT_NIL(p.error());
    task = synnax::task::Task{.rack = rack.key, .name = "drop", .type = "modbus_read"};

    auto read_task = common::ReadTask(
        task,
        ctx,
        x::breaker::Config{
            .name = task.name,
            .base_interval = 10 * x::telem::MILLISECOND,
            .max_retries = x::breaker::RETRY_INFINITELY,
        },
        std::make_unique<ReadTaskSource>(
            std::make_shared<device::Manager>(),
            std::move(*task_cfg)
        ),
        mock_factory
    );
    read_task.start("start_cmd");
    ASSERT_EVENTUALLY_GE(mock_factory->writes->size(), 2);
    slave.drop_connections();
    const auto after_drop = mock_factory->writes->size();
    ASSERT_EVENTUALLY_GE(mock_factory->writes->size(), after_drop + 5);
    read_task.stop("stop_cmd", true);
    for (const auto &status: ctx->statuses)
        EXPECT_NE(status.variant, synnax::status::VARIANT_ERROR) << status.message;
}
}
