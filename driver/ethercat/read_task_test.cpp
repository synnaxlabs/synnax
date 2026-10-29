// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/ethercat/ethercat.h"
#include "driver/ethercat/read_task.h"
#include "driver/ethercat/testutil/master.h"
#include "driver/pipeline/testutil/pipeline.h"
#include "engine/engine.h"

namespace driver::ethercat {
class EtherCATReadTest : public ::testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> client;
    std::shared_ptr<task::MockContext> ctx;
    std::shared_ptr<testutil::Master> mock_master;
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

        mock_master = std::make_shared<testutil::Master>(NETWORK_INTERFACE);
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

/// @brief it should parse a read channel bound to an entry address.
TEST_F(EtherCATReadTest, ParseConfigWithEntryChannel) {
    const auto data_ch = create_data_channel("analog", x::telem::INT16_T);

    auto cfg = this->create_base_config();
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

/// @brief it should take the bit length of a read channel from the slave.
TEST_F(EtherCATReadTest, ParseConfigTakesBitLengthFromTheSlave) {
    const auto data_ch = create_data_channel("analog", x::telem::FLOAT32_T);

    auto cfg = this->create_base_config();
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

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(create_input_channel_config(ch1, 0x6000, 1, "int16"));
    cfg["channels"].push_back(create_input_channel_config(ch2, 0x6000, 2, "int32"));

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(task_cfg.channels.size(), 2);
}

TEST_F(EtherCATReadTest, ParseConfigWithInvalidChannel) {
    auto cfg = this->create_base_config();
    synnax::channel::Channel invalid_ch;
    invalid_ch.key = 99999;
    cfg["channels"].push_back(
        create_input_channel_config(invalid_ch, 0x6000, 1, "int16")
    );

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
}

/// @brief it should report a read channel at an address the slave does not have.
TEST_F(EtherCATReadTest, ParseConfigWithAddressMissingFromSlave) {
    const auto data_ch = create_data_channel("analog", x::telem::INT32_T);

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(create_input_channel_config(data_ch, 0x6000, 3, "int32"));

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    const auto errors = parser.error_json()["errors"];
    ASSERT_EQ(errors.size(), 1);
    EXPECT_EQ(errors[0]["path"], "channels.0.index");
    EXPECT_EQ(errors[0]["message"], "slave Test Slave has no input entry at 0x6000:03");
}

/// @brief it should report a read channel whose name was not resolved to an address.
TEST_F(EtherCATReadTest, ParseConfigWithUnresolvedChannel) {
    const auto data_ch = create_data_channel("analog", x::telem::INT16_T);

    auto cfg = this->create_base_config();
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

/// @brief it should report a second read channel on an entry another channel already
/// uses.
TEST_F(EtherCATReadTest, ParseConfigRejectsTwoChannelsOnOneEntry) {
    const auto ch1 = create_data_channel("ch1", x::telem::INT16_T);
    const auto ch2 = create_data_channel("ch2", x::telem::INT16_T);

    auto cfg = this->create_base_config();
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

/// @brief it should allow a disabled read channel on an entry another channel uses.
TEST_F(EtherCATReadTest, ParseConfigAllowsADisabledChannelOnAUsedEntry) {
    const auto ch1 = create_data_channel("ch1", x::telem::INT16_T);
    const auto ch2 = create_data_channel("ch2", x::telem::INT16_T);

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(create_input_channel_config(ch1, 0x6000, 1, "int16"));
    cfg["channels"].push_back(
        create_input_channel_config(ch2, 0x6000, 1, "int16", true)
    );

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(task_cfg.channels.size(), 1);
}

/// @brief it should bind a read channel to each entry of a slave whose entries all have
/// the same name.
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

    auto cfg = this->create_base_config();
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

/// @brief it should report a read channel with an unknown type.
TEST_F(EtherCATReadTest, ParseConfigReportsAnUnknownChannelType) {
    const auto data_ch = create_data_channel("analog", x::telem::INT16_T);

    auto cfg = this->create_base_config();
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

/// @brief it should report a read channel whose data type is wider than its entry.
TEST_F(EtherCATReadTest, ParseConfigRejectsADataTypeWiderThanTheEntry) {
    const auto data_ch = create_data_channel("analog", x::telem::FLOAT64_T);

    auto cfg = this->create_base_config();
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

TEST_F(EtherCATReadTest, WriterConfigIncludesAllChannels) {
    auto ch1 = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("ch1"),
        x::telem::INT16_T,
        index_channel.key,
        false
    ));

    auto cfg = this->create_base_config();
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

    auto cfg = this->create_base_config();
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

/// @brief it should accept a stream rate below 1 Hz and read the matching number of
/// samples for each stream.
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

/// @brief it should accept a 0.1 Hz stream rate, despite the float precision of a rate.
TEST_F(EtherCATReadTest, StreamRateOfATenthHzIsValid) {
    auto data_ch = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("analog"),
        x::telem::INT16_T,
        this->index_channel.key,
        false
    ));

    auto cfg = this->create_base_config();
    cfg["sample_rate"] = 100;
    cfg["stream_rate"] = 0.1;
    cfg["channels"].push_back(
        this->create_input_channel_config(data_ch, 0x6000, 1, "int16")
    );

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(this->client, parser);
    ASSERT_NIL(parser.error());
    EXPECT_EQ(task_cfg.samples_per_chan, 1000);
}

/// @brief it should reject a stream rate that does not divide the sample rate.
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

/// @brief it should report samples that failed bus cycles dropped as a warning with a
/// message and a description.
TEST_F(EtherCATReadTest, SourceReportsDroppedSamplesAsAWarning) {
    const auto data_ch = this->create_data_channel("analog", x::telem::INT16_T);
    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        this->create_input_channel_config(data_ch, 0x6000, 1, "int16")
    );
    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(this->client, parser);
    ASSERT_NIL(parser.error());
    auto source = ReadTaskSource(this->engine, std::move(task_cfg));
    ASSERT_NIL(source.start());

    const auto receives = this->mock_master->call_count("receive");
    this->mock_master->inject_receive_error(
        x::errors::Error(errors::WORKING_COUNTER_ERROR, "expected 3, got 1")
    );
    ASSERT_EVENTUALLY_GE(this->mock_master->call_count("receive"), receives + 3);
    this->mock_master->clear_injected_errors();

    x::breaker::Breaker brk;
    brk.start();
    x::telem::Frame frame;
    const auto result = source.read(brk, frame);
    ASSERT_NIL(result.error);
    EXPECT_EQ(result.warning.message, "Dropped samples");
    EXPECT_EQ(result.warning.description, "Bus cycles failed: expected 3, got 1.");
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
            if (frame.empty() || frame.series->empty()) return static_cast<int16_t>(0);
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

/// @brief it should retry a read task without a limit while every bus cycle fails,
/// because the bus recovers a lost device without a limit.
TEST_F(EtherCATReadTest, ReadTaskRetriesFailedCyclesWithoutALimit) {
    const auto data_ch = this->create_data_channel("analog", x::telem::INT16_T);
    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        this->create_input_channel_config(data_ch, 0x6000, 1, "int16")
    );
    auto manager = std::make_unique<testutil::Manager>();
    manager->configure(NETWORK_INTERFACE, this->mock_master);
    Factory factory(std::move(manager));
    const synnax::task::Task task{
        .rack = this->rack.key,
        .name = "read",
        .type = READ_TASK_TYPE,
        .config = cfg,
    };
    auto [read_task, ok] = factory.configure_task(this->ctx, task, "");
    ASSERT_TRUE(ok);
    ASSERT_NE(read_task, nullptr);
    this->mock_master->inject_receive_error(
        x::errors::Error(errors::WORKING_COUNTER_ERROR, "expected 3, got 1")
    );
    synnax::task::Command start{.task = task.key, .type = "start"};
    read_task->exec(start);
    // The first read fails after 400 ms, which is 40 cycles at 100 Hz.
    ASSERT_EVENTUALLY_GE_WITH_TIMEOUT(
        this->mock_master->call_count("receive"),
        60,
        std::chrono::seconds(5),
        std::chrono::milliseconds(10)
    );
    synnax::task::Command stop{.task = task.key, .type = "stop"};
    read_task->exec(stop);
    const auto warning = std::ranges::find(
        this->ctx->statuses,
        std::string(synnax::status::VARIANT_WARNING),
        &synnax::task::Status::variant
    );
    ASSERT_NE(warning, this->ctx->statuses.end());
    EXPECT_EQ(warning->message, "Failed to read data");
    EXPECT_EQ(
        warning->description,
        "every bus cycle failed: expected 3, got 1 Next retry 1/∞ in 1.0 s."
    );
}

/// @brief it should pass the startup parameters saved for each slave on the task's
/// network to the master, including a slave the task does not read, and ignore an
/// invalid slave on another network.
TEST_F(EtherCATReadTest, ConfigurePassesSavedSlaveSettingsOnTheNetworkToTheMaster) {
    const auto save_slave =
        [this](const std::string &network, const uint16_t position, const int value) {
            auto dev = this->slave_device;
            dev.key += "_" + network + "_" + std::to_string(position);
            dev.make = DEVICE_MAKE;
            dev.model = SLAVE_DEVICE_MODEL;
            dev.properties["network"] = network;
            dev.properties["position"] = position;
            dev.properties["startup_parameters"] = {
                {{"index", 0x8000},
                 {"sub_index", 0x15},
                 {"data_type", "uint16"},
                 {"value", value}}
            };
            ASSERT_NIL(this->client->devices.create(dev));
        };
    save_slave(NETWORK_INTERFACE, 1, 3);
    save_slave("eth1", 2, 70000);
    const auto data_ch = this->create_data_channel("analog", x::telem::INT16_T);
    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        this->create_input_channel_config(data_ch, 0x6000, 1, "int16")
    );
    auto manager = std::make_unique<testutil::Manager>();
    manager->configure(NETWORK_INTERFACE, this->mock_master);
    Factory factory(std::move(manager));
    auto [read_task, ok] = factory.configure_task(
        this->ctx,
        {.rack = this->rack.key, .name = "read", .type = READ_TASK_TYPE, .config = cfg},
        ""
    );
    ASSERT_TRUE(ok);
    ASSERT_NE(read_task, nullptr);
    EXPECT_EQ(
        this->mock_master->startup_parameters(1),
        (std::vector<slave::StartupParameter>{{
            .index = 0x8000,
            .sub_index = 0x15,
            .data_type = x::telem::UINT16_T,
            .value = 3,
        }})
    );
    EXPECT_TRUE(this->mock_master->startup_parameters(2).empty());
}

/// @brief it should fail to configure, naming the slave, when a slave the task does
/// not read has invalid properties on the task's network.
TEST_F(EtherCATReadTest, ConfigureFailsWhenASlaveOnTheNetworkIsInvalid) {
    auto invalid = this->slave_device;
    invalid.key += "_invalid";
    invalid.name = "EL3102";
    invalid.make = DEVICE_MAKE;
    invalid.model = SLAVE_DEVICE_MODEL;
    invalid.properties["position"] = 1;
    invalid.properties["startup_parameters"] = {
        {{"index", 0x8000}, {"sub_index", 0x15}, {"data_type", "uint8"}, {"value", 300}}
    };
    ASSERT_NIL(this->client->devices.create(invalid));
    const auto data_ch = this->create_data_channel("analog", x::telem::INT16_T);
    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        this->create_input_channel_config(data_ch, 0x6000, 1, "int16")
    );
    auto manager = std::make_unique<testutil::Manager>();
    manager->configure(NETWORK_INTERFACE, this->mock_master);
    Factory factory(std::move(manager));
    auto [read_task, ok] = factory.configure_task(
        this->ctx,
        {.rack = this->rack.key, .name = "read", .type = READ_TASK_TYPE, .config = cfg},
        "start"
    );
    ASSERT_TRUE(ok);
    EXPECT_EQ(read_task, nullptr);
    ASSERT_EQ(this->ctx->statuses.size(), 1);
    const auto &status = this->ctx->statuses[0];
    EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(status.message, "Failed to configure task");
    EXPECT_EQ(
        status.description,
        "EL3102 has invalid properties. Every slave on eth0 starts with the bus, so "
        "fix them before you start the task.\nstartup_parameters.0.value: must be a "
        "whole number from 0 to 255"
    );
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

    auto mismatched_master = std::make_shared<testutil::Master>(NETWORK_INTERFACE);
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

/// @brief it should fail to start the source when the bus no longer has a configured
/// entry.
TEST_F(EtherCATReadTest, SourceStartFailsWhenTheBusLacksAnEntry) {
    const auto data_ch = create_data_channel("sensor", x::telem::INT32_T);

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(create_input_channel_config(data_ch, 0x6000, 2, "int32"));

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());

    auto remapped_master = std::make_shared<testutil::Master>(NETWORK_INTERFACE);
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

/// @brief it should fail to start the source when a configured entry changed its bit
/// length on the bus.
TEST_F(EtherCATReadTest, SourceStartFailsWhenAnEntryChangedBitLength) {
    const auto data_ch = create_data_channel("sensor", x::telem::INT32_T);

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(create_input_channel_config(data_ch, 0x6000, 2, "int32"));

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());

    auto remapped_master = std::make_shared<testutil::Master>(NETWORK_INTERFACE);
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
