// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <stdio.h>

#include "client/cpp/synnax.h"
#include "driver/labjack/task.h"
#include "driver/labjack/reader.h"
#include "driver/testutil/testutil.h"

#include <include/gtest/gtest.h>
#include "glog/logging.h"
#include "nlohmann/json.hpp"

using json = nlohmann::json;

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//                                                   Basic Tests                                                //
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
TEST(read_tests, labjack_t4){
    auto client = std::make_shared<synnax::Synnax>(new_test_client());

    auto [time, tErr] = client->channels.create("idx", synnax::TIMESTAMP, 0, true);
    ASSERT_FALSE(tErr) << tErr.message();

    auto [data, dErr] = client->channels.create("ai", synnax::FLOAT32, time.key, false);
    ASSERT_FALSE(dErr) << dErr.message();

    // TODO: set task_key in init/constructor of reader source, same with name
    auto config = json{
            {"sample_rate", 1000},             // TODO: actually make sure these work
            {"stream_rate", 100},              // TODO: actually make sure these work
            {"device_type", "LabJack T4"},     // TODO: change name
            {"device_key", "T4-001"},          // TODO: change to actual serial number of device we ahve
            {"serial_number", "440123456"},    // TODO: change to actual serial number of device
            {"connection_type", "USB"},
            {"data_saving", true},
            {"channels", json::array({
                 {
                         {"location", "AIN0"},
                         {"enabled", true},
                         {"data_type", "FLOAT32"}, // TODO: make sure this is the actual data typ eof the device
                         {"channel_key", data.key},
                         {"range", 10.0},
                         {"channel_types", "AIN"}
                 }
            })},
            {"index_keys", json::array({time.key})},
            {"channel_map", {
                        {"AIN0", data.key}
                    }}
    };


    auto task = synnax::Task("my_task", "labjack_read", to_string(config));
    auto mockCtx = std::make_shared<task::Context>(client);
    std::this_thread::sleep_for(std::chrono::milliseconds(300)); // TODO: remove? don't know what i need this

    auto reader_task = labjack::ReaderTask::configure(mockCtx, task);

    reader_task.start();
    std::this_thread::sleep_for(std::chrono::seconds(300));
    reader_task.stop();

}