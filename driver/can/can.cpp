// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <array>
#include <chrono>
#include <format>
#include <string>
#include <vector>

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
    for (std::size_t dlc = 0; dlc < FD_LENGTHS.size(); dlc++)
        if (FD_LENGTHS[dlc] == length) return static_cast<std::uint8_t>(dlc);
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

x::errors::Error Bus::send(const Frame &frame) {
    if (this->listen_only)
        return {LISTEN_ONLY_ERROR, "channel " + this->name + " is listen only"};
    if (auto err = validate(frame, this->fd)) return err;
    return this->transmit(frame);
}

x::telem::TimeStamp
Aligner::align(const x::telem::TimeStamp adapter, const x::telem::TimeStamp host) {
    const auto measured = host - adapter;
    if (!this->offset.has_value()) {
        this->offset = this->target = this->smallest = measured;
        this->window_start = this->last = adapter;
        return host;
    }
    this->smallest = std::min(this->smallest, measured);
    if (adapter - this->window_start >= WINDOW) {
        this->target = this->smallest;
        this->smallest = measured;
        this->window_start = adapter;
    }
    const auto budget = std::max(adapter - this->last, x::telem::TimeSpan::ZERO()) /
                        SLEW;
    this->last = adapter;
    auto &offset = *this->offset;
    offset += std::clamp(
        this->target - offset,
        x::telem::TimeSpan::ZERO() - budget,
        budget
    );
    return adapter + offset;
}

namespace bus_error {
namespace {
/// @brief a bit of a BUS_ERROR frame and what it means.
struct Meaning {
    std::uint32_t bit;
    const char *text;
};

constexpr std::array<Meaning, 7> CLASSES = {{
    {TX_TIMEOUT, "transmit timeout"},
    {LOST_ARBITRATION, "lost arbitration"},
    {TRANSCEIVER, "transceiver fault"},
    {NO_ACK, "no acknowledgement"},
    {BUS_OFF, "bus off"},
    {BUS, "bus error"},
    {RESTARTED, "controller restarted"},
}};

constexpr std::array<Meaning, 6> CONTROLLER_DETAILS = {{
    {CONTROLLER_RX_OVERFLOW, "receive overflow, frames lost"},
    {CONTROLLER_TX_OVERFLOW, "transmit overflow"},
    {CONTROLLER_RX_WARNING, "receive error warning"},
    {CONTROLLER_TX_WARNING, "transmit error warning"},
    {CONTROLLER_RX_PASSIVE, "receive error passive"},
    {CONTROLLER_TX_PASSIVE, "transmit error passive"},
}};

constexpr std::array<Meaning, 6> PROTOCOL_KINDS = {{
    {PROTOCOL_BIT, "bit error"},
    {PROTOCOL_FORM, "form error"},
    {PROTOCOL_STUFF, "stuff error"},
    {PROTOCOL_BIT0, "dominant bit error"},
    {PROTOCOL_BIT1, "recessive bit error"},
    {PROTOCOL_OVERLOAD, "overload"},
}};

/// @brief appends the text of each meaning whose bit is set in bits.
/// @returns true when it appended any.
template<std::size_t N>
bool append(
    std::vector<std::string> &parts,
    const std::array<Meaning, N> &meanings,
    const std::uint32_t bits
) {
    const auto before = parts.size();
    for (const auto &[bit, text]: meanings)
        if ((bits & bit) != 0) parts.emplace_back(text);
    return parts.size() > before;
}
}

Frame make(
    const std::uint32_t classes,
    const std::uint8_t details,
    const x::telem::TimeStamp time
) {
    Frame frame{.id = classes, .type = Type::BUS_ERROR, .length = 8};
    frame.data[1] = details;
    frame.time = time;
    frame.clock = Clock::HOST;
    return frame;
}

std::optional<std::string> describe(const Frame &frame) {
    const auto classes = frame.id;
    if ((classes & CONTROLLER) != 0 && (classes & ~(CONTROLLER | COUNTERS)) == 0 &&
        frame.data[1] == CONTROLLER_ACTIVE)
        return std::nullopt;
    std::vector<std::string> parts;
    append(parts, CLASSES, classes);
    if ((classes & PROTOCOL) != 0 && !append(parts, PROTOCOL_KINDS, frame.data[2]))
        parts.emplace_back("protocol violation");
    if ((classes & CONTROLLER) != 0 &&
        !append(parts, CONTROLLER_DETAILS, frame.data[1]) &&
        (frame.data[1] & CONTROLLER_ACTIVE) == 0)
        parts.emplace_back("controller fault");
    if ((classes & COUNTERS) != 0)
        parts.push_back(
            std::format(
                "{} transmit and {} receive errors",
                frame.data[6],
                frame.data[7]
            )
        );
    if (parts.empty()) parts.emplace_back("an unknown bus error");
    std::string text = "the adapter reported ";
    for (std::size_t i = 0; i < parts.size(); i++)
        text += (i == 0 ? "" : ", ") + parts[i];
    return text;
}
}

x::telem::TimeStamp steady_now() {
    return x::telem::TimeStamp(
        std::chrono::duration_cast<std::chrono::nanoseconds>(
            std::chrono::steady_clock::now().time_since_epoch()
        )
            .count()
    );
}

Counter::Counter(const x::telem::TimeSpan tick): tick(tick) {}

std::uint64_t Counter::extend(const std::uint32_t raw, const x::telem::TimeStamp host) {
    std::uint64_t count = raw;
    if (this->last.has_value()) {
        const auto prev = *this->last;
        const std::uint32_t delta = raw - prev.raw;
        const auto elapsed = std::max<std::int64_t>(
            (host - prev.host).nanoseconds(),
            0
        );
        const auto missed = elapsed / this->tick.nanoseconds() -
                            static_cast<std::int64_t>(delta);
        // The nearest whole number of wraps to the counts the host clock saw.
        const std::int64_t wraps = missed > 0 ? (missed + (1LL << 31)) >> 32 : 0;
        count = prev.count + delta + (static_cast<std::uint64_t>(wraps) << 32);
    }
    this->last = Reading{.raw = raw, .count = count, .host = host};
    return count;
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
