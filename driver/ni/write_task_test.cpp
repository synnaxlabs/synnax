// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <atomic>
#include <utility>

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/json/json.h"
#include "x/cpp/test/test.h"

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
        .is_index = true,
        .index = 0
    };
    synnax::channel::Channel state_ch_1 = synnax::channel::Channel{
        .name = make_unique_channel_name("state_ch_1"),
        .data_type = x::telem::FLOAT64_T,
        .is_index = false,
        .index = state_idx_ch.key
    };
    synnax::channel::Channel cmd_ch_1 = synnax::channel::Channel{
        .name = make_unique_channel_name("cmd_ch_1"),
        .data_type = x::telem::FLOAT64_T,
        .is_virtual = true
    };
    synnax::channel::Channel state_ch_2 = synnax::channel::Channel{
        .name = make_unique_channel_name("state_ch_2"),
        .data_type = x::telem::FLOAT64_T,
        .is_index = false,
        .index = state_idx_ch.key
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

        auto dev = synnax::device::Device{
            .key = "abc123",
            .rack = rack.key,
            .location = "dev1",
            .make = "ni",
            .model = "PXI-6255",
            .name = "my_device",
        };
        ASSERT_NIL(client->devices.create(dev));

        task = synnax::task::Task{
            .rack = rack.key,
            .name = "my_task",
            .type = "ni_analog_write",
        };

        const x::json::json j{
            {"data_saving_disabled", true},
            {"state_rate", 25},
            {"device", dev.key},
            {"channels",
             x::json::json::array({
                 {{"type", "ao_voltage"},
                  {"key", "hCzuNC9glqc"},
                  {"port", 0},
                  {"disabled", false},
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
                     {"disabled", false},
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
        cfg = std::make_unique<WriteTaskConfig>(client, p, "ni_analog_write");
        ASSERT_NIL(p.error());

        ctx = std::make_shared<driver::task::MockContext>(client);
        mock_writer_factory = std::make_shared<driver::pipeline::mock::WriterFactory>();
    }

    std::shared_ptr<hardware::mock::Writer<double>> mock_hw;
    std::size_t make_hw_calls = 0;

    std::unique_ptr<common::WriteTask>
    create_task(std::unique_ptr<hardware::mock::Writer<double>> hw) {
        this->mock_hw = std::move(hw);
        WriteTaskSink<double>::MakeHardware make_hw = [this](const WriteTaskConfig &)
            -> std::pair<std::unique_ptr<hardware::Writer<double>>, x::errors::Error> {
            ++this->make_hw_calls;
            return {
                std::make_unique<hardware::mock::SharedWriter<double>>(this->mock_hw),
                x::errors::NIL
            };
        };
        return std::make_unique<common::WriteTask>(
            task,
            ctx,
            x::breaker::default_config(task.name),
            std::make_unique<WriteTaskSink<double>>(
                std::move(*cfg),
                std::move(make_hw)
            ),
            mock_writer_factory,
            mock_streamer_factory
        );
    }
};

/// @brief every start should claim fresh hardware and every stop should release
/// it, so a device that disconnects and returns between runs gets a new DAQmx
/// task on the next start.
TEST_F(SingleChannelAnalogWriteTest, testStartClaimsFreshHardware) {
    parse_config();
    mock_streamer_factory = pipeline::mock::simple_streamer_factory(
        {cmd_ch_2.key},
        std::make_shared<std::vector<x::telem::Frame>>()
    );
    auto wt = create_task(std::make_unique<hardware::mock::Writer<double>>());

    wt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    EXPECT_EQ(make_hw_calls, 1);
    // The fixture and the running sink both hold the mock.
    EXPECT_EQ(mock_hw.use_count(), 2);

    wt->stop("stop_cmd", true);
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 2);
    // The stop released the sink's claim on the hardware.
    EXPECT_EQ(mock_hw.use_count(), 1);

    wt->start("start_cmd_2");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 3);
    EXPECT_EQ(make_hw_calls, 2);
    EXPECT_EQ(ctx->statuses[2].variant, synnax::status::VARIANT_SUCCESS);
    wt->stop("stop_cmd_2", true);
}

/// @brief a writer that counts the instances currently alive.
template<typename T>
class CountingWriter final : public hardware::Writer<T> {
    std::shared_ptr<std::atomic<int>> live;

public:
    explicit CountingWriter(std::shared_ptr<std::atomic<int>> live):
        live(std::move(live)) {
        this->live->fetch_add(1);
    }

    ~CountingWriter() { this->live->fetch_sub(1); }

    x::errors::Error start() override { return x::errors::NIL; }
    x::errors::Error stop() override { return x::errors::NIL; }
    x::errors::Error write(const std::vector<T> &) override { return x::errors::NIL; }
};

/// @brief a start must never build hardware over a live claim: DAQmx names each task
/// after the Synnax task and rejects a duplicate name.
TEST_F(SingleChannelAnalogWriteTest, testStartNeverBuildsOverALiveClaim) {
    parse_config();
    const auto live = std::make_shared<std::atomic<int>>(0);
    size_t builds = 0;
    int live_at_build = 0;
    const std::unique_ptr<common::Sink> sink = std::make_unique<WriteTaskSink<double>>(
        std::move(*cfg),
        [&](const WriteTaskConfig &)
            -> std::pair<std::unique_ptr<hardware::Writer<double>>, x::errors::Error> {
            ++builds;
            live_at_build = std::max(live_at_build, live->load());
            return {std::make_unique<CountingWriter<double>>(live), x::errors::NIL};
        }
    );

    ASSERT_NIL(sink->start());
    EXPECT_EQ(live->load(), 1);
    ASSERT_NIL(sink->start());
    EXPECT_EQ(builds, 2);
    EXPECT_EQ(live_at_build, 0);
    EXPECT_EQ(live->load(), 1);
    ASSERT_NIL(sink->stop());
    EXPECT_EQ(live->load(), 0);
}

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
    EXPECT_EQ(first_state.key, synnax::task::status_key(task));
    EXPECT_EQ(first_state.details.cmd, "start_cmd");
    EXPECT_EQ(first_state.details.task, task.key);
    EXPECT_EQ(first_state.variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(first_state.message, "Task started successfully");
    ASSERT_EVENTUALLY_GE(
        mock_writer_factory->writer_opens.load(std::memory_order_acquire),
        1
    );
    ASSERT_EVENTUALLY_GE(
        mock_streamer_factory->streamer_opens.load(std::memory_order_acquire),
        1
    );
    ASSERT_EVENTUALLY_GE(mock_writer_factory->writes->size(), 6);

    wt->stop("stop_cmd", true);
    ASSERT_EQ(ctx->statuses.size(), 2);
    const auto second_state = ctx->statuses[1];
    EXPECT_EQ(second_state.key, synnax::task::status_key(task));
    EXPECT_EQ(second_state.details.cmd, "stop_cmd");
    EXPECT_EQ(second_state.details.task, task.key);
    EXPECT_EQ(second_state.variant, synnax::status::VARIANT_SUCCESS);
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

/// @brief when a frame contains multiple samples for a channel, only the last
/// sample should be written to hardware.
TEST_F(SingleChannelAnalogWriteTest, testLastWriteWins) {
    parse_config();
    auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    x::telem::Frame fr(1);
    fr.emplace(cmd_ch_2.key, x::telem::Series(std::vector<double>{1.0, 2.0, 3.0}));
    reads->push_back(std::move(fr));
    mock_streamer_factory = pipeline::mock::simple_streamer_factory(
        {cmd_ch_2.key},
        reads
    );
    auto written_data = std::make_shared<std::vector<std::vector<double>>>();
    auto wt = create_task(
        std::make_unique<hardware::mock::Writer<double>>(written_data)
    );

    wt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(
        mock_streamer_factory->streamer_opens.load(std::memory_order_acquire),
        1
    );
    ASSERT_EVENTUALLY_GE(written_data->size(), 1);

    auto last_write = written_data->back();
    ASSERT_EQ(last_write.size(), 2);
    ASSERT_EQ(last_write.at(1), 3.0);

    wt->stop("stop_cmd", true);
}

/// @brief Test that an invalid channel type in the configuration is properly detected
/// and reported
TEST(WriteTaskConfigTest, testInvalidChannelType) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("test_rack"));

    // Create a device
    auto dev = synnax::device::Device{
        .key = "abc123",
        .rack = rack.key,
        .location = "dev1",
        .make = "ni",
        .model = "PXI-6255",
        .name = "test_device",
    };
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
        {"data_saving_disabled", true},
        {"state_rate", 25},
        {"device", dev.key},
        {"channels",
         x::json::json::array(
             {{{"type", "INVALID_CHANNEL_TYPE"}, // Invalid channel type
               {"key", "hCzuNC9glqc"},
               {"port", 0},
               {"disabled", false},
               {"min_val", 0},
               {"max_val", 1},
               {"state_channel", state_ch.key},
               {"cmd_channel", cmd_ch.key},
               {"custom_scale", {{"type", "none"}}},
               {"units", "Volts"}}}
         )}
    };

    auto p = x::json::Parser(j);
    auto cfg = std::make_unique<WriteTaskConfig>(client, p, "ni_analog_write");

    ASSERT_OCCURRED_AS(p.error(), x::errors::VALIDATION);
}
}
