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
#include <map>
#include <memory>
#include <span>
#include <utility>
#include <vector>

#include "client/cpp/channel/types.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/telem.h"

#include "driver/codec/framing/framing.h"
#include "driver/common/common.h"
#include "driver/common/read_task.h"
#include "driver/icd/reader.h"
#include "driver/transport/connection.h"
#include "driver/transport/transport.h"

namespace driver::icd {
/// @brief BATCH is the most frames that one read decodes.
constexpr std::size_t BATCH = 1024;

/// @brief READ_TIMEOUT is the longest that one read waits for bytes, and the quiet time
/// after which the framer drops a partial frame.
const x::telem::TimeSpan READ_TIMEOUT = 50 * x::telem::MILLISECOND;

/// @brief WRITE_TIMEOUT is the longest that sending a poll query waits.
const x::telem::TimeSpan WRITE_TIMEOUT = x::telem::SECOND;

/// @brief Receiver reads the messages that the entries of a task receive on one
/// transport into their channels. Each frame goes to the first entry, in task order,
/// with a message that matches it, and later entries never see it. A frame that is not
/// a JSON document matches no message of a JSON entry. It is not safe for concurrent
/// use.
class Receiver {
    /// @brief Poll sends the query of one entry at its rate.
    struct Poll {
        /// @brief query holds the transport bytes of the query.
        std::vector<std::uint8_t> query;
        x::telem::TimeSpan period;
        /// @brief next is the time at which the receiver sends the next query.
        std::chrono::steady_clock::time_point next;
    };

    /// @brief framer splits the bytes into frames, and is null for datagrams with no
    /// framing, which each hold one message.
    std::unique_ptr<codec::framing::Framer> framer;
    bool datagram = false;
    /// @brief readers holds the reader of each entry, in task order.
    std::vector<Reader> readers;
    std::vector<Poll> polls;
    /// @brief received is the time at which the bytes in the framer arrived.
    x::telem::TimeStamp received;
    /// @brief stamp is the arrival time given to the last frame.
    x::telem::TimeStamp stamp;
    /// @brief quiet is how long reads have waited with no bytes since the last bytes or
    /// the last drop of a partial frame.
    x::telem::TimeSpan quiet;
    transport::Clock now;

    /// @brief reads payload with the first reader that has a message of it, and sets
    /// warning to its decoding error when it is empty. With no such reader, it sets
    /// warning to the first JSON error.
    void dispatch(
        std::span<const std::uint8_t> payload,
        x::telem::Frame &fr,
        common::Warning &warning
    );

    /// @brief decodes the frames of the framer into fr until n reaches BATCH, and sets
    /// warning to the first framing or decoding error when it is empty.
    /// @returns n plus the number of frames.
    std::size_t drain(std::size_t n, x::telem::Frame &fr, common::Warning &warning);

    /// @brief decodes up to BATCH frames from the framer into fr, and sets warning to
    /// the first framing or decoding error when it is empty. At the end of a datagram,
    /// the framer drops each partial frame and reads the frames after it, and then the
    /// receiver drops the bytes left with a warning.
    /// @returns the number of frames.
    std::size_t decode(x::telem::Frame &fr, common::Warning &warning);

    /// @brief drops the bytes in the framer after the error err of the transport.
    common::ReadResult fail(const x::errors::Error &err);

    /// @brief sends each query on t that is due.
    /// @returns the longest that the next transport read can wait, and the error of t.
    std::pair<x::telem::TimeSpan, x::errors::Error> send(transport::Transport &t);

public:
    /// @brief creates a Receiver.
    /// @param entries the entries to read, in task order.
    /// @param icds the ICDs of entries, which must share one framing.
    /// @param channels the channels of the bindings of entries.
    /// @param datagram true when each transport read gives one datagram. A datagram
    /// with no framing holds one message.
    /// @param now is the steady clock that schedules the queries.
    /// @returns CONFIGURATION_ERROR when icds has no ICD that an entry names, a stream
    /// ICD has no framing, two ICDs frame the bytes differently, two entries read one
    /// message of one ICD, a poll rate is not above 0 Hz or is above 1 GHz, the framing
    /// cannot hold a poll query, or the Reader of an entry fails to create.
    static std::pair<Receiver, x::errors::Error> create(
        const std::vector<synnax::icd::ReadEntry> &entries,
        const std::vector<synnax::icd::ICD> &icds,
        const std::map<synnax::channel::Key, synnax::channel::Channel> &channels,
        bool datagram,
        transport::Clock now = std::chrono::steady_clock::now
    );

    /// @brief sends each poll query on t that is due, and then reads into fr up to
    /// BATCH frames: the frames left from the last read, or else the frames of the
    /// bytes that arrive on t within READ_TIMEOUT or before the next query. A frame
    /// takes the time its bytes arrived, or 1 ns after the last frame when that is
    /// later. After READ_TIMEOUT with no bytes, the framer drops a partial frame of a
    /// stream. A frame of no message adds nothing.
    /// @returns the error of t, after which the receiver drops the bytes in the framer.
    /// Framing and decoding errors come back as warnings.
    common::ReadResult read(transport::Transport &t, x::telem::Frame &fr);
};
}
