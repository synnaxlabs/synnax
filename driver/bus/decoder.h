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
#include <span>
#include <string>
#include <vector>

#include "client/cpp/framer/framer.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/telem.h"

#include "driver/bus/config.h"

namespace driver::bus {
/// @brief how long a warning stays on the task status after its cause stops.
const auto WARNING_HOLD = 1 * x::telem::SECOND;

/// @brief collects the decoded samples, raw frames, and warnings of a read task between
/// reads, and moves them into a frame for the task's channels.
class Decoder {
public:
    /// @param cfg the task's config. It must outlive the decoder.
    explicit Decoder(const ReadConfig &cfg);

    /// @returns the config of a writer for every channel the task writes.
    [[nodiscard]] synnax::framer::WriterConfig writer_config() const;

    /// @returns the data channels of every message.
    [[nodiscard]] std::vector<synnax::channel::Channel> channels() const;

    /// @brief records a received frame for the raw channel, when the task has one.
    void raw(std::span<const std::uint8_t> frame);

    /// @brief decodes payload as the given message, received at time. A payload that
    /// does not decode is dropped with a warning. A sample missing a multiplexed field
    /// is dropped, so every channel on an index keeps the same length.
    void decode(
        std::size_t message,
        std::span<const std::uint8_t> payload,
        x::telem::TimeStamp time
    );

    /// @brief adds a warning to the next call to warning, unless it is already there.
    void warn(std::string message);

    /// @brief moves every collected sample and raw frame into fr. Index times strictly
    /// increase.
    void flush(x::telem::Frame &fr);

    /// @returns the warnings since the last call, joined, or the last warning while it
    /// is younger than WARNING_HOLD. Empty otherwise.
    std::string warning(x::telem::TimeStamp now);

    /// @brief forgets the held warning.
    void clear_warning();

private:
    /// @brief decoded samples of one message waiting to go into a frame.
    struct Pending {
        /// @brief one decoded sample per arrival, reused across reads.
        std::vector<codec::Values> values;
        /// @brief the arrival time of each sample.
        std::vector<x::telem::TimeStamp> times;
        /// @brief the last time stamped on the message's index.
        x::telem::TimeStamp last;
    };

    const ReadConfig &cfg;
    std::vector<Pending> pending;
    std::vector<std::string> raws;
    std::vector<std::string> warnings;
    std::string held_warning;
    x::telem::TimeStamp held_at;
};
}
