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
    LOG(INFO) << "Test Control Pipeline with an NI Digital Writer: " << std::endl;
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

    // create synnax task
    auto task = synnax::Task(
            "my_task",
            "NI_digital_writer",
            to_string(config)
    );

    auto mockCtx = std::make_shared<task::MockContext>(client);
    std::this_thread::sleep_for(std::chrono::milliseconds(10));

    // create a writer to write to cmd channel (for test use only)
    auto cmdWriterConfig = synnax::WriterConfig{
                .channels = std::vector<synnax::ChannelKey>{cmd_idx.key, cmd.key},
                .start = TimeStamp::now(),
                .mode = synnax::WriterStreamOnly};

    auto [cmdWriter, wErr] = client->telem.openWriter(cmdWriterConfig);
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
    auto daq_writer = std::make_unique<ni::daqWriter>(taskHandle, mockCtx, task); 
;

    // create a writer to write to STATE channel (for test use only)
    auto ackWriterConfig = synnax::WriterConfig{
                .channels = std::vector<synnax::ChannelKey>{ack_idx.key, ack.key},
                .start = TimeStamp::now(),
                .mode = synnax::WriterStreamOnly};

    // instantiate and initialize the Acquisition pipeline to actually write to the server
    assert(daq_writer->writer_state_source != nullptr);
    auto acq = pipeline::Acquisition(mockCtx, ackWriterConfig, daq_writer->writer_state_source, breaker_config);
    
    // create and open a state streamer to read frames that the acq pipe writes to the server
    auto streamer_config = synnax::StreamerConfig{
                .channels = std::vector<synnax::ChannelKey>{ack_idx.key, ack.key},
                .start = TimeStamp::now(),
        };
    auto [streamer, sErr2] = mockCtx->client->telem.openStreamer(streamer_config);
    ASSERT_FALSE(sErr2) << sErr2.message();

    // instantiate and initialize the Control pipeline
    auto ctrl = pipeline::Control(mockCtx, cmdStreamerConfig, std::move(daq_writer), breaker_config);


    // start the control pipeline
    LOG(INFO) << "Opening Pipeline: " << std::endl;
    ctrl.start();
    acq.start();

    //sleep
    std::this_thread::sleep_for(std::chrono::seconds(1));

    //////////////////////////////////////////// write a 1 to the cmd channel ////////////////////////////////////////////
    LOG(INFO) << "Commanding a logic high: " << std::endl;
    // construct frame
    auto cmd_frame = synnax::Frame(2);
    cmd_frame.add(cmd_idx.key, synnax::Series(std::vector<uint64_t>{synnax::TimeStamp::now().value}, synnax::TIMESTAMP));
    cmd_frame.add(cmd.key, synnax::Series(std::vector<uint8_t>{1}));
    ASSERT_TRUE(cmdWriter.write(std::move(cmd_frame))); // TODO: remove -> isnt necessary

    // do initial read before state update, should be 0
        auto [state_frame, err3] = streamer.read();
        ASSERT_FALSE(err3) << err3.message();
        auto s = state_frame.series->at(1).uint8();
        LOG(INFO) << "State: " << (int)s[0] << std::endl;
        ASSERT_TRUE(s[0] == 0);

    // keep reading state channel and printing state
    for (int i = 0; i < 5; i++){
        auto [state_frame, err3] = streamer.read();
        ASSERT_FALSE(err3) << err3.message();
        auto s = state_frame.series->at(1).uint8();
        LOG(INFO) << "State: " << (int)s[0] << std::endl;
        ASSERT_TRUE(s[0] == 1);
    }
    
    //////////////////////////////////////////// write a 0 to the cmd channel ////////////////////////////////////////////
    // construct frame
    LOG(INFO) << "Commanding a logic low:  " << std::endl;
    auto cmd_frame2 = synnax::Frame(2);
    cmd_frame2.add(cmd_idx.key, synnax::Series(std::vector<uint64_t>{synnax::TimeStamp::now().value}, synnax::TIMESTAMP));
    cmd_frame2.add(cmd.key, synnax::Series(std::vector<uint8_t>{0}));
    ASSERT_TRUE(cmdWriter.write(std::move(cmd_frame2)));


    // keep reading state channel and printing state
    for (int i = 0; i < 5; i++){
        auto [state_frame, err3] = streamer.read();
        ASSERT_FALSE(err3) << err3.message();
        auto s = state_frame.series->at(1).uint8();
        LOG(INFO) << "State: " << (int)s[0] << std::endl;
        ASSERT_TRUE(s[0] == 0);
    }

    // daq_writer->stop();
    acq.stop();
    ctrl.stop();
    LOG(INFO) << "Control Pipeline stopped" << std::endl;
    
}




TEST(ControlPipelineTests, test_control_NI_digital_writer_multiple_channels){
    LOG(INFO) << "Test Control Pipeline with an NI Digital Writer: " << std::endl;
     // create synnax client
    auto client_config = synnax::Config{
                "localhost",
                9090,
                "synnax",
                "seldon"};
    auto client = std::make_shared<synnax::Synnax>(client_config);

    
    // create all the necessary channels in the synnax client
    // index channels
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

    // cmd and ack channel pairs 
    // do1
    auto [ack, aErr] = client->channels.create( // ack channel
            "do1_state",
            synnax::UINT8,
            ack_idx.key,
            false
    );
    ASSERT_FALSE(aErr) << aErr.message();
    auto [cmd, cErr] = client->channels.create( // cmd channel
            "do1_cmd",
            synnax::UINT8,
            cmd_idx.key,
            false
    );
    ASSERT_FALSE(cErr) << cErr.message();

    // do2
    auto [ack2, aErr2] = client->channels.create( // ack channel
            "do2_state",
            synnax::UINT8,
            ack_idx.key,
            false
    );
    ASSERT_FALSE(aErr2) << aErr2.message();
    auto [cmd2, cErr2] = client->channels.create( // cmd channel
            "do2_cmd",
            synnax::UINT8,
            cmd_idx.key,
            false
    );
    ASSERT_FALSE(cErr2) << cErr2.message();

     // do3
    auto [ack3, aErr3] = client->channels.create( // ack channel
            "do3_state",
            synnax::UINT8,
            ack_idx.key,
            false
    );
    ASSERT_FALSE(aErr3) << aErr3.message();
    auto [cmd3, cErr3] = client->channels.create( // cmd channel
            "do3_cmd",
            synnax::UINT8,
            cmd_idx.key,
            false
    );
    ASSERT_FALSE(cErr3) << cErr3.message();

    // do4
    auto [ack4, aErr4] = client->channels.create( // ack channel
            "do4_state",
            synnax::UINT8,
            ack_idx.key,
            false
    );
    ASSERT_FALSE(aErr4) << aErr4.message();
    auto [cmd4, cErr4] = client->channels.create( // cmd channel
            "do4_cmd",
            synnax::UINT8,
            cmd_idx.key,
            false
    );
    ASSERT_FALSE(cErr4) << cErr4.message();

    // create reader config json
    auto config = json{
            {"device_name", "Dev1"},
            {"stream_rate", 1}
    };

    add_index_channel_JSON(config, "do1_idx", cmd_idx.key);
    add_DO_channel_JSON(config, "do1_cmd", cmd.key, ack.key, 0, 0);
    add_DO_channel_JSON(config, "do2_cmd", cmd2.key, ack2.key, 0, 1);
    add_DO_channel_JSON(config, "do3_cmd", cmd3.key, ack3.key, 0, 2);
    add_DO_channel_JSON(config, "do4_cmd", cmd4.key, ack4.key, 0, 3);

    add_drive_state_index_channel_JSON(config, "do_state_idx", ack_idx.key);

    // create synnax task
    auto task = synnax::Task(
            "my_task",
            "NI_digital_writer",
            to_string(config)
    );

    auto mockCtx = std::make_shared<task::MockContext>(client);
    std::this_thread::sleep_for(std::chrono::milliseconds(10));

    // create a writer to write to cmd channel (for test use only)
    auto cmdWriterConfig = synnax::WriterConfig{
                .channels = std::vector<synnax::ChannelKey>{cmd_idx.key, cmd.key, cmd2.key, cmd3.key, cmd4.key},
                .start = TimeStamp::now(),
                .mode = synnax::WriterStreamOnly};

    auto [cmdWriter, wErr] = client->telem.openWriter(cmdWriterConfig);
    ASSERT_FALSE(wErr) << wErr.message();

    // create a streamer to stream do_state channel (for in test use only)
    auto doStateStreamerConfig = synnax::StreamerConfig{
        .channels = std::vector<synnax::ChannelKey>{ack_idx.key, ack.key, ack2.key, ack3.key, ack4.key},
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
        .channels = std::vector<synnax::ChannelKey>{cmd_idx.key, cmd.key, cmd2.key, cmd3.key, cmd4.key},
        .start = TimeStamp::now(),
    };

    // instantiate and initialize the daq writer
    TaskHandle taskHandle;
    DAQmxCreateTask("",&taskHandle);
    auto daq_writer = std::make_unique<ni::daqWriter>(taskHandle, mockCtx, task); 


    // create a writer to write to STATE channel (for test use only)
    auto ackWriterConfig = synnax::WriterConfig{
                .channels = std::vector<synnax::ChannelKey>{ack_idx.key, ack.key, ack2.key, ack3.key, ack4.key},
                .start = TimeStamp::now(),
                .mode = synnax::WriterStreamOnly};

    // instantiate and initialize the Acquisition pipeline to actually write to the server
    assert(daq_writer->writer_state_source != nullptr);
    auto acq = pipeline::Acquisition(mockCtx, ackWriterConfig, daq_writer->writer_state_source, breaker_config);
    
    // create and open a state streamer to read frames that the acq pipe writes to the server
    auto streamer_config = synnax::StreamerConfig{
                .channels = std::vector<synnax::ChannelKey>{ack_idx.key, ack.key, ack2.key, ack3.key, ack4.key},
                .start = TimeStamp::now(),
        };
    auto [streamer, sErr2] = mockCtx->client->telem.openStreamer(streamer_config);
    ASSERT_FALSE(sErr2) << sErr2.message();

    // instantiate and initialize the Control pipeline
    auto ctrl = pipeline::Control(mockCtx, cmdStreamerConfig, std::move(daq_writer), breaker_config);


    // start the control pipeline
    LOG(INFO) << "Opening Pipeline: " << std::endl;
    ctrl.start();
    acq.start();

    //sleep
    std::this_thread::sleep_for(std::chrono::seconds(1));

    // TODO: CODE DEDUP

    //////////////////////////////////////////// write a 1010 to the cmd channel ////////////////////////////////////////////
    {
        LOG(INFO) << "Sending 1 0 1 0: " << std::endl;
        auto cmd_vec = std::vector<uint8_t>{1,0,1,0 };
        // construct frame
        auto cmd_frame = synnax::Frame(2);
        cmd_frame.add(cmd_idx.key, synnax::Series(std::vector<uint64_t>{synnax::TimeStamp::now().value}, synnax::TIMESTAMP));

        cmd_frame.add(cmd.key, synnax::Series(std::vector<uint8_t>{1}));
        cmd_frame.add(cmd2.key, synnax::Series(std::vector<uint8_t>{0}));
        cmd_frame.add(cmd3.key, synnax::Series(std::vector<uint8_t>{1}));
        cmd_frame.add(cmd4.key, synnax::Series(std::vector<uint8_t>{0}));

        ASSERT_TRUE(cmdWriter.write(std::move(cmd_frame))); // TODO: remove -> isnt necessary

        // do initial read before state update, should be 0
            auto [state_frame, err3] = streamer.read();
            ASSERT_FALSE(err3) << err3.message();
            auto s = state_frame.series->at(1).uint8();
            LOG(INFO) << "State: " << (int)s[0] << std::endl;
            ASSERT_TRUE(s[0] == 0);

        // keep reading state channel and printing state
        for (int i = 0; i < 2; i++){
            auto [state_frame, err3] = streamer.read();
            ASSERT_FALSE(err3) << err3.message();
            // auto s = state_frame.series->at(1).uint8();
            // LOG(INFO) << "State: " << (int)s[0] << std::endl;
            // ASSERT_TRUE(s[0] == 1);


            int cmd_count = 0;
            for(int i = 0; i < state_frame.series->size(); i++){
                std::cout << "\n\n Series " << i << ": \n";
                // check series type before casting
                if (state_frame.series->at(i).data_type == synnax::UINT8){
                    auto s =  state_frame.series->at(i).uint8();
                    for (int j = 0; j < s.size(); j++){
                        std::cout << (uint32_t)s[j] << ", ";
                        ASSERT_EQ(s[j], cmd_vec[cmd_count]);
                    }
                    cmd_count++;
                }
                else if(state_frame.series->at(i).data_type == synnax::TIMESTAMP){
                    auto s =  state_frame.series->at(i).uint64();
                    for (int j = 0; j < s.size(); j++){
                        std::cout << s[j] << ", ";
                    }
                }
            }
            std::cout << std::endl;
        }
    }
    //////////////////////////////////////////// write a 1010 to the cmd channel ////////////////////////////////////////////
    {
        LOG(INFO) << "Sending 0 1 0 1: " << std::endl;
        auto cmd_vec = std::vector<uint8_t>{0,1,0,1};
        // construct frame
        auto cmd_frame = synnax::Frame(2);
        cmd_frame.add(cmd_idx.key, synnax::Series(std::vector<uint64_t>{synnax::TimeStamp::now().value}, synnax::TIMESTAMP));

        cmd_frame.add(cmd.key, synnax::Series(std::vector<uint8_t>{0}));
        cmd_frame.add(cmd2.key, synnax::Series(std::vector<uint8_t>{1}));
        cmd_frame.add(cmd3.key, synnax::Series(std::vector<uint8_t>{0}));
        cmd_frame.add(cmd4.key, synnax::Series(std::vector<uint8_t>{1}));

        ASSERT_TRUE(cmdWriter.write(std::move(cmd_frame))); // TODO: remove -> isnt necessary

        // do initial read before state update, should be 0
            auto [state_frame, err3] = streamer.read();
            ASSERT_FALSE(err3) << err3.message();
            auto s = state_frame.series->at(1).uint8();
            LOG(INFO) << "State: " << (int)s[0] << std::endl;
            ASSERT_TRUE(s[0] == 0);

        // keep reading state channel and printing state
        for (int i = 0; i < 2; i++){
            auto [state_frame, err3] = streamer.read();
            ASSERT_FALSE(err3) << err3.message();
            // auto s = state_frame.series->at(1).uint8();
            // LOG(INFO) << "State: " << (int)s[0] << std::endl;
            // ASSERT_TRUE(s[0] == 1);


            int cmd_count = 0;
            for(int i = 0; i < state_frame.series->size(); i++){
                std::cout << "\n\n Series " << i << ": \n";
                // check series type before casting
                if (state_frame.series->at(i).data_type == synnax::UINT8){
                    auto s =  state_frame.series->at(i).uint8();
                    for (int j = 0; j < s.size(); j++){
                        std::cout << (uint32_t)s[j] << ", ";
                        ASSERT_EQ(s[j], cmd_vec[cmd_count]);
                    }
                    cmd_count++;
                }
                else if(state_frame.series->at(i).data_type == synnax::TIMESTAMP){
                    auto s =  state_frame.series->at(i).uint64();
                    for (int j = 0; j < s.size(); j++){
                        std::cout << s[j] << ", ";
                    }
                }
            }
            std::cout << std::endl;
        }
    }
    
    LOG(INFO) << "Closing Pipeline: " << std::endl;
    acq.stop();
    LOG(INFO) << "Acquisition Pipeline stopped" << std::endl;
    ctrl.stop();
    LOG(INFO) << "Control Pipeline stopped" << std::endl;
}



