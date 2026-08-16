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

#include "driver/labjack/read_task.h"

namespace driver::labjack {
/// @brief it should parse analog input channel configuration.
TEST(TestReadChannelParse, testAIChan) {
    const x::json::json cfg{
        {"port", "AIN0"},
        {"disabled", false},
        {"key", "8hYJO9zt6eS"},
        {"channel", 1},
        {"type", "analog"},
        {"range", 5},
        {"scale", {{"type", "linear"}, {"slope", 1}, {"offset", 2}}}
    };
    auto p = x::json::Parser(cfg);
    const auto chan = parse_input_chan(p);
    ASSERT_NIL(p.error());
    const auto ai_chan = dynamic_cast<AIChan *>(chan.get());
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
        {"key", "8hYJO9zt6eS"},
        {"channel", 1},
        {"type", "digital"}
    };
    auto p = x::json::Parser(cfg);
    const auto chan = parse_input_chan(p);
    ASSERT_NIL(p.error());
    const auto di_chan = dynamic_cast<DIChan *>(chan.get());
    ASSERT_NE(di_chan, nullptr);
    ASSERT_EQ(di_chan->port, "DIO0");
    ASSERT_EQ(di_chan->enabled, true);
    ASSERT_EQ(di_chan->synnax_key, 1);
}

/// @brief it should parse thermocouple channel configuration.
TEST(TestReadChannelParse, testTCChan) {
    const x::json::json cfg{
        {"port", "AIN0"},
        {"disabled", false},
        {"key", "8hYJO9zt6eS"},
        {"channel", 0},
        {"type", "thermocouple"},
        {"range", 0},
        {"scale", {{"type", "linear"}, {"slope", 1}, {"offset", 2}}},
        {"thermocouple_type", "K"},
        {"pos_chan", 0},
        {"neg_chan", 199},
        {"units", "K"},
        {"cjc_source", "TEMPERATURE_DEVICE_K"},
        {"cjc_slope", 1},
        {"cjc_offset", 0}
    };
    auto p = x::json::Parser(cfg);
    const auto chan = parse_input_chan(p);
    ASSERT_NIL(p.error());
    const auto tc_chan = dynamic_cast<ThermocoupleChan *>(chan.get());
    ASSERT_NE(tc_chan, nullptr);
    ASSERT_EQ(tc_chan->port, "AIN0_EF_READ_A");
    ASSERT_EQ(tc_chan->enabled, true);
    ASSERT_EQ(tc_chan->synnax_key, 0);
    ASSERT_EQ(tc_chan->type, LJM_ttK);
    ASSERT_EQ(tc_chan->pos_chan, 0);
    ASSERT_EQ(tc_chan->neg_chan, 199);
    ASSERT_EQ(tc_chan->units, LJM_KELVIN);
    ASSERT_EQ(tc_chan->cjc_addr, LJM_TEMPERATURE_DEVICE_K_ADDRESS);
    ASSERT_EQ(tc_chan->cjc_slope, 1);
    ASSERT_EQ(tc_chan->cjc_offset, 0);
}

/// @brief it should reject invalid channel type in configuration.
TEST(TestReadChannelParse, testInvalidChannelType) {
    const x::json::json cfg{
        {"port", "AIN0"},
        {"disabled", false},
        {"key", "8hYJO9zt6eS"},
        {"channel", 1},
        {"type", "INVALID_TYPE"}, // Invalid channel type
        {"range", 5},
        {"scale", {{"type", "linear"}, {"slope", 1}, {"offset", 2}}}
    };
    auto p = x::json::Parser(cfg);
    const auto chan = parse_input_chan(p);
    ASSERT_OCCURRED_AS(p.error(), x::errors::VALIDATION);
}

/// @brief it should derive the backlog threshold from the sample rate when the
/// count is zero.
TEST(TestBacklogWarnOnCount, testZeroDerivesFromSampleRate) {
    const x::json::json cfg{{"device_scan_backlog_warn_on_count", 0}};
    auto p = x::json::Parser(cfg);
    const auto count = parse_backlog_warn_on_count(
        p,
        "device_scan_backlog_warn_on_count",
        x::telem::HERTZ * 50,
        2
    );
    ASSERT_NIL(p.error());
    ASSERT_EQ(count, 100);
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

x::json::json basic_read_task_config() {
    return {
        {"device", "230227d9-02aa-47e4-b370-0d590add1bc1"},
        {"sample_rate", 10},
        {"stream_rate", 5},
        {"data_saving_disabled", false},
        {"channels",
         x::json::json::array(
             {{{"port", "AIN0"},
               {"disabled", false},
               {"key", "8hYJO9zt6eS"},
               {"channel", 0},
               {"type", "thermocouple"},
               {"range", 0},
               {"scale", {{"type", "linear"}, {"slope", 1}, {"offset", 2}}},
               {"thermocouple_type", "K"},
               {"pos_chan", 0},
               {"neg_chan", 199},
               {"units", "K"},
               {"cjc_source", "TEMPERATURE_DEVICE_K"},
               {"cjc_slope", 1},
               {"cjc_offset", 0}},
              {{"port", "DIO4"},
               {"disabled", false},
               {"key", "DYFpBBDlpRt"},
               {"channel", 0},
               {"type", "digital"}},
              {{"port", "AIN6"},
               {"disabled", false},
               {"key", "rHb0YjmhUq3"},
               {"channel", 0},
               {"type", "analog"},
               {"range", 0},
               {"scale", {{"type", "none"}}}}}
         )}
    };
}

/// @brief it should parse basic read task configuration with multiple channels.
TEST(TestReadTaskConfigParse, testBasicReadTaskConfigParse) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("cat"));
    auto dev = synnax::device::Device{
        .key = "230227d9-02aa-47e4-b370-0d590add1bc1",
        .rack = rack.key,
        .location = "dev1",
        .make = "labjack",
        .model = "T7",
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

    auto j = basic_read_task_config();
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

    const auto tc_chan = dynamic_cast<ThermocoupleChan *>(cfg->channels[0].get());
    ASSERT_NE(tc_chan, nullptr);
    ASSERT_EQ(tc_chan->port, "AIN0_EF_READ_A");
    ASSERT_EQ(tc_chan->enabled, true);
    ASSERT_EQ(tc_chan->synnax_key, tc_ch.key);
    ASSERT_EQ(tc_chan->type, LJM_ttK);
    ASSERT_EQ(tc_chan->pos_chan, 0);
    ASSERT_EQ(tc_chan->neg_chan, 199);
    ASSERT_EQ(tc_chan->units, LJM_KELVIN);
    ASSERT_EQ(tc_chan->cjc_addr, LJM_TEMPERATURE_DEVICE_K_ADDRESS);
    ASSERT_EQ(tc_chan->cjc_slope, 1);
    ASSERT_EQ(tc_chan->cjc_offset, 0);

    const auto di_chan = dynamic_cast<DIChan *>(cfg->channels[1].get());
    ASSERT_NE(di_chan, nullptr);
    ASSERT_EQ(di_chan->port, "DIO4");
    ASSERT_EQ(di_chan->enabled, true);
    ASSERT_EQ(di_chan->synnax_key, di_ch.key);

    const auto ai_chan = dynamic_cast<AIChan *>(cfg->channels[2].get());
    ASSERT_NE(ai_chan, nullptr);
    ASSERT_EQ(ai_chan->port, "AIN6");
    ASSERT_EQ(ai_chan->enabled, true);
    ASSERT_EQ(ai_chan->synnax_key, ai_ch.key);
    ASSERT_EQ(ai_chan->range, 0);
}

/// @brief it should reject invalid channel type in read task configuration.
TEST(TestReadTaskConfigParse, testInvalidChannelTypeInConfig) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("cat"));
    auto dev = synnax::device::Device{
        .key = "230227d9-02aa-47e4-b370-0d590add1bc1",
        .rack = rack.key,
        .location = "dev1",
        .make = "labjack",
        .model = "T7",
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
    auto j = basic_read_task_config();
    j["channels"] = x::json::json::array(
        {{{"port", "AIN0"},
          {"disabled", false},
          {"key", "8hYJO9zt6eS"},
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
        .key = "230227d9-02aa-47e4-b370-0d590add1bc1",
        .rack = rack.key,
        .location = "dev1",
        .make = "labjack",
        .model = "T7",
        .name = "test_device",
    };
    ASSERT_NIL(client->devices.create(dev));
    auto ch = ASSERT_NIL_P(client->channels.create(
        make_unique_channel_name("test_channel"),
        x::telem::FLOAT64_T,
        true
    ));

    auto j = basic_read_task_config();
    j["data_saving_disabled"] = false;
    j["channels"] = x::json::json::array(
        {{{"port", "AIN0"},
          {"disabled", false},
          {"key", "8hYJO9zt6eS"},
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
}
