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
#include "driver/labjack/reader.h"
#include "driver/labjack/writer.h"
#include "driver/labjack/scanner.h"
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
        {"type", "T4"},
        {"device", "440022190"},
        {"connection_type", "USB"},
        {"data_saving", true},
        {"channels", json::array({
             {
                     {"port", "AIN0"},
                     {"enabled", true},
                     {"channel", data.key},
                     {"range", 10.0},
                     {"type", "AI"}
             }
        })},
    };

    auto task = synnax::Task("my_task", "labjack_read", to_string(config));
    auto mockCtx = std::make_shared<task::MockContext>(client);

    auto reader_task = labjack::ReaderTask::configure(mockCtx, task);

    auto start_cmd = task::Command{task.key, "start", {}};
    auto stop_cmd = task::Command{task.key, "stop", {}};

    reader_task->exec(start_cmd);
    std::this_thread::sleep_for(std::chrono::seconds(30000));
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
            {"sample_rate", 5000},
            {"stream_rate", 30},
            {"type", "T4"},
            {"device", "440022190"},
            {"connection_type", "USB"},
            {"data_saving", true},
            {"channels", json::array({
                 {
                         {"port", "AIN0"},
                         {"enabled", true},
                         {"channel", data1.key},
                         {"range", 10.0},
                         {"type", "AI"}
                 },
                 {
                         {"port", "AIN1"},
                         {"enabled", true},
                         {"channel", data2.key},
                         {"range", 10.0},
                         {"type", "AI"}
                 }
         })},
    };

    auto task = synnax::Task("my_task", "labjack_read", to_string(config));
    auto mockCtx = std::make_shared<task::MockContext>(client);

    auto reader_task = labjack::ReaderTask::configure(mockCtx, task);

    auto start_cmd = task::Command{task.key, "start", {}};
    auto stop_cmd = task::Command{task.key, "stop", {}};
    auto tare_cmd = task::Command{task.key, "tare", {}};

    reader_task->exec(start_cmd);
    for(int i = 0; i < 100; i++){
        std::this_thread::sleep_for(std::chrono::seconds(5));
        reader_task->exec(tare_cmd);
    }
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
            {"type", "T4"},
            {"device", "440022190"},
            {"connection_type", "USB"},
            {"data_saving", true},
            {"channels", json::array({
                                             {
                                                     {"port", "AIN0"},
                                                     {"enabled", true},
                                                     {"channel", data1.key},
                                                     {"range", 10.0},
                                                     {"type", "AI"}
                                             },
                                             {
                                                     {"location", "AIN1"},
                                                     {"enabled", true},
                                                     {"channel", data2.key},
                                                     {"range", 10.0},
                                                     {"type", "AI"}
                                             },
                                             {
                                                     {"location", "FIO4"},
                                                     {"enabled", true},
                                                     {"data_type", "uint8"},
                                                     {"key", data3.key},
                                                     {"type", "DIN"}
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

// TODO: tests there are no race conditions between reading a device and scanning for it
TEST(read_tests, labjack_scan_and_read){
    auto client = std::make_shared<synnax::Synnax>(new_test_client());

    /////////////////////////////////////////////////////////////////////////// scanner task
    auto scan_task = synnax::Task(
        "my_scan_task",
        "labjackScanner",
        ""
    );

    auto scanner_mock_ctx = std::make_shared<task::MockContext>(client);

    auto scanner = labjack::ScannerTask::configure(scanner_mock_ctx, scan_task);
    ///////////////////////////////////////////////////////////////////////////////

    auto [time, tErr] = client->channels.create("idx", synnax::TIMESTAMP, 0, true);
    ASSERT_FALSE(tErr) << tErr.message();

    auto [data, dErr] = client->channels.create("ai", synnax::FLOAT32, time.key, false);
    ASSERT_FALSE(dErr) << dErr.message();

    auto config = json{
            {"sample_rate", 10000},
            {"stream_rate", 30},
            {"type", "T4"},
            {"device", "440022190"},
            {"connection_type", "USB"},
            {"data_saving", true},
            {"channels", json::array({
                                             {
                                                     {"port", "AIN0"},
                                                     {"enabled", true},
                                                     {"channel", data.key},
                                                     {"range", 10.0},
                                                     {"type", "AI"}
                                             }
                                     })},
    };

    auto task = synnax::Task("my_task", "labjack_read", to_string(config));
    auto mockCtx = std::make_shared<task::MockContext>(client);

    auto reader_task = labjack::ReaderTask::configure(mockCtx, task);
    // create commands
    auto start_cmd = task::Command{task.key, "start", {}};
    auto stop_cmd = task::Command{task.key, "stop", {}};
    for(int i = 0; i < 100; i++){
        reader_task->exec(start_cmd);
        reader_task->exec(stop_cmd);
    }
}

TEST(read_tests, labjack_t4_read_and_write){
    auto client = std::make_shared<synnax::Synnax>(new_test_client());

    auto [time, tErr] = client->channels.create("idx", synnax::TIMESTAMP, 0, true);
    ASSERT_FALSE(tErr) << tErr.message();

    auto [data, dErr] = client->channels.create("ai", synnax::FLOAT32, time.key, false);
    ASSERT_FALSE(dErr) << dErr.message();

//    auto [data, dErr3] = client->channels.create("di", synnax::SY_UINT8, time.key, false);
//    ASSERT_FALSE(dErr3) << dErr3.message();

    auto config = json{
            {"sample_rate", 1000},
            {"stream_rate", 30},
            {"type", "T4"},
            {"device", "440022190"},
            {"connection_type", "USB"},
            {"data_saving", true},
            {"channels", json::array({
                                             {
                                                     {"port","AIN0"},
                                                     {"enabled", true},
                                                     {"key", data.key},
                                                     {"range", 10.0},
                                                     {"type", "AI"}
                                             }
                                             {
                                                 {"location", "FIO4"},
                                                 {"enabled", true},
                                                 {"data_type", "uint8"},
                                                 {"key", data.key},
                                                 {"type", "DIN"}
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

    ////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
    ////////////////////////////////////////////////////////////////////////////////////////////////////////// WRITE TASK
    auto [state_idx, tErr1] = client->channels.create("do_state_idx", synnax::TIMESTAMP, 0, true);
    ASSERT_FALSE(tErr1) << tErr1.message();

    auto [cmd_idx, tErr2] = client->channels.create("do_cmd_idx", synnax::TIMESTAMP,0, true);
    ASSERT_FALSE(tErr2) << tErr2.message();

    auto [state, aErr] = client->channels.create("do_state", synnax::SY_UINT8, state_idx.key, false);
    ASSERT_FALSE(aErr) << aErr.message();

    auto [cmd, cErr] = client->channels.create("do_cmd", synnax::SY_UINT8, cmd_idx.key, false);
    ASSERT_FALSE(cErr) << cErr.message();

    auto writer_config = json{
        {"type", "T4"},
        {"device_key", "440022190"},
        {"serial_number", "440022190"},
        {"connection_type", "USB"},
        {"channels", json::array({
                                         {
                                                 {"port", "FIO4"},
                                                 {"enabled", true},
                                                 {"data_type", "uint8"},
                                                 {"cmd_key", cmd.key},
                                                 {"state_key", state.key},
                                                 {"type", "DO"}
                                         }
                                 })},
        {"data_saving", true},
        {"state_rate", 10}
    };

    auto sy_task = synnax::Task("my_task", "labjack_write", to_string(writer_config));
    auto writer_mock_ctx = std::make_shared<task::MockContext>(client);

    auto writer_task = labjack::WriterTask::configure(writer_mock_ctx, sy_task);

    auto writer_start_cmd = task::Command{sy_task.key, "start", {}};
    auto writer_stop_cmd = task::Command{sy_task.key, "stop", {}};

    reader_task->exec(start_cmd);
    writer_task->exec(writer_start_cmd);
    std::this_thread::sleep_for(std::chrono::seconds(30000));
    writer_task->exec(writer_stop_cmd);
    reader_task->exec(stop_cmd);
}