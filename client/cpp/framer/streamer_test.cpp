// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <thread>
#include <include/gtest/gtest.h>

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/xtest/xtest.h"

void test_downsample(
    const std::vector<int> &raw_data,
    std::vector<int> expected,
    int32_t downsample_factor
);

/// @brief it should correctly receive a frame of streamed telemetry from the DB.
TEST(StreamerTests, testStreamBasic) {
    auto client = new_test_client();
    auto data = ASSERT_NIL_P(client.channels.create(
        "data",
        telem::INT32_T,
        1 * telem::HZ));
    auto now = telem::TimeStamp::now();

    std::vector channels = {data.key};
    auto [streamer, sErr] = client.telem.open_streamer(synnax::StreamerConfig{
        channels,
    });
    auto writer = ASSERT_NIL_P(client.telem.open_writer(synnax::WriterConfig{
        channels,
        now,
        {telem::AUTH_ABSOLUTE},
        telem::ControlSubject{"test_writer"}
    }));

    auto frame = synnax::Frame(1);
    frame.emplace(data.key, telem::Series(1));
    ASSERT_TRUE(writer.write(frame));
    auto [res_frame, recErr] = streamer.read();
    ASSERT_FALSE(recErr) << recErr.message();

    ASSERT_EQ(res_frame.size(), 1);
    ASSERT_EQ(res_frame.series->at(0).values<int>()[0], 1);

    ASSERT_NIL(writer.close());
    ASSERT_NIL(streamer.close());
}

///@brief test streamer set channels after construction.
TEST(StreamerTests, testStreamSetChannels) {
    auto client = new_test_client();
    auto data = ASSERT_NIL_P(client.channels.create(
        "data",
        telem::FLOAT32_T,
        1 * telem::HZ));
    auto now = telem::TimeStamp::now();


    auto streamer = ASSERT_NIL_P(client.telem.open_streamer(synnax::StreamerConfig{
        {},
    }));

    auto setErr = streamer.set_channels({data.key});

    auto writer = ASSERT_NIL_P(client.telem.open_writer(synnax::WriterConfig{
        {data.key},
        now,
        {telem::AUTH_ABSOLUTE},
        telem::ControlSubject{"test_writer"}
    }));
    // Sleep for 5 milliseconds to allow for the streamer to process the updated keys.
    std::this_thread::sleep_for(std::chrono::milliseconds(5));
    ASSERT_FALSE(setErr) << setErr.message();

    auto frame = synnax::Frame(1);
    frame.emplace(
        data.key,
        telem::Series(std::vector<float>{
            1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0
        })
    );
    ASSERT_TRUE(writer.write(frame));
    auto res_frame = ASSERT_NIL_P(streamer.read());

    ASSERT_EQ(res_frame.size(), 1);
    ASSERT_EQ(res_frame.series->at(0).values<float>()[0], 1.0);

    ASSERT_NIL(writer.close());
    ASSERT_NIL(streamer.close());
}

/// @brief it should correctly receive a frame of streamed telemetry from the DB.
TEST(StreamerTests, TestStreamDownsample) {
    const std::vector data = {1, 2, 3, 4, 5, 6, 7, 8, 9, 10};

    test_downsample(data, data, 1);

    std::vector expected = {1, 3, 5, 7, 9};
    test_downsample(data, expected, 2);

    expected = {1, 4, 7, 10};
    test_downsample(data, expected, 3);

    expected = {1, 5, 9};
    test_downsample(data, expected, 4);

    expected = {1, 6};
    test_downsample(data, expected, 5);

    expected = {1, 7};
    test_downsample(data, expected, 6);

    expected = {1, 8};
    test_downsample(data, expected, 7);

    expected = {1, 9};
    test_downsample(data, expected, 8);

    expected = {1, 10};
    test_downsample(data, expected, 9);

    expected = {1};
    test_downsample(data, expected, 10);

    test_downsample(data, data, -1);

    test_downsample(data, data, 0);
}

void test_downsample(
    const std::vector<int> &raw_data,
    std::vector<int> expected,
    int32_t downsample_factor
) {
    auto client = new_test_client();
    auto data = ASSERT_NIL_P(client.channels.create(
        "data",
        telem::INT32_T,
        1 * telem::HZ));
    auto now = telem::TimeStamp::now();
    std::vector channels = {data.key};
    auto writer = ASSERT_NIL_P(client.telem.open_writer(synnax::WriterConfig{
        channels,
        now,
        std::vector{telem::AUTH_ABSOLUTE},
        telem::ControlSubject{"test_writer"}
    }));

    auto [streamer, sErr] = client.telem.open_streamer(synnax::StreamerConfig{
        channels,
        downsample_factor
    });

    // Sleep for 5 milliseconds to allow for the streamer to bootstrap.
    std::this_thread::sleep_for(std::chrono::milliseconds(5));

    auto frame = synnax::Frame(1);
    frame.emplace(
        data.key,
        telem::Series(raw_data)
    );
    ASSERT_TRUE(writer.write(frame));
    auto [res_frame, recErr] = streamer.read();
    ASSERT_FALSE(recErr) << recErr.message();

    for (int i = 0; i < expected.size(); i++)
        ASSERT_EQ(res_frame.series->at(0).values<int>()[i], expected[i]);

    ASSERT_NIL(writer.close());
    ASSERT_NIL(streamer.close());
}

void test_downsample_string(
    const std::vector<std::string> &raw_data,
    const std::vector<std::string> &expected,
    int32_t downsample_factor
) {
    auto client = new_test_client();

    // Create a virtual channel
    synnax::Channel virtual_channel("virtual_string_channel", telem::STRING_T, true);
    ASSERT_NIL(client.channels.create(virtual_channel));

    auto now = telem::TimeStamp::now();
    std::vector channels = {virtual_channel.key};
    auto writer = ASSERT_NIL_P(client.telem.open_writer(synnax::WriterConfig{
        channels,
        now,
        std::vector{telem::AUTH_ABSOLUTE},
        telem::ControlSubject{"test_writer"}
    }));

    auto streamer = ASSERT_NIL_P(client.telem.open_streamer(synnax::StreamerConfig{
        channels,
        downsample_factor
    }));

    // Sleep for 5 milliseconds to allow for the streamer to bootstrap.
    std::this_thread::sleep_for(std::chrono::milliseconds(5));

    auto frame = synnax::Frame(virtual_channel.key,
                               telem::Series(raw_data, telem::STRING_T));
    ASSERT_TRUE(writer.write(frame));
    auto res_frame = ASSERT_NIL_P(streamer.read());

    // Get the downsampled strings
    std::vector<std::string> received_strings = res_frame.series->at(0).strings();

    ASSERT_EQ(received_strings.size(), expected.size());
    for (size_t i = 0; i < expected.size(); i++)
        ASSERT_EQ(received_strings[i], expected[i]);

    ASSERT_NIL(writer.close());
    ASSERT_NIL(streamer.close());
}

TEST(StreamerTests, TestStreamDownsampleString) {
    const std::vector<std::string> data = {
        "a", "b", "c", "d", "e", "f", "g", "h", "i", "j"
    };
    const std::vector<std::string> expected = {"a", "c", "e", "g", "i"};
    test_downsample_string(data, expected, 2);
}
