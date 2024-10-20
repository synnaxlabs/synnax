// Copyright 2024 Synnax Labs, Inc.
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

using json = nlohmann::json;

TEST(NiTaskTests, test_NI_analog_reader_task) {
    LOG(INFO)
            << "Test NI task with NI Analog Read:" <<
            std::endl;

    /////////////////////////////////////////////// setup synnax test infrustructure
    // create synnax client
    auto client_config = synnax::Config{
        "localhost",
        9090,
        "synnax",
        "seldon"
    };
    auto client = std::make_shared<synnax::Synnax>(client_config);

    // create all the necessary channels in the synnax client
    auto [time, tErr] = client->channels.create(
        // index channel for analog input channels
        "time",
        synnax::TIMESTAMP,
        0,
        true
    );
    ASSERT_FALSE(tErr)
            << tErr.

            message();

    auto [data, dErr] = client->channels.create( // analog input channel
        "acq_data",
        synnax::FLOAT32,
        time.key,
        false
    );
    ASSERT_FALSE(dErr)
            << dErr.

            message();

    // create reader config json
    auto config = json{
        {"acq_rate", 2000}, // dont actually need these here
        {"stream_rate", 20}, // same as above
        {"device_name", "Dev1"},
        {"reader_type", "analogReader"}
    };
    add_index_channel_JSON(
        config,
        "time",
        time.key
    );
    add_AI_channel_JSON(
        config,
        "acq_data",
        data.key,
        0,
        -10.0,
        10.0,
        "Default"
    );


    // create synnax task
    auto task = synnax::Task(
        "my_task", // task name
        "niReader", // task type
        to_string(config) // task config
    );

    // print config
    std::cout << "Analog Reader Task Config: " <<
            to_string(config)
            <<
            std::endl;

    auto mockCtx = std::make_shared<task::MockContext>(client);
    std::this_thread::sleep_for(std::chrono::milliseconds(10)
    );

    // create a streamer to read the frames that the pipe writes to the server
    auto streamer_config = synnax::StreamerConfig{
        .channels = std::vector<synnax::ChannelKey>{time.key, data.key},
    };

    auto [streamer, sErr] = mockCtx->client->telem.openStreamer(streamer_config);


    /////////////////////////////////////////////// setup factory and task

    // make ni factory and build reader task
    std::unique_ptr<task::Factory> ni_factory = std::make_unique<ni::Factory>();
    auto [readerTask, ok] = ni_factory->configure_task(mockCtx, task);
    ASSERT_TRUE(ok) << "Failed to configure reader task";

    // create commands
    auto start_cmd = task::Command{task.key, "start", {}};
    auto stop_cmd = task::Command{task.key, "stop", {}};

    /////////////////////////////////////////////// begin acquisition

    // start reader task
    readerTask->
            exec(start_cmd);

    for (
        int i = 0;
        i < 30; i++) {
        auto [frame, err] = streamer.read();
        std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;

        uint32_t ai_count = 0;
        for (
            int i = 0;
            i < frame.series->

            size();

            i++) {
            std::cout << "\n\n Series " << i << ": \n";
            // check series type before casting
            if (frame.series->
                at(i)
                .data_type == synnax::FLOAT32) {
                auto s = frame.series->at(i).float32();
                for (
                    int j = 0;
                    j < s.

                    size();

                    j++) {
                    std::cout << s[j] << ", ";
                    ASSERT_NEAR(s[j],
                                0, 10);
                    // can be any value of a sign wave from -10 to 10
                }
                ai_count++;
            } else if (frame.series->
                       at(i)
                       .data_type == synnax::TIMESTAMP) {
                auto s = frame.series->at(i).uint64();
                for (
                    int j = 0;
                    j < s.

                    size();

                    j++) {
                    std::cout << s[j] << ", ";
                    ASSERT_TRUE((s[j]
                                 <= final_timestamp));
                }
            }
        }
        std::cout <<
                std::endl;
    }
    readerTask->
            exec(stop_cmd);
}


TEST(NiTaskTests, test_NI_digital_reader_task) {
    LOG(INFO)
            << "Test NI Task with NI Digital Read:" <<
            std::endl;

    /////////////////////////////////////////////// setup synnax test infrustructure
    // create synnax client
    auto client_config = synnax::Config{
        "localhost",
        9090,
        "synnax",
        "seldon"
    };
    auto client = std::make_shared<synnax::Synnax>(client_config);

    // create all the necessary channels in the synnax client
    auto [time, tErr] = client->channels.create(
        // index channel for digital input channels
        "time",
        synnax::TIMESTAMP,
        0,
        true
    );
    ASSERT_FALSE(tErr)
            << tErr.

            message();

    auto [data, dErr] = client->channels.create( // analog input channel
        "acq_data2",
        synnax::SY_UINT8,
        time.key,
        false
    );
    ASSERT_FALSE(dErr)
            << dErr.

            message();

    // create reader config json
    auto config = json{
        {"acq_rate", 2000}, // dont actually need these here
        {"stream_rate", 20}, // same as above
        {"device_name", "PXI1Slot2_2"},
        {"reader_type", "digitalReader"}
    };
    add_index_channel_JSON(
        config,
        "time",
        time.key
    );
    add_DI_channel_JSON(
        config,
        "acq_data",
        data.key,
        0,
        0
    );

    // create synnax task
    auto task = synnax::Task(
        "my_task",
        "niReader",
        to_string(config)
    );

    // print config
    std::cout << "Digital Reader Task Config: " << config.dump(4) <<
            std::endl;

    auto mockCtx = std::make_shared<task::MockContext>(client);
    std::this_thread::sleep_for(std::chrono::milliseconds(10)
    );

    // create a streamer to read the frames that the pipe writes to the server
    auto streamer_config = synnax::StreamerConfig{
        .channels = std::vector<synnax::ChannelKey>{time.key, data.key},
    };

    auto [streamer, sErr] = mockCtx->client->telem.openStreamer(streamer_config);


    /////////////////////////////////////////////// setup factory and task

    // make ni factory and build reader task
    std::unique_ptr<task::Factory> ni_factory = std::make_unique<ni::Factory>();
    auto [readerTask, ok] = ni_factory->configure_task(mockCtx, task);
    ASSERT_TRUE(ok)
            << "Failed to configure reader task";

    // create commands
    auto start_cmd = task::Command{task.key, "start", {}};

    auto stop_cmd = task::Command{task.key, "stop", {}};

    /////////////////////////////////////////////// begin acquisition

    // start reader task
    readerTask->
            exec(start_cmd);

    for (
        int i = 0;
        i < 30; i++) {
        auto [frame, err] = streamer.read();
        std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;

        uint32_t ai_count = 0;
        for (
            int i = 0;
            i < frame.series->

            size();

            i++) {
            std::cout << "\n\n Series " << i << ": \n";
            // check series type before casting
            if (frame.series->
                at(i)
                .data_type == synnax::SY_UINT8) {
                auto s = frame.series->at(i).uint8();
                for (
                    int j = 0;
                    j < s.

                    size();

                    j++) {
                    std::cout << (uint32_t) s[j] << ", ";
                    ASSERT_TRUE((s[j]
                                 == 1) || (s[j] == 0));
                }
                ai_count++;
            } else if (frame.series->
                       at(i)
                       .data_type == synnax::TIMESTAMP) {
                auto s = frame.series->at(i).uint64();
                for (
                    int j = 0;
                    j < s.

                    size();

                    j++) {
                    std::cout << s[j] << ", ";
                    ASSERT_TRUE((s[j]
                                 <= final_timestamp));
                }
            }
        }
        std::cout <<
                std::endl;
    }
    readerTask->
            exec(stop_cmd);
}


TEST(NiTaskTests, test_NI_digital_writer_task) {
    LOG(INFO)
            << "Test NI writer task with  NI Digital Writer: " <<
            std::endl;
    // create synnax client
    auto client_config = synnax::Config{
        "localhost",
        9090,
        "synnax",
        "seldon"
    };
    auto client = std::make_shared<synnax::Synnax>(client_config);

    // create all the necessary channels in the synnax client
    auto [ack_idx, tErr1] = client->channels.create( // index channel for acks
        "do_state_idx",
        synnax::TIMESTAMP,
        0,
        true
    );
    ASSERT_FALSE(tErr1)
            << tErr1.

            message();

    auto [cmd_idx, tErr2] = client->channels.create( // index channel for cmd
        "do_cmd_idx",
        synnax::TIMESTAMP,
        0,
        true
    );
    ASSERT_FALSE(tErr2)
            << tErr2.

            message();

    auto [ack, aErr] = client->channels.create( // ack channel
        "do_state",
        synnax::SY_UINT8,
        ack_idx.key,
        false
    );
    ASSERT_FALSE(aErr)
            << aErr.

            message();

    auto [cmd, cErr] = client->channels.create( // cmd channel
        "do_cmd",
        synnax::SY_UINT8,
        cmd_idx.key,
        false
    );
    ASSERT_FALSE(cErr)
            << cErr.

            message();

    // create a writer to write to cmd channel (for test use only)
    auto cmdWriterConfig = synnax::WriterConfig{
        .channels = std::vector<synnax::ChannelKey>{cmd_idx.key, cmd.key},
        .start = TimeStamp::now(),
        .mode = synnax::StreamOnly
    };

    add_index_channel_JSON(
        config,
        "do1_idx",
        cmd_idx.key
    );
    add_DO_channel_JSON(
        config,
        "do_cmd",
        cmd.key,
        ack.key,
        0,
        0
    );
    add_state_index_channel_JSON(
        config,
        "do_state_idx",
        ack_idx.key
    );

    // create synnax task
    auto task = synnax::Task(
        "my_task",
        "niWriter",
        to_string(config)
    );

    // print config
    std::cout << "D9igital Writer Task Config: " << config.dump(4) <<
            std::endl;

    auto mockCtx = std::make_shared<task::MockContext>(client);
    std::this_thread::sleep_for(std::chrono::milliseconds(10)
    );

    // create a writer to write to cmd channel (for test use only)
    auto cmdWriterConfig = synnax::WriterConfig{
        .channels = std::vector<synnax::ChannelKey>{cmd_idx.key, cmd.key},
        .start = TimeStamp::now(),
        .mode = synnax::WriterStreamOnly
    };

    auto [cmdWriter, wErr] = client->telem.openWriter(cmdWriterConfig);
    ASSERT_FALSE(wErr)
            << wErr.

            message();

    // create a streamer to stream do_state channel (for in test use only)
    auto doStateStreamerConfig = synnax::StreamerConfig{
        .channels = std::vector<synnax::ChannelKey>{ack_idx.key, ack.key},
    };
    auto [doStateStreamer, sErr] = client->telem.openStreamer(doStateStreamerConfig);
    ASSERT_FALSE(sErr)
            << sErr.

            message();

    /////////////////////////////////////////////// setup factory and task

    // make ni factory and build reader task
    std::unique_ptr<task::Factory> ni_factory = std::make_unique<ni::Factory>();
    auto [writerTask, ok] = ni_factory->configure_task(mockCtx, task);
    ASSERT_TRUE(ok) << "Failed to configure reader task";

    // create commands
    auto start_cmd = task::Command{task.key, "start", {}};
    auto stop_cmd = task::Command{task.key, "stop", {}};

    /////////////////////////////////////////////// begin Control
    writerTask->
            exec(start_cmd);
    std::this_thread::sleep_for(std::chrono::seconds(1)
    );
    //////////////////////////////////////////// write a 1 to the cmd channel ////////////////////////////////////////////
    LOG(INFO)
            << "Commanding a logic high: " <<
            std::endl;
    // construct frame
    auto cmd_frame = synnax::Frame(2);
    cmd_frame.
            add(
                cmd_idx
                .key,
                synnax::Series(
                    std::vector<uint64_t>{synnax::TimeStamp::now().value},
                    synnax::TIMESTAMP
                )
            );
    cmd_frame.
            add(
                cmd
                .key,
                synnax::Series(std::vector<uint8_t>{1}
                )
            );
    ASSERT_TRUE(cmdWriter
        .
        write(std::move(cmd_frame)
        ));
    // TODO: remove -> isnt necessary

    // do initial read before state update, should be 0
    auto [state_frame, err3] = doStateStreamer.read();
    ASSERT_FALSE(err3)
            << err3.

            message();

    auto s = state_frame.series->at(1).uint8();
    LOG(INFO
        << "State: " << (int) s[0] <<
        std::endl;
    ASSERT_TRUE(s[0]
                == 0);

    // keep reading state channel and printing state
    for (
        int i = 0;
        i < 5; i++) {
        auto [state_frame, err3] = doStateStreamer.read();
        ASSERT_FALSE(err3)
                << err3.

                message();

        auto s = state_frame.series->at(1).uint8();
        LOG(INFO)
                << "State: " << (int) s[0] <<
                std::endl;
        ASSERT_TRUE(s[0]
                    == 1);
    }
    writerTask->
            exec(stop_cmd);
}

TEST(NiTaskTests, test_NI_scanner_task) {
    LOG(INFO)
            << "Test NI Scanner Task:" <<
            std::endl;
    // create properties json
    nlohmann::json config;
    config["properties"] =

            nlohmann::json::array();

    config["properties"].push_back("SerialNumber");
    config["properties"].push_back("DeviceName");

    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto task = synnax::Task(
        "my_task",
        "niScanner",
        to_string(config)
    );

    // print config
    std::cout << "Scanner Task Config: " << config.dump(4) <<
            std::endl;
    auto mockCtx = std::make_shared<task::MockContext>(client);

    /////////////////////////////////////////////// setup factory and task

    // make ni factory and build reader task
    std::unique_ptr<task::Factory> ni_factory = std::make_unique<ni::Factory>();

    LOG(INFO) << "Make Scanner task:" << std::endl;
    auto [scanner_task, ok] = ni_factory->configure_task(mockCtx, task);
    ASSERT_TRUE(ok) << "Failed to configure reader task";

    // create commands
    LOG(INFO)
            << "Send scan cmd:" <<
            std::endl;
    auto scan_cmd = task::Command{task.key, "scan", {}};

    // perform a scan
    scanner_task->
            exec(scan_cmd);
}


/* Configs


/////////////////////////////////////////////////////////////////////////////////////// ANALOG READER

I20240511 20:22:43.922561 21804 reader_test.cpp:165] test_read_one_analog_channel:
{
    "acq_rate": 100,
    "channels": [
        {
            "ack_key": 0,
            "channel_key": 1,
            "channel_type": "index",
            "cmd_key": 0,
            "line": 0,
            "name": "idx",
            "port": 0
        },
        {
            "channel_key": 65531,
            "channel_type": "analogVoltageInput",
            "max_val": 10.0,
            "min_val": -10.0,
            "name": "a1",
            "port": 0,
            "terminal_config": "Default"
        }
    ],
    "device_name": "Dev1",
    "reader_type": "analogReader",
    "stream_rate": 20
}


/////////////////////////////////////////////////////////////////////////////////////// DIGITAL READER

Digital Reader Task Config: {
    "acq_rate": 2000,
    "channels": [
        {
            "ack_key": 0,
            "channel_key": 1048605,
            "channel_type": "index",
            "cmd_key": 0,
            "line": 0,
            "name": "time",
            "port": 0
        },
        {
            "channel_key": 1048606,
            "channel_type": "digitalInput",
            "line": 0,
            "name": "acq_data",
            "port": 0
        }
    ],
    "device_name": "PXI1Slot2_2",
    "reader_type": "digitalReader",
    "stream_rate": 20
}



////////////////////////////////////////////////////////////////////////////////////// DIGITAL WRITER
Digital Writer Task Config: {
    "channels": [
        {
            "ack_key": 0,
            "channel_key": 1048608,
            "channel_type": "index",
            "cmd_key": 0,
            "line": 0,
            "name": "do1_idx",
            "port": 0
        },
        {
            "channel_key": 1048610,
            "channel_type": "digitalOutput",
            "state_key": 1048609,
            "line": 0,
            "name": "do_cmd",
            "port": 0
        },
        {
            "channel_key": 1048607,
            "channel_type": "driveStateIndex",
            "line": 0,
            "name": "do_state_idx",
            "port": 0
        }
    ],
    "device_name": "Dev1",
    "stream_rate": 1
}


////////////////////////////////////////////////////////////////////////////////////// SCANNER

{
    "properties": [
        "SerialNumber",
        "DeviceName"
    ]
}


*/
