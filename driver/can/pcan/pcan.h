// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <optional>
#include <string>
#include <utility>
#include <vector>

#include "driver/can/can.h"
#include "driver/can/pcan/api.h"

/// @brief the PCAN-Basic backend for PEAK adapters on Windows and macOS.
namespace driver::can::pcan {
/// @brief the clock of the CAN FD controllers in PEAK adapters.
constexpr std::uint32_t FD_CLOCK_HZ = 80000000;

/// @returns the handle of a PCAN-Basic channel name such as PCAN_USBBUS1, or nullopt
/// when no USB, PCI, or LAN channel has that name.
[[nodiscard]] std::optional<TPCANHandle> parse_channel(const std::string &name);

/// @returns the BTR0BTR1 code of a classic CAN bitrate.
/// @returns CONFIG_ERROR when PCAN-Basic has no code for the bitrate.
[[nodiscard]] std::pair<TPCANBaudrate, x::errors::Error>
baud_code(std::uint32_t bitrate);

/// @returns the PCAN-Basic bitrate string for a CAN FD bus.
/// @returns CONFIG_ERROR when the 80 MHz clock cannot produce either bitrate exactly.
[[nodiscard]] std::pair<std::string, x::errors::Error>
fd_bitrate(std::uint32_t nominal, std::uint32_t data);

/// @brief the longest a receive sleeps between polls of an empty receive queue.
const auto POLL_INTERVAL = x::telem::MILLISECOND;

/// @brief an initialized PCAN-Basic channel. PCAN-Basic has no blocking read, so
/// receive polls the receive queue every POLL_INTERVAL while it is empty.
class Bus final : public can::Bus {
    std::shared_ptr<API> api;
    TPCANHandle handle;
    bool closed = false;

    /// @brief takes the next frame from the receive queue.
    /// @returns nullopt when the queue is empty.
    std::optional<std::pair<bool, x::errors::Error>> take(Frame &frame);

    [[nodiscard]] x::errors::Error transmit(const Frame &frame) override;

public:
    Bus(std::shared_ptr<API> api,
        TPCANHandle handle,
        std::string name,
        bool fd,
        bool listen_only);
    ~Bus() override;

    [[nodiscard]] std::pair<bool, x::errors::Error>
    receive(Frame &frame, x::telem::TimeSpan timeout) override;

    x::errors::Error close() override;
};

/// @brief finds and opens PEAK channels through PCAN-Basic.
class Backend final : public can::Backend {
    std::shared_ptr<API> api;

public:
    explicit Backend(std::shared_ptr<API> api): api(std::move(api)) {}

    [[nodiscard]] std::pair<std::vector<Channel>, x::errors::Error> scan() override;

    [[nodiscard]] std::pair<std::unique_ptr<can::Bus>, x::errors::Error>
    open(const synnax::can::Properties &props) override;
};

/// @returns the backend over the installed PCAN-Basic library, or an Unavailable
/// backend holding the reason the library did not load.
[[nodiscard]] std::shared_ptr<can::Backend> load();
}
