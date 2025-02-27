// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// ReSharper disable CppUseStructuredBinding

/// std
#include <utility>

/// external
#include "gtest/gtest.h"

/// module
#include "x/cpp/xjson/xjson.h"
#include "x/cpp/xtest/xtest.h"
#include "client/cpp/testutil/testutil.h"

/// internal
#include "driver/ni/write_task.h"
#include "driver/ni/hardware/hardware.h"
#include "driver/errors/errors.h"
#include "driver/pipeline/mock/pipeline.h"

class SingleChannelAnalogWriteTest : public ::testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> sy;
    synnax::Task task;
    std::unique_ptr<ni::WriteTaskConfig> cfg;
    std::shared_ptr<task::MockContext> ctx;
    std::shared_ptr<pipeline::mock::WriterFactory> mock_writer_factory;
    std::shared_ptr<pipeline::mock::StreamerFactory> mock_streamer_factory;
    synnax::Channel state_idx_ch = synnax::Channel(
        "state_idx_ch",
        telem::TIMESTAMP_T,
        0,
        true
    );
    synnax::Channel state_ch = synnax::Channel(
        "state_ch",
        telem::FLOAT64_T,
        state_idx_ch.key,
        false
    );
    synnax::Channel cmd_ch = synnax::Channel(
        "cmd_ch",
        telem::FLOAT64_T,
        true
    );

    void parse_config() {
        sy = std::make_shared<synnax::Synnax>(new_test_client());

        auto idx_err = sy->channels.create(state_idx_ch);
        ASSERT_FALSE(idx_err) << idx_err;

        state_ch.index = state_idx_ch.key;
        auto data_err = sy->channels.create(state_ch);
        ASSERT_FALSE(data_err) << data_err;

        auto cmd_err = sy->channels.create(cmd_ch);
        ASSERT_FALSE(cmd_err) << cmd_err;

        auto [rack, rack_err] = sy->hardware.create_rack("cat");
        ASSERT_FALSE(rack_err) << rack_err;

        synnax::Device dev(
            "abc123",
            "my_device",
            rack.key,
            "dev1",
            "dev1",
            "ni",
            "PXI-6255",
            ""
        );
        auto dev_err = sy->hardware.create_device(dev);
        ASSERT_FALSE(dev_err) << dev_err;

        task = synnax::Task(
            rack.key,
            "my_task",
            "ni_analog_write",
            ""
        );

        json j{
            {"data_saving", false},
            {"state_rate", 25},
            {"device", dev.key},
            {
                "channels", json::array({
                    {
                        {"type", "ao_voltage"},
                        {"key", "hCzuNC9glqc"},
                        {"port", 0},
                        {"enabled", true},
                        {"min_val", 0},
                        {"max_val", 1},
                        {"state_channel", state_ch.key},
                        {"cmd_channel", cmd_ch.key},
                        {"custom_scale", {{"type", "none"}}},
                        {"units", "Volts"}
                    }
                })
            }
        };

        auto p = xjson::Parser(j);
        cfg = std::make_unique<ni::WriteTaskConfig>(sy, p, "ni_analog_write");
        ASSERT_FALSE(p.error()) << p.error();

        ctx = std::make_shared<task::MockContext>(sy);
        mock_writer_factory = std::make_shared<pipeline::mock::WriterFactory>();
    }

    std::unique_ptr<ni::WriteTask<double>> create_task(
        std::unique_ptr<hardware::mock::Writer<double>> mock_hw
    ) {
        return std::make_unique<ni::WriteTask<double>>(
            task,
            ctx,
            std::move(*cfg),
            breaker::default_config(task.name),
            std::move(mock_hw),
            mock_writer_factory,
            mock_streamer_factory
        );
    }
};

TEST_F(SingleChannelAnalogWriteTest, testBasicAnalogWrite) {
    parse_config();
    auto reads = std::make_shared<std::vector<synnax::Frame>>();
    reads->emplace_back(cmd_ch.key, telem::Series(1, telem::UINT8_T));
    mock_streamer_factory =
            pipeline::mock::simple_streamer_factory({cmd_ch.key}, reads);
    auto wt = create_task(std::make_unique<hardware::mock::Writer<double>>());

    wt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->states.size(), 1);
    const auto first_state = ctx->states[0];
    EXPECT_EQ(first_state.key, "start_cmd");
    EXPECT_EQ(first_state.task, task.key);
    EXPECT_EQ(first_state.variant, "success");
    EXPECT_EQ(first_state.details["message"], "Task started successfully");
    ASSERT_EVENTUALLY_GE(mock_writer_factory->writer_opens, 1);
    ASSERT_EVENTUALLY_GE(mock_streamer_factory->streamer_opens, 1);


    wt->stop("stop_cmd", false);
    ASSERT_EQ(ctx->states.size(), 2);
    const auto second_state = ctx->states[1];
    EXPECT_EQ(second_state.key, "stop_cmd");
    EXPECT_EQ(second_state.task, task.key);
    EXPECT_EQ(second_state.variant, "success");
    EXPECT_EQ(second_state.details["message"], "Task stopped successfully");
}
