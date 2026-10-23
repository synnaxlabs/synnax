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

/// @brief an initialized PCAN-Basic channel. receive drains the receive queue, then
/// reads the channel status, then waits on the channel's receive event. A change of bus
/// state arrives as a BUS_ERROR frame. An unplugged adapter only shows in the status,
/// so send reads it too.
class Bus final : public can::Bus {
    std::shared_ptr<API> api;
    TPCANHandle handle;
    std::unique_ptr<ReceiveEvent> event;
    /// @brief the bus light, bus heavy, and bus passive bits of the last status.
    TPCANStatus state = PCAN_ERROR_OK;
    bool closed = false;

    /// @brief takes the next frame from the receive queue.
    /// @returns nullopt when the queue is empty.
    std::optional<std::pair<bool, x::errors::Error>> take(Frame &frame);

    /// @brief applies a status from Read or GetStatus.
    /// @returns a BUS_ERROR frame when the bus state changed, the error of a fault in
    /// status, or nullopt.
    std::optional<std::pair<bool, x::errors::Error>>
    apply(TPCANStatus status, Frame &frame);

    /// @returns the error of a fault in status, or NIL for a bus state.
    [[nodiscard]] x::errors::Error fault(TPCANStatus status);

    [[nodiscard]] x::errors::Error transmit(const Frame &frame) override;

public:
    Bus(std::shared_ptr<API> api,
        TPCANHandle handle,
        std::unique_ptr<ReceiveEvent> event,
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

/// @returns an error whose message is context followed by PCAN-Basic's text for status.
/// Its type is UNREACHABLE_ERROR when the adapter is absent or busy or the bus is off,
/// CONFIG_ERROR when the driver is missing or a parameter is wrong,
/// TEMPORARY_HARDWARE_ERROR when the transmit queue is full, and
/// CRITICAL_HARDWARE_ERROR otherwise.
[[nodiscard]] x::errors::Error
translate(API &api, TPCANStatus status, const std::string &context);

/// @returns the backend over the installed PCAN-Basic library, or an Unavailable
/// backend holding the reason the library did not load.
[[nodiscard]] std::shared_ptr<can::Backend> load();
}
