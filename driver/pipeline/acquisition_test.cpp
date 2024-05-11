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
#include "client/cpp/synnax.h"
#include "driver/ni/ni.h"
#include "Acquisition.h"
#include "driver/testutil/testutil.h"
#include "nlohmann/json.hpp"
#include "driver/breaker/breaker.h"

using json = nlohmann::json;

/// @brief it should use niReader and perform a acuisition workflow which
/// includes init, start, stop, and read functions and commits a frame to synnax
TEST(AcquisitionPipelineTests, test_acquisition_NI_analog_reader){
        LOG(INFO) << "Test Acquisition Pipeline with NI Analog Read:" << std::endl;

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
        ni::NiDAQmxInterface::CreateTask("",&taskHandle);

        auto reader = std::make_shared<ni::daqReader>(taskHandle, mockCtx, task);

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
        auto acquisition_pipe = pipeline::Acquisition(mockCtx, writerConfig, reader, breaker_config); 

        // create a streamer to read the frames that the pipe writes to the server
        auto streamer_config = synnax::StreamerConfig{
                .channels = std::vector<synnax::ChannelKey>{time.key, data.key},
                .start = TimeStamp::now(),
        };

        auto [streamer, sErr] = mockCtx->client->telem.openStreamer(streamer_config);


        
        acquisition_pipe.start();

        for(int i = 0; i < 100; i++){
                auto [frame, err] = streamer.read();
                std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;

                uint32_t ai_count = 0;
                for(int i = 0; i < frame.series->size(); i++){
                        std::cout << "\n\n Series " << i << ": \n";
                        // check series type before casting
                        if (frame.series->at(i).data_type == synnax::FLOAT32){
                                auto s =  frame.series->at(i).float32();
                                for (int j = 0; j < s.size(); j++){
                                        std::cout << s[j] << ", ";
                                        ASSERT_NEAR(s[j], 0, 10); // can be any value of a sign wave from -10 to 10
                                }
                                ai_count++;
                        }
                        else if(frame.series->at(i).data_type == synnax::TIMESTAMP){
                                auto s =  frame.series->at(i).uint64();
                                for (int j = 0; j < s.size(); j++){
                                        std::cout << s[j] << ", ";
                                        ASSERT_TRUE((s[j] <= final_timestamp));
                                }
                        }
                }
                std::cout << std::endl;
        }
        // std::this_thread::sleep_for(std::chrono::seconds(10));
        acquisition_pipe.stop();
}



/// @brief it should use niReader and perform a acuisition workflow which
/// includes init, start, stop, and read functions and commits a frame to synnax
TEST(AcquisitionPipelineTests, test_acquisition_NI_digital_reader){
        LOG(INFO) << "Test Acq Digital Read:" << std::endl;


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
                synnax::UINT8,
                time.key,
                false
        );
        ASSERT_FALSE(dErr) << dErr.message();

        // create reader config json
        auto config = json{
            {"acq_rate", 2000}, // dont actually need these here
            {"stream_rate", 20}, // same as above
            {"device_name", "PXI1Slot2_2"},
            {"reader_type", "digitalReader"}
        };
        add_index_channel_JSON(config, "time", time.key);
        add_DI_channel_JSON(config, "acq_data", data.key, 0, 0);

        // create synnax task
        auto task = synnax::Task(
                "my_task",
                "NI_digitalReader",
                to_string(config)
        );

        auto mockCtx = std::make_shared<task::MockContext>(client);
        std::this_thread::sleep_for(std::chrono::milliseconds(10));

        // now create a daqReader
        TaskHandle taskHandle;
        ni::NiDAQmxInterface::CreateTask("",&taskHandle);

        auto reader = std::make_shared<ni::daqReader>(taskHandle, mockCtx, task);

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
        auto acquisition_pipe = pipeline::Acquisition(mockCtx, writerConfig, reader, breaker_config); 

        // create a streamer to read the frames that the pipe writes to the server
        auto streamer_config = synnax::StreamerConfig{
                .channels = std::vector<synnax::ChannelKey>{time.key, data.key},
                .start = TimeStamp::now(),
        };

        auto [streamer, sErr] = mockCtx->client->telem.openStreamer(streamer_config);


        
        acquisition_pipe.start();

        for(int i = 0; i < 100; i++){
                auto [frame, err] = streamer.read();
                std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;

                uint32_t ai_count = 0;
                for(int i = 0; i < frame.series->size(); i++){
                        std::cout << "\n\n Series " << i << ": \n";
                        // check series type before casting
                        if (frame.series->at(i).data_type == synnax::UINT8){
                                auto s =  frame.series->at(i).uint8();
                                for (int j = 0; j < s.size(); j++){
                                        std::cout << (uint32_t)s[j] << ", ";
                                        ASSERT_TRUE((s[j] == 1) || (s[j] == 0));   
                                }
                                ai_count++;
                        }
                        else if(frame.series->at(i).data_type == synnax::TIMESTAMP){
                                auto s =  frame.series->at(i).uint64();
                                for (int j = 0; j < s.size(); j++){
                                        std::cout << s[j] << ", ";
                                        ASSERT_TRUE((s[j] <= final_timestamp));
                                }
                        }
                }
                std::cout << std::endl;
        }
        // std::this_thread::sleep_for(std::chrono::seconds(10));
        acquisition_pipe.stop();
}

