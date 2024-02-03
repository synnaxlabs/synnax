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
    //create task
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

    reader.init(channel_configs, 1000, 20);
    reader.start();
    auto [frame, err] = reader.read();

    std::cout << "Frame size: " <<  frame.size() << std::endl;
    //iterate through each series and print the data
    for (int i = 0; i < frame.series->size(); i++){\
        std::cout << "\n\n Series " << i << ": \n";
        auto s =  frame.series->at(i).float32();
        for (int j = 0; j < s.size(); j++){
            std::cout << s[j] << ", ";
        }
    }

   reader.stop();
}

TEST(NiReaderTests, testReadandInitDigital){
//TODO add asserts (elham)

    std::cout << "Test Read and Init Digital: " << std::endl;
    //create task
    TaskHandle taskHandle;
    DAQmxCreateTask("",&taskHandle);
    auto reader = ni::niDaqReader(taskHandle);

    // create a channel config vector
    std::vector<ni::channel_config> channel_configs;
    channel_configs.push_back(ni::channel_config({"Dev1/port0/line0", 65531,  ni::DIGITAL_IN , -1.0, 1.0}));
    channel_configs.push_back(ni::channel_config({"Dev1/port0/line1", 65532,  ni::DIGITAL_IN , -1.0, 1.0}));
    channel_configs.push_back(ni::channel_config({"Dev1/port0/line2", 65533,  ni::DIGITAL_IN , -1.0, 1.0}));
    channel_configs.push_back(ni::channel_config({"Dev1/port0/line3", 65534,  ni::DIGITAL_IN , -1.0, 1.0}));
    reader.init(channel_configs, 1000, 20);
    reader.start();
    auto [frame, err] = reader.read();

    std::cout << "Frame size: " <<  frame.size() << std::endl;
    //iterate through each series and print the data
    for (int i = 0; i < frame.series->size(); i++){\
        std::cout << "\n\n Series " << i << ": \n";
        auto s =  frame.series->at(i).float32();
        for (int j = 0; j < s.size(); j++){
            std::cout << s[j] << ", ";
        }
    }
    reader.stop();
}