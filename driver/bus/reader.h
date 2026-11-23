// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <chrono>
#include <cstddef>
#include <cstdint>
#include <functional>
#include <memory>
#include <optional>
#include <utility>
#include <vector>

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/telem.h"

#include "driver/codec/framing/framing.h"
#include "driver/common/common.h"
#include "driver/common/read_task.h"
#include "driver/icd/reader.h"
#include "driver/transport/transport.h"

namespace driver::bus {
/// @brief BATCH is the most frames that one read decodes.
constexpr std::size_t BATCH = 1024;

/// @brief READ_TIMEOUT is the longest that one read waits for bytes, and the quiet time
/// after which the framer drops a partial frame.
const x::telem::TimeSpan READ_TIMEOUT = 50 * x::telem::MILLISECOND;

/// @brief WRITE_TIMEOUT is the longest that sending a poll query waits.
const x::telem::TimeSpan WRITE_TIMEOUT = x::telem::SECOND;

/// @brief Clock returns the time on a steady clock, which schedules poll queries.
using Clock = std::function<std::chrono::steady_clock::time_point()>;

/// @brief Poll is a query that a reader sends at a fixed rate. The reader reads the
/// replies as any frame.
struct Poll {
    /// @brief query is the frame to send.
    std::vector<std::uint8_t> query;
    /// @brief rate is how often the reader sends the query.
    x::telem::Rate rate;
};

/// @brief Reader reads the frames of a byte stream into the channels of their ICD
/// messages. It is not safe for concurrent use.
class Reader {
    std::unique_ptr<codec::framing::Framer> framer;
    icd::Reader messages;
    /// @brief received is the time at which the bytes in the framer arrived.
    x::telem::TimeStamp received;
    /// @brief stamp is the arrival time given to the last frame.
    x::telem::TimeStamp stamp;
    /// @brief quiet is how long reads have waited with no bytes since the last bytes or
    /// the last drop of a partial frame.
    x::telem::TimeSpan quiet;
    /// @brief query holds the stream bytes of the poll query, and is empty with no
    /// poll.
    std::vector<std::uint8_t> query;
    x::telem::TimeSpan period;
    Clock now;
    /// @brief next is the time at which the reader sends the next query.
    std::chrono::steady_clock::time_point next;

    Reader() = default;
    Reader(
        std::unique_ptr<codec::framing::Framer> framer,
        icd::Reader messages,
        Clock now
    );

    /// @brief decodes up to BATCH frames from the framer into fr, and sets warning to
    /// the first framing or decoding error when it is empty.
    /// @returns the number of frames.
    std::size_t decode(x::telem::Frame &fr, common::Warning &warning);

    /// @brief drops the bytes in the framer after the error err of the transport.
    common::ReadResult fail(const x::errors::Error &err);

    /// @brief sends the query on t when it is due.
    /// @returns the longest that the next transport read can wait, and the error of t.
    std::pair<x::telem::TimeSpan, x::errors::Error> send(transport::Transport &t);

public:
    /// @brief creates a Reader.
    /// @param framer splits the stream into frames, and encodes the poll query.
    /// @param messages reads each frame into the channels of its message.
    /// @param poll is the query to send at its rate, or nothing.
    /// @param now is the steady clock that schedules the query.
    /// @returns CONFIGURATION_ERROR when the poll rate is not above 0 Hz, is above
    /// 1 GHz, or the framing cannot hold the poll query.
    static std::pair<Reader, x::errors::Error> create(
        std::unique_ptr<codec::framing::Framer> framer,
        icd::Reader messages,
        std::optional<Poll> poll = std::nullopt,
        Clock now = std::chrono::steady_clock::now
    );

    /// @brief sends the poll query on t when it is due, and then reads into fr up to
    /// BATCH frames: the frames left from the last read, or else the frames of the
    /// bytes that arrive on t within READ_TIMEOUT or before the next query. A frame
    /// takes the time its bytes arrived, or 1 ns after the last frame when that is
    /// later. After READ_TIMEOUT with no bytes, the framer drops a partial frame. A
    /// frame of no message adds nothing.
    /// @returns the error of t, after which the reader drops the bytes in the framer.
    /// Framing and decoding errors come back as warnings.
    common::ReadResult read(transport::Transport &t, x::telem::Frame &fr);
};
}
