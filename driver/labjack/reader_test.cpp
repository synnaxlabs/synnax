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

    auto config = json{
            {"sample_rate", 10000},
            {"stream_rate", 30},
            {"device_type", "T4"},
            {"device_key", "440022190"},
            {"serial_number", "440022190"},
            {"connection_type", "USB"},
            {"data_saving", true},
            {"channels", json::array({
                 {
                         {"location", "AIN0"},
                         {"enabled", true},
                         {"data_type", "float32"},
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
    auto mockCtx = std::make_shared<task::MockContext>(client);

    auto reader_task = labjack::ReaderTask::configure(mockCtx, task);
    // create commands
    auto start_cmd = task::Command{task.key, "start", {}};
    auto stop_cmd = task::Command{task.key, "stop", {}};
    reader_task->exec(start_cmd);
    std::this_thread::sleep_for(std::chrono::seconds(30000));
//    std::this_thread::sleep_for(std::chrono::seconds(2));
    reader_task->exec(stop_cmd);

}

TEST(read_tests, labjack_t4_multi_ain){
    auto client = std::make_shared<synnax::Synnax>(new_test_client());

    auto [time, tErr] = client->channels.create("idx", synnax::TIMESTAMP, 0, true);
    ASSERT_FALSE(tErr) << tErr.message();

    auto [data1, dErr1] = client->channels.create("ai_1", synnax::FLOAT32, time.key, false);
    ASSERT_FALSE(dErr1) << dErr1.message();

    auto [data2, dErr2] = client->channels.create("ai_2", synnax::FLOAT32, time.key, false);
    ASSERT_FALSE(dErr2) << dErr2.message();


    auto config = json{
            {"sample_rate", 10000},
            {"stream_rate", 30},
            {"device_type", "T4"},
            {"device_key", "440022190"},
            {"serial_number", "440022190"},
            {"connection_type", "USB"},
            {"data_saving", true},
            {"channels", json::array({
                             {
                                     {"location", "AIN0"},
                                     {"enabled", true},
                                     {"data_type", "float32"},
                                     {"channel_key", data1.key},
                                     {"range", 10.0},
                                     {"channel_types", "AIN"}
                             },
                             {
                                 {"location", "AIN1"},
                                 {"enabled", true},
                                 {"data_type", "float32"},
                                 {"channel_key", data2.key},
                                 {"range", 10.0},
                                 {"channel_types", "AIN"}
                             }
                })},
            {"index_keys", json::array({time.key})},
    };

    auto task = synnax::Task("my_task", "labjack_read", to_string(config));
    auto mockCtx = std::make_shared<task::MockContext>(client);

    auto reader_task = labjack::ReaderTask::configure(mockCtx, task);
    // create commands
    auto start_cmd = task::Command{task.key, "start", {}};
    auto stop_cmd = task::Command{task.key, "stop", {}};
    reader_task->exec(start_cmd);
    std::this_thread::sleep_for(std::chrono::seconds(30000));
    //    std::this_thread::sleep_for(std::chrono::seconds(2));
    reader_task->exec(stop_cmd);

}

TEST(read_tests, labjack_t4_ai_fio){
    auto client = std::make_shared<synnax::Synnax>(new_test_client());

    auto [time, tErr] = client->channels.create("idx", synnax::TIMESTAMP, 0, true);
    ASSERT_FALSE(tErr) << tErr.message();

    auto [data1, dErr1] = client->channels.create("ai_1", synnax::FLOAT32, time.key, false);
    ASSERT_FALSE(dErr1) << dErr1.message();

    auto [data2, dErr2] = client->channels.create("ai_2", synnax::FLOAT32, time.key, false);
    ASSERT_FALSE(dErr2) << dErr2.message();

    auto [data3, dErr3] = client->channels.create("di", synnax::SY_UINT8, time.key, false);
    ASSERT_FALSE(dErr3) << dErr3.message();

    auto config = json{
            {"sample_rate", 5000},
            {"stream_rate", 30},
            {"device_type", "T4"},
            {"device_key", "440022190"},
            {"serial_number", "440022190"},
            {"connection_type", "USB"},
            {"data_saving", true},
            {"channels", json::array({
                                             {
                                                     {"location", "AIN0"},
                                                     {"enabled", true},
                                                     {"data_type", "float32"},
                                                     {"channel_key", data1.key},
                                                     {"range", 10.0},
                                                     {"channel_types", "AIN"}
                                             },
                                             {
                                                     {"location", "AIN1"},
                                                     {"enabled", true},
                                                     {"data_type", "float32"},
                                                     {"channel_key", data2.key},
                                                     {"range", 10.0},
                                                     {"channel_types", "AIN"}
                                             },
                                             {
                                                     {"location", "FIO4"},
                                                     {"enabled", true},
                                                     {"data_type", "uint8"},
                                                     {"channel_key", data3.key},
                                                     {"channel_types", "DIN"}
                                             }
                                     })},
            {"index_keys", json::array({time.key})},
    };

    auto task = synnax::Task("my_task", "labjack_read", to_string(config));
    auto mockCtx = std::make_shared<task::MockContext>(client);

    auto reader_task = labjack::ReaderTask::configure(mockCtx, task);
    // create commands
    auto start_cmd = task::Command{task.key, "start", {}};
    auto stop_cmd = task::Command{task.key, "stop", {}};
    reader_task->exec(start_cmd);
    std::this_thread::sleep_for(std::chrono::seconds(30000));
    reader_task->exec(stop_cmd);

}