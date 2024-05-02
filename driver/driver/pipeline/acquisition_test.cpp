// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//
// Created by Synnax on 2/4/2024.
//

/// std
#include <stdio.h>
#include <thread>

/// GTest
#include <include/gtest/gtest.h>

/// Internal
#include "client/cpp/synnax/synnax.h"
#include "driver/driver/ni/ni_reader.h"
#include "Acquisition.h"
#include "driver/driver/testutil/testutil.h"
#include "nlohmann/json.hpp"
#include "driver/driver/breaker/breaker.h"

using json = nlohmann::json;

/// @brief it should use niReader and perform a acuisition workflow which
/// includes init, start, stop, and read functions and commits a frame to synnax
TEST(AcquisitionPipelineTests, test_acquisition_NI_analog_reader){
        LOG(INFO) << "Test Acq Analog Read:" << std::endl;


        // create synnax client
        auto client_config = synnax::Config{
                "localhost",
                9090,
                "synnax",
                "seldon"};
        auto client = std::make_shared<synnax::Synnax>(client_config);
        // create all the necessary channels in the synnax client
        auto [time, tErr] = client->channels.create( // index channel for analog input channels
                "time",
                synnax::TIMESTAMP,
                0,
                true
        );
        ASSERT_FALSE(tErr) << tErr.message();

        auto [data, dErr] = client->channels.create( // analog input channel
                "acq_data",
                synnax::FLOAT32,
                time.key,
                false
        );
        ASSERT_FALSE(dErr) << dErr.message();

        // create reader config json
        auto config = json{
            {"acq_rate", 2000}, // dont actually need these here
            {"stream_rate", 20}, // same as above
            {"device_name", "Dev1"},
            {"reader_type", "analogReader"}
        };
        add_index_channel_JSON(config, "time", time.key);
        add_AI_channel_JSON(config, "acq_data", data.key, 0, -10.0, 10.0);

        // create synnax task
        auto task = synnax::Task(
                "my_task",
                "NI_analogReader",
                to_string(config)
        );

        auto mockCtx = std::make_shared<task::MockContext>(client);
        std::this_thread::sleep_for(std::chrono::milliseconds(10));

        // now create a daqReader
        TaskHandle taskHandle;
        DAQmxCreateTask("",&taskHandle);

        auto reader = std::make_unique<ni::daqReader>(taskHandle, mockCtx, task);

        // now create test writer config
        // auto now = synnax::TimeStamp::now();
        // auto writerConfig = synnax::WriterConfig{
        //         std::vector<synnax::ChannelKey>{time.key, data.key},
        //         now,
        //         std::vector<synnax::Authority>{synnax::ABSOLUTTE, synnax::ABSOLUTTE},
        //         synnax::ControlSubject{"test_writer"},
        // };

        auto writerConfig = synnax::WriterConfig{
                .channels = std::vector<synnax::ChannelKey>{time.key, data.key},
                .start = TimeStamp::now(),
                .mode = synnax::WriterStreamOnly};

        // create breaker config     
        auto breaker_config = breaker::Config{
                .name = task.name,
                .base_interval = 1 * SECOND,
                .max_retries = 20,
                .scale = 1.2,
        };


        // instantiate the acquisition pipe
        auto acquisition_pipe = pipeline::Acquisition(mockCtx, writerConfig, std::move(reader), breaker_config); 
        
        acquisition_pipe.start();
        std::this_thread::sleep_for(std::chrono::seconds(30));
        acquisition_pipe.stop();
}






//////////////////////////////////////////////////////////////////////////////////////////////////////////// OLD


/*

TEST(AcqTests, testAcqNiAnalogReader){
    //TODO add asserts (elham)
    std::cout << "Test Acq Analog Read:" << std::endl;
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
    //print keys
    std::cout << "Time Key: " << time.key << std::endl;
    std::cout << "Data Key: " << data.key << std::endl;
    // make and init daqReade unique ptrr
    auto reader = std::make_unique<ni::niDaqReader>(taskHandle);
    reader->init(channel_configs, 1000, 500);

    // create a test writer
    auto now = synnax::TimeStamp::now();
    auto writerConfig = synnax::WriterConfig{
            std::vector<synnax::ChannelKey>{time.key, data.key},
            now,
            std::vector<synnax::Authority>{synnax::ABSOLUTTE, synnax::ABSOLUTTE},
            synnax::ControlSubject{"test_writer"},
    };

    // instantiate the acq
    auto acq = Acquisition::Acquisition(writerConfig, client, std::move(reader));
    acq.start();
    std::this_thread::sleep_for(std::chrono::seconds(200));
    acq.stop();

}

TEST(AcqTests, testAcqNiDigitalReader){
    std::cout << "Test Acq Digital Reads: " << std::endl;

    //create synnax client config
    auto client_config = synnax::Config{
            "localhost",
            9090,
            "synnax",
            "seldon"};
    auto client = std::make_shared<synnax::Synnax>(client_config);

    // create all the necessary channels in the synnax client
    auto [idx, tErr1] = client->channels.create( // index channel for digital input channels
            "idx",
            synnax::TIMESTAMP,
            0,
            true
    );
    ASSERT_FALSE(tErr1) << tErr1.message();
    auto [d1, dErr] = client->channels.create( // digital input channel
            "d1",
            synnax::UINT8,
            idx.key,
            false
    );
    ASSERT_FALSE(dErr) << dErr.message();

    //create config json
    auto config = json{
            {"acq_rate", 100}, // dont actually need these here
            {"stream_rate", 20}, // same as above
            {"hardware", "PXI1Slot2_2"}
    };
    add_index_channel_JSON(config, "idx", idx.key);
    add_DI_channel_JSON(config, "d1", d1.key, 0, 0);

    //create daqReader
    TaskHandle taskHandle;
    DAQmxCreateTask("",&taskHandle);
    auto reader = std::make_unique<ni::niDaqReader>(taskHandle);
    reader->init(config, config["acq_rate"], config["stream_rate"]);

    // create a test writer
    auto now = synnax::TimeStamp::now();
    auto writerConfig = synnax::WriterConfig{
            std::vector<synnax::ChannelKey>{idx.key, d1.key},
            now,
            std::vector<synnax::Authority>{synnax::ABSOLUTTE, synnax::ABSOLUTTE},
            synnax::ControlSubject{"test_writer"},
    };

    // instantiate the acq
    auto acq = Acquisition::Acquisition(writerConfig, client, std::move(reader));
    acq.start();
    std::this_thread::sleep_for(std::chrono::seconds(200));
    acq.stop();
}

*/