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

#include "x/cpp/errors/errors.h"

#include "driver/errors/errors.h"

namespace driver::codec::framing {
/// @brief a framing that cannot split a stream.
const x::errors::Error CONFIG_ERROR = driver::errors::CONFIGURATION_ERROR.sub(
    "framing"
);

/// @brief MAX_SIZE is the largest frame in bytes, without what the framing adds.
constexpr std::size_t MAX_SIZE = std::size_t{64} * 1024;

/// @brief Framer splits a byte stream into frames, and puts frames on the stream. A
/// frame holds no bytes that the framing adds, such as a delimiter. It is not safe for
/// concurrent use.
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

    /// @brief tells the framer that the stream went quiet. A framing with no boundary
    /// of its own drops a partial frame here, so that the next frame starts in step.
    /// @returns VALIDATION when it drops bytes.
    virtual x::errors::Error idle() = 0;

    /// @brief drops all written bytes, as after a reconnect.
    virtual void clear() = 0;

    /// @brief appends the stream bytes of frame to out. Writing them to a framer of the
    /// same framing gives frame again.
    /// @returns VALIDATION when the framing cannot hold frame: it is empty, larger than
    /// MAX_SIZE, holds a delimiter, or is not the fixed length.
    virtual x::errors::Error encode(
        std::span<const std::uint8_t> frame,
        std::vector<std::uint8_t> &out
    ) const = 0;
};

/// @brief creates a framer that ends each frame with delimiter. It drops an empty frame
/// with no error.
/// @returns CONFIG_ERROR when delimiter is empty.
std::pair<std::unique_ptr<Framer>, x::errors::Error>
delimited(std::vector<std::uint8_t> delimiter);

/// @brief creates a framer whose frames each hold length bytes.
/// @returns CONFIG_ERROR when length is zero or larger than MAX_SIZE.
std::pair<std::unique_ptr<Framer>, x::errors::Error> fixed(std::size_t length);

/// @brief creates a framer that stuffs each frame with consistent overhead byte
/// stuffing (COBS) and ends it with a zero byte.
std::unique_ptr<Framer> cobs();

/// @brief creates a framer that escapes each frame with the serial line internet
/// protocol (SLIP, RFC 1055) and ends it with an END byte.
std::unique_ptr<Framer> slip();
}
