// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//
// Created by Synnax on 1/29/2024.
//

#include <include/gtest/gtest.h>
#include "glog/logging.h"

#include "client/cpp/synnax.h"
#include "driver/ni/ni.h"
#include <stdio.h>
#include "nlohmann/json.hpp"
#include "driver/testutil/testutil.h"
#include <map>


 using json = nlohmann::json;

TEST(ReaderTests, test_read_one_analog_channel){
    LOG(INFO) << "test_read_one_analog_channel: "<< std::endl;

    // Create NI readerconfig
    auto config = json{
            {"sample_rate", 100}, 
            {"stream_rate", 20}, 
            {"device_location", "Dev1"},
            {"type", "ni_analog_read"},
            {"test", true},    
            {"device", ""}
    };
    json scale_config = json{
        {"type", "none"}
    };

    /////////////////////////////////////////////// setup synnax test infrustructure
    // create synnax client
    auto client_config = synnax::Config{
            "localhost",
            9090,
            "synnax",
            "seldon"};
    auto client = std::make_shared<synnax::Synnax>(client_config);
    
    // create all the necessary channels in the synnax client
    auto [time, tErr] = client->channels.create( // index channel for analog input channels
            "idx",
            synnax::TIMESTAMP,
            0,
            true
    );
    ASSERT_FALSE(tErr) << tErr.message();
    auto [data, dErr] = client->channels.create( // analog input channel
            "ai",
            synnax::FLOAT32,
            time.key,
            false
    );
    ASSERT_FALSE(dErr) << dErr.message();

    add_AI_channel_JSON(config, "a1", data.key, 0, -10.0, 10.0, "Default", scale_config);

  
    // create synnax task
    auto task = synnax::Task(
            "my_task",          // task name
            "ni_analog_read",   // task type
            to_string(config)   // task config
    );

    auto mockCtx = std::make_shared<task::MockContext>(client);

    std::this_thread::sleep_for(std::chrono::milliseconds(300));
    // Now construct NI reader
    TaskHandle taskHandle;  
    ni::NiDAQmxInterface::CreateTask("",&taskHandle);

    auto reader = ni::AnalogReadSource( taskHandle, 
                                        mockCtx, 
                                        task); // analog reader
    if(reader.init() != 0) std::cout << "Failed to initialize reader" << std::endl;
    reader.start();
    std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
    auto [frame, err] = reader.read();
    std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;

     //iterate through each series and print the data
    uint32_t ai_count = 0;
    for(int i = 0; i < frame.series->size(); i++){
        std::cout << "\n\n Series " << i << ": \n";
        // check series type before casting
        if (frame.series->at(i).data_type == synnax::FLOAT32){
            auto s =  frame.series->at(i).float32();
            for (int j = 0; j < s.size(); j++){
                std::cout << s[j] << ", ";
                // ASSERT_TRUE((s[j] == 1) || (s[j] == 0));
                ASSERT_NEAR(s[j], ai_count, 1);
            }
            ai_count++;
        }
        else if(frame.series->at(i).data_type == synnax::TIMESTAMP){
            auto s =  frame.series->at(i).uint64();
            for (int j = 0; j < s.size(); j++){
                std::cout << s[j] << ", ";
            }
        }
    }
    std::cout << std::endl;
    reader.stop();
}

TEST(ReaderTests, test_read_multiple_analog_channel){
    LOG(INFO) << "test_read_one_analog_channel: "<< std::endl;

    // Create NI readerconfig
    auto config = json{
            {"sample_rate", 100}, 
            {"stream_rate", 20}, 
            {"device_location", "Dev1"},
            {"type", "ni_analog_read"},
            {"test", true},    
            {"device", ""}
    };
    json scale_config = json{
        {"type", "none"}
    };

    /////////////////////////////////////////////// setup synnax test infrustructure
    // create synnax client
    auto client_config = synnax::Config{
            "localhost",
            9090,
            "synnax",
            "seldon"};
    auto client = std::make_shared<synnax::Synnax>(client_config);
    
    // create all the necessary channels in the synnax client
    auto [time, tErr] = client->channels.create( // index channel for analog input channels
            "idx",
            synnax::TIMESTAMP,
            0,
            true
    );
    ASSERT_FALSE(tErr) << tErr.message();
    auto [data, dErr] = client->channels.create( // analog input channel
            "ai",
            synnax::FLOAT32,
            time.key,
            false
    );
    ASSERT_FALSE(dErr) << dErr.message();
    auto [data1, dErr2] = client->channels.create( // analog input channel
            "ai2",
            synnax::FLOAT32,
            time.key,
            false
    );
    ASSERT_FALSE(dErr2) << dErr.message();
    auto [data2, dErr3] = client->channels.create( // analog input channel
            "ai3",
            synnax::FLOAT32,
            time.key,
            false
    );
    ASSERT_FALSE(dErr3) << dErr.message();
    auto [data3, dErr4] = client->channels.create( // analog input channel
            "ai4",
            synnax::FLOAT32,
            time.key,
            false
    );
    ASSERT_FALSE(dErr4) << dErr.message();
    

    add_AI_channel_JSON(config, "a1", data.key, 0, -10.0, 10.0, "Default", scale_config);
    add_AI_channel_JSON(config, "a2", data1.key, 1, -10.0, 10.0, "Default", scale_config);
    add_AI_channel_JSON(config, "a3", data2.key, 2, -10.0, 10.0, "Default", scale_config);
    add_AI_channel_JSON(config, "a4", data3.key, 3, -10.0, 10.0, "Default", scale_config);

  
    // create synnax task
    auto task = synnax::Task(
            "my_task",          // task name
            "ni_analog_read",   // task type
            to_string(config)   // task config
    );

    auto mockCtx = std::make_shared<task::MockContext>(client);

    std::this_thread::sleep_for(std::chrono::milliseconds(300));
    // Now construct NI reader
    TaskHandle taskHandle;  
    ni::NiDAQmxInterface::CreateTask("",&taskHandle);

    auto reader = ni::AnalogReadSource( taskHandle, 
                                        mockCtx, 
                                        task); // analog reader
    if(reader.init() != 0) std::cout << "Failed to initialize reader" << std::endl;
    reader.start();

    auto [frame, err] = reader.read();

     //iterate through each series and print the data
    for(int i = 0; i < 30; i++ ) { // test for 50 read cycles
        std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
        auto [frame, err] = reader.read();
        std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;
        //iterate through each series and print the data
        for(int i = 0; i < frame.series->size(); i++){
            std::cout << "\n\n Series " << i << ": \n";
            // check series type before casting
            if (frame.series->at(i).data_type == synnax::FLOAT32){
                auto s =  frame.series->at(i).float32();
                for (int j = 0; j < s.size(); j++){
                    std::cout << s[j] << ", ";
                    ASSERT_NEAR(s[j], 0, 10); // can be any value of a sine wave from -10 to 10
                }
            }
            else if(frame.series->at(i).data_type == synnax::TIMESTAMP){
                auto s =  frame.series->at(i).uint64();
                for (int j = 0; j < s.size(); j++){
                    std::cout << s[j] << ", ";
                }
            }
        }
        std::cout << std::endl;
    }
    reader.stop();
}
