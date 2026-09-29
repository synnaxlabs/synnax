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
    std::string name;
    bool fd;
    Counter counter;
    bool closed = false;

public:
    Bus(std::shared_ptr<API> api,
        canHandle rx,
        std::optional<canHandle> tx,
        std::string name,
        bool fd);
    ~Bus() override;

    [[nodiscard]] std::pair<bool, x::errors::Error>
    receive(Frame &frame, x::telem::TimeSpan timeout) override;

    [[nodiscard]] x::errors::Error send(const Frame &frame) override;

    x::errors::Error close() override;
};

/// @brief finds and opens Kvaser channels through CANlib. A channel's name is its
/// CANlib channel number, such as 0.
class Backend final : public can::Backend {
    std::shared_ptr<API> api;

public:
    explicit Backend(std::shared_ptr<API> api): api(std::move(api)) {}

    [[nodiscard]] std::pair<std::vector<Channel>, x::errors::Error> scan() override;

    [[nodiscard]] std::pair<std::unique_ptr<can::Bus>, x::errors::Error>
    open(const synnax::can::Properties &props) override;
};

/// @returns the backend over the installed CANlib, or an Unavailable backend holding
/// the reason CANlib did not load.
[[nodiscard]] std::shared_ptr<can::Backend> load();
}
