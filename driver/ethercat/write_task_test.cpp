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
#include "driver/ethercat/write_task.h"
#include "driver/pipeline/mock/pipeline.h"

class EtherCATWriteTest : public ::testing::Test {
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
            {"device", "eth0"},
            {"state_rate", 10.0},
            {"channels", json::array()}
        };
    }

    json create_output_channel_config(
        const synnax::ChannelKey &command_key,
        uint16_t slave_position,
        uint16_t index,
        uint8_t subindex,
        uint8_t bit_length,
        synnax::ChannelKey state_key = 0
    ) {
        json cfg = {
            {"type", "output"},
            {"enabled", true},
            {"channel", command_key},
            {"slave_position", slave_position},
            {"index", index},
            {"subindex", subindex},
            {"bit_length", bit_length}
        };
        if (state_key != 0) cfg["state_channel"] = state_key;
        return cfg;
    }
};

TEST_F(EtherCATWriteTest, ParseConfigWithValidChannel) {
    auto cmd_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("cmd"),
        telem::INT16_T,
        index_channel.key,
        false
    ));

    auto cfg = create_base_config();
    cfg["channels"].push_back(
        create_output_channel_config(cmd_ch.key, 0, 0x7000, 1, 16)
    );

    auto parser = xjson::Parser(cfg);
    ethercat::WriteTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(task_cfg.channels.size(), 1);
    EXPECT_EQ(task_cfg.device_key, "eth0");
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
        create_output_channel_config(cmd_ch.key, 0, 0x7000, 1, 16, state_ch.key)
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
        create_output_channel_config(ch1.key, 0, 0x7000, 1, 16)
    );
    cfg["channels"].push_back(
        create_output_channel_config(ch2.key, 0, 0x7000, 2, 32)
    );

    auto parser = xjson::Parser(cfg);
    ethercat::WriteTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(task_cfg.channels.size(), 2);
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
        create_output_channel_config(ch1.key, 0, 0x7000, 1, 16)
    );
    cfg["channels"].push_back(
        create_output_channel_config(ch2.key, 0, 0x7000, 2, 32)
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
        create_output_channel_config(cmd_ch.key, 0, 0x7000, 1, 16)
    );

    auto parser = xjson::Parser(cfg);
    ethercat::WriteTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());

    auto sink = ethercat::WriteTaskSink(engine, std::move(task_cfg));
    ASSERT_NIL(sink.start());
    EXPECT_TRUE(engine->running());
    ASSERT_NIL(sink.stop());
}
