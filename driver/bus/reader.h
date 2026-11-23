// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstddef>
#include <memory>

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

/// @brief READ_TIMEOUT is the longest that one read waits for bytes.
const x::telem::TimeSpan READ_TIMEOUT = 50 * x::telem::MILLISECOND;

/// @brief Reader reads the frames of a byte stream into the channels of their ICD
/// messages. It is not safe for concurrent use.
class Reader {
    std::unique_ptr<codec::framing::Framer> framer;
    icd::Reader messages;
    /// @brief received is the time at which the bytes in the framer arrived.
    x::telem::TimeStamp received;
    /// @brief stamp is the arrival time given to the last frame.
    x::telem::TimeStamp stamp;

    /// @brief decodes up to BATCH frames from the framer into fr, and sets warning to
    /// the first framing or decoding error when it is empty.
    /// @returns the number of frames.
    std::size_t decode(x::telem::Frame &fr, common::Warning &warning);

public:
    /// @param framer splits the stream into frames.
    /// @param messages reads each frame into the channels of its message.
    Reader(std::unique_ptr<codec::framing::Framer> framer, icd::Reader messages);

    /// @brief reads into fr up to BATCH frames: the frames left from the last read, or
    /// else the frames of the bytes that arrive on t within READ_TIMEOUT. A frame takes
    /// the time its bytes arrived, or 1 ns after the last frame when that is later.
    /// After READ_TIMEOUT with no bytes, the framer drops a partial frame. A frame of
    /// no message adds nothing.
    /// @returns the error of t, after which the reader drops the bytes in the framer.
    /// Framing and decoding errors come back as warnings.
    common::ReadResult read(transport::Transport &t, x::telem::Frame &fr);
};
}
