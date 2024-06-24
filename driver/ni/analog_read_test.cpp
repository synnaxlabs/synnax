// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <map>
#include <stdio.h>

#include <include/gtest/gtest.h>
#include "glog/logging.h"
#include "nlohmann/json.hpp"

#include "client/cpp/synnax.h"
#include "driver/ni/ni.h"
#include "driver/testutil/testutil.h"

using json = nlohmann::json;

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//                                                                                                              //
//                                                   Basic Tests                                                //                
//                                                                                                              //
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////
//                      ANALOG BASIC                             //
///////////////////////////////////////////////////////////////////
// TEST(read_tests, one_analog_channel){

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
//         {"type", "none"}
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

//     add_AI_channel_JSON(config, "a1", data.key, 0, -10.0, 10.0, "Default", scale_config);

  
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


//     auto b = breaker::Breaker(breaker::Config{"my-breaker", 1*SECOND, 1, 1});
// //     b.start();

//     if(reader.init() != 0) std::cout << "Failed to initialize reader" << std::endl;
//     reader.start();
//     std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
//     auto [frame, err] = reader.read(b);
//     std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;

//      //iterate through each series and print the data
//     uint32_t ai_count = 0;
//     for(int i = 0; i < frame.series->size(); i++){
//         std::cout << "\n\n Series " << i << ": \n";
//         std::cout << frame.series->at(i);
//     }

//     std::cout << std::endl;
//     reader.stop();
// }

// TEST(read_tests, multiple_analog_channels){

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

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//                                                                                                              //
//                                                   Scaling Tests                                              //                  
//                                                                                                              //
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////
//                          LINEAR                               //
///////////////////////////////////////////////////////////////////

// TEST(read_tests, analog_linear_scaling){

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

///////////////////////////////////////////////////////////////////
//                          MAP                                  //
///////////////////////////////////////////////////////////////////
// TEST(read_tests, analog_map_scaling){

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


///////////////////////////////////////////////////////////////////
//                          TABLE                                //
///////////////////////////////////////////////////////////////////
// TEST(read_tests, analog_table_scaling){

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


///////////////////////////////////////////////////////////////////
//                          POLYNOMIAL                           //
///////////////////////////////////////////////////////////////////
// TEST(read_tests, analog_polynomial_scaling){

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
//                                                                                                              //
//                                                   Channnel Tests                                             //                  
//                                                                                                              //
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////
//                          Helper                               //
///////////////////////////////////////////////////////////////////
void analog_channel_helper(json config, json scale_config, json channel_config){
    LOG(INFO) << "Begin Test: "<< std::endl;
    
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
            "RTD",
            synnax::FLOAT32,
            time.key,
            false
    );
    ASSERT_FALSE(dErr) << dErr.message();
    
    channel_config["channel"] = data.key;
    
    channel_config["custom_scale"] = scale_config;

    config["channels"] = json::array();
    config["channels"].push_back(channel_config);
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

    if(reader.init() != 0) std::cout << "Failed to initialize reader" << std::endl;
    reader.start();
    std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
    auto [frame, err] = reader.read(b);
    std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;

     //iterate through each series and print the data
    uint32_t ai_count = 0;
    for(int i = 0; i < frame.series->size(); i++){
        std::cout << "\nSeries " << i << ":    " << frame.series->at(i) << "\n";
    }

    std::cout << std::endl;
    reader.stop();
}

///////////////////////////////////////////////////////////////////
//                          Voltage                              //
///////////////////////////////////////////////////////////////////
/// @brief Voltage
TEST(read_tests, one_analog_voltage_channel){

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

    if(reader.init() != 0) std::cout << "Failed to initialize reader" << std::endl;
    reader.start();
    std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
    auto [frame, err] = reader.read(b);
    std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;

     //iterate through each series and print the data
    uint32_t ai_count = 0;
    for(int i = 0; i < frame.series->size(); i++){
        std::cout << "\nSeries " << i << ":    " << frame.series->at(i) << "\n";
    }

    std::cout << std::endl;
    reader.stop();
}

///////////////////////////////////////////////////////////////////
//                          Torque                               //
///////////////////////////////////////////////////////////////////\
///@brief torque bridge linear
TEST(read_tests, one_torque_linear_bridge_channel){
    // Create NI readerconfig json
    auto config = json{
            {"sample_rate", 5}, 
            {"stream_rate", 1}, 
            {"device_location", "Dev3"},
            {"type", "ni_analog_read"},
            {"test", true},    
            {"device", ""}
    };
   
    auto channel_config = json{
        {"name", "test_ni_channel"},
        {"type", "ai_torque_bridge_two_point_lin"},
        {"port", 0},
        {"max_val", 1},
        {"min_val", 0},
        {"units", "NewtonMeters"},
        {"enabled", true},
        {"key", "key"},
        {"bridge_config", "HalfBridge"},
        {"voltage_excit_source","Internal"}, 
        {"voltage_excit_val", 2.5}, // same as below
        {"nominal_bridge_resistance", 1}, // TODO: figure out what a relistic val is 
        {"first_electrical_val", 0.0},
        {"second_electrical_val", 1.0},
        {"electrical_units", "VoltsPerVolt"},
        {"first_physical_val", 0.0},
        {"second_physical_val", 10.0},
        {"physical_units", "NewtonMeters"}
    };
    auto scale_config = json{
        {"type","none"}
    };
    analog_channel_helper(config, scale_config, channel_config);
}
///@brief torque bridge table
TEST(read_tests, one_torque_table_bridge_channel){
    // Create NI readerconfig json
    auto config = json{
            {"sample_rate", 5}, 
            {"stream_rate", 1}, 
            {"device_location", "Dev3"},
            {"type", "ni_analog_read"},
            {"test", true},    
            {"device", ""}
    };
   
    auto channel_config = json{
        {"name", "test_ni_channel"},
        {"type", "ai_torque_bridge_table"},
        {"port", 0},
        {"max_val", 1},
        {"min_val", 0},
        {"units", "NewtonMeters"},
        {"enabled", true},
        {"key", "key"},
        {"bridge_config", "HalfBridge"},
        {"voltage_excit_source","Internal"}, 
        {"voltage_excit_val", 2.5}, // same as below
        {"nominal_bridge_resistance", 1}, // TODO: figure out what a relistic val is 
        {"electrical_units", "VoltsPerVolt"},
        {"physical_units", "NewtonMeters"},
        {"electrical_vals", {0.0, 1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0}},
        {"physical_vals", {0.0, 50.0, 100.0, 150.0, 200.0, 250.0, 300.0, 350.0, 400.0, 450.0, 500.0}},
        {"num_physical_vals", 11},
        {"num_electrical_vals", 11}

    };

    auto scale_config = json{
        {"type","none"}
    };
    analog_channel_helper(config, scale_config, channel_config);
}
///@brief torque bridge polynomial
TEST(read_tests, one_torque_polynomial_bridge_channel){
    // Create NI readerconfig json
    auto config = json{
            {"sample_rate", 5}, 
            {"stream_rate", 1}, 
            {"device_location", "Dev3"},
            {"type", "ni_analog_read"},
            {"test", true},    
            {"device", ""}
    };
   
    auto channel_config = json{
        {"name", "test_ni_channel"},
        {"type", "ai_torque_bridge_polynomial"},
        {"port", 0},
        {"max_val", 1},
        {"min_val", 0},
        {"units", "NewtonMeters"},
        {"enabled", true},
        {"key", "key"},
        {"bridge_config", "HalfBridge"},
        {"voltage_excit_source","Internal"}, 
        {"voltage_excit_val", 2.5}, // same as below
        {"nominal_bridge_resistance", 1}, // TODO: figure out what a relistic val is 
         {"electrical_units", "VoltsPerVolt"},
        {"physical_units", "NewtonMeters"},
        {"forward_coeffs", {1, 3, 2}},
        {"num_forward_coeffs", 3},
        {"num_reverse_coeffs", 3}
    };

    auto scale_config = json{
        {"type","none"}
    };
    analog_channel_helper(config, scale_config, channel_config);
}
///////////////////////////////////////////////////////////////////
//                          Velocity                             //
///////////////////////////////////////////////////////////////////
///@brief velocity
TEST(read_tests, one_velocity_channel){
    // Create NI readerconfig json
    auto config = json{
        {"sample_rate", 100}, 
        {"stream_rate", 20}, 
        {"device_location", "Dev9"},
        {"type", "ni_analog_read"},
        {"test", true},    
        {"device", ""}
    };
   
    auto channel_config = json{
        {"name", "test_ni_channel"},
        {"type", "ai_velocity_iepe"},
        {"port", 0},
        {"max_val", 0.1},
        {"min_val", -0.1},
        {"units", "MetersPerSecond"},
        {"enabled", true},
        {"key", "key"},
        {"terminal_config", "Default"}, // TODO try pseudo differential
        {"voltage_excit_source","Internal"},
        {"voltage_excit_val", 0.0},
        {"sensitivity", 50},
        {"sensitivity_units", "MillivoltsPerMillimeterPerSecond"}
    };

    auto scale_config = json{
        {"type","none"}
    };
    analog_channel_helper(config, scale_config, channel_config);
}
///////////////////////////////////////////////////////////////////
//                          Force                                //
///////////////////////////////////////////////////////////////////
///@brief Force Bridge Polynomial
TEST(read_tests, one_force_polynomial_bridge_channel){
    // Create NI readerconfig json
    auto config = json{
            {"sample_rate", 5}, 
            {"stream_rate", 1}, 
            {"device_location", "Dev3"},
            {"type", "ni_analog_read"},
            {"test", true},    
            {"device", ""}
    };
   
    auto channel_config = json{
        {"name", "test_ni_channel"},
        {"type", "ai_force_bridge_polynomial"},
        {"port", 0},
        {"max_val", 1},
        {"min_val", 0},
        {"units", "Newtons"},
        {"enabled", true},
        {"key", "key"},
        {"bridge_config", "HalfBridge"},
        {"voltage_excit_source","Internal"}, 
        {"voltage_excit_val", 2.5}, // same as below
        {"nominal_bridge_resistance", 1}, // TODO: figure out what a relistic val is 
        {"electrical_units", "VoltsPerVolt"},
        {"physical_units", "Newtons"},
        {"forward_coeffs", {1, 3, 2}},
        {"num_forward_coeffs", 3},
        {"num_reverse_coeffs", 3}
    };

    auto scale_config = json{
        {"type","none"}
    };
    analog_channel_helper(config, scale_config, channel_config);
}
///@brief Force Bridge Table
TEST(read_tests, one_force_table_bridge_channel){
    // Create NI readerconfig json
    auto config = json{
            {"sample_rate", 5}, 
            {"stream_rate", 1}, 
            {"device_location", "Dev3"},
            {"type", "ni_analog_read"},
            {"test", true},    
            {"device", ""}
    };
   
    auto channel_config = json{
        {"name", "test_ni_channel"},
        {"type", "ai_force_bridge_table"},
        {"port", 0},
        {"max_val", 1},
        {"min_val", 0},
        {"units", "Newtons"},
        {"enabled", true},
        {"key", "key"},
        {"bridge_config", "HalfBridge"},
        {"voltage_excit_source","Internal"}, 
        {"voltage_excit_val", 2.5}, // same as below
        {"nominal_bridge_resistance", 1}, // TODO: figure out what a relistic val is 
        {"electrical_units", "VoltsPerVolt"},
        {"physical_units", "Newtons"},
        {"electrical_vals", {0.0, 1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0}},
        {"physical_vals", {0.0, 50.0, 100.0, 150.0, 200.0, 250.0, 300.0, 350.0, 400.0, 450.0, 500.0}},
        {"num_physical_vals", 11},
        {"num_electrical_vals", 11}
    };

    auto scale_config = json{
        {"type","none"}
    };
    analog_channel_helper(config, scale_config, channel_config);
}
///@brief Force Bridge Linear
TEST(read_tests, one_force_linear_bridge_channel){
    // Create NI readerconfig json
    auto config = json{
            {"sample_rate", 5}, 
            {"stream_rate", 1}, 
            {"device_location", "Dev3"},
            {"type", "ni_analog_read"},
            {"test", true},    
            {"device", ""}
    };
   
    auto channel_config = json{
        {"name", "test_ni_channel"},
        {"type", "ai_force_bridge_two_point_lin"},
        {"port", 0},
        {"max_val", 1},
        {"min_val", 0},
        {"units", "Newtons"},
        {"enabled", true},
        {"key", "key"},
        {"bridge_config", "HalfBridge"},
        {"voltage_excit_source","Internal"}, 
        {"voltage_excit_val", 2.5}, // same as below
        {"nominal_bridge_resistance", 1}, // TODO: figure out what a relistic val is 
        {"first_electrical_val", 0.0},
        {"second_electrical_val", 1.0},
        {"electrical_units", "VoltsPerVolt"},
        {"first_physical_val", 0.0},
        {"second_physical_val", 10.0},
        {"physical_units", "Newtons"}
    };

    auto scale_config = json{
        {"type","none"}
    };
    analog_channel_helper(config, scale_config, channel_config);
}
///@brief force IEPE
TEST(read_tests, one_force_iepe_channel){
    // Create NI readerconfig json
    auto config = json{
        {"sample_rate", 100}, 
        {"stream_rate", 20}, 
        {"device_location", "Dev9"},
        {"type", "ni_analog_read"},
        {"test", true},    
        {"device", ""}
    };
   
    auto channel_config = json{
        {"name", "test_ni_channel"},
        {"type", "ai_force_iepe"},
        {"port", 0},
        {"max_val", 0.1},
        {"min_val", -0.1},
        {"units", "Newtons"},
        {"enabled", true},
        {"key", "key"},
        {"terminal_config", "Default"}, // TODO try pseudo differential
        {"voltage_excit_source","Internal"},
        {"voltage_excit_val", 0.0},
        {"sensitivity", 50},
        {"sensitivity_units", "mVoltsPerNewton"}
    };

    auto scale_config = json{
        {"type","none"}
    };
    analog_channel_helper(config, scale_config, channel_config);
}
///////////////////////////////////////////////////////////////////
//                          Pressure                             //
///////////////////////////////////////////////////////////////////
///@brief pressure bridge polynomial
TEST(read_tests, one_pressure_polynomial_bridge_channel){
    // Create NI readerconfig json
    auto config = json{
            {"sample_rate", 5}, 
            {"stream_rate", 1}, 
            {"device_location", "Dev3"},
            {"type", "ni_analog_read"},
            {"test", true},    
            {"device", ""}
    };
   
    auto channel_config = json{
        {"name", "test_ni_channel"},
        {"type", "ai_pressure_bridge_polynomial"},
        {"port", 0},
        {"max_val", 1},
        {"min_val", 0},
        {"units", "Pascals"},
        {"enabled", true},
        {"key", "key"},
        {"bridge_config", "HalfBridge"},
        {"voltage_excit_source","Internal"}, 
        {"voltage_excit_val", 2.5}, // same as below
        {"nominal_bridge_resistance", 1}, // TODO: figure out what a relistic val is 
        {"electrical_units", "VoltsPerVolt"},
        {"physical_units", "Pascals"},
        {"forward_coeffs", {1, 3, 2}},
        {"num_forward_coeffs", 3},
        {"num_reverse_coeffs", 3}
    };

    auto scale_config = json{
        {"type","none"}
    };

    analog_channel_helper(config, scale_config, channel_config);
}
///@brief pressure bridge table
TEST(read_tests, one_pressure_table_bridge_channel){
    // Create NI readerconfig json
    auto config = json{
            {"sample_rate", 5}, 
            {"stream_rate", 1}, 
            {"device_location", "Dev3"},
            {"type", "ni_analog_read"},
            {"test", true},    
            {"device", ""}
    };
   
    auto channel_config = json{
        {"name", "test_ni_channel"},
        {"type", "ai_pressure_bridge_table"},
        {"port", 0},
        {"max_val", 25},
        {"min_val", 0},
        {"units", "Pascals"},
        {"enabled", true},
        {"key", "key"},
        {"bridge_config", "HalfBridge"},
        {"voltage_excit_source","Internal"}, 
        {"voltage_excit_val", 2.5}, // same as below
        {"nominal_bridge_resistance", 1}, // TODO: figure out what a relistic val is 
        {"electrical_units", "VoltsPerVolt"},
        {"physical_units", "Pascals"},
        {"electrical_vals", {0.0, 1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0}},
        {"physical_vals", {0.0, 50.0, 100.0, 150.0, 200.0, 250.0, 300.0, 350.0, 400.0, 450.0, 500.0}},
        {"num_physical_vals", 11},
        {"num_electrical_vals", 11}
    };

    auto scale_config = json{
        {"type","none"}
    };

    analog_channel_helper(config, scale_config, channel_config);
}
///@brief pressure bridge linear
TEST(read_tests, one_pressure_linear_bridge_channel){
    // Create NI readerconfig json
    auto config = json{
            {"sample_rate", 5}, 
            {"stream_rate", 1}, 
            {"device_location", "Dev3"},
            {"type", "ni_analog_read"},
            {"test", true},    
            {"device", ""}
    };
   
    auto channel_config = json{
        {"name", "test_ni_channel"},
        {"type", "ai_bridge"},
        {"port", 0},
        {"max_val", 0.5},
        {"min_val", -0.5},
        {"units", "VoltsPerVolt"},
        {"enabled", true},
        {"key", "key"},
        {"bridge_config", "HalfBridge"},
        {"voltage_excit_source","Internal"}, 
        {"voltage_excit_val", 2.5}, // same as below
        {"nominal_bridge_resistance", 1}, // TODO: figure out what a relistic val is 
        {"first_electrical_val", 0.0},
        {"second_eletrical_val", 1.0},
        {"electrical_units", "Volts"},
        {"first_physical_val", 0.0},
        {"second_physical_val", 10.0},
        {"physical_units", "Pascals"}
    };

    auto scale_config = json{
        {"type","none"}
    };

    analog_channel_helper(config, scale_config, channel_config);
}
///////////////////////////////////////////////////////////////////
//                         Temperature                           //
///////////////////////////////////////////////////////////////////
///@brief Thermocouple 
TEST(read_tests, one_analog_thermocouple_channel){
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
            {"sample_rate", 5}, 
            {"stream_rate", 1}, 
            {"device_location", "Dev2"},
            {"type", "ni_analog_read"},
            {"test", true},    
            {"device", ""}
    };
   
    auto channel_config = json{
        {"name", "test_ni_channel"},
        {"type", "ai_thrmcpl"},
        {"port", 0},
        {"max_val", 500},
        {"min_val", 65},
        {"units", "K"},
        {"enabled", true},
        {"channel",data.key},
        {"key", "key"},
        {"thermocouple_type", "J"},
        {"cjc_source", "BuiltIn"},
        {"cjc_val", 25.0}
    };

    auto scale_config = json{
        {"type","none"}
    };

    channel_config["custom_scale"] = scale_config;

    config["channels"] = json::array();
    config["channels"].push_back(channel_config);
  
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

    if(reader.init() != 0) std::cout << "Failed to initialize reader" << std::endl;
    reader.start();
    std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
    auto [frame, err] = reader.read(b);
    std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;

     //iterate through each series and print the data
    uint32_t ai_count = 0;
    for(int i = 0; i < frame.series->size(); i++){
        std::cout << "\nSeries " << i << ":    " << frame.series->at(i) << "\n";
    }

    std::cout << std::endl;
    reader.stop();
}
///@brief RTD
TEST(read_tests, one_analog_RTD_channel){
    // Create NI readerconfig json
    auto config = json{
            {"sample_rate", 5}, 
            {"stream_rate", 1}, 
            {"device_location", "PXI1Slot3"},
            {"type", "ni_analog_read"},
            {"test", true},    
            {"device", ""}
    };
   
    auto channel_config = json{
        {"name", "test_ni_channel"},
        {"type", "ai_rtd"},
        {"port", 0},
        {"units", "C"},
        {"enabled", true},
        {"key", "key"},
        {"max_val", 100.0},
        {"min_val", 0.0},
        {"units", "C"},
        {"rtd_type", "PT375"},
        {"resistance_config", "4Wire"},
        {"r0", 100.0},
        {"voltage_excit_source","Internal"},
        {"voltage_excit_val",0.0009}
    };

    auto scale_config = json{
        {"type","none"}
    };

    analog_channel_helper(config, scale_config, channel_config);
}
///@brief Temperature Built in sensor
TEST(read_tests, one_analog_temp_built_in_sensor_channel){
    // Create NI readerconfig json
    auto config = json{
            {"sample_rate", 5}, 
            {"stream_rate", 1}, 
            {"device_location", "Dev1"},
            {"type", "ni_analog_read"},
            {"test", true},    
            {"device", ""}
    };
   
    auto channel_config = json{
        {"name", "test_ni_channel"},
        {"type", "ai_temp_built_in_sensor"},
        {"port", 0},
        {"units", "C"},
        {"enabled", true},
        {"key", "key"},
    };

    auto scale_config = json{
        {"type","none"}
    };

    analog_channel_helper(config, scale_config, channel_config);
}
///////////////////////////////////////////////////////////////////////////////////
//                                    Acceleration                               //
///////////////////////////////////////////////////////////////////////////////////
///@brief Acceleration
TEST(read_tests, one_acceleration_channel){
    // Create NI readerconfig json
    auto config = json{
        {"sample_rate", 100}, 
        {"stream_rate", 20}, 
        {"device_location", "Dev9"},
        {"type", "ni_analog_read"},
        {"test", true},    
        {"device", ""}
    };
   
    auto channel_config = json{
        {"name", "test_ni_channel"},
        {"type", "ai_accel"},
        {"port", 0},
        {"max_val", 100.0},
        {"min_val", -100.0},
        {"units", "AccelUnit_g"},
        {"enabled", true},
        {"key", "key"},
        {"terminal_config", "Default"}, // TODO try pseudo differential
        {"voltage_excit_source","Internal"},
        {"voltage_excit_val", 0.0},
        {"sensitivity", 50},
        {"sensitivity_units", "mVoltsPerG"}
    };

    auto scale_config = json{
        {"type","none"}
    };
    analog_channel_helper(config, scale_config, channel_config);
}
///////////////////////////////////////////////////////////////////
//                         Current                               //
///////////////////////////////////////////////////////////////////
///@brief Current
TEST(read_tests, one_analog_current_channel){

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
            "current",
            synnax::FLOAT32,
            time.key,
            false
    );
    ASSERT_FALSE(dErr) << dErr.message();

    // Create NI readerconfig json
    auto config = json{
            {"sample_rate", 5}, 
            {"stream_rate", 1}, 
            {"device_location", "PXI1Slot2"},
            {"type", "ni_analog_read"},
            {"test", true},    
            {"device", ""}
    };
   
    auto channel_config = json{
        {"name", "test_ni_channel"},
        {"type", "ai_current"},
        {"port", 0},
        {"max_val", 0.0004},
        {"min_val", 0},
        {"units", "Amps"},
        {"enabled", true},
        {"channel",data.key},
        {"key", "key"},
        {"shunt_resistor_loc", "Default"},
        {"ext_shunt_resistor_val", 249.0},
        {"terminal_config", "Default"}
    };

    auto scale_config = json{
        {"type","none"}
    };

    channel_config["custom_scale"] = scale_config;

    config["channels"] = json::array();
    config["channels"].push_back(channel_config);
  
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

    if(reader.init() != 0) std::cout << "Failed to initialize reader" << std::endl;
    reader.start();
    std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
    auto [frame, err] = reader.read(b);
    std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;

     //iterate through each series and print the data
    uint32_t ai_count = 0;
    for(int i = 0; i < frame.series->size(); i++){
        std::cout << "\nSeries " << i << ":    " << frame.series->at(i) << "\n";
    }

    std::cout << std::endl;
    reader.stop();
}
///@brief Microphone
TEST(read_tests, one_microphone_channel){
    // Create NI readerconfig json
    auto config = json{
            {"sample_rate", 5}, 
            {"stream_rate", 1}, 
            {"device_location", "Dev9"},
            {"type", "ni_analog_read"},
            {"test", true},    
            {"device", ""}
    };
   
    auto channel_config = json{
        {"name", "test_ni_channel"},
        {"type", "ai_microphone"},
        {"port", 0},
        {"units", "Pascals"},
        {"enabled", true},
        {"key", "key"},
        {"voltage_excit_source","Internal"},
        {"voltage_excit_val", 0.0},
        {"terminal_config", "PseudoDiff"},
        {"mic_sensitivity", 50},
        {"max_snd_press_level",120}
    };


    auto scale_config = json{
        {"type","none"}
    };

    analog_channel_helper(config, scale_config, channel_config);
}
///@brief resistance
TEST(read_tests, one_resistance_channel){
    // Create NI readerconfig json
    auto config = json{
            {"sample_rate", 5}, 
            {"stream_rate", 1}, 
            {"device_location", "Dev3"},
            {"type", "ni_analog_read"},
            {"test", true},    
            {"device", ""}
    };
   
    auto channel_config = json{
        {"name", "test_ni_channel"},
        {"type", "ai_resistance"},
        {"port", 0},
        {"max_val", 10000},
        {"min_val", 0},
        {"units", "Ohms"},
        {"enabled", true},
        {"key", "key"},
        {"voltage_excit_source","Internal"},
        {"voltage_excit_val", 0.0005},
        {"resistance_config", "2Wire"},
    };

    auto scale_config = json{
        {"type","none"}
    };

    analog_channel_helper(config, scale_config, channel_config);
}
///@brief strain gage
TEST(read_tests, one_strain_gage){
    // Create NI readerconfig json
    auto config = json{
            {"sample_rate", 5}, 
            {"stream_rate", 1}, 
            {"device_location", "Dev3"},
            {"type", "ni_analog_read"},
            {"test", true},    
            {"device", ""}
    };
   
    auto channel_config = json{
        {"name", "test_ni_channel"},
        {"type", "ai_strain_gage"},
        {"port", 0},
        {"max_val", 0.005},
        {"min_val", -0.005},
        {"units", "Strain"},
        {"enabled", true},
        {"key", "key"},
        {"voltage_excit_source", "Internal"},
        {"strain_config", "FullBridgeI"},
        {"voltage_excit_val", 2.5},
        {"initial_bridge_voltage", 0.0},
        {"nominal_gage_resistance", 120},
        {"poisson_ratio", 0.3},
        {"gage_factor", 2.0},
        {"lead_wire_resistance", 0.0}
    };

    auto scale_config = json{
        {"type","none"}
    };

    analog_channel_helper(config, scale_config, channel_config);
}
///@brief Bridge Channel
TEST(read_tests, one_bridge_channel){
    // Create NI readerconfig json
    auto config = json{
            {"sample_rate", 5}, 
            {"stream_rate", 1}, 
            {"device_location", "Dev3"},
            {"type", "ni_analog_read"},
            {"test", true},    
            {"device", ""}
    };
   
    auto channel_config = json{
        {"name", "test_ni_channel"},
        {"type", "ai_bridge"},
        {"port", 0},
        {"max_val", 0.5},
        {"min_val", -0.5},
        {"units", "VoltsPerVolt"},
        {"enabled", true},
        {"key", "key"},
        {"bridge_config", "HalfBridge"},
        {"voltage_excit_source","Internal"}, 
        {"voltage_excit_val", 2.5}, // same as below
        {"nominal_bridge_resistance", 1}, // TODO: figure out what a relistic val is 
    };

    auto scale_config = json{
        {"type","none"}
    };

    analog_channel_helper(config, scale_config, channel_config);
}
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//                                                                                                              //
//                                              Error Handling                                                  //                  
//                                                                                                              //
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
// trying to configure a channel for a dev ice that doesnn't have that  channel

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//                                                                                                              //
//                                         Channel types without dedicated hardware found                       //                  
//                                                                                                              //
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
/*
///@brief RMS Voltage 
// TODO: find a device which actually supports this channel
TEST(read_tests, one_voltage_RMS_channel){
    // Create NI readerconfig json
    auto config = json{
        {"sample_rate", 100}, 
        {"stream_rate", 20}, 
        {"device_location", "Dev9"},
        {"type", "ni_analog_read"},
        {"test", true},    
        {"device", ""}
    };
   
    auto channel_config = json{
        {"name", "test_ni_channel"},
        {"type", "ai_voltage_rms"},
        {"port", 0},
        {"max_val", 10},
        {"min_val", 0},
        {"units", "Amps"},
        {"enabled", true},
        {"key", "key"},
        {"terminal_config", "Default"}
    };

    auto scale_config = json{
        {"type","none"}
    };
    // analog_channel_helper(config, scale_config, channel_config);
}

///@brief voltage with excitation
/// TODO: cant find a device which supports this channel
TEST(read_tests, one_voltage_with_excitation_channel){
    // Create NI readerconfig json
    auto config = json{
        {"sample_rate", 100}, 
        {"stream_rate", 20}, 
        {"device_location", "Dev3"},
        {"type", "ni_analog_read"},
        {"test", true},    
        {"device", ""}
    };
   
    auto channel_config = json{
        {"name", "test_ni_channel"},
        {"type", "ai_voltage_rms"},
        {"port", 0},
        {"max_val", -0.025},
        {"min_val", 0.025},
        {"units", "Amps"}, // FIXME
        {"enabled", true},
        {"key", "key"},
        {"terminal_config", "Default"},
        {"bridge_config", "FullBridge"},
        {"voltage_excit_source", "Internal"},
        {"voltage_excit_val", 2.5},
        {"use_excit_for_scaling", true}

    };

    auto scale_config = json{
        {"type","none"}
    };
    // analog_channel_helper(config, scale_config, channel_config);
}


///@brief ThermistorIEX
/// TODO: cant find a device that supports it specifically
TEST(read_tests, one_analog_thermistor_IEX_channel){
    // Create NI readerconfig json
    auto config = json{
            {"sample_rate", 5}, 
            {"stream_rate", 1}, 
            {"device_location", "Dev3"},
            {"type", "ni_analog_read"},
            {"test", true},    
            {"device", ""}
    };
   
    auto channel_config = json{
        {"name", "test_ni_channel"},
        {"type", "ai_thrmstr_iex"},
        {"port", 0},
        {"units", "C"},
        {"enabled", true},
        {"key", "key"},
        {"min_val", -5900.0},
        {"max_val", 10.0},
        {"resistance_config", "4Wire"},
        {"voltage_excit_source","External"},
        {"voltage_excit_val",0.0009},
        {"a", 0.003354016},
        {"b", 0.000256985},
        {"c", 0.000002620131}
    };

    auto scale_config = json{
        {"type","none"}
    };

    analog_channel_helper(config, scale_config, channel_config);
}

///@brief ThermistorVEX

///@brief Acceleration 4 wire dc voltage
// TODO: find devices which support this channel
TEST(read_tests, one_acceleration_4_wire_channel){
    // Create NI readerconfig json
    auto config = json{
        {"sample_rate", 100}, 
        {"stream_rate", 20}, 
        {"device_location", "Dev10"},
        {"type", "ni_analog_read"},
        {"test", true},    
        {"device", ""}
    };
   
    auto channel_config = json{
        {"name", "test_ni_channel"},
        {"type", "ai_accel_4_wire_dc_voltage"},
        {"port", 0},
        {"max_val", 100.0},
        {"min_val", -100.0},
        {"units", "AccelUnit_g"},
        {"enabled", true},
        {"key", "key"},
        {"terminal_config", "Default"}, // TODO try pseudo differential
        {"voltage_excit_source","Internal"},
        {"voltage_excit_val", 0.0},
        {"sensitivity", 50},
        {"sensitivity_units", "mVoltsPerG"},
        {"use_excit_for_scaling", false}
    };

    auto scale_config = json{
        {"type","none"}
    };
    // analog_channel_helper(config, scale_config, channel_config);
}

///@brief Acceleration Charge
// TODO: find devices which support this channel
TEST(read_tests, one_acceleration_charge_channel){
    // Create NI readerconfig json
    auto config = json{
        {"sample_rate", 100}, 
        {"stream_rate", 20}, 
        {"device_location", "Dev11"},
        {"type", "ni_analog_read"},
        {"test", true},    
        {"device", ""}
    };
   
    auto channel_config = json{
        {"name", "test_ni_channel"},
        {"type", "ai_accel_charge"},
        {"port", 0},
        {"max_val", 100.0},
        {"min_val", -100.0},
        {"units", "AccelUnit_g"},
        {"enabled", true},
        {"key", "key"},
        {"terminal_config", "Default"}, // TODO try pseudo differential
        {"voltage_excit_source","Internal"},
        {"voltage_excit_val", 0.0},
        {"sensitivity", 50},
        {"sensitivity_units", "mVoltsPerG"},
        {"use_excit_for_scaling", false}
    };

    auto scale_config = json{
        {"type","none"}
    };
    // analog_channel_helper(config, scale_config, channel_config);
}

///@brief rosette strain gage
//TODO: needs 3 channels to work
TEST(read_tests, one_rosette_strain_gage){
    // Create NI readerconfig json
    auto config = json{
            {"sample_rate", 5}, 
            {"stream_rate", 1}, 
            {"device_location", "Dev3"},
            {"type", "ni_analog_read"},
            {"test", true},    
            {"device", ""}
    };
   
    auto channel_config = json{
        {"name", "test_ni_channel"},
        {"type", "ai_rosette_strain_gage"},
        {"port", 0},
        {"max_val", 0.005},
        {"min_val", -0.005},
        {"units", "Strain"},
        {"enabled", true},
        {"key", "key"},
        {"voltage_excit_source", "Internal"},
        {"strain_config", "FullBridgeI"},
        {"voltage_excit_val", 2.5},
        {"initial_bridge_voltage", 0.0},
        {"nominal_gage_resistance", 120},
        {"poisson_ratio", 0.3},
        {"gage_factor", 2.0},
        {"lead_wire_resistance", 0.0},
        {"rosette_type", "RectangularRosette"},
        {"rosette_meas_type", "PrincipalStrain1"},
        {"gage_orientation", 0.0}
    };

    auto scale_config = json{
        {"type","none"}
    };

    analog_channel_helper(config, scale_config, channel_config);
}


///@brief frequency
TEST(read_tests, one_frequency_channel){
    // Create NI readerconfig json
    auto config = json{
            {"sample_rate", 5}, 
            {"stream_rate", 1}, 
            {"device_location", "Dev3"},
            {"type", "ni_analog_read"},
            {"test", true},    
            {"device", ""}
    };
   
    auto channel_config = json{
        {"name", "test_ni_channel"},
        {"type", "ai_freq_voltage"},
        {"port", 0},
        {"units", "Hz"},
        {"enabled", true},
        {"key", "key"},
        {"min_val", 1.0},
        {"max_val", 10000.0},
        {"threshold_level", 0.0},
        {"hysteresis", 0.2}
    };


    auto scale_config = json{
        {"type","none"}
    };

    analog_channel_helper(config, scale_config, channel_config);
}


///@brief charge

///@brief Current RMS
// TODO: find device which supports current RMS
TEST(read_tests, one_current_RMS_channel){
    // Create NI readerconfig json
    auto config = json{
            {"sample_rate", 5}, 
            {"stream_rate", 1}, 
            {"device_location", "Dev8"},
            {"type", "ni_analog_read"},
            {"test", true},    
            {"device", ""}
    };
   
    auto channel_config = json{
        {"name", "test_ni_channel"},
        {"type", "ai_current_rms"},
        {"port", 0},
        {"max_val", 0.0004},
        {"min_val", 0},
        {"units", "Amps"},
        {"enabled", true},
        {"key", "key"},
        {"shunt_resistor_loc", "Default"},
        {"ext_shunt_resistor_val", 249.0},
        {"terminal_config", "Default"}
    };

    auto scale_config = json{
        {"type","none"}
    };

    // analog_channel_helper(config, scale_config, channel_config);
}

*/