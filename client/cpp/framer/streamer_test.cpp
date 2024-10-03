// Copyright 2024 Synnax Labs, Inc.
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

void test_downsample(
    std::vector<int> raw_data,
    std::vector<int> expected,
    int32_t downsample_factor
);
/// @brief it should correctly receive a frame of streamed telemetry from the DB.
TEST(FramerTests, testStreamBasic) {
    auto client = new_test_client();
    auto [data, cErr] = client.channels.create(
        "data",
        synnax::INT32,
        1 * synnax::HZ);
    ASSERT_FALSE(cErr) << cErr.message();
    auto now = synnax::TimeStamp::now();
    std::vector<synnax::ChannelKey> channels = {data.key};
    auto [writer, wErr] = client.telem.openWriter(synnax::WriterConfig{
        channels,
        now,
        std::vector<synnax::Authority>{synnax::AUTH_ABSOLUTE},
        synnax::ControlSubject{"test_writer"}
    });
    ASSERT_FALSE(wErr) << wErr.message();

    auto [streamer, sErr] = client.telem.openStreamer(synnax::StreamerConfig{
        channels,
    });

    // Sleep for 5 milliseconds to allow for the streamer to bootstrap.
    std::this_thread::sleep_for(std::chrono::milliseconds(5));

    auto frame = synnax::Frame(1);
    frame.add(
        data.key,
        synnax::Series(std::vector<int>{1}));
    ASSERT_TRUE(writer.write(std::move(frame)));
    auto [res_frame, recErr] = streamer.read();
    ASSERT_FALSE(recErr) << recErr.message();

    ASSERT_EQ(res_frame.size(), 1);
    ASSERT_EQ(res_frame.series->at(0).values<int>()[0], 1);

    auto wcErr = writer.close();
    ASSERT_FALSE(cErr) << cErr.message();
    auto wsErr = streamer.close();
    ASSERT_FALSE(wsErr) << wsErr.message();
}

///@brief test streamer set channels after construction.
TEST(FramerTests, testStreamSetChannels) {
    auto client = new_test_client();
    auto [data, cErr] = client.channels.create(
        "data",
        synnax::FLOAT32,
        1 * synnax::HZ);
    ASSERT_FALSE(cErr) << cErr.message();
    auto now = synnax::TimeStamp::now();
    auto [writer, wErr] = client.telem.openWriter(synnax::WriterConfig{
        {data.key},
        now,
        std::vector<synnax::Authority>{synnax::AUTH_ABSOLUTE},
        synnax::ControlSubject{"test_writer"}
    });
    ASSERT_FALSE(wErr) << wErr.message();

    auto [streamer, sErr] = client.telem.openStreamer(synnax::StreamerConfig{
        {},
    });

    auto setErr = streamer.setChannels({data.key});
    // Sleep for 5 milliseconds to allow for the streamer to process the updated keys.
    std::this_thread::sleep_for(std::chrono::milliseconds(5));
    ASSERT_FALSE(setErr) << setErr.message();

    auto frame = synnax::Frame(1);
    frame.add(
        data.key,
        synnax::Series(std::vector<std::float_t>{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0}));
    ASSERT_TRUE(writer.write(std::move(frame)));
    auto [res_frame, recErr] = streamer.read();
    ASSERT_FALSE(recErr) << recErr.message();

    ASSERT_EQ(res_frame.size(), 1);
    ASSERT_EQ(res_frame.series->at(0).values<float>()[0], 1.0);

    auto wcErr = writer.close();
    ASSERT_FALSE(cErr) << cErr.message();
    auto wsErr = streamer.close();
    ASSERT_FALSE(wsErr) << wsErr.message();
}

/// @brief it should correctly receive a frame of streamed telemetry from the DB.
TEST(FramerTests, TestStreamDownsample) {
    std::vector<int> data = {1, 2, 3, 4, 5, 6, 7, 8, 9, 10};

    test_downsample(data,data,1);

    std::vector<int> expected = {1, 3, 5, 7, 9};
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

    test_downsample(data, data,-1);

    test_downsample(data, data,0);
}

void test_downsample(
    std::vector<int> raw_data,
    std::vector<int> expected,
    int32_t downsample_factor
) {
    auto client = new_test_client();
    auto [data, cErr] = client.channels.create(
        "data",
        synnax::INT32,
        1 * synnax::HZ);
    ASSERT_FALSE(cErr) << cErr.message();
    auto now = synnax::TimeStamp::now();
    std::vector<synnax::ChannelKey> channels = {data.key};
    auto [writer, wErr] = client.telem.openWriter(synnax::WriterConfig{
        channels,
        now,
        std::vector<synnax::Authority>{synnax::AUTH_ABSOLUTE},
        synnax::ControlSubject{"test_writer"}
    });
    ASSERT_FALSE(wErr) << wErr.message();

    auto [streamer, sErr] = client.telem.openStreamer(synnax::StreamerConfig{
        channels,
        downsample_factor
    });

    // Sleep for 5 milliseconds to allow for the streamer to bootstrap.
    std::this_thread::sleep_for(std::chrono::milliseconds(5));

    auto frame = synnax::Frame(1);
    frame.add(
        data.key,
    synnax::Series(raw_data)
    );
    ASSERT_TRUE(writer.write(std::move(frame)));
    auto [res_frame, recErr] = streamer.read();
    ASSERT_FALSE(recErr) << recErr.message();

    for (int i = 0; i < expected.size(); i++)
        ASSERT_EQ(res_frame.series->at(0).values<int>()[i], expected[i]);

    auto wcErr = writer.close();
    ASSERT_FALSE(cErr) << cErr.message();
    auto wsErr = streamer.close();
    ASSERT_FALSE(wsErr) << wsErr.message();
}

void test_downsample_string(
    const std::vector<std::string>& raw_data,
    const std::vector<std::string>& expected,
    int32_t downsample_factor
) {
    auto client = new_test_client();

    // Create a virtual channel
    synnax::Channel virtual_channel("virtual_string_channel", synnax::STRING, true);
    auto err = client.channels.create(virtual_channel);
    ASSERT_FALSE(err) << err.message();

    auto now = synnax::TimeStamp::now();
    std::vector<synnax::ChannelKey> channels = {virtual_channel.key};
    auto [writer, wErr] = client.telem.openWriter(synnax::WriterConfig{
        channels,
        now,
        std::vector<synnax::Authority>{synnax::AUTH_ABSOLUTE},
        synnax::ControlSubject{"test_writer"}
    });
    ASSERT_FALSE(wErr) << wErr.message();

    auto [streamer, sErr] = client.telem.openStreamer(synnax::StreamerConfig{
        channels,
        downsample_factor
    });
    ASSERT_FALSE(sErr) << sErr.message();

    // Sleep for 5 milliseconds to allow for the streamer to bootstrap.
    std::this_thread::sleep_for(std::chrono::milliseconds(5));

    auto frame = synnax::Frame(1);
    frame.add(virtual_channel.key, synnax::Series(raw_data, synnax::STRING));
    ASSERT_TRUE(writer.write(std::move(frame)));
    auto [res_frame, recErr] = streamer.read();
    ASSERT_FALSE(recErr) << recErr.message();

    // Get the downsampled strings
    std::vector<std::string> received_strings = res_frame.series->at(0).strings();

    ASSERT_EQ(received_strings.size(), expected.size());
    for (size_t i = 0; i < expected.size(); i++)
        ASSERT_EQ(received_strings[i], expected[i]);

    auto wcErr = writer.close();
    ASSERT_FALSE(wcErr) << wcErr.message();
    auto wsErr = streamer.close();
    ASSERT_FALSE(wsErr) << wsErr.message();
}

TEST(FramerTests, TestStreamDownsampleString) {
    std::vector<std::string> data = {"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"};

    std::vector<std::string> expected = {"a", "c", "e", "g", "i"};
    test_downsample_string(data, expected, 2);

}