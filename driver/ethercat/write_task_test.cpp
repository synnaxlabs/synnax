// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <thread>

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/ethercat/ethercat.h"
#include "driver/ethercat/testutil/master.h"
#include "driver/ethercat/write_task.h"
#include "driver/pipeline/testutil/pipeline.h"
#include "engine/engine.h"

namespace driver::ethercat {
class EtherCATWriteTest : public ::testing::Test {
protected:
    /// @brief the cancel token given to configure_task and exec.
    x::breaker::Breaker cancel;

    std::shared_ptr<synnax::Synnax> client;
    std::shared_ptr<task::MockContext> ctx;
    std::shared_ptr<testutil::Master> mock_master;
    std::shared_ptr<engine::Engine> engine;
    synnax::channel::Channel index_channel;
    synnax::rack::Rack rack;
    synnax::device::Device slave_device;
    const uint32_t SLAVE_SERIAL = 12345;
    const std::string NETWORK_INTERFACE = "eth0";

    /// @brief returns the properties of a slave of the saved model on the bus.
    [[nodiscard]] static slave::Properties
    slave_properties(const uint16_t position, const uint32_t serial) {
        return {
            .position = position,
            .vendor_id = 0x1,
            .product_code = 0x2,
            .serial = serial,
            .name = "Test Module",
            .output_pdos = {
                {.pdo_index = 0x1600,
                 .index = 0x7000,
                 .sub_index = 1,
                 .bit_length = 16,
                 .is_input = false,
                 .name = "control_word",
                 .data_type = x::telem::INT16_T},
                {.pdo_index = 0x1600,
                 .index = 0x7000,
                 .sub_index = 2,
                 .bit_length = 32,
                 .is_input = false,
                 .name = "setpoint",
                 .data_type = x::telem::INT32_T},
            },
        };
    }

    void SetUp() override {
        this->cancel.start();
        client = std::make_shared<synnax::Synnax>(new_test_client());

        index_channel = synnax::channel::Channel{
            .name = make_unique_channel_name("time_channel"),
            .data_type = x::telem::TIMESTAMP_T,
            .is_index = true,
        };
        ASSERT_NIL(client->channels.create(index_channel));

        rack = ASSERT_NIL_P(client->racks.create("test_rack"));

        ctx = std::make_shared<task::MockContext>(client);

        slave_device = create_slave_device(
            SLAVE_SERIAL,
            x::json::json::array(),
            {{{"name", "control_word"},
              {"index", 0x7000},
              {"sub_index", 1},
              {"bit_length", 16},
              {"data_type", "int16"}},
             {{"name", "setpoint"},
              {"index", 0x7000},
              {"sub_index", 2},
              {"bit_length", 32},
              {"data_type", "int32"}}}
        );

        mock_master = std::make_shared<testutil::Master>();
        mock_master->add_slave(slave_properties(1, SLAVE_SERIAL));
        engine = std::make_shared<engine::Engine>(NETWORK_INTERFACE, mock_master);
    }

    [[nodiscard]] synnax::device::Device create_slave_device(
        uint32_t serial,
        const x::json::json &input_pdos,
        const x::json::json &output_pdos
    ) const {
        x::json::json props = {
            {"serial", serial},
            {"vendor_id", 0x1},
            {"product_code", 0x2},
            {"revision", 1},
            {"name", "Test Module"},
            {"network", NETWORK_INTERFACE},
            {"position", 1},
            {"enabled", true},
            {"pdos", {{"inputs", input_pdos}, {"outputs", output_pdos}}}
        };
        synnax::device::Device dev{
            .key = make_unique_channel_name("ecat_slave_" + std::to_string(serial)),
            .rack = rack.key,
            .location = std::to_string(serial),
            .make = "DEWESoft",
            .model = "TestModule",
            .name = "Test Module SN:" + std::to_string(serial),
            .properties = props,
        };
        const auto err = client->devices.create(dev);
        EXPECT_NIL(err);
        return dev;
    }

    x::json::json create_base_config() {
        return {
            {"data_saving_disabled", true},
            {"state_rate", 10.0},
            {"channels", x::json::json::array()}
        };
    }

    x::json::json create_output_channel_config(
        const synnax::channel::Key &command_key,
        uint16_t index,
        uint8_t sub_index,
        const std::string &data_type,
        synnax::channel::Key state_key = 0,
        bool disabled = false
    ) {
        x::json::json cfg = {
            {"type", "entry"},
            {"device", slave_device.key},
            {"index", index},
            {"sub_index", sub_index},
            {"data_type", data_type},
            {"cmd_channel", command_key},
            {"disabled", disabled}
        };
        if (state_key != 0) cfg["state_channel"] = state_key;
        return cfg;
    }

    x::json::json create_unresolved_output_channel_config(
        const synnax::channel::Key &command_key,
        const std::string &pdo_name
    ) {
        return {
            {"type", "unresolved"},
            {"device", slave_device.key},
            {"pdo", pdo_name},
            {"cmd_channel", command_key},
            {"disabled", false}
        };
    }

    synnax::channel::Channel create_command_channel(
        const std::string &name,
        const x::telem::DataType &data_type
    ) {
        return ASSERT_NIL_P(client->channels.create(
            make_unique_channel_name(name),
            data_type,
            index_channel.key,
            false
        ));
    }

    void TearDown() override { this->cancel.stop(); }
};

/// @brief it should pair each state channel with the command channel that names it, not
/// with the channel at the same position.
TEST_F(EtherCATWriteTest, StateChannelPairsWithItsOwnCommandChannel) {
    auto cmd_1 = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("cmd_1"),
        x::telem::INT16_T,
        index_channel.key,
        false
    ));
    auto cmd_2 = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("cmd_2"),
        x::telem::INT32_T,
        index_channel.key,
        false
    ));
    auto state_2 = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("state_2"),
        x::telem::INT32_T,
        index_channel.key,
        false
    ));

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        create_output_channel_config(cmd_1.key, 0x7000, 1, "int16")
    );
    cfg["channels"].push_back(
        create_output_channel_config(cmd_2.key, 0x7000, 2, "int32", state_2.key)
    );

    auto parser = x::json::Parser(cfg);
    WriteTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());
    const auto outs = task_cfg.output_channels();
    ASSERT_EQ(outs.size(), 2);
    EXPECT_EQ(outs[0].command, cmd_1.key);
    EXPECT_FALSE(outs[0].state.has_value());
    EXPECT_EQ(outs[1].command, cmd_2.key);
    ASSERT_TRUE(outs[1].state.has_value());
    EXPECT_EQ(outs[1].state.value().key, state_2.key);
}

/// @brief it should report a second write channel on an entry another channel already
/// uses.
TEST_F(EtherCATWriteTest, ParseConfigRejectsTwoChannelsOnOneEntry) {
    const auto ch1 = create_command_channel("ch1", x::telem::INT16_T);
    const auto ch2 = create_command_channel("ch2", x::telem::INT16_T);

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        create_output_channel_config(ch1.key, 0x7000, 1, "int16")
    );
    cfg["channels"].push_back(
        create_output_channel_config(ch2.key, 0x7000, 1, "int16")
    );

    auto parser = x::json::Parser(cfg);
    WriteTaskConfig task_cfg(client, parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    const auto errors = parser.error_json()["errors"];
    ASSERT_EQ(errors.size(), 1);
    EXPECT_EQ(errors[0]["path"], "channels.1.index");
    EXPECT_EQ(
        errors[0]["message"],
        "another channel already uses entry 0x7000:01 on module Test Module"
    );
}

/// @brief it should report only the entry error of each write channel whose entry it
/// cannot find.
TEST_F(EtherCATWriteTest, ParseConfigReportsEachChannelWithNoEntry) {
    const auto ch1 = create_command_channel("ch1", x::telem::INT16_T);
    const auto ch2 = create_command_channel("ch2", x::telem::INT16_T);

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        create_unresolved_output_channel_config(ch1.key, "Missing")
    );
    cfg["channels"].push_back(
        create_unresolved_output_channel_config(ch2.key, "Missing")
    );

    auto parser = x::json::Parser(cfg);
    WriteTaskConfig task_cfg(client, parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    const auto errors = parser.error_json()["errors"];
    ASSERT_EQ(errors.size(), 2);
    EXPECT_EQ(errors[0]["path"], "channels.0.pdo");
    EXPECT_EQ(errors[1]["path"], "channels.1.pdo");
    EXPECT_EQ(
        errors[1]["message"],
        "'Missing' matches no output entry on module Test Module. Pick an entry."
    );
}

/// @brief it should allow a disabled write channel on an entry another channel uses.
TEST_F(EtherCATWriteTest, ParseConfigAllowsADisabledChannelOnAUsedEntry) {
    const auto ch1 = create_command_channel("ch1", x::telem::INT16_T);
    const auto ch2 = create_command_channel("ch2", x::telem::INT16_T);

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        create_output_channel_config(ch1.key, 0x7000, 1, "int16")
    );
    cfg["channels"].push_back(
        create_output_channel_config(ch2.key, 0x7000, 1, "int16", 0, true)
    );

    auto parser = x::json::Parser(cfg);
    WriteTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(task_cfg.outputs.size(), 1);
}

/// @brief it should ignore a disabled write channel on a slave device that does not
/// exist.
TEST_F(EtherCATWriteTest, ParseConfigIgnoresTheDeviceOfADisabledChannel) {
    const auto enabled = create_command_channel("enabled", x::telem::INT16_T);
    const auto disabled = create_command_channel("disabled", x::telem::INT16_T);

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        create_output_channel_config(enabled.key, 0x7000, 1, "int16")
    );
    auto disabled_cfg = create_output_channel_config(
        disabled.key,
        0x7000,
        1,
        "int16",
        0,
        true
    );
    disabled_cfg["device"] = "nonexistent_slave_key";
    cfg["channels"].push_back(disabled_cfg);

    auto parser = x::json::Parser(cfg);
    WriteTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(task_cfg.outputs.size(), 1);
    EXPECT_EQ(task_cfg.outputs[0]->command_key, enabled.key);
}

/// @brief it should report each write channel on a slave device that does not exist.
TEST_F(EtherCATWriteTest, InvalidSlaveDevice) {
    auto cmd_ch = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("cmd"),
        x::telem::INT16_T,
        this->index_channel.key,
        false
    ));

    x::json::json cfg = {
        {"data_saving_disabled", true},
        {"state_rate", 10.0},
        {"channels",
         {{{"type", "entry"},
           {"device", "nonexistent_slave_key"},
           {"index", 0x7000},
           {"sub_index", 1},
           {"data_type", "int16"},
           {"cmd_channel", cmd_ch.key},
           {"disabled", false}}}}
    };
    auto second = cfg["channels"][0];
    second["sub_index"] = 2;
    cfg["channels"].push_back(second);

    auto parser = x::json::Parser(cfg);
    WriteTaskConfig task_cfg(this->client, parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    const auto errors = parser.error_json()["errors"];
    ASSERT_EQ(errors.size(), 2);
    EXPECT_EQ(errors[0]["path"], "channels.0.device");
    EXPECT_EQ(errors[1]["path"], "channels.1.device");
    EXPECT_EQ(
        errors[1]["message"],
        R"(Found no device where key = "nonexistent_slave_key")"
    );
}

/// @brief it should send no status for a start that a stop ends while it waits for the
/// start of another task, so the stop answers it.
TEST_F(EtherCATWriteTest, StopDuringAStartSendsNoStatusForTheStart) {
    auto manager = std::make_unique<testutil::Manager>();
    manager->configure(NETWORK_INTERFACE, this->mock_master);
    Factory factory(std::move(manager), this->client, this->rack.key);
    const auto configure = [&](const std::string &name,
                               const uint8_t sub_index,
                               const x::telem::DataType &data_type) {
        const auto cmd_ch = create_command_channel(name, data_type);
        auto cfg = this->create_base_config();
        cfg["channels"].push_back(create_output_channel_config(
            cmd_ch.key,
            0x7000,
            sub_index,
            data_type.name()
        ));
        const synnax::task::Task task{
            .rack = this->rack.key,
            .name = name,
            .type = WRITE_TASK_TYPE,
            .config = cfg,
        };
        auto [tsk, ok] = factory.configure_task(this->ctx, task, "", this->cancel);
        EXPECT_TRUE(ok);
        return std::pair{task, std::move(tsk)};
    };
    auto [first, first_task] = configure("first", 1, x::telem::INT16_T);
    auto [second, second_task] = configure("second", 2, x::telem::INT32_T);
    this->mock_master->hold_activate();
    std::thread starter([&] {
        synnax::task::Command start{.task = first.key, .type = "start", .key = "a"};
        first_task->exec(start, this->cancel);
    });
    this->mock_master->wait_until_activating();
    x::breaker::Breaker canceled;
    synnax::task::Command start{.task = second.key, .type = "start", .key = "b"};
    second_task->exec(start, canceled);
    this->mock_master->release_activate();
    starter.join();
    for (const auto &status: this->ctx->snapshot()) {
        if (status.details.cmd == "b") {
            EXPECT_EQ(status.variant, synnax::status::VARIANT_LOADING)
                << status.message;
        }
    }
    synnax::task::Command stop{.task = first.key, .type = "stop"};
    first_task->exec(stop, this->cancel);
}

/// @brief it should start a write task that the Factory configures, with the bus at the
/// execution rate of the task, and send only the steps that the start reached before it
/// answers the start.
TEST_F(EtherCATWriteTest, FactoryStartsAWriteTaskAtItsExecutionRate) {
    const auto cmd_ch = create_command_channel("cmd", x::telem::INT16_T);
    auto cfg = this->create_base_config();
    cfg["execution_rate"] = 250;
    cfg["channels"].push_back(
        create_output_channel_config(cmd_ch.key, 0x7000, 1, "int16")
    );
    auto manager = std::make_unique<testutil::Manager>();
    manager->configure(NETWORK_INTERFACE, this->mock_master);
    Factory factory(std::move(manager), this->client, this->rack.key);
    const synnax::task::Task task{
        .rack = this->rack.key,
        .name = "write",
        .type = WRITE_TASK_TYPE,
        .config = cfg,
    };
    auto [write_task, ok] = factory.configure_task(this->ctx, task, "", this->cancel);
    ASSERT_TRUE(ok);
    ASSERT_NE(write_task, nullptr);
    synnax::task::Command start{.task = task.key, .type = "start", .key = "start_cmd"};
    write_task->exec(start, this->cancel);
    ASSERT_EVENTUALLY_EQ_F(
        [&] {
            const auto sent = this->ctx->snapshot();
            return sent.empty() ? std::string() : sent.back().message;
        },
        std::string("Started task")
    );
    const auto statuses = this->ctx->snapshot();
    std::vector<std::string> messages;
    for (const auto &status: statuses) {
        messages.push_back(status.message);
        EXPECT_EQ(status.details.cmd, "start_cmd");
    }
    // Only the latest step waits to be sent, so the sender can skip the scan.
    const std::vector<std::string> every{
        "Scanning eth0",
        "Starting eth0",
        "Started task"
    };
    const std::vector<std::string> latest{"Starting eth0", "Started task"};
    ASSERT_TRUE(messages == every || messages == latest) << messages.size();
    EXPECT_EQ(statuses[statuses.size() - 2].variant, synnax::status::VARIANT_LOADING);
    EXPECT_EQ(this->mock_master->plan().period, x::telem::MILLISECOND * 4);
    synnax::task::Command stop{.task = task.key, .type = "stop"};
    write_task->exec(stop, this->cancel);
}

/// @brief it should reject a write task whose channels are all disabled.
TEST_F(EtherCATWriteTest, ParseConfigRejectsATaskWithNoEnabledChannel) {
    const auto ch = create_command_channel("cmd", x::telem::INT16_T);
    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        create_output_channel_config(ch.key, 0x7000, 1, "int16", 0, true)
    );
    auto parser = x::json::Parser(cfg);
    WriteTaskConfig task_cfg(this->client, parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    const auto errors = parser.error_json()["errors"];
    ASSERT_EQ(errors.size(), 1);
    EXPECT_EQ(errors[0]["path"], "channels");
    EXPECT_EQ(errors[0]["message"], "task must have at least one enabled channel");
}

/// @brief it should fail a start that discovers the slaves while another task starts
/// the bus, once the start timeout passes.
TEST_F(EtherCATWriteTest, SinkStartBehindAnotherStartFailsAtTheStartTimeout) {
    const auto cmd_ch = create_command_channel("cmd", x::telem::INT16_T);
    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        create_output_channel_config(cmd_ch.key, 0x7000, 1, "int16")
    );
    engine::Config engine_cfg;
    engine_cfg.start_timeout = x::telem::MILLISECOND * 100;
    this->engine = std::make_shared<engine::Engine>(
        NETWORK_INTERFACE,
        this->mock_master,
        engine_cfg
    );
    const auto sink = [&] {
        auto parser = x::json::Parser(cfg);
        WriteTaskConfig task_cfg(this->client, parser);
        EXPECT_NIL(parser.error());
        return std::make_unique<WriteTaskSink>(this->engine, std::move(task_cfg));
    };
    const auto first = sink();
    const auto second = sink();
    this->mock_master->hold_activate();
    std::thread starter([&] { ASSERT_NIL(first->start("first", this->cancel)); });
    this->mock_master->wait_until_activating();
    const auto err = second->start("second", this->cancel);
    this->mock_master->release_activate();
    starter.join();
    ASSERT_OCCURRED_AS(err, errors::ACTIVATION_ERROR);
    EXPECT_EQ(err.data, "Another task is still using eth0.");
    ASSERT_NIL(first->stop());
}

/// @brief it should write the stop value of each channel when the task stops.
TEST_F(EtherCATWriteTest, SinkWritesTheStopValueWhenTheTaskStops) {
    const auto cmd_ch = create_command_channel("cmd", x::telem::INT16_T);
    auto cfg = this->create_base_config();
    auto ch = create_output_channel_config(cmd_ch.key, 0x7000, 1, "int16");
    ch["stop_value"] = -3;
    cfg["channels"].push_back(ch);
    auto parser = x::json::Parser(cfg);
    WriteTaskConfig task_cfg(this->client, parser);
    ASSERT_NIL(parser.error());

    auto sink = WriteTaskSink(this->engine, std::move(task_cfg));
    ASSERT_NIL(sink.start("start_cmd", this->cancel));
    x::telem::Frame frame(cmd_ch.key, x::telem::Series(static_cast<int16_t>(0x5678)));
    ASSERT_NIL(sink.write(frame));
    ASSERT_EVENTUALLY_EQ(
        this->mock_master->get_output<int16_t>(0),
        static_cast<int16_t>(0x5678)
    );
    ASSERT_NIL(sink.stop());
    EXPECT_FALSE(this->engine->running());
    EXPECT_EQ(this->mock_master->get_output<int16_t>(0), -3);
}

/// @brief it should hold a warning when the bus did not apply the startup parameters of
/// a slave that another program sets up.
TEST_F(EtherCATWriteTest, SinkHoldsAWarningForSkippedStartupSettings) {
    this->mock_master->set_settings(
        1,
        {{"mode", {{"shared", true}, {"shared_with", "DewesoftX"}}}}
    );
    engine::Config engine_cfg;
    engine_cfg.slave_configs = [this] {
        return std::pair{
            std::vector{engine::SavedSlave{
                .identity =
                    {
                        .position = 1,
                        .vendor_id = 0x1,
                        .product_code = 0x2,
                        .serial = SLAVE_SERIAL,
                        .name = "Test Module",
                    },
                .config =
                    {.startup_parameters =
                         {
                             {.index = 0x8040,
                              .sub_index = 1,
                              .data_type = x::telem::UINT16_T,
                              .value = 6},
                         }},
            }},
            x::errors::NIL
        };
    };
    this->engine = std::make_shared<engine::Engine>(
        NETWORK_INTERFACE,
        this->mock_master,
        engine_cfg
    );
    const auto cmd_ch = create_command_channel("cmd", x::telem::INT16_T);
    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        create_output_channel_config(cmd_ch.key, 0x7000, 1, "int16")
    );
    auto parser = x::json::Parser(cfg);
    WriteTaskConfig task_cfg(this->client, parser);
    ASSERT_NIL(parser.error());

    auto sink = WriteTaskSink(this->engine, std::move(task_cfg));
    ASSERT_NIL(sink.start("start_cmd", this->cancel));
    EXPECT_EQ(
        sink.held_warning(),
        (common::Warning{
            .message = "Skipped startup settings",
            .description = "Module 1 (Test Module) is set up for DewesoftX, so Synnax "
                           "did not apply its 1 startup setting. Remove it, or switch "
                           "the device to Synnax in Configure.",
        })
    );
    ASSERT_NIL(sink.stop());
}

/// @brief it should hold a warning while its slave is lost, tell when the slave
/// returns, and end the warning at the next command.
TEST_F(EtherCATWriteTest, SinkHoldsAWarningWhileItsSlaveIsLost) {
    const auto cmd_ch = create_command_channel("cmd", x::telem::INT16_T);
    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        create_output_channel_config(cmd_ch.key, 0x7000, 1, "int16")
    );
    auto parser = x::json::Parser(cfg);
    WriteTaskConfig task_cfg(this->client, parser);
    ASSERT_NIL(parser.error());
    auto sink = WriteTaskSink(this->engine, std::move(task_cfg));
    ASSERT_NIL(sink.start("start_cmd", this->cancel));
    EXPECT_TRUE(sink.held_warning().empty());

    this->mock_master->set_slave_lost(1, true);
    ASSERT_EVENTUALLY_EQ_F(
        [&] { return sink.held_warning().description; },
        std::string("Lost contact with module 1 (Test Module).")
    );
    EXPECT_EQ(sink.held_warning().message, "Interrupted outputs");
    this->mock_master->set_slave_lost(1, false);
    ASSERT_EVENTUALLY_EQ_F(
        [&] { return sink.held_warning().description; },
        std::string(
            "Module 1 (Test Module) returned and took the last commanded values."
        )
    );
    x::telem::Frame frame(cmd_ch.key, x::telem::Series(static_cast<int16_t>(1)));
    ASSERT_NIL(sink.write(frame));
    EXPECT_TRUE(sink.held_warning().empty());
    ASSERT_NIL(sink.stop());
}

/// @brief it should write to a slave with its serial number at the position where it is
/// on the bus, not at its saved position.
TEST_F(EtherCATWriteTest, SinkWritesASlaveThatMoved) {
    this->mock_master->replace_slave(1, slave_properties(1, 999));
    this->mock_master->add_slave(slave_properties(2, SLAVE_SERIAL));
    const auto cmd_ch = create_command_channel("cmd", x::telem::INT16_T);
    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        create_output_channel_config(cmd_ch.key, 0x7000, 1, "int16")
    );
    auto parser = x::json::Parser(cfg);
    WriteTaskConfig task_cfg(this->client, parser);
    ASSERT_NIL(parser.error());
    auto sink = WriteTaskSink(this->engine, std::move(task_cfg));
    ASSERT_NIL(sink.start("start_cmd", this->cancel));
    x::telem::Frame frame(cmd_ch.key, x::telem::Series(static_cast<int16_t>(0x5678)));
    ASSERT_NIL(sink.write(frame));
    ASSERT_EVENTUALLY_EQ(
        this->mock_master->get_output<int16_t>(6),
        static_cast<int16_t>(0x5678)
    );
    EXPECT_EQ(this->mock_master->get_output<int16_t>(0), 0);
    ASSERT_NIL(sink.stop());
}

/// @brief it should wake the next state read for a command written before the state
/// loop waits.
TEST_F(EtherCATWriteTest, SinkStateReadWakesForCommandSetBeforeWait) {
    const auto cmd_ch = create_command_channel("cmd", x::telem::INT16_T);
    const auto state_ch = create_command_channel("state", x::telem::INT16_T);

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        create_output_channel_config(cmd_ch.key, 0x7000, 1, "int16", state_ch.key)
    );

    auto parser = x::json::Parser(cfg);
    WriteTaskConfig task_cfg(this->client, parser);
    ASSERT_NIL(parser.error());
    const auto period = task_cfg.state_rate.period().chrono();

    auto sink = WriteTaskSink(this->engine, std::move(task_cfg));
    ASSERT_NIL(sink.start("start_cmd", this->cancel));

    x::telem::Frame frame(cmd_ch.key, x::telem::Series(static_cast<int16_t>(0x5678)));
    ASSERT_NIL(sink.write(frame));

    x::breaker::Breaker breaker;
    x::telem::Frame fr;
    pipeline::Authorities authorities;
    const auto start = std::chrono::steady_clock::now();
    ASSERT_NIL(sink.read(breaker, fr, authorities));
    const auto elapsed = std::chrono::steady_clock::now() - start;
    EXPECT_EQ(fr.at<int16_t>(state_ch.key, 0), static_cast<int16_t>(0x5678));
    EXPECT_LT(elapsed, period / 2);

    ASSERT_NIL(sink.stop());
}

/// @brief it should write only the last sample of a channel in a frame.
TEST_F(EtherCATWriteTest, SinkLastWriteWins) {
    auto cmd_ch = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("cmd"),
        x::telem::INT16_T,
        this->index_channel.key,
        false
    ));

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        create_output_channel_config(cmd_ch.key, 0x7000, 1, "int16")
    );

    auto parser = x::json::Parser(cfg);
    WriteTaskConfig task_cfg(this->client, parser);
    ASSERT_NIL(parser.error());

    auto sink = WriteTaskSink(this->engine, std::move(task_cfg));
    ASSERT_NIL(sink.start("start_cmd", this->cancel));

    x::telem::Series series(x::telem::INT16_T, 3);
    series.write(static_cast<int16_t>(0x1111));
    series.write(static_cast<int16_t>(0x2222));
    series.write(static_cast<int16_t>(0x3333));
    x::telem::Frame frame(cmd_ch.key, std::move(series));

    ASSERT_NIL(sink.write(frame));

    ASSERT_EVENTUALLY_EQ(
        this->mock_master->get_output<int16_t>(0),
        static_cast<int16_t>(0x3333)
    );

    ASSERT_NIL(sink.stop());
}

/// @brief it should write each channel of a frame to its own output entry.
TEST_F(EtherCATWriteTest, SinkWritesMultipleChannels) {
    auto ch1 = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("ch1"),
        x::telem::INT16_T,
        this->index_channel.key,
        false
    ));
    auto ch2 = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("ch2"),
        x::telem::INT32_T,
        this->index_channel.key,
        false
    ));

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        create_output_channel_config(ch1.key, 0x7000, 1, "int16")
    );
    cfg["channels"].push_back(
        create_output_channel_config(ch2.key, 0x7000, 2, "int32")
    );

    auto parser = x::json::Parser(cfg);
    WriteTaskConfig task_cfg(this->client, parser);
    ASSERT_NIL(parser.error());

    auto sink = WriteTaskSink(this->engine, std::move(task_cfg));
    ASSERT_NIL(sink.start("start_cmd", this->cancel));

    x::telem::Series series1(x::telem::INT16_T, 1);
    series1.write(static_cast<int16_t>(0x1234));
    x::telem::Series series2(x::telem::INT32_T, 1);
    series2.write(static_cast<int32_t>(0xDEADBEEF));

    x::telem::Frame frame(2);
    frame.emplace(ch1.key, std::move(series1));
    frame.emplace(ch2.key, std::move(series2));

    ASSERT_NIL(sink.write(frame));
    ASSERT_EVENTUALLY_EQ(
        this->mock_master->get_output<int16_t>(0),
        static_cast<int16_t>(0x1234)
    );
    ASSERT_EVENTUALLY_EQ(
        this->mock_master->get_output<int32_t>(2),
        static_cast<int32_t>(0xDEADBEEF)
    );

    ASSERT_NIL(sink.stop());
}

/// @brief it should accept a frame that has no command for a channel.
TEST_F(EtherCATWriteTest, SinkIgnoresMissingChannelInFrame) {
    auto cmd_ch = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("cmd"),
        x::telem::INT16_T,
        this->index_channel.key,
        false
    ));

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        create_output_channel_config(cmd_ch.key, 0x7000, 1, "int16")
    );

    auto parser = x::json::Parser(cfg);
    WriteTaskConfig task_cfg(this->client, parser);
    ASSERT_NIL(parser.error());

    auto sink = WriteTaskSink(this->engine, std::move(task_cfg));
    ASSERT_NIL(sink.start("start_cmd", this->cancel));

    x::telem::Frame empty_frame(0);
    ASSERT_NIL(sink.write(empty_frame));

    ASSERT_NIL(sink.stop());
}

/// @brief it should refuse to start when a slave on the bus is not the device that
/// the task names.
TEST_F(EtherCATWriteTest, SinkStartFailsOnTopologyMismatch) {
    auto cmd_ch = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("cmd"),
        x::telem::INT16_T,
        this->index_channel.key,
        false
    ));

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        create_output_channel_config(cmd_ch.key, 0x7000, 1, "int16")
    );

    auto parser = x::json::Parser(cfg);
    WriteTaskConfig task_cfg(this->client, parser);
    ASSERT_NIL(parser.error());

    auto mismatched_master = std::make_shared<testutil::Master>();
    mismatched_master->add_slave(
        slave::Properties{
            .position = 1,
            .vendor_id = 0x99,
            .product_code = 0x2,
            .serial = SLAVE_SERIAL,
            .name = "Test Module",
        }
    );
    auto mismatched_engine = std::make_shared<engine::Engine>(
        NETWORK_INTERFACE,
        mismatched_master
    );

    auto sink = WriteTaskSink(mismatched_engine, std::move(task_cfg));
    const auto err = sink.start("start_cmd", this->cancel);
    ASSERT_OCCURRED_AS(err, errors::TOPOLOGY_MISMATCH);
    EXPECT_EQ(err.data, "Test Module (serial 12345) is not on the bus.");
}

}
