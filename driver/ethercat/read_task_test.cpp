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

#include "driver/ethercat/cyclic_engine.h"
#include "driver/ethercat/mock/master.h"
#include "driver/ethercat/read_task.h"
#include "driver/pipeline/mock/pipeline.h"

class EtherCATReadTest : public ::testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> client;
    std::shared_ptr<task::MockContext> ctx;
    std::shared_ptr<ethercat::mock::MockMaster> mock_master;
    std::shared_ptr<ethercat::CyclicEngine> engine;
    synnax::Channel index_channel;
    synnax::Rack rack;

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

        mock_master = std::make_shared<ethercat::mock::MockMaster>("eth0");
        mock_master->add_slave(
            ethercat::mock::MockSlaveConfig(0, 0x1, 0x2, "Test Slave")
        );
        engine = std::make_shared<ethercat::CyclicEngine>(
            mock_master,
            ethercat::CyclicEngineConfig(telem::MILLISECOND * 10)
        );
    }

    json create_base_config() {
        return {
            {"data_saving", false},
            {"sample_rate", 100},
            {"stream_rate", 10},
            {"device", "eth0"},
            {"channels", json::array()}
        };
    }

    json create_input_channel_config(
        const synnax::Channel &channel,
        uint16_t slave_position,
        uint16_t index,
        uint8_t subindex,
        uint8_t bit_length
    ) {
        return {
            {"type", "input"},
            {"enabled", true},
            {"channel", channel.key},
            {"slave_position", slave_position},
            {"index", index},
            {"subindex", subindex},
            {"bit_length", bit_length}
        };
    }
};

TEST_F(EtherCATReadTest, ParseConfigWithValidChannel) {
    auto data_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("analog"),
        telem::INT16_T,
        index_channel.key,
        false
    ));

    auto cfg = create_base_config();
    cfg["channels"].push_back(
        create_input_channel_config(data_ch, 0, 0x6000, 1, 16)
    );

    auto parser = xjson::Parser(cfg);
    ethercat::ReadTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(task_cfg.channels.size(), 1);
    EXPECT_EQ(task_cfg.device_key, "eth0");
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
    cfg["channels"].push_back(create_input_channel_config(ch1, 0, 0x6000, 1, 16));
    cfg["channels"].push_back(create_input_channel_config(ch2, 0, 0x6000, 2, 32));

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
        create_input_channel_config(invalid_ch, 0, 0x6000, 1, 16)
    );

    auto parser = xjson::Parser(cfg);
    ethercat::ReadTaskConfig task_cfg(client, parser);
    ASSERT_OCCURRED_AS(parser.error(), xerrors::VALIDATION);
}

TEST_F(EtherCATReadTest, WriterConfigIncludesAllChannels) {
    auto ch1 = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("ch1"),
        telem::INT16_T,
        index_channel.key,
        false
    ));

    auto cfg = create_base_config();
    cfg["channels"].push_back(create_input_channel_config(ch1, 0, 0x6000, 1, 16));

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
        create_input_channel_config(data_ch, 0, 0x6000, 1, 16)
    );

    auto parser = xjson::Parser(cfg);
    ethercat::ReadTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());

    auto source = ethercat::ReadTaskSource(engine, std::move(task_cfg));
    ASSERT_NIL(source.start());
    EXPECT_TRUE(engine->running());
    ASSERT_NIL(source.stop());
}
