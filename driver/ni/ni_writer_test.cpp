// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//
// Created by Synnax on 2/28/2024.
//

#include <include/gtest/gtest.h>
#include "synnax/synnax.h"
#include "ni_reader.h"
#include <stdio.h>
#include "driver/testutil/testutil.h"


TEST(NiWriterTests, testDigitalWriteLine){
    std::cout << "Test Init and Digital Write: " << std::endl;

    TaskHandle taskHandle;
    DAQmxCreateTask("", &taskHandle);
    auto writer = ni::niDaqWriter(taskHandle);

    // create a json for config
    auto config = json{
            {"acq_rate", 300},
            {"stream_rate", 30},
            {"hardware", "Dev1"}
            };

    // add a digital channel to the config
    uint32_t cmd_key = 65531;
    uint32_t ack_key = 65532;
    uint32_t port = 0;
    uint32_t line = 0;
    uint32_t ack_index_key = 65533;
    uint32_t cmd_index_key = 65534;

    // add channel to the config
    add_DO_channel_JSON(config, "test_digital_out", cmd_key, ack_key, port, line);

    // init the writer
    writer.init(config,ack_index_key);
    writer.start();


    /// create a synnax frame with a command to write a digital line high
    auto now = (synnax::TimeStamp::now()).value;
    auto frame = synnax::Frame(2);

    frame.add(cmd_index_key, synnax::Series(std::vector<uint64_t>{now}));
    frame.add(cmd_key, synnax::Series(std::vector<uint8_t>{1}));

    // write the frame out to the daq hardware
    auto [f, err] = writer.writeDigital(std::move(frame));

    // check if acknowledgement is correct
    std::cout << "Check Acknowledgement" << std::endl;
    auto ack = f.series->at(1).uint8();
    ASSERT_TRUE( ack[0] == 1);


    /// create a synnax frame with a command to write a digital line low
    now = (synnax::TimeStamp::now()).value;
    frame = synnax::Frame(2);

    frame.add(cmd_index_key, synnax::Series(std::vector<uint64_t>{now}));
    frame.add(cmd_key, synnax::Series(std::vector<uint8_t>{0}));
    auto [f1, err1] = writer.writeDigital(std::move(frame));

    // check if acknowledgement is correct
    std::cout << "Check Acknowledgement" << std::endl;
    auto ack1 = f1.series->at(1).uint8();
    ASSERT_TRUE( ack1[0] == 1);
}

TEST(NiWriterTests, testDigitalWriteMultipleLinesOnePort){

}
//TEST(NiWriterTests, testDigitalWriteMultipleLinesMultiplePorts)

