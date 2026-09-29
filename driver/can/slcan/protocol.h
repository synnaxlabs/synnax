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
#include <optional>
#include <string>
#include <string_view>
#include <utility>

#include "x/cpp/errors/errors.h"

#include "driver/can/can.h"

/// @brief the Lawicel ASCII protocol that slcan adapters speak over a serial port. A
/// line ends in a carriage return. The adapter answers each command with an empty line
/// on success or a BEL character on failure.
namespace driver::can::slcan {
/// @brief the command that takes the adapter off the bus.
constexpr std::string_view CLOSE_COMMAND = "C\r";
/// @brief the longest line the adapter sends: an extended CAN FD frame with 64 bytes
/// and a timestamp.
constexpr std::size_t MAX_LINE_LENGTH = 1 + 8 + 1 + 2 * MAX_FD_LENGTH + 4;

/// @returns the command that sets the nominal bitrate, or CONFIG_ERROR when the
/// protocol has no command for the bitrate.
[[nodiscard]] std::pair<std::string, x::errors::Error>
bitrate_command(std::uint32_t bitrate);

/// @returns the command that sets the CAN FD data bitrate, or CONFIG_ERROR when the
/// protocol has no command for the bitrate.
[[nodiscard]] std::pair<std::string, x::errors::Error>
data_bitrate_command(std::uint32_t bitrate);

/// @returns the command that puts the adapter on the bus.
/// @param listen_only true to open the adapter so it never transmits or acknowledges.
[[nodiscard]] std::string open_command(bool listen_only);

/// @returns the line that transmits the frame, including the carriage return.
/// @param frame a frame that validate accepts.
[[nodiscard]] std::string encode(const Frame &frame);

/// @brief parses a frame line without its carriage return. A trailing four digit
/// timestamp is accepted and dropped.
/// @returns FRAME_ERROR when the line is not a well formed frame.
[[nodiscard]] std::pair<Frame, x::errors::Error> decode(std::string_view line);

/// @brief a complete message from the adapter.
enum class Reply : std::uint8_t {
    /// @brief the adapter accepted a command.
    ACCEPTED,
    /// @brief the adapter rejected a command.
    REJECTED,
    /// @brief the adapter received a frame.
    FRAME,
    /// @brief a line that is not a frame, such as a transmit acknowledgment or a
    /// version.
    OTHER,
    /// @brief a frame line that decode rejects, or a line longer than MAX_LINE_LENGTH.
    MALFORMED,
};

/// @brief splits the byte stream from the adapter into replies.
class Decoder {
    std::string line;
    bool overflowed = false;

public:
    /// @brief adds one byte from the adapter.
    /// @param c the byte.
    /// @param frame receives the frame when the reply is FRAME.
    /// @returns the reply that c completes, or nullopt when c does not complete one.
    std::optional<Reply> push(char c, Frame &frame);
};
}
