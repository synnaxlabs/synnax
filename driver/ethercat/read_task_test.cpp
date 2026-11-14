// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <bit>

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/ethercat/coe/coe.h"
#include "driver/ethercat/ethercat.h"
#include "driver/ethercat/read_task.h"
#include "driver/ethercat/testutil/master.h"
#include "driver/pipeline/testutil/pipeline.h"
#include "engine/engine.h"

namespace driver::ethercat {
class EtherCATReadTest : public ::testing::Test {
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

    /// @brief returns a slave on the network with the given enabled value, ready to
    /// save.
    [[nodiscard]] synnax::device::Device create_configured_slave(
        const std::string &network,
        const uint16_t position,
        const x::json::json &enabled
    ) const {
        auto dev = this->slave_device;
        dev.key += "_" + network + "_" + std::to_string(position);
        dev.make = DEVICE_MAKE;
        dev.model = SLAVE_DEVICE_MODEL;
        dev.properties["network"] = network;
        dev.properties["position"] = position;
        dev.properties["enabled"] = enabled;
        return dev;
    }

    /// @brief sets whether the last scan found the device at key.
    void set_connected(const std::string &key, const bool connected) const {
        const auto dev = ASSERT_NIL_P(
            this->client->devices.retrieve(key, {.include_status = true})
        );
        auto status = dev.status.value();
        status.details.connected = connected;
        ASSERT_NIL(this->client->statuses.set(status));
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

        mock_master = std::make_shared<testutil::Master>();
        mock_master->add_slave(
            slave::Properties{
                .position = 1,
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
        engine = std::make_shared<engine::Engine>(NETWORK_INTERFACE, mock_master);
    }

    synnax::device::Device create_slave_device(
        uint32_t serial,
        const x::json::json &input_pdos,
        const x::json::json &output_pdos
    ) {
        x::json::json props = {
            {"serial", serial},
            {"vendor_id", 0x1},
            {"product_code", 0x2},
            {"revision", 1},
            {"name", "Test Slave"},
            {"network", NETWORK_INTERFACE},
            {"position", 1},
            {"enabled", true},
            {"pdos", {{"inputs", input_pdos}, {"outputs", output_pdos}}}
        };
        synnax::device::Device dev{
            .key = make_unique_channel_name("ecat_slave_" + std::to_string(serial)),
            .rack = rack.key,
            .location = NETWORK_INTERFACE + ".Slot 1",
            .make = "DEWESoft",
            .model = "TestModule",
            .name = "Test Slave SN:" + std::to_string(serial),
            .properties = props,
        };
        auto err = client->devices.create(dev);
        EXPECT_NIL(err);
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
        auto cfg = this->create_base_config();
        cfg["sample_rate"] = 10;
        cfg["stream_rate"] = 10;
        auto ch = create_input_channel_config(data_ch, 0x6000, 1, "int16");
        ch["scale"] = scale;
        cfg["channels"].push_back(ch);
        auto parser = x::json::Parser(cfg);
        ReadTaskConfig task_cfg(client, parser);
        EXPECT_NIL(parser.error());
        auto source = std::make_unique<ReadTaskSource>(engine, std::move(task_cfg));
        EXPECT_NIL(source->start());
        return source;
    }

    /// @brief saves a catalog for the slave that calibrates its int32 entry at
    /// 0x6000:02 with the gain at 0x8040:15 and the offset at 0x8040:14.
    void calibrate_slave() {
        this->slave_device.properties["settings"] = {
            {"channels",
             {{{"entries",
                {{{"index", 0x6000},
                  {"sub_index", 2},
                  {"calibration",
                   {{"data_type", "int32"},
                    {"index", 0x8040},
                    {"gain_sub_index", 0x15},
                    {"offset_sub_index", 0x14},
                    {"divisor", 1}}}}}}}}},
        };
        ASSERT_NIL(this->client->devices.del(this->slave_device.key));
        ASSERT_NIL(this->client->devices.create(this->slave_device));
    }

    /// @brief returns a config that reads the entry at 0x6000:02 into data_ch, with the
    /// calibration of the slave.
    x::json::json create_sensor_config(const synnax::channel::Channel &data_ch) {
        auto cfg = this->create_base_config();
        cfg["sample_rate"] = 10;
        cfg["stream_rate"] = 10;
        auto ch = this->create_input_channel_config(data_ch, 0x6000, 2, "int32");
        ch["calibrated"] = true;
        cfg["channels"].push_back(ch);
        return cfg;
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

    void TearDown() override { this->cancel.stop(); }
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
    EXPECT_EQ(task_cfg.slaves.network, "eth0");
    EXPECT_EQ(task_cfg.channels[0]->entry.index, 0x6000);
    EXPECT_EQ(task_cfg.channels[0]->entry.sub_index, 1);
    EXPECT_EQ(task_cfg.channels[0]->entry.bit_length, 16);
    EXPECT_EQ(task_cfg.channels[0]->entry.data_type, x::telem::INT16_T);
}

/// @brief it should report a read channel that does not exist in the Core.
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
    const auto errors = parser.error_json()["errors"];
    ASSERT_EQ(errors.size(), 1);
    EXPECT_EQ(errors[0]["path"], "channels");
    EXPECT_EQ(errors[0]["message"], "channels matching keys 99999 not found.");
}

/// @brief it should report only the entry error of a read channel whose entry it cannot
/// find.
TEST_F(EtherCATReadTest, ParseConfigReportsAChannelWithNoEntry) {
    const auto ch = create_data_channel("ch", x::telem::INT16_T);
    auto cfg = this->create_base_config();
    cfg["channels"].push_back(create_unresolved_input_channel_config(ch, "Missing"));

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    const auto errors = parser.error_json()["errors"];
    ASSERT_EQ(errors.size(), 1);
    EXPECT_EQ(errors[0]["path"], "channels.0.pdo");
    EXPECT_EQ(
        errors[0]["message"],
        "'Missing' matches no input entry on slave Test Slave. Pick an entry."
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

/// @brief it should start when a disabled channel names a slave that is not on the
/// bus, and enable a channel whose config has no disabled field.
TEST_F(EtherCATReadTest, StartIgnoresTheSlaveOfADisabledChannel) {
    const auto enabled_ch = create_data_channel("enabled", x::telem::INT16_T);
    const auto disabled_ch = create_data_channel("disabled", x::telem::INT16_T);
    auto absent = this->create_configured_slave(NETWORK_INTERFACE, 5, true);
    ASSERT_NIL(this->client->devices.create(absent));

    auto cfg = this->create_base_config();
    auto enabled = create_input_channel_config(enabled_ch, 0x6000, 1, "int16");
    enabled.erase("disabled");
    cfg["channels"].push_back(enabled);
    auto disabled = create_input_channel_config(disabled_ch, 0x6000, 1, "int16", true);
    disabled["device"] = absent.key;
    cfg["channels"].push_back(disabled);

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(task_cfg.channels.size(), 1);
    EXPECT_EQ(task_cfg.channels[0]->synnax_key, enabled_ch.key);
    ReadTaskSource source(engine, std::move(task_cfg));
    ASSERT_NIL(source.start());
    ASSERT_NIL(source.stop());
}

/// @brief it should reject a scaled read channel whose Synnax channel is not a float
/// type.
TEST_F(EtherCATReadTest, ParseConfigRejectsAScaledIntegerChannel) {
    const auto data_ch = create_data_channel("analog", x::telem::INT16_T);

    auto cfg = this->create_base_config();
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

/// @brief it should write the data channels and their index channel.
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

/// @brief it should report a read channel on a slave device that does not exist.
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
    auto second = cfg["channels"][0];
    second["sub_index"] = 2;
    cfg["channels"].push_back(second);

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(this->client, parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    const auto errors = parser.error_json()["errors"];
    ASSERT_EQ(errors.size(), 2);
    EXPECT_EQ(errors[0]["path"], "channels.0.device");
    EXPECT_EQ(errors[1]["path"], "channels.1.device");
    EXPECT_EQ(
        errors[1]["message"],
        R"(Found no device where key = "nonexistent_device_key")"
    );
}

/// @brief it should reject each channel on a slave that is on another network than the
/// slaves before it.
TEST_F(EtherCATReadTest, SlaveOnAnotherNetworkIsInvalid) {
    auto other = this->slave_device;
    other.key += "_eth1";
    other.properties["network"] = "eth1";
    ASSERT_NIL(this->client->devices.create(other));
    const auto data_ch = this->create_data_channel("analog", x::telem::INT16_T);
    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        this->create_input_channel_config(data_ch, 0x6000, 1, "int16")
    );
    auto off_network = this->create_input_channel_config(data_ch, 0x6000, 1, "int16");
    off_network["device"] = other.key;
    cfg["channels"].push_back(off_network);
    off_network["sub_index"] = 2;
    cfg["channels"].push_back(off_network);

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(this->client, parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    const auto errors = parser.error_json()["errors"];
    ASSERT_EQ(errors.size(), 2);
    EXPECT_EQ(errors[0]["path"], "channels.1.device");
    EXPECT_EQ(errors[1]["path"], "channels.2.device");
    EXPECT_EQ(errors[1]["message"], "all slaves must be on the same network");
}

/// @brief it should report a channel on a slave device whose saved properties are not
/// valid.
TEST_F(EtherCATReadTest, ParseConfigRejectsASlaveWithInvalidProperties) {
    auto broken = this->slave_device;
    broken.key = make_unique_channel_name("ecat_slave_broken");
    broken.properties["serial"] = "not a number";
    ASSERT_NIL(this->client->devices.create(broken));
    const auto data_ch = this->create_data_channel("analog", x::telem::INT16_T);
    auto cfg = this->create_base_config();
    auto ch = this->create_input_channel_config(data_ch, 0x6000, 1, "int16");
    ch["device"] = broken.key;
    cfg["channels"].push_back(ch);
    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(this->client, parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    const auto errors = parser.error_json()["errors"];
    ASSERT_EQ(errors.size(), 1);
    EXPECT_EQ(errors[0]["path"], "channels.0.device");
    EXPECT_EQ(errors[0]["message"], "serial: expected a number, got 'not a number'");
}

/// @brief it should reject a task whose channels are all disabled.
TEST_F(EtherCATReadTest, ParseConfigRejectsATaskWithNoEnabledChannel) {
    const auto data_ch = this->create_data_channel("analog", x::telem::INT16_T);
    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        this->create_input_channel_config(data_ch, 0x6000, 1, "int16", true)
    );
    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(this->client, parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    const auto errors = parser.error_json()["errors"];
    ASSERT_EQ(errors.size(), 1);
    EXPECT_EQ(errors[0]["path"], "channels");
    EXPECT_EQ(errors[0]["message"], "task must have at least one enabled channel");
}

/// @brief it should write the cycle time of each sample to the index channel of the
/// frame.
TEST_F(EtherCATReadTest, SourceWritesTheSampleTimesToTheIndexChannel) {
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
    x::breaker::Breaker brk;
    brk.start();
    x::telem::Frame frame;
    ASSERT_NIL(source.read(brk, frame).error);
    const auto index = std::ranges::find(*frame.channels, this->index_channel.key);
    ASSERT_NE(index, frame.channels->end());
    const auto &times = frame.series->at(index - frame.channels->begin());
    ASSERT_EQ(times.size(), 10);
    EXPECT_EQ(frame.series->at(0).size(), 10);
    for (size_t i = 1; i < times.size(); ++i)
        EXPECT_GT(times.at<int64_t>(i), times.at<int64_t>(i - 1));
    brk.stop();
    ASSERT_NIL(source.stop());
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

/// @brief it should skip the map scale check on a disabled channel.
TEST_F(EtherCATReadTest, ParseConfigSkipsTheMapScaleCheckOnADisabledChannel) {
    const auto valid_ch = create_data_channel("valid", x::telem::INT16_T);
    const auto disabled_ch = create_data_channel("disabled", x::telem::FLOAT64_T);

    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        create_input_channel_config(valid_ch, 0x6000, 1, "int16")
    );
    auto disabled = create_input_channel_config(disabled_ch, 0x6000, 2, "int32", true);
    disabled["scale"] = {
        {"type", "map"},
        {"pre_scaled_min", 4},
        {"pre_scaled_max", 4},
        {"scaled_min", 0},
        {"scaled_max", 10},
    };
    cfg["channels"].push_back(disabled);

    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, parser);
    ASSERT_NIL(parser.error());
    ASSERT_EQ(task_cfg.channels.size(), 1);
    EXPECT_EQ(task_cfg.channels[0]->synnax_key, valid_ch.key);
}

/// @brief it should apply a linear scale to each sample it reads.
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

/// @brief it should turn each raw sample into a physical value with the gain and offset
/// that the slave holds.
TEST_F(EtherCATReadTest, SourceAppliesTheCalibrationOfTheSlave) {
    this->calibrate_slave();
    this->mock_master->set_object(1, 0x8040, 0x15, std::bit_cast<uint32_t>(2.0F));
    this->mock_master->set_object(1, 0x8040, 0x14, std::bit_cast<uint32_t>(0.5F));
    const auto data_ch = this->create_data_channel("sensor", x::telem::FLOAT64_T);
    auto parser = x::json::Parser(this->create_sensor_config(data_ch));
    ReadTaskConfig task_cfg(this->client, parser);
    ASSERT_NIL(parser.error());
    auto source = ReadTaskSource(this->engine, std::move(task_cfg));
    ASSERT_NIL(source.start());
    this->mock_master->set_input<int32_t>(2, -100);
    x::breaker::Breaker brk;
    brk.start();
    ASSERT_EVENTUALLY_EQ(
        [&] {
            x::telem::Frame frame;
            source.read(brk, frame);
            if (frame.empty()) return 0.0;
            return frame.series->at(0).at<double>(0);
        }(),
        -199.5
    );
    brk.stop();
    ASSERT_NIL(source.stop());
}

/// @brief it should apply the scale of a calibrated channel to the physical values.
TEST_F(EtherCATReadTest, SourceScalesACalibratedChannelAfterItsCalibration) {
    this->calibrate_slave();
    this->mock_master->set_object(1, 0x8040, 0x15, std::bit_cast<uint32_t>(2.0F));
    this->mock_master->set_object(1, 0x8040, 0x14, std::bit_cast<uint32_t>(0.5F));
    const auto data_ch = this->create_data_channel("sensor", x::telem::FLOAT64_T);
    auto cfg = this->create_sensor_config(data_ch);
    cfg["channels"][0]["scale"] = {{"type", "linear"}, {"slope", 10}, {"offset", 1}};
    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(this->client, parser);
    ASSERT_NIL(parser.error());
    auto source = ReadTaskSource(this->engine, std::move(task_cfg));
    ASSERT_NIL(source.start());
    this->mock_master->set_input<int32_t>(2, -100);
    x::breaker::Breaker brk;
    brk.start();
    ASSERT_EVENTUALLY_EQ(
        [&] {
            x::telem::Frame frame;
            source.read(brk, frame);
            if (frame.empty()) return 0.0;
            return frame.series->at(0).at<double>(0);
        }(),
        -1994.0
    );
    brk.stop();
    ASSERT_NIL(source.stop());
}

/// @brief it should fail to start, naming the channel, when it cannot read the
/// calibration from the slave.
TEST_F(EtherCATReadTest, StartFailsWhenTheCalibrationCannotBeRead) {
    this->calibrate_slave();
    const auto data_ch = this->create_data_channel("sensor", x::telem::FLOAT32_T);
    auto parser = x::json::Parser(this->create_sensor_config(data_ch));
    ReadTaskConfig task_cfg(this->client, parser);
    ASSERT_NIL(parser.error());
    auto source = ReadTaskSource(this->engine, std::move(task_cfg));
    const auto err = source.start();
    ASSERT_OCCURRED_AS(err, coe::OBJECT_MISSING);
    EXPECT_EQ(
        err.data,
        "Failed to read the calibration of channel " + data_ch.name +
            ": failed to read 0x8040:15: Object does not exist in the object "
            "dictionary."
    );
}

/// @brief it should reject a calibrated channel whose Synnax channel is not a float
/// channel.
TEST_F(EtherCATReadTest, ParseConfigRejectsACalibratedIntegerChannel) {
    this->calibrate_slave();
    const auto data_ch = this->create_data_channel("sensor", x::telem::INT32_T);
    auto parser = x::json::Parser(this->create_sensor_config(data_ch));
    ReadTaskConfig task_cfg(this->client, parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    const auto errors = parser.error_json()["errors"];
    ASSERT_EQ(errors.size(), 1);
    EXPECT_EQ(errors[0]["path"], "channels");
    EXPECT_EQ(
        errors[0]["message"],
        "channel " + data_ch.name +
            " is int32, but a calibrated channel must be float32 or float64. Pick a "
            "float channel."
    );
}

/// @brief it should read each channel from its own PDO entry.
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
    Factory factory(std::move(manager), this->client, this->rack.key);
    const synnax::task::Task task{
        .rack = this->rack.key,
        .name = "read",
        .type = READ_TASK_TYPE,
        .config = cfg,
    };
    auto [read_task, ok] = factory.configure_task(this->ctx, task, "", this->cancel);
    ASSERT_TRUE(ok);
    ASSERT_NE(read_task, nullptr);
    this->mock_master->inject_receive_error(
        x::errors::Error(errors::WORKING_COUNTER_ERROR, "expected 3, got 1")
    );
    synnax::task::Command start{.task = task.key, .type = "start"};
    read_task->exec(start, this->cancel);
    // The first read fails after 400 ms, which is 40 cycles at 100 Hz.
    ASSERT_EVENTUALLY_GE_WITH_TIMEOUT(
        this->mock_master->call_count("receive"),
        60,
        std::chrono::seconds(5),
        std::chrono::milliseconds(10)
    );
    synnax::task::Command stop{.task = task.key, .type = "stop"};
    read_task->exec(stop, this->cancel);
    const auto warning = std::ranges::find(
        this->ctx->statuses,
        std::string(synnax::status::VARIANT_WARNING),
        &synnax::task::Status::variant
    );
    ASSERT_NE(warning, this->ctx->statuses.end());
    EXPECT_EQ(warning->message, "Failed to read data");
    EXPECT_EQ(
        warning->description,
        "Every bus cycle failed: expected 3, got 1. Next retry 1/∞ in 1.0 s."
    );
}

/// @brief it should apply the settings saved for each slave on the task's network when
/// the bus starts, including a setting saved after configure and a slave the task does
/// not read, and ignore an invalid slave on another network.
TEST_F(EtherCATReadTest, StartAppliesSavedSlaveSettingsOnTheNetwork) {
    const auto data_ch = this->create_data_channel("analog", x::telem::INT16_T);
    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        this->create_input_channel_config(data_ch, 0x6000, 1, "int16")
    );
    auto manager = std::make_unique<testutil::Manager>();
    manager->configure(NETWORK_INTERFACE, this->mock_master);
    Factory factory(std::move(manager), this->client, this->rack.key);
    const synnax::task::Task task{
        .rack = this->rack.key,
        .name = "read",
        .type = READ_TASK_TYPE,
        .config = cfg,
    };
    auto [read_task, ok] = factory.configure_task(this->ctx, task, "", this->cancel);
    ASSERT_TRUE(ok);
    ASSERT_NE(read_task, nullptr);
    auto on_network = this->create_configured_slave(NETWORK_INTERFACE, 2, false);
    ASSERT_NIL(this->client->devices.create(on_network));
    auto off_network = this->create_configured_slave("eth1", 3, nullptr);
    ASSERT_NIL(this->client->devices.create(off_network));
    synnax::task::Command start{.task = task.key, .type = "start"};
    read_task->exec(start, this->cancel);
    EXPECT_EQ(this->ctx->statuses.back().message, "Started task");
    EXPECT_FALSE(this->mock_master->config(2).enabled);
    EXPECT_TRUE(this->mock_master->config(3).enabled);
    synnax::task::Command stop{.task = task.key, .type = "stop"};
    read_task->exec(stop, this->cancel);
}

/// @brief it should write the startup parameters of each slave to it, in order, when
/// the task starts the bus.
TEST_F(EtherCATReadTest, StartWritesTheStartupParameters) {
    const auto data_ch = this->create_data_channel("analog", x::telem::INT16_T);
    auto cfg = this->create_base_config();
    const auto ch = this->create_input_channel_config(data_ch, 0x6000, 1, "int16");
    cfg["channels"].push_back(ch);
    cfg["startup_parameters"] = {{
        {"device", ch["device"]},
        {"parameters",
         {
             {{"index", 0x8040},
              {"sub_index", 0x1C},
              {"data_type", "float32"},
              {"value", 5}},
             {{"index", 0x8040},
              {"sub_index", 1},
              {"data_type", "uint16"},
              {"value", 6}},
         }},
    }};
    auto manager = std::make_unique<testutil::Manager>();
    manager->configure(NETWORK_INTERFACE, this->mock_master);
    Factory factory(std::move(manager), this->client, this->rack.key);
    const synnax::task::Task task{
        .rack = this->rack.key,
        .name = "read",
        .type = READ_TASK_TYPE,
        .config = cfg,
    };
    auto [read_task, ok] = factory.configure_task(this->ctx, task, "", this->cancel);
    ASSERT_TRUE(ok);
    synnax::task::Command start{.task = task.key, .type = "start"};
    read_task->exec(start, this->cancel);
    EXPECT_EQ(this->ctx->statuses.back().message, "Started task");
    const auto params = this->mock_master->config(1).startup_parameters;
    ASSERT_EQ(params.size(), 2);
    EXPECT_EQ(params[0].sub_index, 0x1C);
    EXPECT_EQ(params[0].data_type, x::telem::FLOAT32_T);
    EXPECT_EQ(params[0].value, 5);
    EXPECT_EQ(params[1].sub_index, 1);
    EXPECT_EQ(params[1].value, 6);
    synnax::task::Command stop{.task = task.key, .type = "stop"};
    read_task->exec(stop, this->cancel);
}

/// @brief it should reject a startup parameter whose value does not fit its data type.
TEST_F(EtherCATReadTest, ParseConfigRejectsAParameterThatDoesNotFitItsType) {
    const auto data_ch = this->create_data_channel("analog", x::telem::INT16_T);
    auto cfg = this->create_base_config();
    const auto ch = this->create_input_channel_config(data_ch, 0x6000, 1, "int16");
    cfg["channels"].push_back(ch);
    cfg["startup_parameters"] = {{
        {"device", ch["device"]},
        {"parameters",
         {{{"index", 0x8040},
           {"sub_index", 1},
           {"data_type", "uint16"},
           {"value", -1}}}},
    }};
    auto parser = x::json::Parser(cfg);
    ReadTaskConfig task_cfg(client, parser);
    ASSERT_OCCURRED_AS(parser.error(), x::errors::VALIDATION);
    const auto errors = parser.error_json()["errors"];
    ASSERT_EQ(errors.size(), 1);
    EXPECT_EQ(errors[0]["path"], "startup_parameters.0.parameters.0.value");
    EXPECT_EQ(errors[0]["message"], "must be a whole number from 0 to 65535");
}

/// @brief it should run a slave in the mode chosen on its saved device when the bus
/// starts.
TEST_F(EtherCATReadTest, StartRunsTheModeChosenOnASavedDevice) {
    const dc::Choice choice{.index = 1, .assign_activate = 0x0700};
    auto saved = this->create_configured_slave(NETWORK_INTERFACE, 1, true);
    saved.properties["op_mode"] = choice.to_json();
    ASSERT_NIL(this->client->devices.create(saved));
    const auto data_ch = this->create_data_channel("analog", x::telem::INT16_T);
    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        this->create_input_channel_config(data_ch, 0x6000, 1, "int16")
    );
    auto manager = std::make_unique<testutil::Manager>();
    manager->configure(NETWORK_INTERFACE, this->mock_master);
    Factory factory(std::move(manager), this->client, this->rack.key);
    const synnax::task::Task task{
        .rack = this->rack.key,
        .name = "read",
        .type = READ_TASK_TYPE,
        .config = cfg,
    };
    auto [read_task, ok] = factory.configure_task(this->ctx, task, "", this->cancel);
    ASSERT_TRUE(ok);
    ASSERT_NE(read_task, nullptr);
    synnax::task::Command start{.task = task.key, .type = "start"};
    read_task->exec(start, this->cancel);
    EXPECT_EQ(this->ctx->statuses.back().message, "Started task");
    EXPECT_EQ(this->mock_master->config(1).op_mode, choice);
    synnax::task::Command stop{.task = task.key, .type = "stop"};
    read_task->exec(stop, this->cancel);
}

/// @brief it should pass the startup parameters saved on a device to the master when
/// the bus starts.
TEST_F(EtherCATReadTest, StartPassesTheStartupParametersOfASavedDevice) {
    auto saved = this->create_configured_slave(NETWORK_INTERFACE, 1, true);
    saved.properties["startup_parameters"] = {
        {{"index", 0x8000}, {"sub_index", 0x15}, {"data_type", "uint16"}, {"value", 3}}
    };
    ASSERT_NIL(this->client->devices.create(saved));
    const auto data_ch = this->create_data_channel("analog", x::telem::INT16_T);
    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        this->create_input_channel_config(data_ch, 0x6000, 1, "int16")
    );
    auto manager = std::make_unique<testutil::Manager>();
    manager->configure(NETWORK_INTERFACE, this->mock_master);
    Factory factory(std::move(manager), this->client, this->rack.key);
    const synnax::task::Task task{
        .rack = this->rack.key,
        .name = "read",
        .type = READ_TASK_TYPE,
        .config = cfg,
    };
    auto [read_task, ok] = factory.configure_task(this->ctx, task, "", this->cancel);
    ASSERT_TRUE(ok);
    ASSERT_NE(read_task, nullptr);
    synnax::task::Command start{.task = task.key, .type = "start"};
    read_task->exec(start, this->cancel);
    EXPECT_EQ(this->ctx->statuses.back().message, "Started task");
    EXPECT_EQ(
        this->mock_master->config(1).startup_parameters,
        (std::vector<slave::StartupParameter>{{
            .index = 0x8000,
            .sub_index = 0x15,
            .data_type = x::telem::UINT16_T,
            .value = 3,
        }})
    );
    synnax::task::Command stop{.task = task.key, .type = "stop"};
    read_task->exec(stop, this->cancel);
}

/// @brief it should drop the settings of a deleted device at the next bus start.
TEST_F(EtherCATReadTest, StartDropsTheSettingsOfADeletedDevice) {
    auto saved = this->create_configured_slave(NETWORK_INTERFACE, 1, false);
    ASSERT_NIL(this->client->devices.create(saved));
    const auto data_ch = this->create_data_channel("analog", x::telem::INT16_T);
    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        this->create_input_channel_config(data_ch, 0x6000, 1, "int16")
    );
    auto manager = std::make_unique<testutil::Manager>();
    manager->configure(NETWORK_INTERFACE, this->mock_master);
    Factory factory(std::move(manager), this->client, this->rack.key);
    const synnax::task::Task task{
        .rack = this->rack.key,
        .name = "read",
        .type = READ_TASK_TYPE,
        .config = cfg,
    };
    auto [read_task, ok] = factory.configure_task(this->ctx, task, "", this->cancel);
    ASSERT_TRUE(ok);
    ASSERT_NE(read_task, nullptr);
    synnax::task::Command start{.task = task.key, .type = "start"};
    synnax::task::Command stop{.task = task.key, .type = "stop"};
    read_task->exec(start, this->cancel);
    ASSERT_FALSE(this->mock_master->config(1).enabled);
    read_task->exec(stop, this->cancel);
    ASSERT_NIL(this->client->devices.del(saved.key));
    read_task->exec(start, this->cancel);
    EXPECT_EQ(this->ctx->statuses.back().message, "Started task");
    EXPECT_TRUE(this->mock_master->config(1).enabled);
    read_task->exec(stop, this->cancel);
}

/// @brief it should fail to start, naming the slave and the network by the name of its
/// master, when a slave the task does not read has invalid properties on the task's
/// network.
TEST_F(EtherCATReadTest, StartFailsWhenASlaveOnTheNetworkIsInvalid) {
    this->mock_master->set_name("Ethernet 2");
    auto invalid = this->create_configured_slave(NETWORK_INTERFACE, 1, nullptr);
    invalid.name = "EL3102";
    ASSERT_NIL(this->client->devices.create(invalid));
    const auto data_ch = this->create_data_channel("analog", x::telem::INT16_T);
    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        this->create_input_channel_config(data_ch, 0x6000, 1, "int16")
    );
    auto manager = std::make_unique<testutil::Manager>();
    manager->configure(NETWORK_INTERFACE, this->mock_master);
    Factory factory(std::move(manager), this->client, this->rack.key);
    const synnax::task::Task task{
        .rack = this->rack.key,
        .name = "read",
        .type = READ_TASK_TYPE,
        .config = cfg,
    };
    auto [read_task, ok] = factory.configure_task(this->ctx, task, "", this->cancel);
    ASSERT_TRUE(ok);
    ASSERT_NE(read_task, nullptr);
    synnax::task::Command start{.task = task.key, .type = "start"};
    read_task->exec(start, this->cancel);
    const auto &status = this->ctx->statuses.back();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(status.message, "Failed to start task");
    EXPECT_EQ(
        status.description,
        "EL3102 has invalid properties. Every slave on Ethernet 2 starts with the bus, "
        "so fix them before you start the task.\nenabled: type must be boolean, but is "
        "null"
    );
}

/// @brief it should fail to start, naming the network, when the Driver cannot read the
/// slave settings from the Core.
TEST_F(EtherCATReadTest, StartFailsWhenTheSlaveSettingsCannotBeRead) {
    auto client_cfg = test_client_config();
    client_cfg.password = "not-the-password";
    auto unauthorized = std::make_shared<synnax::Synnax>(client_cfg);
    const auto data_ch = this->create_data_channel("analog", x::telem::INT16_T);
    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        this->create_input_channel_config(data_ch, 0x6000, 1, "int16")
    );
    auto manager = std::make_unique<testutil::Manager>();
    manager->configure(NETWORK_INTERFACE, this->mock_master);
    Factory factory(std::move(manager), std::move(unauthorized), this->rack.key);
    const synnax::task::Task task{
        .rack = this->rack.key,
        .name = "read",
        .type = READ_TASK_TYPE,
        .config = cfg,
    };
    auto [read_task, ok] = factory.configure_task(this->ctx, task, "", this->cancel);
    ASSERT_TRUE(ok);
    ASSERT_NE(read_task, nullptr);
    synnax::task::Command start{.task = task.key, .type = "start"};
    read_task->exec(start, this->cancel);
    const auto &status = this->ctx->statuses.back();
    EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(status.message, "Failed to start task");
    EXPECT_EQ(
        status.description,
        "The Driver could not read the slave settings for eth0 from the Core: invalid "
        "credentials: auth error"
    );
}

/// @brief it should start with the connected device when a disconnected device is saved
/// at its position with different settings.
TEST_F(EtherCATReadTest, StartSkipsADisconnectedDeviceAtAPosition) {
    auto removed = this->create_configured_slave(NETWORK_INTERFACE, 2, false);
    removed.name = "EL3102";
    ASSERT_NIL(this->client->devices.create(removed));
    this->set_connected(removed.key, false);
    auto replacement = this->create_configured_slave(NETWORK_INTERFACE, 2, true);
    replacement.key += "_replacement";
    replacement.name = "EL3102";
    replacement.properties["serial"] = 54321;
    ASSERT_NIL(this->client->devices.create(replacement));
    this->set_connected(replacement.key, true);
    const auto data_ch = this->create_data_channel("analog", x::telem::INT16_T);
    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        this->create_input_channel_config(data_ch, 0x6000, 1, "int16")
    );
    auto manager = std::make_unique<testutil::Manager>();
    manager->configure(NETWORK_INTERFACE, this->mock_master);
    Factory factory(std::move(manager), this->client, this->rack.key);
    const synnax::task::Task task{
        .rack = this->rack.key,
        .name = "read",
        .type = READ_TASK_TYPE,
        .config = cfg,
    };
    auto [read_task, ok] = factory.configure_task(this->ctx, task, "", this->cancel);
    ASSERT_TRUE(ok);
    ASSERT_NE(read_task, nullptr);
    synnax::task::Command start{.task = task.key, .type = "start"};
    read_task->exec(start, this->cancel);
    EXPECT_EQ(this->ctx->statuses.back().message, "Started task");
    EXPECT_TRUE(this->mock_master->config(2).enabled);
    synnax::task::Command stop{.task = task.key, .type = "stop"};
    read_task->exec(stop, this->cancel);
}

/// @brief it should fail to start, naming both devices, when two devices at one
/// position have different settings, and start once their settings match.
TEST_F(EtherCATReadTest, StartFailsWhenDevicesAtAPositionDisagree) {
    auto removed = this->create_configured_slave(NETWORK_INTERFACE, 2, false);
    removed.name = "EL3102";
    ASSERT_NIL(this->client->devices.create(removed));
    auto replacement = this->create_configured_slave(NETWORK_INTERFACE, 2, true);
    replacement.key += "_replacement";
    replacement.name = "EL3102";
    replacement.properties["serial"] = 54321;
    ASSERT_NIL(this->client->devices.create(replacement));
    const auto data_ch = this->create_data_channel("analog", x::telem::INT16_T);
    auto cfg = this->create_base_config();
    cfg["channels"].push_back(
        this->create_input_channel_config(data_ch, 0x6000, 1, "int16")
    );
    auto manager = std::make_unique<testutil::Manager>();
    manager->configure(NETWORK_INTERFACE, this->mock_master);
    Factory factory(std::move(manager), this->client, this->rack.key);
    const synnax::task::Task task{
        .rack = this->rack.key,
        .name = "read",
        .type = READ_TASK_TYPE,
        .config = cfg,
    };
    auto [read_task, ok] = factory.configure_task(this->ctx, task, "", this->cancel);
    ASSERT_TRUE(ok);
    ASSERT_NE(read_task, nullptr);
    synnax::task::Command start{.task = task.key, .type = "start"};
    read_task->exec(start, this->cancel);
    EXPECT_EQ(this->ctx->statuses.back().message, "Failed to start task");
    EXPECT_EQ(
        this->ctx->statuses.back().description,
        "EL3102 (serial 12345) and EL3102 (serial 54321) are both saved at position 2 "
        "on eth0 with different settings. Give them the same settings, or delete the "
        "device that is no longer on the bus."
    );
    ASSERT_NIL(this->client->devices.dispatch(
        replacement.key,
        {synnax::device::set({.path = {"properties", "enabled"}, .value = false})}
    ));
    read_task->exec(start, this->cancel);
    EXPECT_EQ(this->ctx->statuses.back().message, "Started task");
    EXPECT_FALSE(this->mock_master->config(2).enabled);
    synnax::task::Command stop{.task = task.key, .type = "stop"};
    read_task->exec(stop, this->cancel);
}

/// @brief it should return an empty frame when the breaker is stopped.
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

/// @brief it should refuse to start when a slave on the bus is not the device that
/// the task names.
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

    auto mismatched_master = std::make_shared<testutil::Master>();
    mismatched_master->add_slave(
        slave::Properties{
            .position = 1,
            .vendor_id = 0x99,
            .product_code = 0x2,
            .serial = SLAVE_SERIAL,
            .name = "Test Slave",
        }
    );
    auto mismatched_engine = std::make_shared<engine::Engine>(
        NETWORK_INTERFACE,
        mismatched_master
    );

    auto source = ReadTaskSource(mismatched_engine, std::move(task_cfg));
    const auto err = source.start();
    ASSERT_OCCURRED_AS(err, errors::TOPOLOGY_MISMATCH);
    EXPECT_EQ(
        err.data,
        "Slave 1 (Test Slave): expected vendor_id 0x00000001, found 0x00000099."
    );
}

}
