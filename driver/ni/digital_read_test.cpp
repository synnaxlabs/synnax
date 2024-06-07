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
//v
#include <include/gtest/gtest.h>
#include "glog/logging.h"

#include "client/cpp/synnax.h"
#include "driver/ni/ni.h"
#include <stdio.h>
#include "nlohmann/json.hpp"
#include "driver/testutil/testutil.h"
#include <map>


TEST(read_tests, one_digital_channel){
    LOG(INFO) << "test_read_one_digital_channel: "; //<< std::endl;

    // Create NI readerconfig
    auto config = json{
            {"sample_rate", 100}, // dont actually need these here
            {"stream_rate", 20}, // same as above
            {"device_location", "PXI1Slot2_2"},
            {"type", "digital_read"}
            {"test", true},
            {"device", ""}
    };
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
            "d1",
            synnax::UINT8,
            time.key,
            false
    );

    add_DI_channel_JSON(config, "d1", data.key, 0, 0);


    // Synnax infrustructure
    auto task = synnax::Task(
        "my_task",          // task name
        "ni_digital_read",  // task type
        to_string(config)   // task config
    );

    auto mockCtx = std::make_shared<task::MockContext>(client);

    std::this_thread::sleep_for(std::chrono::milliseconds(300));
    // Now construct NI reader
    TaskHandle taskHandle;  
    ni::NiDAQmxInterface::CreateTask("",&taskHandle);

    auto reader = ni::DigitalReadSource(  taskHandle, 
                                          mockCtx, 
                                          task);

    if(reader.init() != 0) std::cout << "Failed to initialize reader" << std::endl;
    reader.start();

    std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
    auto [frame, err] = reader.read();
    std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;

     //iterate through each series and print the data
    for(int i = 0; i < frame.series->size(); i++){
        std::cout << "\n\n Series " << i << ": \n";
        // check series type before casting
        if (frame.series->at(i).data_type == synnax::UINT8){
            auto s =  frame.series->at(i).uint8();
            for (int j = 0; j < s.size(); j++){
                std::cout << (uint32_t)s[j] << ", ";
                ASSERT_TRUE((s[j] == 1) || (s[j] == 0));
            }
        }
        else if(frame.series->at(i).data_type == synnax::TIMESTAMP){
            auto s =  frame.series->at(i).uint64();
            for (int j = 0; j < s.size(); j++){
                std::cout << s[j] << ", ";
                ASSERT_TRUE((s[j] >= initial_timestamp) && (s[j] <= final_timestamp));
            }
        }
    }

    std::cout << std::endl;
    reader.stop()
}



TEST(read_tests, multiple_digital_channels){
    LOG(INFO) << "test_read_multiple_digital_channel: "; //<< std::endl;

    // Create NI readerconfig
    auto config = json{
            {"sample_rate", 1000}, // dont actually need these here
            {"stream_rate", 20}, // same as above
            {"device_location", "PXI1Slot2_2"},
            {"type", "digital_read"},
            {"test", true},
            {"device", ""}
    };

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
    auto [data1, dErr1] = client->channels.create( // analog input channel
            "d1",
            synnax::UINT8,
            time.key,
            false
    );
    ASSERT_FALSE(dErr1) << tErr.message();
    auto [data2, dErr2] = client->channels.create( // analog input channel
            "d2",
            synnax::UINT8,
            time.key,
            false
    );
    ASSERT_FALSE(dErr2) << tErr.message();
    auto [data3, dErr3] = client->channels.create( // analog input channel
            "d3",
            synnax::UINT8,
            time.key,
            false
    );
    ASSERT_FALSE(dErr3) << tErr.message();
    auto [data4, dErr4] = client->channels.create( // analog input channel
            "d4",
            synnax::UINT8,
            time.key,
            false
    );
    add_index_channel_JSON(config, "idx", 1);
    add_DI_channel_JSON(config, "d1", data1.key, 0, 0);
    add_DI_channel_JSON(config, "d2", data2.key, 0, 1);
    add_DI_channel_JSON(config, "d3", data3.key, 0, 2);
    add_DI_channel_JSON(config, "d4", data4.key, 0, 3);



    auto task = synnax::Task(
        "my_task",
        "ni_digital_read",
        to_string(config)
    );

    auto mockCtx = std::make_shared<task::MockContext>(client);

    std::this_thread::sleep_for(std::chrono::milliseconds(300));
    // Now construct NI reader
    TaskHandle taskHandle;  
    ni::NiDAQmxInterface::CreateTask("",&taskHandle);

    auto reader = ni:DigitalReadSource( taskHandle, 
                                        mockCtx, 
                                        task);
    if(reader.init() != 0) std::cout << "Failed to initialize reader" << std::endl;
    reader.start();

    for(int i = 0; i < 15; i++ ) { // test for 50 read cycles
        std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
        auto [frame, err] = reader.read();
        std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;

        //iterate through each series and print the data
        for(int i = 0; i < frame.series->size(); i++){
            std::cout << "\n\n Series " << i << ": \n";
            // check series type before casting
            if (frame.series->at(i).data_type == synnax::UINT8){
                auto s =  frame.series->at(i).uint8();
                for (int j = 0; j < s.size(); j++){
                    std::cout << (uint32_t)s[j] << ", ";
                    ASSERT_TRUE((s[j] == 1) || (s[j] == 0));   
                }
            }
            else if(frame.series->at(i).data_type == synnax::TIMESTAMP){
                auto s =  frame.series->at(i).uint64();
                for (int j = 0; j < s.size(); j++){
                    std::cout << s[j] << ", ";
                    ASSERT_TRUE((s[j] >= initial_timestamp) && (s[j] <= final_timestamp));
                }
            }
        }
        std::cout << std::endl;
    }
    reader.stop();
}