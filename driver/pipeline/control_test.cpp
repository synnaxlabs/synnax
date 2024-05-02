// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//
// Created by Synnax on 3/6/2024.
//

/// std
#include <stdio.h>
#include <thread>

/// GTest
#include <include/gtest/gtest.h>

/// Internal
#include "client/cpp/synnax.h"
#include "driver/ni/ni_reader.h"
#include "Acquisition.h"
#include "control.h"
#include "driver/testutil/testutil.h"
#include "nlohmann/json.hpp"
#include "driver/breaker/breaker.h"


TEST(ControlPipelineTests, test_control_NI_digital_writer){
    LOG(INFO) << "Test Control Pipeline with NI Digital Write: " << std::endl;
     // create synnax client
    auto client_config = synnax::Config{
                "localhost",
                9090,
                "synnax",
                "seldon"};
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
            synnax::UINT8,
            ack_idx.key,
            false
    );
    ASSERT_FALSE(aErr) << aErr.message();
    auto [cmd, cErr] = client->channels.create( // cmd channel
            "do_cmd",
            synnax::UINT8,
            cmd_idx.key,
            false
    );
    ASSERT_FALSE(cErr) << cErr.message();

    // create reader config json
    auto config = json{
            {"device_name", "Dev1"},
            {"stream_rate", 1}
    };

    add_index_channel_JSON(config, "do1_idx", cmd_idx.key);
    add_DO_channel_JSON(config, "do_cmd", cmd.key, ack.key, 0, 0);
    add_drive_state_index_channel_JSON(config, "do_state_idx", ack_idx.key);

    // create a writer to write to cmd channel (for test use only)
    auto cmdWriterConfig = synnax::WriterConfig{
                .channels = std::vector<synnax::ChannelKey>{cmd_idx.key, cmd.key},
                .start = TimeStamp::now(),
                .mode = synnax::WriterStreamOnly};

    auto [cmdWriter, wErr] = client->telem.openWriter(cndWriterConfig);
    ASSERT_FALSE(wErr) << wErr.message();

    // create a streamer to stream do_state channel (for in test use only)
    auto doStateStreamerConfig = synnax::StreamerConfig{
        .channels = std::vector<synnax::ChannelKey>{ack_idx.key, ack.key},
        .start = TimeStamp::now(),
    };
    auto [doStateStreamer, sErr] = client->telem.openStreamer(doStateStreamerConfig);
    ASSERT_FALSE(sErr) << sErr.message();

    // create breaker config
    auto breaker_config = breaker::Config{
                .name = task.name,
                .base_interval = 1 * SECOND,
                .max_retries = 20,
                .scale = 1.2,
    };

    // create streamer config to pass into the control pipeline to poll for commands
    auto cmdStreamerConfig = synnax::StreamerConfig{
        .channels = std::vector<synnax::ChannelKey>{cmd_idx.key, cmd.key},
        .start = TimeStamp::now(),
    };

    // instantiate and initialize the daq writer
    TaskHandle taskHandle;
    DAQmxCreateTask("",&taskHandle);
    auto daq_writer = std::make_unique<ni::niDaqWriter>(taskHandle); 

    // instantiate and initialize the Control pipeline
    auto ctrl = pipeline::Control(mockCtx, cmdStreamerConfig, std::move(daq_writer), breaker_config);



    // create a writer to write to STATE channel (for test use only)
    auto ackWriterConfig = synnax::WriterConfig{
                .channels = std::vector<synnax::ChannelKey>{ack_idx.key, ack.key},
                .start = TimeStamp::now(),
                .mode = synnax::WriterStreamOnly};

    // instantiate and initialize the Acquisition pipeline to actually write to the server
    auto acq = pipeline::Acquisition(mockCtx, ackWriterConfig, daq_writer->writer_state_source, breaker_config);

    // create a state streamer to read frames that the acq pipe writes to the server
    auto streamer_config = synnax::StreamerConfig{
                .channels = std::vector<synnax::ChannelKey>{ack_idx.key, ack.key},
                .start = TimeStamp::now(),
        };

    auto [streamer, sErr] = mockCtx->client->telem.openStreamer(streamer_config);

    // start the control pipeline
    ctrl.start();
    acq.start();

    // write to the cmd channel
    auto frame = synnax::Frame(2);
    frame.add(cmd_idx.key, synnax::Series(std::vector<uint64_t>{time}, synnax::TIMESTAMP));
    frame.add(cmd.key, synnax::Series(std::vector<uint8_t>{1}));
    ASSERT_TRUE(cmdWriter.write(std::move(frame)));
    auto [end, ok] = cmdWriter.commit();
    // read from the ack channel
    auto [frame, err] = streamer.read();
    // check that the frame is correct
    auto s = frame.series->at(1).uint8();
    ASSERT_TRUE(s[0] == 1);

    // now write a 0 to the cmd channel
    auto frame = synnax::Frame(2);
    frame.add(cmd_idx.key, synnax::Series(std::vector<uint64_t>{time}, synnax::TIMESTAMP));
    frame.add(cmd.key, synnax::Series(std::vector<uint8_t>{0}));
    ASSERT_TRUE(cmdWriter.write(std::move(frame)));
    auto [end, ok] = cmdWriter.commit();
    // read from the ack channel
    auto [frame, err] = streamer.read();
    // check that the frame is correct
    auto s = frame.series->at(1).uint8();
    ASSERT_TRUE(s[0] == 1);



    acq.stop();
    ctrl.stop();
}




/////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/// @brief it should use niReader and perform a control workflow
/// which includes init, start, stop, and write functions and also commits ack frames to server
TEST(CtrlTests, testCtrlNi){

    std::cout << "Test Ctrl: " << std::endl;

    /// set up test infrustructure
    // create synnax client config
    auto client_config = synnax::Config{
            "localhost",
            9090,
            "synnax",
            "seldon"};
    auto client = std::make_shared<synnax::Synnax>(client_config);

    // create all the necessary channels in the synnax client
    auto [ack_idx, tErr1] = client->channels.create( // index channel for acks
            "ack_idx",
            synnax::TIMESTAMP,
            0,
            true
    );
    ASSERT_FALSE(tErr1) << tErr1.message();
    auto [cmd_idx, tErr2] = client->channels.create( // index channel for cmd
        "cmd_idx",
        synnax::TIMESTAMP,
        0,
        true
    );
    ASSERT_FALSE(tErr2) << tErr2.message();
    auto [ack, aErr] = client->channels.create( // ack channel
            "ack",
            synnax::UINT8,
            ack_idx.key,
            false
    );
    ASSERT_FALSE(aErr) << aErr.message();
    auto [cmd, cErr] = client->channels.create( // cmd channel
            "cmd",
            synnax::UINT8,
            cmd_idx.key,
            false
    );
    ASSERT_FALSE(cErr) << cErr.message();

    // create config json
    auto config = json{
            {"acq_rate", 300}, // dont actually need these here
            {"stream_rate", 30}, // same as above
            {"hardware", "Dev1"}
    };
    add_index_channel_JSON(config, "ack_idx", ack_idx.key);
    add_DO_channel_JSON(config, "cmd", cmd.key, ack.key, 0, 0);

    // create a writer to write to cmd channel (for test use only)
    auto now = synnax::TimeStamp::now();
    auto cmdWriterConfig = synnax::WriterConfig{
        std::vector<synnax::ChannelKey>{cmd_idx.key, cmd.key},
        now,
        std::vector<synnax::Authority>{synnax::ABSOLUTTE, synnax::ABSOLUTTE},
        synnax::Subject{"test_cmd_writer"},
    };
    auto [cmdWriter,wErr] = client->telem.openWriter(cmdWriterConfig);
    ASSERT_FALSE(wErr) << wErr.message();

    // create a streamer to stream ack channel (for in test use only)
    auto ackStreamerConfig = synnax::StreamerConfig{
        std::vector<synnax::ChannelKey>{ack_idx.key, ack.key},
        synnax::TimeStamp::now(),
    };
    auto [ackStreamer, sErr] = client->telem.openStreamer(ackStreamerConfig);
    ASSERT_FALSE(sErr) << wErr.message();

    // create writer config
    now = synnax::TimeStamp::now();
    auto writerConfig = synnax::WriterConfig{
        std::vector<synnax::ChannelKey>{ack_idx.key, ack.key},
        now,
        std::vector<synnax::Authority>{synnax::ABSOLUTTE, synnax::ABSOLUTTE},
        synnax::Subject{"test_ctrl_loop"},
    };

    std::cout << " Test ack key: " << ack.key << std::endl;
    std::cout << " Test ack_idx key: " << ack_idx.key << std::endl;
    std::cout << " Test cmd key: " << cmd.key << std::endl;
    std::cout << " Test cmd_idx key: " << cmd_idx.key << std::endl;
    // create streamer config for the daq writer
    now = synnax::TimeStamp::now();
    auto streamerConfig = synnax::StreamerConfig{
        std::vector<synnax::ChannelKey>{cmd_idx.key, cmd.key},
        now,
    };

    // instantiate and initialize the daq writer
    TaskHandle taskHandle;
    DAQmxCreateTask("",&taskHandle);
    auto daq_writer = std::make_unique<ni::niDaqWriter>(taskHandle);
    daq_writer->init(config, ack_idx.key);

    // instantiate and initialize the Ctrl pipeline
    auto ctrl = pipeline::Ctrl(streamerConfig, writerConfig, client, std::move(daq_writer));

    /// start the pipeline
    std::cout << std::endl;

    std::cout << "Starting the pipeline" << std::endl;
    ctrl.start();
    std::this_thread::sleep_for(std::chrono::seconds(2));

    /// now write to the command channel, should expect an acknowledgement to be written to the ack channel
    // construct cmd frame to set channel high
    auto time = (synnax::TimeStamp::now()).value;
    std::cout << "Time: " << time << std::endl;
    auto frame = synnax::Frame(2);
    frame.add(cmd_idx.key, synnax::Series(std::vector<uint64_t>{time}, synnax::TIMESTAMP));
    frame.add(cmd.key, synnax::Series(std::vector<uint8_t>{1}));

    // write frame to cmd channel
    ASSERT_TRUE(cmdWriter.write(std::move(frame)));
    auto [end, ok] = cmdWriter.commit();
    std::this_thread::sleep_for(std::chrono::seconds(1)); // sleep (TODO: remove this later)



    // read from ack channel
    auto [ack_frame, recErr] = ackStreamer.read();
    ASSERT_FALSE(recErr) << recErr.message();
    std::cout << "TEST: Ack frame size: " << ack_frame.size() << std::endl;
    //ASSERT_EQ(ack_frame.series->at(1).uint8()[0], 1);
    ASSERT_TRUE(ack_frame.series->at(1).uint8()[0] == 1); // assert ack frame is correct

    // stop the pipeline
    std::cout << "TEST: Ack frame size: " << ack_frame.size() << std::endl;
    ctrl.stop();
    std::cout << "Pipeline stopped" << std::endl;

    // close the writer and streamer
    auto wcErr = cmdWriter.close();
    ASSERT_FALSE(wcErr) << wcErr.message();
    auto wsErr = ackStreamer.closeSend();
    ASSERT_FALSE(wsErr) << wsErr.message();
}