// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include "arc/cpp/runtime/core/types.h"

#include "gtest/gtest.h"

// Tests for Arc-specific message types with SPSC queue

TEST(ArcQueueTest, ChannelUpdateMessage) {
    queue::SPSC<arc::ChannelUpdate> q(16);

    // Create a channel update with shared_ptr<Series>
    auto data = std::make_shared<telem::Series>(
        std::vector<float>{1.0f, 2.0f, 3.0f}
    );
    auto time = std::make_shared<telem::Series>(
        telem::Series::linspace(
            telem::TimeStamp(0),
            telem::TimeStamp(3000000000),
            3
        )
    );

    arc::ChannelUpdate update{1, data, time};
    EXPECT_TRUE(q.push(std::move(update)));

    // Pop and verify
    arc::ChannelUpdate result;
    EXPECT_TRUE(q.pop(result));
    EXPECT_EQ(result.channel_id, 1);
    EXPECT_NE(result.data, nullptr);
    EXPECT_EQ(result.data->size(), 3);
    EXPECT_EQ(result.data->at<float>(0), 1.0f);
}

TEST(ArcQueueTest, ChannelOutputMessage) {
    queue::SPSC<arc::ChannelOutput> q(16);

    // Create a channel output
    arc::ChannelOutput output{
        42,  // channel_id
        telem::SampleValue(3.14),
        telem::TimeStamp::now()
    };

    EXPECT_TRUE(q.push(std::move(output)));

    // Pop and verify
    arc::ChannelOutput result;
    EXPECT_TRUE(q.pop(result));
    EXPECT_EQ(result.channel_id, 42);
    EXPECT_EQ(std::get<double>(result.value), 3.14);
}
