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

TEST(NiTaskTests, test_NI_digital_writer_task) {
    LOG(INFO) << "Test NI writer task with  NI Digital Writer: " << std::endl;
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
    ASSERT_FALSE(tErr1) << tErr1.message();

    auto [cmd_idx, tErr2] = client->channels.create( // index channel for cmd
        "do_cmd_idx",
        synnax::TIMESTAMP,
        0,
        true
    );
    ASSERT_FALSE(tErr2) << tErr2.message();

    auto [ack, aErr] = client->channels.create( // ack channel
        "do_state",
        synnax::SY_UINT8,
        ack_idx.key,
        false
    );
    ASSERT_FALSE(aErr) << aErr.message();

    auto [cmd, cErr] = client->channels.create( // cmd channel
        "do_cmd",
        synnax::SY_UINT8,
        cmd_idx.key,
        false
    );
    ASSERT_FALSE(cErr) << cErr.message();

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
    std::cout << "D9igital Writer Task Config: " << config.dump(4) << std::endl;

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
    ASSERT_FALSE(wErr) << wErr.message();

    // create a streamer to stream do_state channel (for in test use only)
    auto doStateStreamerConfig = synnax::StreamerConfig{
        .channels = std::vector<synnax::ChannelKey>{ack_idx.key, ack.key},
    };
    auto [doStateStreamer, sErr] = client->telem.openStreamer(doStateStreamerConfig);
    ASSERT_FALSE(sErr) << sErr.message();

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
    LOG(INFO) << "Commanding a logic high: " << std::endl;
    // construct frame
    auto cmd_frame = synnax::Frame(2);
    cmd_frame.add(
        cmd_idx
        .key,
        synnax::Series(
            std::vector<uint64_t>{synnax::TimeStamp::now().value},
            synnax::TIMESTAMP
        )
    );

    cmd_frame.add(
        cmd
        .key,
        synnax::Series(std::vector<uint8_t>{1}
        )
    );

    ASSERT_TRUE(cmdWriter.write(std::move(cmd_frame)));
    // TODO: remove -> isnt necessary

    // do initial read before state update, should be 0
    auto [state_frame, err3] = doStateStreamer.read();
    ASSERT_FALSE(err3) << err3.message();

    auto s = state_frame.series->at(1).uint8();
    LOG(INFO) << "State: " << (int) s[0] << std::endl;
    ASSERT_TRUE(s[0] == 0);

    // keep reading state channel and printing state
    for (int i = 0; i < 5; i++) {
        auto [state_frame, err3] = doStateStreamer.read();
        ASSERT_FALSE(err3) << err3.message();

        auto s = state_frame.series->at(1).uint8();
        LOG(INFO) << "State: " << (int) s[0] << std::endl;
        ASSERT_TRUE(s[0] == 1);
    }
    writerTask->exec(stop_cmd);
}
