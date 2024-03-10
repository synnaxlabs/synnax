// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// Std.
#include <thread>

/// GTest
#include <include/gtest/gtest.h>

/// Internal
#include "synnax/synnax.h"
#include "synnax/testutil/testutil.h"

/// @brief it should correctly receive a frame of streamed telemetry from the DB.
TEST(FramerTests, testStreamBasic)
{
        auto client = new_test_client();
        auto [data, cErr] = client.channels.create(
            "data",
            synnax::FLOAT32,
            1 * synnax::HZ);
        ASSERT_FALSE(cErr) << cErr.message();
        auto now = synnax::TimeStamp::now();
        std::vector<synnax::ChannelKey> channels = {data.key};
        auto [writer, wErr] = client.telem.openWriter(synnax::WriterConfig{
            .channels = channels,
            .start = now,
            .authorities = std::vector<synnax::Authority>{synnax::ABSOLUTE},
            .subject = synnax::Subject{.name = "test_writer"}});
        ASSERT_FALSE(wErr) << wErr.message();

        auto [streamer, sErr] = client.telem.openStreamer(synnax::StreamerConfig{
            .channels = channels,
        });
        ASSERT_FALSE(sErr) << sErr.message();

        // Sleep for 5 milliseconds to allow for the streamer to bootstrap.
        std::this_thread::sleep_for(std::chrono::milliseconds(5));

        auto frame = synnax::Frame(1);
        frame.add(
            data.key,
            synnax::Series(std::vector<std::float_t>{1.0}));
        ASSERT_TRUE(writer.write(std::move(frame)));
        auto [res_frame, recErr] = streamer.read();
        ASSERT_FALSE(recErr) << recErr.message();

        ASSERT_EQ(res_frame.size(), 1);
        ASSERT_EQ(res_frame.series->at(0).float32()[0], 1.0);

        auto wcErr = writer.close();
        ASSERT_FALSE(cErr) << cErr.message();
        auto wsErr = streamer.close();
        ASSERT_FALSE(wsErr) << wsErr.message();
}