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

/// @brief creates a framer that ends each frame with delimiter. It drops an empty frame
/// with no error.
/// @returns CONFIG_ERROR when delimiter is empty.
std::pair<std::unique_ptr<Framer>, x::errors::Error>
delimited(std::vector<std::uint8_t> delimiter);

/// @brief creates a framer whose frames each hold length bytes.
/// @returns CONFIG_ERROR when length is zero or larger than MAX_SIZE.
std::pair<std::unique_ptr<Framer>, x::errors::Error> fixed(std::size_t length);

/// @brief creates a framer that stuffs each frame with consistent overhead byte
/// stuffing (COBS) and ends it with a zero byte. It drops an empty frame with no error.
std::unique_ptr<Framer> cobs();

/// @brief creates a framer that escapes each frame with the serial line internet
/// protocol (SLIP, RFC 1055) and ends it with an END byte. It drops an empty frame with
/// no error.
std::unique_ptr<Framer> slip();

/// @brief ChecksumAlgorithm is a CRC that ends a sync frame.
enum class ChecksumAlgorithm : std::uint8_t {
    /// @brief CRC16_CCITT_FALSE has polynomial 0x1021 and initial value 0xFFFF, with no
    /// reflection.
    CRC16_CCITT_FALSE,
    /// @brief CRC16_MODBUS has polynomial 0x8005 and initial value 0xFFFF, reflected.
    CRC16_MODBUS,
    /// @brief CRC32 is the CRC of Ethernet and zlib.
    CRC32,
};

/// @brief Checksum is the CRC that ends a sync frame.
struct Checksum {
    /// @brief algorithm is the CRC that the checksum holds.
    ChecksumAlgorithm algorithm = ChecksumAlgorithm::CRC32;
    /// @brief big_endian is true when the checksum puts its most significant byte
    /// first.
    bool big_endian = false;
};

/// @brief SyncConfig describes frames that start with a sync sequence and a length
/// field.
struct SyncConfig {
    /// @brief sync is the byte sequence that starts each frame.
    std::vector<std::uint8_t> sync;
    /// @brief length_offset is the offset of the length field from the frame start.
    std::size_t length_offset = 0;
    /// @brief length_size is the number of bytes in the length field: 1, 2, or 4.
    std::size_t length_size = 1;
    /// @brief big_endian is true when the length field puts its most significant byte
    /// first.
    bool big_endian = false;
    /// @brief length_adjustment is added to the length field to give the number of
    /// bytes after the length field, checksum included.
    std::int64_t length_adjustment = 0;
    /// @brief checksum is the CRC that ends each frame and covers every byte before it,
    /// or nothing when frames have no checksum.
    std::optional<Checksum> checksum;
    /// @brief max_length is the most bytes in a frame, sync included. A length field
    /// that gives a larger frame marks a corrupt frame.
    std::size_t max_length = MAX_SIZE;
};

/// @brief creates a framer for frames that start with a sync sequence and a length
/// field. A frame that it reads starts at the sync sequence and has no checksum. It
/// finds the next sync sequence after bytes that are not a frame. Its encode writes the
/// sync sequence and the length into the header of the frame, and appends the checksum.
/// @returns CONFIG_ERROR when the sync sequence is empty, the length field is not 1, 2,
/// or 4 bytes or overlaps the sync sequence, or max_length is larger than MAX_SIZE or
/// too small for the header.
std::pair<std::unique_ptr<Framer>, x::errors::Error> sync(SyncConfig config);
}
