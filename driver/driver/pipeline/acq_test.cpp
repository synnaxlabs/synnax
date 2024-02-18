//
// Created by Synnax on 2/4/2024.
//

/// stdd
#include <stdio.h>
#include <thread>

/// GTest
#include <include/gtest/gtest.h>

/// Internal
#include "synnax/synnax.h"
#include "synnax/testutil/testutil.h"
#include "acq.h"

/// @brief it should use niReader and perform a acuisition workflow which
/// includes init, start, stop, and read functions and commits a frame to synnax

TEST(AcqTests, testAcqNi){
    //TODO add asserts (elham)

    std::cout << "Test Acq: " << std::endl;

    // create task
    TaskHandle taskHandle;
    DAQmxCreateTask("",&taskHandle);



    auto client_config = synnax::Config{
            "localhost",
            9090,
            "synnax",
            "seldon"};

    auto client = std::make_shared<synnax::Synnax>(client_config);


    auto [time, tErr] = client->channels.create(
            "time",
            synnax::TIMESTAMP,
            0,
            true
    );

    ASSERT_FALSE(tErr) << tErr.message();

    auto [data, dErr] = client->channels.create(
            "data",
            synnax::FLOAT32,
            time.key,
            false
    );
    ASSERT_FALSE(dErr) << dErr.message();

    std::vector<ni::channel_config> channel_configs;
    channel_configs.push_back(ni::channel_config({"", time.key, ni::INDEX_CHANNEL , 0, 0}));
    channel_configs.push_back(ni::channel_config({"Dev1/ai0", data.key, ni::ANALOG_VOLTAGE_IN , -10.0, 10.0}));

    // make and init daqReade unique ptrr
    auto reader = std::make_unique<ni::niDaqReader>(taskHandle);
    reader->init(channel_configs, 200, 20);

    // create a test writer
    auto now = synnax::TimeStamp::now();
    auto writerConfig = synnax::WriterConfig{
            std::vector<synnax::ChannelKey>{time.key, data.key},
            now,
            std::vector<synnax::Authority>{synnax::ABSOLUTTE, synnax::ABSOLUTTE},
            synnax::Subject{"test_writer"},
    };

    // instantiate the acq
    auto acq = Acq::Acq(writerConfig, client, std::move(reader));
    printf("Initialized acq");
//    std::cout << "Starting acq" << std::endl;
    acq.start();
    std::this_thread::sleep_for(std::chrono::seconds(200)); // let the acq run for 10 seconds, should expect frames to be commited
//    std::cout << "Stopping acq" << std::endl;
    acq.stop();

}