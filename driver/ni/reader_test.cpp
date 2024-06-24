// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
//                                          Functional Tests                                                    //
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
TEST(ReaderTests, test_read_one_digital_channel
){
LOG(INFO)
<< "test_read_one_digital_channel: "; //<< std::endl;

// Create NI readerconfig
auto config = json{
        {"sample_rate", 100}, // dont actually need these here
        {"stream_rate", 20}, // same as above
        {"device_name", "PXI1Slot2_2"},
        {"reader_type", "digitalReader"}
};
add_index_channel_JSON(config,
"idx", 1);
add_DI_channel_JSON(config,
"d1", 65531, 0, 0);


// Synnax infrustructure
auto client = std::make_shared<synnax::Synnax>(new_test_client());

auto task = synnax::Task(
        "my_task",
        "NI_digitalRead",
        to_string(config)
);

auto mockCtx = std::make_shared<task::MockContext>(client);

std::this_thread::sleep_for(std::chrono::milliseconds(300)
);
// Now construct NI reader
TaskHandle taskHandle;
ni::NiDAQmxInterface::CreateTask("",&taskHandle);

auto reader = ni::daqReader(taskHandle,
                            mockCtx,
                            task);

std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
auto [frame, err] = reader.read();
std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;

//iterate through each series and print the data
for(
int i = 0;
i<frame.series->

size();

i++){
std::cout << "\n\n Series " << i << ": \n";
// check series type before casting
if (frame.series->
at(i)
.data_type == synnax::UINT8){
auto s = frame.series->at(i).uint8();
for (
int j = 0;
j<s.

size();

j++){
std::cout << (uint32_t)s[j] << ", ";
ASSERT_TRUE((s[j]
== 1) || (s[j] == 0));
}
}
else if(frame.series->
at(i)
.data_type == synnax::TIMESTAMP){
auto s = frame.series->at(i).uint64();
for (
int j = 0;
j<s.

size();

j++){
std::cout << s[j] << ", ";
ASSERT_TRUE((s[j]
>= initial_timestamp) && (s[j] <= final_timestamp));
}
}
}
std::cout <<
std::endl;
}


TEST(ReaderTests, test_read_multiple_digital_channel
){
LOG(INFO)
<< "test_read_multiple_digital_channel: "; //<< std::endl;

// Create NI readerconfig
auto config = json{
        {"sample_rate", 1000}, // dont actually need these here
        {"stream_rate", 20}, // same as above
        {"device_name", "PXI1Slot2_2"},
        {"reader_type", "digitalReader"}
};
add_index_channel_JSON(config,
"idx", 1);
add_DI_channel_JSON(config,
"d1", 65531, 0, 0);
add_DI_channel_JSON(config,
"d1", 65531, 0, 1);
add_DI_channel_JSON(config,
"d1", 65532, 0, 2);
add_DI_channel_JSON(config,
"d1", 65533, 0, 3);
add_DI_channel_JSON(config,
"d1", 65534, 0, 4);
add_DI_channel_JSON(config,
"d1", 65535, 0, 5);
add_DI_channel_JSON(config,
"d1", 65536, 0, 6);
add_DI_channel_JSON(config,
"d1", 65537, 0, 7);


// Synnax infrustructure
auto client = std::make_shared<synnax::Synnax>(new_test_client());

auto task = synnax::Task(
        "my_task",
        "NI_digitalRead",
        to_string(config)
);

auto mockCtx = std::make_shared<task::MockContext>(client);

std::this_thread::sleep_for(std::chrono::milliseconds(300)
);
// Now construct NI reader
TaskHandle taskHandle;
ni::NiDAQmxInterface::CreateTask("",&taskHandle);

auto reader = ni::daqReader(taskHandle,
                            mockCtx,
                            task);
for(
int i = 0;
i < 50; i++ ) { // test for 50 read cycles
std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
auto [frame, err] = reader.read();
std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;


//iterate through each series and print the data
for(
int i = 0;
i<frame.series->

size();

i++){
std::cout << "\n\n Series " << i << ": \n";
// check series type before casting
if (frame.series->
at(i)
.data_type == synnax::UINT8){
auto s = frame.series->at(i).uint8();
for (
int j = 0;
j<s.

size();

j++){
std::cout << (uint32_t)s[j] << ", ";
ASSERT_TRUE((s[j]
== 1) || (s[j] == 0));
}
}
else if(frame.series->
at(i)
.data_type == synnax::TIMESTAMP){
auto s = frame.series->at(i).uint64();
for (
int j = 0;
j<s.

size();

j++){
std::cout << s[j] << ", ";
ASSERT_TRUE((s[j]
>= initial_timestamp) && (s[j] <= final_timestamp));
}
}
}
std::cout <<
std::endl;
}
}


TEST(ReaderTests, test_read_one_analog_channel
){
LOG(INFO)
<< "test_read_one_analog_channel: "; //<< std::endl;

// Create NI readerconfig
auto config = json{
        {"sample_rate", 100}, // dont actually need these here
        {"stream_rate", 20}, // same as above
        {"device_name", "Dev1"},
        {"reader_type", "analogReader"}
};
add_index_channel_JSON(config,
"idx", 1);
add_AI_channel_JSON(config,
"a1", 65531, 0, -10.0, 10.0, "Default");

//print json as a string
// std::cout << config.dump(4) << std::endl;

// Synnax infrustructure
auto client = std::make_shared<synnax::Synnax>(new_test_client());

auto task = synnax::Task(
        "my_task",
        "NI_analogRead",
        to_string(config)
);

auto mockCtx = std::make_shared<task::MockContext>(client);

std::this_thread::sleep_for(std::chrono::milliseconds(300)
);
// Now construct NI reader
TaskHandle taskHandle;
ni::NiDAQmxInterface::CreateTask("",&taskHandle);

auto reader = ni::daqReader(taskHandle,
                            mockCtx,
                            task); // analog reader

std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
auto [frame, err] = reader.read();
std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;

//iterate through each series and print the data
uint32_t ai_count = 0;
for(
int i = 0;
i<frame.series->

size();

i++){
std::cout << "\n\n Series " << i << ": \n";
// check series type before casting
if (frame.series->
at(i)
.data_type == synnax::FLOAT32){
auto s = frame.series->at(i).float32();
for (
int j = 0;
j<s.

size();

j++){
std::cout << s[j] << ", ";
// ASSERT_TRUE((s[j] == 1) || (s[j] == 0));
ASSERT_NEAR(s[j], ai_count,
1);
}
ai_count++;
}
else if(frame.series->
at(i)
.data_type == synnax::TIMESTAMP){
auto s = frame.series->at(i).uint64();
for (
int j = 0;
j<s.

size();

j++){
std::cout << s[j] << ", ";
ASSERT_TRUE((s[j]
>= initial_timestamp) && (s[j] <= final_timestamp));
}
}
}
std::cout <<
std::endl;
}


TEST(ReaderTests, test_read_multiple_analog_channels
){
LOG(INFO)
<< "test_read_multiple_analog_channels: "; //<< std::endl;

// Create NI readerconfig
auto config = json{
        {"sample_rate", 2000}, // dont actually need these here
        {"stream_rate", 20}, // same as above
        {"device_name", "Dev1"},
        {"reader_type", "analogReader"}
};
add_index_channel_JSON(config,
"idx", 1);
add_AI_channel_JSON(config,
"a0", 65531, 0, -10.0, 10.0, "Default");
add_AI_channel_JSON(config,
"a1", 65532, 1, -10.0, 10.0, "Default");
add_AI_channel_JSON(config,
"a2", 65534, 2, -10.0, 10.0, "Default");
add_AI_channel_JSON(config,
"a3", 65535, 3, -10.0, 10.0, "Default");
add_AI_channel_JSON(config,
"a4", 65536, 4, -10.0, 10.0, "Default");

//print json as a string
//std::cout << config.dump(4) << std::endl;

// Synnax infrustructure
auto client = std::make_shared<synnax::Synnax>(new_test_client());

auto task = synnax::Task(
        "my_task",
        "NI_analogRead",
        to_string(config)
);

auto mockCtx = std::make_shared<task::MockContext>(client);

std::this_thread::sleep_for(std::chrono::milliseconds(300)
);
// Now construct NI reader
TaskHandle taskHandle;
ni::NiDAQmxInterface::CreateTask("",&taskHandle);

auto reader = ni::daqReader(taskHandle,
                            mockCtx,
                            task); // analog reader


for(
int i = 0;
i < 50; i++ ) { // test for 50 read cycles
std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
auto [frame, err] = reader.read();
std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;

//iterate through each series and print the data
for(
int i = 0;
i<frame.series->

size();

i++){
std::cout << "\n\n Series " << i << ": \n";
// check series type before casting
if (frame.series->
at(i)
.data_type == synnax::FLOAT32){
auto s = frame.series->at(i).float32();
for (
int j = 0;
j<s.

size();

j++){
std::cout << s[j] << ", ";
ASSERT_NEAR(s[j],
0, 10); // can be any value of a sign wave from -10 to 10
}
}
else if(frame.series->
at(i)
.data_type == synnax::TIMESTAMP){
auto s = frame.series->at(i).uint64();
for (
int j = 0;
j<s.

size();

j++){
std::cout << s[j] << ", ";
ASSERT_TRUE((s[j]
>= initial_timestamp) && (s[j] <= final_timestamp));
}
}
}
std::cout <<
std::endl;
}


}

TEST(WriterTests, test_write_one_digital_channel
){
LOG(INFO)
<< "test_read_one_digital_channel: "; //<< std::endl;

// Create NI readerconfig
auto config = json{
        {"device_name", "Dev1"},
        {"stream_rate", 1}
};
add_index_channel_JSON(config,
"do1_idx", 1);
add_DO_channel_JSON(config,
"do1_command", 65531, 65532, 0, 0);
add_drive_state_index_channel_JSON(config,
"do1_state", 2);


// Synnax infrustructure
auto client = std::make_shared<synnax::Synnax>(new_test_client());

auto task = synnax::Task(
        "my_task",
        "NI_digitalWriter",
        to_string(config)
);

auto mockCtx = std::make_shared<task::MockContext>(client);

std::this_thread::sleep_for(std::chrono::milliseconds(300)
);

// Now construct NI writer
TaskHandle taskHandle;
ni::NiDAQmxInterface::CreateTask("",&taskHandle);
auto writer = ni::daqWriter(taskHandle,
                            mockCtx,
                            task);

// construct synnax frame to write a 1
auto cmd_vec = std::vector<uint8_t>{1};
auto cmd_frame = synnax::Frame(2);
cmd_frame.add(  1,
synnax::Series(std::vector<uint64_t>{(synnax::TimeStamp::now()).value},
        synnax::TIMESTAMP
));
cmd_frame.add(  65531,
synnax::Series(cmd_vec)
);


std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;

auto werr = writer.write(std::move(cmd_frame));
auto [frame, err] = writer.writer_state_source->read();
std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;


//  iterate through each series and print the data
int cmd_count = 0;
for(
int i = 0;
i<frame.series->

size();

i++){
std::cout << "\n\n Series " << i << ": \n";
// check series type before casting
if (frame.series->
at(i)
.data_type == synnax::UINT8){
auto s = frame.series->at(i).uint8();
for (
int j = 0;
j<s.

size();

j++){
std::cout << (uint32_t)s[j] << ", ";
ASSERT_EQ(s[j], cmd_vec[cmd_count]
);
}
cmd_count++;
}
else if(frame.series->
at(i)
.data_type == synnax::TIMESTAMP){
auto s = frame.series->at(i).uint64();
for (
int j = 0;
j<s.

size();

j++){
std::cout << s[j] << ", ";
ASSERT_TRUE((s[j]
>= initial_timestamp) && (s[j] <= final_timestamp));
}
}
}
std::cout <<
std::endl;

// now write a 0
cmd_vec = std::vector<uint8_t>{0};
cmd_frame = synnax::Frame(2);
cmd_frame.add(  1,
synnax::Series(std::vector<uint64_t>{(synnax::TimeStamp::now()).value},
        synnax::TIMESTAMP
));
cmd_frame.add(  65531,
synnax::Series(cmd_vec)
);

initial_timestamp = (synnax::TimeStamp::now()).value;

auto werr2 = writer.write(std::move(cmd_frame));

auto [frame2, err2] = writer.writer_state_source->read();

final_timestamp = (synnax::TimeStamp::now()).value;

cmd_count = 0;
for(
int i = 0;
i<frame2.series->

size();

i++){
std::cout << "\n\n Series " << i << ": \n";
// check series type before casting
if (frame2.series->
at(i)
.data_type == synnax::UINT8){
auto s = frame2.series->at(i).uint8();
for (
int j = 0;
j<s.

size();

j++){
std::cout << (uint32_t)s[j] << ", ";
ASSERT_EQ(s[j], cmd_vec[cmd_count]
);
}
cmd_count++;
}
else if(frame2.series->
at(i)
.data_type == synnax::TIMESTAMP){
auto s = frame2.series->at(i).uint64();
for (
int j = 0;
j<s.

size();

j++){
std::cout << s[j] << ", ";
ASSERT_TRUE((s[j]
>= initial_timestamp) && (s[j] <= final_timestamp));
}
}
}
std::cout <<
std::endl;


}


TEST(NiWriterTests, test_write_multiple_digital_channel
){
LOG(INFO)
<< "test_write_multiple_digital_channel: "; //<< std::endl;

// Create NI readerconfig
auto config = json{
        {"device_name", "Dev1"},
        {"stream_rate", 1}
};

add_index_channel_JSON(config,
"do_idx", 1);
add_DO_channel_JSON(config,
"do1_command", 65531, 65532, 0, 0);
add_DO_channel_JSON(config,
"do1_command", 65533, 65534, 0, 1);
add_DO_channel_JSON(config,
"do1_command", 65535, 65536, 0, 2);
add_DO_channel_JSON(config,
"do1_command", 65537, 65538, 0, 3);
add_drive_state_index_channel_JSON(config,
"do1_state", 2);


// Synnax infrustructure
auto client = std::make_shared<synnax::Synnax>(new_test_client());

auto task = synnax::Task(
        "my_task",
        "NI_digitalWriter",
        to_string(config)
);

auto mockCtx = std::make_shared<task::MockContext>(client);

std::this_thread::sleep_for(std::chrono::milliseconds(300)
);

// Now construct NI writer
TaskHandle taskHandle;
ni::NiDAQmxInterface::CreateTask("",&taskHandle);
auto writer = ni::daqWriter(taskHandle,
                            mockCtx,
                            task);

// construct synnax frame to write
// auto map = std::map<uint32_t, std::vector<uint8_t>>{
//     {65531, {1,0,1,1}},
//     {65533, {0,1,0,0}},
//     {65535, {1,1,0,1}},
//     {65537, {0,0,1,0}}
// };
auto cmd_vec = std::vector<uint8_t>{1, 0, 1, 1};
auto cmd_frame = synnax::Frame(2);
cmd_frame.add(  1,
synnax::Series(std::vector<uint64_t>{(synnax::TimeStamp::now()).value},
        synnax::TIMESTAMP
));


for(
int i = 0;
i<cmd_vec.

size();

i++){
cmd_frame.add(  65531 + i*2,
synnax::Series(cmd_vec[i])
);
}


std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;

auto werr = writer.write(std::move(cmd_frame));
auto [frame, err] = writer.writer_state_source->read();

std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;


//  iterate through each series and print the data
int cmd_count = 0;
for(
int i = 0;
i<frame.series->

size();

i++){
std::cout << "\n\n Series " << i << ": \n";
// check series type before casting
if (frame.series->
at(i)
.data_type == synnax::UINT8){
auto s = frame.series->at(i).uint8();
for (
int j = 0;
j<s.

size();

j++){
std::cout << (uint32_t)s[j] << ", ";
ASSERT_EQ(s[j], cmd_vec[cmd_count]
);
}
cmd_count++;
}
else if(frame.series->
at(i)
.data_type == synnax::TIMESTAMP){
auto s = frame.series->at(i).uint64();
for (
int j = 0;
j<s.

size();

j++){
std::cout << s[j] << ", ";
ASSERT_TRUE((s[j]
>= initial_timestamp) && (s[j] <= final_timestamp));
}
}
}
std::cout <<
std::endl;


// write again
cmd_vec = std::vector<uint8_t>{0, 1, 0, 0};
cmd_frame = synnax::Frame(2);
cmd_frame.add(  1,
synnax::Series(std::vector<uint64_t>{(synnax::TimeStamp::now()).value},
        synnax::TIMESTAMP
));
for(
int i = 0;
i<cmd_vec.

size();

i++){
cmd_frame.add(  65531 + i*2,
synnax::Series(cmd_vec[i])
);
}
initial_timestamp = (synnax::TimeStamp::now()).value;

auto werr2 = writer.write(std::move(cmd_frame));
auto [frame2, err2] = writer.writer_state_source->read();

final_timestamp = (synnax::TimeStamp::now()).value;

cmd_count = 0;
for(
int i = 0;
i<frame.series->

size();

i++){
std::cout << "\n\n Series " << i << ": \n";
// check series type before casting
if (frame2.series->
at(i)
.data_type == synnax::UINT8){
auto s = frame2.series->at(i).uint8();
for (
int j = 0;
j<s.

size();

j++){
std::cout << (uint32_t)s[j] << ", ";
ASSERT_EQ(s[j], cmd_vec[cmd_count]
);
}
cmd_count++;
}
else if(frame2.series->
at(i)
.data_type == synnax::TIMESTAMP){
auto s = frame2.series->at(i).uint64();
for (
int j = 0;
j<s.

size();

j++){
std::cout << s[j] << ", ";
ASSERT_TRUE((s[j]
>= initial_timestamp) && (s[j] <= final_timestamp));
}
}
}
std::cout <<
std::endl;
}

