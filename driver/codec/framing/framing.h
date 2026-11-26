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
#include <cstdint>
#include <memory>
#include <optional>
#include <span>
#include <utility>
#include <vector>

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"

#include "driver/errors/errors.h"

namespace driver::codec::framing {
/// @brief a framing that cannot split a stream.
const x::errors::Error CONFIG_ERROR = driver::errors::CONFIGURATION_ERROR.sub(
    "framing"
);

/// @brief MAX_SIZE is the largest frame in bytes.
constexpr std::size_t MAX_SIZE = std::size_t{64} * 1024;

/// @brief Framer splits a byte stream into frames, and puts frames on the stream. A
/// frame holds no delimiter, stuffing, or checksum. It is not safe for concurrent use.
class Framer {
public:
    virtual ~Framer() = default;

    /// @brief adds a chunk of the stream, which arrives in pieces of any size.
    virtual void write(std::span<const std::uint8_t> chunk) = 0;

    /// @brief returns the next complete frame in the written bytes, or nothing when
    /// there is none. The frame is valid until the next call to write, idle, or clear.
    /// @returns VALIDATION when the framer drops bytes that cannot be a frame, such as
    /// more than MAX_SIZE bytes with no delimiter, or a frame with invalid stuffing.
    /// Call next again to read the frames after them.
    virtual std::pair<std::optional<std::span<const std::uint8_t>>, x::errors::Error>
    next() = 0;

    /// @brief tells the framer that the stream went quiet. A fixed or sync framer drops
    /// a partial frame here, so that the next frame starts in step. A delimited, COBS,
    /// or SLIP framer keeps it until its delimiter. Call next again to read the frames
    /// after the dropped bytes.
    /// @returns VALIDATION when it drops bytes.
    virtual x::errors::Error idle() = 0;

    /// @brief drops all written bytes, as after a reconnect.
    virtual void clear() = 0;

    /// @brief appends the stream bytes of frame to out. Writing them to a framer of the
    /// same framing gives frame again, with any sync header filled in.
    /// @returns VALIDATION when the framing cannot hold frame: it is empty, larger than
    /// MAX_SIZE, holds a delimiter, is not the fixed length, or does not fit a sync
    /// header and length.
    virtual x::errors::Error encode(
        std::span<const std::uint8_t> frame,
        std::vector<std::uint8_t> &out
    ) const = 0;
};

/// @brief creates the framer of framing. A delimited, COBS, or SLIP framer drops an
/// empty frame with no error. A sync framer reads a frame from its sync sequence, with
/// no checksum, and its encode fills in the sync sequence and the length and appends
/// the checksum.
/// @returns CONFIG_ERROR when the sync length field overlaps the sync sequence, the
/// sync max_length is less than the bytes up to the end of the length field, or framing
/// holds an unknown length size, byte order, or checksum algorithm.
std::pair<std::unique_ptr<Framer>, x::errors::Error>
create(const synnax::icd::Framing &framing);
}
