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
#include <functional>
#include <memory>
#include <span>
#include <utility>
#include <vector>

#include "client/cpp/bus/types.gen.h"
#include "x/cpp/errors/errors.h"

#include "driver/errors/errors.h"

namespace driver::codec::framing {
/// @brief base error for framing failures.
const x::errors::Error BASE_ERROR = driver::errors::BASE_ERROR.sub("framing");
/// @brief a frame that grew past MAX_SIZE on the read path.
const x::errors::Error OVERFLOW_ERROR = BASE_ERROR.sub("overflow");
/// @brief a payload the framing cannot put on the wire.
const x::errors::Error ENCODE_ERROR = BASE_ERROR.sub("encode");
/// @brief a framing configuration that is invalid.
const x::errors::Error CONFIG_ERROR = driver::errors::CONFIGURATION_ERROR.sub(
    "framing"
);

/// @brief MAX_SIZE bounds a frame in bytes, including its delimiter or checksum.
constexpr std::size_t MAX_SIZE = 64 * 1024;

/// @brief Handler receives one complete, non-empty frame. The frame is valid only
/// during the call, and the handler must not call back into the framer.
using Handler = std::function<void(std::span<const std::uint8_t> frame)>;

/// @brief Framer splits a byte stream into frames and puts frames on the wire. A frame
/// excludes what the framing adds: a delimiter, byte stuffing, or a checksum. A sync
/// frame keeps its sync sequence and length field, so message layouts can hold header
/// fields.
class Framer {
public:
    virtual ~Framer() = default;

    /// @brief consumes a chunk of a stream that arrives in arbitrary pieces, and calls
    /// handler for each frame it completes. It resynchronizes after garbage or a
    /// corrupt frame. Not safe for concurrent use.
    /// @returns OVERFLOW_ERROR when a frame grew past MAX_SIZE. The framer drops that
    /// frame, resynchronizes, and still handles the frames after it.
    virtual x::errors::Error
    write(std::span<const std::uint8_t> chunk, const Handler &handler) = 0;

    /// @returns the number of framing errors so far: dropped garbage, corrupt frames,
    /// failed checksums, and overflows.
    [[nodiscard]] virtual std::size_t errors() const = 0;

    /// @brief drops any partial frame, as after a reconnect.
    virtual void reset() = 0;

    /// @brief appends the wire bytes of one frame to out. Feeding them to write()
    /// yields the frame again. For sync framing, the frame starts at the sync sequence
    /// and ends before the checksum, and encode overwrites its sync and length bytes.
    /// Safe to call from any thread, even during write().
    /// @returns ENCODE_ERROR when the frame does not fit the framing: a delimiter
    /// inside it, a size other than the fixed length, a length the length field cannot
    /// hold, or a wire size over MAX_SIZE.
    virtual x::errors::Error encode(
        std::span<const std::uint8_t> frame,
        std::vector<std::uint8_t> &out
    ) const = 0;
};

/// @brief creates the framer that a task's framing selects.
/// @returns CONFIG_ERROR when the framing is invalid, such as a sync sequence that is
/// not hex or a length size other than 1, 2, or 4.
std::pair<std::unique_ptr<Framer>, x::errors::Error>
create(const synnax::bus::Framing &framing);
}
