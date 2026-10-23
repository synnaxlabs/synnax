// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <array>
#include <format>
#include <stdexcept>

#include "driver/can/slcan/protocol.h"

namespace driver::can::slcan {
namespace {
/// @brief a bitrate and the command that sets it.
using Command = std::pair<std::uint32_t, std::string_view>;

// Section 2.0 of the LAWICEL CAN232 manual.
constexpr std::array<Command, 9> LAWICEL_BITRATES = {{
    {10000, "S0\r"},
    {20000, "S1\r"},
    {50000, "S2\r"},
    {100000, "S3\r"},
    {125000, "S4\r"},
    {250000, "S5\r"},
    {500000, "S6\r"},
    {800000, "S7\r"},
    {1000000, "S8\r"},
}};

// enum can_bitrate in canable2-fw inc/can.h. canable-fw, for the first CANable, lacks
// S9 and ignores it without an answer.
constexpr std::array<Command, 10> CANABLE_BITRATES = {{
    {10000, "S0\r"},
    {20000, "S1\r"},
    {50000, "S2\r"},
    {100000, "S3\r"},
    {125000, "S4\r"},
    {250000, "S5\r"},
    {500000, "S6\r"},
    {750000, "S7\r"},
    {1000000, "S8\r"},
    {83333, "S9\r"},
}};

// The S commands in the CANable 2.5 firmware manual.
constexpr std::array<Command, 10> CANABLE_2_5_BITRATES = {{
    {10000, "S0\r"},
    {20000, "S1\r"},
    {50000, "S2\r"},
    {100000, "S3\r"},
    {125000, "S4\r"},
    {250000, "S5\r"},
    {500000, "S6\r"},
    {800000, "S7\r"},
    {1000000, "S8\r"},
    {83333, "S9\r"},
}};

constexpr std::array<Command, 2> DATA_BITRATES = {{
    {2000000, "Y2\r"},
    {5000000, "Y5\r"},
}};

std::optional<std::uint8_t> hex_digit(const char c) {
    if (c >= '0' && c <= '9') return c - '0';
    if (c >= 'A' && c <= 'F') return c - 'A' + 10;
    if (c >= 'a' && c <= 'f') return c - 'a' + 10;
    return std::nullopt;
}

std::optional<std::uint32_t> hex(const std::string_view digits) {
    std::uint32_t value = 0;
    for (const char c: digits) {
        const auto digit = hex_digit(c);
        if (!digit.has_value()) return std::nullopt;
        value = value << 4 | *digit;
    }
    return value;
}

std::string name(const Firmware firmware) {
    switch (firmware) {
        case Firmware::LAWICEL:
            return "LAWICEL slcan firmware";
        case Firmware::CANABLE:
            return "CANable firmware";
        case Firmware::CANABLE_2_5:
            return "CANable 2.5 firmware";
    }
    throw std::invalid_argument("unknown slcan firmware");
}

x::errors::Error malformed(const std::string_view line, const std::string &reason) {
    return {
        FRAME_ERROR,
        "malformed slcan frame '" + std::string(line) + "': " + reason
    };
}
}

Firmware identify(const std::string_view version) {
    // CANable 2.5 answers start with '+', and a LAWICEL version is V and four digits.
    if (version.starts_with('+')) return Firmware::CANABLE_2_5;
    if (version.size() == 5 && version.starts_with('V')) return Firmware::LAWICEL;
    return Firmware::CANABLE;
}

bool answers(const Firmware firmware) {
    return firmware != Firmware::CANABLE;
}

std::pair<std::string, x::errors::Error>
bitrate_command(const Firmware firmware, const std::uint32_t bitrate) {
    const auto what = name(firmware) + " cannot run a classic CAN bus";
    std::pair<std::string_view, x::errors::Error> found;
    switch (firmware) {
        case Firmware::LAWICEL:
            found = find_bitrate(LAWICEL_BITRATES, bitrate, what);
            break;
        case Firmware::CANABLE:
            found = find_bitrate(CANABLE_BITRATES, bitrate, what);
            break;
        case Firmware::CANABLE_2_5:
            found = find_bitrate(CANABLE_2_5_BITRATES, bitrate, what);
            break;
    }
    return {std::string(found.first), found.second};
}

std::pair<std::string, x::errors::Error>
data_bitrate_command(const std::uint32_t bitrate) {
    auto [command, err] = find_bitrate(
        DATA_BITRATES,
        bitrate,
        "slcan cannot run a CAN FD data phase"
    );
    return {std::string(command), err};
}

std::pair<std::vector<std::string>, x::errors::Error>
open_commands(const Firmware firmware, const bool listen_only) {
    if (!listen_only) return {{"O\r"}, x::errors::NIL};
    switch (firmware) {
        case Firmware::LAWICEL:
            return {{"L\r"}, x::errors::NIL};
        case Firmware::CANABLE_2_5:
            return {{"M1\r", "O\r"}, x::errors::NIL};
        case Firmware::CANABLE:
            // can_enable in src/can.c sets the normal mode, which discards M1.
            return {
                {},
                {CONFIG_ERROR,
                 "CANable firmware cannot open the bus listen only. Flash the CANable "
                 "2.5 or candleLight firmware to listen only"}
            };
    }
    throw std::invalid_argument("unknown slcan firmware");
}

std::string encode(const Frame &frame) {
    char kind = 't';
    if (frame.type == Type::REMOTE)
        kind = 'r';
    else if (frame.fd)
        kind = frame.bitrate_switched ? 'b' : 'd';
    if (frame.extended) kind = static_cast<char>(kind - 'a' + 'A');
    std::string line(1, kind);
    line += frame.extended ? std::format("{:08X}", frame.id)
                           : std::format("{:03X}", frame.id);
    line += std::format("{:X}", *length_to_dlc(frame.length));
    if (frame.type != Type::REMOTE)
        for (const auto byte: frame.payload())
            line += std::format("{:02X}", byte);
    line += '\r';
    return line;
}

std::pair<Frame, x::errors::Error> decode(const std::string_view line) {
    Frame frame;
    if (line.empty()) return {frame, malformed(line, "the line is empty")};
    const char kind = line[0];
    switch (kind) {
        case 't':
        case 'T':
            break;
        case 'r':
        case 'R':
            frame.type = Type::REMOTE;
            break;
        case 'b':
        case 'B':
            frame.bitrate_switched = true;
            frame.fd = true;
            break;
        case 'd':
        case 'D':
            frame.fd = true;
            break;
        default:
            return {frame, malformed(line, "unknown frame type")};
    }
    frame.extended = kind >= 'A' && kind <= 'Z';
    const std::size_t id_digits = frame.extended ? 8 : 3;
    if (line.size() < 1 + id_digits + 1)
        return {frame, malformed(line, "the line is too short")};
    const auto id = hex(line.substr(1, id_digits));
    if (!id.has_value()) return {frame, malformed(line, "the identifier is not hex")};
    frame.id = *id;
    if (frame.id > (frame.extended ? MAX_EXTENDED_ID : MAX_STANDARD_ID))
        return {frame, malformed(line, "the identifier is out of range")};
    const auto dlc = hex_digit(line[1 + id_digits]);
    if (!dlc.has_value() || (!frame.fd && *dlc > MAX_CLASSIC_LENGTH))
        return {frame, malformed(line, "the length code is invalid")};
    frame.length = dlc_to_length(*dlc, frame.fd);
    auto rest = line.substr(2 + id_digits);
    if (frame.type != Type::REMOTE) {
        const std::size_t data_digits = 2 * frame.length;
        if (rest.size() < data_digits)
            return {frame, malformed(line, "the data is shorter than its length")};
        for (std::size_t i = 0; i < frame.length; i++) {
            const auto byte = hex(rest.substr(2 * i, 2));
            if (!byte.has_value())
                return {frame, malformed(line, "the data is not hex")};
            frame.data[i] = static_cast<std::uint8_t>(*byte);
        }
        rest = rest.substr(data_digits);
    }
    if (!rest.empty() && (rest.size() != 4 || !hex(rest).has_value()))
        return {frame, malformed(line, "unexpected characters after the data")};
    return {frame, x::errors::NIL};
}

std::optional<Message> Decoder::push(const char c) {
    if (c == '\a') {
        this->line.clear();
        this->overflowed = false;
        return Message{.reply = Reply::REJECTED};
    }
    if (c != '\r') {
        if (this->line.size() < MAX_LINE_LENGTH)
            this->line += c;
        else
            this->overflowed = true;
        return std::nullopt;
    }
    Message msg{.text = std::move(this->line)};
    const bool overflowed = this->overflowed;
    this->line.clear();
    this->overflowed = false;
    if (overflowed) {
        msg.reply = Reply::MALFORMED;
        return msg;
    }
    if (msg.text.empty()) {
        msg.reply = Reply::ACCEPTED;
        return msg;
    }
    switch (msg.text[0]) {
        case 't':
        case 'T':
        case 'r':
        case 'R':
        case 'd':
        case 'D':
        case 'b':
        case 'B': {
            auto [decoded, err] = decode(msg.text);
            if (err) {
                msg.reply = Reply::MALFORMED;
                return msg;
            }
            return Message{.reply = Reply::FRAME, .frame = decoded};
        }
        default:
            return msg;
    }
}
}
