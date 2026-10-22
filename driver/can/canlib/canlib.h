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
#include <mutex>
#include <optional>
#include <string>
#include <utility>
#include <vector>

#include "driver/can/can.h"
#include "driver/can/canlib/api.h"

/// @brief the CANlib backend for Kvaser adapters on Windows.
namespace driver::can::canlib {
/// @brief the resolution CANlib stamps received frames with.
const auto TIMER_SCALE = x::telem::MICROSECOND * 10;

/// @returns the CANlib constant of a classic bitrate.
/// @returns CONFIG_ERROR when CANlib has no constant for the bitrate.
[[nodiscard]] std::pair<long, x::errors::Error> bitrate_constant(std::uint32_t bitrate);

/// @returns the CANlib constant of a CAN FD arbitration or data bitrate.
/// @returns CONFIG_ERROR when CANlib has no constant for the bitrate.
[[nodiscard]] std::pair<long, x::errors::Error>
fd_bitrate_constant(std::uint32_t bitrate, bool data);

/// @brief a CANlib channel on the bus. It reads through one handle and writes through a
/// second, as Kvaser advises for a handle per thread.
class Bus final : public can::Bus {
    std::shared_ptr<API> api;
    canHandle rx;
    std::optional<canHandle> tx;
    Counter counter{TIMER_SCALE};
    /// @brief a received frame that receive returns after the overrun it followed.
    std::optional<Frame> pending;
    bool closed = false;

    [[nodiscard]] x::errors::Error transmit(const Frame &frame) override;

public:
    Bus(std::shared_ptr<API> api,
        canHandle rx,
        std::optional<canHandle> tx,
        std::string name,
        bool fd);
    ~Bus() override;

    [[nodiscard]] std::pair<bool, x::errors::Error>
    receive(Frame &frame, x::telem::TimeSpan timeout) override;

    x::errors::Error close() override;
};

/// @brief finds and opens Kvaser channels through CANlib. A channel's name is the EAN
/// and serial number of its card and its place on the card, such as
/// 73-30130-00685-0:12345:1, because CANlib renumbers channels when it enumerates the
/// cards. Safe for concurrent use.
class Backend final : public can::Backend {
    std::shared_ptr<API> api;
    /// @brief held from an enumeration to the opens that use its channel numbers.
    std::mutex mu;

    /// @brief a channel of the last enumeration.
    struct Found {
        int number;
        std::string name;
        std::string description;
        bool is_virtual;
    };

    /// @brief enumerates the cards and lists their channels.
    std::pair<std::vector<Found>, x::errors::Error> enumerate();

public:
    explicit Backend(std::shared_ptr<API> api): api(std::move(api)) {}

    [[nodiscard]] std::pair<std::vector<Channel>, x::errors::Error> scan() override;

    /// @returns CONFIG_ERROR when the name is not of a card and a channel on it, and
    /// UNREACHABLE_ERROR when no card has the channel or another program holds it.
    [[nodiscard]] std::pair<std::unique_ptr<can::Bus>, x::errors::Error>
    open(const synnax::can::Properties &props) override;
};

/// @returns an error whose message is context followed by CANlib's text for status. Its
/// type is UNREACHABLE_ERROR when the card is absent, removed, or held by another
/// program, CONFIG_ERROR when the driver is missing or a parameter is wrong,
/// TEMPORARY_HARDWARE_ERROR when the transmit buffer is full, and
/// CRITICAL_HARDWARE_ERROR otherwise.
[[nodiscard]] x::errors::Error
translate(API &api, canStatus status, const std::string &context);

/// @returns the backend over the installed CANlib, or an Unavailable backend holding
/// the reason CANlib did not load.
[[nodiscard]] std::shared_ptr<can::Backend> load();
}
