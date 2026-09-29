// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cstring>
#include <string>

#include "driver/can/gs_usb/protocol.h"

namespace driver::can::gs_usb {
namespace {
std::uint32_t
read_u32(const std::span<const std::uint8_t> bytes, const std::size_t at) {
    return static_cast<std::uint32_t>(bytes[at]) |
           static_cast<std::uint32_t>(bytes[at + 1]) << 8 |
           static_cast<std::uint32_t>(bytes[at + 2]) << 16 |
           static_cast<std::uint32_t>(bytes[at + 3]) << 24;
}

void write_u32(std::uint8_t *bytes, const std::uint32_t value) {
    bytes[0] = static_cast<std::uint8_t>(value);
    bytes[1] = static_cast<std::uint8_t>(value >> 8);
    bytes[2] = static_cast<std::uint8_t>(value >> 16);
    bytes[3] = static_cast<std::uint8_t>(value >> 24);
}

x::errors::Error short_reply(const std::string &what, const std::size_t size) {
    return {
        CRITICAL_HARDWARE_ERROR,
        "gs_usb adapter sent a " + std::to_string(size) + "-byte " + what
    };
}

/// @returns the timing limits at a byte offset of a BT_CONST answer.
TimingLimits
read_limits(const std::span<const std::uint8_t> bytes, const std::size_t at) {
    return {
        .tseg1_min = read_u32(bytes, at),
        .tseg1_max = read_u32(bytes, at + 4),
        .tseg2_min = read_u32(bytes, at + 8),
        .tseg2_max = read_u32(bytes, at + 12),
        .sjw_max = read_u32(bytes, at + 16),
        .brp_min = read_u32(bytes, at + 20),
        .brp_max = read_u32(bytes, at + 24),
        .brp_inc = read_u32(bytes, at + 28),
    };
}
}

std::pair<DeviceConfig, x::errors::Error>
decode_device_config(const std::span<const std::uint8_t> bytes) {
    if (bytes.size() < 12) return {{}, short_reply("device config", bytes.size())};
    return {
        {
            .channels = static_cast<std::uint32_t>(bytes[3]) + 1,
            .software_version = read_u32(bytes, 4),
            .hardware_version = read_u32(bytes, 8),
        },
        x::errors::NIL
    };
}

std::pair<BtConst, x::errors::Error>
decode_bt_const(const std::span<const std::uint8_t> bytes, const bool extended) {
    const std::size_t size = extended ? 72 : 40;
    if (bytes.size() < size)
        return {{}, short_reply("bit timing constants reply", bytes.size())};
    BtConst c{
        .features = read_u32(bytes, 0),
        .clock_hz = read_u32(bytes, 4),
        .nominal = read_limits(bytes, 8),
    };
    if (extended) c.data = read_limits(bytes, 40);
    return {c, x::errors::NIL};
}

std::array<std::uint8_t, 20> encode_bittiming(const Timing &timing) {
    std::array<std::uint8_t, 20> out{};
    const auto prop_seg = timing.tseg1 / 2;
    write_u32(out.data(), prop_seg);
    write_u32(out.data() + 4, timing.tseg1 - prop_seg);
    write_u32(out.data() + 8, timing.tseg2);
    write_u32(out.data() + 12, timing.sjw);
    write_u32(out.data() + 16, timing.brp);
    return out;
}

std::array<std::uint8_t, 8>
encode_mode(const std::uint32_t mode, const std::uint32_t flags) {
    std::array<std::uint8_t, 8> out{};
    write_u32(out.data(), mode);
    write_u32(out.data() + 4, flags);
    return out;
}

std::size_t encode_frame(
    const Frame &frame,
    const std::uint32_t echo_id,
    const std::uint8_t channel,
    const std::span<std::uint8_t, MAX_FRAME_SIZE> out
) {
    const auto data_size = frame.fd ? MAX_FD_LENGTH : MAX_CLASSIC_LENGTH;
    std::fill_n(out.begin(), HEADER_SIZE + data_size, 0);
    auto id = frame.id;
    if (frame.extended) id |= ID_EXTENDED;
    if (frame.type == Type::REMOTE) id |= ID_REMOTE;
    std::uint8_t flags = 0;
    if (frame.fd) flags |= FRAME_FD;
    if (frame.bitrate_switched) flags |= FRAME_BRS;
    write_u32(out.data(), echo_id);
    write_u32(out.data() + 4, id);
    out[8] = frame.fd ? *length_to_dlc(frame.length) : frame.length;
    out[9] = channel;
    out[10] = flags;
    if (frame.type != Type::REMOTE)
        std::memcpy(out.data() + HEADER_SIZE, frame.data.data(), frame.length);
    return HEADER_SIZE + data_size;
}

std::pair<HostFrame, x::errors::Error>
decode_frame(const std::span<const std::uint8_t> bytes, const bool timestamps) {
    if (bytes.size() < HEADER_SIZE) return {{}, short_reply("frame", bytes.size())};
    const std::uint8_t flags = bytes[10];
    const bool fd = (flags & FRAME_FD) != 0;
    const auto data_size = fd ? MAX_FD_LENGTH : MAX_CLASSIC_LENGTH;
    const auto size = HEADER_SIZE + data_size + (timestamps ? 4 : 0);
    if (bytes.size() < size) return {{}, short_reply("frame", bytes.size())};
    HostFrame hf;
    hf.echo_id = read_u32(bytes, 0);
    const auto id = read_u32(bytes, 4);
    hf.channel = bytes[9];
    hf.overflowed = (flags & FRAME_OVERFLOW) != 0;
    auto &f = hf.frame;
    f.extended = (id & ID_EXTENDED) != 0;
    f.id = id & (f.extended ? MAX_EXTENDED_ID : MAX_STANDARD_ID);
    f.type = Type::DATA;
    if ((id & ID_REMOTE) != 0) f.type = Type::REMOTE;
    if ((id & ID_ERROR) != 0) {
        f.type = Type::BUS_ERROR;
        f.id = id & MAX_EXTENDED_ID;
    }
    f.fd = fd;
    f.bitrate_switched = (flags & FRAME_BRS) != 0;
    f.error_passive = (flags & FRAME_ESI) != 0;
    f.length = dlc_to_length(bytes[8], fd);
    std::memcpy(f.data.data(), bytes.data() + HEADER_SIZE, data_size);
    if (timestamps) hf.timestamp_us = read_u32(bytes, HEADER_SIZE + data_size);
    return {hf, x::errors::NIL};
}
}
