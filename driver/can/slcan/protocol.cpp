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

#include "driver/can/slcan/protocol.h"

namespace driver::can::slcan {
namespace {
struct Rate {
    std::uint32_t bitrate;
    const char *command;
};

constexpr std::array<Rate, 10> BITRATES = {{
    {10000, "S0"},
    {20000, "S1"},
    {50000, "S2"},
    {100000, "S3"},
    {125000, "S4"},
    {250000, "S5"},
    {500000, "S6"},
    {750000, "S7"},
    {1000000, "S8"},
    {83333, "S9"},
}};

constexpr std::array<Rate, 2> DATA_BITRATES = {{
    {2000000, "Y2"},
    {5000000, "Y5"},
}};

template<std::size_t N>
std::pair<std::string, x::errors::Error> command(
    const std::array<Rate, N> &rates,
    const std::uint32_t bitrate,
    const std::string &kind
) {
    std::string supported;
    for (const auto &rate: rates) {
        if (rate.bitrate == bitrate)
            return {std::string(rate.command) + "\r", x::errors::NIL};
        if (!supported.empty()) supported += ", ";
        supported += std::to_string(rate.bitrate);
    }
    return {
        "",
        {CONFIG_ERROR,
         "slcan has no " + kind + " of " + std::to_string(bitrate) +
             " bit/s. Supported: " + supported}
    };
}

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

x::errors::Error malformed(const std::string_view line, const std::string &reason) {
    return {
        FRAME_ERROR,
        "malformed slcan frame '" + std::string(line) + "': " + reason
    };
}
}

std::pair<std::string, x::errors::Error> bitrate_command(const std::uint32_t bitrate) {
    return command(BITRATES, bitrate, "bitrate");
}

std::pair<std::string, x::errors::Error>
data_bitrate_command(const std::uint32_t bitrate) {
    return command(DATA_BITRATES, bitrate, "data bitrate");
}

std::string open_command(const bool listen_only) {
    return listen_only ? "L\r" : "O\r";
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

std::optional<Reply> Decoder::push(const char c, Frame &frame) {
    if (c == '\a') {
        this->line.clear();
        this->overflowed = false;
        return Reply::REJECTED;
    }
    if (c != '\r') {
        if (this->line.size() < MAX_LINE_LENGTH)
            this->line += c;
        else
            this->overflowed = true;
        return std::nullopt;
    }
    const std::string line = std::move(this->line);
    const bool overflowed = this->overflowed;
    this->line.clear();
    this->overflowed = false;
    if (overflowed) return Reply::MALFORMED;
    if (line.empty()) return Reply::ACCEPTED;
    switch (line[0]) {
        case 't':
        case 'T':
        case 'r':
        case 'R':
        case 'd':
        case 'D':
        case 'b':
        case 'B': {
            auto [decoded, err] = decode(line);
            if (err) return Reply::MALFORMED;
            frame = decoded;
            return Reply::FRAME;
        }
        default:
            return Reply::OTHER;
    }
}
}
