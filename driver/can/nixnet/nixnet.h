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
#include <memory>
#include <optional>
#include <span>
#include <string>
#include <utility>
#include <vector>

#include "driver/can/can.h"
#include "driver/can/nixnet/api.h"

/// @brief the NI-XNET backend for NI CAN interfaces on Windows, Linux, and NI Linux
/// Real-Time.
namespace driver::can::nixnet {
/// @brief the size of a raw frame's header, before its payload.
constexpr std::size_t HEADER_SIZE = 16;
/// @brief the size of the largest raw CAN frame.
constexpr std::size_t MAX_RAW_SIZE = HEADER_SIZE + MAX_FD_LENGTH;

/// @returns the size of the raw frame that holds a payload: the header and the payload
/// padded to a multiple of 8 bytes, never less than 8.
[[nodiscard]] std::size_t raw_size(std::uint8_t length);

/// @brief encodes a frame in NI-XNET's raw frame format.
/// @param frame a frame that validate accepts.
/// @param fd true when the interface runs CAN FD, where a classic frame needs its own
/// type.
/// @param out receives the raw frame. It must hold raw_size(frame.length) bytes.
/// @returns the number of bytes written.
std::size_t encode(const Frame &frame, bool fd, std::span<std::uint8_t> out);

/// @brief decodes the raw frame at the start of bytes.
/// @returns the number of bytes the raw frame occupies. CRITICAL_HARDWARE_ERROR when
/// bytes end before the frame does.
[[nodiscard]] std::pair<std::size_t, x::errors::Error>
decode(std::span<const std::uint8_t> bytes, Frame &frame);

/// @brief the longest a receive sleeps between polls of an empty input session.
const auto POLL_INTERVAL = x::telem::MILLISECOND;

/// @brief an NI-XNET interface with an input stream session and, unless listen only, an
/// output stream session. receive polls the input session every POLL_INTERVAL while it
/// is empty.
class Bus final : public can::Bus {
    std::shared_ptr<API> api;
    nxSessionRef_t in;
    std::optional<nxSessionRef_t> out;
    std::string name;
    bool fd;
    std::array<std::uint8_t, 4096> buffer{};
    std::size_t offset = 0;
    std::size_t size = 0;
    bool closed = false;

public:
    Bus(std::shared_ptr<API> api,
        nxSessionRef_t in,
        std::optional<nxSessionRef_t> out,
        std::string name,
        bool fd);
    ~Bus() override;

    [[nodiscard]] std::pair<bool, x::errors::Error>
    receive(Frame &frame, x::telem::TimeSpan timeout) override;

    [[nodiscard]] x::errors::Error send(const Frame &frame) override;

    x::errors::Error close() override;
};

/// @brief finds and opens NI-XNET CAN interfaces. A channel's name is its interface
/// name, such as CAN1.
class Backend final : public can::Backend {
    std::shared_ptr<API> api;

public:
    explicit Backend(std::shared_ptr<API> api): api(std::move(api)) {}

    [[nodiscard]] std::pair<std::vector<Channel>, x::errors::Error> scan() override;

    [[nodiscard]] std::pair<std::unique_ptr<can::Bus>, x::errors::Error>
    open(const synnax::can::Properties &props) override;
};

/// @returns the backend over the installed NI-XNET, or an Unavailable backend holding
/// the reason NI-XNET did not load.
[[nodiscard]] std::shared_ptr<can::Backend> load();
}
