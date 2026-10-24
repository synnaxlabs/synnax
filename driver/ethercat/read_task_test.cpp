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
#include "driver/ethercat/read_task.h"
#include "driver/pipeline/mock/pipeline.h"
#include "engine/engine.h"

namespace driver::ethercat {
class EtherCATReadTest : public ::testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> client;
    std::shared_ptr<task::MockContext> ctx;
    std::shared_ptr<mock::Master> mock_master;
    std::shared_ptr<engine::Engine> engine;
    synnax::channel::Channel index_channel;
    synnax::rack::Rack rack;
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
            x::json::json::array()
        );

        mock_master = std::make_shared<mock::Master>(NETWORK_INTERFACE);
        mock_master->add_slave(
            slave::Properties{
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
                     .data_type = x::telem::INT16_T},
                    {.pdo_index = 0x1A00,
                     .index = 0x6000,
                     .sub_index = 2,
                     .bit_length = 32,
                     .is_input = true,
                     .name = "sensor_value",
                     .data_type = x::telem::INT32_T},
                },
            }
        );
        engine = std::make_shared<engine::Engine>(mock_master);
    }

    synnax::device::Device create_slave_device(
        uint32_t serial,
        const x::json::json &input_pdos,
        const x::json::json &output_pdos,
        uint16_t position = 0
    ) {
        x::json::json props = {
            {"serial", serial},
            {"vendor_id", 0x1},
            {"product_code", 0x2},
            {"revision", 1},
            {"name", "Test Slave"},
            {"network", NETWORK_INTERFACE},
            {"position", position},
            {"enabled", true},
            {"pdos", {{"inputs", input_pdos}, {"outputs", output_pdos}}}
        };
        synnax::device::Device dev{
            .key = "ecat_slave_" + std::to_string(serial),
            .rack = rack.key,
            .location = NETWORK_INTERFACE + ".Slot 0",
            .make = "DEWESoft",
            .model = "TestModule",
            .name = "Test Slave SN:" + std::to_string(serial),
            .properties = props,
        };
        auto err = client->devices.create(dev);
        EXPECT_TRUE(!err) << err.message();
        return dev;
    }

    x::json::json create_base_config() {
        return {
            {"data_saving_disabled", true},
            {"sample_rate", 100},
            {"stream_rate", 10},
            {"channels", x::json::json::array()}
        };
    }

    x::json::json create_input_channel_config(
        const synnax::channel::Channel &channel,
        uint16_t index,
        uint8_t sub_index,
        const std::string &data_type,
        bool disabled = false
    ) {
        return {
            {"type", "entry"},
            {"device", slave_device.key},
            {"index", index},
            {"sub_index", sub_index},
            {"data_type", data_type},
            {"channel", channel.key},
            {"disabled", disabled}
        };
    }

    x::json::json create_unresolved_input_channel_config(
        const synnax::channel::Channel &channel,
        const std::string &pdo_name
    ) {
        return {
            {"type", "unresolved"},
            {"device", slave_device.key},
            {"pdo", pdo_name},
            {"channel", channel.key},
            {"disabled", false}
        };
    }

    /// @brief starts a source that reads the int16 entry at 0x6000:01 into a float64
    /// channel with the given scale.
    std::unique_ptr<ReadTaskSource> start_scaled_source(const x::json::json &scale) {
        const auto data_ch = create_data_channel("scaled", x::telem::FLOAT64_T);
        auto cfg = create_base_config();
        cfg["sample_rate"] = 10;
        cfg["stream_rate"] = 10;
        auto ch = create_input_channel_config(data_ch, 0x6000, 1, "int16");
        ch["scale"] = scale;
        cfg["channels"].push_back(ch);
        auto parser = x::json::Parser(cfg);
        ReadTaskConfig task_cfg(client, parser);
        EXPECT_FALSE(parser.error()) << parser.error().message();
        auto source = std::make_unique<ReadTaskSource>(engine, std::move(task_cfg));
        EXPECT_FALSE(source->start());
        return source;
    }

    synnax::channel::Channel
    create_data_channel(const std::string &name, const x::telem::DataType &data_type) {
        return ASSERT_NIL_P(client->channels.create(
            make_unique_channel_name(name),
            data_type,
            index_channel.key,
            false
        ));
    }
};

TEST_F(EtherCATReadTest, ParseConfigWithEntryChannel) {
    const auto data_ch = create_data_channel("analog", x::telem::INT16_T);

    auto cfg = create_base_config();
    cfg["channels"].push_back(create_input_channel_config(data_ch, 0x6000, 1, "int16"));

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(task_cfg.channels.size(), 1);
    EXPECT_EQ(task_cfg.interface_name, "eth0");
    EXPECT_EQ(task_cfg.channels[0]->index, 0x6000);
    EXPECT_EQ(task_cfg.channels[0]->sub_index, 1);
    EXPECT_EQ(task_cfg.channels[0]->bit_length, 16);
    EXPECT_EQ(task_cfg.channels[0]->data_type, x::telem::INT16_T);
}

TEST_F(EtherCATReadTest, ParseConfigTakesBitLengthFromTheSlave) {
    const auto data_ch = create_data_channel("analog", x::telem::FLOAT32_T);

    auto cfg = create_base_config();
    cfg["channels"].push_back(
        create_input_channel_config(data_ch, 0x6000, 2, "float32")
    );

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(task_cfg.channels.size(), 1);
    EXPECT_EQ(task_cfg.channels[0]->index, 0x6000);
    EXPECT_EQ(task_cfg.channels[0]->sub_index, 2);
    EXPECT_EQ(task_cfg.channels[0]->bit_length, 32);
    EXPECT_EQ(task_cfg.channels[0]->data_type, x::telem::FLOAT32_T);
}

TEST_F(EtherCATReadTest, ParseConfigWithMultipleChannels) {
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

    auto cfg = create_base_config();
    cfg["channels"].push_back(create_input_channel_config(ch1, 0x6000, 1, "int16"));
    cfg["channels"].push_back(create_input_channel_config(ch2, 0x6000, 2, "int32"));

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(task_cfg.channels.size(), 2);
}

TEST_F(EtherCATReadTest, ParseConfigWithInvalidChannel) {
    auto cfg = create_base_config();
    synnax::channel::Channel invalid_ch;
    invalid_ch.key = 99999;
    cfg["channels"].push_back(
        create_input_channel_config(invalid_ch, 0x6000, 1, "int16")
    );

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
}

TEST_F(EtherCATReadTest, ParseConfigWithAddressMissingFromSlave) {
    const auto data_ch = create_data_channel("analog", x::telem::INT32_T);

    auto cfg = create_base_config();
    cfg["channels"].push_back(create_input_channel_config(data_ch, 0x6000, 3, "int32"));

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    const auto errors = parser.error_json()["errors"];
    ASSERT_EQ(errors.size(), 1);
    EXPECT_EQ(errors[0]["path"], "channels.0.index");
    EXPECT_EQ(errors[0]["message"], "slave Test Slave has no input entry at 0x6000:03");
}

TEST_F(EtherCATReadTest, ParseConfigWithUnresolvedChannel) {
    const auto data_ch = create_data_channel("analog", x::telem::INT16_T);

    auto cfg = create_base_config();
    cfg["channels"].push_back(
        create_unresolved_input_channel_config(data_ch, "Analog Input Ch1")
    );

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    const auto errors = parser.error_json()["errors"];
    ASSERT_EQ(errors.size(), 1);
    EXPECT_EQ(errors[0]["path"], "channels.0.pdo");
    EXPECT_EQ(
        errors[0]["message"],
        "'Analog Input Ch1' matches no input entry on slave Test Slave. Pick an "
        "entry."
    );
}

TEST_F(EtherCATReadTest, ParseConfigRejectsTwoChannelsOnOneEntry) {
    const auto ch1 = create_data_channel("ch1", x::telem::INT16_T);
    const auto ch2 = create_data_channel("ch2", x::telem::INT16_T);

    auto cfg = create_base_config();
    cfg["channels"].push_back(create_input_channel_config(ch1, 0x6000, 1, "int16"));
    cfg["channels"].push_back(create_input_channel_config(ch2, 0x6000, 1, "int16"));

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    const auto errors = parser.error_json()["errors"];
    ASSERT_EQ(errors.size(), 1);
    EXPECT_EQ(errors[0]["path"], "channels.1.index");
    EXPECT_EQ(
        errors[0]["message"],
        "another channel already uses entry 0x6000:01 on slave Test Slave"
    );
}

TEST_F(EtherCATReadTest, ParseConfigAllowsADisabledChannelOnAUsedEntry) {
    const auto ch1 = create_data_channel("ch1", x::telem::INT16_T);
    const auto ch2 = create_data_channel("ch2", x::telem::INT16_T);

    auto cfg = create_base_config();
    cfg["channels"].push_back(create_input_channel_config(ch1, 0x6000, 1, "int16"));
    cfg["channels"].push_back(
        create_input_channel_config(ch2, 0x6000, 1, "int16", true)
    );

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(task_cfg.channels.size(), 1);
}

TEST_F(EtherCATReadTest, ParseConfigReachesEveryEntryOfASlaveThatRepeatsOneName) {
    x::json::json inputs = x::json::json::array();
    for (int i = 0; i < 8; i++)
        inputs.push_back(
            {{"name", "Input"},
             {"pdo_name", "Channel " + std::to_string(i + 1)},
             {"pdo_index", 0x1A00 + i},
             {"index", 0x6000 + 0x10 * i},
             {"sub_index", 1},
             {"bit_length", 1},
             {"data_type", "uint8"}}
        );
    slave_device = create_slave_device(
        SLAVE_SERIAL + 1,
        inputs,
        x::json::json::array(),
        1
    );

    auto cfg = create_base_config();
    for (int i = 0; i < 8; i++) {
        const auto ch = create_data_channel("input", x::telem::UINT8_T);
        cfg["channels"].push_back(
            create_input_channel_config(ch, 0x6000 + 0x10 * i, 1, "uint8")
        );
    }

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(task_cfg.channels.size(), 8);
    for (int i = 0; i < 8; i++) {
        EXPECT_EQ(task_cfg.channels[i]->slave_position, 1);
        EXPECT_EQ(task_cfg.channels[i]->index, 0x6000 + 0x10 * i);
        EXPECT_EQ(task_cfg.channels[i]->sub_index, 1);
        EXPECT_EQ(task_cfg.channels[i]->bit_length, 1);
    }
}

TEST_F(EtherCATReadTest, ParseConfigReportsAnUnknownChannelType) {
    const auto data_ch = create_data_channel("analog", x::telem::INT16_T);

    auto cfg = create_base_config();
    cfg["channels"].push_back(
        {{"type", "automatic"},
         {"device", slave_device.key},
         {"pdo", "status_word"},
         {"channel", data_ch.key},
         {"disabled", false}}
    );

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    const auto errors = parser.error_json()["errors"];
    ASSERT_EQ(errors.size(), 1);
    EXPECT_EQ(errors[0]["path"], "channels.0.type");
    EXPECT_EQ(errors[0]["message"], "unknown ReadChannel type: automatic");
}

TEST_F(EtherCATReadTest, ParseConfigRejectsADataTypeWiderThanTheEntry) {
    const auto data_ch = create_data_channel("analog", x::telem::FLOAT64_T);

    auto cfg = create_base_config();
    cfg["channels"].push_back(
        create_input_channel_config(data_ch, 0x6000, 1, "float64")
    );

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    const auto errors = parser.error_json()["errors"];
    ASSERT_EQ(errors.size(), 1);
    EXPECT_EQ(errors[0]["path"], "channels.0.data_type");
    EXPECT_EQ(
        errors[0]["message"],
        "data type float64 is 8 bytes but entry 0x6000:01 on slave Test Slave holds 2 "
        "bytes. Pick a 2 byte type."
    );
}

TEST_F(EtherCATReadTest, ParseConfigRejectsAScaledIntegerChannel) {
    const auto data_ch = create_data_channel("analog", x::telem::INT16_T);

    auto cfg = create_base_config();
    auto ch = create_input_channel_config(data_ch, 0x6000, 1, "int16");
    ch["scale"] = {{"type", "linear"}, {"slope", 2}, {"offset", 1}};
    cfg["channels"].push_back(ch);

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    const auto errors = parser.error_json()["errors"];
    ASSERT_EQ(errors.size(), 1);
    EXPECT_EQ(errors[0]["path"], "channels");
    EXPECT_EQ(
        errors[0]["message"],
        "channel " + data_ch.name +
            " is int16, but a scaled channel must be float32 or float64. Pick a float "
            "channel."
    );
}

TEST_F(EtherCATReadTest, WriterConfigIncludesAllChannels) {
    auto ch1 = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("ch1"),
        x::telem::INT16_T,
        index_channel.key,
        false
    ));

    auto cfg = create_base_config();
    cfg["channels"].push_back(create_input_channel_config(ch1, 0x6000, 1, "int16"));

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());

    auto writer_cfg = task_cfg.writer_config();
    EXPECT_EQ(writer_cfg.channels.size(), 2);
}

TEST_F(EtherCATReadTest, SourceStartRegistersWithEngine) {
    auto data_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("analog"),
        x::telem::INT16_T,
        index_channel.key,
        false
    ));

    auto cfg = create_base_config();
    cfg["channels"].push_back(create_input_channel_config(data_ch, 0x6000, 1, "int16"));

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());

    auto source = ReadTaskSource(engine, std::move(task_cfg));
    ASSERT_NIL(source.start());
    EXPECT_TRUE(engine->running());
    ASSERT_NIL(source.stop());
}

TEST_F(EtherCATReadTest, InvalidSlaveDevice) {
    auto data_ch = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("analog"),
        x::telem::INT16_T,
        this->index_channel.key,
        false
    ));

    x::json::json cfg = {
        {"data_saving_disabled", true},
        {"sample_rate", 100},
        {"stream_rate", 10},
        {"channels",
         {{{"type", "entry"},
           {"device", "nonexistent_device_key"},
           {"index", 0x6000},
           {"sub_index", 1},
           {"data_type", "int16"},
           {"channel", data_ch.key},
           {"disabled", false}}}}
    };

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(this->client, parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
}

TEST_F(EtherCATReadTest, StreamRateBelowOneHzIsValid) {
    auto data_ch = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("analog"),
        x::telem::INT16_T,
        this->index_channel.key,
        false
    ));

    auto cfg = this->create_base_config();
    cfg["sample_rate"] = 2;
    cfg["stream_rate"] = 0.5;
    cfg["channels"].push_back(
        this->create_input_channel_config(data_ch, 0x6000, 1, "int16")
    );

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(this->client, parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(task_cfg.samples_per_chan, 4);
}

TEST_F(EtherCATReadTest, StreamRateThatDoesNotDivideSampleRateIsInvalid) {
    auto cfg = this->create_base_config();
    cfg["sample_rate"] = 10;
    cfg["stream_rate"] = 3;

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(this->client, parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    ASSERT_EQ(parser.errors->size(), 1);
    EXPECT_EQ(parser.errors->at(0)["path"], "stream_rate");
    EXPECT_EQ(
        parser.errors->at(0)["message"],
        "sample_rate must be divisible by stream_rate"
    );
}

TEST_F(EtherCATReadTest, SourceReadsDataFromEngine) {
    auto data_ch = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("analog"),
        x::telem::INT16_T,
        this->index_channel.key,
        false
    ));

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(create_input_channel_config(data_ch, 0x6000, 1, "int16"));

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(this->client, parser);
    ASSERT_NIL(parser.error());

    auto source = ReadTaskSource(this->engine, std::move(task_cfg));
    ASSERT_NIL(source.start());

    this->mock_master->set_input<int16_t>(0, 0x1234);

    x::breaker::Breaker brk;
    brk.start();

    x::telem::Frame frame;
    auto result = source.read(brk, frame);
    ASSERT_NIL(result.error);
    EXPECT_FALSE(frame.empty());

    brk.stop();
    ASSERT_NIL(source.stop());
}

TEST_F(EtherCATReadTest, SourceReadsCorrectValueFromEngine) {
    auto data_ch = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("analog"),
        x::telem::INT16_T,
        this->index_channel.key,
        false
    ));

    auto cfg = this->create_base_config();
    cfg["sample_rate"] = 10;
    cfg["stream_rate"] = 10;
    cfg["channels"].push_back(create_input_channel_config(data_ch, 0x6000, 1, "int16"));

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(this->client, parser);
    ASSERT_NIL(parser.error());

    auto source = ReadTaskSource(this->engine, std::move(task_cfg));
    ASSERT_NIL(source.start());

    this->mock_master->set_input<int16_t>(0, 0x5678);

    x::breaker::Breaker brk;
    brk.start();

    ASSERT_EVENTUALLY_EQ(
        [&] {
            x::telem::Frame frame;
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

TEST_F(EtherCATReadTest, SourceAppliesALinearScale) {
    const auto source = start_scaled_source(
        {{"type", "linear"}, {"slope", 2.5}, {"offset", -10}}
    );
    this->mock_master->set_input<int16_t>(0, -40);
    x::breaker::Breaker brk;
    brk.start();
    ASSERT_EVENTUALLY_EQ(
        [&] {
            x::telem::Frame frame;
            source->read(brk, frame);
            if (frame.empty()) return 0.0;
            return frame.series->at(0).at<double>(0);
        }(),
        -110.0
    );
    brk.stop();
    ASSERT_NIL(source->stop());
}

TEST_F(EtherCATReadTest, SourceAppliesAMapScale) {
    const auto source = start_scaled_source(
        {{"type", "map"},
         {"pre_scaled_min", 0},
         {"pre_scaled_max", 32000},
         {"scaled_min", 0},
         {"scaled_max", 10}}
    );
    this->mock_master->set_input<int16_t>(0, 8000);
    x::breaker::Breaker brk;
    brk.start();
    ASSERT_EVENTUALLY_EQ(
        [&] {
            x::telem::Frame frame;
            source->read(brk, frame);
            if (frame.empty()) return 0.0;
            return frame.series->at(0).at<double>(0);
        }(),
        2.5
    );
    brk.stop();
    ASSERT_NIL(source->stop());
}

TEST_F(EtherCATReadTest, SourceReadsMultipleChannelValues) {
    auto ch1 = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("status"),
        x::telem::INT16_T,
        this->index_channel.key,
        false
    ));
    auto ch2 = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("sensor"),
        x::telem::INT32_T,
        this->index_channel.key,
        false
    ));

    auto cfg = this->create_base_config();
    cfg["sample_rate"] = 10;
    cfg["stream_rate"] = 10;
    cfg["channels"].push_back(create_input_channel_config(ch1, 0x6000, 1, "int16"));
    cfg["channels"].push_back(create_input_channel_config(ch2, 0x6000, 2, "int32"));

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(this->client, parser);
    ASSERT_NIL(parser.error());

    auto source = ReadTaskSource(this->engine, std::move(task_cfg));
    ASSERT_NIL(source.start());

    this->mock_master->set_input<int16_t>(0, static_cast<int16_t>(0xABCD));
    this->mock_master->set_input<int32_t>(2, 0x12345678);

    x::breaker::Breaker brk;
    brk.start();

    int16_t status_value = 0;
    int32_t sensor_value = 0;
    ASSERT_EVENTUALLY_EQ(
        [&] {
            x::telem::Frame frame;
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
        x::telem::INT16_T,
        this->index_channel.key,
        false
    ));

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(create_input_channel_config(data_ch, 0x6000, 1, "int16"));

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(this->client, parser);
    ASSERT_NIL(parser.error());

    auto source = ReadTaskSource(this->engine, std::move(task_cfg));
    ASSERT_NIL(source.start());

    x::breaker::Breaker brk;

    x::telem::Frame frame;
    auto result = source.read(brk, frame);
    ASSERT_NIL(result.error);
    EXPECT_TRUE(frame.empty());

    ASSERT_NIL(source.stop());
}

TEST_F(EtherCATReadTest, SourceStartFailsOnTopologyMismatch) {
    auto data_ch = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("analog"),
        x::telem::INT16_T,
        this->index_channel.key,
        false
    ));

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(create_input_channel_config(data_ch, 0x6000, 1, "int16"));

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(this->client, parser);
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

    auto source = ReadTaskSource(mismatched_engine, std::move(task_cfg));
    ASSERT_OCCURRED_AS(source.start(), errors::TOPOLOGY_MISMATCH);
}

TEST_F(EtherCATReadTest, SourceStartFailsWhenTheBusLacksAnEntry) {
    const auto data_ch = create_data_channel("sensor", x::telem::INT32_T);

    auto cfg = create_base_config();
    cfg["channels"].push_back(create_input_channel_config(data_ch, 0x6000, 2, "int32"));

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());

    auto remapped_master = std::make_shared<mock::Master>(NETWORK_INTERFACE);
    remapped_master->add_slave(
        slave::Properties{
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
                 .data_type = x::telem::INT16_T},
            },
        }
    );
    auto remapped_engine = std::make_shared<engine::Engine>(remapped_master);

    auto source = ReadTaskSource(remapped_engine, std::move(task_cfg));
    const auto err = source.start();
    ASSERT_OCCURRED_AS(err, errors::PDO_MAPPING_ERROR);
    EXPECT_EQ(err.data, "PDO 0x6000:02 is not in the process image of slave 0");
    EXPECT_FALSE(remapped_engine->running());
}

TEST_F(EtherCATReadTest, SourceStartFailsWhenAnEntryChangedBitLength) {
    const auto data_ch = create_data_channel("sensor", x::telem::INT32_T);

    auto cfg = create_base_config();
    cfg["channels"].push_back(create_input_channel_config(data_ch, 0x6000, 2, "int32"));

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());

    auto remapped_master = std::make_shared<mock::Master>(NETWORK_INTERFACE);
    remapped_master->add_slave(
        slave::Properties{
            .position = 0,
            .vendor_id = 0x1,
            .product_code = 0x2,
            .serial = SLAVE_SERIAL,
            .name = "Test Slave",
            .input_pdos = {
                {.pdo_index = 0x1A00,
                 .index = 0x6000,
                 .sub_index = 2,
                 .bit_length = 16,
                 .is_input = true,
                 .name = "sensor_value",
                 .data_type = x::telem::INT16_T},
            },
        }
    );
    auto remapped_engine = std::make_shared<engine::Engine>(remapped_master);

    auto source = ReadTaskSource(remapped_engine, std::move(task_cfg));
    const auto err = source.start();
    ASSERT_OCCURRED_AS(err, errors::PDO_MAPPING_ERROR);
    EXPECT_EQ(
        err.data,
        "PDO 0x6000:02 on slave 0 is 16 bits, not 32. Scan the network again"
    );
    EXPECT_FALSE(remapped_engine->running());
}
}
