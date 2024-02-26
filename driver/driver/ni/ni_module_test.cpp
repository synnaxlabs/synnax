//
// Created by Synnax on 2/24/2024.
//
#include <stdio.h>
#include <thread>

/// GTest
#include <include/gtest/gtest.h>

/// Internal
#include "synnax/synnax.h"
#include "synnax/testutil/testutil.h"
#include "driver/pipeline/acq.h"
#include "driver/ni/ni_module.h"


TEST(NiTasksTests, testAnalogReaderTask_1_index){

    std::cout << "Test Analog Reader Task: " << std::endl;
   auto config = json{
            {"acq_rate", 300},
            {"stream_rate", 30},
            {"device", "Dev1"},
            {"channels", json::array({
                    json{
                            {"key", "fvJ70Zg4syFZOXY7GbaoX"},
                            {"type", "index"},
                            {"enabled", true},
                            {"port", 0},
                            {"channel", 65538},
                            {"name", "pxi_ai_idx"}},
                    json{
                            {"key", "dBN_7kXytvkNH8YA0N6c-"},
                            {"type", "analogVoltageInput"},
                            {"enabled", true},
                            {"port", 1},
                            {"channel", 65539},
                            {"name", "pxi_ai_1"}
                    }})
            }
   };

    // Create a Factory
    auto factory = niTaskFactory();

    TaskHandle taskHandle;
    DAQmxCreateTask("",&taskHandle);
    auto client_config = synnax::Config{
            "localhost",
            9090,
            "synnax",
            "seldon"};

    auto client = std::make_shared<synnax::Synnax>(client_config);

    // iterate through the configs nd create each channel in the client
    //find the index channel and get the key
    ChannelKey index_key;
    for(auto &channel : config["channels"]){
        if (channel["type"] == "index"){
            auto [index_channel, index_err] = client->channels.create(
                channel["name"],
                synnax::TIMESTAMP,
                0,
                true);
            index_key = index_channel.key;
            ASSERT_FALSE(index_err) << index_err.message();
            channel["channel"] = index_key;
            std::cout << "Index Key: " << index_key << std::endl;
            std::cout << "According to JSON: " << channel["channel"] << std::endl;
        }
    }

    for (auto &channel : config["channels"]){
        if (channel["type"] == "index"){
            continue;
        }
        auto [analog_channel, analog_err] = client->channels.create(
                channel["name"],
                synnax::FLOAT32,
                index_key,
                false);

        channel["channel"] = analog_channel.key;
        ASSERT_FALSE(analog_err) << analog_err.message();
        std::cout << "Key: " << analog_channel.key << std::endl;
        std::cout << "According to JSON: " << channel["channel"] << std::endl;
    }

    std::cout << "Creating config" << std::endl;
    // create the analog reader task
    json config_err;
    auto readerTask = factory.createAnalogReaderTask(taskHandle, client, config, config_err);
    // start acquisition and end after 200s
    std::cout << "Starting acquisition" << std::endl;
    readerTask->startAcquisition();
    std::this_thread::sleep_for(std::chrono::seconds(200));
    readerTask->stopAcquisition();
}


json