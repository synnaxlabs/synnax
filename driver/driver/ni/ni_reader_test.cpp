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

#include "client/cpp/synnax/synnax.h"
#include "driver/driver/ni/ni_reader.h"
#include <stdio.h>
#include "nlohmann/json.hpp"
#include "driver/driver/testutil/testutil.h"


 using json = nlohmann::json;

/*
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Functional Tests                                                                                             //
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////

TEST(NiReaderTests, testReadandInitAnalog){
    //TODO add asserts (elham)
    std::cout << "Test read and init Analog: " << std::endl;

//    create task
    TaskHandle taskHandle;
    DAQmxCreateTask("",&taskHandle);
    auto reader = ni::niDaqReader(taskHandle);

    // create a channel config vector
    std::vector<ni::channel_config> channel_configs;
    channel_configs.push_back(ni::channel_config({"Dev1/ai0", 65531,  ni::ANALOG_VOLTAGE_IN , -10.0, 10.0}));
    channel_configs.push_back(ni::channel_config({"Dev1/ai1", 65532,  ni::ANALOG_VOLTAGE_IN , -10.0, 10.0}));
    channel_configs.push_back(ni::channel_config({"Dev1/ai2", 65533,  ni::ANALOG_VOLTAGE_IN , -10.0, 10.0}));
    channel_configs.push_back(ni::channel_config({"Dev1/ai3", 65534,  ni::ANALOG_VOLTAGE_IN , -10.0, 10.0}));
    channel_configs.push_back(ni::channel_config({"Dev1/ai4", 65535,  ni::ANALOG_VOLTAGE_IN , -10.0, 10.0}));
    channel_configs.push_back(ni::channel_config({"Dev1/ai5", 65536,  ni::ANALOG_VOLTAGE_IN , -10.0, 10.0}));
    channel_configs.push_back(ni::channel_config({"Dev1/ai6", 65537,  ni::ANALOG_VOLTAGE_IN , -10.0, 10.0}));
    channel_configs.push_back(ni::channel_config({"Dev1/ai7", 65538,  ni::ANALOG_VOLTAGE_IN , -10.0, 10.0}));
    channel_configs.push_back(ni::channel_config({"", 0,  ni::INDEX_CHANNEL ,  0, 0}));

    reader.init(channel_configs, 1000, 20);
    reader.start();
    auto [frame, err] = reader.read();

    assert(frame.series->size() == 9);
//    std::cout << "Frame size: " <<  frame.size() << std::endl;

//    iterate through each series and print the data
    for (int i = 0; i < frame.series->size(); i++){
        std::cout << "\n\n Series " << i << ": \n";
        // check series type before casting
        if (frame.series->at(i).data_type == synnax::FLOAT32){
            auto s =  frame.series->at(i).float32();
            for (int j = 0; j < s.size(); j++){
                std::cout << s[j]<< ", ";
            }
        }
        else if(frame.series->at(i).data_type == synnax::TIMESTAMP){
            auto s =  frame.series->at(i).uint64();
            for (int j = 0; j < s.size(); j++){
                std::cout << s[j] << ", ";
            }
        }
    }
   reader.stop();
}

TEST(NiReaderTests, testReadandInitDigital){
    std::cout << "Test Read and Init Digital: " << std::endl;
    //create task
    TaskHandle taskHandle;
    DAQmxCreateTask("",&taskHandle);
    auto reader = ni::niDaqReader(taskHandle);

    // create a channel config vector
    std::vector<ni::channel_config> channel_configs;
    channel_configs.push_back(ni::channel_config({"PXI1Slot2_2/port0/line0", 65531,  ni::DIGITAL_IN , -1.0, 1.0}));
    channel_configs.push_back(ni::channel_config({"PXI1Slot2_2/port0/line1", 65532,  ni::DIGITAL_IN , -1.0, 1.0}));
    channel_configs.push_back(ni::channel_config({"PXI1Slot2_2/port0/line2", 65533,  ni::DIGITAL_IN , -1.0, 1.0}));
    channel_configs.push_back(ni::channel_config({"PXI1Slot2_2/port0/line3", 65534,  ni::DIGITAL_IN , -1.0, 1.0}));
    channel_configs.push_back(ni::channel_config({"", 65538,  ni::INDEX_CHANNEL ,  0, 0}));

    reader.init(channel_configs, 1000, 20);
    reader.start();
    auto [frame, err] = reader.read();

//    assert(frame.series->size() == 5);
//    std::cout << "Frame size: " <<  frame.size() << std::endl;

    //iterate through each series and print the data
    for(int i = 0; i < frame.series->size(); i++){
        std::cout << "\n\n Series " << i << ": \n";
        // check series type before casting
        if (frame.series->at(i).data_type == synnax::UINT8){
            auto s =  frame.series->at(i).uint8();
            for (int j = 0; j < s.size(); j++){
                std::cout << (uint32_t)s[j] << ", ";
            }
        }
        else if(frame.series->at(i).data_type == synnax::TIMESTAMP){
            auto s =  frame.series->at(i).uint64();
            for (int j = 0; j < s.size(); j++){
                std::cout << s[j] << ", ";
            }
        }
    }
    reader.stop();
}

TEST(NiReaderTests, TestsReadFromMultipleTasks){
    std::cout << "Test Read from Multiple Tasks: " << std::endl;
    //create task
    TaskHandle taskHandle;
    DAQmxCreateTask("",&taskHandle);
    auto reader = ni::niDaqReader(taskHandle);

//    // create a channel config vector
    std::vector<ni::channel_config> channel_configs;
    channel_configs.push_back(ni::channel_config({"Dev1/ai0", 65531,  ni::DIGITAL_IN , -1.0, 1.0}));
    channel_configs.push_back(ni::channel_config({"Dev1/ai1", 65532,  ni::DIGITAL_IN , -1.0, 1.0}));
    channel_configs.push_back(ni::channel_config({"Dev1/ai2", 65533,  ni::DIGITAL_IN , -1.0, 1.0}));
    channel_configs.push_back(ni::channel_config({"", 0,  ni::INDEX_CHANNEL ,  0, 0}));
    reader.init(channel_configs, 1000, 20);

    TaskHandle taskHandle2;
    DAQmxCreateTask("",&taskHandle2);
    auto reader2 = ni::niDaqReader(taskHandle2);
    std::vector<ni::channel_config> channel_configs2;
    channel_configs2.push_back(ni::channel_config({"PXI1Slot2_2/port0/line0", 65531,  ni::ANALOG_VOLTAGE_IN , -10.0, 10.0}));
    channel_configs2.push_back(ni::channel_config({"PXI1Slot2_2/port0/line1", 65532,  ni::ANALOG_VOLTAGE_IN , -10.0, 10.0}));
//    reader2.init(channel_configs2, 1000, 20);

    printf("Starting reader 1\n");
    reader.start();
    printf("Starting reader 2\n");
//    reader2.start();
//
//    auto [frame, err] = reader.read();
//    auto [frame2, err2] = reader2.read();
//    reader2.stop();
    reader.stop();
}

TEST(NiReaderTests, TestJSONParsingDigital){
    //create task
    TaskHandle taskHandle;
    DAQmxCreateTask("",&taskHandle);
    auto reader = ni::niDaqReader(taskHandle);

    //create config json
    auto config = json{
        {"acq_rate", 100}, // dont actually need these here
        {"stream_rate", 20}, // same as above
        {"device", "PXI1Slot2_2"}
    };
    add_index_channel_JSON(config, "idx", idx.key);
    add_DI_channel_JSON(config, "d1", d1.key, 0, 0);
    reader->init(config, config["acq_rate"], config["stream_rate"]);

        auto [frame, err] = reader.read();

        //    assert(frame.series->size() == 5);
        //    std::cout << "Frame size: " <<  frame.size() << std::endl;

        //iterate through each series and print the data
        for(int i = 0; i < frame.series->size(); i++){
                std::cout << "\n\n Series " << i << ": \n";
                // check series type before casting
                if (frame.series->at(i).data_type == synnax::UINT8){
                auto s =  frame.series->at(i).uint8();
                for (int j = 0; j < s.size(); j++){
                std::cout << (uint32_t)s[j] << ", ";
            }
        }
        else if(frame.series->at(i).data_type == synnax::TIMESTAMP){
        auto s =  frame.series->at(i).uint64();
        for (int j = 0; j < s.size(); j++){
                std::cout << s[j] << ", ";
                }
            }
        }
        reader.stop();
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//                                      Error Handling                                                          //
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////

TEST(NiReaderTests, TestsReadFromMultipleTasks){
    std::cout << "Test Read from Multiple Tasks: " << std::endl;
    //create task
    TaskHandle taskHandle;
    DAQmxCreateTask("",&taskHandle);
    auto reader = ni::niDaqReader(taskHandle);

    //    // create a channel config vector
    std::vector<ni::channel_config> channel_configs;
    channel_configs.push_back(ni::channel_config({"Dev1/ai0", 65531,  ni::DIGITAL_IN , -1.0, 1.0}));
    channel_configs.push_back(ni::channel_config({"Dev1/ai1", 65532,  ni::DIGITAL_IN , -1.0, 1.0}));
    channel_configs.push_back(ni::channel_config({"Dev1/ai2", 65533,  ni::DIGITAL_IN , -1.0, 1.0}));
    channel_configs.push_back(ni::channel_config({"", 0,  ni::INDEX_CHANNEL ,  0, 0}));
    reader.init(channel_configs, 1000, 20);

    TaskHandle taskHandle2;
    DAQmxCreateTask("",&taskHandle2);
    auto reader2 = ni::niDaqReader(taskHandle2);/
    std::vector<ni::channel_config> channel_configs2;
    channel_configs2.push_back(ni::channel_config({"PXI1Slot2_2/port0/line0", 65531,  ni::ANALOG_VOLTAGE_IN , -10.0, 10.0}));
    channel_configs2.push_back(ni::channel_config({"PXI1Slot2_2/port0/line1", 65532,  ni::ANALOG_VOLTAGE_IN , -10.0, 10.0}));
        reader2.init(channel_configs2, 1000, 20);

    printf("Starting reader 1\n");
    reader.start();
    printf("Starting reader 2\n");
    reader2.start();

    auto [frame, err] = reader.read();
    auto [frame2, err2] = reader2.read();
    reader2.stop();
    reader.stop();
}

// TODO: Create Function stubs to link with ni_writer.cpp and ni_reader.cpp to be able to test all the diff error sequences

*/

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Functional Tests                                                                                             //
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////

TEST(NiReaderTests, test_read_one_digital_channel){
    LOG(INFO) << "test_read_one_digital_channel: "; //<< std::endl;
    //TODO: need to open server for state writer?

    // Create NI readerconfig
    auto config = json{
            {"acq_rate", 100}, // dont actually need these here
            {"stream_rate", 20}, // same as above
            {"device_name", "PXI1Slot2_2"}
    };
    add_index_channel_JSON(config, "idx", 1);
    add_DI_channel_JSON(config, "d1", 65531, 0, 0);


    // Synnax infrustructure
    auto client = std::make_shared<synnax::Synnax>(new_test_client());

    auto task = synnax::Task(
        "my_task",
        "NI_digitalRead",
        to_string(config)
    );

    auto mockCtx = std::make_shared<task::MockContext>(client);

    std::this_thread::sleep_for(std::chrono::milliseconds(300));
    // Now construct NI reader
    TaskHandle taskHandle;  
    DAQmxCreateTask("",&taskHandle);

    auto reader = ni::niDaqReader(  taskHandle, 
                                    mockCtx, 
                                    task, 
                                    true);
    reader.start();
    auto [frame, err] = reader.read();


     //iterate through each series and print the data
    for(int i = 0; i < frame.series->size(); i++){
        std::cout << "\n\n Series " << i << ": \n";
        // check series type before casting
        if (frame.series->at(i).data_type == synnax::UINT8){
            auto s =  frame.series->at(i).uint8();
            for (int j = 0; j < s.size(); j++){
                std::cout << (uint32_t)s[j] << ", ";
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
    reader.stop();
}
