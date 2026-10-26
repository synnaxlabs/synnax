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
#include "x/cpp/test/test.h"

#include "driver/ethercat/mock/master.h"
#include "driver/ethercat/write_task.h"
#include "driver/pipeline/mock/pipeline.h"
#include "engine/engine.h"

namespace driver::ethercat {
class EtherCATWriteTest : public ::testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> client;
    std::shared_ptr<task::MockContext> ctx;
    std::shared_ptr<mock::Master> mock_master;
    std::shared_ptr<engine::Engine> engine;
    synnax::channel::Channel index_channel;
    synnax::rack::Rack rack;
    synnax::device::Device network_device;
    synnax::device::Device slave_device;
    const uint32_t SLAVE_SERIAL = 12345;
    const std::string NETWORK_INTERFACE = "eth0";

    void SetUp() override {
        client = std::make_shared<synnax::Synnax>(new_test_client());

        index_channel = synnax::channel::Channel{
            .name = make_unique_channel_name("time_channel"),
            .data_type = x::telem::TIMESTAMP_T,
            .is_index = true,
        };
        ASSERT_NIL(client->channels.create(index_channel));

        rack = ASSERT_NIL_P(client->racks.create("test_rack"));

        ctx = std::make_shared<task::MockContext>(client);

        network_device = create_network_device(NETWORK_INTERFACE);
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

        mock_master = std::make_shared<mock::Master>(NETWORK_INTERFACE);
        mock_master->add_slave(
            slave::Properties{
                .position = 0,
                .vendor_id = 0x1,
                .product_code = 0x2,
                .serial = SLAVE_SERIAL,
                .name = "Test Slave",
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
            }
        );
        engine = std::make_shared<engine::Engine>(mock_master);
    }

    synnax::device::Device create_network_device(const std::string &interface) {
        x::json::json props = {{"interface", interface}, {"rate", 100.0}};
        synnax::device::Device dev{
            .key = "ecat_network_" + interface,
            .rack = rack.key,
            .location = interface,
            .make = "EtherCAT",
            .model = "Network",
            .name = "Test Network",
            .properties = props,
        };
        const auto err = client->devices.create(dev);
        EXPECT_TRUE(!err) << err.message();
        return dev;
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
            {"name", "Test Slave"},
            {"network", NETWORK_INTERFACE},
            {"position", 0},
            {"enabled", true},
            {"pdos", {{"inputs", input_pdos}, {"outputs", output_pdos}}}
        };
        synnax::device::Device dev{
            .key = "ecat_slave_" + std::to_string(serial),
            .rack = rack.key,
            .location = std::to_string(serial),
            .make = "DEWESoft",
            .model = "TestModule",
            .name = "Test Slave SN:" + std::to_string(serial),
            .properties = props,
        };
        const auto err = client->devices.create(dev);
        EXPECT_TRUE(!err) << err.message();
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
};

/// @brief it should parse a write channel bound to an entry address.
TEST_F(EtherCATWriteTest, ParseConfigWithEntryChannel) {
    const auto cmd_ch = create_command_channel("cmd", x::telem::INT16_T);

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        create_output_channel_config(cmd_ch.key, 0x7000, 1, "int16")
    );

    auto parser = x::json::Parser(cfg);
    WriteTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(task_cfg.outputs.size(), 1);
    EXPECT_EQ(task_cfg.outputs[0]->index, 0x7000);
    EXPECT_EQ(task_cfg.outputs[0]->sub_index, 1);
    EXPECT_EQ(task_cfg.outputs[0]->bit_length, 16);
    EXPECT_EQ(task_cfg.outputs[0]->data_type, x::telem::INT16_T);
}

/// @brief it should take the bit length of a write channel from the slave.
TEST_F(EtherCATWriteTest, ParseConfigTakesBitLengthFromTheSlave) {
    const auto cmd_ch = create_command_channel("cmd", x::telem::FLOAT32_T);

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        create_output_channel_config(cmd_ch.key, 0x7000, 2, "float32")
    );

    auto parser = x::json::Parser(cfg);
    WriteTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(task_cfg.outputs.size(), 1);
    EXPECT_EQ(task_cfg.outputs[0]->index, 0x7000);
    EXPECT_EQ(task_cfg.outputs[0]->sub_index, 2);
    EXPECT_EQ(task_cfg.outputs[0]->bit_length, 32);
    EXPECT_EQ(task_cfg.outputs[0]->data_type, x::telem::FLOAT32_T);
}

TEST_F(EtherCATWriteTest, ParseConfigWithStateChannel) {
    auto cmd_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("cmd"),
        x::telem::INT16_T,
        index_channel.key,
        false
    ));
    auto state_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("state"),
        x::telem::INT16_T,
        index_channel.key,
        false
    ));

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        create_output_channel_config(cmd_ch.key, 0x7000, 1, "int16", state_ch.key)
    );

    auto parser = x::json::Parser(cfg);
    WriteTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(task_cfg.outputs.size(), 1);
    ASSERT_TRUE(task_cfg.outputs[0]->state.has_value());
    EXPECT_EQ(task_cfg.outputs[0]->state.value().key, state_ch.key);
}

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

TEST_F(EtherCATWriteTest, ParseConfigWithMultipleChannels) {
    auto ch1 = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("ch1"),
        x::telem::INT16_T,
        index_channel.key,
        false
    ));
    auto ch2 = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("ch2"),
        x::telem::INT32_T,
        index_channel.key,
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
    WriteTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(task_cfg.outputs.size(), 2);
}

/// @brief it should report a write channel at an address the slave does not have.
TEST_F(EtherCATWriteTest, ParseConfigWithAddressMissingFromSlave) {
    const auto cmd_ch = create_command_channel("cmd", x::telem::INT32_T);

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        create_output_channel_config(cmd_ch.key, 0x7000, 3, "int32")
    );

    auto parser = x::json::Parser(cfg);
    WriteTaskConfig task_cfg(client, parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    const auto errors = parser.error_json()["errors"];
    ASSERT_EQ(errors.size(), 1);
    EXPECT_EQ(errors[0]["path"], "channels.0.index");
    EXPECT_EQ(
        errors[0]["message"],
        "slave Test Slave has no output entry at 0x7000:03"
    );
}

/// @brief it should report a write channel whose name was not resolved to an address.
TEST_F(EtherCATWriteTest, ParseConfigWithUnresolvedChannel) {
    const auto cmd_ch = create_command_channel("cmd", x::telem::INT16_T);

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        create_unresolved_output_channel_config(cmd_ch.key, "Analog Output Ch2")
    );

    auto parser = x::json::Parser(cfg);
    WriteTaskConfig task_cfg(client, parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    const auto errors = parser.error_json()["errors"];
    ASSERT_EQ(errors.size(), 1);
    EXPECT_EQ(errors[0]["path"], "channels.0.pdo");
    EXPECT_EQ(
        errors[0]["message"],
        "'Analog Output Ch2' matches no output entry on slave Test Slave. Pick an "
        "entry."
    );
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
        "another channel already uses entry 0x7000:01 on slave Test Slave"
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

/// @brief it should bind a write channel to each entry of a slave whose entries all
/// have the same name.
TEST_F(EtherCATWriteTest, ParseConfigReachesEveryEntryOfASlaveThatRepeatsOneName) {
    x::json::json outputs = x::json::json::array();
    for (int i = 0; i < 8; i++)
        outputs.push_back(
            {{"name", "Output"},
             {"pdo_name", "Channel " + std::to_string(i + 1)},
             {"pdo_index", 0x1600 + i},
             {"index", 0x7000 + 0x10 * i},
             {"sub_index", 1},
             {"bit_length", 1},
             {"data_type", "uint8"}}
        );
    slave_device = create_slave_device(
        SLAVE_SERIAL + 1,
        x::json::json::array(),
        outputs
    );

    auto cfg = this->create_base_config();
    for (int i = 0; i < 8; i++) {
        const auto ch = create_command_channel("output", x::telem::UINT8_T);
        cfg["channels"].push_back(
            create_output_channel_config(ch.key, 0x7000 + 0x10 * i, 1, "uint8")
        );
    }

    auto parser = x::json::Parser(cfg);
    WriteTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(task_cfg.outputs.size(), 8);
    for (int i = 0; i < 8; i++) {
        EXPECT_EQ(task_cfg.outputs[i]->index, 0x7000 + 0x10 * i);
        EXPECT_EQ(task_cfg.outputs[i]->sub_index, 1);
        EXPECT_EQ(task_cfg.outputs[i]->bit_length, 1);
    }
}

/// @brief it should list the command channel of every output in bus order.
TEST_F(EtherCATWriteTest, OutputChannelsHoldEveryCommandChannel) {
    auto ch1 = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("ch1"),
        x::telem::INT16_T,
        index_channel.key,
        false
    ));
    auto ch2 = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("ch2"),
        x::telem::INT32_T,
        index_channel.key,
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
    WriteTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());

    const auto outs = task_cfg.output_channels();
    ASSERT_EQ(outs.size(), 2);
    EXPECT_EQ(outs[0].command, ch1.key);
    EXPECT_EQ(outs[1].command, ch2.key);
}

TEST_F(EtherCATWriteTest, SinkStartRegistersWithEngine) {
    auto cmd_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("cmd"),
        x::telem::INT16_T,
        index_channel.key,
        false
    ));

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        create_output_channel_config(cmd_ch.key, 0x7000, 1, "int16")
    );

    auto parser = x::json::Parser(cfg);
    WriteTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());

    auto sink = WriteTaskSink(engine, std::move(task_cfg));
    ASSERT_NIL(sink.start());
    EXPECT_TRUE(engine->running());
    ASSERT_NIL(sink.stop());
}

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

    auto parser = x::json::Parser(cfg);
    WriteTaskConfig task_cfg(this->client, parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
}

/// @brief it should report a write channel with an unknown type.
TEST_F(EtherCATWriteTest, ParseConfigReportsAnUnknownChannelType) {
    const auto cmd_ch = create_command_channel("cmd", x::telem::INT16_T);

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        {{"type", "manual"},
         {"device", slave_device.key},
         {"index", 0x7000},
         {"sub_index", 1},
         {"bit_length", 16},
         {"data_type", "int16"},
         {"cmd_channel", cmd_ch.key},
         {"disabled", false}}
    );

    auto parser = x::json::Parser(cfg);
    WriteTaskConfig task_cfg(client, parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    const auto errors = parser.error_json()["errors"];
    ASSERT_EQ(errors.size(), 1);
    EXPECT_EQ(errors[0]["path"], "channels.0.type");
    EXPECT_EQ(errors[0]["message"], "unknown WriteChannel type: manual");
}

/// @brief it should report a write channel whose data type is wider than its entry.
TEST_F(EtherCATWriteTest, ParseConfigRejectsADataTypeWiderThanTheEntry) {
    const auto cmd_ch = create_command_channel("cmd", x::telem::FLOAT64_T);

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        create_output_channel_config(cmd_ch.key, 0x7000, 1, "float64")
    );

    auto parser = x::json::Parser(cfg);
    WriteTaskConfig task_cfg(client, parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    const auto errors = parser.error_json()["errors"];
    ASSERT_EQ(errors.size(), 1);
    EXPECT_EQ(errors[0]["path"], "channels.0.data_type");
    EXPECT_EQ(
        errors[0]["message"],
        "data type float64 is 8 bytes but entry 0x7000:01 on slave Test Slave holds 2 "
        "bytes. Pick a 2 byte type."
    );
}

TEST_F(EtherCATWriteTest, SinkWritesDataToEngine) {
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
    ASSERT_NIL(sink.start());

    x::telem::Series series(x::telem::INT16_T, 1);
    series.write(static_cast<int16_t>(0x5678));
    x::telem::Frame frame(cmd_ch.key, std::move(series));

    ASSERT_NIL(sink.write(frame));

    ASSERT_EVENTUALLY_EQ(
        this->mock_master->get_output<int16_t>(0),
        static_cast<int16_t>(0x5678)
    );

    ASSERT_NIL(sink.stop());
}

/// @brief when a frame contains multiple samples for a channel, only the last
/// sample should be written to hardware.
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
    ASSERT_NIL(sink.start());

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
    ASSERT_NIL(sink.start());

    x::telem::Series series1(x::telem::INT16_T, 1);
    series1.write(static_cast<int16_t>(0x1234));
    x::telem::Series series2(x::telem::INT32_T, 1);
    series2.write(static_cast<int32_t>(0xDEADBEEF));

    x::telem::Frame frame(2);
    frame.emplace(ch1.key, std::move(series1));
    frame.emplace(ch2.key, std::move(series2));

    ASSERT_NIL(sink.write(frame));

    ASSERT_NIL(sink.stop());
}

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
    ASSERT_NIL(sink.start());

    x::telem::Frame empty_frame(0);
    ASSERT_NIL(sink.write(empty_frame));

    ASSERT_NIL(sink.stop());
}

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

    auto mismatched_master = std::make_shared<mock::Master>(NETWORK_INTERFACE);
    mismatched_master->add_slave(
        slave::Properties{
            .position = 0,
            .vendor_id = 0x99,
            .product_code = 0x2,
            .serial = SLAVE_SERIAL,
            .name = "Test Slave",
        }
    );
    auto mismatched_engine = std::make_shared<engine::Engine>(mismatched_master);

    auto sink = WriteTaskSink(mismatched_engine, std::move(task_cfg));
    ASSERT_OCCURRED_AS(sink.start(), errors::TOPOLOGY_MISMATCH);
}

/// @brief it should fail to start the sink when the bus no longer has a configured
/// entry.
TEST_F(EtherCATWriteTest, SinkStartFailsWhenTheBusLacksAnEntry) {
    const auto cmd_ch = create_command_channel("setpoint", x::telem::INT32_T);

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        create_output_channel_config(cmd_ch.key, 0x7000, 2, "int32")
    );

    auto parser = x::json::Parser(cfg);
    WriteTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());

    auto remapped_master = std::make_shared<mock::Master>(NETWORK_INTERFACE);
    remapped_master->add_slave(
        slave::Properties{
            .position = 0,
            .vendor_id = 0x1,
            .product_code = 0x2,
            .serial = SLAVE_SERIAL,
            .name = "Test Slave",
            .output_pdos = {
                {.pdo_index = 0x1600,
                 .index = 0x7000,
                 .sub_index = 1,
                 .bit_length = 16,
                 .is_input = false,
                 .name = "control_word",
                 .data_type = x::telem::INT16_T},
            },
        }
    );
    auto remapped_engine = std::make_shared<engine::Engine>(remapped_master);

    auto sink = WriteTaskSink(remapped_engine, std::move(task_cfg));
    const auto err = sink.start();
    ASSERT_OCCURRED_AS(err, errors::PDO_MAPPING_ERROR);
    EXPECT_EQ(err.data, "PDO 0x7000:02 is not in the process image of slave 0");
    EXPECT_FALSE(remapped_engine->running());
}

/// @brief it should fail to start the sink when a configured entry changed its bit
/// length on the bus.
TEST_F(EtherCATWriteTest, SinkStartFailsWhenAnEntryChangedBitLength) {
    const auto cmd_ch = create_command_channel("setpoint", x::telem::INT32_T);

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        create_output_channel_config(cmd_ch.key, 0x7000, 2, "int32")
    );

    auto parser = x::json::Parser(cfg);
    WriteTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());

    auto remapped_master = std::make_shared<mock::Master>(NETWORK_INTERFACE);
    remapped_master->add_slave(
        slave::Properties{
            .position = 0,
            .vendor_id = 0x1,
            .product_code = 0x2,
            .serial = SLAVE_SERIAL,
            .name = "Test Slave",
            .output_pdos = {
                {.pdo_index = 0x1600,
                 .index = 0x7000,
                 .sub_index = 2,
                 .bit_length = 16,
                 .is_input = false,
                 .name = "setpoint",
                 .data_type = x::telem::INT16_T},
            },
        }
    );
    auto remapped_engine = std::make_shared<engine::Engine>(remapped_master);

    auto sink = WriteTaskSink(remapped_engine, std::move(task_cfg));
    const auto err = sink.start();
    ASSERT_OCCURRED_AS(err, errors::PDO_MAPPING_ERROR);
    EXPECT_EQ(
        err.data,
        "PDO 0x7000:02 on slave 0 is 16 bits, not 32. Scan the network again"
    );
    EXPECT_FALSE(remapped_engine->running());
}
}
