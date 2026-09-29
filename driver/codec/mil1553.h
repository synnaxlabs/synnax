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
#include <span>
#include <vector>

#include "client/cpp/library/types.gen.h"
#include "x/cpp/errors/errors.h"

namespace driver::codec::mil1553 {
/// @brief the most data words one transfer carries.
constexpr std::size_t MAX_WORDS = 32;
/// @brief the largest remote terminal address. Address 31 is broadcast.
constexpr std::uint8_t MAX_RT = 30;
/// @brief the broadcast remote terminal address.
constexpr std::uint8_t BROADCAST = 31;
/// @brief the largest data subaddress. Subaddresses 0 and 31 carry mode codes.
constexpr std::uint8_t MAX_SUBADDRESS = 30;

/// @brief Command is a MIL-STD-1553 command word.
struct Command {
    /// @brief rt is the remote terminal address, from 0 to 31.
    std::uint8_t rt = 0;
    /// @brief transmit is true when the remote terminal transmits data to the bus
    /// controller, and false when it receives data.
    bool transmit = false;
    /// @brief subaddress is the subaddress, or 0 and 31 for a mode code.
    std::uint8_t subaddress = 0;
    /// @brief count is the number of data words from 1 to 32, or the mode code.
    std::uint8_t count = 1;

    /// @returns the command for a message identifier.
    static Command from(const synnax::library::Mil1553Identifier &id);

    /// @returns the command held in a 16-bit command word.
    static Command decode(std::uint16_t word);

    /// @returns the 16-bit command word. A count of 32 encodes as 0.
    [[nodiscard]] std::uint16_t encode() const;

    /// @returns true when the command carries a mode code.
    [[nodiscard]] bool mode_code() const {
        return this->subaddress == 0 || this->subaddress == 31;
    }

    bool operator==(const Command &) const = default;
};

/// @brief Status is a MIL-STD-1553 status word.
struct Status {
    /// @brief rt is the address of the remote terminal that sent the word.
    std::uint8_t rt = 0;
    /// @brief message_error is true when the terminal rejected the last message.
    bool message_error = false;
    /// @brief instrumentation is the instrumentation bit.
    bool instrumentation = false;
    /// @brief service_request is true when the terminal asks for service.
    bool service_request = false;
    /// @brief broadcast_received is true when the last command was a broadcast.
    bool broadcast_received = false;
    /// @brief busy is true when the terminal could not move data.
    bool busy = false;
    /// @brief subsystem_flag is true when the terminal's subsystem has a fault.
    bool subsystem_flag = false;
    /// @brief dynamic_bus_control is true when the terminal accepts bus control.
    bool dynamic_bus_control = false;
    /// @brief terminal_flag is true when the terminal has a fault.
    bool terminal_flag = false;

    /// @returns the status held in a 16-bit status word.
    static Status decode(std::uint16_t word);

    /// @returns the 16-bit status word, with the reserved bits zero.
    [[nodiscard]] std::uint16_t encode() const;

    bool operator==(const Status &) const = default;
};

/// @brief writes data words into a payload, each word big-endian, so that a
/// big-endian binary field at start bit 7 begins at the most significant bit of
/// the first word.
void to_payload(std::span<const std::uint16_t> words, std::vector<std::uint8_t> &out);

/// @brief reads data words from a payload written by to_payload. A trailing odd
/// byte is ignored.
/// @returns the number of words read.
std::size_t
from_payload(std::span<const std::uint8_t> payload, std::span<std::uint16_t> words);

/// @brief checks that a message can travel as a MIL-STD-1553 transfer: it has a
/// mil1553 identifier with an address from 0 to 30, a subaddress from 1 to 30, a
/// direction, and a word count from 1 to 32; a binary format; a length of twice the
/// word count when set; and fields inside the data words.
/// @returns LAYOUT_ERROR naming the message when it cannot.
x::errors::Error validate(const synnax::library::MessageEntry &message);
}
