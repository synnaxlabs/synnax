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
#include "synnax/synnax.h"
#include "ni_reader.h"
#include <stdio.h>

/// @brief it should read data from a daq and correctly construct a synnax frame from the dataI

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
    auto [frame, err] = reader.read();\

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