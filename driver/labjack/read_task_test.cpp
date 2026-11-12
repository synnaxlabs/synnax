// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cmath>
#include <map>
#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/json/json.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/labjack/device/mock.h"
#include "driver/labjack/read_task.h"

namespace driver::labjack {
/// @brief the error LJM returns while the device is unplugged.
const auto UNREACHABLE = x::errors::Error(
    ljm::UNREACHABLE.sub("LJME_DEVICE_NOT_FOUND"),
    ljm::UNREACHABLE_MESSAGE
);

/// @brief it should parse analog input channel configuration.
TEST(TestReadChannelParse, testAIChan) {
    const x::json::json cfg{
        {"port", "AIN0"},
        {"disabled", false},
        {"key", "ain0"},
        {"channel", 1},
        {"type", "analog"},
        {"range", 5},
        {"scale", {{"type", "linear"}, {"slope", 1}, {"offset", 2}}}
    };
    auto p = x::json::Parser(cfg);
    const auto chan = parse_input_chan(p);
    ASSERT_NIL(p.error());
    auto *const ai_chan = dynamic_cast<AIChan *>(chan.get());
    ASSERT_NE(ai_chan, nullptr);
    ASSERT_EQ(ai_chan->port, "AIN0");
    ASSERT_EQ(ai_chan->enabled, true);
    ASSERT_EQ(ai_chan->synnax_key, 1);
    ASSERT_EQ(ai_chan->range, 5);
}

/// @brief it should parse digital input channel configuration.
TEST(TestReadChannelParse, testDIChan) {
    const x::json::json cfg{
        {"port", "DIO0"},
        {"disabled", false},
        {"key", "ain0"},
        {"channel", 1},
        {"type", "digital"}
    };
    auto p = x::json::Parser(cfg);
    const auto chan = parse_input_chan(p);
    ASSERT_NIL(p.error());
    auto *const di_chan = dynamic_cast<DIChan *>(chan.get());
    ASSERT_NE(di_chan, nullptr);
    ASSERT_EQ(di_chan->port, "DIO0");
    ASSERT_EQ(di_chan->enabled, true);
    ASSERT_EQ(di_chan->synnax_key, 1);
}

/// @brief it should parse thermocouple channel configuration.
TEST(TestReadChannelParse, testTCChan) {
    const x::json::json cfg{
        {"port", "AIN3"},
        {"disabled", false},
        {"key", "ain0"},
        {"channel", 0},
        {"type", "thermocouple"},
        {"range", 0},
        {"scale", {{"type", "linear"}, {"slope", 1}, {"offset", 2}}},
        {"thermocouple_type", "K"},
        {"units", "K"},
        {"cjc_source", "TEMPERATURE_DEVICE_K"},
        {"cjc_slope", 1},
        {"cjc_offset", 0}
    };
    auto p = x::json::Parser(cfg);
    const auto chan = parse_input_chan(p);
    ASSERT_NIL(p.error());
    auto *const tc_chan = dynamic_cast<ThermocoupleChan *>(chan.get());
    ASSERT_NE(tc_chan, nullptr);
    ASSERT_EQ(tc_chan->port, "AIN3");
    ASSERT_EQ(tc_chan->enabled, true);
    ASSERT_EQ(tc_chan->synnax_key, 0);
    ASSERT_EQ(tc_chan->type, LJM_ttK);
    ASSERT_EQ(tc_chan->pos_chan, 3);
    ASSERT_EQ(tc_chan->neg_chan, LJM_GND);
    ASSERT_EQ(tc_chan->units, LJM_KELVIN);
    ASSERT_EQ(tc_chan->cjc_source, DEVICE_CJC_SOURCE);
    ASSERT_EQ(tc_chan->cjc_slope, 1);
    ASSERT_EQ(tc_chan->cjc_offset, 0);
}

/// @brief parses a K-type thermocouple channel on the port.
std::unique_ptr<InputChan> parse_thermocouple(
    const std::string &port,
    const std::string &cjc_source,
    const std::string &units = "K",
    const std::optional<std::uint16_t> neg_chan = std::nullopt
) {
    x::json::json cfg{
        {"port", port},
        {"channel", 1},
        {"type", "thermocouple"},
        {"thermocouple_type", "K"},
        {"units", units},
        {"cjc_source", cjc_source},
        {"cjc_slope", 1},
        {"cjc_offset", 0}
    };
    if (neg_chan.has_value()) cfg["neg_chan"] = *neg_chan;
    auto p = x::json::Parser(cfg);
    auto chan = parse_input_chan(p);
    EXPECT_NIL(p.error());
    return chan;
}

/// @brief parses an analog channel on AIN0 with the max voltage.
std::unique_ptr<InputChan> parse_analog(const double range) {
    const x::json::json cfg{
        {"port", "AIN0"},
        {"channel", 1},
        {"type", "analog"},
        {"range", range},
        {"neg_chan", 1},
    };
    auto p = x::json::Parser(cfg);
    auto chan = parse_input_chan(p);
    EXPECT_NIL(p.error());
    return chan;
}

/// @brief it should read an analog channel with no negative channel against ground.
TEST(TestReadChannelParse, testAIChanAbsentNegChanIsGround) {
    const x::json::json cfg{
        {"port", "AIN0"},
        {"channel", 1},
        {"type", "analog"},
        {"range", 10},
    };
    auto p = x::json::Parser(cfg);
    const auto chan = parse_input_chan(p);
    ASSERT_NIL(p.error());
    const auto *const ai = dynamic_cast<AIChan *>(chan.get());
    ASSERT_NE(ai, nullptr);
    ASSERT_EQ(ai->neg_chan, LJM_GND);
}

/// @brief it should reject an analog channel with a max voltage of zero.
TEST(TestReadChannelParse, testAIChanZeroRange) {
    const x::json::json cfg{
        {"port", "AIN0"},
        {"channel", 1},
        {"type", "analog"},
        {"range", 0},
    };
    auto p = x::json::Parser(cfg);
    parse_input_chan(p);
    ASSERT_OCCURRED_AS(p.error(), x::errors::VALIDATION);
    ASSERT_EQ(p.errors->size(), 1);
    EXPECT_EQ(p.errors->at(0)["path"], "range");
    EXPECT_EQ(p.errors->at(0)["message"], "Max voltage must be greater than 0");
}

/// @brief it should reject an analog channel with an invalid scale.
TEST(TestReadChannelParse, testAIChanInvalidScale) {
    const x::json::json cfg{
        {"port", "AIN0"},
        {"channel", 1},
        {"type", "analog"},
        {"range", 10},
        {"scale", {{"type", "polynomial"}, {"coeffs", x::json::json::array()}}},
    };
    auto p = x::json::Parser(cfg);
    parse_input_chan(p);
    ASSERT_EQ(p.errors->size(), 1);
    EXPECT_EQ(p.errors->at(0)["path"], "scale.coeffs");
}

/// @brief it should reject a thermocouple channel with an invalid scale.
TEST(TestReadChannelParse, testTCChanInvalidScale) {
    const x::json::json cfg{
        {"port", "AIN0"},
        {"channel", 1},
        {"type", "thermocouple"},
        {"thermocouple_type", "K"},
        {"neg_chan", 1},
        {"units", "C"},
        {"cjc_source", "TEMPERATURE_DEVICE_K"},
        {"cjc_slope", 1},
        {"cjc_offset", 0},
        {"scale",
         {{"type", "table"}, {"pre_scaled_vals", {0, 1}}, {"scaled_vals", {0}}}},
    };
    auto p = x::json::Parser(cfg);
    parse_input_chan(p);
    ASSERT_EQ(p.errors->size(), 2);
    EXPECT_EQ(p.errors->at(0)["path"], "scale.pre_scaled_vals");
    EXPECT_EQ(p.errors->at(1)["path"], "scale.scaled_vals");
}

/// @brief it should read an AIN CJC source on its own port, on every model.
TEST(TestReadChannelParse, testTCChanAINCJCSource) {
    const auto chan = parse_thermocouple("AIN0", "AIN2");
    const auto *const tc_chan = dynamic_cast<ThermocoupleChan *>(chan.get());
    ASSERT_NE(tc_chan, nullptr);
    EXPECT_EQ(tc_chan->cjc_register(T7), "AIN2");
    EXPECT_EQ(tc_chan->cjc_register(T8), "AIN2");
}

/// @brief it should read the device CJC source from the sensor at the terminal of the
/// thermocouple on a T8, and from the device sensor on a T7.
TEST(TestReadChannelParse, testTCChanDeviceCJCRegister) {
    const auto chan = parse_thermocouple("AIN3", DEVICE_CJC_SOURCE);
    const auto *const tc_chan = dynamic_cast<ThermocoupleChan *>(chan.get());
    ASSERT_NE(tc_chan, nullptr);
    EXPECT_EQ(tc_chan->cjc_register(T7), "TEMPERATURE_DEVICE_K");
    EXPECT_EQ(tc_chan->cjc_register(T8), "TEMPERATURE3");
}

/// @brief it should reject an AIN CJC source without a valid AIN index.
TEST(TestReadChannelParse, testTCChanInvalidAINCJCSource) {
    const x::json::json cfg{
        {"port", "AIN0"},
        {"channel", 1},
        {"type", "thermocouple"},
        {"cjc_source", "AINx"},
    };
    auto p = x::json::Parser(cfg);
    parse_input_chan(p);
    ASSERT_OCCURRED_AS(p.error(), x::errors::VALIDATION);
    ASSERT_EQ(p.errors->size(), 1);
    EXPECT_EQ(p.errors->at(0)["path"], "cjc_source");
    EXPECT_EQ(p.errors->at(0)["message"], "Invalid CJC source: AINx");
}

/// @brief it should turn off the thermocouple feature of the device and set the
/// thermocouple range on a T8, with no negative channel write.
TEST(TestReadChannelApply, testT8Thermocouple) {
    const auto chan = parse_thermocouple("AIN3", "TEMPERATURE_DEVICE_K");
    const auto dev = std::make_shared<device::Mock>();
    ASSERT_NIL(chan->apply(dev, T8));
    const std::map<std::string, double> expected{
        {"AIN3_EF_INDEX", 0},
        {"AIN3_RANGE", 0.15},
        {"AIN3_RESOLUTION_INDEX", 0},
    };
    EXPECT_EQ(dev->written_names(), expected);
}

/// @brief it should turn off the thermocouple feature of the device, and set the
/// thermocouple range and the negative channel on a T7.
TEST(TestReadChannelApply, testT7Thermocouple) {
    const auto chan = parse_thermocouple("AIN2", "TEMPERATURE_DEVICE_K", "K", 3);
    const auto dev = std::make_shared<device::Mock>();
    ASSERT_NIL(chan->apply(dev, T7));
    const std::map<std::string, double> expected{
        {"AIN2_EF_INDEX", 0},
        {"AIN2_RANGE", 0.1},
        {"AIN2_RESOLUTION_INDEX", 0},
        {"AIN2_NEGATIVE_CH", 3},
    };
    EXPECT_EQ(dev->written_names(), expected);
}

/// @brief it should refuse the air temperature CJC source on a T8, which has no air
/// temperature sensor.
TEST(TestReadChannelApply, testT8AirCJC) {
    const auto chan = parse_thermocouple("AIN1", "TEMPERATURE_AIR_K");
    const auto dev = std::make_shared<device::Mock>();
    const auto err = chan->apply(dev, T8);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(
        err.data,
        "AIN1 uses the air temperature as its CJC source, which the T8 does not have. "
        "Select another CJC source."
    );
    EXPECT_TRUE(dev->written_names().empty());
}

/// @brief it should refuse a thermocouple on a T4, which has no thermocouple inputs.
TEST(TestReadChannelApply, testT4Thermocouple) {
    const auto chan = parse_thermocouple("AIN2", "TEMPERATURE_DEVICE_K");
    const auto dev = std::make_shared<device::Mock>();
    const auto err = chan->apply(dev, T4);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(
        err.data,
        "AIN2 is a thermocouple channel, but the T4 does not support thermocouples."
    );
    EXPECT_TRUE(dev->written_names().empty());
}

/// @brief returns the temperature that the thermocouple channel converts from the
/// voltage and the CJC reading.
std::pair<double, x::errors::Error> temperature(
    const std::unique_ptr<InputChan> &chan,
    const device::Mock &dev,
    const double volts,
    const double cjc
) {
    const auto *const tc = dynamic_cast<ThermocoupleChan *>(chan.get());
    EXPECT_NE(tc, nullptr);
    return tc->temperature(dev, volts, cjc);
}

/// @brief it should convert a thermocouple voltage to a temperature in the units of
/// the channel.
TEST(TestThermocoupleTemperature, testConvertsToUnits) {
    const device::Mock dev;
    const auto kelvin = parse_thermocouple("AIN0", DEVICE_CJC_SOURCE, "K");
    EXPECT_DOUBLE_EQ(ASSERT_NIL_P(temperature(kelvin, dev, 0.001, 300)), 325);
    const auto celsius = parse_thermocouple("AIN0", DEVICE_CJC_SOURCE, "C");
    EXPECT_DOUBLE_EQ(ASSERT_NIL_P(temperature(celsius, dev, 0.001, 300)), 51.85);
    const auto fahrenheit = parse_thermocouple("AIN0", DEVICE_CJC_SOURCE, "F");
    EXPECT_NEAR(ASSERT_NIL_P(temperature(fahrenheit, dev, 0.001, 300)), 125.33, 1e-9);
}

/// @brief it should convert the CJC reading to Kelvin with the CJC slope and offset.
TEST(TestThermocoupleTemperature, testAppliesCJCSlopeAndOffset) {
    const x::json::json cfg{
        {"port", "AIN0"},
        {"channel", 1},
        {"type", "thermocouple"},
        {"thermocouple_type", "K"},
        {"units", "K"},
        {"cjc_source", "AIN2"},
        {"cjc_slope", 100},
        {"cjc_offset", 250},
    };
    auto p = x::json::Parser(cfg);
    const auto chan = parse_input_chan(p);
    ASSERT_NIL(p.error());
    const device::Mock dev;
    EXPECT_DOUBLE_EQ(ASSERT_NIL_P(temperature(chan, dev, 0.001, 0.5)), 325);
}

/// @brief it should return NaN for a voltage above the range of every thermocouple
/// type, without a call to LJM.
TEST(TestThermocoupleTemperature, testVoltageAboveEveryTypeIsNaN) {
    device::Mock dev;
    dev.tc_errors = {x::errors::Error("must not be called")};
    const auto chan = parse_thermocouple("AIN0", DEVICE_CJC_SOURCE);
    EXPECT_TRUE(std::isnan(ASSERT_NIL_P(temperature(chan, dev, -0.5, 300))));
    EXPECT_EQ(dev.tc_errors.size(), 1);
}

/// @brief it should return NaN when LJM rejects the voltage for the thermocouple type.
TEST(TestThermocoupleTemperature, testLJMOutOfRangeIsNaN) {
    device::Mock dev;
    dev.tc_errors = {ljm::VOLTAGE_OUT_OF_RANGE, ljm::TEMPERATURE_OUT_OF_RANGE};
    const auto chan = parse_thermocouple("AIN0", DEVICE_CJC_SOURCE);
    EXPECT_TRUE(std::isnan(ASSERT_NIL_P(temperature(chan, dev, 0.07, 300))));
    EXPECT_TRUE(std::isnan(ASSERT_NIL_P(temperature(chan, dev, 0.07, 300))));
}

/// @brief it should return any other conversion error.
TEST(TestThermocoupleTemperature, testOtherErrorsAreReturned) {
    device::Mock dev;
    dev.tc_errors = {ljm::CRITICAL_ERROR.sub("LJME_INVALID_PARAMETER")};
    const auto chan = parse_thermocouple("AIN0", DEVICE_CJC_SOURCE);
    ASSERT_OCCURRED_AS_P(temperature(chan, dev, 0.001, 300), ljm::CRITICAL_ERROR);
}

/// @brief it should select the smallest T8 range that covers the max voltage, with
/// no negative channel write.
TEST(TestReadChannelApply, testT8AnalogRange) {
    for (const auto &[max_voltage, range]:
         std::vector<std::pair<double, double>>{{5, 9.6}, {10, 11}, {0.075, 0.075}}) {
        const auto chan = parse_analog(max_voltage);
        const auto dev = std::make_shared<device::Mock>();
        ASSERT_NIL(chan->apply(dev, T8));
        const std::map<std::string, double> expected{
            {"AIN0_RESOLUTION_INDEX", 0},
            {"AIN0_RANGE", range},
        };
        EXPECT_EQ(dev->written_names(), expected);
    }
}

/// @brief it should select the smallest T7 range that covers the max voltage, and
/// write the negative channel.
TEST(TestReadChannelApply, testT7AnalogRange) {
    const auto chan = parse_analog(0.05);
    const auto dev = std::make_shared<device::Mock>();
    ASSERT_NIL(chan->apply(dev, T7));
    const std::map<std::string, double> expected{
        {"AIN0_RESOLUTION_INDEX", 0},
        {"AIN0_RANGE", 0.1},
        {"AIN0_NEGATIVE_CH", 1},
    };
    EXPECT_EQ(dev->written_names(), expected);
}

/// @brief it should leave the fixed T4 ranges alone.
TEST(TestReadChannelApply, testT4AnalogRange) {
    const auto chan = parse_analog(10);
    const auto dev = std::make_shared<device::Mock>();
    ASSERT_NIL(chan->apply(dev, T4));
    const std::map<std::string, double> expected{{"AIN0_RESOLUTION_INDEX", 0}};
    EXPECT_EQ(dev->written_names(), expected);
}

/// @brief it should refuse a max voltage above the largest range of the device.
TEST(TestReadChannelApply, testAnalogRangeAboveDeviceMax) {
    const auto chan = parse_analog(12);
    const auto dev = std::make_shared<device::Mock>();
    const auto err = chan->apply(dev, T8);
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(
        err.data,
        "AIN0 has a max voltage of 12 V, but the largest range of the device is ±11 V."
    );
}

/// @brief it should reject invalid channel type in configuration.
TEST(TestReadChannelParse, testInvalidChannelType) {
    const x::json::json cfg{
        {"port", "AIN0"},
        {"disabled", false},
        {"key", "ain0"},
        {"channel", 1},
        {"type", "INVALID_TYPE"}, // Invalid channel type
        {"range", 5},
        {"scale", {{"type", "linear"}, {"slope", 1}, {"offset", 2}}}
    };
    auto p = x::json::Parser(cfg);
    const auto chan = parse_input_chan(p);
    ASSERT_OCCURRED_AS(p.error(), x::errors::VALIDATION);
}

/// @brief it should derive the backlog threshold when the count is absent.
TEST(TestBacklogWarnOnCount, testAbsentDerivesFromSampleRate) {
    const auto cfg = x::json::json::object();
    auto p = x::json::Parser(cfg);
    const auto count = parse_backlog_warn_on_count(
        p,
        "ljm_scan_backlog_warn_on_count",
        x::telem::HERTZ * 50,
        1
    );
    ASSERT_NIL(p.error());
    ASSERT_EQ(count, 50);
}

/// @brief it should keep an explicitly configured backlog threshold.
TEST(TestBacklogWarnOnCount, testExplicitCountKept) {
    const x::json::json cfg{{"device_scan_backlog_warn_on_count", 7}};
    auto p = x::json::Parser(cfg);
    const auto count = parse_backlog_warn_on_count(
        p,
        "device_scan_backlog_warn_on_count",
        x::telem::HERTZ * 50,
        2
    );
    ASSERT_NIL(p.error());
    ASSERT_EQ(count, 7);
}

x::json::json basic_read_task_config(const std::string &device) {
    return {
        {"device", device},
        {"sample_rate", 10},
        {"stream_rate", 5},
        {"data_saving_disabled", false},
        {"channels",
         x::json::json::array(
             {{{"port", "AIN0"},
               {"disabled", false},
               {"key", "ain0"},
               {"channel", 0},
               {"type", "thermocouple"},
               {"range", 0},
               {"scale", {{"type", "linear"}, {"slope", 1}, {"offset", 2}}},
               {"thermocouple_type", "K"},
               {"units", "K"},
               {"cjc_source", "TEMPERATURE_DEVICE_K"},
               {"cjc_slope", 1},
               {"cjc_offset", 0}},
              {{"port", "DIO4"},
               {"disabled", false},
               {"key", "dio4"},
               {"channel", 0},
               {"type", "digital"}},
              {{"port", "AIN6"},
               {"disabled", false},
               {"key", "ain6"},
               {"channel", 0},
               {"type", "analog"},
               {"range", 10},
               {"scale", {{"type", "none"}}}}}
         )}
    };
}

/// @brief it should parse basic read task configuration with multiple channels.
TEST(TestReadTaskConfigParse, testBasicReadTaskConfigParse) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("cat"));
    auto dev = synnax::device::Device{
        .key = make_unique_channel_name("labjack"),
        .rack = rack.key,
        .location = "dev1",
        .make = "labjack",
        .model = T7,
        .name = "my_device",
    };
    ASSERT_NIL(client->devices.create(dev));

    // Create channels for each input type
    auto tc_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("tc_channel"),
        x::telem::FLOAT64_T,
        true
    ));
    auto di_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("di_channel"),
        x::telem::UINT8_T,
        true
    ));
    auto ai_ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("ai_channel"),
        x::telem::FLOAT64_T,
        true
    ));

    auto j = basic_read_task_config(dev.key);
    j["channels"][0]["channel"] = tc_ch.key;
    j["channels"][1]["channel"] = di_ch.key;
    j["channels"][2]["channel"] = ai_ch.key;

    auto p = x::json::Parser(j);
    auto cfg = std::make_unique<ReadTaskConfig>(client, p);
    ASSERT_NIL(p.error());

    ASSERT_EQ(cfg->sample_rate, x::telem::HERTZ * 10);
    ASSERT_EQ(cfg->stream_rate, x::telem::HERTZ * 5);
    ASSERT_EQ(cfg->data_saving_disabled, false);
    ASSERT_EQ(cfg->device_scan_backlog_warn_on_count, 20);
    ASSERT_EQ(cfg->ljm_scan_backlog_warn_on_count, 10);
    ASSERT_EQ(cfg->channels.size(), 3);

    auto *const tc_chan = dynamic_cast<ThermocoupleChan *>(cfg->channels[0].get());
    ASSERT_NE(tc_chan, nullptr);
    ASSERT_EQ(tc_chan->port, "AIN0");
    ASSERT_EQ(tc_chan->enabled, true);
    ASSERT_EQ(tc_chan->synnax_key, tc_ch.key);
    ASSERT_EQ(tc_chan->type, LJM_ttK);
    ASSERT_EQ(tc_chan->pos_chan, 0);
    ASSERT_EQ(tc_chan->neg_chan, LJM_GND);
    ASSERT_EQ(tc_chan->units, LJM_KELVIN);
    ASSERT_EQ(tc_chan->cjc_source, DEVICE_CJC_SOURCE);
    ASSERT_EQ(tc_chan->cjc_slope, 1);
    ASSERT_EQ(tc_chan->cjc_offset, 0);

    auto *const di_chan = dynamic_cast<DIChan *>(cfg->channels[1].get());
    ASSERT_NE(di_chan, nullptr);
    ASSERT_EQ(di_chan->port, "DIO4");
    ASSERT_EQ(di_chan->enabled, true);
    ASSERT_EQ(di_chan->synnax_key, di_ch.key);

    auto *const ai_chan = dynamic_cast<AIChan *>(cfg->channels[2].get());
    ASSERT_NE(ai_chan, nullptr);
    ASSERT_EQ(ai_chan->port, "AIN6");
    ASSERT_EQ(ai_chan->enabled, true);
    ASSERT_EQ(ai_chan->synnax_key, ai_ch.key);
    ASSERT_EQ(ai_chan->range, 10);
}

/// @brief it should reject invalid channel type in read task configuration.
TEST(TestReadTaskConfigParse, testInvalidChannelTypeInConfig) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("cat"));
    auto dev = synnax::device::Device{
        .key = make_unique_channel_name("labjack"),
        .rack = rack.key,
        .location = "dev1",
        .make = "labjack",
        .model = T7,
        .name = "my_device",
    };
    ASSERT_NIL(client->devices.create(dev));

    // Create a channel
    auto ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("test_channel"),
        x::telem::FLOAT64_T,
        true
    ));

    // Create a config with an invalid channel type
    auto j = basic_read_task_config(dev.key);
    j["channels"] = x::json::json::array(
        {{{"port", "AIN0"},
          {"disabled", false},
          {"key", "ain0"},
          {"channel", ch.key},
          {"type", "UNKNOWN_CHANNEL_TYPE"}, // Invalid channel type
          {"range", 5}}}
    );

    auto p = x::json::Parser(j);
    auto cfg = std::make_unique<ReadTaskConfig>(client, p);

    ASSERT_OCCURRED_AS(p.error(), x::errors::VALIDATION);
}

/// @brief it should enable auto commit in writer config for data availability.
TEST(TestReadTaskConfigParse, testLabJackDriverSetsAutoCommitTrue) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("test_rack"));
    auto dev = synnax::device::Device{
        .key = make_unique_channel_name("labjack"),
        .rack = rack.key,
        .location = "dev1",
        .make = "labjack",
        .model = T7,
        .name = "test_device",
    };
    ASSERT_NIL(client->devices.create(dev));
    auto ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("test_channel"),
        x::telem::FLOAT64_T,
        true
    ));

    auto j = basic_read_task_config(dev.key);
    j["data_saving_disabled"] = false;
    j["channels"] = x::json::json::array(
        {{{"port", "AIN0"},
          {"disabled", false},
          {"key", "ain0"},
          {"channel", ch.key},
          {"type", "analog"},
          {"range", 5},
          {"scale", {{"type", "none"}}}}}
    );

    auto p = x::json::Parser(j);
    auto cfg = std::make_unique<ReadTaskConfig>(client, p);
    ASSERT_NIL(p.error());

    // Verify that writer_config has enable_auto_commit set to true
    auto writer_cfg = cfg->writer();
    ASSERT_TRUE(writer_cfg.enable_auto_commit);
}

class SourceClaimTest : public ::testing::Test {
protected:
    std::unique_ptr<ReadTaskConfig> cfg;
    std::shared_ptr<device::MockManager> devs;

    /// @brief parses a single-channel config. A thermocouple channel selects the
    /// UnarySource path; an analog channel selects the StreamSource path.
    void parse_config(const bool thermocouple) {
        auto client = std::make_shared<synnax::Synnax>(new_test_client());
        auto rack = ASSERT_NIL_P(client->racks.create("cat"));
        auto dev = synnax::device::Device{
            .key = make_unique_channel_name("labjack"),
            .rack = rack.key,
            .location = "dev1",
            .make = "labjack",
            .model = T7,
            .name = "my_device",
        };
        ASSERT_NIL(client->devices.create(dev));
        auto ch = ASSERT_NIL_P(client->channels.create(
            make_unique_channel_name("claim_channel"),
            x::telem::FLOAT64_T,
            true
        ));
        auto j = basic_read_task_config(dev.key);
        if (thermocouple)
            j["channels"] = x::json::json::array(
                {{{"port", "AIN0"},
                  {"disabled", false},
                  {"key", "ain0"},
                  {"channel", ch.key},
                  {"type", "thermocouple"},
                  {"range", 0},
                  {"scale", {{"type", "none"}}},
                  {"thermocouple_type", "K"},
                  {"units", "K"},
                  {"cjc_source", "TEMPERATURE_DEVICE_K"},
                  {"cjc_slope", 1},
                  {"cjc_offset", 0}}}
            );
        else
            j["channels"] = x::json::json::array(
                {{{"port", "AIN0"},
                  {"disabled", false},
                  {"key", "ain0"},
                  {"channel", ch.key},
                  {"type", "analog"},
                  {"range", 5},
                  {"scale", {{"type", "none"}}}}}
            );
        auto p = x::json::Parser(j);
        this->cfg = std::make_unique<ReadTaskConfig>(client, p);
        ASSERT_NIL(p.error());
        this->devs = std::make_shared<device::MockManager>();
    }
};

/// @brief it should claim the device on start and release it on stop, so a device
/// that disconnects and returns between runs gets a fresh handle on the next start.
TEST_F(SourceClaimTest, testUnarySourceClaimsDeviceOnStart) {
    parse_config(true);
    UnarySource source(devs, std::move(*cfg));
    EXPECT_EQ(devs->acquire_call_count, 0);
    EXPECT_EQ(devs->dev.use_count(), 1);
    ASSERT_NIL(source.start());
    EXPECT_EQ(devs->acquire_call_count, 1);
    EXPECT_EQ(devs->dev.use_count(), 2);
    ASSERT_NIL(source.stop());
    EXPECT_EQ(devs->dev.use_count(), 1);
    ASSERT_NIL(source.start());
    EXPECT_EQ(devs->acquire_call_count, 2);
    ASSERT_NIL(source.stop());
}

/// @brief it should surface an acquire failure on start and retry the acquisition
/// on the next start.
TEST_F(SourceClaimTest, testUnarySourceAcquireFailureRetriesOnNextStart) {
    parse_config(true);
    devs->acquire_errors = {UNREACHABLE};
    UnarySource source(devs, std::move(*cfg));
    ASSERT_OCCURRED_AS(source.start(), ljm::UNREACHABLE);
    EXPECT_EQ(devs->dev.use_count(), 1);
    ASSERT_NIL(source.start());
    EXPECT_EQ(devs->acquire_call_count, 2);
    ASSERT_NIL(source.stop());
}

/// @brief it should claim the device on start and release it on stop.
TEST_F(SourceClaimTest, testStreamSourceClaimsDeviceOnStart) {
    parse_config(false);
    StreamSource source(devs, std::move(*cfg));
    EXPECT_EQ(devs->acquire_call_count, 0);
    EXPECT_EQ(devs->dev.use_count(), 1);
    ASSERT_NIL(source.start());
    EXPECT_EQ(devs->acquire_call_count, 1);
    EXPECT_EQ(devs->dev.use_count(), 2);
    ASSERT_NIL(source.stop());
    EXPECT_EQ(devs->dev.use_count(), 1);
    ASSERT_NIL(source.start());
    EXPECT_EQ(devs->acquire_call_count, 2);
    ASSERT_NIL(source.stop());
}

/// @brief it should surface an acquire failure on start and retry the acquisition
/// on the next start.
TEST_F(SourceClaimTest, testStreamSourceAcquireFailureRetriesOnNextStart) {
    parse_config(false);
    devs->acquire_errors = {UNREACHABLE};
    StreamSource source(devs, std::move(*cfg));
    ASSERT_OCCURRED_AS(source.start(), ljm::UNREACHABLE);
    EXPECT_EQ(devs->dev.use_count(), 1);
    ASSERT_NIL(source.start());
    EXPECT_EQ(devs->acquire_call_count, 2);
    ASSERT_NIL(source.stop());
}

/// @brief it should claim the device again on restart.
TEST_F(SourceClaimTest, testStreamSourceRestartReclaimsDevice) {
    parse_config(false);
    StreamSource source(devs, std::move(*cfg));
    ASSERT_NIL(source.start());
    EXPECT_EQ(devs->acquire_call_count, 1);
    ASSERT_NIL(source.restart());
    EXPECT_EQ(devs->acquire_call_count, 2);
    EXPECT_EQ(devs->dev.use_count(), 2);
    ASSERT_NIL(source.stop());
    EXPECT_EQ(devs->dev.use_count(), 1);
}

/// @brief a restart must drop its claim before acquiring. The manager shares one
/// handle per device while a holder is alive, so a source that keeps the old claim
/// across the acquire gets the dead handle back instead of a reopened one.
TEST_F(SourceClaimTest, testStreamSourceRestartOpensAFreshHandle) {
    parse_config(false);
    size_t opens = 0;
    devs->open = [&opens] {
        ++opens;
        return std::make_shared<device::Mock>();
    };
    StreamSource source(devs, std::move(*cfg));
    ASSERT_NIL(source.start());
    EXPECT_EQ(opens, 1);
    ASSERT_NIL(source.restart());
    EXPECT_EQ(opens, 2);
    ASSERT_NIL(source.stop());
}

/// @brief a read after a restart that failed to acquire must claim the device again
/// instead of dereferencing a released one.
TEST_F(SourceClaimTest, testStreamSourceReadReclaimsAfterFailedRestart) {
    parse_config(false);
    devs->acquire_errors = {
        x::errors::NIL,
        x::errors::Error(ljm::CRITICAL_ERROR, "device not found")
    };
    StreamSource source(devs, std::move(*cfg));
    ASSERT_NIL(source.start());
    ASSERT_OCCURRED_AS(source.restart(), ljm::CRITICAL_ERROR);

    x::breaker::Breaker breaker(x::breaker::default_config("read"));
    breaker.start();
    x::telem::Frame fr(0);
    const auto res = source.read(breaker, fr);
    breaker.stop();
    ASSERT_NIL(res.error);
    EXPECT_EQ(devs->acquire_call_count, 3);
    ASSERT_NIL(source.stop());
}

/// @brief it should hold no device after a failed restart, so a later stop cannot end
/// the stream of another task on the shared handle.
TEST_F(SourceClaimTest, testStreamSourceFailedRestartHoldsNoDevice) {
    parse_config(false);
    const auto mock = std::make_shared<device::Mock>();
    devs->dev = mock;
    StreamSource source(devs, std::move(*cfg));
    ASSERT_NIL(source.start());
    EXPECT_EQ(mock.use_count(), 3);
    mock->set_should_fail(true);
    ASSERT_OCCURRED_AS(source.restart(), x::errors::Error("mock failure"));
    EXPECT_EQ(mock.use_count(), 2);
}

/// @brief it should refuse a second stream on one device, write nothing to the device,
/// and leave the first stream running.
TEST_F(SourceClaimTest, testStreamSourceSecondStartLeavesFirstStreamRunning) {
    parse_config(false);
    auto first_cfg = std::move(this->cfg);
    parse_config(false);
    const auto mock = std::make_shared<device::Mock>();
    devs->dev = mock;
    StreamSource first(devs, std::move(*first_cfg));
    StreamSource second(devs, std::move(*cfg));
    ASSERT_NIL(first.start());
    const auto writes = mock->write_count();
    EXPECT_GT(writes, 0);
    ASSERT_OCCURRED_AS(second.start(), ljm::STREAM_IS_ACTIVE);
    EXPECT_EQ(mock->write_count(), writes);
    ASSERT_NIL(second.stop());
    ASSERT_OCCURRED_AS(second.start(), ljm::STREAM_IS_ACTIVE);
    ASSERT_NIL(first.stop());
    ASSERT_NIL(second.start());
    ASSERT_NIL(second.stop());
}

/// @brief reads once from the source with a running breaker.
common::ReadResult read_once(common::Source &source, x::telem::Frame &fr) {
    x::breaker::Breaker breaker(x::breaker::default_config("read"));
    breaker.start();
    auto res = source.read(breaker, fr);
    breaker.stop();
    return res;
}

/// @brief it should release the device after a temporary read error, then claim it and
/// apply the channel settings again on the next read.
TEST_F(SourceClaimTest, testUnarySourceReconnectsAfterTemporaryError) {
    parse_config(false);
    const auto mock = std::make_shared<device::Mock>();
    devs->dev = mock;
    mock->read_errors = {UNREACHABLE};
    UnarySource source(devs, std::move(*cfg));
    ASSERT_NIL(source.start());
    const auto writes = mock->write_count();
    x::telem::Frame fr(0);
    ASSERT_OCCURRED_AS(read_once(source, fr).error, ljm::UNREACHABLE);
    EXPECT_EQ(mock.use_count(), 2);
    ASSERT_NIL(read_once(source, fr).error);
    EXPECT_EQ(devs->acquire_call_count, 2);
    EXPECT_EQ(mock->write_count(), 2 * writes);
    ASSERT_NIL(source.stop());
}

/// @brief it should read one value of each channel per sample, for the samples of
/// one stream period.
TEST_F(SourceClaimTest, testUnarySourceReadsSamplesPerChan) {
    parse_config(false);
    const auto mock = std::make_shared<device::Mock>();
    mock->read_values = {{"AIN0", 1.5}};
    devs->dev = mock;
    UnarySource source(devs, std::move(*cfg));
    ASSERT_NIL(source.start());
    x::telem::Frame fr(0);
    ASSERT_NIL(read_once(source, fr).error);
    ASSERT_EQ(fr.size(), 1);
    const auto &s = fr.series->at(0);
    ASSERT_EQ(s.size(), 2);
    EXPECT_EQ(s.at<double>(0), 1.5);
    EXPECT_EQ(s.at<double>(1), 1.5);
    ASSERT_NIL(source.stop());
}

/// @brief it should keep the samples and warn while the device uses its default
/// calibration.
TEST_F(SourceClaimTest, testUnarySourceWarnsOfDefaultCalibration) {
    parse_config(false);
    const auto mock = std::make_shared<device::Mock>();
    mock->read_values = {{"AIN0", 1.5}};
    mock->invalid_calibration = true;
    devs->dev = mock;
    UnarySource source(devs, std::move(*cfg));
    ASSERT_NIL(source.start());
    x::telem::Frame fr(0);
    const auto res = read_once(source, fr);
    ASSERT_NIL(res.error);
    EXPECT_EQ(res.warning.message, "Using default calibration");
    EXPECT_EQ(fr.series->at(0).at<double>(0), 1.5);
    ASSERT_NIL(source.stop());
}

/// @brief it should keep the scans and warn while the device uses its default
/// calibration, and warn of skipped samples first.
TEST_F(SourceClaimTest, testStreamSourceWarnsOfDefaultCalibration) {
    parse_config(false);
    const auto mock = std::make_shared<device::Mock>();
    mock->stream_data = {1, 2};
    mock->invalid_calibration = true;
    devs->dev = mock;
    StreamSource source(devs, std::move(*cfg));
    ASSERT_NIL(source.start());
    x::telem::Frame fr(0);
    auto res = read_once(source, fr);
    ASSERT_NIL(res.error);
    EXPECT_EQ(res.warning.message, "Using default calibration");
    EXPECT_EQ(fr.series->at(0).size(), 2);
    mock->stream_data = {LJM_DUMMY_VALUE, 2};
    res = read_once(source, fr);
    ASSERT_NIL(res.error);
    EXPECT_EQ(res.warning.message, "Skipped samples");
    ASSERT_NIL(source.stop());
}

/// @brief it should stop the stream after a temporary read error, then start it again
/// on the next read.
TEST_F(SourceClaimTest, testStreamSourceRestartsAfterTemporaryError) {
    parse_config(false);
    const auto mock = std::make_shared<device::Mock>();
    mock->stream_data = {1, 2};
    mock->stream_read_errors = {UNREACHABLE};
    devs->dev = mock;
    StreamSource source(devs, std::move(*cfg));
    ASSERT_NIL(source.start());
    x::telem::Frame fr(0);
    ASSERT_OCCURRED_AS(read_once(source, fr).error, ljm::UNREACHABLE);
    EXPECT_FALSE(mock->is_streaming());
    EXPECT_EQ(mock.use_count(), 2);
    ASSERT_NIL(read_once(source, fr).error);
    EXPECT_EQ(mock->stream_start_count(), 2);
    EXPECT_TRUE(mock->is_streaming());
    ASSERT_NIL(source.stop());
}

/// @brief it should drop the scans that LJM fills with its dummy value and warn that
/// the device skipped samples.
TEST_F(SourceClaimTest, testStreamSourceDropsSkippedScans) {
    parse_config(false);
    const auto mock = std::make_shared<device::Mock>();
    mock->stream_data = {LJM_DUMMY_VALUE, 2};
    devs->dev = mock;
    StreamSource source(devs, std::move(*cfg));
    ASSERT_NIL(source.start());
    x::telem::Frame fr(0);
    const auto res = read_once(source, fr);
    ASSERT_NIL(res.error);
    EXPECT_EQ(res.warning.message, "Skipped samples");
    ASSERT_EQ(fr.size(), 1);
    const auto &s = fr.series->at(0);
    ASSERT_EQ(s.size(), 1);
    EXPECT_EQ(s.at<double>(0), 2);
    ASSERT_NIL(source.stop());
}

/// @brief it should write no frame when the device skipped every scan of a read.
TEST_F(SourceClaimTest, testStreamSourceWritesNothingWhenEveryScanSkipped) {
    parse_config(false);
    const auto mock = std::make_shared<device::Mock>();
    mock->stream_data = {LJM_DUMMY_VALUE, LJM_DUMMY_VALUE};
    devs->dev = mock;
    StreamSource source(devs, std::move(*cfg));
    ASSERT_NIL(source.start());
    x::telem::Frame fr(0);
    const auto res = read_once(source, fr);
    ASSERT_NIL(res.error);
    EXPECT_EQ(res.warning.message, "Skipped samples");
    EXPECT_TRUE(fr.empty());
    ASSERT_NIL(source.stop());
}

class ReadTaskTest : public ::testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> client = std::make_shared<synnax::Synnax>(
        new_test_client()
    );
    std::shared_ptr<device::Mock> dev = std::make_shared<device::Mock>();
    std::shared_ptr<device::MockManager> devs = std::make_shared<device::MockManager>(
        this->dev
    );

    void SetUp() override { this->dev->read_values["FIRMWARE_VERSION"] = 1.0061; }

    /// @brief returns a K-type thermocouple channel on the port, in Kelvin.
    x::json::json thermocouple(
        const std::string &port,
        const std::string &cjc_source = DEVICE_CJC_SOURCE
    ) {
        const auto ch = ASSERT_NIL_P(this->client->channels.create(
            make_unique_channel_name("tc"),
            x::telem::FLOAT64_T,
            true
        ));
        return {
            {"port", port},
            {"channel", ch.key},
            {"type", "thermocouple"},
            {"scale", {{"type", "none"}}},
            {"thermocouple_type", "K"},
            {"units", "K"},
            {"cjc_source", cjc_source},
            {"cjc_slope", 1},
            {"cjc_offset", 0},
        };
    }

    /// @brief returns an analog channel on the port.
    x::json::json analog(const std::string &port) {
        const auto ch = ASSERT_NIL_P(this->client->channels.create(
            make_unique_channel_name("ai"),
            x::telem::FLOAT64_T,
            true
        ));
        return {
            {"port", port},
            {"channel", ch.key},
            {"type", "analog"},
            {"range", 10},
            {"scale", {{"type", "none"}}},
        };
    }

    /// @brief returns a parser of a task on a new device of the model.
    x::json::Parser parser(
        const std::string &model,
        const x::json::json &channels,
        const int sample_rate = 10,
        const int stream_rate = 5
    ) {
        const auto rack = ASSERT_NIL_P(this->client->racks.create("tc"));
        synnax::device::Device device{
            .key = make_unique_channel_name("labjack"),
            .rack = rack.key,
            .location = "dev1",
            .make = "labjack",
            .model = model,
            .name = "my_device",
        };
        const auto create_err = this->client->devices.create(device);
        EXPECT_NIL(create_err);
        auto j = basic_read_task_config(device.key);
        j["channels"] = channels;
        j["sample_rate"] = sample_rate;
        j["stream_rate"] = stream_rate;
        return x::json::Parser(j);
    }

    /// @brief parses a task on a new device of the model.
    std::pair<ReadTaskConfig, x::errors::Error> parse(
        const std::string &model,
        const x::json::json &channels,
        const int sample_rate = 10,
        const int stream_rate = 5
    ) {
        auto p = this->parser(model, channels, sample_rate, stream_rate);
        ReadTaskConfig cfg(this->client, p);
        return {std::move(cfg), p.error()};
    }
};

/// @brief it should read the device CJC sensor after the channels on a T7, and poll.
TEST_F(ReadTaskTest, testT7ReadsDeviceSensor) {
    const auto cfg = ASSERT_NIL_P(
        this->parse(T7, x::json::json::array({thermocouple("AIN0"), analog("AIN1")}))
    );
    const std::vector<std::string> expected{"AIN0", "AIN1", "TEMPERATURE_DEVICE_K"};
    EXPECT_EQ(cfg.registers, expected);
    ASSERT_EQ(cfg.thermocouples.size(), 1);
    EXPECT_EQ(cfg.thermocouples[0].second->cjc_index, 2);
    EXPECT_TRUE(cfg.polled());
}

/// @brief it should read the sensor at the terminal of each thermocouple on a T8, and
/// stream.
TEST_F(ReadTaskTest, testT8ReadsTerminalSensors) {
    const auto cfg = ASSERT_NIL_P(this->parse(
        T8,
        x::json::json::array({thermocouple("AIN3"), thermocouple("AIN5")})
    ));
    const std::vector<std::string> expected{
        "AIN3",
        "AIN5",
        "TEMPERATURE3",
        "TEMPERATURE5"
    };
    EXPECT_EQ(cfg.registers, expected);
    EXPECT_FALSE(cfg.polled());
}

/// @brief it should read a CJC port once when a channel of the task also reads it.
TEST_F(ReadTaskTest, testSharesCJCPortWithChannel) {
    const auto cfg = ASSERT_NIL_P(this->parse(
        T7,
        x::json::json::array({thermocouple("AIN0", "AIN2"), analog("AIN2")})
    ));
    const std::vector<std::string> expected{"AIN0", "AIN2"};
    EXPECT_EQ(cfg.registers, expected);
    EXPECT_EQ(cfg.thermocouples[0].second->cjc_index, 1);
}

/// @brief it should reject a T8 task with thermocouples above 250 Hz, and allow the
/// same rate on a T7.
TEST_F(ReadTaskTest, testT8RateLimit) {
    auto p = this->parser(T8, x::json::json::array({thermocouple("AIN0")}), 500);
    const ReadTaskConfig cfg(this->client, p);
    ASSERT_OCCURRED_AS(p.error(), x::errors::VALIDATION);
    const auto err = p.error_json()["errors"][0];
    EXPECT_EQ(err["path"], "sample_rate");
    EXPECT_EQ(err["message"], "A T8 task with thermocouples samples at 250 Hz or less");
    ASSERT_NIL_P(this->parse(T7, x::json::json::array({thermocouple("AIN0")}), 500));
}

/// @brief it should reject a thermocouple whose CJC source is the port of another
/// thermocouple.
TEST_F(ReadTaskTest, testRejectsThermocoupleAsCJCSource) {
    auto p = this->parser(
        T7,
        x::json::json::array({thermocouple("AIN2"), thermocouple("AIN0", "AIN2")})
    );
    const ReadTaskConfig cfg(this->client, p);
    ASSERT_OCCURRED_AS(p.error(), x::errors::VALIDATION);
    const auto err = p.error_json()["errors"][0];
    EXPECT_EQ(err["path"], "channels");
    EXPECT_EQ(
        err["message"],
        "AIN2 is a thermocouple channel, so it cannot be the CJC source of AIN0"
    );
}

/// @brief it should convert each polled thermocouple voltage with the CJC reading of
/// the same request.
TEST_F(ReadTaskTest, testUnarySourceConverts) {
    auto cfg = ASSERT_NIL_P(
        this->parse(T7, x::json::json::array({thermocouple("AIN0")}))
    );
    this->dev->read_values = {{"AIN0", 0.001}, {"TEMPERATURE_DEVICE_K", 300}};
    UnarySource source(this->devs, std::move(cfg));
    ASSERT_NIL(source.start());
    x::telem::Frame fr(0);
    const auto res = read_once(source, fr);
    ASSERT_NIL(res.error);
    EXPECT_TRUE(res.warning.empty());
    const auto &s = fr.series->at(0);
    ASSERT_EQ(s.size(), 2);
    EXPECT_DOUBLE_EQ(s.at<double>(0), 325);
    EXPECT_DOUBLE_EQ(s.at<double>(1), 325);
    ASSERT_NIL(source.stop());
}

/// @brief it should write NaN and warn for a thermocouple out of range.
TEST_F(ReadTaskTest, testUnarySourceWarnsOutOfRange) {
    auto cfg = ASSERT_NIL_P(
        this->parse(T7, x::json::json::array({thermocouple("AIN0")}))
    );
    this->dev->read_values = {{"AIN0", 9.9}, {"TEMPERATURE_DEVICE_K", 300}};
    UnarySource source(this->devs, std::move(cfg));
    ASSERT_NIL(source.start());
    x::telem::Frame fr(0);
    const auto res = read_once(source, fr);
    ASSERT_NIL(res.error);
    EXPECT_EQ(res.warning.message, "Thermocouple out of range");
    EXPECT_EQ(
        res.warning.description,
        "AIN0 reads a voltage outside the range of its thermocouple type. Check for an "
        "open thermocouple or a loose wire. The Driver writes NaN until the voltage is "
        "in range."
    );
    EXPECT_TRUE(std::isnan(fr.series->at(0).at<double>(0)));
    ASSERT_NIL(source.stop());
}

/// @brief it should stream each thermocouple with its terminal sensor on a T8, and
/// write only the channels to the frame.
TEST_F(ReadTaskTest, testStreamSourceConverts) {
    auto cfg = ASSERT_NIL_P(
        this->parse(T8, x::json::json::array({thermocouple("AIN3")}))
    );
    this->dev->stream_data = {0.001, 300, 0.002, 301};
    StreamSource source(this->devs, std::move(cfg));
    ASSERT_NIL(source.start());
    x::telem::Frame fr(0);
    ASSERT_NIL(read_once(source, fr).error);
    ASSERT_EQ(fr.size(), 1);
    const auto &s = fr.series->at(0);
    ASSERT_EQ(s.size(), 2);
    EXPECT_DOUBLE_EQ(s.at<double>(0), 325);
    EXPECT_DOUBLE_EQ(s.at<double>(1), 351);
    ASSERT_NIL(source.stop());
}

/// @brief it should refuse to stream thermocouples from a T8 with old firmware.
TEST_F(ReadTaskTest, testT8OldFirmware) {
    auto cfg = ASSERT_NIL_P(
        this->parse(T8, x::json::json::array({thermocouple("AIN0")}))
    );
    this->dev->read_values["FIRMWARE_VERSION"] = 1.0025;
    StreamSource source(this->devs, std::move(cfg));
    const auto err = source.start();
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(
        err.data,
        "Thermocouples on a T8 need firmware 1.0038 or later, but the device has "
        "firmware 1.0025. Update the firmware with Kipling."
    );
    EXPECT_FALSE(this->dev->is_streaming());
}

/// @brief it should refuse to stream thermocouples from a T8 with an old LJM.
TEST_F(ReadTaskTest, testT8OldLJM) {
    auto cfg = ASSERT_NIL_P(
        this->parse(T8, x::json::json::array({thermocouple("AIN0")}))
    );
    this->dev->library_version = 1.2302;
    StreamSource source(this->devs, std::move(cfg));
    const auto err = source.start();
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    EXPECT_EQ(
        err.data,
        "Thermocouples on a T8 need LJM 1.2304 or later, but the Driver loaded LJM "
        "1.2302. Install the latest LJM."
    );
}

/// @brief it should keep the timestamps of the scans after a skipped scan in place, so
/// the gap of the skipped scan stays in the index.
TEST_F(ReadTaskTest, testStreamSourceKeepsGapOfSkippedScan) {
    const auto index = ASSERT_NIL_P(
        this->client->channels
            .create(make_unique_channel_name("time"), x::telem::TIMESTAMP_T, 0, true)
    );
    const auto data = ASSERT_NIL_P(this->client->channels.create(
        make_unique_channel_name("ai"),
        x::telem::FLOAT64_T,
        index.key
    ));
    auto channel = analog("AIN0");
    channel["channel"] = data.key;
    auto cfg = ASSERT_NIL_P(this->parse(T7, x::json::json::array({channel}), 20, 5));
    this->dev->stream_data = {1, LJM_DUMMY_VALUE, 3, 4};
    StreamSource source(this->devs, std::move(cfg));
    ASSERT_NIL(source.start());
    x::telem::Frame fr(0);
    ASSERT_NIL(read_once(source, fr).error);
    ASSERT_EQ(fr.size(), 2);
    ASSERT_EQ(fr.series->at(0).size(), 3);
    EXPECT_EQ(fr.at<double>(data.key, 1), 3);
    const auto t0 = fr.at<x::telem::TimeStamp>(index.key, 0);
    const auto t2 = fr.at<x::telem::TimeStamp>(index.key, 1);
    const auto t3 = fr.at<x::telem::TimeStamp>(index.key, 2);
    EXPECT_EQ(t2 - t0, (t3 - t2) * 2);
    ASSERT_NIL(source.stop());
}
}
