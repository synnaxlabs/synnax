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
#include "driver/ni/read_task.h"
#include "driver/ni/hardware/hardware.h"
#include "driver/errors/errors.h"
#include "driver/pipeline/mock/pipeline.h"

/// @brief it should correctly parse a basic analog read task.
namespace {
json base_analog_config() {
    return {
        {"data_saving", false},
        {"sample_rate", 25},
        {"stream_rate", 25},
        {
            "channels", json::array({
                {
                    {"type", "ai_accel"},
                    {"key", "ks1VnWdrSVA"},
                    {"port", 0},
                    {"enabled", true},
                    {"name", ""},
                    {"channel", ""}, // Will be overridden
                    {"terminal_config", "Cfg_Default"},
                    {"min_val", 0},
                    {"max_val", 1},
                    {"sensitivity", 0},
                    {"current_excit_source", "Internal"},
                    {"current_excit_val", 0},
                    {"custom_scale", {{"type", "none"}}},
                    {"units", "g"},
                    {"sensitivity_units", "mVoltsPerG"},
                    {"device", ""} // Will be overridden
                }
            })
        }
    };
}
}

TEST(ReadTaskConfigTest, testBasicAnalogReadTaskConfigParse) {
    auto sy = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(sy->hardware.create_rack("cat"));
    auto dev = synnax::Device(
        "abc123",
        "my_device",
        rack.key,
        "dev1",
        "dev1",
        "ni",
        "PXI-6255",
        ""
    );
    auto ch = ASSERT_NIL_P(sy->channels.create("virtual",telem::FLOAT64_T,true));

    auto j = base_analog_config();
    j["channels"][0]["device"] = dev.key;
    j["channels"][0]["channel"] = ch.key;

    auto p = xjson::Parser(j);
    auto cfg = std::make_unique<ni::ReadTaskConfig>(sy, p, "ni_analog_read");
    ASSERT_NIL(p.error());
}

/// @brief it should return a validation error if the device does not exist.
TEST(ReadTaskConfigTest, testNonExistingAnalogReadDevice) {
    auto sy = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(sy->hardware.create_rack("cat"));
    auto ch = ASSERT_NIL_P(sy->channels.create("virtual",telem::FLOAT64_T,true));

    auto j = base_analog_config();
    j["channels"][0]["device"] = "definitely_not_an_existing_device";
    j["channels"][0]["channel"] = ch.key;

    auto p = xjson::Parser(j);
    auto cfg = std::make_unique<ni::ReadTaskConfig>(sy, p, "ni_analog_read");
    ASSERT_OCCURRED_AS(p.error(), xerrors::VALIDATION);
}

/// @brief it should return a validation error if the channel does not exist.
TEST(ReadTaskConfigTest, testNonExistentAnalogReadChannel) {
    auto sy = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(sy->hardware.create_rack("cat"));
    auto dev = synnax::Device(
        "abc123",
        "my_device",
        rack.key,
        "dev1",
        "dev1",
        "ni",
        "PXI-6255",
        ""
    );
    ASSERT_NIL(sy->hardware.create_device(dev));

    auto j = base_analog_config();
    j["channels"][0]["device"] = dev.key;
    j["channels"][0]["channel"] = 12121212;

    auto p = xjson::Parser(j);
    auto cfg = std::make_unique<ni::ReadTaskConfig>(sy, p, "ni_analog_read");
    ASSERT_OCCURRED_AS(p.error(), xerrors::VALIDATION);
}

/// @brief it should return a validation error if the sample rate is less than the stream rate.
TEST(ReadTaskConfigTest, testSampleRateLessThanStreamRate) {
    auto sy = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(sy->hardware.create_rack("cat"));
    auto dev = synnax::Device(
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

    auto ch = ASSERT_NIL_P(sy->channels.create("virtual", telem::FLOAT64_T, true));

    auto j = base_analog_config();
    j["channels"][0]["device"] = dev.key;
    j["channels"][0]["channel"] = ch.key;
    j["sample_rate"] = 10;

    auto p = xjson::Parser(j);
    auto cfg = std::make_unique<ni::ReadTaskConfig>(sy, p, "ni_analog_read");
    ASSERT_OCCURRED_AS(p.error(), xerrors::VALIDATION);
}

/// @brief it should return a validation error if no channels in the task are enabled.
TEST(ReadTaskConfigTest, testNoEnabledChannels) {}

class AnalogReadTest : public ::testing::Test {
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

    void parse_config() {
        sy = std::make_shared<synnax::Synnax>(new_test_client());

        auto idx_err = sy->channels.create(index_channel);
        ASSERT_FALSE(idx_err) << idx_err;

        data_channel.index = index_channel.key;
        auto data_err = sy->channels.create(data_channel);
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
            {
                "channels", json::array({
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
                })
            }
        };

        auto p = xjson::Parser(j);
        cfg = std::make_unique<ni::ReadTaskConfig>(sy, p, "ni_analog_read");
        ASSERT_FALSE(p.error()) << p.error();

        ctx = std::make_shared<task::MockContext>(sy);
        mock_factory = std::make_shared<pipeline::mock::WriterFactory>();
    }

    std::unique_ptr<ni::ReadTask<double>> create_task(
        std::unique_ptr<hardware::mock::Reader<double>> mock_hw
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

/// @brief it should run a basic analog read task using a mock hardware implementation.
TEST_F(AnalogReadTest, testBasicAnalogRead) {
    parse_config();
    auto rt = create_task(std::make_unique<hardware::mock::Reader<double>>());

    rt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->states.size(), 1);
    const auto first_state = ctx->states[0];
    EXPECT_EQ(first_state.key, "start_cmd");
    EXPECT_EQ(first_state.task, task.key);
    EXPECT_EQ(first_state.variant, "success");
    EXPECT_EQ(first_state.details["message"], "Task started successfully");
    ASSERT_EVENTUALLY_GE(mock_factory->writer_opens, 1);
    rt->stop("stop_cmd", false);
    ASSERT_EQ(ctx->states.size(), 2);
    const auto second_state = ctx->states[1];
    EXPECT_EQ(second_state.key, "stop_cmd");
    EXPECT_EQ(second_state.task, task.key);
    EXPECT_EQ(second_state.variant, "success");
    EXPECT_EQ(second_state.details["message"], "Task stopped successfully");
    ASSERT_GE(mock_factory->writes->size(), 1);
    auto &fr = mock_factory->writes->at(0);
    ASSERT_EQ(fr.size(), 2);
    ASSERT_EQ(fr.length(), 1);
    ASSERT_EQ(fr.contains(data_channel.key), true);
    ASSERT_EQ(fr.contains(index_channel.key), true);
    ASSERT_EQ(fr.at<double>(data_channel.key, 0), 0.5);
    ASSERT_GE(fr.at<uint64_t>(index_channel.key, 0), 0);
}

/// @breif it should communicate an error when the hardware fails to start.
TEST_F(AnalogReadTest, testErrorOnStart) {
    parse_config();
    const auto rt = create_task(std::make_unique<hardware::mock::Reader<double>>(
        std::vector{
            xerrors::Error(driver::CRITICAL_HARDWARE_ERROR, "Failed to start hardware")
        }
    ));
    rt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->states.size(), 1);
    const auto state = ctx->states[0];
    EXPECT_EQ(state.key, "start_cmd");
    EXPECT_EQ(state.task, task.key);
    EXPECT_EQ(state.variant, "error");
    EXPECT_EQ(state.details["message"],
              "[sy.driver.hardware.critical] Failed to start hardware");
    rt->stop(false);
}

/// @brief it should communicate an error when the hardware fails to stop.
TEST_F(AnalogReadTest, testErrorOnStop) {
    parse_config();
    auto rt = create_task(std::make_unique<hardware::mock::Reader<double>>(
        std::vector{xerrors::NIL},
        std::vector{
            xerrors::Error(driver::CRITICAL_HARDWARE_ERROR, "Failed to stop hardware")
        }
    ));
    rt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->states.size(), 1);
    const auto start_state = ctx->states[0];
    EXPECT_EQ(start_state.variant, "success");
    rt->stop("stop_cmd", false);
    ASSERT_EVENTUALLY_GE(ctx->states.size(), 2);
    const auto stop_state = ctx->states[1];
    EXPECT_EQ(stop_state.key, "stop_cmd");
    EXPECT_EQ(stop_state.task, task.key);
    EXPECT_EQ(stop_state.variant, "error");
    EXPECT_EQ(stop_state.details["message"],
              "[sy.driver.hardware.critical] Failed to stop hardware");
}

/// @brief it should communicate an error when the hardware fails to read.
TEST_F(AnalogReadTest, testErrorOnRead) {}

/// @brief it should correctly coerce read data types to the channel data type.
TEST_F(AnalogReadTest, testDataTypeCoersion) {
    data_channel.data_type = telem::FLOAT32_T;
    parse_config();

    auto rt = create_task(std::make_unique<hardware::mock::Reader<double>>(
        std::vector{xerrors::NIL},
        std::vector{xerrors::NIL},
        std::vector<std::pair<std::vector<double>, xerrors::Error>>{
            {{1.23456789}, xerrors::NIL}
        }
    ));

    rt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->states.size(), 1);
    const auto start_state = ctx->states[0];
    EXPECT_EQ(start_state.variant, "success");

    ASSERT_EVENTUALLY_GE(mock_factory->writer_opens, 1);
    rt->stop("stop_cmd", false);
    ASSERT_EVENTUALLY_GE(ctx->states.size(), 2);
    const auto stop_state = ctx->states[1];
    EXPECT_EQ(stop_state.variant, "success");

    ASSERT_GE(mock_factory->writes->size(), 1);

    auto &fr = mock_factory->writes->at(0);
    ASSERT_EQ(fr.size(), 2);
    ASSERT_EQ(fr.length(), 1);
    ASSERT_EQ(fr.contains(data_channel.key), true);

    // Verify that the data was properly coerced to float32
    // The value should be stored as float32 and show some precision loss
    auto value = fr.at<float>(data_channel.key, 0);
    ASSERT_EQ(sizeof(value), sizeof(float)); // Verify it's actually float32
    EXPECT_FLOAT_EQ(value, 1.23456789f); // Should match float32 precision

    // Optional: Verify that the original double value and the float value are different
    // due to precision loss
    EXPECT_NE(static_cast<double>(value), 1.23456789);
}

/// @brief it should not double communicate state if the task is already started.
TEST_F(AnalogReadTest, testDoubleStart) {
    parse_config();
    const auto rt = create_task(std::make_unique<hardware::mock::Reader<double>>());

    rt->start("start_cmd1");
    rt->start("start_cmd2"); // Second start should be ignored

    ASSERT_EVENTUALLY_GE(ctx->states.size(), 1);
    EXPECT_EQ(ctx->states.size(), 1); // Should only have one state message
    const auto state = ctx->states[0];
    EXPECT_EQ(state.key, "start_cmd1");
    EXPECT_EQ(state.task, task.key);
    EXPECT_EQ(state.variant, "success");
    EXPECT_EQ(state.details["message"], "Task started successfully");

    rt->stop("stop_cmd", false);
}

/// @brief it should not double communicate state if the task is already stopped.
TEST_F(AnalogReadTest, testDoubleStop) {
    parse_config();
    const auto rt = create_task(std::make_unique<hardware::mock::Reader<double>>());

    rt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->states.size(), 1);

    rt->stop("stop_cmd1", false);
    rt->stop("stop_cmd2", false); // Second stop should be ignored

    ASSERT_EVENTUALLY_GE(ctx->states.size(), 2);
    EXPECT_EQ(ctx->states.size(), 2);
    // Should only have two state messages (start + stop)
    const auto stop_state = ctx->states[1];
    EXPECT_EQ(stop_state.key, "stop_cmd1");
    EXPECT_EQ(stop_state.task, task.key);
    EXPECT_EQ(stop_state.variant, "success");
    EXPECT_EQ(stop_state.details["message"], "Task stopped successfully");
}

class DigitalReadTest : public ::testing::Test {
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
        "digital_channel",
        telem::UINT8_T, // Digital data is typically boolean/uint8
        index_channel.key,
        false
    );

    void parse_config() {
        sy = std::make_shared<synnax::Synnax>(new_test_client());

        auto idx_err = sy->channels.create(index_channel);
        ASSERT_FALSE(idx_err) << idx_err;

        data_channel.index = index_channel.key;
        auto data_err = sy->channels.create(data_channel);
        ASSERT_FALSE(data_err) << data_err;

        auto [rack, rack_err] = sy->hardware.create_rack("digital_rack");
        ASSERT_FALSE(rack_err) << rack_err;

        synnax::Device dev(
            "130227d9-02aa-47e4-b370-0d590add1bc1",
            "digital_device",
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
            "digital_task",
            "ni_digital_read",
            ""
        );

        json j{
            {"data_saving", true},
            {"sample_rate", 25},
            {"stream_rate", 25},
            {"device", dev.key},
            {
                "channels", json::array({
                    {
                        {"type", "digital_input"},
                        {"key", "hCzuNC9glqc"},
                        {"port", 0},
                        {"enabled", true},
                        {"line", 1},
                        {"channel", data_channel.key},
                    }
                })
            }
        };

        auto p = xjson::Parser(j);
        cfg = std::make_unique<ni::ReadTaskConfig>(sy, p, "ni_digital_read");
        ASSERT_FALSE(p.error()) << p.error();

        auto client = std::make_shared<synnax::Synnax>(new_test_client());
        ctx = std::make_shared<task::MockContext>(client);
        mock_factory = std::make_shared<pipeline::mock::WriterFactory>();
    }

    std::unique_ptr<ni::ReadTask<uint8_t>> create_task(
        std::unique_ptr<hardware::mock::Reader<uint8_t>> mock_hw
    ) {
        return std::make_unique<ni::ReadTask<uint8_t>>(
            task,
            ctx,
            std::move(*cfg),
            breaker::default_config(task.name),
            std::move(mock_hw),
            mock_factory
        );
    }
};

/// @brief it should run a basic digital read task using a mock hardware implementation.
TEST_F(DigitalReadTest, testBasicDigitalRead) {
    parse_config();
    auto rt = create_task(std::make_unique<hardware::mock::Reader<uint8_t>>(
        std::vector{xerrors::NIL},
        std::vector{xerrors::NIL},
        std::vector<std::pair<std::vector<uint8_t>, xerrors::Error>>{
            {{1}, xerrors::NIL}
        } // Digital high
    ));

    rt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->states.size(), 1);
    const auto first_state = ctx->states[0];
    EXPECT_EQ(first_state.key, "start_cmd");
    EXPECT_EQ(first_state.task, task.key);
    EXPECT_EQ(first_state.variant, "success");
    EXPECT_EQ(first_state.details["message"], "Task started successfully");
    ASSERT_EVENTUALLY_GE(mock_factory->writer_opens, 1);

    rt->stop("stop_cmd", false);
    ASSERT_EVENTUALLY_GE(ctx->states.size(), 2);
    const auto second_state = ctx->states[1];
    EXPECT_EQ(second_state.key, "stop_cmd");
    EXPECT_EQ(second_state.task, task.key);
    EXPECT_EQ(second_state.variant, "success");
    EXPECT_EQ(second_state.details["message"], "Task stopped successfully");

    ASSERT_GE(mock_factory->writes->size(), 1);
    auto &fr = mock_factory->writes->at(0);
    ASSERT_EQ(fr.size(), 2);
    ASSERT_EQ(fr.length(), 1);
    ASSERT_TRUE(fr.contains(data_channel.key));
    ASSERT_TRUE(fr.contains(index_channel.key));
    ASSERT_EQ(fr.at<uint8_t>(data_channel.key, 0), 1); // Verify digital high
    ASSERT_GE(fr.at<uint64_t>(index_channel.key, 0), 0);
}

/// @brief it should interpolate timestamps between clock start and end times.
TEST(SampleClockTest, testHardwareTimedSampleClock) {
    ni::HardwareTimedSampleClock clock(1 * telem::HZ);
    breaker::Breaker breaker;
    clock.reset();
    const auto start = clock.wait(breaker);
    const auto end = clock.end(10);
    EXPECT_EQ(end - start, 9 * telem::SECOND);
}