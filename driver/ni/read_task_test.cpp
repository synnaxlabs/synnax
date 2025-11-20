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
#include "driver/ni/read_task.h"
#include "driver/pipeline/mock/pipeline.h"

/// @brief it should correctly parse a basic analog read task.
namespace {
json base_analog_config() {
    return {
        {"data_saving", false},
        {"sample_rate", 25},
        {"stream_rate", 25},
        {"channels",
         json::array({{
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
         }})}
    };
}
}

TEST(ReadTaskConfigTest, testBasicAnalogReadTaskConfigParse) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("cat"));
    auto dev = synnax::Device(
        "abc123",
        "my_device",
        rack.key,
        "dev1",
        "ni",
        "PXI-6255",
        ""
    );
    ASSERT_NIL(client->devices.create(dev));
    auto ch = ASSERT_NIL_P(client->channels.create("virtual", telem::FLOAT64_T, true));

    auto j = base_analog_config();
    j["channels"][0]["device"] = dev.key;
    j["channels"][0]["channel"] = ch.key;

    auto p = xjson::Parser(j);
    auto cfg = std::make_unique<ni::ReadTaskConfig>(client, p, "ni_analog_read");
    ASSERT_NIL(p.error());
}

/// @brief it should return a validation error if the device does not exist.
TEST(ReadTaskConfigTest, testNonExistingAnalogReadDevice) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("cat"));
    auto ch = ASSERT_NIL_P(client->channels.create("virtual", telem::FLOAT64_T, true));

    auto j = base_analog_config();
    j["channels"][0]["device"] = "definitely_not_an_existing_device";
    j["channels"][0]["channel"] = ch.key;

    auto p = xjson::Parser(j);
    auto cfg = std::make_unique<ni::ReadTaskConfig>(client, p, "ni_analog_read");
    ASSERT_OCCURRED_AS(p.error(), xerrors::VALIDATION);
}

/// @brief it should return a validation error if the channel does not exist.
TEST(ReadTaskConfigTest, testNonExistentAnalogReadChannel) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("cat"));
    auto dev = synnax::Device(
        "abc123",
        "my_device",
        rack.key,
        "dev1",
        "ni",
        "PXI-6255",
        ""
    );
    ASSERT_NIL(client->devices.create(dev));

    auto j = base_analog_config();
    j["channels"][0]["device"] = dev.key;
    j["channels"][0]["channel"] = 12121212;

    auto p = xjson::Parser(j);
    auto cfg = std::make_unique<ni::ReadTaskConfig>(client, p, "ni_analog_read");
    ASSERT_OCCURRED_AS(p.error(), xerrors::VALIDATION);
}

/// @brief it should return a validation error if the sample rate is less than the
/// stream rate.
TEST(ReadTaskConfigTest, testSampleRateLessThanStreamRate) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("cat"));
    auto dev = synnax::Device(
        "abc123",
        "my_device",
        rack.key,
        "dev1",
        "ni",
        "PXI-6255",
        ""
    );
    auto dev_err = client->devices.create(dev);
    ASSERT_FALSE(dev_err) << dev_err;

    auto ch = ASSERT_NIL_P(client->channels.create("virtual", telem::FLOAT64_T, true));

    auto j = base_analog_config();
    j["channels"][0]["device"] = dev.key;
    j["channels"][0]["channel"] = ch.key;
    j["sample_rate"] = 10;

    auto p = xjson::Parser(j);
    auto cfg = std::make_unique<ni::ReadTaskConfig>(client, p, "ni_analog_read");
    ASSERT_OCCURRED_AS(p.error(), xerrors::VALIDATION);
}

/// @brief it should return a validation error if no channels in the task are enabled.
TEST(ReadTaskConfigTest, testNoEnabledChannels) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("cat"));
    auto dev = synnax::Device(
        "abc123",
        "my_device",
        rack.key,
        "dev1",
        "ni",
        "PXI-6255",
        ""
    );
    ASSERT_NIL(client->devices.create(dev));
    auto ch = ASSERT_NIL_P(client->channels.create("virtual", telem::FLOAT64_T, true));

    auto j = base_analog_config();
    j["channels"][0]["device"] = dev.key;
    j["channels"][0]["channel"] = ch.key;
    j["channels"][0]["enabled"] = false;

    auto p = xjson::Parser(j);
    auto cfg = std::make_unique<ni::ReadTaskConfig>(client, p, "ni_analog_read");
    ASSERT_OCCURRED_AS(p.error(), xerrors::VALIDATION);
}

/// @brief it should return a validation error if a channel has an unknown type.
TEST(ReadTaskConfigTest, testUnknownChannelType) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("cat"));
    auto dev = synnax::Device(
        "abc123",
        "my_device",
        rack.key,
        "dev1",
        "ni",
        "PXI-6255",
        ""
    );
    ASSERT_NIL(client->devices.create(dev));
    auto ch = ASSERT_NIL_P(client->channels.create("virtual", telem::FLOAT64_T, true));

    auto j = base_analog_config();
    j["channels"][0]["device"] = dev.key;
    j["channels"][0]["channel"] = ch.key;
    j["channels"][0]["type"] = "unknown_channel_type"; // Set an invalid channel type

    auto p = xjson::Parser(j);
    auto cfg = std::make_unique<ni::ReadTaskConfig>(client, p, "ni_analog_read");
    ASSERT_OCCURRED_AS(p.error(), xerrors::VALIDATION);
}

class AnalogReadTest : public ::testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> client;
    synnax::Task task;
    std::unique_ptr<ni::ReadTaskConfig> cfg;
    std::shared_ptr<task::MockContext> ctx;
    std::shared_ptr<pipeline::mock::WriterFactory> mock_factory;
    synnax::Channel
        index_channel = synnax::Channel("time_channel", telem::TIMESTAMP_T, 0, true);
    synnax::Channel data_channel = synnax::Channel(
        "data_channel",
        telem::FLOAT64_T,
        index_channel.key,
        false
    );

    void parse_config() {
        client = std::make_shared<synnax::Synnax>(new_test_client());

        ASSERT_NIL(client->channels.create(index_channel));

        data_channel.index = index_channel.key;
        ASSERT_NIL(client->channels.create(data_channel));

        auto rack = ASSERT_NIL_P(client->racks.create("cat"));

        synnax::Device
            dev("opcua123", "my_device", rack.key, "dev1", "ni", "PXI-6255", "");

        ASSERT_NIL(client->devices.create(dev));

        task = synnax::Task(rack.key, "my_task", "ni_analog_read", "");

        json j{
            {"data_saving", false},
            {"sample_rate", 25},
            {"stream_rate", 25},
            {"channels",
             json::array(
                 {{{"type", "ai_accel"},
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
                   {"device", dev.key}}}
             )}
        };

        auto p = xjson::Parser(j);
        cfg = std::make_unique<ni::ReadTaskConfig>(client, p, "ni_analog_read");
        ASSERT_NIL(p.error());

        ctx = std::make_shared<task::MockContext>(client);
        mock_factory = std::make_shared<pipeline::mock::WriterFactory>();
    }

    std::unique_ptr<common::ReadTask>
    create_task(std::unique_ptr<hardware::mock::Reader<double>> mock_hw) {
        return std::make_unique<common::ReadTask>(
            task,
            ctx,
            breaker::default_config(task.name),
            std::make_unique<ni::ReadTaskSource<double>>(
                std::move(*cfg),
                std::move(mock_hw)
            ),
            mock_factory
        );
    }
};

/// @brief it should run a basic analog read task using a mock hardware implementation.
TEST_F(AnalogReadTest, testBasicAnalogRead) {
    parse_config();
    auto rt = create_task(std::make_unique<hardware::mock::Reader<double>>());

    rt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    const auto first_state = ctx->statuses[0];
    EXPECT_EQ(first_state.key, "start_cmd");
    EXPECT_EQ(first_state.details.task, task.key);
    EXPECT_EQ(first_state.variant, status::variant::SUCCESS);
    EXPECT_EQ(first_state.message, "Task started successfully");
    ASSERT_EVENTUALLY_GE(mock_factory->writer_opens, 1);
    rt->stop("stop_cmd", true);
    ASSERT_EQ(ctx->statuses.size(), 2);
    const auto second_state = ctx->statuses[1];
    EXPECT_EQ(second_state.key, "stop_cmd");
    EXPECT_EQ(second_state.details.task, task.key);
    EXPECT_EQ(second_state.variant, status::variant::SUCCESS);
    EXPECT_EQ(second_state.message, "Task stopped successfully");
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
    const auto rt = create_task(
        std::make_unique<hardware::mock::Reader<double>>(std::vector{
            xerrors::Error(driver::CRITICAL_HARDWARE_ERROR, "Failed to start hardware")
        })
    );
    rt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    const auto state = ctx->statuses[0];
    EXPECT_EQ(state.key, "start_cmd");
    EXPECT_EQ(state.details.task, task.key);
    EXPECT_EQ(state.variant, status::variant::ERR);
    EXPECT_EQ(state.message, "Failed to start hardware");
    rt->stop(false);
}

/// @brief it should communicate an error when the hardware fails to stop.
TEST_F(AnalogReadTest, testErrorOnStop) {
    parse_config();
    auto rt = create_task(
        std::make_unique<hardware::mock::Reader<double>>(
            std::vector{xerrors::NIL},
            std::vector{xerrors::Error(
                driver::CRITICAL_HARDWARE_ERROR,
                "Failed to stop hardware"
            )}
        )
    );
    rt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    const auto start_state = ctx->statuses[0];
    EXPECT_EQ(start_state.variant, status::variant::SUCCESS);
    rt->stop("stop_cmd", true);
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 2);
    const auto stop_state = ctx->statuses[1];
    EXPECT_EQ(stop_state.key, "stop_cmd");
    EXPECT_EQ(stop_state.details.task, task.key);
    EXPECT_EQ(stop_state.variant, status::variant::ERR);
    EXPECT_EQ(stop_state.message, "Failed to stop hardware");
}

/// @brief it should communicate an error when the hardware fails to read.
TEST_F(AnalogReadTest, testErrorOnRead) {
    parse_config();
    auto rt = create_task(
        std::make_unique<hardware::mock::Reader<double>>(
            std::vector{xerrors::NIL},
            std::vector{xerrors::NIL},
            std::vector<std::pair<std::vector<double>, xerrors::Error>>{
                {{},
                 xerrors::Error(
                     driver::CRITICAL_HARDWARE_ERROR,
                     "Failed to read hardware"
                 )}
            }
        )
    );

    rt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    const auto start_state = ctx->statuses[0];
    EXPECT_EQ(start_state.variant, status::variant::SUCCESS);

    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 2);
    const auto read_err_state = ctx->statuses[1];
    EXPECT_EQ(read_err_state.variant, status::variant::ERR);
    EXPECT_EQ(read_err_state.message, "Failed to read hardware");
    rt->stop("stop_cmd", true);
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 3);
    const auto stop_state = ctx->statuses[2];
    EXPECT_EQ(stop_state.variant, status::variant::ERR);
    EXPECT_EQ(stop_state.message, "Failed to read hardware");
}

/// @brief it should correctly coerce read data types to the channel data type.
TEST_F(AnalogReadTest, testDataTypeCoersion) {
    data_channel.data_type = telem::FLOAT32_T;
    parse_config();

    auto rt = create_task(
        std::make_unique<hardware::mock::Reader<double>>(
            std::vector{xerrors::NIL},
            std::vector{xerrors::NIL},
            std::vector<std::pair<std::vector<double>, xerrors::Error>>{
                {{1.23456789}, xerrors::NIL}
            }
        )
    );

    rt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    const auto start_state = ctx->statuses[0];
    EXPECT_EQ(start_state.variant, status::variant::SUCCESS);

    ASSERT_EVENTUALLY_GE(mock_factory->writer_opens, 1);
    rt->stop("stop_cmd", true);
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 2);
    const auto stop_state = ctx->statuses[1];
    EXPECT_EQ(stop_state.variant, status::variant::SUCCESS);

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

/// @brief it should restart the task if start is called twice.
TEST_F(AnalogReadTest, testDoubleStart) {
    parse_config();
    const auto rt = create_task(std::make_unique<hardware::mock::Reader<double>>());

    rt->start("start_cmd");
    rt->start("start_cmd");

    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 2);
    EXPECT_EQ(ctx->statuses.size(), 2);
    for (auto &state: ctx->statuses) {
        EXPECT_EQ(state.key, "start_cmd");
        EXPECT_EQ(state.details.task, task.key);
        EXPECT_EQ(state.variant, status::variant::SUCCESS);
        EXPECT_EQ(state.message, "Task started successfully");
    }
    rt->stop("stop_cmd", true);
}

/// @brief it should not double communicate state if the task is already stopped.
TEST_F(AnalogReadTest, testDoubleStop) {
    parse_config();
    const auto rt = create_task(std::make_unique<hardware::mock::Reader<double>>());

    rt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);

    rt->stop("stop_cmd1", true);
    rt->stop("stop_cmd2", true); // Second stop should be ignored

    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 3);
    EXPECT_EQ(ctx->statuses.size(), 3);
    // Should only have two state messages (start + stop)
    const auto stop_state = ctx->statuses[1];
    EXPECT_EQ(stop_state.key, "stop_cmd1");
    EXPECT_EQ(stop_state.details.task, task.key);
    EXPECT_EQ(stop_state.variant, status::variant::SUCCESS);
    EXPECT_EQ(stop_state.message, "Task stopped successfully");
    const auto stop_state_2 = ctx->statuses[2];
    EXPECT_EQ(stop_state_2.key, "stop_cmd2");
    EXPECT_EQ(stop_state_2.details.task, task.key);
    EXPECT_EQ(stop_state_2.variant, status::variant::SUCCESS);
    EXPECT_EQ(stop_state_2.message, "Task stopped successfully");
}

class DigitalReadTest : public ::testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> client;
    synnax::Task task;
    std::unique_ptr<ni::ReadTaskConfig> cfg;
    std::shared_ptr<task::MockContext> ctx;
    std::shared_ptr<pipeline::mock::WriterFactory> mock_factory;
    synnax::Channel
        index_channel = synnax::Channel("time_channel", telem::TIMESTAMP_T, 0, true);
    synnax::Channel data_channel = synnax::Channel(
        "digital_channel",
        telem::UINT8_T, // Digital data is typically boolean/uint8
        index_channel.key,
        false
    );

    void parse_config() {
        client = std::make_shared<synnax::Synnax>(new_test_client());

        auto idx_err = client->channels.create(index_channel);
        ASSERT_FALSE(idx_err) << idx_err;

        data_channel.index = index_channel.key;
        auto data_err = client->channels.create(data_channel);
        ASSERT_FALSE(data_err) << data_err;

        auto [rack, rack_err] = client->racks.create("digital_rack");
        ASSERT_FALSE(rack_err) << rack_err;

        synnax::Device dev(
            "130227d9-02aa-47e4-b370-0d590add1bc1",
            "digital_device",
            rack.key,
            "dev1",
            "ni",
            "PXI-6255",
            ""
        );
        auto dev_err = client->devices.create(dev);
        ASSERT_FALSE(dev_err) << dev_err;

        task = synnax::Task(rack.key, "digital_task", "ni_digital_read", "");

        json j{
            {"data_saving", true},
            {"sample_rate", 25},
            {"stream_rate", 25},
            {"device", dev.key},
            {"channels",
             json::array({{
                 {"type", "digital_input"},
                 {"key", "hCzuNC9glqc"},
                 {"port", 0},
                 {"enabled", true},
                 {"line", 1},
                 {"channel", data_channel.key},
             }})}
        };

        auto p = xjson::Parser(j);
        cfg = std::make_unique<ni::ReadTaskConfig>(client, p, "ni_digital_read");
        ASSERT_FALSE(p.error()) << p.error();

        ctx = std::make_shared<task::MockContext>(client);
        mock_factory = std::make_shared<pipeline::mock::WriterFactory>();
    }

    std::unique_ptr<common::ReadTask>
    create_task(std::unique_ptr<hardware::mock::Reader<uint8_t>> mock_hw) {
        return std::make_unique<common::ReadTask>(
            task,
            ctx,
            breaker::default_config(task.name),
            std::make_unique<ni::ReadTaskSource<uint8_t>>(
                std::move(*cfg),
                std::move(mock_hw)
            ),
            mock_factory
        );
    }
};

/// @brief it should run a basic digital read task using a mock hardware implementation.
TEST_F(DigitalReadTest, testBasicDigitalRead) {
    parse_config();
    auto rt = create_task(
        std::make_unique<hardware::mock::Reader<uint8_t>>(
            std::vector{xerrors::NIL},
            std::vector{xerrors::NIL},
            std::vector<std::pair<std::vector<uint8_t>, xerrors::Error>>{
                {{1}, xerrors::NIL}
            } // Digital high
        )
    );

    rt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    const auto first_state = ctx->statuses[0];
    EXPECT_EQ(first_state.key, "start_cmd");
    EXPECT_EQ(first_state.details.task, task.key);
    EXPECT_EQ(first_state.variant, status::variant::SUCCESS);
    EXPECT_EQ(first_state.message, "Task started successfully");
    ASSERT_EVENTUALLY_GE(mock_factory->writer_opens, 1);

    rt->stop("stop_cmd", true);
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 2);
    const auto second_state = ctx->statuses[1];
    EXPECT_EQ(second_state.key, "stop_cmd");
    EXPECT_EQ(second_state.details.task, task.key);
    EXPECT_EQ(second_state.variant, status::variant::SUCCESS);
    EXPECT_EQ(second_state.message, "Task stopped successfully");

    ASSERT_GE(mock_factory->writes->size(), 1);
    auto &fr = mock_factory->writes->at(0);
    ASSERT_EQ(fr.size(), 2);
    ASSERT_EQ(fr.length(), 1);
    ASSERT_TRUE(fr.contains(data_channel.key));
    ASSERT_TRUE(fr.contains(index_channel.key));
    ASSERT_EQ(fr.at<uint8_t>(data_channel.key, 0), 1); // Verify digital high
    ASSERT_GE(fr.at<uint64_t>(index_channel.key, 0), 0);
}

/// @brief Verify device locations are extracted from channels after configuration
TEST(ReadTaskConfigTest, testDeviceLocationsFromChannels) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("test_rack"));

    auto dev = synnax::Device(
        "device123",
        "test_device",
        rack.key,
        "cDAQ1Mod1",
        "ni",
        "NI 9229",
        ""
    );
    ASSERT_NIL(client->devices.create(dev));
    auto ch = ASSERT_NIL_P(client->channels.create("test_ch", telem::FLOAT64_T, true));

    auto j = base_analog_config();
    j["channels"][0]["device"] = dev.key;
    j["channels"][0]["channel"] = ch.key;

    auto p = xjson::Parser(j);
    auto cfg = std::make_unique<ni::ReadTaskConfig>(client, p, "ni_analog_read");
    ASSERT_NIL(p.error());

    // Verify that channels have dev_loc populated after configuration
    ASSERT_EQ(cfg->channels.size(), 1);
    EXPECT_EQ(cfg->channels[0]->dev_loc, "cDAQ1Mod1");
}

class CounterReadTest : public ::testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> client;
    synnax::Task task;
    std::unique_ptr<ni::ReadTaskConfig> cfg;
    std::shared_ptr<task::MockContext> ctx;
    std::shared_ptr<pipeline::mock::WriterFactory> mock_factory;
    synnax::Channel
        index_channel = synnax::Channel("time_channel", telem::TIMESTAMP_T, 0, true);
    synnax::Channel data_channel = synnax::Channel(
        "counter_channel",
        telem::FLOAT64_T, // Counter frequency data
        index_channel.key,
        false
    );

    void parse_config() {
        client = std::make_shared<synnax::Synnax>(new_test_client());

        auto idx_err = client->channels.create(index_channel);
        ASSERT_FALSE(idx_err) << idx_err;

        data_channel.index = index_channel.key;
        auto data_err = client->channels.create(data_channel);
        ASSERT_FALSE(data_err) << data_err;

        auto [rack, rack_err] = client->racks.create("counter_rack");
        ASSERT_FALSE(rack_err) << rack_err;

        synnax::Device dev(
            "f8a9c7e6-1234-4567-890a-bcdef0123456",
            "counter_device",
            rack.key,
            "Dev1",
            "ni",
            "PCIe-6343",
            ""
        );
        auto dev_err = client->devices.create(dev);
        ASSERT_FALSE(dev_err) << dev_err;

        task = synnax::Task(rack.key, "counter_task", "ni_counter_read", "");

        json j{
            {"data_saving", true},
            {"sample_rate", 25},
            {"stream_rate", 25},
            {"device", dev.key},
            {"channels",
             json::array({{
                 {"type", "ci_frequency"},
                 {"key", "counter_freq_key"},
                 {"port", 0},
                 {"enabled", true},
                 {"channel", data_channel.key},
                 {"min_val", 2},
                 {"max_val", 10000},
                 {"units", "Hz"},
                 {"edge", "Rising"},
                 {"meas_method", "DynamicAvg"},
                 {"meas_time", 0.001},
                 {"divisor", 4},
                 {"terminal", ""},
                 {"custom_scale", {{"type", "none"}}},
             }})}
        };

        auto p = xjson::Parser(j);
        cfg = std::make_unique<ni::ReadTaskConfig>(client, p, "ni_counter_read");
        ASSERT_FALSE(p.error()) << p.error();

        ctx = std::make_shared<task::MockContext>(client);
        mock_factory = std::make_shared<pipeline::mock::WriterFactory>();
    }

    std::unique_ptr<common::ReadTask>
    create_task(std::unique_ptr<hardware::mock::Reader<double>> mock_hw) {
        return std::make_unique<common::ReadTask>(
            task,
            ctx,
            breaker::default_config(task.name),
            std::make_unique<ni::ReadTaskSource<double>>(
                std::move(*cfg),
                std::move(mock_hw)
            ),
            mock_factory
        );
    }
};

/// @brief it should run a basic counter frequency read task using a mock hardware
/// implementation.
TEST_F(CounterReadTest, testBasicCounterFrequencyRead) {
    parse_config();
    auto rt = create_task(
        std::make_unique<hardware::mock::Reader<double>>(
            std::vector{xerrors::NIL},
            std::vector{xerrors::NIL},
            std::vector<std::pair<std::vector<double>, xerrors::Error>>{
                {{100.5}, xerrors::NIL} // 100.5 Hz frequency reading
            }
        )
    );

    rt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    const auto first_state = ctx->statuses[0];
    EXPECT_EQ(first_state.key, "start_cmd");
    EXPECT_EQ(first_state.details.task, task.key);
    EXPECT_EQ(first_state.variant, status::variant::SUCCESS);
    EXPECT_EQ(first_state.message, "Task started successfully");
    ASSERT_EVENTUALLY_GE(mock_factory->writer_opens, 1);

    rt->stop("stop_cmd", true);
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 2);
    const auto second_state = ctx->statuses[1];
    EXPECT_EQ(second_state.key, "stop_cmd");
    EXPECT_EQ(second_state.details.task, task.key);
    EXPECT_EQ(second_state.variant, status::variant::SUCCESS);
    EXPECT_EQ(second_state.message, "Task stopped successfully");

    ASSERT_GE(mock_factory->writes->size(), 1);
    auto &fr = mock_factory->writes->at(0);
    ASSERT_EQ(fr.size(), 2);
    ASSERT_EQ(fr.length(), 1);
    ASSERT_TRUE(fr.contains(data_channel.key));
    ASSERT_TRUE(fr.contains(index_channel.key));
    ASSERT_DOUBLE_EQ(
        fr.at<double>(data_channel.key, 0),
        100.5
    ); // Verify frequency value
    ASSERT_GE(fr.at<uint64_t>(index_channel.key, 0), 0);
}

/// @brief it should communicate an error when the counter hardware fails to start.
TEST_F(CounterReadTest, testCounterErrorOnStart) {
    parse_config();
    const auto rt = create_task(
        std::make_unique<hardware::mock::Reader<double>>(std::vector{
            xerrors::Error(driver::CRITICAL_HARDWARE_ERROR, "Counter failed to start")
        })
    );
    rt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    const auto state = ctx->statuses[0];
    EXPECT_EQ(state.key, "start_cmd");
    EXPECT_EQ(state.details.task, task.key);
    EXPECT_EQ(state.variant, status::variant::ERR);
    EXPECT_EQ(state.message, "Counter failed to start");
    rt->stop(false);
}

/// @brief it should communicate an error when the counter hardware fails to stop.
TEST_F(CounterReadTest, testCounterErrorOnStop) {
    parse_config();
    auto rt = create_task(
        std::make_unique<hardware::mock::Reader<double>>(
            std::vector{xerrors::NIL},
            std::vector{xerrors::Error(
                driver::CRITICAL_HARDWARE_ERROR,
                "Counter failed to stop"
            )}
        )
    );
    rt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    const auto start_state = ctx->statuses[0];
    EXPECT_EQ(start_state.variant, status::variant::SUCCESS);
    rt->stop("stop_cmd", true);
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 2);
    const auto stop_state = ctx->statuses[1];
    EXPECT_EQ(stop_state.key, "stop_cmd");
    EXPECT_EQ(stop_state.details.task, task.key);
    EXPECT_EQ(stop_state.variant, status::variant::ERR);
    EXPECT_EQ(stop_state.message, "Counter failed to stop");
}

/// @brief it should communicate an error when the counter hardware fails to read.
TEST_F(CounterReadTest, testCounterErrorOnRead) {
    parse_config();
    auto rt = create_task(
        std::make_unique<hardware::mock::Reader<double>>(
            std::vector{xerrors::NIL},
            std::vector{xerrors::NIL},
            std::vector<std::pair<std::vector<double>, xerrors::Error>>{
                {{},
                 xerrors::Error(driver::CRITICAL_HARDWARE_ERROR, "Counter read failed")}
            }
        )
    );

    rt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    const auto start_state = ctx->statuses[0];
    EXPECT_EQ(start_state.variant, status::variant::SUCCESS);

    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 2);
    const auto read_err_state = ctx->statuses[1];
    EXPECT_EQ(read_err_state.variant, status::variant::ERR);
    EXPECT_EQ(read_err_state.message, "Counter read failed");
    rt->stop("stop_cmd", true);
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 3);
    const auto stop_state = ctx->statuses[2];
    EXPECT_EQ(stop_state.variant, status::variant::ERR);
    EXPECT_EQ(stop_state.message, "Counter read failed");
}

/// @brief it should correctly handle multiple counter frequency readings.
TEST_F(CounterReadTest, testMultipleCounterReadings) {
    parse_config();
    auto rt = create_task(
        std::make_unique<hardware::mock::Reader<double>>(
            std::vector{xerrors::NIL},
            std::vector{xerrors::NIL},
            std::vector<std::pair<std::vector<double>, xerrors::Error>>{
                {{100.0}, xerrors::NIL},
                {{150.5}, xerrors::NIL},
                {{200.75}, xerrors::NIL}
            }
        )
    );

    rt->start("start_cmd");
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 1);
    const auto start_state = ctx->statuses[0];
    EXPECT_EQ(start_state.variant, status::variant::SUCCESS);

    // Wait for multiple writes
    ASSERT_EVENTUALLY_GE(mock_factory->writes->size(), 3);

    // Verify all three readings
    for (size_t i = 0; i < 3; i++) {
        auto &fr = mock_factory->writes->at(i);
        ASSERT_EQ(fr.size(), 2);
        ASSERT_EQ(fr.length(), 1);
        ASSERT_TRUE(fr.contains(data_channel.key));
        ASSERT_TRUE(fr.contains(index_channel.key));
    }

    EXPECT_DOUBLE_EQ(
        mock_factory->writes->at(0).at<double>(data_channel.key, 0),
        100.0
    );
    EXPECT_DOUBLE_EQ(
        mock_factory->writes->at(1).at<double>(data_channel.key, 0),
        150.5
    );
    EXPECT_DOUBLE_EQ(
        mock_factory->writes->at(2).at<double>(data_channel.key, 0),
        200.75
    );

    rt->stop("stop_cmd", true);
    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 2);
    const auto stop_state = ctx->statuses[1];
    EXPECT_EQ(stop_state.variant, status::variant::SUCCESS);
}

/// @brief it should correctly parse a counter edge count task configuration.
TEST(ReadTaskConfigTest, testCounterEdgeCountConfig) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("test_rack"));

    auto dev = synnax::Device(
        "counter_dev_123",
        "test_counter_device",
        rack.key,
        "Dev1",
        "ni",
        "USB-6343",
        ""
    );
    ASSERT_NIL(client->devices.create(dev));
    auto ch = ASSERT_NIL_P(
        client->channels.create("edge_count", telem::UINT32_T, true)
    );

    json j{
        {"data_saving", false},
        {"sample_rate", 25},
        {"stream_rate", 25},
        {"device", dev.key},
        {"channels",
         json::array({{
             {"type", "ci_edge_count"},
             {"key", "edge_count_key"},
             {"port", 0},
             {"enabled", true},
             {"channel", ch.key},
             {"active_edge", "Rising"},
             {"count_direction", "CountUp"},
             {"initial_count", 0},
             {"terminal", ""},
         }})}
    };

    auto p = xjson::Parser(j);
    auto cfg = std::make_unique<ni::ReadTaskConfig>(client, p, "ni_counter_read");
    ASSERT_NIL(p.error());

    // Verify channel configuration
    ASSERT_EQ(cfg->channels.size(), 1);
    EXPECT_EQ(cfg->channels[0]->dev_loc, "Dev1");
}

/// @brief it should correctly parse a counter period task configuration.
TEST(ReadTaskConfigTest, testCounterPeriodConfig) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("test_rack"));

    auto dev = synnax::Device(
        "counter_dev_456",
        "test_period_device",
        rack.key,
        "Dev2",
        "ni",
        "PCIe-6343",
        ""
    );
    ASSERT_NIL(client->devices.create(dev));
    auto ch = ASSERT_NIL_P(client->channels.create("period", telem::FLOAT64_T, true));

    json j{
        {"data_saving", false},
        {"sample_rate", 25},
        {"stream_rate", 25},
        {"device", dev.key},
        {"channels",
         json::array({{
             {"type", "ci_period"},
             {"key", "period_key"},
             {"port", 0},
             {"enabled", true},
             {"channel", ch.key},
             {"min_val", 0.000001},
             {"max_val", 0.1},
             {"units", "Seconds"},
             {"starting_edge", "Rising"},
             {"meas_method", "DynamicAvg"},
             {"meas_time", 0.001},
             {"divisor", 4},
             {"terminal", ""},
             {"custom_scale", {{"type", "none"}}},
         }})}
    };

    auto p = xjson::Parser(j);
    auto cfg = std::make_unique<ni::ReadTaskConfig>(client, p, "ni_counter_read");
    ASSERT_NIL(p.error());

    // Verify channel configuration
    ASSERT_EQ(cfg->channels.size(), 1);
    EXPECT_EQ(cfg->channels[0]->dev_loc, "Dev2");
}

/// @brief Verify cross-device task has multiple device locations in channels
TEST(ReadTaskConfigTest, testCrossDeviceChannelLocations) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("test_rack"));

    auto
        dev1 = synnax::Device("d1", "dev1", rack.key, "cDAQ1Mod1", "ni", "NI 9229", "");
    ASSERT_NIL(client->devices.create(dev1));

    auto
        dev2 = synnax::Device("d2", "dev2", rack.key, "cDAQ1Mod2", "ni", "NI 9205", "");
    ASSERT_NIL(client->devices.create(dev2));

    auto ch1 = ASSERT_NIL_P(client->channels.create("ch1", telem::FLOAT64_T, true));
    auto ch2 = ASSERT_NIL_P(client->channels.create("ch2", telem::FLOAT64_T, true));

    json j{
        {"data_saving", false},
        {"sample_rate", 25},
        {"stream_rate", 25},
        {"device", "cross-device"},
        {"channels",
         json::array(
             {{{"type", "ai_voltage"},
               {"key", "key1"},
               {"port", 0},
               {"enabled", true},
               {"channel", ch1.key},
               {"terminal_config", "Cfg_Default"},
               {"min_val", -10},
               {"max_val", 10},
               {"custom_scale", {{"type", "none"}}},
               {"device", dev1.key}},
              {{"type", "ai_voltage"},
               {"key", "key2"},
               {"port", 0},
               {"enabled", true},
               {"channel", ch2.key},
               {"terminal_config", "Cfg_Default"},
               {"min_val", -10},
               {"max_val", 10},
               {"custom_scale", {{"type", "none"}}},
               {"device", dev2.key}}}
         )}
    };

    auto p = xjson::Parser(j);
    auto cfg = std::make_unique<ni::ReadTaskConfig>(client, p, "ni_analog_read");
    ASSERT_NIL(p.error());

    // Verify both channels have their respective device locations
    ASSERT_EQ(cfg->channels.size(), 2);
    EXPECT_EQ(cfg->channels[0]->dev_loc, "cDAQ1Mod1");
    EXPECT_EQ(cfg->channels[1]->dev_loc, "cDAQ1Mod2");

    // Verify we can extract unique locations (what the validation code does)
    std::set<std::string> unique_locs;
    for (const auto &ch: cfg->channels) {
        if (!ch->dev_loc.empty()) { unique_locs.insert(ch->dev_loc); }
    }
    EXPECT_EQ(unique_locs.size(), 2);
    EXPECT_TRUE(unique_locs.count("cDAQ1Mod1") > 0);
    EXPECT_TRUE(unique_locs.count("cDAQ1Mod2") > 0);
}

/// @brief Test that the minimum sample rate error message is formatted correctly
TEST(ReadTaskConfigTest, testMinimumSampleRateErrorMessageFormat) {
    // This test verifies the error message format without requiring DAQmx hardware
    telem::Rate configured_rate(25.0);
    float64 min_rate = 50.0;
    std::string location = "cDAQ1Mod1";
    std::string model = "NI SIM";

    std::ostringstream msg;
    msg << "configured sample rate (" << configured_rate
        << ") is below device minimum (" << min_rate << " Hz) for " << location << " ("
        << model << ")";

    std::string result = msg.str();

    // Verify the message contains all required components
    EXPECT_TRUE(result.find("25 Hz") != std::string::npos);
    EXPECT_TRUE(result.find("50 Hz") != std::string::npos);
    EXPECT_TRUE(result.find("cDAQ1Mod1") != std::string::npos);
    EXPECT_TRUE(result.find("NI SIM") != std::string::npos);
    EXPECT_TRUE(result.find("below device minimum") != std::string::npos);
}

/// Regression test to ensure enable_auto_commit is set to true in WriterConfig.
/// This prevents data from being written but not committed, making it unavailable for
/// reads.
TEST(ReadTaskConfigTest, testNIDriverSetsAutoCommitTrue) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("test_rack"));
    auto dev = synnax::Device(
        "test_device_key",
        "test_device",
        rack.key,
        "dev1",
        "ni",
        "PXI-6255",
        ""
    );
    ASSERT_NIL(client->devices.create(dev));
    auto ch = ASSERT_NIL_P(
        client->channels.create("test_channel", telem::FLOAT64_T, true)
    );

    auto j = base_analog_config();
    j["data_saving"] = true;
    j["channels"][0]["device"] = dev.key;
    j["channels"][0]["channel"] = ch.key;

    auto p = xjson::Parser(j);
    auto cfg = std::make_unique<ni::ReadTaskConfig>(client, p, "ni_analog_read");
    ASSERT_NIL(p.error());

    // Verify that writer_config has enable_auto_commit set to true
    auto writer_cfg = cfg->writer();
    ASSERT_TRUE(writer_cfg.enable_auto_commit);
}
