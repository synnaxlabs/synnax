// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdint>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/telem/video.h"
#include "x/cpp/testutil/testutil.h"

namespace x::telem {
const VideoFrame KEY{
    .keyframe = true,
    .codec = VideoCodec::H264,
    .data = {0, 0, 0, 1, 0x67},
};
const VideoFrame DELTA{
    .keyframe = false,
    .codec = VideoCodec::H264,
    .data = {0, 0, 1, 0x41},
};

/// @brief it should prefix the access unit with the flags and codec bytes.
TEST(VideoFrameTest, testEncode) {
    ASSERT_EQ(KEY.encode(), (std::vector<std::uint8_t>{1, 1, 0, 0, 0, 1, 0x67}));
    ASSERT_EQ(DELTA.encode(), (std::vector<std::uint8_t>{0, 1, 0, 0, 1, 0x41}));
}

/// @brief it should decode an encoded frame.
TEST(VideoFrameTest, testDecode) {
    ASSERT_EQ(ASSERT_NIL_P(VideoFrame::decode(KEY.encode())), KEY);
    ASSERT_EQ(ASSERT_NIL_P(VideoFrame::decode(DELTA.encode())), DELTA);
}

/// @brief it should decode a frame with an empty access unit.
TEST(VideoFrameTest, testDecodeEmptyAccessUnit) {
    const std::vector<std::uint8_t> sample{0, 1};
    ASSERT_TRUE(ASSERT_NIL_P(VideoFrame::decode(sample)).data.empty());
}

/// @brief it should reject a sample shorter than the header.
TEST(VideoFrameTest, testDecodeShortSample) {
    const std::vector<std::uint8_t> sample{1};
    const auto [frame, err] = VideoFrame::decode(sample);
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "video sample has 1 bytes, but the header needs 2");
}

/// @brief it should reject a sample with unknown flags.
TEST(VideoFrameTest, testDecodeUnknownFlags) {
    const std::vector<std::uint8_t> sample{3, 1};
    const auto [frame, err] = VideoFrame::decode(sample);
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "video sample has unknown flags 0x03");
}

/// @brief it should reject a sample with an unknown codec.
TEST(VideoFrameTest, testDecodeUnknownCodec) {
    const std::vector<std::uint8_t> sample{1, 9};
    const auto [frame, err] = VideoFrame::decode(sample);
    ASSERT_OCCURRED_AS(err, errors::VALIDATION);
    ASSERT_EQ(err.data, "video sample has unknown codec 9");
}
}
