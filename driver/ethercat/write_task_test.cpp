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
#include "x/cpp/xtest/xtest.h"

#include "driver/ethercat/mock/master.h"
#include "driver/ethercat/write_task.h"
#include "driver/pipeline/mock/pipeline.h"
#include "engine/engine.h"

class EtherCATWriteTest : public ::testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> client;
    std::shared_ptr<task::MockContext> ctx;
    std::shared_ptr<ethercat::mock::Master> mock_master;
    std::shared_ptr<ethercat::engine::Engine> engine;
    synnax::Channel index_channel;
    synnax::Rack rack;
    synnax::Device network_device;
    synnax::Device slave_device;
    const uint32_t SLAVE_SERIAL = 12345;
    const std::string NETWORK_INTERFACE = "eth0";

    void SetUp() override {
        client = std::make_shared<synnax::Synnax>(new_test_client());

        index_channel = synnax::Channel(
            make_unique_channel_name("time_channel"),
            telem::TIMESTAMP_T,
            0,
            true
        );
        ASSERT_NIL(client->channels.create(index_channel));

        rack = ASSERT_NIL_P(client->racks.create("test_rack"));

        ctx = std::make_shared<task::MockContext>(client);

        network_device = create_network_device(NETWORK_INTERFACE);
        slave_device = create_slave_device(
            SLAVE_SERIAL,
            json::array(),
            {{{"name", "control_word"},
              {"index", 0x7000},
              {"subindex", 1},
              {"bit_length", 16},
              {"data_type", "int16"}},
             {{"name", "setpoint"},
              {"index", 0x7000},
              {"subindex", 2},
              {"bit_length", 32},
              {"data_type", "int32"}}}
        );

        mock_master = std::make_shared<ethercat::mock::Master>(NETWORK_INTERFACE);
        mock_master->add_slave(
            ethercat::mock::MockSlaveConfig(0, 0x1, 0x2, SLAVE_SERIAL, "Test Slave")
        );
        engine = std::make_shared<ethercat::engine::Engine>(mock_master);
    }

    synnax::Device create_network_device(const std::string &interface) {
        json props = {{"interface", interface}, {"rate", 100.0}};
        synnax::Device dev(
            "ecat_network_" + interface,
            "Test Network",
            rack.key,
            interface,
            "EtherCAT",
            "Network",
            props.dump()
        );
        auto err = client->devices.create(dev);
        EXPECT_TRUE(!err) << err.message();
        return dev;
    }

    synnax::Device create_slave_device(
        uint32_t serial,
        const json &input_pdos,
        const json &output_pdos
    ) {
        json props = {
            {"serial", serial},
            {"vendor_id", 0x1},
            {"product_code", 0x2},
            {"revision", 1},
            {"name", "Test Slave"},
            {"network", NETWORK_INTERFACE},
            {"position", 0},
            {"pdos", {{"inputs", input_pdos}, {"outputs", output_pdos}}}
        };
        synnax::Device dev(
            "ecat_slave_" + std::to_string(serial),
            "Test Slave SN:" + std::to_string(serial),
            rack.key,
            std::to_string(serial),
            "DEWESoft",
            "TestModule",
            props.dump()
        );
        auto err = client->devices.create(dev);
        EXPECT_TRUE(!err) << err.message();
        return dev;
    }

    json create_base_config() {
        return {
            {"data_saving", false},
            {"device", network_device.key},
            {"state_rate", 10.0},
            {"channels", json::array()}
        };
    }

    json create_automatic_output_channel_config(
        const synnax::ChannelKey &command_key,
        const std::string &pdo_name,
        synnax::ChannelKey state_key = 0
    ) {
        json cfg = {
            {"type", "automatic"},
            {"device", slave_device.key},
            {"pdo", pdo_name},
            {"cmd_channel", command_key},
            {"enabled", true}
        };
        if (state_key != 0) cfg["state_channel"] = state_key;
        return cfg;
    }

    json create_manual_output_channel_config(
        const synnax::ChannelKey &command_key,
        uint16_t index,
        uint8_t subindex,
        uint8_t bit_length,
        const std::string &data_type,
        synnax::ChannelKey state_key = 0
    ) {
        json cfg = {
            {"type", "manual"},
            {"device", slave_device.key},
            {"index", index},
            {"subindex", subindex},
            {"bit_length", bit_length},
            {"data_type", data_type},
            {"cmd_channel", command_key},
            {"enabled", true}
        };
        if (state_key != 0) cfg["state_channel"] = state_key;
        return cfg;
    }
};

TEST_F(EtherCATWriteTest, ParseConfigWithAutomaticChannel) {
    auto cmd_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("cmd"),
        telem::INT16_T,
        index_channel.key,
        false
    ));

    auto cfg = create_base_config();
    cfg["channels"].push_back(
        create_automatic_output_channel_config(cmd_ch.key, "control_word")
    );

    auto parser = xjson::Parser(cfg);
    ethercat::WriteTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(task_cfg.channels.size(), 1);
    EXPECT_EQ(task_cfg.interface_name, "eth0");
    EXPECT_EQ(task_cfg.channels[0]->index, 0x7000);
    EXPECT_EQ(task_cfg.channels[0]->subindex, 1);
    EXPECT_EQ(task_cfg.channels[0]->bit_length, 16);
}

TEST_F(EtherCATWriteTest, ParseConfigWithManualChannel) {
    auto cmd_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("cmd"),
        telem::INT32_T,
        index_channel.key,
        false
    ));

    auto cfg = create_base_config();
    cfg["channels"].push_back(
        create_manual_output_channel_config(cmd_ch.key, 0x7000, 2, 32, "int32")
    );

    auto parser = xjson::Parser(cfg);
    ethercat::WriteTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(task_cfg.channels.size(), 1);
    EXPECT_EQ(task_cfg.channels[0]->index, 0x7000);
    EXPECT_EQ(task_cfg.channels[0]->subindex, 2);
    EXPECT_EQ(task_cfg.channels[0]->bit_length, 32);
}

TEST_F(EtherCATWriteTest, ParseConfigWithStateChannel) {
    auto cmd_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("cmd"),
        telem::INT16_T,
        index_channel.key,
        false
    ));
    auto state_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("state"),
        telem::INT16_T,
        index_channel.key,
        false
    ));

    auto cfg = create_base_config();
    cfg["channels"].push_back(
        create_automatic_output_channel_config(cmd_ch.key, "control_word", state_ch.key)
    );

    auto parser = xjson::Parser(cfg);
    ethercat::WriteTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(task_cfg.channels.size(), 1);
    EXPECT_EQ(task_cfg.state_channels.size(), 1);
}

TEST_F(EtherCATWriteTest, ParseConfigWithMultipleChannels) {
    auto ch1 = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("ch1"),
        telem::INT16_T,
        index_channel.key,
        false
    ));
    auto ch2 = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("ch2"),
        telem::INT32_T,
        index_channel.key,
        false
    ));

    auto cfg = create_base_config();
    cfg["channels"].push_back(
        create_automatic_output_channel_config(ch1.key, "control_word")
    );
    cfg["channels"].push_back(
        create_automatic_output_channel_config(ch2.key, "setpoint")
    );

    auto parser = xjson::Parser(cfg);
    ethercat::WriteTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(task_cfg.channels.size(), 2);
}

TEST_F(EtherCATWriteTest, ParseConfigWithInvalidPDOName) {
    auto cmd_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("cmd"),
        telem::INT16_T,
        index_channel.key,
        false
    ));

    auto cfg = create_base_config();
    cfg["channels"].push_back(
        create_automatic_output_channel_config(cmd_ch.key, "nonexistent_pdo")
    );

    auto parser = xjson::Parser(cfg);
    ethercat::WriteTaskConfig task_cfg(client, parser);
    ASSERT_OCCURRED_AS(parser.error(), xerrors::VALIDATION);
}

TEST_F(EtherCATWriteTest, CmdKeysReturnsAllCommandChannels) {
    auto ch1 = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("ch1"),
        telem::INT16_T,
        index_channel.key,
        false
    ));
    auto ch2 = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("ch2"),
        telem::INT32_T,
        index_channel.key,
        false
    ));

    auto cfg = create_base_config();
    cfg["channels"].push_back(
        create_automatic_output_channel_config(ch1.key, "control_word")
    );
    cfg["channels"].push_back(
        create_automatic_output_channel_config(ch2.key, "setpoint")
    );

    auto parser = xjson::Parser(cfg);
    ethercat::WriteTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());

    auto cmd_keys = task_cfg.cmd_keys();
    EXPECT_EQ(cmd_keys.size(), 2);
    EXPECT_EQ(cmd_keys[0], ch1.key);
    EXPECT_EQ(cmd_keys[1], ch2.key);
}

TEST_F(EtherCATWriteTest, SinkStartRegistersWithEngine) {
    auto cmd_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("cmd"),
        telem::INT16_T,
        index_channel.key,
        false
    ));

    auto cfg = create_base_config();
    cfg["channels"].push_back(
        create_automatic_output_channel_config(cmd_ch.key, "control_word")
    );

    auto parser = xjson::Parser(cfg);
    ethercat::WriteTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());

    auto sink = ethercat::WriteTaskSink(engine, std::move(task_cfg));
    ASSERT_NIL(sink.start());
    EXPECT_TRUE(engine->running());
    ASSERT_NIL(sink.stop());
}

TEST_F(EtherCATWriteTest, InvalidSlaveDevice) {
    auto cmd_ch = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("cmd"),
        telem::INT16_T,
        this->index_channel.key,
        false
    ));

    json cfg = {
        {"data_saving", false},
        {"state_rate", 10.0},
        {"channels",
         {{{"type", "manual"},
           {"device", "nonexistent_slave_key"},
           {"index", 0x7000},
           {"subindex", 1},
           {"bit_length", 16},
           {"data_type", "int16"},
           {"cmd_channel", cmd_ch.key},
           {"enabled", true}}}}
    };

    auto parser = xjson::Parser(cfg);
    ethercat::WriteTaskConfig task_cfg(this->client, parser);
    ASSERT_OCCURRED_AS(parser.error(), xerrors::VALIDATION);
}

TEST_F(EtherCATWriteTest, ParseConfigWithMixedChannelTypes) {
    auto auto_ch = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("auto_ch"),
        telem::INT16_T,
        this->index_channel.key,
        false
    ));
    auto manual_ch = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("manual_ch"),
        telem::INT32_T,
        this->index_channel.key,
        false
    ));

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        this->create_automatic_output_channel_config(auto_ch.key, "control_word")
    );
    cfg["channels"].push_back(
        this->create_manual_output_channel_config(manual_ch.key, 0x7000, 3, 32, "int32")
    );

    auto parser = xjson::Parser(cfg);
    ethercat::WriteTaskConfig task_cfg(this->client, parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(task_cfg.channels.size(), 2);
    EXPECT_EQ(task_cfg.channels[0]->index, 0x7000);
    EXPECT_EQ(task_cfg.channels[0]->subindex, 1);
    EXPECT_EQ(task_cfg.channels[1]->index, 0x7000);
    EXPECT_EQ(task_cfg.channels[1]->subindex, 3);
}

TEST_F(EtherCATWriteTest, SinkWritesDataToEngine) {
    auto cmd_ch = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("cmd"),
        telem::INT16_T,
        this->index_channel.key,
        false
    ));

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        this->create_automatic_output_channel_config(cmd_ch.key, "control_word")
    );

    auto parser = xjson::Parser(cfg);
    ethercat::WriteTaskConfig task_cfg(this->client, parser);
    ASSERT_NIL(parser.error());

    auto sink = ethercat::WriteTaskSink(this->engine, std::move(task_cfg));
    ASSERT_NIL(sink.start());

    telem::Series series(telem::INT16_T, 1);
    series.write(static_cast<int16_t>(0x5678));
    telem::Frame frame(cmd_ch.key, std::move(series));

    ASSERT_NIL(sink.write(frame));

    ASSERT_EVENTUALLY_EQ(
        this->mock_master->get_output<int16_t>(0),
        static_cast<int16_t>(0x5678)
    );

    ASSERT_NIL(sink.stop());
}

TEST_F(EtherCATWriteTest, SinkWritesMultipleChannels) {
    auto ch1 = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("ch1"),
        telem::INT16_T,
        this->index_channel.key,
        false
    ));
    auto ch2 = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("ch2"),
        telem::INT32_T,
        this->index_channel.key,
        false
    ));

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        this->create_automatic_output_channel_config(ch1.key, "control_word")
    );
    cfg["channels"].push_back(
        this->create_automatic_output_channel_config(ch2.key, "setpoint")
    );

    auto parser = xjson::Parser(cfg);
    ethercat::WriteTaskConfig task_cfg(this->client, parser);
    ASSERT_NIL(parser.error());

    auto sink = ethercat::WriteTaskSink(this->engine, std::move(task_cfg));
    ASSERT_NIL(sink.start());

    telem::Series series1(telem::INT16_T, 1);
    series1.write(static_cast<int16_t>(0x1234));
    telem::Series series2(telem::INT32_T, 1);
    series2.write(static_cast<int32_t>(0xDEADBEEF));

    telem::Frame frame(2);
    frame.emplace(ch1.key, std::move(series1));
    frame.emplace(ch2.key, std::move(series2));

    ASSERT_NIL(sink.write(frame));

    ASSERT_NIL(sink.stop());
}

TEST_F(EtherCATWriteTest, SinkIgnoresMissingChannelInFrame) {
    auto cmd_ch = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("cmd"),
        telem::INT16_T,
        this->index_channel.key,
        false
    ));

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        this->create_automatic_output_channel_config(cmd_ch.key, "control_word")
    );

    auto parser = xjson::Parser(cfg);
    ethercat::WriteTaskConfig task_cfg(this->client, parser);
    ASSERT_NIL(parser.error());

    auto sink = ethercat::WriteTaskSink(this->engine, std::move(task_cfg));
    ASSERT_NIL(sink.start());

    telem::Frame empty_frame(0);
    ASSERT_NIL(sink.write(empty_frame));

    ASSERT_NIL(sink.stop());
}

TEST_F(EtherCATWriteTest, SinkStartFailsOnTopologyMismatch) {
    auto cmd_ch = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("cmd"),
        telem::INT16_T,
        this->index_channel.key,
        false
    ));

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        this->create_automatic_output_channel_config(cmd_ch.key, "control_word")
    );

    auto parser = xjson::Parser(cfg);
    ethercat::WriteTaskConfig task_cfg(this->client, parser);
    ASSERT_NIL(parser.error());

    auto mismatched_master = std::make_shared<ethercat::mock::Master>(
        NETWORK_INTERFACE
    );
    mismatched_master->add_slave(
        ethercat::mock::MockSlaveConfig(0, 0x99, 0x2, SLAVE_SERIAL, "Test Slave")
    );
    auto mismatched_engine = std::make_shared<ethercat::engine::Engine>(
        mismatched_master
    );

    auto sink = ethercat::WriteTaskSink(mismatched_engine, std::move(task_cfg));
    ASSERT_OCCURRED_AS(sink.start(), ethercat::TOPOLOGY_MISMATCH);
}
