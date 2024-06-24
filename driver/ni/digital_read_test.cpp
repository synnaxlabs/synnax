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

#include <map>
#include <stdio.h>

#include "client/cpp/synnax.h"
#include "driver/ni/ni.h"
#include "driver/testutil/testutil.h"

#include <include/gtest/gtest.h>
#include "glog/logging.h"

#include "nlohmann/json.hpp"

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//                                                                                                              //
//                                                   Basic Tests                                                //                
//                                                                                                              //
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
TEST(read_tests, one_digital_channel){
    // create synnax client
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    
    // create all the necessary channels in the synnax client
    auto [time, tErr] = client->channels.create("idx",synnax::TIMESTAMP,0,true);
    ASSERT_FALSE(tErr) << tErr.message();

    auto [data, dErr] = client->channels.create("di_channel",synnax::FLOAT32,time.key,false);
    ASSERT_FALSE(dErr) << dErr.message();


    // Create NI readerconfig
    auto config = json{
            {"sample_rate", 100}, // dont actually need these here
            {"stream_rate", 20}, // same as above
            {"device_location", "Dev1"},
            {"type", "digital_read"},
            {"test", true},
            {"device", ""}
    };
    add_DI_channel_JSON(config, "d1", data.key, 0, 0);
    
    // create synnax task
    auto task = synnax::Task("my_task", "ni_digital_read", to_string(config));
    auto mockCtx = std::make_shared<task::MockContext>(client);
    std::this_thread::sleep_for(std::chrono::milliseconds(300));

    // Now construct NI reader
    TaskHandle taskHandle;  
    ni::NiDAQmxInterface::CreateTask("",&taskHandle);

    auto reader = ni::DigitalReadSource( taskHandle, mockCtx, task);
    auto b = breaker::Breaker(breaker::Config{"my-breaker", 1*SECOND, 1, 1});
    
    if(reader.init() != 0) std::cout << "Failed to initialize reader" << std::endl;
    reader.start();
    
    for(int i = 0; i < 15; i++ ) { // test for 50 read cycles
        std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
        auto [frame, err] = reader.read(b);
        std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;
        LOG(INFO) << frame << "\n";
    }

    reader.stop();
}



TEST(read_tests, multiple_digital_channels){
    // setup synnax test infrustructure
    auto client = std::make_shared<synnax::Synnax>(new_test_client());
    // create all the necessary channels in the synnax client
    auto [time, tErr] = client->channels.create( "idx", synnax::TIMESTAMP, 0, true);
    ASSERT_FALSE(tErr) << tErr.message();

    auto [data, dErr] = client->channels.create( "di",synnax::FLOAT32,time.key,false);
    ASSERT_FALSE(dErr) << dErr.message();

    auto [data1, dErr2] = client->channels.create( "di2",synnax::FLOAT32,time.key,false);
    ASSERT_FALSE(dErr2) << dErr.message();

    auto [data2, dErr3] = client->channels.create( "di3",synnax::FLOAT32,time.key,false);
    ASSERT_FALSE(dErr3) << dErr.message();

    auto [data3, dErr4] = client->channels.create( "di4",synnax::FLOAT32,time.key,false);
    ASSERT_FALSE(dErr4) << dErr.message();

    // Create NI readerconfig
    auto config = json{
            {"sample_rate", 1000}, // dont actually need these here
            {"stream_rate", 20}, // same as above
            {"device_location", "Dev1"},
            {"type", "digital_read"},
            {"test", true},
            {"device", ""}
    };

    add_DI_channel_JSON(config, "d1", data.key, 0, 0);
    add_DI_channel_JSON(config, "d2", data1.key, 0, 1);
    add_DI_channel_JSON(config, "d3", data2.key, 0, 2);
    add_DI_channel_JSON(config, "d4", data3.key, 0, 3);

    // create synnax task (name, type, config)
    auto task = synnax::Task( "my_task", "ni_analog_read", to_string(config));
    auto mockCtx = std::make_shared<task::MockContext>(client);
    std::this_thread::sleep_for(std::chrono::milliseconds(300));


    // Now construct NI reader
    TaskHandle taskHandle;  
    ni::NiDAQmxInterface::CreateTask("",&taskHandle);
    auto reader = ni::DigitalReadSource(taskHandle, mockCtx, task);
    auto b = breaker::Breaker(breaker::Config{"my-breaker", 1*SECOND, 1, 1});

    if(reader.init() != 0) std::cout << "Failed to initialize reader" << std::endl;
    reader.start();
    for(int i = 0; i < 15; i++ ) { // test for 50 read cycles
        std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
        auto [frame, err] = reader.read(b);
        std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;
        LOG(INFO) << frame << "\n";
    }
    reader.stop();
}
