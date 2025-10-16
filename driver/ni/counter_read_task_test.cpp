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
#include "driver/ni/counter_read_task.h"
#include "driver/ni/hardware/hardware.h"
#include "driver/pipeline/mock/pipeline.h"

/// @brief it should correctly parse a basic counter read task.
namespace {
json base_counter_config() {
    return {
        {"data_saving", false},
        {"sample_rate", 1000},
        {"stream_rate", 25},
        {"device", ""},
        {"channels",
         json::array({{
             {"type", "ci_frequency"},
             {"key", "ks1VnWdrSVA"},
             {"port", 0},
             {"enabled", true},
             {"name", ""},
             {"channel", ""}, // Will be overridden
             {"min_val", 0},
             {"max_val", 10000},
             {"units", "Hz"},
             {"edge", "Rising"},
             {"meas_method", "DynamicAvg"},
             {"terminal", "PFI0"},
             {"custom_scale", {{"type", "none"}}},
             {"device", ""} // Will be overridden
         }})}
    };
}
}

TEST(CounterReadTaskConfigTest, testBasicCounterReadTaskConfigParse) {
    auto sy = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(sy->hardware.create_rack("cat"));
    auto dev = synnax::Device(
        "abc123",
        "my_device",
        rack.key,
        "dev1",
        "ni",
        "PXI-6255",
        ""
    );
    ASSERT_NIL(sy->hardware.create_device(dev));
    auto ch = ASSERT_NIL_P(sy->channels.create("virtual", telem::FLOAT64_T, true));

    auto j = base_counter_config();
    j["device"] = dev.key;
    j["channels"][0]["device"] = dev.key;
    j["channels"][0]["channel"] = ch.key;

    auto p = xjson::Parser(j);
    auto cfg = std::make_unique<ni::CounterReadTaskConfig>(sy, p);
    ASSERT_NIL(p.error());
}

/// @brief it should return a validation error if the device does not exist.
TEST(CounterReadTaskConfigTest, testNonExistingCounterReadDevice) {
    auto sy = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(sy->hardware.create_rack("cat"));
    auto ch = ASSERT_NIL_P(sy->channels.create("virtual", telem::FLOAT64_T, true));

    auto j = base_counter_config();
    j["device"] = "definitely_not_an_existing_device";
    j["channels"][0]["device"] = "definitely_not_an_existing_device";
    j["channels"][0]["channel"] = ch.key;

    auto p = xjson::Parser(j);
    auto cfg = std::make_unique<ni::CounterReadTaskConfig>(sy, p);
    ASSERT_OCCURRED_AS(p.error(), xerrors::VALIDATION);
}

/// @brief it should return a validation error if the channel does not exist.
TEST(CounterReadTaskConfigTest, testNonExistentCounterReadChannel) {
    auto sy = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(sy->hardware.create_rack("cat"));
    auto dev = synnax::Device(
        "abc123",
        "my_device",
        rack.key,
        "dev1",
        "ni",
        "PXI-6255",
        ""
    );
    ASSERT_NIL(sy->hardware.create_device(dev));

    auto j = base_counter_config();
    j["device"] = dev.key;
    j["channels"][0]["device"] = dev.key;
    j["channels"][0]["channel"] = 12121212;

    auto p = xjson::Parser(j);
    auto cfg = std::make_unique<ni::CounterReadTaskConfig>(sy, p);
    ASSERT_OCCURRED_AS(p.error(), xerrors::VALIDATION);
}

/// @brief it should return a validation error if the sample rate is less than the
/// stream rate.
TEST(CounterReadTaskConfigTest, testSampleRateLessThanStreamRate) {
    auto sy = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(sy->hardware.create_rack("cat"));
    auto dev = synnax::Device(
        "abc123",
        "my_device",
        rack.key,
        "dev1",
        "ni",
        "PXI-6255",
        ""
    );
    auto dev_err = sy->hardware.create_device(dev);
    ASSERT_FALSE(dev_err) << dev_err;

    auto ch = ASSERT_NIL_P(sy->channels.create("virtual", telem::FLOAT64_T, true));

    auto j = base_counter_config();
    j["device"] = dev.key;
    j["channels"][0]["device"] = dev.key;
    j["channels"][0]["channel"] = ch.key;
    j["sample_rate"] = 10;
    j["stream_rate"] = 25;

    auto p = xjson::Parser(j);
    auto cfg = std::make_unique<ni::CounterReadTaskConfig>(sy, p);
    ASSERT_OCCURRED_AS(p.error(), xerrors::VALIDATION);
}

/// @brief it should return a validation error if no channels in the task are enabled.
TEST(CounterReadTaskConfigTest, testNoEnabledChannels) {
    auto sy = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(sy->hardware.create_rack("cat"));
    auto dev = synnax::Device(
        "abc123",
        "my_device",
        rack.key,
        "dev1",
        "ni",
        "PXI-6255",
        ""
    );
    ASSERT_NIL(sy->hardware.create_device(dev));
    auto ch = ASSERT_NIL_P(sy->channels.create("virtual", telem::FLOAT64_T, true));

    auto j = base_counter_config();
    j["device"] = dev.key;
    j["channels"][0]["device"] = dev.key;
    j["channels"][0]["channel"] = ch.key;
    j["channels"][0]["enabled"] = false;

    auto p = xjson::Parser(j);
    auto cfg = std::make_unique<ni::CounterReadTaskConfig>(sy, p);
    ASSERT_OCCURRED_AS(p.error(), xerrors::VALIDATION);
}

/// @brief it should return a validation error if an unknown channel type is provided.
TEST(CounterReadTaskConfigTest, testUnknownChannelType) {
    auto sy = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(sy->hardware.create_rack("cat"));
    auto dev = synnax::Device(
        "abc123",
        "my_device",
        rack.key,
        "dev1",
        "ni",
        "PXI-6255",
        ""
    );
    ASSERT_NIL(sy->hardware.create_device(dev));
    auto ch = ASSERT_NIL_P(sy->channels.create("virtual", telem::FLOAT64_T, true));

    auto j = base_counter_config();
    j["device"] = dev.key;
    j["channels"][0]["device"] = dev.key;
    j["channels"][0]["channel"] = ch.key;
    j["channels"][0]["type"] = "unknown_counter_type";

    auto p = xjson::Parser(j);
    auto cfg = std::make_unique<ni::CounterReadTaskConfig>(sy, p);
    ASSERT_OCCURRED_AS(p.error(), xerrors::VALIDATION);
}

/// @brief it should correctly parse a counter frequency channel with all parameters.
TEST(CounterReadTaskConfigTest, testCounterFrequencyChannelParse) {
    auto sy = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(sy->hardware.create_rack("cat"));
    auto dev = synnax::Device(
        "abc123",
        "my_device",
        rack.key,
        "dev1",
        "ni",
        "PXI-6255",
        ""
    );
    ASSERT_NIL(sy->hardware.create_device(dev));
    auto ch = ASSERT_NIL_P(sy->channels.create("virtual", telem::FLOAT64_T, true));

    auto j = base_counter_config();
    j["device"] = dev.key;
    j["channels"][0]["device"] = dev.key;
    j["channels"][0]["channel"] = ch.key;
    j["channels"][0]["units"] = "Ticks";
    j["channels"][0]["edge"] = "Falling";
    j["channels"][0]["meas_method"] = "LowFreq1Ctr";
    j["channels"][0]["terminal"] = "PFI15";

    auto p = xjson::Parser(j);
    auto cfg = std::make_unique<ni::CounterReadTaskConfig>(sy, p);
    ASSERT_NIL(p.error());
}

/// @brief it should correctly validate port uniqueness within the same device.
TEST(CounterReadTaskConfigTest, testPortCollisionSameDevice) {
    auto sy = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(sy->hardware.create_rack("cat"));
    auto dev = synnax::Device(
        "abc123",
        "my_device",
        rack.key,
        "dev1",
        "ni",
        "PXI-6255",
        ""
    );
    ASSERT_NIL(sy->hardware.create_device(dev));
    auto ch1 = ASSERT_NIL_P(sy->channels.create("virtual1", telem::FLOAT64_T, true));
    auto ch2 = ASSERT_NIL_P(sy->channels.create("virtual2", telem::FLOAT64_T, true));

    auto j = base_counter_config();
    j["device"] = dev.key;
    j["channels"][0]["device"] = dev.key;
    j["channels"][0]["channel"] = ch1.key;
    j["channels"][0]["port"] = 0;

    // Add second channel with same port
    auto second_channel = j["channels"][0];
    second_channel["key"] = "ks2VnWdrSVB";
    second_channel["channel"] = ch2.key;
    second_channel["port"] = 0; // Same port - should fail
    j["channels"].push_back(second_channel);

    auto p = xjson::Parser(j);
    auto cfg = std::make_unique<ni::CounterReadTaskConfig>(sy, p);
    ASSERT_OCCURRED_AS(p.error(), xerrors::VALIDATION);
}
