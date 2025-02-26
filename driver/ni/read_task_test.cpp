// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// ReSharper disable CppUseStructuredBinding
#include "gtest/gtest.h"

#include "x/cpp/xjson/xjson.h"
#include "driver/ni/read_task.h"

#include "client/cpp/testutil/testutil.h"
#include "driver/errors/errors.h"
#include "driver/pipeline/mock/pipeline.h"
#include "x/cpp/xtest/xtest.h"


template<typename T>
class MockHardwareInterface final : public ni::HardwareInterface<T> {
public:
    explicit MockHardwareInterface(
        const std::vector<xerrors::Error> &start_errors = {xerrors::NIL},
        const std::vector<xerrors::Error> &stop_errors = {xerrors::NIL},
        std::vector<std::pair<std::vector<T>, xerrors::Error>> read_responses = {{{0.5}, xerrors::NIL}}
    ) : start_errors(start_errors),
        stop_errors(stop_errors),
        read_responses(read_responses),
        start_cal_count(0),
        stop_call_count(0),
        read_call_count(0) {}

    xerrors::Error start() override {
        auto err = start_errors[std::min(start_cal_count, start_errors.size() - 1)];
        start_cal_count++;
        return err;
    }

    xerrors::Error stop() override {
        auto err = stop_errors[std::min(stop_call_count, stop_errors.size() - 1)];
        stop_call_count++;
        return err;
    }

    std::pair<size_t, xerrors::Error> read(
        size_t samples_per_channel,
        std::vector<T> &data
    ) override {
        auto response = read_responses[std::min(read_call_count, read_responses.size() - 1)];
        read_call_count++;
        if (!response.first.empty())
            std::copy(response.first.begin(), response.first.end(), data.begin());
        return {response.first.size(), response.second};
    }

private:
    std::vector<xerrors::Error> start_errors;
    std::vector<xerrors::Error> stop_errors;
    std::vector<std::pair<std::vector<T>, xerrors::Error>> read_responses;
    size_t start_cal_count;
    size_t stop_call_count;
    size_t read_call_count;
};

class SingleChannelReadTask : public ::testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> sy;
    synnax::Task task;
    std::unique_ptr<ni::ReadTaskConfig> cfg;
    std::shared_ptr<task::MockContext> ctx;
    std::shared_ptr<pipeline::mock::WriterFactory> mock_factory;
    synnax::Channel index_channel = synnax::Channel(
        "time_channel",
        telem::TIMESTAMP_T,
        0,
        true
    );
    synnax::Channel data_channel = synnax::Channel(
        "data_channel",
        telem::FLOAT64_T,
        index_channel.key,
        false
    );

    void parseConfig() {
        sy = std::make_shared<synnax::Synnax>(new_test_client());
        
        auto idx_err = sy->channels.create(index_channel);
        ASSERT_FALSE(idx_err) << idx_err;

        data_channel.index = index_channel.key;
        auto data_err= sy->channels.create(data_channel);
        ASSERT_FALSE(data_err) << data_err;

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
            "ni_analog_read",
            ""
        );

        json j{
            {"data_saving", false},
            {"sample_rate", 25},
            {"stream_rate", 25},
            {"channels", json::array({
                {
                    {"type", "ai_accel"},
                    {"key", "ks1VnWdrSVA"},
                    {"port", 0},
                    {"enabled", true},
                    {"name", ""},
                    {"channel", data_channel.key},
                    {"terminal_config", "Cfg_Default"},
                    {"min_val", 0},
                    {"max_val", 1},
                    {"sensitivity", 0},
                    {"current_excit_source", "Internal"},
                    {"current_excit_val", 0},
                    {"custom_scale", {{"type", "none"}}},
                    {"units", "g"},
                    {"sensitivity_units", "mVoltsPerG"},
                    {"device", dev.key}
                }
            })}
        };

        auto p = xjson::Parser(j);
        cfg = std::make_unique<ni::ReadTaskConfig>(sy, p, "ni_analog_read");
        ASSERT_FALSE(p.error()) << p.error();

        auto client = std::make_shared<synnax::Synnax>(new_test_client());
        ctx = std::make_shared<task::MockContext>(client);
        mock_factory = std::make_shared<pipeline::mock::WriterFactory>();
    }

    std::unique_ptr<ni::ReadTask<double>> createReadTask(
        std::unique_ptr<MockHardwareInterface<double>> mock_hw
    ) {
        return std::make_unique<ni::ReadTask<double>>(
            task,
            ctx,
            std::move(*cfg),
            breaker::default_config(task.name),
            std::move(mock_hw),
            mock_factory
        );
    }
};

TEST_F(SingleChannelReadTask, testBasicAnalogRead) {
    parseConfig();
    auto rt = createReadTask(std::make_unique<MockHardwareInterface<double>>());
    
    rt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->states.size(), 1);
    const auto first_state = ctx->states[0];
    EXPECT_EQ(first_state.key, "start_cmd");
    EXPECT_EQ(first_state.task, task.key);
    EXPECT_EQ(first_state.variant, "success");
    EXPECT_EQ(first_state.details["message"], "Task started successfully");

    rt->stop("stop_cmd");
    ASSERT_EVENTUALLY_GE(ctx->states.size(), 2);
    const auto second_state = ctx->states[1];
    EXPECT_EQ(second_state.key, "stop_cmd");
    EXPECT_EQ(second_state.task, task.key);
    EXPECT_EQ(second_state.variant, "success");
    EXPECT_EQ(second_state.details["message"], "Task stopped successfully");
    ASSERT_EQ(mock_factory->writer_opens, 1);
    ASSERT_GE(mock_factory->writes->size(), 1);
    auto &fr = mock_factory->writes->at(0);
    ASSERT_EQ(fr.size(), 2);
    ASSERT_EQ(fr.length(), 1);
    ASSERT_EQ(fr.contains(data_channel.key), true);
    ASSERT_EQ(fr.contains(index_channel.key), true);
    ASSERT_EQ(fr.at<double>(data_channel.key, 0), 0.5);
    ASSERT_GE(fr.at<uint64_t>(index_channel.key, 0), 0);
}

TEST_F(SingleChannelReadTask, testErrorOnStart) {
    parseConfig();
    const auto rt = createReadTask(std::make_unique<MockHardwareInterface<double>>(
        std::vector{xerrors::Error(driver::CRITICAL_HARDWARE_ERROR, "Failed to start hardware")}
    ));
    rt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->states.size(), 1);
    const auto state = ctx->states[0];
    EXPECT_EQ(state.key, "start_cmd");
    EXPECT_EQ(state.task, task.key);
    EXPECT_EQ(state.variant, "error");
    EXPECT_EQ(state.details["message"], "[sy.driver.hardware.critical] Failed to start hardware");
    rt->stop();
}

TEST_F(SingleChannelReadTask, testErrorOnStop) {
    parseConfig();
    auto rt = createReadTask(std::make_unique<MockHardwareInterface<double>>(
        std::vector{xerrors::NIL},
        std::vector{xerrors::Error(driver::CRITICAL_HARDWARE_ERROR, "Failed to stop hardware")}
    ));
    rt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->states.size(), 1);
    const auto start_state = ctx->states[0];
    EXPECT_EQ(start_state.variant, "success");
    rt->stop("stop_cmd");
    ASSERT_EVENTUALLY_GE(ctx->states.size(), 2);
    const auto stop_state = ctx->states[1];
    EXPECT_EQ(stop_state.key, "stop_cmd");
    EXPECT_EQ(stop_state.task, task.key);
    EXPECT_EQ(stop_state.variant, "error");
    EXPECT_EQ(stop_state.details["message"], "[sy.driver.hardware.critical] Failed to stop hardware");
}

/// make data channel float32 and then assert that the output data is actually float32
TEST_F(SingleChannelReadTask, testDataTypeCoersion) {
    // Override the default float64 data channel with a float32 channel
    data_channel.data_type = telem::FLOAT32_T;
    parseConfig();

    auto rt = createReadTask(std::make_unique<MockHardwareInterface<double>>(
        std::vector{xerrors::NIL},
        std::vector{xerrors::NIL},
        std::vector<std::pair<std::vector<double>, xerrors::Error>>{{{1.23456789}, xerrors::NIL}}
    ));
    
    rt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->states.size(), 1);
    const auto start_state = ctx->states[0];
    EXPECT_EQ(start_state.variant, "success");

    rt->stop("stop_cmd");
    ASSERT_EVENTUALLY_GE(ctx->states.size(), 2);
    const auto stop_state = ctx->states[1];
    EXPECT_EQ(stop_state.variant, "success");
    
    ASSERT_EQ(mock_factory->writer_opens, 1);
    ASSERT_GE(mock_factory->writes->size(), 1);
    
    auto &fr = mock_factory->writes->at(0);
    ASSERT_EQ(fr.size(), 2);
    ASSERT_EQ(fr.length(), 1);
    ASSERT_EQ(fr.contains(data_channel.key), true);
    
    // Verify that the data was properly coerced to float32
    // The value should be stored as float32 and show some precision loss
    auto value = fr.at<float>(data_channel.key, 0);
    ASSERT_EQ(sizeof(value), sizeof(float));  // Verify it's actually float32
    EXPECT_FLOAT_EQ(value, 1.23456789f);     // Should match float32 precision
    
    // Optional: Verify that the original double value and the float value are different
    // due to precision loss
    EXPECT_NE(static_cast<double>(value), 1.23456789);
}