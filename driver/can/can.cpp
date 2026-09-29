// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <format>
#include <string>

#include "driver/can/can.h"

namespace driver::can {
namespace {
constexpr std::array<std::uint8_t, 16> FD_LENGTHS =
    {0, 1, 2, 3, 4, 5, 6, 7, 8, 12, 16, 20, 24, 32, 48, 64};
}

std::uint8_t dlc_to_length(const std::uint8_t dlc, const bool fd) {
    if (dlc > 15) return fd ? MAX_FD_LENGTH : MAX_CLASSIC_LENGTH;
    if (!fd && dlc > 8) return MAX_CLASSIC_LENGTH;
    return FD_LENGTHS[dlc];
}

std::optional<std::uint8_t> length_to_dlc(const std::uint8_t length) {
    for (std::uint8_t dlc = 0; dlc < FD_LENGTHS.size(); dlc++)
        if (FD_LENGTHS[dlc] == length) return dlc;
    return std::nullopt;
}

x::errors::Error validate(const Frame &frame, const bool fd) {
    const auto max_id = frame.extended ? MAX_EXTENDED_ID : MAX_STANDARD_ID;
    if (frame.id > max_id)
        return {
            FRAME_ERROR,
            "identifier 0x" + std::format("{:X}", frame.id) + " exceeds the " +
                (frame.extended ? "29" : "11") + "-bit range"
        };
    if (frame.type == Type::BUS_ERROR)
        return {FRAME_ERROR, "a bus error frame cannot be sent"};
    if (frame.fd && !fd)
        return {FRAME_ERROR, "a CAN FD frame cannot be sent on a classic CAN bus"};
    if (frame.fd && frame.type == Type::REMOTE)
        return {FRAME_ERROR, "CAN FD has no remote frames"};
    if (!frame.fd && frame.bitrate_switched)
        return {FRAME_ERROR, "only a CAN FD frame can switch bitrates"};
    const auto max_length = frame.fd ? MAX_FD_LENGTH : MAX_CLASSIC_LENGTH;
    if (frame.length > max_length || !length_to_dlc(frame.length).has_value())
        return {
            FRAME_ERROR,
            "length " + std::to_string(frame.length) + " is not a valid " +
                (frame.fd ? "CAN FD" : "classic CAN") + " length"
        };
    return x::errors::NIL;
}

std::uint64_t Counter::extend(const std::uint32_t raw) {
    if (this->started && raw < this->last) this->high += 1ULL << 32;
    this->started = true;
    this->last = raw;
    return this->high | raw;
}

std::pair<std::unique_ptr<Bus>, x::errors::Error>
open(const Backends &backends, const synnax::can::Properties &props) {
    const auto it = backends.find(props.backend);
    if (it == backends.end())
        return {
            nullptr,
            x::errors::Error(
                CONFIG_ERROR,
                "unknown CAN backend '" + props.backend + "'"
            )
        };
    return it->second->open(props);
}
}
