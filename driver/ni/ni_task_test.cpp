// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//
// Created by Synnax on 2/4/2024.
//
/// std
#include <stdio.h>
#include <thread>

/// GTest
#include <include/gtest/gtest.h>

/// Internal
#include "client/cpp/synnax.h"
#include "driver/ni/ni_reader.h"
#include "driver/testutil/testutil.h"
#include "nlohmann/json.hpp"

using json = nlohmann::json;

TEST(NiTaskTests, test_NI_reader_task){
    LOG(INFO) << "Test Acquisition Pipeline with NI Analog Read:" << std::endl;

    /////////////////////////////////////////////// setup synnax test infrustructure
    // create synnax client
    auto client_config = synnax::Config{
            "localhost",
            9090,
            "synnax",
            "seldon"};
    auto client = std::make_shared<synnax::Synnax>(client_config);
    
    // create all the necessary channels in the synnax client
    auto [time, tErr] = client->channels.create( // index channel for analog input channels
            "time",
            synnax::TIMESTAMP,
            0,
            true
    );
    ASSERT_FALSE(tErr) << tErr.message();
    auto [data, dErr] = client->channels.create( // analog input channel
            "acq_data",
            synnax::FLOAT32,
            time.key,
            false
    );
    ASSERT_FALSE(dErr) << dErr.message();

    // create reader config json
    auto config = json{
        {"acq_rate", 2000}, // dont actually need these here
        {"stream_rate", 20}, // same as above
        {"device_name", "Dev1"},
        {"reader_type", "analogReader"}
    };
    add_index_channel_JSON(config, "time", time.key);
    add_AI_channel_JSON(config, "acq_data", data.key, 0, -10.0, 10.0);

    // create synnax task
    auto task = synnax::Task(
            "my_task",
            "NI_analogReader",
            to_string(config)
    );

    auto mockCtx = std::make_shared<task::MockContext>(client);
    std::this_thread::sleep_for(std::chrono::milliseconds(10));


    /////////////////////////////////////////////// setup factory and task

    // make ni factory and build reader task
    std::unique_ptr<task::Factory> opc_factory = std::make_unique<ni::Factory>();
    auto [readerTask, ok] = ni::ReaderTask::configure(mockCtx, task);
    ASSERT_TRUE(ok) << "Failed to configure reader task";

    // create commands
    auto start_cmd = task::Command{
        .task_key = task.key,
        .command = "start",
        .args = {}
    };

    auto stop_cmd = task::Command{
        .task_key = task.key,
        .command = "stop",
        .args = {}
    };

    /////////////////////////////////////////////// begin acquisition

    // start reader task
    readerTask->exec(start_cmd);
    std::this_thread::sleep_for(std::chrono::seconds(5));
    readerTask->exec(stop_cmd);
}