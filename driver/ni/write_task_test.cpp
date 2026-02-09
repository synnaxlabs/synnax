// Copyright 2026 Synnax Labs, Inc.
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
#include "x/cpp/json/json.h"
#include "x/cpp/test/test.h"

#include "driver/errors/errors.h"
#include "driver/ni/hardware/hardware.h"
#include "driver/ni/write_task.h"
#include "driver/pipeline/mock/pipeline.h"

namespace driver::ni {
class SingleChannelAnalogWriteTest : public ::testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> client;
    synnax::task::Task task;
    std::unique_ptr<WriteTaskConfig> cfg;
    std::shared_ptr<task::MockContext> ctx;
    std::shared_ptr<pipeline::mock::WriterFactory> mock_writer_factory;
    std::shared_ptr<pipeline::mock::StreamerFactory> mock_streamer_factory;
    synnax::channel::Channel state_idx_ch = synnax::channel::Channel{
        .name = make_unique_channel_name("state_idx_ch"),
        .data_type = x::telem::TIMESTAMP_T,
        .index = 0,
        .is_index = true
    };
    synnax::channel::Channel state_ch_1 = synnax::channel::Channel{
        .name = make_unique_channel_name("state_ch_1"),
        .data_type = x::telem::FLOAT64_T,
        .index = state_idx_ch.key,
        .is_index = false
    };
    synnax::channel::Channel cmd_ch_1 = synnax::channel::Channel{
        .name = make_unique_channel_name("cmd_ch_1"),
        .data_type = x::telem::FLOAT64_T,
        .is_virtual = true
    };
    synnax::channel::Channel state_ch_2 = synnax::channel::Channel{
        .name = make_unique_channel_name("state_ch_2"),
        .data_type = x::telem::FLOAT64_T,
        .index = state_idx_ch.key,
        .is_index = false
    };
    synnax::channel::Channel cmd_ch_2 = synnax::channel::Channel{
        .name = make_unique_channel_name("cmd_ch_2"),
        .data_type = x::telem::FLOAT64_T,
        .is_virtual = true
    };

    void parse_config() {
        client = std::make_shared<synnax::Synnax>(new_test_client());

        ASSERT_NIL(client->channels.create(state_idx_ch));

        state_ch_1.index = state_idx_ch.key;
        state_ch_2.index = state_idx_ch.key;
        ASSERT_NIL(client->channels.create(state_ch_1));
        ASSERT_NIL(client->channels.create(state_ch_2));
        ASSERT_NIL(client->channels.create(cmd_ch_1));
        ASSERT_NIL(client->channels.create(cmd_ch_2));

        const auto rack = ASSERT_NIL_P(client->racks.create("cat"));

        synnax::device::Device
            dev("abc123", "my_device", rack.key, "dev1", "ni", "PXI-6255", "");
        ASSERT_NIL(client->devices.create(dev));

        task = synnax::task::Task{
            .key = synnax::task::create_key(rack.key, 0),
            .name = "my_task",
            .type = "ni_analog_write",
            .config = ""
        };

        const x::json::json j{
            {"data_saving", false},
            {"state_rate", 25},
            {"device", dev.key},
            {"channels",
             x::json::json::array({
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

        auto p = x::json::Parser(j);
        cfg = std::make_unique<WriteTaskConfig>(client, p);
        ASSERT_NIL(p.error());

        ctx = std::make_shared<task::MockContext>(client);
        mock_writer_factory = std::make_shared<pipeline::mock::WriterFactory>();
    }

    std::unique_ptr<common::WriteTask>
    create_task(std::unique_ptr<hardware::mock::Writer<double>> mock_hw) {
        return std::make_unique<common::WriteTask>(
            task,
            ctx,
            x::breaker::default_config(task.name),
            std::make_unique<WriteTaskSink<double>>(
                std::move(*cfg),
                std::move(mock_hw)
            ),
            mock_writer_factory,
            mock_streamer_factory
        );
    }
};

/// @brief it should write analog values and update state channels correctly.
TEST_F(SingleChannelAnalogWriteTest, testBasicAnalogWrite) {
    parse_config();
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    constexpr double v = 1;
    reads->emplace_back(cmd_ch_2.key, x::telem::Series(v, x::telem::FLOAT64_T));
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
    EXPECT_EQ(first_state.variant, x::status::VARIANT_SUCCESS);
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
    EXPECT_EQ(second_state.variant, x::status::VARIANT_SUCCESS);
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
    auto dev = synnax::device::Device(
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
    auto state_idx_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("state_idx"),
        x::telem::TIMESTAMP_T,
        0,
        true
    ));
    auto state_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("state_ch"),
        x::telem::FLOAT64_T,
        state_idx_ch.key,
        false
    ));
    auto cmd_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("cmd_ch"),
        x::telem::FLOAT64_T,
        true
    ));

    // Create a configuration with an invalid channel type
    x::json::json j{
        {"data_saving", false},
        {"state_rate", 25},
        {"device", dev.key},
        {"channels",
         x::json::json::array(
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

    auto p = x::json::Parser(j);
    auto cfg = std::make_unique<WriteTaskConfig>(client, p);

    ASSERT_OCCURRED_AS(p.error(), x::errors::VALIDATION);
}
}
