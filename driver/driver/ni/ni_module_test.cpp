//
// Created by Synnax on 2/24/2024.
//
#include <stdio.h>
#include <thread>

/// GTest
#include <include/gtest/gtest.h>

/// Internal
#include "synnax/synnax.h"
//#include "synnax/testutil/testutil.h"
#include "driver/pipeline/acq.h"
#include "driver/ni/ni_module.h"
#include "driver/testutil/testutil.h"


//TEST(NiTasksTests, testAnalogReaderTask_1_index){
//    std::cout << "Test Analog Reader Task: " << std::endl;
//   auto config = json{
//            {"acq_rate", 300},
//            {"stream_rate", 30},
//            {"device", "Dev1"},
//            {"channels", json::array({
//                    json{
//                            {"key", "fvJ70Zg4syFZOXY7GbaoX"},
//                            {"type", "index"},
//                            {"enabled", true},
//                            {"port", 0},
//                            {"channel", 65538},
//                            {"name", "pxi_ai_idx"}},
//                    json{
//                            {"key", "dBN_7kXytvkNH8YA0N6c-"},
//                            {"type", "analogVoltageInput"},
//                            {"enabled", true},
//                            {"port", 1},
//                            {"channel", 65539},
//                            {"name", "pxi_ai_1"}
//                    }})
//            }
//   };
//
//    // Create a Factory
//    auto factory = niTaskFactory();
//
//    TaskHandle taskHandle;
//    DAQmxCreateTask("",&taskHandle);
//    auto client_config = synnax::Config{
//            "localhost",
//            9090,
//            "synnax",
//            "seldon"};
//
//    auto client = std::make_shared<synnax::Synnax>(client_config);
//
//    // iterate through the configs nd create each channel in the client
//    //find the index channel and get the key
//    ChannelKey index_key;
//    for(auto &channel : config["channels"]){
//        if (channel["type"] == "index"){
//            auto [index_channel, index_err] = client->channels.create(
//                channel["name"],
//                synnax::TIMESTAMP,
//                0,
//                true);
//            index_key = index_channel.key;
//            ASSERT_FALSE(index_err) << index_err.message();
//            channel["channel"] = index_key;
//            std::cout << "Index Key: " << index_key << std::endl;
//            std::cout << "According to JSON: " << channel["channel"] << std::endl;
//        }
//    }
//
//    for (auto &channel : config["channels"]){
//        if (channel["type"] == "index"){
//            continue;
//        }
//        auto [analog_channel, analog_err] = client->channels.create(
//                channel["name"],
//                synnax::FLOAT32,
//                index_key,
//                false);
//
//        channel["channel"] = analog_channel.key;
//        ASSERT_FALSE(analog_err) << analog_err.message();
//        std::cout << "Key: " << analog_channel.key << std::endl;
//        std::cout << "According to JSON: " << channel["channel"] << std::endl;
//    }
//
//    std::cout << "Creating config" << std::endl;
//    // create the analog reader task
//    json config_err;
//    auto readerTask = factory.createAnalogReaderTask(taskHandle, client, config, config_err);
//    // start acquisition and end after 200s
//    std::cout << "Starting acquisition" << std::endl;
//    readerTask->startAcquisition();
//    std::this_thread::sleep_for(std::chrono::seconds(200));
//    readerTask->stopAcquisition();
//}
//
//
//TEST(NiTasksTests, testDigitalReaderTask_1){
//    std::cout << "Test Acq Digital Reads: " << std::endl;
//
//    //create synnax client config
//    auto client_config = synnax::Config{
//            "localhost",
//            9090,
//            "synnax",
//            "seldon"};
//    auto client = std::make_shared<synnax::Synnax>(client_config);
//
//    // create all the necessary channels in the synnax client
//    auto [idx, tErr1] = client->channels.create( // index channel for digital input channels
//            "idx",
//            synnax::TIMESTAMP,
//            0,
//            true
//    );
//    ASSERT_FALSE(tErr1) << tErr1.message();
//    auto [d1, dErr] = client->channels.create( // digital input channel
//            "d1",
//            synnax::UINT8,
//            idx.key,
//            false
//    );
//    ASSERT_FALSE(dErr) << dErr.message();
//
//    //create config json
//    auto config = json{
//            {"acq_rate", 100}, // dont actually need these here
//            {"stream_rate", 20}, // same as above
//            {"device", "PXI1Slot2_2"}
//    };
//    add_index_channel_JSON(config, "idx", idx.key);
//    add_DI_channel_JSON(config, "d1", d1.key, 0, 0);
//
//    //set up factory
//    json config_err;
//    auto factory = niTaskFactory();
//
//    // create task
//    TaskHandle taskHandle;
//    DAQmxCreateTask("",&taskHandle);
//    auto readerTask = factory.createDigitalReaderTask(taskHandle, client, config, config_err);
//
//    //begin running task
//    std::cout << "Starting acquisition" << std::endl;
//    readerTask->startAcquisition();
//    std::this_thread::sleep_for(std::chrono::seconds(200));
//    readerTask->stopAcquisition();
//}
//
TEST(NiTasksTests, testsDigitalWriterTask_1){
    std::cout << "Test Digital Writer Task: " << std::endl;

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
            {"device", "Dev1"}
    };
    add_ackIndex_channel_JSON(config, "ack_idx", ack_idx.key);
    add_DO_channel_JSON(config, "cmd", cmd.key, ack.key, 0, 0);


    // create a writer to write to cmd channel (for test use only)
    auto cmdWriterConfig = synnax::WriterConfig{
        std::vector<synnax::ChannelKey>{cmd_idx.key, cmd.key},
        synnax::TimeStamp::now(),
        std::vector<synnax::Authority>{synnax::ABSOLUTTE, synnax::ABSOLUTTE},
        synnax::Subject{"test_cmd_writer"}
    };
    auto [cmdWriter,wErr] = client->telem.openWriter(cmdWriterConfig);
    ASSERT_FALSE(wErr) << wErr.message();

    // create a streamer to stream ack channel (for in test use only)
    auto ackStreamerConfig = synnax::StreamerConfig{
        std::vector<synnax::ChannelKey>{ack_idx.key, ack.key},
        synnax::TimeStamp::now(),
    };
    auto [ackStreamer, zErr] = client->telem.openStreamer(ackStreamerConfig);
    ASSERT_FALSE(zErr) << zErr.message();

    // set up factory
    json config_err;
    auto factory = niTaskFactory();

    // create task
    TaskHandle taskHandle;
    DAQmxCreateTask("",&taskHandle);
    auto writerTask = factory.createDigitalWriterTask(taskHandle, client, config, config_err);

    // begin running task
    std::cout << "Starting ctrl" << std::endl;
    writerTask->startAcquisition();
    std::this_thread::sleep_for(std::chrono::seconds(2));

    // now write to the command channel, should expect an acknowledgement to be written to the ack channel
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

    std::cout << "Reading from ack channel" << std::endl;
    // read from the ack channel
    auto [ack_frame, ack_err] = ackStreamer.read();
    ASSERT_TRUE(ack_err.ok()) << ack_err.message();
    std::cout << "Ack Frame size: " << ack_frame.size() << std::endl;

    ASSERT_TRUE(ack_frame.series->at(1).uint8()[0] == 1);
    writerTask->stopAcquisition();
}