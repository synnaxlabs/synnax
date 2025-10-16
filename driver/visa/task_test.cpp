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

#include "driver/visa/read_task.h"
#include "driver/visa/write_task.h"

using namespace visa;

TEST(VISAReadTaskConfigTest, testBasicReadTaskConfigParse) {
    auto sy = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(sy->hardware.create_rack("visa_rack"));

    auto dev = synnax::Device(
        "visa-device-1",
        "my_visa_device",
        rack.key,
        "dev1",
        "visa",
        "Keysight 34465A",
        json{
            {"connection", {
                {"resource_name", "TCPIP0::192.168.1.100::INSTR"},
                {"timeout_ms", 5000},
                {"term_char", "\n"},
                {"term_char_enabled", true}
            }}
        }.dump()
    );
    ASSERT_NIL(sy->hardware.create_device(dev));

    // Create channels
    auto voltage_ch = ASSERT_NIL_P(
        sy->channels.create("voltage", telem::FLOAT64_T, true)
    );
    auto current_ch = ASSERT_NIL_P(
        sy->channels.create("current", telem::FLOAT64_T, true)
    );

    // Build config JSON
    json cfg{
        {"device", "visa-device-1"},
        {"sample_rate", 10},
        {"stream_rate", 5},
        {"data_saving", true},
        {"channels", json::array({
            {
                {"channel", voltage_ch.key},
                {"scpi_command", "MEAS:VOLT:DC?"},
                {"format", "float"},
                {"enabled", true}
            },
            {
                {"channel", current_ch.key},
                {"scpi_command", "MEAS:CURR:DC?"},
                {"format", "float"},
                {"enabled", true}
            }
        })}
    };

    // Parse config
    synnax::Task task;
    task.config = cfg.dump();
    auto [read_cfg, err] = ReadTaskConfig::parse(sy, task);
    ASSERT_NIL(err);

    // Validate parsed config
    ASSERT_EQ(read_cfg.sample_rate, telem::HERTZ * 10);
    ASSERT_EQ(read_cfg.stream_rate, telem::HERTZ * 5);
    ASSERT_EQ(read_cfg.data_saving, true);
    ASSERT_EQ(read_cfg.channels.size(), 2);
    ASSERT_EQ(read_cfg.data_channel_count, 2);

    // Check first channel (voltage)
    ASSERT_EQ(read_cfg.channels[0].synnax_key, voltage_ch.key);
    ASSERT_EQ(read_cfg.channels[0].scpi_command, "MEAS:VOLT:DC?");
    ASSERT_EQ(read_cfg.channels[0].format, channel::ResponseFormat::FLOAT);
    ASSERT_TRUE(read_cfg.channels[0].enabled);
    ASSERT_EQ(read_cfg.channels[0].ch.key, voltage_ch.key);

    // Check second channel (current)
    ASSERT_EQ(read_cfg.channels[1].synnax_key, current_ch.key);
    ASSERT_EQ(read_cfg.channels[1].scpi_command, "MEAS:CURR:DC?");
    ASSERT_EQ(read_cfg.channels[1].format, channel::ResponseFormat::FLOAT);
    ASSERT_TRUE(read_cfg.channels[1].enabled);
    ASSERT_EQ(read_cfg.channels[1].ch.key, current_ch.key);

    // Verify connection config
    ASSERT_EQ(read_cfg.conn.resource_name, "TCPIP0::192.168.1.100::INSTR");
    ASSERT_EQ(read_cfg.conn.timeout_ms, 5000);
}

TEST(VISAReadTaskConfigTest, testDisabledChannelNotCountedInDataChannelCount) {
    auto sy = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(sy->hardware.create_rack("visa_rack"));

    auto dev = synnax::Device(
        "visa-device-2",
        "my_visa_device",
        rack.key,
        "dev2",
        "visa",
        "Keysight 34465A",
        json{{"connection", {{"resource_name", "TCPIP0::192.168.1.100::INSTR"}}}}.dump()
    );
    ASSERT_NIL(sy->hardware.create_device(dev));

    auto ch1 = ASSERT_NIL_P(sy->channels.create("ch1", telem::FLOAT64_T, true));
    auto ch2 = ASSERT_NIL_P(sy->channels.create("ch2", telem::FLOAT64_T, true));
    auto ch3 = ASSERT_NIL_P(sy->channels.create("ch3", telem::FLOAT64_T, true));

    json cfg{
        {"device", "visa-device-2"},
        {"sample_rate", 10},
        {"stream_rate", 10},
        {"channels", json::array({
            {{"channel", ch1.key}, {"scpi_command", "MEAS:VOLT:DC?"}, {"format", "float"}, {"enabled", true}},
            {{"channel", ch2.key}, {"scpi_command", "MEAS:CURR:DC?"}, {"format", "float"}, {"enabled", false}},
            {{"channel", ch3.key}, {"scpi_command", "MEAS:RES?"}, {"format", "float"}, {"enabled", true}}
        })}
    };

    synnax::Task task;
    task.config = cfg.dump();
    auto [read_cfg, err] = ReadTaskConfig::parse(sy, task);
    ASSERT_NIL(err);

    // Only 2 channels enabled, so data_channel_count should be 2
    ASSERT_EQ(read_cfg.data_channel_count, 2);
    ASSERT_EQ(read_cfg.channels.size(), 3);
    ASSERT_TRUE(read_cfg.channels[0].enabled);
    ASSERT_FALSE(read_cfg.channels[1].enabled);
    ASSERT_TRUE(read_cfg.channels[2].enabled);
}

TEST(VISAReadTaskConfigTest, testResponseFormatParsing) {
    auto sy = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(sy->hardware.create_rack("visa_rack"));

    auto dev = synnax::Device(
        "visa-device-4",
        "my_visa_device",
        rack.key,
        "dev4",
        "visa",
        "Keysight 34465A",
        json{{"connection", {{"resource_name", "TCPIP0::192.168.1.100::INSTR"}}}}.dump()
    );
    ASSERT_NIL(sy->hardware.create_device(dev));

    auto float_ch = ASSERT_NIL_P(sy->channels.create("float", telem::FLOAT64_T, true));
    auto bool_ch = ASSERT_NIL_P(sy->channels.create("bool", telem::UINT8_T, true));
    auto str_ch = ASSERT_NIL_P(sy->channels.create("string", telem::STRING_T, true));

    json cfg{
        {"device", "visa-device-4"},
        {"sample_rate", 1},
        {"stream_rate", 1},
        {"channels", json::array({
            {{"channel", float_ch.key}, {"scpi_command", "MEAS:VOLT?"}, {"format", "float"}},
            {{"channel", bool_ch.key}, {"scpi_command", "SYST:BEEP?"}, {"format", "boolean"}},
            {{"channel", str_ch.key}, {"scpi_command", "SYST:ERR?"}, {"format", "string"}}
        })}
    };

    synnax::Task task;
    task.config = cfg.dump();
    auto [read_cfg, err] = ReadTaskConfig::parse(sy, task);
    ASSERT_NIL(err);

    ASSERT_EQ(read_cfg.channels[0].format, channel::ResponseFormat::FLOAT);
    ASSERT_EQ(read_cfg.channels[1].format, channel::ResponseFormat::BOOLEAN);
    ASSERT_EQ(read_cfg.channels[2].format, channel::ResponseFormat::STRING);
}
