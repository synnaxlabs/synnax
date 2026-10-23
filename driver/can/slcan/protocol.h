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
#include <vector>

#include "x/cpp/errors/errors.h"

#include "driver/can/can.h"

/// @brief the slcan ASCII protocol that adapters speak over a serial port. A line ends
/// in a carriage return. Firmwares differ in which commands they answer, how they open
/// a listen only bus, and which bitrate S7 sets.
namespace driver::can::slcan {
/// @brief the command that takes the adapter off the bus.
constexpr std::string_view CLOSE_COMMAND = "C\r";
/// @brief the command that every firmware answers with a line that names it.
constexpr std::string_view VERSION_COMMAND = "V\r";
/// @brief the longest line the adapter sends: an extended CAN FD frame with 64 bytes
/// and a timestamp.
constexpr std::size_t MAX_LINE_LENGTH = 1 + 8 + 1 + 2 * MAX_FD_LENGTH + 4;

/// @brief the firmware an slcan adapter runs.
enum class Firmware : std::uint8_t {
    /// @brief LAWICEL CAN232 and CANUSB, and adapters that copy them. Every command
    /// gets a carriage return or a bell, L opens the bus listen only, and S7 is 800
    /// kbit/s.
    LAWICEL,
    /// @brief the stock CANable and CANable 2.0 firmware. Only V gets an answer, O
    /// always joins the bus in normal mode, and S7 is 750 kbit/s.
    CANABLE,
    /// @brief the CANable 2.5 firmware. Every command but C gets a carriage return or a
    /// bell, M1 makes the next O open the bus listen only, and S7 is 800 kbit/s.
    CANABLE_2_5,
};

/// @returns the firmware that sent version, its answer to VERSION_COMMAND without the
/// carriage return. An answer from neither LAWICEL nor CANable 2.5 firmware, such as
/// the git version that stock CANable firmware sends, means CANABLE.
[[nodiscard]] Firmware identify(std::string_view version);

/// @returns true when the firmware answers each command that configures the bus.
[[nodiscard]] bool answers(Firmware firmware);

/// @returns the command that sets the nominal bitrate, or CONFIG_ERROR when the
/// firmware has no command for the bitrate.
[[nodiscard]] std::pair<std::string, x::errors::Error>
bitrate_command(Firmware firmware, std::uint32_t bitrate);

/// @returns the command that sets the CAN FD data bitrate, or CONFIG_ERROR when the
/// protocol has no command for the bitrate.
[[nodiscard]] std::pair<std::string, x::errors::Error>
data_bitrate_command(std::uint32_t bitrate);

/// @returns the commands that put the adapter on the bus, or CONFIG_ERROR when the
/// firmware cannot open the bus listen only.
/// @param listen_only true to open the bus so the adapter never transmits or
/// acknowledges.
[[nodiscard]] std::pair<std::vector<std::string>, x::errors::Error>
open_commands(Firmware firmware, bool listen_only);

/// @returns the line that transmits the frame, including the carriage return.
/// @param frame a frame that validate accepts.
[[nodiscard]] std::string encode(const Frame &frame);

/// @brief parses a frame line without its carriage return. A trailing four digit
/// timestamp is accepted and dropped.
/// @returns FRAME_ERROR when the line is not a well formed frame.
[[nodiscard]] std::pair<Frame, x::errors::Error> decode(std::string_view line);

/// @brief the kind of a complete message from the adapter.
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

/// @brief a complete message from the adapter.
struct Message {
    Reply reply = Reply::OTHER;
    /// @brief the frame, when reply is FRAME.
    Frame frame;
    /// @brief the line without its carriage return, when reply is OTHER or MALFORMED. A
    /// line longer than MAX_LINE_LENGTH keeps only its start.
    std::string text;
};

/// @brief splits the byte stream from the adapter into messages.
class Decoder {
    std::string line;
    bool overflowed = false;

public:
    /// @brief adds one byte from the adapter.
    /// @returns the message that c completes, or nullopt when c does not complete one.
    std::optional<Message> push(char c);
};
}
