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
#include <optional>
#include <span>
#include <string>
#include <vector>

#include "client/cpp/framer/framer.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/mono_clock.h"
#include "x/cpp/telem/telem.h"

#include "driver/bus/clock.h"
#include "driver/bus/config.h"
#include "driver/bus/warnings.h"
#include "driver/common/read_task.h"

namespace driver::bus {
/// @brief how long a read task waits after start, or after its last matching frame,
/// before it warns that nothing arrives or that nothing matches.
const auto SILENCE = 10 * x::telem::SECOND;
/// @brief the number of periods a message with a known period may miss before its task
/// warns.
constexpr int MISSED_PERIODS = 3;

/// @returns the warning a read task sends for problems, joined by "; ", or an empty
/// warning when problems is empty.
common::Warning read_warning(const std::vector<std::string> &problems);

/// @brief collects the decoded samples, raw frames, and warnings of a read task between
/// reads, and moves them into a frame for the task's channels.
class Decoder {
public:
    /// @param cfg the task's config. It must outlive the decoder.
    /// @param now the steady clock that times the warnings.
    explicit Decoder(const ReadConfig &cfg, x::telem::NowFunc now = steady_now);

    /// @returns the config of a writer for every channel the task writes.
    [[nodiscard]] synnax::framer::WriterConfig writer_config() const;

    /// @returns the data channels of every message.
    [[nodiscard]] std::vector<synnax::channel::Channel> channels() const;

    /// @brief forgets every warning and restarts the clocks of the warnings about
    /// silence. Call it when the task starts.
    void reset();

    /// @brief restarts the clocks of the warnings about silence and missed periods.
    /// Call it while the device is unreachable and when it reconnects, as the reconnect
    /// warning already explains the silence.
    void restart();

    /// @brief records a received frame, matched or not: on the raw channel when the
    /// task has one, and for the warnings about silence and unmatched frames.
    void raw(std::span<const std::uint8_t> frame);

    /// @brief decodes payload as the given message, received at time. A payload that
    /// does not decode is dropped with a warning. Each group whose fields are present
    /// gets one sample, stamped on its index.
    void decode(
        std::size_t message,
        std::span<const std::uint8_t> payload,
        x::telem::TimeStamp time
    );

    /// @brief records a reply to the query of message that arrived after its timeout,
    /// which the task discards, and warns about it.
    void late(std::size_t message);

    /// @brief adds a warning that holds for WARNING_HOLD after its last call.
    void warn(const std::string &message);

    /// @brief moves every collected sample and raw frame into fr. The times on each
    /// index strictly increase.
    void flush(x::telem::Frame &fr);

    /// @returns the warnings that hold now as one read_warning, or an empty warning.
    /// They are, in order: nothing received since start, frames that match no message,
    /// messages that missed MISSED_PERIODS periods, and the warnings from warn.
    common::Warning warning();

private:
    /// @brief the decoded samples of one group waiting to go into a frame.
    struct PendingGroup {
        /// @brief the index into Pending::values of each sample.
        std::vector<std::size_t> samples;
        /// @brief the time stamped on each sample.
        std::vector<x::telem::TimeStamp> times;
        /// @brief the last time stamped on the group's index.
        x::telem::TimeStamp last = x::telem::TimeStamp::min();
    };

    /// @brief the decoded samples of one message waiting to go into a frame.
    struct Pending {
        /// @brief one decoded payload per arrival, reused across reads.
        std::vector<codec::Values> values;
        /// @brief the number of values in use.
        std::size_t used = 0;
        /// @brief one entry per group of the message.
        std::vector<PendingGroup> groups;
        /// @brief the steady time the message last arrived.
        x::telem::TimeStamp arrived = x::telem::TimeStamp(0);
    };

    const ReadConfig &cfg;
    x::telem::NowFunc now;
    std::vector<Pending> pending;
    std::vector<std::string> raws;
    Warnings warnings;
    /// @brief the steady time of the last reset.
    x::telem::TimeStamp started = x::telem::TimeStamp(0);
    /// @brief the steady time of the last received frame, if any arrived since start.
    std::optional<x::telem::TimeStamp> received;
    /// @brief the steady time of the last frame that matched a message.
    x::telem::TimeStamp matched = x::telem::TimeStamp(0);
};
}
