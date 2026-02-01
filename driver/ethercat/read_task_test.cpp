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
#include "driver/ethercat/read_task.h"
#include "driver/pipeline/mock/pipeline.h"
#include "engine/engine.h"

class EtherCATReadTest : public ::testing::Test {
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

        network_device = create_network_device("eth0");
        slave_device = create_slave_device(
            SLAVE_SERIAL,
            {{{"name", "status_word"},
              {"index", 0x6000},
              {"subindex", 1},
              {"bit_length", 16},
              {"data_type", "int16"}},
             {{"name", "sensor_value"},
              {"index", 0x6000},
              {"subindex", 2},
              {"bit_length", 32},
              {"data_type", "int32"}}},
            json::array()
        );

        mock_master = std::make_shared<ethercat::mock::Master>("eth0");
        mock_master->add_slave(
            ethercat::mock::MockSlaveConfig(0, 0x1, 0x2, SLAVE_SERIAL, "Test Slave")
        );
        engine = std::make_shared<ethercat::engine::Engine>(
            mock_master,
            ethercat::engine::Config(telem::MILLISECOND * 10)
        );
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
            {"sample_rate", 100},
            {"stream_rate", 10},
            {"device", network_device.key},
            {"channels", json::array()}
        };
    }

    json create_automatic_input_channel_config(
        const synnax::Channel &channel,
        const std::string &pdo_name
    ) {
        return {
            {"type", "automatic"},
            {"device", slave_device.key},
            {"pdo", pdo_name},
            {"channel", channel.key},
            {"enabled", true}
        };
    }

    json create_manual_input_channel_config(
        const synnax::Channel &channel,
        uint16_t index,
        uint8_t subindex,
        uint8_t bit_length,
        const std::string &data_type
    ) {
        return {
            {"type", "manual"},
            {"device", slave_device.key},
            {"index", index},
            {"subindex", subindex},
            {"bit_length", bit_length},
            {"data_type", data_type},
            {"channel", channel.key},
            {"enabled", true}
        };
    }
};

TEST_F(EtherCATReadTest, ParseConfigWithAutomaticChannel) {
    auto data_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("analog"),
        telem::INT16_T,
        index_channel.key,
        false
    ));

    auto cfg = create_base_config();
    cfg["channels"].push_back(
        create_automatic_input_channel_config(data_ch, "status_word")
    );

    auto parser = xjson::Parser(cfg);
    ethercat::ReadTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(task_cfg.channels.size(), 1);
    EXPECT_EQ(task_cfg.interface_name, "eth0");
    EXPECT_EQ(task_cfg.channels[0]->index, 0x6000);
    EXPECT_EQ(task_cfg.channels[0]->subindex, 1);
    EXPECT_EQ(task_cfg.channels[0]->bit_length, 16);
}

TEST_F(EtherCATReadTest, ParseConfigWithManualChannel) {
    auto data_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("analog"),
        telem::INT32_T,
        index_channel.key,
        false
    ));

    auto cfg = create_base_config();
    cfg["channels"].push_back(
        create_manual_input_channel_config(data_ch, 0x6000, 2, 32, "int32")
    );

    auto parser = xjson::Parser(cfg);
    ethercat::ReadTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(task_cfg.channels.size(), 1);
    EXPECT_EQ(task_cfg.channels[0]->index, 0x6000);
    EXPECT_EQ(task_cfg.channels[0]->subindex, 2);
    EXPECT_EQ(task_cfg.channels[0]->bit_length, 32);
}

TEST_F(EtherCATReadTest, ParseConfigWithMultipleChannels) {
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
        create_automatic_input_channel_config(ch1, "status_word")
    );
    cfg["channels"].push_back(
        create_automatic_input_channel_config(ch2, "sensor_value")
    );

    auto parser = xjson::Parser(cfg);
    ethercat::ReadTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(task_cfg.channels.size(), 2);
}

TEST_F(EtherCATReadTest, ParseConfigWithInvalidChannel) {
    auto cfg = create_base_config();
    synnax::Channel invalid_ch;
    invalid_ch.key = 99999;
    cfg["channels"].push_back(
        create_automatic_input_channel_config(invalid_ch, "status_word")
    );

    auto parser = xjson::Parser(cfg);
    ethercat::ReadTaskConfig task_cfg(client, parser);
    ASSERT_OCCURRED_AS(parser.error(), xerrors::VALIDATION);
}

TEST_F(EtherCATReadTest, ParseConfigWithInvalidPDOName) {
    auto data_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("analog"),
        telem::INT16_T,
        index_channel.key,
        false
    ));

    auto cfg = create_base_config();
    cfg["channels"].push_back(
        create_automatic_input_channel_config(data_ch, "nonexistent_pdo")
    );

    auto parser = xjson::Parser(cfg);
    ethercat::ReadTaskConfig task_cfg(client, parser);
    ASSERT_OCCURRED_AS(parser.error(), xerrors::VALIDATION);
}

TEST_F(EtherCATReadTest, ParseConfigWithMixedChannelTypes) {
    auto auto_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("auto_ch"),
        telem::INT16_T,
        index_channel.key,
        false
    ));
    auto manual_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("manual_ch"),
        telem::INT32_T,
        index_channel.key,
        false
    ));

    auto cfg = create_base_config();
    cfg["channels"].push_back(
        create_automatic_input_channel_config(auto_ch, "status_word")
    );
    cfg["channels"].push_back(
        create_manual_input_channel_config(manual_ch, 0x6000, 3, 32, "int32")
    );

    auto parser = xjson::Parser(cfg);
    ethercat::ReadTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(task_cfg.channels.size(), 2);
    EXPECT_EQ(task_cfg.channels[0]->index, 0x6000);
    EXPECT_EQ(task_cfg.channels[0]->subindex, 1);
    EXPECT_EQ(task_cfg.channels[1]->index, 0x6000);
    EXPECT_EQ(task_cfg.channels[1]->subindex, 3);
}

TEST_F(EtherCATReadTest, WriterConfigIncludesAllChannels) {
    auto ch1 = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("ch1"),
        telem::INT16_T,
        index_channel.key,
        false
    ));

    auto cfg = create_base_config();
    cfg["channels"].push_back(
        create_automatic_input_channel_config(ch1, "status_word")
    );

    auto parser = xjson::Parser(cfg);
    ethercat::ReadTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());

    auto writer_cfg = task_cfg.writer_config();
    EXPECT_EQ(writer_cfg.channels.size(), 2);
}

TEST_F(EtherCATReadTest, SourceStartRegistersWithEngine) {
    auto data_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("analog"),
        telem::INT16_T,
        index_channel.key,
        false
    ));

    auto cfg = create_base_config();
    cfg["channels"].push_back(
        create_automatic_input_channel_config(data_ch, "status_word")
    );

    auto parser = xjson::Parser(cfg);
    ethercat::ReadTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());

    auto source = ethercat::ReadTaskSource(engine, std::move(task_cfg));
    ASSERT_NIL(source.start());
    EXPECT_TRUE(engine->running());
    ASSERT_NIL(source.stop());
}

TEST_F(EtherCATReadTest, InvalidNetworkDevice) {
    json cfg = {
        {"data_saving", false},
        {"sample_rate", 100},
        {"stream_rate", 10},
        {"device", "nonexistent_device_key"},
        {"channels", json::array()}
    };

    auto parser = xjson::Parser(cfg);
    ethercat::ReadTaskConfig task_cfg(client, parser);
    ASSERT_OCCURRED_AS(parser.error(), xerrors::VALIDATION);
}
