// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <array>
#include <cstddef>
#include <cstdint>
#include <optional>
#include <span>
#include <utility>
#include <vector>

#include "x/cpp/errors/errors.h"

#include "driver/can/can.h"
#include "driver/can/timing.h"

/// @brief the gs_usb protocol that candleLight and CANable firmware speak, written from
/// the Linux kernel driver's documented behavior and the MIT-licensed Python gs_usb
/// package. Every multi-byte field is little-endian.
namespace driver::can::gs_usb {
/// @brief a USB vendor and product id pair.
struct UsbId {
    std::uint16_t vendor;
    std::uint16_t product;
};

/// @brief the ids of the adapters that speak gs_usb.
constexpr std::array<UsbId, 5> USB_IDS = {{
    {0x1D50, 0x606F},
    {0x1209, 0x2323},
    {0x1CD2, 0x606F},
    {0x16D0, 0x10B8},
    {0x1209, 0xCA01},
}};

/// @brief the USB interface every gs_usb adapter exposes.
constexpr int INTERFACE = 0;
/// @brief the bulk endpoint the adapter sends frames on.
constexpr std::uint8_t ENDPOINT_IN = 0x81;
/// @brief the bulk endpoint the adapter receives frames on.
constexpr std::uint8_t ENDPOINT_OUT = 0x02;
/// @brief the request type of a vendor request to the interface, host to device.
constexpr std::uint8_t REQUEST_OUT = 0x41;
/// @brief the request type of a vendor request to the interface, device to host.
constexpr std::uint8_t REQUEST_IN = 0xC1;

/// @brief the vendor requests of the protocol.
enum class Request : std::uint8_t {
    BITTIMING = 1,
    MODE = 2,
    BT_CONST = 4,
    DEVICE_CONFIG = 5,
    DATA_BITTIMING = 10,
    BT_CONST_EXT = 11,
};

/// @brief the mode that stops the channel.
constexpr std::uint32_t MODE_RESET = 0;
/// @brief the mode that starts the channel.
constexpr std::uint32_t MODE_START = 1;

/// @brief mode flags, which double as the feature bits of BtConst.
constexpr std::uint32_t FEATURE_LISTEN_ONLY = 1U << 0;
constexpr std::uint32_t FEATURE_HW_TIMESTAMP = 1U << 4;
constexpr std::uint32_t FEATURE_FD = 1U << 8;
constexpr std::uint32_t FEATURE_BT_CONST_EXT = 1U << 10;

/// @brief the host frame flags.
constexpr std::uint8_t FRAME_OVERFLOW = 1U << 0;
constexpr std::uint8_t FRAME_FD = 1U << 1;
constexpr std::uint8_t FRAME_BRS = 1U << 2;
constexpr std::uint8_t FRAME_ESI = 1U << 3;

/// @brief the echo id of a frame the adapter received from the bus, as opposed to the
/// echo of a frame the host sent.
constexpr std::uint32_t ECHO_ID_RX = 0xFFFFFFFF;

/// @brief the identifier flags, shared with SocketCAN.
constexpr std::uint32_t ID_EXTENDED = 0x80000000;
constexpr std::uint32_t ID_REMOTE = 0x40000000;
constexpr std::uint32_t ID_ERROR = 0x20000000;

/// @brief the size of a host frame's header.
constexpr std::size_t HEADER_SIZE = 12;
/// @brief the size of the largest host frame: an FD frame with a timestamp.
constexpr std::size_t MAX_FRAME_SIZE = HEADER_SIZE + MAX_FD_LENGTH + 4;

/// @brief the adapter's answer to DEVICE_CONFIG.
struct DeviceConfig {
    /// @brief the number of CAN channels.
    std::uint32_t channels = 0;
    std::uint32_t software_version = 0;
    std::uint32_t hardware_version = 0;
};

/// @brief the adapter's answer to BT_CONST or BT_CONST_EXT.
struct BtConst {
    /// @brief the FEATURE_* bits the adapter supports.
    std::uint32_t features = 0;
    /// @brief the CAN controller clock in hertz.
    std::uint32_t clock_hz = 0;
    /// @brief the ranges of the nominal bit timing registers.
    TimingLimits nominal;
    /// @brief the ranges of the data phase registers. BT_CONST_EXT only.
    TimingLimits data;
};

/// @brief a frame from the adapter.
struct HostFrame {
    /// @brief ECHO_ID_RX for a received frame, or the id of the sent frame it echoes.
    std::uint32_t echo_id = 0;
    /// @brief the CAN channel of the adapter the frame belongs to.
    std::uint8_t channel = 0;
    /// @brief true when the adapter lost frames before this one.
    bool overflowed = false;
    /// @brief the frame, without a time.
    Frame frame;
    /// @brief the adapter's microsecond timestamp, when timestamps are on.
    std::optional<std::uint32_t> timestamp_us;
};

/// @brief decodes the answer to DEVICE_CONFIG.
/// @returns CRITICAL_HARDWARE_ERROR when bytes is shorter than 12 bytes.
[[nodiscard]] std::pair<DeviceConfig, x::errors::Error>
decode_device_config(std::span<const std::uint8_t> bytes);

/// @brief decodes the 40-byte answer to BT_CONST or the 72-byte answer to BT_CONST_EXT.
/// @returns CRITICAL_HARDWARE_ERROR when bytes is shorter than extended needs.
[[nodiscard]] std::pair<BtConst, x::errors::Error>
decode_bt_const(std::span<const std::uint8_t> bytes, bool extended);

/// @returns the body of a BITTIMING or DATA_BITTIMING request.
[[nodiscard]] std::vector<std::uint8_t> encode_bittiming(const Timing &timing);

/// @returns the body of a MODE request.
[[nodiscard]] std::vector<std::uint8_t>
encode_mode(std::uint32_t mode, std::uint32_t flags);

/// @returns a frame encoded as a host frame without a timestamp.
/// @param frame a frame that validate accepts.
/// @param echo_id the id the adapter echoes once it sends the frame.
/// @param channel the adapter channel to send on.
[[nodiscard]] std::vector<std::uint8_t>
encode_frame(const Frame &frame, std::uint32_t echo_id, std::uint8_t channel);

/// @returns the adapter channel of a host frame the adapter received from the bus, or
/// nullopt for the echo of a frame the host sent. CRITICAL_HARDWARE_ERROR when bytes
/// are shorter than a header.
[[nodiscard]] std::pair<std::optional<std::uint8_t>, x::errors::Error>
decode_channel(std::span<const std::uint8_t> bytes);

/// @brief decodes a host frame from the adapter.
/// @param timestamped true when the adapter runs with hardware timestamps on.
/// @returns CRITICAL_HARDWARE_ERROR when bytes is shorter than the frame its flags
/// describe.
[[nodiscard]] std::pair<HostFrame, x::errors::Error>
decode_frame(std::span<const std::uint8_t> bytes, bool timestamped);
}
