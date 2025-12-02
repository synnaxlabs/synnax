// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <utility>

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/xjson/xjson.h"
#include "x/cpp/xtest/xtest.h"

#include "driver/errors/errors.h"
#include "driver/ni/hardware/hardware.h"
#include "driver/ni/write_task.h"
#include "driver/pipeline/mock/pipeline.h"

class SingleChannelAnalogWriteTest : public ::testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> client;
    synnax::Task task;
    std::unique_ptr<ni::WriteTaskConfig> cfg;
    std::shared_ptr<task::MockContext> ctx;
    std::shared_ptr<pipeline::mock::WriterFactory> mock_writer_factory;
    std::shared_ptr<pipeline::mock::StreamerFactory> mock_streamer_factory;
    synnax::Channel state_idx_ch = synnax::Channel(
        make_unique_channel_name("state_idx_ch"),
        telem::TIMESTAMP_T,
        0,
        true
    );
    synnax::Channel state_ch_1 = synnax::Channel(
        make_unique_channel_name("state_ch_1"),
        telem::FLOAT64_T,
        state_idx_ch.key,
        false
    );
    synnax::Channel cmd_ch_1 = synnax::Channel(
        make_unique_channel_name("cmd_ch_1"),
        telem::FLOAT64_T,
        true
    );
    synnax::Channel state_ch_2 = synnax::Channel(
        make_unique_channel_name("state_ch_2"),
        telem::FLOAT64_T,
        state_idx_ch.key,
        false
    );
    synnax::Channel cmd_ch_2 = synnax::Channel(
        make_unique_channel_name("cmd_ch_2"),
        telem::FLOAT64_T,
        true
    );

    void parse_config() {
        client = std::make_shared<synnax::Synnax>(new_test_client());

        auto idx_err = client->channels.create(state_idx_ch);
        ASSERT_FALSE(idx_err) << idx_err;

        state_ch_1.index = state_idx_ch.key;
        state_ch_2.index = state_idx_ch.key;
        auto data_err = client->channels.create(state_ch_1);
        ASSERT_FALSE(data_err) << data_err;
        data_err = client->channels.create(state_ch_2);
        ASSERT_FALSE(data_err) << data_err;
        auto cmd_err = client->channels.create(cmd_ch_1);
        ASSERT_FALSE(cmd_err) << cmd_err;
        cmd_err = client->channels.create(cmd_ch_2);

        auto [rack, rack_err] = client->racks.create("cat");
        ASSERT_FALSE(rack_err) << rack_err;

        synnax::Device
            dev("abc123", "my_device", rack.key, "dev1", "ni", "PXI-6255", "");
        auto dev_err = client->devices.create(dev);
        ASSERT_FALSE(dev_err) << dev_err;

        task = synnax::Task(rack.key, "my_task", "ni_analog_write", "");

        json j{
            {"data_saving", false},
            {"state_rate", 25},
            {"device", dev.key},
            {"channels",
             json::array({
                 {{"type", "ao_voltage"},
                  {"key", "hCzuNC9glqc"},
                  {"port", 0},
                  {"enabled", true},
                  {"min_val", 0},
                  {"max_val", 1},
                  {"state_channel", state_ch_1.key},
                  {"cmd_channel", cmd_ch_1.key},
                  {"custom_scale", {{"type", "none"}}},
                  {"units", "Volts"}},
                 {

                     {"type", "ao_voltage"},
                     {"key", "hCzuNC9glqc"},
                     {"port", 1},
                     {"enabled", true},
                     {"min_val", 0},
                     {"max_val", 1},
                     {"state_channel", state_ch_2.key},
                     {"cmd_channel", cmd_ch_2.key},
                     {"custom_scale", {{"type", "none"}}},
                     {"units", "Volts"}
                 },
             })}
        };

        auto p = xjson::Parser(j);
        cfg = std::make_unique<ni::WriteTaskConfig>(client, p);
        ASSERT_FALSE(p.error()) << p.error();

        ctx = std::make_shared<task::MockContext>(client);
        mock_writer_factory = std::make_shared<pipeline::mock::WriterFactory>();
    }

    std::unique_ptr<common::WriteTask>
    create_task(std::unique_ptr<hardware::mock::Writer<double>> mock_hw) {
        return std::make_unique<common::WriteTask>(
            task,
            ctx,
            breaker::default_config(task.name),
            std::make_unique<ni::WriteTaskSink<double>>(
                std::move(*cfg),
                std::move(mock_hw)
            ),
            mock_writer_factory,
            mock_streamer_factory
        );
    }
};

TEST_F(SingleChannelAnalogWriteTest, testBasicAnalogWrite) {
    parse_config();
    auto reads = std::make_shared<std::vector<synnax::Frame>>();
    constexpr double v = 1;
    reads->emplace_back(cmd_ch_2.key, telem::Series(v, telem::FLOAT64_T));
    mock_streamer_factory = pipeline::mock::simple_streamer_factory(
        {cmd_ch_2.key},
        reads
    );
    auto written_data = std::make_shared<std::vector<std::vector<double>>>();
    auto wt = create_task(
        std::make_unique<hardware::mock::Writer<double>>(written_data)
    );

    wt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    const auto first_state = ctx->statuses[0];
    EXPECT_EQ(first_state.key, task.status_key());
    EXPECT_EQ(first_state.details.cmd, "start_cmd");
    EXPECT_EQ(first_state.details.task, task.key);
    EXPECT_EQ(first_state.variant, status::variant::SUCCESS);
    EXPECT_EQ(first_state.message, "Task started successfully");
    ASSERT_EVENTUALLY_GE(mock_writer_factory->writer_opens, 1);
    ASSERT_EVENTUALLY_GE(mock_streamer_factory->streamer_opens, 1);
    ASSERT_EVENTUALLY_GE(mock_writer_factory->writes->size(), 6);

    wt->stop("stop_cmd", true);
    ASSERT_EQ(ctx->statuses.size(), 2);
    const auto second_state = ctx->statuses[1];
    EXPECT_EQ(second_state.key, task.status_key());
    EXPECT_EQ(second_state.details.cmd, "stop_cmd");
    EXPECT_EQ(second_state.details.task, task.key);
    EXPECT_EQ(second_state.variant, status::variant::SUCCESS);
    ASSERT_EQ(second_state.message, "Task stopped successfully");

    auto first = std::move(
        mock_writer_factory->writes->at(mock_writer_factory->writes->size() - 1)
    );
    ASSERT_EQ(first.size(), 3);
    ASSERT_EQ(first.length(), 1);
    ASSERT_TRUE(first.contains(state_ch_1.key));
    ASSERT_TRUE(first.contains(state_ch_2.key));
    ASSERT_TRUE(first.contains(state_idx_ch.key));
    ASSERT_EQ(first.at<double>(state_ch_1.key, 0), 0);
    ASSERT_EQ(first.at<double>(state_ch_2.key, 0), 1);

    ASSERT_EQ(written_data->size(), 1);
    ASSERT_EQ(written_data->at(0).size(), 2);
    ASSERT_EQ(written_data->at(0).at(0), 0);
    ASSERT_EQ(written_data->at(0).at(1), 1);
}

/// @brief Test that an invalid channel type in the configuration is properly detected
/// and reported
TEST(WriteTaskConfigTest, testInvalidChannelType) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("test_rack"));

    // Create a device
    auto dev = synnax::Device(
        "abc123",
        "test_device",
        rack.key,
        "dev1",
        "ni",
        "PXI-6255",
        ""
    );
    ASSERT_NIL(client->devices.create(dev));

    // Create state and command channels
    auto state_idx_ch = ASSERT_NIL_P(
        client->channels
            .create(make_unique_channel_name("state_idx"), telem::TIMESTAMP_T, 0, true)
    );
    auto state_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("state_ch"),
        telem::FLOAT64_T,
        state_idx_ch.key,
        false
    ));
    auto cmd_ch = ASSERT_NIL_P(
        client->channels.create(make_unique_channel_name("cmd_ch"), telem::FLOAT64_T, true)
    );

    // Create a configuration with an invalid channel type
    json j{
        {"data_saving", false},
        {"state_rate", 25},
        {"device", dev.key},
        {"channels",
         json::array(
             {{{"type", "INVALID_CHANNEL_TYPE"}, // Invalid channel type
               {"key", "hCzuNC9glqc"},
               {"port", 0},
               {"enabled", true},
               {"min_val", 0},
               {"max_val", 1},
               {"state_channel", state_ch.key},
               {"cmd_channel", cmd_ch.key},
               {"custom_scale", {{"type", "none"}}},
               {"units", "Volts"}}}
         )}
    };

    auto p = xjson::Parser(j);
    auto cfg = std::make_unique<ni::WriteTaskConfig>(client, p);

    ASSERT_OCCURRED_AS(p.error(), xerrors::VALIDATION);
}
