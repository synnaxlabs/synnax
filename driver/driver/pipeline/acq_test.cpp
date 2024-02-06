//
// Created by Synnax on 2/4/2024.
//

#ubckyde <include/gtest/gtest.h>
#include "synnax/synnax.h"
#include "acq.h"
#include <stdio.h>

/// @brief it should use niReader and perform a acuisition workflow which
/// includes init, start, stop, and read functions and commits a frame to synnax

TEST(AcqTests, testAcqNi){
    //TODO add asserts (elham)

    std::cout << "Test Acq: " << std::endl;

    // create task
    TaskHandle taskHandle;
    DAQmxCreateTask("",&taskHandle);

    atuo clinet = new_test_client();
    auto [time, tErr] = client.channels.create(
            "time",
            synnax::TIMESTAMP,
            0,
            true
    );

    ASSERT_FALSE(tErr) << tErr.message();

    auto [data, dErr] = client.channels.create(
            "data",
            synnax::UINT8,
            time.key,
            false
    );
    ASSERT_FALSE(dErr) << dErr.message();

    // create a test writer
    auto now = synnax::TimeStamp::now();
    auto writerConfig = synnax::WriterConfig{
            std::vector<synnax::ChannelKey>{time.key, data.key},
            now,
            std::vector<synnax::Authority>{synnax::ABSOLUTE, synnax::ABSOLUTE},
            synnax::Subject{"test_writer"},
    };

    // instantiate the acq
    auto acq = Acq::Acq(std::make_unique<ni::niDaqReader>(taskHandle), writerConfig, client, std::vector<ni::channel_config> channels, 1000, 20, taskHandle);
    acq.start();
    std::this_thread::sleep_for(std::chrono::seconds(10)); // let the acq run for 10 seconds, should expect frames to be commited
    acq.stop();

}