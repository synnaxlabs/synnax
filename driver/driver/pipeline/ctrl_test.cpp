//
// Created by Synnax on 3/6/2024.
//

#include <stdio.h>
#include <thread>

#include <include/gtest/gtest.h>

#include "synnax/synnax.h"
#include "driver/testutil/testutil.h"
#include "ctrl.h"
#include <thread>

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
    auto [ack_idx, tErr] = client->channels.create( // index channel for acks
            "ack_idx",
            synnax::TIMESTAMP,
            0,
            true
    );
    ASSERT_FALSE(tErr) << tErr.message();
    auto [cmd_idx, tErr] = client->channels.create( // index channel for cmd
        "ack_idx",
        synnax::TIMESTAMP,
        0,
        true
    );
    ASSERT_FALSE(tErr) << tErr.message();
    auto [ack, aErr] = client->channels.create( // ack channel
            "ack",
            synnax::UINT8,
            ack_idx.key,
            false
    );
    ASSERT_FALSE(tErr) << tErr.message();
    auto [cmd, cErr] = client->channels.create( // cmd channel
            "cmd",
            synnax::UINT8,
            cmd_idx.key,
            false
    );
    ASSERT_FALSE(tErr) << tErr.message();

    // create config json
    auto config = json{
            {"acq_rate", 300}, // dont actually need these here
            {"stream_rate", 30}, // same as above
            {"device", "Dev1"}
    };

    add_index_channel_JSON(config, "ack_idx", ack_idx.key);
    add_DO_channel_JSON(config, "cmd", cmd.key, ack.key, 0, 0);

    // create a writer to write to cmd channel (for test use only)
    auto now = synnax::TimeStamp::now();
    auto writerConfig = synnax::WriterConfig{
        std::vector<synnax::ChannelKey>{cmd.key, cmd_idx.key},
        now,
        std::vector<synnax::Authority>{synnax::ABSOLUTTE, synnax::ABSOLUTTE},
        synnax::Subject{"test_cmd_writer"},
    };
    auto [cmdWriter,wErr] = client->telem.openWriter(writerConfig);
    ASSERT_FALSE(wErr) << wErr.message();

    // create a streamer to stream ack channel (for in test use only)
    auto streamerConfig = synnax::StreamerConfig{
        std::vector<synnax::ChannelKey>{ack.key, ack_idx.key},
        synnax::TimeStamp::now(),
    };
    auto [ackStreamer, sErr] = client->telem.openStreamer(streamerConfig);
    ASSERT_FALSE(sErr) << wErr.message();

    // create writer config
    auto now = synnax::TimeStamp::now();
    auto writerConfig = synnax::WriterConfig{
        std::vector<synnax::ChannelKey>{ack.key, ack_idx.key},
        now,
        std::vector<synnax::Authority>{synnax::ABSOLUTTE, synnax::ABSOLUTTE},
        synnax::Subject{"test_ctrl_loop"},
    };

    // create streamer config for the daq writer
    now = synnax::TimeStamp::now();
    auto streamerConfig = synnax::StreamerConfig{
        std::vector<synnax::ChannelKey>{cmd.key, cmd_idx.key},
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
    ctrl.start();
    std::this_thread::sleep_for(std::chrono::seconds(1));

    /// now write to the command channel, should expect an acknowledgement to be written to the ack channel
    // construct cmd frame to set channel high
    auto now = (synnax::TimeStamp::now()).value;
    auto frame = synnax::Frame(2);
    frame.add(cmd_idx.key, synnax::Series(std::vector<uint64_t>{now}));
    frame.add(cmd.key, synnax::Series(std::vector<uint8_t>{1}));

    // write frame to cmd channel
    ASSERT_TRUE(cmdWriter->write(std::move(frame)));
    std::this_thread::sleep_for(std::chrono::seconds(1)); // sleep (TODO: remove this later)
    auto [ack_frame, recErr] = ackStreamer->read(); // read the ack frame from the ack channel
    ASSERT_FALSE(recErr) << recErr.message();

//    ASSERT_EQ(ack_frame.series->at(1).uint8()[0], 1);
    ASSERT_TRUE(ack_frame.series->at(1).uint8()[0] == 1); // assert ack frame is correct

    // stop the pipeline
    ctrl.stop();

    // close the writer and streamer
    auto wcErr = cmdWriter->close();
    ASSERT_FALSE(wcErr) << wcErr.message();
    auto wsErr = ackStreamer->close();
    ASSERT_FALSE(wsErr) << wsErr.message();
}