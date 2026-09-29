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

#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/telem.h"

#include "driver/bus/config.h"
#include "driver/bus/read.h"
#include "driver/codec/plan.h"

namespace driver::bus {
/// @brief decodes the payloads of a read config's messages into samples and moves
/// them into Synnax frames. It also collects the warnings a read reports. Not safe
/// for concurrent use.
class Decoder {
public:
    /// @param cfg the resolved read config. It must outlive the decoder.
    explicit Decoder(const ReadConfig &cfg);

    Decoder(const Decoder &) = delete;
    Decoder &operator=(const Decoder &) = delete;

    /// @returns the config of the writer of every index, field, and raw channel.
    [[nodiscard]] synnax::framer::WriterConfig writer_config() const;

    /// @returns the channel of every field.
    [[nodiscard]] std::vector<synnax::channel::Channel> channels() const;

    /// @brief decodes payload as the message at index message of the config, stamped
    /// at time or just after the message's last sample. A payload that fails to
    /// decode becomes a warning, and one that leaves a field out is dropped, so every
    /// channel on an index keeps the same length.
    void decode(
        std::size_t message,
        std::span<const std::uint8_t> payload,
        x::telem::TimeStamp time
    );

    /// @brief adds frame to the raw channel, when the config has one.
    void raw(std::span<const std::uint8_t> frame);

    /// @brief adds a warning to the next call of warning.
    void warn(std::string warning);

    /// @brief moves every decoded sample and raw frame into fr.
    void flush(x::telem::Frame &fr);

    /// @returns the warnings added since the last call, joined, or the last warnings
    /// for WARNING_HOLD after no new ones arrive.
    std::string warning(x::telem::TimeStamp now);

    /// @brief discards the held warning, for a restarted task.
    void reset();

private:
    /// @brief decoded samples of one message waiting to go into a frame.
    struct Pending {
        /// @brief one decoded sample per arrival, reused across flushes.
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
    std::string held;
    x::telem::TimeStamp held_at;
};
}
