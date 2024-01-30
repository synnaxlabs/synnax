//
// Created by Synnax on 1/29/2024.
//

#include <include/gtest/gtest.h>
#include "synnax/synnax.h"
#include "ni_reader.h"
#include <stdio.h>

/// @brief it should read data from a daq and correctly construct a synnax frame from the dataI

TEST(NiReaderTests, testReadandInit){

    //create niDaqReader
    auto reader = ni::niDaqReader();
    // create a channel config vector
    std::vector<ni::ChannelConfig> channel_configs;
    // add a channel config instance
    channel_configs.push_back(ni::ChannelConfig("Dev1/ai0", 65537,  ANALOG_VOLTAGE_IN, 10.0, -10.0);
    // call init
    reader.init(channel_configs, 1000, 20);
    // call start
    reader.start();
    // call read
    auto [frame, err] = reader.read();
    //print frame size
    std::cout << "Frame size: " <<  frame.size() << std::endl;
    // end task
    reader.stop();
}