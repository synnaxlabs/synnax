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

        slave_device = create_slave_device(
            SLAVE_SERIAL,
            {{{"name", "status_word"},
              {"index", 0x6000},
              {"sub_index", 1},
              {"bit_length", 16},
              {"data_type", "int16"}},
             {{"name", "sensor_value"},
              {"index", 0x6000},
              {"sub_index", 2},
              {"bit_length", 32},
              {"data_type", "int32"}}},
            json::array()
        );

        mock_master = std::make_shared<ethercat::mock::Master>(NETWORK_INTERFACE);
        mock_master->add_slave(
            ethercat::slave::Properties{
                .position = 0,
                .vendor_id = 0x1,
                .product_code = 0x2,
                .serial = SLAVE_SERIAL,
                .name = "Test Slave",
                .input_pdos = {
                    {.pdo_index = 0x1A00,
                     .index = 0x6000,
                     .sub_index = 1,
                     .bit_length = 16,
                     .is_input = true,
                     .name = "status_word",
                     .data_type = telem::INT16_T},
                    {.pdo_index = 0x1A00,
                     .index = 0x6000,
                     .sub_index = 2,
                     .bit_length = 32,
                     .is_input = true,
                     .name = "sensor_value",
                     .data_type = telem::INT32_T},
                },
            }
        );
        engine = std::make_shared<ethercat::engine::Engine>(mock_master);
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
            {"enabled", true},
            {"pdos", {{"inputs", input_pdos}, {"outputs", output_pdos}}}
        };
        synnax::Device dev(
            "ecat_slave_" + std::to_string(serial),
            "Test Slave SN:" + std::to_string(serial),
            rack.key,
            NETWORK_INTERFACE + ".Slot 0",
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
        uint8_t sub_index,
        uint8_t bit_length,
        const std::string &data_type
    ) {
        return {
            {"type", "manual"},
            {"device", slave_device.key},
            {"index", index},
            {"sub_index", sub_index},
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
    EXPECT_EQ(task_cfg.channels[0]->sub_index, 1);
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
    EXPECT_EQ(task_cfg.channels[0]->sub_index, 2);
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
    EXPECT_EQ(task_cfg.channels[0]->sub_index, 1);
    EXPECT_EQ(task_cfg.channels[1]->index, 0x6000);
    EXPECT_EQ(task_cfg.channels[1]->sub_index, 3);
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

TEST_F(EtherCATReadTest, InvalidSlaveDevice) {
    auto data_ch = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("analog"),
        telem::INT16_T,
        this->index_channel.key,
        false
    ));

    json cfg = {
        {"data_saving", false},
        {"sample_rate", 100},
        {"stream_rate", 10},
        {"channels",
         {{{"type", "automatic"},
           {"device", "nonexistent_device_key"},
           {"pdo", "status_word"},
           {"channel", data_ch.key},
           {"enabled", true}}}}
    };

    auto parser = xjson::Parser(cfg);
    ethercat::ReadTaskConfig task_cfg(this->client, parser);
    ASSERT_OCCURRED_AS(parser.error(), xerrors::VALIDATION);
}

TEST_F(EtherCATReadTest, SourceReadsDataFromEngine) {
    auto data_ch = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("analog"),
        telem::INT16_T,
        this->index_channel.key,
        false
    ));

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        this->create_automatic_input_channel_config(data_ch, "status_word")
    );

    auto parser = xjson::Parser(cfg);
    ethercat::ReadTaskConfig task_cfg(this->client, parser);
    ASSERT_NIL(parser.error());

    auto source = ethercat::ReadTaskSource(this->engine, std::move(task_cfg));
    ASSERT_NIL(source.start());

    this->mock_master->set_input<int16_t>(0, 0x1234);

    breaker::Breaker brk;
    brk.start();

    telem::Frame frame;
    auto result = source.read(brk, frame);
    ASSERT_NIL(result.error);
    EXPECT_FALSE(frame.empty());

    brk.stop();
    ASSERT_NIL(source.stop());
}

TEST_F(EtherCATReadTest, SourceReadsCorrectValueFromEngine) {
    auto data_ch = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("analog"),
        telem::INT16_T,
        this->index_channel.key,
        false
    ));

    auto cfg = this->create_base_config();
    cfg["sample_rate"] = 10;
    cfg["stream_rate"] = 10;
    cfg["channels"].push_back(
        this->create_automatic_input_channel_config(data_ch, "status_word")
    );

    auto parser = xjson::Parser(cfg);
    ethercat::ReadTaskConfig task_cfg(this->client, parser);
    ASSERT_NIL(parser.error());

    auto source = ethercat::ReadTaskSource(this->engine, std::move(task_cfg));
    ASSERT_NIL(source.start());

    this->mock_master->set_input<int16_t>(0, 0x5678);

    breaker::Breaker brk;
    brk.start();

    ASSERT_EVENTUALLY_EQ(
        [&] {
            telem::Frame frame;
            source.read(brk, frame);
            if (frame.empty() || frame.series->size() < 1)
                return static_cast<int16_t>(0);
            return frame.series->at(0).at<int16_t>(0);
        }(),
        static_cast<int16_t>(0x5678)
    );

    brk.stop();
    ASSERT_NIL(source.stop());
}

TEST_F(EtherCATReadTest, SourceReadsMultipleChannelValues) {
    auto ch1 = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("status"),
        telem::INT16_T,
        this->index_channel.key,
        false
    ));
    auto ch2 = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("sensor"),
        telem::INT32_T,
        this->index_channel.key,
        false
    ));

    auto cfg = this->create_base_config();
    cfg["sample_rate"] = 10;
    cfg["stream_rate"] = 10;
    cfg["channels"].push_back(
        this->create_automatic_input_channel_config(ch1, "status_word")
    );
    cfg["channels"].push_back(
        this->create_automatic_input_channel_config(ch2, "sensor_value")
    );

    auto parser = xjson::Parser(cfg);
    ethercat::ReadTaskConfig task_cfg(this->client, parser);
    ASSERT_NIL(parser.error());

    auto source = ethercat::ReadTaskSource(this->engine, std::move(task_cfg));
    ASSERT_NIL(source.start());

    this->mock_master->set_input<int16_t>(0, 0xABCD);
    this->mock_master->set_input<int32_t>(2, 0x12345678);

    breaker::Breaker brk;
    brk.start();

    int16_t status_value = 0;
    int32_t sensor_value = 0;
    ASSERT_EVENTUALLY_EQ(
        [&] {
            telem::Frame frame;
            source.read(brk, frame);
            if (frame.empty() || frame.series->size() < 2) return 0;
            status_value = frame.series->at(0).at<int16_t>(0);
            sensor_value = frame.series->at(1).at<int32_t>(0);
            if (status_value == static_cast<int16_t>(0xABCD) &&
                sensor_value == static_cast<int32_t>(0x12345678))
                return 1;
            return 0;
        }(),
        1
    );

    brk.stop();
    ASSERT_NIL(source.stop());
}

TEST_F(EtherCATReadTest, SourceReturnsEmptyFrameWhenBreakerStopped) {
    auto data_ch = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("analog"),
        telem::INT16_T,
        this->index_channel.key,
        false
    ));

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        this->create_automatic_input_channel_config(data_ch, "status_word")
    );

    auto parser = xjson::Parser(cfg);
    ethercat::ReadTaskConfig task_cfg(this->client, parser);
    ASSERT_NIL(parser.error());

    auto source = ethercat::ReadTaskSource(this->engine, std::move(task_cfg));
    ASSERT_NIL(source.start());

    breaker::Breaker brk;

    telem::Frame frame;
    auto result = source.read(brk, frame);
    ASSERT_NIL(result.error);
    EXPECT_TRUE(frame.empty());

    ASSERT_NIL(source.stop());
}

TEST_F(EtherCATReadTest, SourceStartFailsOnTopologyMismatch) {
    auto data_ch = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("analog"),
        telem::INT16_T,
        this->index_channel.key,
        false
    ));

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        this->create_automatic_input_channel_config(data_ch, "status_word")
    );

    auto parser = xjson::Parser(cfg);
    ethercat::ReadTaskConfig task_cfg(this->client, parser);
    ASSERT_NIL(parser.error());

    auto mismatched_master = std::make_shared<ethercat::mock::Master>(
        NETWORK_INTERFACE
    );
    mismatched_master->add_slave(
        ethercat::slave::Properties{
            .position = 0,
            .vendor_id = 0x99,
            .product_code = 0x2,
            .serial = SLAVE_SERIAL,
            .name = "Test Slave",
        }
    );
    auto mismatched_engine = std::make_shared<ethercat::engine::Engine>(
        mismatched_master
    );

    auto source = ethercat::ReadTaskSource(mismatched_engine, std::move(task_cfg));
    ASSERT_OCCURRED_AS(source.start(), ethercat::TOPOLOGY_MISMATCH);
}
