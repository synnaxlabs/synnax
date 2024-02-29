//
// Created by Synnax on 2/28/2024.
//

#include <include/gtest/gtest.h>
#include "synnax/synnax.h"
#include "ni_reader.h"
#include <stdio.h>
#include "synnax/testutil/testutil.h"


TEST(NiWriterTests, testDigitalWriteLine){
    std::cout << "Test Init and Digital Write: " << std::endl;

    TaskHandle taskHandle;
    DAQmxCreateTask("", &taskHandle);
    auto writer = ni::niDaqWriter(taskHandle);

    // create a json for config
    auto config = json{
            {"acq_rate", 300},
            {"stream_rate", 30},
            {"device", "Dev1"}
            }
    };

    // add a digital channel to the config
    uint32_t cmd_key = 65531;
    uint32_t ack_key = 65532;
    uint32_t port = 0;
    uint32_t line = 0;
    uint32_t ack_index_key = 65533;
    uint32_t cmd_index_key = 65534;

    add_DO_channel_JSON(&config, "test_digital_out", cmd_key, ack_key, port, line);

    // init the writer
    writer.init(config,ack_index_key);
    writer.start();

    // create a synnax frame with a command
    // get the current time
    auto now = (synnax::TimeStamp::now()).value;
    auto frame = synnax::Frame(2);
    frame.add(cmd_index_key, synnax::Series(std::vector<uint32_t>{now}), synnax::TIMESTAMP);
    frame.add(cmd_key, synnax::Series(std::vector<uint32_t>{1}), synnax::UINT32);

    // write the frame
    auto [f, err] = writer.writeDigital(frame);

    // check if acknowledgement is correct

    auto ack = f.series->at(0).uint32();

    ASSERT_TRUE( ack[0] == 1);




    // construct another frame to write a 0
    auto now = (synnax::TimeStamp::now()).value;
    auto frame = synnax::Frame(2);
    frame.add(cmd_index_key, synnax::Series(std::vector<uint32_t>{now}), synnax::TIMESTAMP);
    frame.add(cmd_key, synnax::Series(std::vector<uint32_t>{1}), synnax::UINT32);
}

//TEST(NiWriterTests, testDigitalWriteMultipleLinesOnePort){
//
//}

//TEST(NiWriterTests, testDigitalWriteMultipleLinesMultiplePorts){
//
//}

