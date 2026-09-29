// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <tuple>
#include <vector>

#include "x/cpp/binary/binary.h"

#include "driver/can/gs_usb/protocol.h"

namespace driver::can::gs_usb {
namespace {
x::errors::Error short_reply(const std::string &what, const std::size_t size) {
    return {
        CRITICAL_HARDWARE_ERROR,
        "gs_usb adapter sent a " + std::to_string(size) + "-byte " + what
    };
}

/// @returns the timing limits that reader is at in a BT_CONST answer.
TimingLimits read_limits(x::binary::Reader &reader) {
    TimingLimits limits;
    limits.tseg1_min = reader.uint32();
    limits.tseg1_max = reader.uint32();
    limits.tseg2_min = reader.uint32();
    limits.tseg2_max = reader.uint32();
    limits.sjw_max = reader.uint32();
    limits.brp_min = reader.uint32();
    limits.brp_max = reader.uint32();
    limits.brp_inc = reader.uint32();
    return limits;
}
}

std::pair<DeviceConfig, x::errors::Error>
decode_device_config(const std::span<const std::uint8_t> bytes) {
    if (bytes.size() < 12) return {{}, short_reply("device config", bytes.size())};
    x::binary::Reader reader(bytes.data(), bytes.size(), 3);
    DeviceConfig config;
    config.channels = static_cast<std::uint32_t>(reader.uint8()) + 1;
    config.software_version = reader.uint32();
    config.hardware_version = reader.uint32();
    return {config, x::errors::NIL};
}

std::pair<BtConst, x::errors::Error>
decode_bt_const(const std::span<const std::uint8_t> bytes, const bool extended) {
    const std::size_t size = extended ? 72 : 40;
    if (bytes.size() < size)
        return {{}, short_reply("bit timing constants reply", bytes.size())};
    x::binary::Reader reader(bytes.data(), bytes.size());
    BtConst c;
    c.features = reader.uint32();
    c.clock_hz = reader.uint32();
    c.nominal = read_limits(reader);
    if (extended) c.data = read_limits(reader);
    return {c, x::errors::NIL};
}

std::vector<std::uint8_t> encode_bittiming(const Timing &timing) {
    std::vector<std::uint8_t> out;
    x::binary::Writer writer(out, 20);
    const auto prop_seg = timing.tseg1 / 2;
    writer.uint32(prop_seg);
    writer.uint32(timing.tseg1 - prop_seg);
    writer.uint32(timing.tseg2);
    writer.uint32(timing.sjw);
    writer.uint32(timing.brp);
    return out;
}

std::vector<std::uint8_t>
encode_mode(const std::uint32_t mode, const std::uint32_t flags) {
    std::vector<std::uint8_t> out;
    x::binary::Writer writer(out, 8);
    writer.uint32(mode);
    writer.uint32(flags);
    return out;
}

std::vector<std::uint8_t> encode_frame(
    const Frame &frame,
    const std::uint32_t echo_id,
    const std::uint8_t channel
) {
    const auto data_size = frame.fd ? MAX_FD_LENGTH : MAX_CLASSIC_LENGTH;
    std::vector<std::uint8_t> out;
    x::binary::Writer writer(out, HEADER_SIZE + data_size);
    auto id = frame.id;
    if (frame.extended) id |= ID_EXTENDED;
    if (frame.type == Type::REMOTE) id |= ID_REMOTE;
    std::uint8_t flags = 0;
    if (frame.fd) flags |= FRAME_FD;
    if (frame.bitrate_switched) flags |= FRAME_BRS;
    writer.uint32(echo_id);
    writer.uint32(id);
    writer.uint8(frame.fd ? *length_to_dlc(frame.length) : frame.length);
    writer.uint8(channel);
    writer.uint8(flags);
    writer.uint8(0);
    if (frame.type != Type::REMOTE) writer.write(frame.data.data(), frame.length);
    return out;
}

std::pair<std::optional<std::uint8_t>, x::errors::Error>
decode_channel(const std::span<const std::uint8_t> bytes) {
    if (bytes.size() < HEADER_SIZE)
        return {std::nullopt, short_reply("frame", bytes.size())};
    x::binary::Reader reader(bytes.data(), bytes.size());
    if (reader.uint32() != ECHO_ID_RX) return {std::nullopt, x::errors::NIL};
    std::ignore = reader.uint32();
    std::ignore = reader.uint8();
    return {reader.uint8(), x::errors::NIL};
}

std::pair<HostFrame, x::errors::Error>
decode_frame(const std::span<const std::uint8_t> bytes, const bool timestamped) {
    if (bytes.size() < HEADER_SIZE) return {{}, short_reply("frame", bytes.size())};
    const bool fd = (bytes[10] & FRAME_FD) != 0;
    const auto data_size = fd ? MAX_FD_LENGTH : MAX_CLASSIC_LENGTH;
    const auto size = HEADER_SIZE + data_size + (timestamped ? 4 : 0);
    if (bytes.size() < size) return {{}, short_reply("frame", bytes.size())};
    x::binary::Reader reader(bytes.data(), bytes.size());
    HostFrame hf;
    hf.echo_id = reader.uint32();
    const auto id = reader.uint32();
    const auto dlc = reader.uint8();
    hf.channel = reader.uint8();
    const auto flags = reader.uint8();
    std::ignore = reader.uint8();
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
    f.length = dlc_to_length(dlc, fd);
    reader.read(f.data.data(), data_size);
    if (timestamped) hf.timestamp_us = reader.uint32();
    return {hf, x::errors::NIL};
}
}
