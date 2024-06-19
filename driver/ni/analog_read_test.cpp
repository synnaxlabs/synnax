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

#include "client/cpp/synnax.h"
#include "driver/ni/ni.h"
#include <stdio.h>
#include "nlohmann/json.hpp"
#include "driver/testutil/testutil.h"
#include <map>

using json = nlohmann::json;

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//                                          Basic Tests                                                         //
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
TEST(read_tests, one_analog_channel){
    LOG(INFO) << "test_read_one_analog_channel: "<< std::endl;

    // Create NI readerconfig
    auto config = json{
            {"sample_rate", 100}, 
            {"stream_rate", 20}, 
            {"device_location", "Dev1"},
            {"type", "ni_analog_read"},
            {"test", true},    
            {"device", ""}
    };
    json scale_config = json{
        {"type", "none"}
    };

    // create synnax client
    auto client_config = synnax::Config{
            "localhost",
            9090,
            "synnax",
            "seldon"};
    auto client = std::make_shared<synnax::Synnax>(client_config);
    
    // create all the necessary channels in the synnax client
    auto [time, tErr] = client->channels.create( // index channel for analog input channels
            "idx",
            synnax::TIMESTAMP,
            0,
            true
    );
    ASSERT_FALSE(tErr) << tErr.message();
    auto [data, dErr] = client->channels.create( // analog input channel
            "ai",
            synnax::FLOAT32,
            time.key,
            false
    );
    ASSERT_FALSE(dErr) << dErr.message();

    add_AI_channel_JSON(config, "a1", data.key, 0, -10.0, 10.0, "Default", scale_config);

  
    // create synnax task
    auto task = synnax::Task(
            "my_task",          // task name
            "ni_analog_read",   // task type
            to_string(config)   // task config
    );

    auto mockCtx = std::make_shared<task::MockContext>(client);

    std::this_thread::sleep_for(std::chrono::milliseconds(300));
    // Now construct NI reader
    TaskHandle taskHandle;  
    ni::NiDAQmxInterface::CreateTask("",&taskHandle);

    auto reader = ni::AnalogReadSource( taskHandle, 
                                        mockCtx, 
                                        task); // analog reader


    auto b = breaker::Breaker(breaker::Config{"my-breaker", 1*SECOND, 1, 1});
//     b.start();

    if(reader.init() != 0) std::cout << "Failed to initialize reader" << std::endl;
    reader.start();
    std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
    auto [frame, err] = reader.read(b);
    std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;

     //iterate through each series and print the data
    uint32_t ai_count = 0;
    for(int i = 0; i < frame.series->size(); i++){
        std::cout << "\n\n Series " << i << ": \n";
        std::cout << frame.series->at(i);
    }

    std::cout << std::endl;
    reader.stop();
}

// TEST(read_tests, multiple_analog_channels){
//     LOG(INFO) << "test_read_one_analog_channel: "<< std::endl;

//     // Create NI readerconfig
//     auto config = json{
//             {"sample_rate", 100}, 
//             {"stream_rate", 20}, 
//             {"device_location", "Dev1"},
//             {"type", "ni_analog_read"}, //TODO: change to analog_read
//             {"test", true},    
//             {"device", ""}
//     };
//     json scale_config = json{
//         {"type", "none"}
//     };

//     /////////////////////////////////////////////// setup synnax test infrustructure
//     // create synnax client
//     auto client_config = synnax::Config{
//             "localhost",
//             9090,
//             "synnax",
//             "seldon"};
//     auto client = std::make_shared<synnax::Synnax>(client_config);
    
//     // create all the necessary channels in the synnax client
//     auto [time, tErr] = client->channels.create( // index channel for analog input channels
//             "idx",
//             synnax::TIMESTAMP,
//             0,
//             true
//     );
//     ASSERT_FALSE(tErr) << tErr.message();
//     auto [data, dErr] = client->channels.create( // analog input channel
//             "ai",
//             synnax::FLOAT32,
//             time.key,
//             false
//     );
//     ASSERT_FALSE(dErr) << dErr.message();
//     auto [data1, dErr2] = client->channels.create( // analog input channel
//             "ai2",
//             synnax::FLOAT32,
//             time.key,
//             false
//     );
//     ASSERT_FALSE(dErr2) << dErr.message();
//     auto [data2, dErr3] = client->channels.create( // analog input channel
//             "ai3",
//             synnax::FLOAT32,
//             time.key,
//             false
//     );
//     ASSERT_FALSE(dErr3) << dErr.message();
//     auto [data3, dErr4] = client->channels.create( // analog input channel
//             "ai4",
//             synnax::FLOAT32,
//             time.key,
//             false
//     );
//     ASSERT_FALSE(dErr4) << dErr.message();
    

//     add_AI_channel_JSON(config, "a1", data.key, 0, -10.0, 10.0, "Default", scale_config);
//     add_AI_channel_JSON(config, "a2", data1.key, 1, -10.0, 10.0, "Default", scale_config);
//     add_AI_channel_JSON(config, "a3", data2.key, 2, -10.0, 10.0, "Default", scale_config);
//     add_AI_channel_JSON(config, "a4", data3.key, 3, -10.0, 10.0, "Default", scale_config);

  
//     // create synnax task
//     auto task = synnax::Task(
//             "my_task",          // task name
//             "ni_analog_read",   // task type
//             to_string(config)   // task config
//     );

//     auto mockCtx = std::make_shared<task::MockContext>(client);

//     std::this_thread::sleep_for(std::chrono::milliseconds(300));
//     // Now construct NI reader
//     TaskHandle taskHandle;  
//     ni::NiDAQmxInterface::CreateTask("",&taskHandle);

//     auto reader = ni::AnalogReadSource( taskHandle, 
//                                         mockCtx, 
//                                         task); // analog reader
//     if(reader.init() != 0) std::cout << "Failed to initialize reader" << std::endl;
//     reader.start();

//     auto [frame, err] = reader.read();

//      //iterate through each series and print the data
//     for(int i = 0; i < 30; i++ ) { // test for 50 read cycles
//         std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
//         auto [frame, err] = reader.read();
//         std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;
//         //iterate through each series and print the data
//         for(int i = 0; i < frame.series->size(); i++){
//             std::cout << "\n\n Series " << i << ": \n";
//             // check series type before casting
//             if (frame.series->at(i).data_type == synnax::FLOAT32){
//                 auto s =  frame.series->at(i).float32();
//                 for (int j = 0; j < s.size(); j++){
//                     std::cout << s[j] << ", ";
//                     ASSERT_NEAR(s[j], 0, 10); // can be any value of a sine wave from -10 to 10
//                 }
//             }
//             else if(frame.series->at(i).data_type == synnax::TIMESTAMP){
//                 auto s =  frame.series->at(i).uint64();
//                 for (int j = 0; j < s.size(); j++){
//                     std::cout << s[j] << ", ";
//                 }
//             }
//         }
//         std::cout << std::endl;
//     }
//     reader.stop();
// }

// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// //                                             Scaling Tests                                                   //
// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// TEST(read_tests, analog_linear_scaling){
//     LOG(INFO) << "analog_linear_scaling: "<< std::endl;

//     // Create NI readerconfig
//     auto config = json{
//             {"sample_rate", 100}, 
//             {"stream_rate", 20}, 
//             {"device_location", "Dev1"},
//             {"type", "ni_analog_read"},
//             {"test", true},    
//             {"device", ""}
//     };
//     json scale_config = json{
//         {"type", "linear"},
//         {"pre_scaled_units", "Volts"},
//         {"scaled_units", "Volts"},
//         {"slope", 0.5},
//         {"y_intercept", 5}
//     };

//     // create synnax client
//     auto client_config = synnax::Config{
//             "localhost",
//             9090,
//             "synnax",
//             "seldon"};
//     auto client = std::make_shared<synnax::Synnax>(client_config);
    
//     // create all the necessary channels in the synnax client
//     auto [time, tErr] = client->channels.create( // index channel for analog input channels
//             "idx",
//             synnax::TIMESTAMP,
//             0,
//             true
//     );
//     ASSERT_FALSE(tErr) << tErr.message();
//     auto [data, dErr] = client->channels.create( // analog input channel
//             "ai",
//             synnax::FLOAT32,
//             time.key,
//             false
//     );
//     ASSERT_FALSE(dErr) << dErr.message();

//     add_AI_channel_JSON(config, "a1", data.key, 0, 0, 10.0, "Default", scale_config);

  
//     // create synnax task
//     auto task = synnax::Task(
//             "my_task",          // task name
//             "ni_analog_read",   // task type
//             to_string(config)   // task config
//     );

//     auto mockCtx = std::make_shared<task::MockContext>(client);

//     std::this_thread::sleep_for(std::chrono::milliseconds(300));
//     // Now construct NI reader
//     TaskHandle taskHandle;  
//     ni::NiDAQmxInterface::CreateTask("",&taskHandle);

//     auto reader = ni::AnalogReadSource( taskHandle, 
//                                         mockCtx, 
//                                         task); // analog reader
//     if(reader.init() != 0) std::cout << "Failed to initialize reader" << std::endl;
//     reader.start();
//     std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
//     auto [frame, err] = reader.read();
//     std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;

//      //iterate through each series and print the data
//     for(int i = 0; i < frame.series->size(); i++){
//         std::cout << "\n\n Series " << i << ": \n";
//         // check series type before casting
//         if (frame.series->at(i).data_type == synnax::FLOAT32){
//             auto s =  frame.series->at(i).float32();
//             for (int j = 0; j < s.size(); j++){
//                 std::cout << s[j] << ", ";
//                 ASSERT_NEAR(s[j], 5, 1);
//             }
//         }
//         else if(frame.series->at(i).data_type == synnax::TIMESTAMP){
//             auto s =  frame.series->at(i).uint64();
//             for (int j = 0; j < s.size(); j++){
//                 std::cout << s[j] << ", ";
//             }
//         }
//     }
    
//     std::cout << std::endl;
//     reader.stop();
// }

// TEST(read_tests, analog_map_scaling){
//     LOG(INFO) << "analog_map_scaling: "<< std::endl;

//     // Create NI readerconfig
//     auto config = json{
//             {"sample_rate", 100}, 
//             {"stream_rate", 20}, 
//             {"device_location", "Dev1"},
//             {"type", "ni_analog_read"},
//             {"test", true},    
//             {"device", ""}
//     };
//     json scale_config = json{
//         {"type", "map"},
//         {"pre_scaled_units", "Volts"},
//         {"scaled_units", "Volts"},
//         {"pre_scaled_min", 0.0},
//         {"pre_scaled_max", 10.0},
//         {"scaled_min", 0},
//         {"scaled_max", 100.0}
//     };

//     // create synnax client
//     auto client_config = synnax::Config{
//             "localhost",
//             9090,
//             "synnax",
//             "seldon"};
//     auto client = std::make_shared<synnax::Synnax>(client_config);
    
//     // create all the necessary channels in the synnax client
//     auto [time, tErr] = client->channels.create( // index channel for analog input channels
//             "idx",
//             synnax::TIMESTAMP,
//             0,
//             true
//     );
//     ASSERT_FALSE(tErr) << tErr.message();
//     auto [data, dErr] = client->channels.create( // analog input channel
//             "ai",
//             synnax::FLOAT32,
//             time.key,
//             false
//     );
//     ASSERT_FALSE(dErr) << dErr.message();

//     add_AI_channel_JSON(config, "a1", data.key, 0, 0, 100, "Default", scale_config);

  
//     // create synnax task
//     auto task = synnax::Task(
//             "my_task",          // task name
//             "ni_analog_read",   // task type
//             to_string(config)   // task config
//     );

//     auto mockCtx = std::make_shared<task::MockContext>(client);

//     std::this_thread::sleep_for(std::chrono::milliseconds(300));
//     // Now construct NI reader
//     TaskHandle taskHandle;  
//     ni::NiDAQmxInterface::CreateTask("",&taskHandle);

//     auto reader = ni::AnalogReadSource( taskHandle, 
//                                         mockCtx, 
//                                         task); // analog reader
//     if(reader.init() != 0) std::cout << "Failed to initialize reader" << std::endl;
//     reader.start();
//     std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
//     auto [frame, err] = reader.read();
//     std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;

//      //iterate through each series and print the data
//     for(int i = 0; i < frame.series->size(); i++){
//         std::cout << "\n\n Series " << i << ": \n";
//         // check series type before casting
//         if (frame.series->at(i).data_type == synnax::FLOAT32){
//             auto s =  frame.series->at(i).float32();
//             for (int j = 0; j < s.size(); j++){
//                 std::cout << s[j] << ", ";
//                 ASSERT_NEAR(s[j], 50, 5);
//             }
//         }
//         else if(frame.series->at(i).data_type == synnax::TIMESTAMP){
//             auto s =  frame.series->at(i).uint64();
//             for (int j = 0; j < s.size(); j++){
//                 std::cout << s[j] << ", ";
//             }
//         }
//     }

//     std::cout << std::endl;
//     reader.stop();
// }


// TEST(read_tests, analog_table_scaling){
//     LOG(INFO) << "analog_table_scaling: "<< std::endl;

//     // Create NI readerconfig
//     auto config = json{
//             {"sample_rate", 100}, 
//             {"stream_rate", 20}, 
//             {"device_location", "Dev1"},
//             {"type", "ni_analog_read"},
//             {"test", true},    
//             {"device", ""}
//     };
//     json scale_config = json{
//         {"type", "table"},
//         {"pre_scaled_units", "Volts"},
//         {"scaled_units", "Volts"},
//         {"num_points", 11},
//         {"pre_scaled_vals", {0.0, 1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0}},
//         {"scaled_vals", {0.0, 50.0, 100.0, 150.0, 200.0, 250.0, 300.0, 350.0, 400.0, 450.0, 500.0}}
//     };

//     // create synnax client
//     auto client_config = synnax::Config{
//             "localhost",
//             9090,
//             "synnax",
//             "seldon"};
//     auto client = std::make_shared<synnax::Synnax>(client_config);
    
//     // create all the necessary channels in the synnax client
//     auto [time, tErr] = client->channels.create( // index channel for analog input channels
//             "idx",
//             synnax::TIMESTAMP,
//             0,
//             true
//     );
//     ASSERT_FALSE(tErr) << tErr.message();
//     auto [data, dErr] = client->channels.create( // analog input channel
//             "ai",
//             synnax::FLOAT32,
//             time.key,
//             false
//     );
//     ASSERT_FALSE(dErr) << dErr.message();

//     add_AI_channel_JSON(config, "a1", data.key, 0, 0, 500.0, "Default", scale_config);

  
//     // create synnax task
//     auto task = synnax::Task(
//             "my_task",          // task name
//             "ni_analog_read",   // task type
//             to_string(config)   // task config
//     );

//     auto mockCtx = std::make_shared<task::MockContext>(client);

//     std::this_thread::sleep_for(std::chrono::milliseconds(300));
//     // Now construct NI reader
//     TaskHandle taskHandle;  
//     ni::NiDAQmxInterface::CreateTask("",&taskHandle);

//     auto reader = ni::AnalogReadSource( taskHandle, 
//                                         mockCtx, 
//                                         task); // analog reader
//     if(reader.init() != 0) std::cout << "Failed to initialize reader" << std::endl;
//     reader.start();
//     std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
//     auto [frame, err] = reader.read();
//     std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;

//      //iterate through each series and print the data
//     for(int i = 0; i < frame.series->size(); i++){
//         std::cout << "\n\n Series " << i << ": \n";
//         // check series type before casting
//         if (frame.series->at(i).data_type == synnax::FLOAT32){
//             auto s =  frame.series->at(i).float32();
//             for (int j = 0; j < s.size(); j++){
//                 std::cout << s[j] << ", ";
//                 ASSERT_NEAR(s[j], 250, 20);
//             }
//         }
//         else if(frame.series->at(i).data_type == synnax::TIMESTAMP){
//             auto s =  frame.series->at(i).uint64();
//             for (int j = 0; j < s.size(); j++){
//                 std::cout << s[j] << ", ";
//             }
//         }
//     }

//     std::cout << std::endl;
//     reader.stop();
// }

// TEST(read_tests, analog_polynomial_scaling){
//     LOG(INFO) << "analog_table_scaling: "<< std::endl;

//     // Create NI readerconfig
//     auto config = json{
//             {"sample_rate", 100}, 
//             {"stream_rate", 20}, 
//             {"device_location", "Dev1"},
//             {"type", "ni_analog_read"},
//             {"test", true},    
//             {"device", ""}
//     };
//     json scale_config = json{
//         {"type", "polynomial"},
//         {"pre_scaled_units", "Volts"},
//         {"scaled_units", "Volts"},
//         {"poly_order", 2},
//         {"coeffs", {300.0, 300.0, 43.0}},
//         {"num_coeffs", 3},
//         {"min_x", 0.0},
//         {"max_x", 10.0}
//     };

//     // create synnax client
//     auto client_config = synnax::Config{
//             "localhost",
//             9090,
//             "synnax",
//             "seldon"};
//     auto client = std::make_shared<synnax::Synnax>(client_config);
    
//     // create all the necessary channels in the synnax client
//     auto [time, tErr] = client->channels.create( // index channel for analog input channels
//             "idx",
//             synnax::TIMESTAMP,
//             0,
//             true
//     );
//     ASSERT_FALSE(tErr) << tErr.message();
//     auto [data, dErr] = client->channels.create( // analog input channel
//             "ai",
//             synnax::FLOAT32,
//             time.key,
//             false
//     );
//     ASSERT_FALSE(dErr) << dErr.message();

//     add_AI_channel_JSON(config, "a1", data.key, 0, 0, 10.0, "Default", scale_config);

  
//     // create synnax task
//     auto task = synnax::Task(
//             "my_task",          // task name
//             "ni_analog_read",   // task type
//             to_string(config)   // task config
//     );

//     auto mockCtx = std::make_shared<task::MockContext>(client);

//     std::this_thread::sleep_for(std::chrono::milliseconds(300));
//     // Now construct NI reader
//     TaskHandle taskHandle;  
//     ni::NiDAQmxInterface::CreateTask("",&taskHandle);

//     auto reader = ni::AnalogReadSource( taskHandle, 
//                                         mockCtx, 
//                                         task); // analog reader
//     if(reader.init() != 0) std::cout << "Failed to initialize reader" << std::endl;
//     reader.start();
//     std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
//     auto [frame, err] = reader.read();
//     std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;

//      //iterate through each series and print the data
//     for(int i = 0; i < frame.series->size(); i++){
//         std::cout << "\n\n Series " << i << ": \n";
//         // check series type before casting
//         if (frame.series->at(i).data_type == synnax::FLOAT32){
//             auto s =  frame.series->at(i).float32();
//             for (int j = 0; j < s.size(); j++){
//                 std::cout << s[j] << ", ";
//                 ASSERT_NEAR(s[j], 117, 2);
//             }
//         }
//         else if(frame.series->at(i).data_type == synnax::TIMESTAMP){
//             auto s =  frame.series->at(i).uint64();
//             for (int j = 0; j < s.size(); j++){
//                 std::cout << s[j] << ", ";
//             }
//         }
//     }

//     std::cout << std::endl;
//     reader.stop();
// }


//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//                                             Channnel Tests                                                   //
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
/// @brief Voltage
TEST(read_tests, one_analog_voltage_channel){
    LOG(INFO) << "test_read_one_analog_channel: "<< std::endl;

    // Create NI readerconfig
    auto config = json{
            {"sample_rate", 100}, 
            {"stream_rate", 20}, 
            {"device_location", "Dev1"},
            {"type", "ni_analog_read"},
            {"test", true},    
            {"device", ""}
    };
    json scale_config = json{
        {"type", "none"}
    };

    // create synnax client
    auto client_config = synnax::Config{
            "localhost",
            9090,
            "synnax",
            "seldon"};
    auto client = std::make_shared<synnax::Synnax>(client_config);
    
    // create all the necessary channels in the synnax client
    auto [time, tErr] = client->channels.create( // index channel for analog input channels
            "idx",
            synnax::TIMESTAMP,
            0,
            true
    );
    ASSERT_FALSE(tErr) << tErr.message();
    auto [data, dErr] = client->channels.create( // analog input channel
            "ai",
            synnax::FLOAT32,
            time.key,
            false
    );
    ASSERT_FALSE(dErr) << dErr.message();

    add_AI_channel_JSON(config, "a1", data.key, 0, -10.0, 10.0, "Default", scale_config);

  
    // create synnax task
    auto task = synnax::Task(
            "my_task",          // task name
            "ni_analog_read",   // task type
            to_string(config)   // task config
    );

    auto mockCtx = std::make_shared<task::MockContext>(client);

    std::this_thread::sleep_for(std::chrono::milliseconds(300));
    // Now construct NI reader
    TaskHandle taskHandle;  
    ni::NiDAQmxInterface::CreateTask("",&taskHandle);

    auto reader = ni::AnalogReadSource( taskHandle, 
                                        mockCtx, 
                                        task); // analog reader


    auto b = breaker::Breaker(breaker::Config{"my-breaker", 1*SECOND, 1, 1});
//     b.start();

    if(reader.init() != 0) std::cout << "Failed to initialize reader" << std::endl;
    reader.start();
    std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
    auto [frame, err] = reader.read(b);
    std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;

     //iterate through each series and print the data
    uint32_t ai_count = 0;
    for(int i = 0; i < frame.series->size(); i++){
        std::cout << "\n\n Series " << i << ": \n";
        std::cout << frame.series->at(i);
    }

    std::cout << std::endl;
    reader.stop();
}

///@brief RMS Voltage 

///@brief voltage with excitation

///@brief Thermocouple 
TEST(read_tests, one_analog_thermocouple_channel){
    LOG(INFO) << "one_analog_thermocouple_channel: "<< std::endl;

    // create synnax client
    auto client_config = synnax::Config{
            "localhost",
            9090,
            "synnax",
            "seldon"};
    auto client = std::make_shared<synnax::Synnax>(client_config);
    
    // create all the necessary channels in the synnax client
    auto [time, tErr] = client->channels.create( // index channel for analog input channels
            "idx",
            synnax::TIMESTAMP,
            0,
            true
    );
    ASSERT_FALSE(tErr) << tErr.message();
    auto [data, dErr] = client->channels.create( // analog input channel
            "thermocouple",
            synnax::FLOAT32,
            time.key,
            false
    );
    ASSERT_FALSE(dErr) << dErr.message();

    // Create NI readerconfig json
    auto config = json{
            {"sample_rate", 100}, 
            {"stream_rate", 20}, 
            {"device_location", "Dev1"},
            {"type", "ni_analog_read"},
            {"test", true},    
            {"device", ""}
    };
   
    auto channel_config = json{
        {"name", "test_ni_channel"}
        {"type", "thermocouple"},
        {"max_val", 100.0},
        {"min_val", 0.0},
        {"units", "C"},
        {"enabled", true},
        {"channel",data.key},
        {"key", "key"},
        {"thermocouple_type", "J"},
        {"reference_junction_type", "internal"},
        {"reference_junction_temp", 25.0}
        {"custom_scale", {"type","none"}} 
    };

    config[channels] = json::array();
    config[channels].push_back(channel_config);
  
    // create synnax task
    auto task = synnax::Task(
            "my_task",          // task name
            "ni_analog_read",   // task type
            to_string(config)   // task config
    );

    auto mockCtx = std::make_shared<task::MockContext>(client);

    std::this_thread::sleep_for(std::chrono::milliseconds(300));
    // Now construct NI reader
    TaskHandle taskHandle;  
    ni::NiDAQmxInterface::CreateTask("",&taskHandle);

    auto reader = ni::AnalogReadSource( taskHandle, 
                                        mockCtx, 
                                        task); // analog reader


    auto b = breaker::Breaker(breaker::Config{"my-breaker", 1*SECOND, 1, 1});
//     b.start();

    if(reader.init() != 0) std::cout << "Failed to initialize reader" << std::endl;
    reader.start();
    std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
    auto [frame, err] = reader.read(b);
    std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;

     //iterate through each series and print the data
    uint32_t ai_count = 0;
    for(int i = 0; i < frame.series->size(); i++){
        std::cout << "\n\n Series " << i << ": \n";
        std::cout << frame.series->at(i);
    }

    std::cout << std::endl;
    reader.stop();
}


///@brief Temperature Built in sensor

///@brief ThermistorIEX

///@brief ThermistorVEX

///@brief Accleration

///@brief Acceleration 4 wire dc voltage

///@brief Acceleration Charge

///@brief Force Bridge Polynomial

///@brief Force Bridge Table

///@brief Force Bridge Linear

///@brief pressre bridge polynomial

///@brief pressure bridge table

///@brief pressure bridge linear

///@brief torque bridge polynomial

///@brief torque bridge table

///@brief torque bridge linear

///@brief charge

///@brief Current

///@brief force brdige polynomial

///@brief force bridge table

///@brief force bridge linear

///@brief force IEPE

///@brief frequency


///@brief Microphone

///@brief pressure


///@brief pressure bridge table


///@brief pressure bridge linear


///@brief resistance


///@brief rosette strain gauge

///@brief strain gauge


///@brief RTD


///@brief TorqueBr

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//                                             Error Handling                                                   //
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// TODO:
// driver errors 
    // - invalid channel
        // - invalid scale (analog)
            // linear
                // - missing slope or offset or both
                // - invalid slope & offset or both (i.e. isnt comptaible with specified max and min)
            // map
                // - missing attributes
                // - prescaled_min >= prescaled_max || scaled_min >= scaled_max
                // - invalid prescaled_min, prescaled_max, scaled_min, scaled_max
            // polynomial
            // table
        // - min >= max val (analog)
        // - invalid terminal config (analog)
        // - invalid port (digital and analog)
        // - invalid line (analog)
    // - invalid device
    // - stream rate > sample rate

// vendor/hardware errors
// double start
// double stop
// read function failed state
// write function failed