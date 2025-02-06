// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <map>
#include <stdio.h>

#include "client/cpp/synnax.h"
#include "driver/ni/ni.h"
#include "driver/testutil/testutil.h"

#include <gtest/gtest.h>
#include "glog/logging.h"
#include "nidaqmx/nidaqmx_prod.h"

#include "nlohmann/json.hpp"

using json = nlohmann::json;

/* 
Devices Identifiers in NI MAX
Dev1 : NI USB-6289 (simulated device)
*/

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//                                                                                                              //
//                                                   Basic Tests                                                //                
//                                                                                                              //
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////

void digital_channel_helper(json config, json channel_config) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());

    auto [time, tErr] = client->channels.create(
        "idx",
        synnax::TIMESTAMP,
        0,
        true);
    ASSERT_FALSE(tErr) << tErr.message();

    auto [data, dErr] = client->channels.create(
        "di_channel",
        synnax::FLOAT32,
        time.key,
        false);
    ASSERT_FALSE(dErr) << dErr.message();

    channel_config["channel"] = data.key;
    channel_config["enabled"] = true;
    config["channels"] = json::array();
    config["channels"].push_back(channel_config);

    auto task = synnax::Task(
        "my_task",
        "ni_digital_read",
        to_string(config));

    auto mockCtx = std::make_shared<task::MockContext>(client);
    std::this_thread::sleep_for(std::chrono::milliseconds(300));

    auto [dmx, dmx_err] = DAQmxProd::load();
    ASSERT_FALSE(dmx_err) << dmx_err.message();

    TaskHandle taskHandle;
    dmx->CreateTask("", &taskHandle);

    auto reader = ni::DigitalReadSource(dmx, taskHandle, mockCtx, task);
    auto b = breaker::Breaker(
        breaker::Config{
            "my-breaker",
            1 * SECOND,
            1,
            1
        });

    if (reader.init() != 0)
        LOG(ERROR) << "Failed to initialize reader" << std::endl;
    reader.start("");

    for (int i = 0; i < 15; i++) {
        std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
        auto [frame, err] = reader.read(b);
        std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;
        VLOG(1) << frame << "\n";
    }

    reader.stop("");
}

TEST(read_tests, one_digital_channel) {
    auto config = json{
        {"sample_rate", 100},
        {"stream_rate", 20},
        {"device_location", "Dev1"},
        {"type", "ni_digital_read"},
        {"test", true},
        {"device", ""}
    };

    auto channel_config = json{
        {"name", "test_di_channel"},
        {"type", "di"},
        {"port", 0},
        {"line", 0},
        {"enabled", true},
        {"key", "key"}
    };

    digital_channel_helper(config, channel_config);
}

TEST(read_tests, multiple_digital_channels) {
    auto client = std::make_shared<synnax::Synnax>(new_test_client());

    auto [time, tErr] = client->channels.create("idx", synnax::TIMESTAMP, 0, true);
    ASSERT_FALSE(tErr) << tErr.message();

    auto [data, dErr] = client->channels.create("di", synnax::FLOAT32, time.key, false);
    ASSERT_FALSE(dErr) << dErr.message();

    auto [data1, dErr2] = client->channels.create("di2", synnax::FLOAT32, time.key, false);
    ASSERT_FALSE(dErr2) << dErr.message();

    auto [data2, dErr3] = client->channels.create("di3", synnax::FLOAT32, time.key, false);
    ASSERT_FALSE(dErr3) << dErr.message();

    auto [data3, dErr4] = client->channels.create("di4", synnax::FLOAT32, time.key, false);
    ASSERT_FALSE(dErr4) << dErr.message();

    auto config = json{
        {"sample_rate", 1000},
        {"stream_rate", 20},
        {"device_location", "Dev1"},
        {"type", "ni_digital_read"},
        {"test", true},
        {"device", ""}
    };

    json channels = json::array();
    
    channels.push_back({
        {"name", "d1"},
        {"type", "di"},
        {"port", 0},
        {"line", 0},
        {"enabled", true},
        {"channel", data.key}
    });

    channels.push_back({
        {"name", "d2"},
        {"type", "di"},
        {"port", 0},
        {"line", 1},
        {"enabled", true},
        {"channel", data1.key}
    });

    channels.push_back({
        {"name", "d3"},
        {"type", "di"},
        {"port", 0},
        {"line", 2},
        {"enabled", true},
        {"channel", data2.key}
    });

    channels.push_back({
        {"name", "d4"},
        {"type", "di"},
        {"port", 0},
        {"line", 3},
        {"enabled", true},
        {"channel", data3.key}
    });

    config["channels"] = channels;

    auto task = synnax::Task("my_task", "ni_digital_read", to_string(config));
    auto mockCtx = std::make_shared<task::MockContext>(client);
    std::this_thread::sleep_for(std::chrono::milliseconds(300));

    auto [dmx, dmx_err] = DAQmxProd::load();
    ASSERT_FALSE(dmx_err) << dmx_err.message();

    TaskHandle taskHandle;
    dmx->CreateTask("", &taskHandle);

    auto reader = ni::DigitalReadSource(dmx, taskHandle, mockCtx, task);
    auto b = breaker::Breaker(breaker::Config{"my-breaker", 1 * SECOND, 1, 1});

    if (reader.init() != 0) LOG(ERROR) << "Failed to initialize reader" << std::endl;
    reader.start("");

    for (int i = 0; i < 15; i++) {
        std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
        auto [frame, err] = reader.read(b);
        std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;
        VLOG(1) << frame << "\n";
    }

    reader.stop("");
}
