// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <stdio.h>
#include <thread>

#include <include/gtest/gtest.h>

#include "client/cpp/synnax.h"
#include "driver/ni/ni.h"
#include "driver/testutil/testutil.h"
#include "nlohmann/json.hpp"

// macro define for the devices needed for each test
#define SIMULATED_AO_DEVICE "0577EE88-E26D-11EF-804F-FB40AD45A9A9" // Simulated NI-9263
#define AO_DEVICE "01BB4D51"  // Physical NI-9263
#define DO_DEVICE "7B997D92-D8F3-11EF-8063-D5E44C514171"

using json = nlohmann::json;

TEST(NiTaskTests, test_NI_analog_writer_task) {
    LOG(INFO) << "Test NI writer task with  NI Digital Writer: " << std::endl;
    auto client_config = synnax::Config{
        "localhost",
        9090,
        "synnax",
        "seldon"
    };
    auto client = std::make_shared<synnax::Synnax>(client_config);

    auto [ack_idx, tErr1] = client->channels.create(
        "ao_state_idx",
        telem::TIMESTAMP,
        0,
        true
    );
    ASSERT_FALSE(tErr1) << tErr1.message();

    auto [cmd_idx, tErr2] = client->channels.create(
        "ao_cmd_idx",
        telem::TIMESTAMP,
        0,
        true
    );
    ASSERT_FALSE(tErr2) << tErr2.message();

    auto [ack, aErr] = client->channels.create(
        "ao_state",
        telem::FLOAT64,
        ack_idx.key,
        false
    );
    ASSERT_FALSE(aErr) << aErr.message();

    auto [cmd, cErr] = client->channels.create(
        "ao_cmd",
        telem::FLOAT64,
        cmd_idx.key,
        false
    );
    ASSERT_FALSE(cErr) << cErr.message();

    auto config = json{
        {
            "channels", json::array({
                {
                    {"cmd_channel", cmd.key},
                    {"enabled", true},
                    {"key", "w1GsZJokuR6"},
                    {"port", 1},
                    {"state_channel", ack.key},
                    {"type", "ao_voltage"},
                    {"min_val", 0},
                    {"max_val", 10},
                    {"units", "Volts"}
                }
            })
        },
        {"data_saving", true},
        {"device", AO_DEVICE},
        {"state_rate", 10}
    };

    auto task = synnax::Task(
        "my_task",
        "ni_analog_write",
        to_string(config)
    );

    std::cout << "Analog Writer Task Config: " << config.dump(4) << std::endl;

    auto mockCtx = std::make_shared<task::MockContext>(client);
    std::this_thread::sleep_for(std::chrono::milliseconds(10)
    );

    auto ni_factory = ni::Factory::create();

    auto [writerTask, ok] = ni_factory->configure_task(mockCtx, task);
    ASSERT_TRUE(ok) << "Failed to configure writer task";

    auto start_cmd = task::Command{task.key, "start", {}};
    auto stop_cmd = task::Command{task.key, "stop", {}};

    writerTask->exec(start_cmd);
    std::this_thread::sleep_for(std::chrono::seconds(500));
    writerTask->exec(stop_cmd);
}

TEST(NiTaskTests, test_NI_digital_writer_task) {
    LOG(INFO) << "Test NI writer task with  NI Digital Writer: " << std::endl;
    auto client_config = synnax::Config{
        "localhost",
        9090,
        "synnax",
        "seldon"
    };
    auto client = std::make_shared<synnax::Synnax>(client_config);

    auto [ack_idx, tErr1] = client->channels.create(
        "do_state_idx",
        telem::TIMESTAMP,
        0,
        true
    );
    ASSERT_FALSE(tErr1) << tErr1.message();

    auto [cmd_idx, tErr2] = client->channels.create(
        "do_cmd_idx",
        telem::TIMESTAMP,
        0,
        true
    );
    ASSERT_FALSE(tErr2) << tErr2.message();

    auto [ack, aErr] = client->channels.create(
        "do_state",
        telem::SY_UINT8,
        ack_idx.key,
        false
    );
    ASSERT_FALSE(aErr) << aErr.message();

    auto [cmd, cErr] = client->channels.create(
        "do_cmd",
        telem::SY_UINT8,
        cmd_idx.key,
        false
    );
    ASSERT_FALSE(cErr) << cErr.message();

    auto config = json{
        {
            "channels", json::array({
                {
                    {"cmd_channel", cmd.key},
                    {"enabled", true},
                    {"key", "w1GsZJokuR6"},
                    {"line", 0},
                    {"port", 0},
                    {"state_channel", ack.key},
                }
            })
        },
        {"data_saving", true},
        {"device", DO_DEVICE},
        {"state_rate", 10}
    };

    auto task = synnax::Task(
        "my_task",
        "ni_digital_write",
        to_string(config)
    );

    std::cout << "Digital Writer Task Config: " << config.dump(4) << std::endl;

    auto mockCtx = std::make_shared<task::MockContext>(client);
    std::this_thread::sleep_for(std::chrono::milliseconds(10)
    );

    auto ni_factory = ni::Factory::create();
    auto [writerTask, ok] = ni_factory->configure_task(mockCtx, task);

    ASSERT_TRUE(ok) << "Failed to configure writer task";

    auto start_cmd = task::Command{task.key, "start", {}};
    auto stop_cmd = task::Command{task.key, "stop", {}};

    writerTask->exec(start_cmd);
    std::this_thread::sleep_for(std::chrono::seconds(500));
    writerTask->exec(stop_cmd);
}
