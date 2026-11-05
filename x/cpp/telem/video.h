// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <format>
#include <span>
#include <utility>
#include <vector>

#include "x/cpp/errors/errors.h"

namespace x::telem {
/// @brief the encoding of the access unit in a VideoFrame.
enum class VideoCodec : std::uint8_t {
    /// @brief H.264 in Annex B format, with no B-frames.
    H264 = 1,
};

/// @brief one encoded picture. It is a single sample of a video series.
struct VideoFrame {
    /// @brief true when the frame decodes without the frames before it. A keyframe
    /// carries its codec parameter sets.
    bool keyframe = false;
    /// @brief the encoding of data.
    VideoCodec codec = VideoCodec::H264;
    /// @brief one access unit.
    std::vector<std::uint8_t> data;

    /// @returns the frame as a sample of a video series.
    [[nodiscard]] std::vector<std::uint8_t> encode() const {
        std::vector<std::uint8_t> sample;
        sample.reserve(HEADER_SIZE + this->data.size());
        sample.push_back(this->keyframe ? KEYFRAME_FLAG : 0);
        sample.push_back(static_cast<std::uint8_t>(this->codec));
        sample.insert(sample.end(), this->data.begin(), this->data.end());
        return sample;
    }

    /// @returns the frame that a sample of a video series holds, or
    /// errors::VALIDATION when the header is malformed or names an unknown codec.
    static std::pair<VideoFrame, errors::Error>
    decode(const std::span<const std::uint8_t> sample) {
        if (sample.size() < HEADER_SIZE)
            return {
                {},
                errors::Error(
                    errors::VALIDATION,
                    std::format(
                        "video sample has {} bytes, but the header needs {}",
                        sample.size(),
                        HEADER_SIZE
                    )
                ),
            };
        const auto flags = sample[0];
        if ((flags & ~KEYFRAME_FLAG) != 0)
            return {
                {},
                errors::Error(
                    errors::VALIDATION,
                    std::format("video sample has unknown flags {:#04x}", flags)
                ),
            };
        if (sample[1] != static_cast<std::uint8_t>(VideoCodec::H264))
            return {
                {},
                errors::Error(
                    errors::VALIDATION,
                    std::format("video sample has unknown codec {}", sample[1])
                ),
            };
        return {
            VideoFrame{
                .keyframe = flags == KEYFRAME_FLAG,
                .codec = static_cast<VideoCodec>(sample[1]),
                .data = {sample.begin() + HEADER_SIZE, sample.end()},
            },
            errors::NIL,
        };
    }

    bool operator==(const VideoFrame &other) const = default;

private:
    static constexpr std::size_t HEADER_SIZE = 2;
    static constexpr std::uint8_t KEYFRAME_FLAG = 1;
};
}
